/**
 * @name Hardcoded secrets in string literals
 * @kind problem
 * @id dart/hardcoded-secrets
 * @problem.severity warning
 */

import Dart
import semmle.util.regex

predicate looksLikeSecret(string s) {
  s.regexpMatch("(?i)(api[_-]?key|secret|token)") or
  s.regexpMatch("AKIA[0-9A-Z]{16}")
}

from string f, int line, string val
where hasStringLiteral(f, line, val) and looksLikeSecret(val)
select f, "Line " + line.toString() + ": possible hardcoded secret: " + val
