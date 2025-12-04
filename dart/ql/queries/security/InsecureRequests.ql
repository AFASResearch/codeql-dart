/**
 * @name Flutter: Hardcoded HTTP URLs
 * @description Detects insecure HTTP usage (`http://`) instead of HTTPS.
 * @kind problem
 * @problem.severity error
 * @id dart/flutter-hardcoded-http
 * @tags security, external/cwe/cwe-319
 */

import Dart

from Dart::StringLiteral s
where s.getText().regexpMatch("^\"?http://")
select s, "Insecure HTTP URL detected – use HTTPS instead."