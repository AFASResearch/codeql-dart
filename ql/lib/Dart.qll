// Low-level tables from your DB (names must match your dbscheme)
predicate file(File f) { file(f) }
predicate function(File f, Function n, int s, int e) { function(f, n, s, e) }
predicate call(Function caller, Function callee, File f, int l) { call(caller, callee, f, l) }

// 👇 Add this if your dbscheme has a string literal table:
predicate hasStringLiteral(File f, int line, string val) {
  stringLiteral(f, line, val)
}

// Existing classes (your originals, kept as-is)
class DartFile extends string { DartFile() { exists(File f | file(f) and this = f) } }

class DartFunction extends string {
  DartFunction() { exists(File f, Function n, int s, int e |
    function(f, n, s, e) and this = n) }
  File getFile() { result = f | exists(Function n, int s, int e |
    function(result, this, s, e)) }
}

class CallSite extends string {
  CallSite() { exists(Function caller, Function callee, File f, int l |
    call(caller, callee, f, l) and this = callee) }
}
