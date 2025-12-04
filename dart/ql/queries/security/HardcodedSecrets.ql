/**
 * @name Flutter: Hardcoded Secrets
 * @description Detects possible hardcoded API keys, tokens, or passwords.
 * @kind problem
 * @problem.severity error
 * @id dart/flutter-hardcoded-secrets
 * @tags security, external/cwe/cwe-798
 */

import Dart

from Dart::StringLiteral s
where s.getText().regexpMatch("(?i)(api[_-]?key|secret|token|pwd|password)")
select s, "Possible secret in code: " + s.getText()
