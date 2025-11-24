import Dart

/**
 * @name Insecure HTTP usage
 * @description Flags string literals/imports that contain "http://" (non-TLS).
 * @kind problem
 * @problem.severity warning
 * @id customdb.insecure-http
 * @tags security external/cwe/cwe-319
 */

from Dart::File f, int line, int elt, string msg
where
  // String literals containing "http://"
  exists(Dart::StringLiteral lit |
    f = lit.getFile() and
    regexpMatch(lit.getText(), "http://") and
    line = lit.getStartLine() and
    elt = lit and
    msg = "Insecure HTTP literal: consider using HTTPS."
  )
  or
  // Imports whose (unquoted) URI contains "http://"
  exists(Dart::Import i |
    f = i.getFile() and
    regexpMatch(i.getUriUnquoted(), "http://") and
    line = i.getUriStartLine() and
    elt = i and
    msg = "Insecure HTTP import: " + i.getUriUnquoted()
  )
select f, line, elt, msg
