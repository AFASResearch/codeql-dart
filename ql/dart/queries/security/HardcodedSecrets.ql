/**
 * @name Hardcoded secret or credential
 * @description Detects probable hardcoded credentials in Dart.
 * @kind problem
 * @problem.severity error
 * @tags security
 */
import Dart

from StringLiteral s
where looksSecretishValue(s.getText())
select s.getFile(), s.getSL(), s.getSC(), s.getEL(), s.getEC(),
  "Possible hardcoded secret: " + s.getText()

union

from IdentOccur id, StringLiteral s
where id.getFile() = s.getFile()
  and looksSensitiveName(id.getText())
  // keep it practical: only flag if a “suspicious” identifier is near a long-ish literal
  and s.getText() regexp "'[^']{12,}'"
  and abs(id.getSL() - s.getSL()) <= 5
select s.getFile(), s.getSL(), s.getSC(), s.getEL(), s.getEC(),
  "Identifier '" + id.getText() + "' near suspicious literal " + s.getText()
