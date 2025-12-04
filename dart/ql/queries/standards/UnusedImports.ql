/**
 * @name Dart: Suspicious unused imports
 * @description Flags imports whose prefix or last path segment does not appear as an identifier.
 * @kind problem
 * @problem.severity recommendation
 * @id dart/unused-import-heuristic
 * @tags maintainability
 */

import Dart

/**
 * True if this identifier plausibly refers to this import.
 *
 * - Either the import has a prefix matching the identifier
 * - Or the last segment of the URI (after the final "/") equals the identifier text.
 */
predicate importReferencedByIdent(Dart::Import imp, Dart::Ident id) {
  // Case 1: prefix
  imp.getPrefix() = id.getText()
  or
  // Case 2: last path segment of the URI equals the identifier
  exists(string before |
    imp.getUriUnquoted() = before + id.getText() and
    not id.getText().regexpMatch(".*/")
  )
}

from Dart::Import imp
where not exists(Dart::Ident id | importReferencedByIdent(imp, id))
select imp,
  "Import may be unused (no identifiers reference its prefix or module name)."
