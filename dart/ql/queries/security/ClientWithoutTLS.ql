/**
 * @name Flutter: Insecure HttpClient Usage
 * @description Detects HttpClient created without validating certificates or TLS callback.
 * @kind problem
 * @problem.severity error
 * @id dart/flutter-insecure-httpclient
 * @tags security, external/cwe/cwe-295
 */

import Dart

from Dart::Function c
where
  c.getName() = "HttpClient" and
  not exists(Dart::Ident flag |
    flag.getText().regexpMatch("badCertificateCallback")
  )
select c, "HttpClient created without TLS/SSL verification."
