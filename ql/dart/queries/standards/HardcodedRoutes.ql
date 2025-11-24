/**
 * @name Flutter: Hardcoded route names
 * @description Finds string literals that look like route names (start with '/').
 * @kind problem
 * @problem.severity recommendation
 * @id dart/flutter-hardcoded-routes
 * @tags maintainability
 */

import Dart

from Dart::StringLiteral s
where
  s.getText().regexpMatch("^['\"]/[A-Za-z0-9_/]*['\"]$")
select s,
  "Hardcoded route name: " + s.getText()