/**
 * @name Flutter: Dynamic Code Execution
 * @description Detects eval-like or dynamic execution that should be reviewed.
 * @kind problem
 * @problem.severity warning
 * @id dart/flutter-dynamic-execution
 * @tags security, external/cwe/cwe-94
 */

import Dart

from Dart::Function call
where call.getName().regexpMatch("eval|run|Isolate\\.spawnUri")
select call, "Dynamic execution – review carefully."
