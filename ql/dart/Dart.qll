module Dart {

  // --- Raw database relations (externally provided) --------------------------

  external predicate file(int id, string path);
  external predicate functions(int id, int file, string name, int start_line, int start_col, int end_line, int end_col);
  external predicate func_async_kind(int function, string kind);
  external predicate ident_occurs(int id, int file, string text, int start_line, int start_col, int end_line, int end_col);
  external predicate string_literals(int id, int file, string text, int start_line, int start_col, int end_line, int end_col);
  external predicate imports(
    int id, int file, string prefix, string uri_text,
    int uri_start_line, int uri_start_col, int uri_end_line, int uri_end_col
  );
  external predicate file_stats(int file, int funcs, int strings);

  // --- Core types ------------------------------------------------------------

  class File extends int {
    File() { file(this, _) }
    string getPath() { exists(string p | file(this, p) and result = p) }
    string toString() { result = getPath() }
  }

  class Function extends int {
    Function() { functions(this, _, _, _, _, _, _) }

    File   getFile()      { exists(int    f  | functions(this, f, _, _, _, _, _) and result = f) }
    string getName()      { exists(string n  | functions(this, _, n, _, _, _, _) and result = n) }
    int    getStartLine() { exists(int    sl | functions(this, _, _, sl, _, _, _) and result = sl) }
    int    getStartCol()  { exists(int    sc | functions(this, _, _, _, sc, _, _) and result = sc) }
    int    getEndLine()   { exists(int    el | functions(this, _, _, _, _, el, _) and result = el) }
    int    getEndCol()    { exists(int    ec | functions(this, _, _, _, _, _, ec) and result = ec) }

    string getAsyncKind() { exists(string k | func_async_kind(this, k) and result = k) }

    predicate isAsync()     { this.getAsyncKind() = "async" }
    predicate isSyncStar()  { this.getAsyncKind() = "sync*" }
    predicate isAsyncStar() { this.getAsyncKind() = "async*" }

    string toString() { result = this.getName() }
  }

  class Ident extends int {
    Ident() { ident_occurs(this, _, _, _, _, _, _) }

    File   getFile()      { exists(int    f  | ident_occurs(this, f,  _, _, _, _, _) and result = f) }
    string getText()      { exists(string t  | ident_occurs(this, _,  t, _, _, _, _) and result = t) }
    int    getStartLine() { exists(int    sl | ident_occurs(this, _,  _, sl, _, _, _) and result = sl) }
    int    getStartCol()  { exists(int    sc | ident_occurs(this, _,  _, _, sc, _, _) and result = sc) }
    int    getEndLine()   { exists(int    el | ident_occurs(this, _,  _, _, _, el, _) and result = el) }
    int    getEndCol()    { exists(int    ec | ident_occurs(this, _,  _, _, _, _, ec) and result = ec) }

    string toString() { result = this.getText() }
  }

  class StringLiteral extends int {
    StringLiteral() { string_literals(this, _, _, _, _, _, _) }

    File   getFile()      { exists(int    f  | string_literals(this, f, _, _, _, _, _) and result = f) }
    string getText()      { exists(string t  | string_literals(this, _, t, _, _, _, _) and result = t) }
    int    getStartLine() { exists(int    sl | string_literals(this, _, _, sl, _, _, _) and result = sl) }
    int    getStartCol()  { exists(int    sc | string_literals(this, _, _, _, sc, _, _) and result = sc) }
    int    getEndLine()   { exists(int    el | string_literals(this, _, _, _, _, el, _) and result = el) }
    int    getEndCol()    { exists(int    ec | string_literals(this, _, _, _, _, _, ec) and result = ec) }

    string toString() { result = this.getText() }
  }

  class Import extends int {
    Import() { imports(this, _, _, _, _, _, _, _) }

    File   getFile()        { exists(int    f  | imports(this, f, _, _, _, _, _, _) and result = f) }
    string getPrefix()      { exists(string p  | imports(this, _, p, _, _, _, _, _) and result = p) }
    string getUriText()     { exists(string u  | imports(this, _, _, u, _, _, _, _) and result = u) }
    int    getUriStartLine(){ exists(int    sl | imports(this, _, _, _, sl, _, _, _) and result = sl) }
    int    getUriStartCol() { exists(int    sc | imports(this, _, _, _, _, sc, _, _) and result = sc) }
    int    getUriEndLine()  { exists(int    el | imports(this, _, _, _, _, _, el, _) and result = el) }
    int    getUriEndCol()   { exists(int    ec | imports(this, _, _, _, _, _, _, ec) and result = ec) }

    string getUriUnquoted() {
      exists(string u |
        u = getUriText() and
        (
          exists(string core | u = "'"  + core + "'"  and result = core) or
          exists(string core | u = "\"" + core + "\"" and result = core) or
          result = u
        )
      )
    }

    string toString() { result = this.getUriText() }
  }

  class FileStats extends int {
    FileStats() { exists(int f | f = this and file_stats(f, _, _)) }

    File getFile() { result = this }
    int  getNumFunctions() { exists(int n | file_stats(this, n, _) and result = n) }
    int  getNumStrings()   { exists(int n | file_stats(this, _, n) and result = n) }

    string toString() { result = "stats(" + getFile().getPath() + ")" }
  }

  // --- Containment helpers (INLINE forms, no shared predicate) ---------------

  /** Ident occurs inside Function's span (same file, start inclusive, end exclusive). */
  predicate identWithinFunction(Ident id, Function fn) {
    id.getFile() = fn.getFile() and
    (
      id.getStartLine() > fn.getStartLine() or
      id.getStartLine() = fn.getStartLine() and id.getStartCol() >= fn.getStartCol()
    ) and
    (
      id.getStartLine() < fn.getEndLine() or
      id.getStartLine() = fn.getEndLine() and id.getStartCol() < fn.getEndCol()
    )
  }

  /** String literal occurs inside Function's span. */
  predicate stringWithinFunction(StringLiteral s, Function fn) {
    s.getFile() = fn.getFile() and
    (
      s.getStartLine() > fn.getStartLine() or
      s.getStartLine() = fn.getStartLine() and s.getStartCol() >= fn.getStartCol()
    ) and
    (
      s.getStartLine() < fn.getEndLine() or
      s.getStartLine() = fn.getEndLine() and s.getStartCol() < fn.getEndCol()
    )
  }

  string importModule(Import i) { result = i.getUriUnquoted() }

  predicate hasSelfNamedIdentifier(Function fn) {
    exists(Ident id |
      identWithinFunction(id, fn) and
      id.getText() = fn.getName()
    )
  }
}
