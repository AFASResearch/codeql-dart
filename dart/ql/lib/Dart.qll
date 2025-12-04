module Dart {

  // --- Core types ------------------------------------------------------------

  class File extends @file {
    File() { file(this, _) }
    string getPath() { exists(string p | file(this, p) and result = p) }
    string toString() { result = getPath() }
  }

  class Function extends @function {
    Function() { functions(this, _, _, _, _, _, _) }

    File   getFile()      { exists(File    f  | functions(this, f, _, _, _, _, _) and result = f) }
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

    predicate hasSelfNamedIdentifier() {
      exists(Ident id |
        id.withinFunction(this) and
        id.getText() = this.getName()
      )
    }
  }

  class Ident extends @ident {
    Ident() { ident_occurs(this, _, _, _, _, _, _) }

    File   getFile()      { exists(File    f  | ident_occurs(this, f,  _, _, _, _, _) and result = f) }
    string getText()      { exists(string t  | ident_occurs(this, _,  t, _, _, _, _) and result = t) }
    int    getStartLine() { exists(int    sl | ident_occurs(this, _,  _, sl, _, _, _) and result = sl) }
    int    getStartCol()  { exists(int    sc | ident_occurs(this, _,  _, _, sc, _, _) and result = sc) }
    int    getEndLine()   { exists(int    el | ident_occurs(this, _,  _, _, _, el, _) and result = el) }
    int    getEndCol()    { exists(int    ec | ident_occurs(this, _,  _, _, _, _, ec) and result = ec) }

    string toString() { result = this.getText() }

    /** Ident occurs inside Function's span (same file, start inclusive, end exclusive). */
    predicate withinFunction(Function fn) {
      this.getFile() = fn.getFile() and
      (
        this.getStartLine() > fn.getStartLine() or
        this.getStartLine() = fn.getStartLine() and this.getStartCol() >= fn.getStartCol()
      ) and
      (
        this.getStartLine() < fn.getEndLine() or
        this.getStartLine() = fn.getEndLine() and this.getStartCol() < fn.getEndCol()
      )
    }
  }

  class StringLiteral extends @string_literal {
    StringLiteral() { string_literals(this, _, _, _, _, _, _) }

    File   getFile()      { exists(File    f  | string_literals(this, f, _, _, _, _, _) and result = f) }
    string getText()      { exists(string t  | string_literals(this, _, t, _, _, _, _) and result = t) }
    int    getStartLine() { exists(int    sl | string_literals(this, _, _, sl, _, _, _) and result = sl) }
    int    getStartCol()  { exists(int    sc | string_literals(this, _, _, _, sc, _, _) and result = sc) }
    int    getEndLine()   { exists(int    el | string_literals(this, _, _, _, _, el, _) and result = el) }
    int    getEndCol()    { exists(int    ec | string_literals(this, _, _, _, _, _, ec) and result = ec) }

    string toString() { result = this.getText() }

    /** String literal occurs inside Function's span. */
  predicate withinFunction(Function fn) {
    this.getFile() = fn.getFile() and
    (
      this.getStartLine() > fn.getStartLine() or
      this.getStartLine() = fn.getStartLine() and this.getStartCol() >= fn.getStartCol()
    ) and
    (
      this.getStartLine() < fn.getEndLine() or
      this.getStartLine() = fn.getEndLine() and this.getStartCol() < fn.getEndCol()
    )
  }
  }

  class Import extends File {
    Import() { exists(File f | f = this and imports(f, _, _, _, _, _, _)) }

    File   getFile()        { exists(File    f  | imports(f, _, _, _, _, _, _) and result = f) }
    string getPrefix()      { exists(string p  | imports(_, p, _, _, _, _, _) and result = p) }
    string getUriText()     { exists(string u  | imports(_, _, u, _, _, _, _) and result = u) }
    int    getUriStartLine(){ exists(int    sl | imports(_, _, _, sl, _, _, _) and result = sl) }
    int    getUriStartCol() { exists(int    sc | imports(_, _, _, _, sc, _, _) and result = sc) }
    int    getUriEndLine()  { exists(int    el | imports(_, _, _, _, _, el, _) and result = el) }
    int    getUriEndCol()   { exists(int    ec | imports(_, _, _, _, _, _, ec) and result = ec) }

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

  }

  class FileStats extends File {
    FileStats() { exists(File f | f = this and file_stats(f, _, _)) }

    File getFile() { result = this }
    int  getNumFunctions() { exists(int n | file_stats(this, n, _) and result = n) }
    int  getNumStrings()   { exists(int n | file_stats(this, _, n) and result = n) }

  }

  // --- Containment helpers (INLINE forms, no shared predicate) ---------------

  

  

  string importModule(Import i) { result = i.getUriUnquoted() }

  
}
