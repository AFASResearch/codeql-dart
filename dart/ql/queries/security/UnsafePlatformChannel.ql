/**
 * @name Flutter: Unsafe Platform Channels
 * @description Detects hardcoded MethodChannel names which should be centralized.
 * @kind problem
 * @problem.severity recommendation
 * @id dart/flutter-unsafe-platform-channel
 * @tags security, maintainability
 */

import Dart

from Dart::StringLiteral s
where
  s.getText().regexpMatch("^\"[A-Za-z0-9_.]+\"$") and
  exists(Dart::Ident id | id.getText() = "MethodChannel")
select s, "Hardcoded MethodChannel name – consider moving this to a shared constant."
