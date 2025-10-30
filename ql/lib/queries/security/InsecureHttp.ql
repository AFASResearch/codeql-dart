/**
 * @name Insecure HTTP usage
 * @description Uses http:// URIs that may expose data
 * @kind problem
 * @problem.severity warning
 * @id dart/insecure-http
 */

import Dart

from string lit, File f, int line
where
  stringLiteral(f, line, lit) and
  lit regexp "^http://"
select f, line, "Insecure HTTP URL: " + lit
