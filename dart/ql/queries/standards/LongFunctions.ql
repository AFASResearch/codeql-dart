/**
 * @name Dart: Long functions
 * @description Flags functions whose span exceeds a configurable line threshold.
 * @kind problem
 * @problem.severity recommendation
 * @id dart/long-function
 * @tags maintainability
 */

import Dart

/**
 * Minimum number of lines a function can have before it is considered "long".
 * Adjust to taste.
 */
predicate isLongFunction(Dart::Function f) {
  f.getEndLine() - f.getStartLine() >= 50
}

from Dart::Function f
where isLongFunction(f)
select f,
  "Long function (" +
  (f.getEndLine() - f.getStartLine()).toString() +
  " lines): " + f.getName()
