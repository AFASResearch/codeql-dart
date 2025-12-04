package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tsdart "github.com/UserNobody14/tree-sitter-dart/bindings/go"
	trap "github.com/github/codeql-go/extractor/trap"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// --- ID generation ------------------------------------------------------- // NEW

type IdGen struct {
	nextFileID     int
	nextFunctionID int
	nextStringID   int
	nextIdentID    int
	nextImportID   int
}

func (g *IdGen) NewFileID() int {
	g.nextFileID++
	return g.nextFileID
}

func (g *IdGen) NewFunctionID() int {
	g.nextFunctionID++
	return g.nextFunctionID
}

func (g *IdGen) NewStringID() int {
	g.nextStringID++
	return g.nextStringID
}

func (g *IdGen) NewIdentID() int {
	g.nextIdentID++
	return g.nextIdentID
}

func (g *IdGen) NewImportID() int {
	g.nextImportID++
	return g.nextImportID
}

// One global id generator is fine as long as you don't parallelise extraction.
// If you later do, move IdGen into a per-worker struct.
var globalIds IdGen            // NEW
var fileIDs = map[string]int{} // NEW: path → file-id

// ------------------------------------------------------------------------ //

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// set target if it's empty, using value
func setIfEmpty(target, value string) {
	if os.Getenv(target) == "" && value != "" {
		_ = os.Setenv(target, value)
	}
}

func main() {
	// Resolve dirs (DART first, then generic fallbacks)
	trapDir := firstEnv("CODEQL_EXTRACTOR_DART_TRAP_DIR", "CODEQL_TRAP_DIR", "TRAP_DIR")
	if trapDir == "" {
		trapDir = "./trap"
	} // local default
	srcArcDir := firstEnv("CODEQL_EXTRACTOR_DART_SOURCE_ARCHIVE_DIR", "SOURCE_ARCHIVE_DIR")

	// Propagate for legacy readers (no hardcoded paths, just copying values)
	setIfEmpty("CODEQL_TRAP_DIR", trapDir)
	setIfEmpty("CODEQL_EXTRACTOR_GO_TRAP_DIR", trapDir)
	setIfEmpty("CODEQL_EXTRACTOR_GO_SOURCE_ARCHIVE_DIR", srcArcDir)

	_ = os.MkdirAll(trapDir, 0o755)
	if srcArcDir != "" {
		_ = os.MkdirAll(srcArcDir, 0o755)
	}

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
	if lang == nil {
		return fmt.Errorf("nil language (dart grammar not linked)")
	}
	if err := parser.SetLanguage(lang); err != nil {
		return fmt.Errorf("SetLanguage failed: %w", err)
	}

	if t0 := parser.Parse([]byte("void main() {}"), nil); t0 == nil {
		return fmt.Errorf("sanity parse failed: nil tree (init)")
	} else {
		t0.Close()
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	// 1) don't index our own traps
	absTrap, _ := filepath.Abs(trapRoot)
	if strings.HasPrefix(strings.ToLower(absFile+string(os.PathSeparator)),
		strings.ToLower(absTrap+string(os.PathSeparator))) {
		return nil
	}

	// 2) make rel strictly repo-relative (no drive letters)
	// srcRoot must be the real project root, NOT trapRoot or cwd
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
	if err != nil {
		return err
	}
	if len(src) == 0 {
		return fmt.Errorf("empty file: %s", rel)
	}

	// Parse
	tree := parser.Parse(src, nil)
	if tree == nil {
		return fmt.Errorf("parse failed for %s: nil tree", rel)
	}
	defer tree.Close()

	root := tree.RootNode()

	// 1) Archive (optional)
	if srcArcRoot != "" {
		dstSrc := filepath.Join(srcArcRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dstSrc), 0o755); err != nil {
			return err
		}
		if err := copyFile(absFile, dstSrc); err != nil {
			return err
		}
	}

	// 2) TRAP writer
	trapPath := filepath.FromSlash(rel) // no ".trap"
	if err := os.MkdirAll(filepath.Dir(trapPath), 0o755); err != nil {
		return err
	}

	w, err := trap.NewWriter(trapPath, nil)
	if err != nil {
		return err
	}
	defer w.Close()

	// --- file() with id ----------------------------------------------------

	trapFile := filepath.ToSlash(rel) // path as seen in dbscheme
	fileID, ok := fileIDs[trapFile]
	if !ok {
		fileID = globalIds.NewFileID()
		fileIDs[trapFile] = fileID
		if err := w.Emit("file", []any{fileID, trapFile}); err != nil {
			return err
		}
	}

	// Walk AST and emit all other relations for this file
	if err := extractDart(root, src, rel, w, fileID, &globalIds); err != nil { // CHANGED
		return err
	}

	return nil
}

// Helper: tekst van een field
func childText(n *sitter.Node, src []byte, field string) string {
	c := n.ChildByFieldName(field)
	if c == nil {
		return ""
	}
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

// climb up a few levels to find an import/export/part directive
func findDirectiveAncestor(n *sitter.Node, maxUp int) *sitter.Node {
	for cur, steps := n, 0; cur != nil && steps <= maxUp; cur, steps = cur.Parent(), steps+1 {
		k := cur.Kind()
		// be generous with possible names
		if k == "import_directive" || k == "import_or_export" ||
			k == "export_directive" || k == "part_directive" ||
			k == "part_of_directive" || k == "library_import" ||
			strings.Contains(k, "import") || strings.Contains(k, "export") || strings.Contains(k, "part") {
			return cur
		}
	}
	return nil
}

func pos1(n *sitter.Node) (sl, sc, el, ec int) {
	sp, ep := n.StartPosition(), n.EndPosition()
	return int(sp.Row) + 1, int(sp.Column) + 1, int(ep.Row) + 1, int(ep.Column)
}

// Assumes:
//   - w.Emit(relation, []any{...}) exists
//   - src []byte is the file content
//   - rel is the path relative to DB root
// Adds relations (examples):
//   function_decl(file, name, sl, sc, el, ec)
//   param(file, funcNameOrId, index, name, kind, hasDefault, defaultText)
//   func_async_kind(file, funcNameOrId, "async"|"sync*"|"async*")
//   call_expr(file, id, calleeText, sl, sc, el, ec)
//   call_arg(file, callId, index, exprText)
//   call_named_arg(file, callId, "label", exprText)
//   ident_occurs(file, text, sl, sc, el, ec)
//   imports(file, uri, isDeferred, prefix)
//   export_decl(file, uri)
//   part_decl(file, uri)
//   part_of_decl(file, nameOrUri)

func extractDart(root *sitter.Node, src []byte, rel string, w *trap.Writer, fileID int, ids *IdGen) error { // CHANGED
	funcsCnt, stringsCnt := 0, 0
	trapFile := filepath.ToSlash(rel)

	// --- helpers -------------------------------------------------------------

	textOf := func(n *sitter.Node) string {
		if n == nil {
			return ""
		}
		return string(src[n.StartByte():n.EndByte()])
	}
	emitPos := func(n *sitter.Node) (sl, sc, el, ec int) {
		sp, ep := n.StartPosition(), n.EndPosition()
		return int(sp.Row) + 1, int(sp.Column) + 1, int(ep.Row) + 1, int(ep.Column) + 1
	}
	childByKind := func(n *sitter.Node, kind string) *sitter.Node {
		for i := uint(0); i < n.ChildCount(); i++ {
			ch := n.Child(i)
			if ch != nil && ch.Kind() == kind {
				return ch
			}
		}
		return nil
	}
	// quick literal shrinker for TRAP size
	shrink := func(s string, limit int) string {
		if len(s) > limit {
			return s[:limit]
		}
		return s
	}

	// anonymous function ids (stable per file order) – still used for names, but
	// the real primary key is now the numeric function id.
	anonFuncSeq := 0
	newAnonId := func(n *sitter.Node) string {
		anonFuncSeq++
		sl, sc, el, ec := emitPos(n)
		return fmt.Sprintf("%s:%d:%d-%d:%d#lambda%d", trapFile, sl, sc, el, ec, anonFuncSeq)
	}

	// Best-effort: detect async/sync*/async* on a function-like node
	funcAsyncKind := func(n *sitter.Node) string {
		full := textOf(n)
		if strings.Contains(full, "async*") {
			return "async*"
		}
		if strings.Contains(full, "sync*") {
			return "sync*"
		}
		if strings.Contains(full, "async") {
			return "async"
		}
		return "" // sync
	}

	// collect parameters from common shapes
	emitParams := func(funcID int, funcName string, fn *sitter.Node) {
		// Try a field first
		formals := fn.ChildByFieldName("parameters")
		if formals == nil {
			// common kind names in tree-sitter-dart: "formal_parameter_part", "formal_parameter_list"
			formals = childByKind(fn, "formal_parameter_list")
			if formals == nil {
				formals = childByKind(fn, "formal_parameter_part")
			}
		}
		if formals == nil {
			return
		}

		idx := 0
		var walkParams func(n *sitter.Node)
		walkParams = func(n *sitter.Node) {
			kind := n.Kind()
			switch kind {
			case "simple_formal_parameter", "field_formal_parameter":
				nameNode := n.ChildByFieldName("name")
				name := textOf(nameNode)
				if name == "" {
					for i := uint(0); i < n.ChildCount(); i++ {
						if ch := n.Child(i); ch != nil && strings.Contains(ch.Kind(), "identifier") {
							name = textOf(ch)
							break
						}
					}
				}
				hasDefault, defTxt := false, ""
				kindTag := "positional"
				_ = w.Emit("param", []any{fileID, funcID, idx, name, kindTag, hasDefault, defTxt})
				idx++

			case "default_formal_parameter":
				p := n.ChildByFieldName("parameter")
				name := ""
				if p != nil {
					if nameNode := p.ChildByFieldName("name"); nameNode != nil {
						name = textOf(nameNode)
					} else {
						for i := uint(0); i < p.ChildCount(); i++ {
							if ch := p.Child(i); strings.Contains(ch.Kind(), "identifier") {
								name = textOf(ch)
								break
							}
						}
					}
				}
				def := n.ChildByFieldName("default_value")
				defTxt := shrink(textOf(def), 200)
				kindTag := "positional"
				_ = w.Emit("param", []any{fileID, funcID, idx, name, kindTag, true, defTxt})
				idx++

			case "named_parameter", "named_formal_parameters":
				for i := uint(0); i < n.ChildCount(); i++ {
					ch := n.Child(i)
					if ch == nil {
						continue
					}
					if ch.Kind() == "default_formal_parameter" {
						p := ch.ChildByFieldName("parameter")
						name := ""
						if p != nil {
							if nameNode := p.ChildByFieldName("name"); nameNode != nil {
								name = textOf(nameNode)
							} else {
								for j := uint(0); j < p.ChildCount(); j++ {
									if id := p.Child(j); strings.Contains(id.Kind(), "identifier") {
										name = textOf(id)
										break
									}
								}
							}
						}
						def := ch.ChildByFieldName("default_value")
						defTxt := shrink(textOf(def), 200)
						_ = w.Emit("param", []any{fileID, funcID, idx, name, "named", true, defTxt})
						idx++
					} else if ch.Kind() == "simple_formal_parameter" || ch.Kind() == "field_formal_parameter" {
						name := ""
						if nameNode := ch.ChildByFieldName("name"); nameNode != nil {
							name = textOf(nameNode)
						} else {
							for j := uint(0); j < ch.ChildCount(); j++ {
								if id := ch.Child(j); strings.Contains(id.Kind(), "identifier") {
									name = textOf(id)
									break
								}
							}
						}
						_ = w.Emit("param", []any{fileID, funcID, idx, name, "named", false, ""})
						idx++
					}
				}

			default:
				for i := uint(0); i < n.ChildCount(); i++ {
					if ch := n.Child(i); ch != nil {
						walkParams(ch)
					}
				}
			}
		}
		walkParams(formals)
	}

	emitFunction := func(n *sitter.Node, explicitName string) int {
		funcsCnt++
		name := explicitName
		if name == "" {
			if nameNode := n.ChildByFieldName("name"); nameNode != nil {
				name = textOf(nameNode)
			}
		}
		if name == "" {
			name = newAnonId(n)
		}
		sl, sc, el, ec := emitPos(n)
		funcID := ids.NewFunctionID()
		_ = w.Emit("functions", []any{funcID, fileID, name, sl, sc, el, ec})
		if ak := funcAsyncKind(n); ak != "" {
			_ = w.Emit("func_async_kind", []any{funcID, ak})
		}
		emitParams(funcID, name, n)
		return funcID
	}

	emitCall := func(n *sitter.Node) {
		// unchanged, except still using trapFile for callId/call_* relations,
		// which you can later move to id-based too if you add them to dbscheme.
		args := n.ChildByFieldName("arguments")
		if args == nil {
			args = childByKind(n, "argument_list")
		}
		if args == nil {
			return
		}

		var callee string
		if fn := n.ChildByFieldName("function"); fn != nil {
			callee = textOf(fn)
		} else if t := n.ChildByFieldName("target"); t != nil {
			if m := n.ChildByFieldName("method"); m != nil {
				callee = textOf(t) + "." + textOf(m)
			} else {
				callee = textOf(t)
			}
		} else if r := n.ChildByFieldName("receiver"); r != nil {
			callee = textOf(r)
		} else {
			start := n.StartByte()
			if args != nil {
				callee = string(src[start:args.StartByte()])
				callee = strings.TrimSpace(callee)
			}
		}

		sl, sc, el, ec := emitPos(n)
		callId := fmt.Sprintf("%s:%d:%d-%d:%d#call", trapFile, sl, sc, el, ec)
		_ = w.Emit("call_expr", []any{trapFile, callId, shrink(callee, 300), sl, sc, el, ec})

		idx := 0
		for i := uint(0); i < args.NamedChildCount(); i++ {
			arg := args.NamedChild(i)
			if arg == nil {
				continue
			}
			ak := arg.Kind()
			switch ak {
			case "named_argument":
				lbl := ""
				if ln := arg.ChildByFieldName("name"); ln != nil {
					lbl = textOf(ln)
				} else if ln := arg.ChildByFieldName("label"); ln != nil {
					lbl = textOf(ln)
					lbl = strings.TrimSuffix(lbl, ":")
				} else {
					for j := uint(0); j < arg.ChildCount(); j++ {
						if id := arg.Child(j); id != nil && strings.Contains(id.Kind(), "identifier") {
							lbl = textOf(id)
							break
						}
					}
				}
				val := arg.ChildByFieldName("value")
				if val == nil {
					if arg.ChildCount() > 0 {
						val = arg.Child(arg.ChildCount() - 1)
					}
				}
				_ = w.Emit("call_named_arg", []any{trapFile, callId, lbl, shrink(textOf(val), 400)})

			default:
				_ = w.Emit("call_arg", []any{trapFile, callId, idx, shrink(textOf(arg), 400)})
				idx++
			}
		}
	}

	emitImportish := func(n *sitter.Node) {
		if dir := findDirectiveAncestor(n, 3); dir != nil {
			trimmed := strings.TrimLeft(textOf(dir), " \t\r\n")

			prefix := ""
			if p := dir.ChildByFieldName("prefix"); p != nil {
				prefix = textOf(p)
			} else if strings.HasPrefix(trimmed, "import") {
				if idx := strings.Index(trimmed, " as "); idx >= 0 {
					after := strings.TrimSpace(trimmed[idx+4:])
					j := 0
					for j < len(after) {
						c := after[j]
						if !(c == '_' || c == '$' ||
							(c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
							(c >= '0' && c <= '9')) {
							break
						}
						j++
					}
					prefix = after[:j]
				}
			}

			usl, usc, uel, uec := pos1(n)
			uriText := string(src[n.StartByte():n.EndByte()])

			switch {
			case strings.HasPrefix(trimmed, "import"):
				_ = w.Emit("imports", []any{fileID, prefix, uriText, usl, usc, uel, uec})
			case strings.HasPrefix(trimmed, "export"):
				_ = w.Emit("exports", []any{fileID, uriText, usl, usc, uel, uec})
			case strings.HasPrefix(trimmed, "part of"):
				_ = w.Emit("part_of_decl", []any{fileID, uriText, usl, usc, uel, uec})
			case strings.HasPrefix(trimmed, "part "):
				_ = w.Emit("part_decl", []any{fileID, uriText, usl, usc, uel, uec})
			}
		}

	}

	// --- Walk ---------------------------------------------------------------

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		kind := n.Kind()

		// string literals with id + fileID
		if kind == "string_literal" {
			stringsCnt++
			sl, sc, el, ec := emitPos(n)
			txt := textOf(n)
			strID := ids.NewStringID()
			_ = w.Emit("string_literals", []any{strID, fileID, shrink(txt, 200), sl, sc, el, ec})
		}

		// identifier occurrences with id + fileID
		if strings.Contains(kind, "identifier") {
			sl, sc, el, ec := emitPos(n)
			identID := ids.NewIdentID()
			_ = w.Emit("ident_occurs", []any{identID, fileID, textOf(n), sl, sc, el, ec})
		}

		// imports/exports/parts
		switch kind {
		case "import_or_export", "import_directive", "export_directive", "part_directive", "part_of_directive":
			emitImportish(n)
		}

		// function / method declarations / closures
		if strings.Contains(kind, "function_declaration") ||
			strings.Contains(kind, "method_declaration") ||
			strings.Contains(kind, "function_expression") ||
			strings.Contains(kind, "constructor_declaration") {
			_ = emitFunction(n, "")
		}

		// call-ish nodes
		if strings.Contains(kind, "call_expression") ||
			strings.Contains(kind, "method_invocation") ||
			strings.Contains(kind, "invocation") {
			emitCall(n)
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if ch := n.Child(i); ch != nil {
				walk(ch)
			}
		}
	}

	walk(root)

	_ = w.Emit("file_stats", []any{fileID, funcsCnt, stringsCnt}) // CHANGED: fileID, not path
	fmt.Printf("OK %s (%d funcs, %d strings)\n", rel, funcsCnt, stringsCnt)
	return nil
}
