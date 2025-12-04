/**
 * @name Dart: Functions with many string literals
 * @description Flags functions that contain many string literals, which may indicate magic strings or missing i18n.
 * @kind problem
 * @problem.severity recommendation
 * @id dart/many-string-literals
 * @tags maintainability
 */

import Dart

/**
 * Count of string literals in a function.
 */
predicate stringCountInFunction(Dart::Function f, int count) {
  count = strictcount(Dart::StringLiteral s | s.withinFunction(s, f))
}


from Dart::Function f, int c
where stringCountInFunction(f, c) and c >= 10
select f,
  "Function contains " + c.toString() +
  " string literals – consider extracting constants/i18n."
