package main

import (
	"context"
	"io"
	"io/fs"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	trap "github.com/github/codeql-go/extractor/trap"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tsdart "github.com/UserNobody14/tree-sitter-dart/bindings/go"
)

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// set target if it's empty, using value
func setIfEmpty(target, value string) { if os.Getenv(target) == "" && value != "" { _ = os.Setenv(target, value) } }


func main() {
	// Resolve dirs (DART first, then generic fallbacks)
	trapDir   := firstEnv("CODEQL_EXTRACTOR_DART_TRAP_DIR", "CODEQL_TRAP_DIR", "TRAP_DIR")
	if trapDir == "" { trapDir = "./trap" }          // local default
	srcArcDir := firstEnv("CODEQL_EXTRACTOR_DART_SOURCE_ARCHIVE_DIR", "SOURCE_ARCHIVE_DIR")

	// Propagate for legacy readers (no hardcoded paths, just copying values)
	setIfEmpty("CODEQL_TRAP_DIR", trapDir)
	setIfEmpty("CODEQL_EXTRACTOR_GO_TRAP_DIR", trapDir)
	setIfEmpty("CODEQL_EXTRACTOR_GO_SOURCE_ARCHIVE_DIR", srcArcDir)

	_ = os.MkdirAll(trapDir, 0o755)
	if srcArcDir != "" { _ = os.MkdirAll(srcArcDir, 0o755) }

	// If empty, we silently skip archiving (still valid for quick tests).
    ctx := context.Background()

	// Modes:
	//   --index <file>     : index a single file (CLI calls this per file)
	//   <source-root> or . : walk and extract all .dart files
	args := os.Args[1:]
	if len(args) >= 2 && args[0] == "--index" {
		file := args[1]
		if err := indexOne(ctx, trapDir, srcArcDir, ".", file); err != nil {
			die("index error: %v", err)
		}
		return
	}

	// Project mode: source root from arg or "."
	srcRoot := "."
	if len(args) >= 1 {
		srcRoot = args[0]
	}
	if err := walkAndIndex(ctx, trapDir, srcArcDir, srcRoot); err != nil {
		die("extract error: %v", err)
	}
}

func walkAndIndex(ctx context.Context, trapRoot, srcArcRoot, srcRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == ".dart_tool" || base == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".dart") {
			return nil
		}
		return indexOne(ctx, trapRoot, srcArcRoot, srcRoot, path)
	})
}

func indexOne(ctx context.Context, trapRoot, srcArcRoot, srcRoot, file string) error {
    parser := sitter.NewParser()
    defer parser.Close()

    lang := sitter.NewLanguage(tsdart.Language())
    if lang == nil { return fmt.Errorf("nil language (dart grammar not linked)") }
    if err := parser.SetLanguage(lang); err != nil { return fmt.Errorf("SetLanguage failed: %w", err) }

    if t0 := parser.Parse([]byte("void main() {}"), nil); t0 == nil {
        return fmt.Errorf("sanity parse failed: nil tree (init)")
    } else { t0.Close() }

    absFile, err := filepath.Abs(file)
    if err != nil { return err }

    // 1) don't index our own traps
	absTrap, _ := filepath.Abs(trapRoot)
	if strings.HasPrefix(strings.ToLower(absFile+string(os.PathSeparator)),
		strings.ToLower(absTrap+string(os.PathSeparator))) {
		return nil
	}

	// 2) make rel strictly repo-relative (no drive letters)
	if srcRoot == "" {
		return fmt.Errorf("srcRoot must be set to your project root")
	}
	absBase, err := filepath.Abs(srcRoot)

	rel, err := filepath.Rel(absBase, absFile)
	if err != nil {
		// fallback: just the filename, keeps it inside trap root
		rel = filepath.Base(absFile)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))


	// If the file is outside absBase, or still looks absolute, collapse to leaf.
	if strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, `..\`) || strings.HasPrefix(rel, "/") {
		rel = filepath.ToSlash(filepath.Base(absFile))
	}
	// if a drive leaked in anyway (e.g. absolute path), normalize to a safe leaf
	if strings.Contains(rel, ":") {
		rel = filepath.ToSlash(filepath.Base(absFile))
	}

	if strings.HasPrefix(rel, ".dart_tool/") ||
	   strings.HasPrefix(rel, "build/") ||
	   strings.HasPrefix(rel, ".external/") {
		return nil // quietly skip generated or cached dirs
	}

    // Read source
    src, err := os.ReadFile(absFile)
    if err != nil { return err }
    if len(src) == 0 { return fmt.Errorf("empty file: %s", rel) }

    // Parse
    tree := parser.Parse(src, nil)
    if tree == nil { return fmt.Errorf("parse failed for %s: nil tree", rel) }
    defer tree.Close()

    root := tree.RootNode()

    // 1) Archive (optional)
    if srcArcRoot != "" {
        dstSrc := filepath.Join(srcArcRoot, filepath.FromSlash(rel))
        if err := os.MkdirAll(filepath.Dir(dstSrc), 0o755); err != nil { return err }
        if err := copyFile(absFile, dstSrc); err != nil { return err }
    }

    // 2) TRAP writer
    trapPath := filepath.Join(trapRoot, filepath.FromSlash(rel)) // no ".trap"
	if err := os.MkdirAll(filepath.Dir(trapPath), 0o755); err != nil { return err }

	w, err := trap.NewWriter(trapPath, nil)
	if err != nil { return err }
	defer w.Close()

	if err := w.Emit("file", []any{rel}); err != nil { return err }

    // Walk AST – count only; print once per file
    funcsCnt, stringsCnt := 0, 0

    trapFile := filepath.ToSlash(rel)

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		kind := n.Kind()

		if strings.Contains(kind, "function") || strings.Contains(kind, "method") {
			funcsCnt++
			sp := n.StartPosition()
			ep := n.EndPosition()
			// Optional: get a name if your grammar exposes a child named "name"
			var name string
			if nameNode := n.ChildByFieldName("name"); nameNode != nil {
				name = string(src[nameNode.StartByte():nameNode.EndByte()])
			}
			_ = w.Emit("function_decl", []any{
				trapFile,
				name,                 // STRING (can be "")
				int(sp.Row) + 1,      // start line (1-based)
				int(sp.Column) + 1,   // start column (1-based)
				int(ep.Row) + 1,      // end line
				int(ep.Column) + 1,   // end column
			})
		}

		if kind == "string_literal" {
			stringsCnt++
			sp := n.StartPosition()
			ep := n.EndPosition()
			txt := string(src[n.StartByte():n.EndByte()])
			if len(txt) > 200 { // keep TRAP small
				txt = txt[:200]
			}
			_ = w.Emit("string_lit", []any{
				trapFile,
				int(sp.Row) + 1, int(sp.Column) + 1,
				int(ep.Row) + 1, int(ep.Column) + 1,
				txt,
			})
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if ch := n.Child(i); ch != nil {
				walk(ch)
			}
		}
	}

	walk(root)

	// optional summary row
	_ = w.Emit("file_stats", []any{trapFile, funcsCnt, stringsCnt})

    fmt.Printf("OK %s (%d funcs, %d strings)\n", rel, funcsCnt, stringsCnt)
    return nil
}

// Helper: tekst van een field
func childText(n *sitter.Node, src []byte, field string) string {
    c := n.ChildByFieldName(field)
    if c == nil { return "" }
    return string(src[c.StartByte():c.EndByte()])
}

func mustMkdir(p string) {
_ = os.MkdirAll(p, 0o755)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func toSlash(p string) string {
	// TRAP paths are usually slash-separated; normalize Windows backslashes.
	return strings.ReplaceAll(p, "\\", "/")
}

func die(fmtStr string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, fmtStr+"\n", a...)
	os.Exit(1)
}
