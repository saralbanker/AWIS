# EDR-002 — CLI framework

**Decision:** `spf13/cobra`

**Rationale:** AWIS exposes a broad multi-level command tree (PRD §15: `submit`, `signal`, `status`, `trace`, `workflow validate|list|show`, `plugin …`, `config …`, and more); cobra's automatic help generation, persistent flags, and command-group model match that shape exactly. stdlib `flag` would require hand-rolling all of that scaffolding. The dependency is not added now — the require entry lands at M14 when the CLI is wired.

Decided at M00 per IMP §27.M0 risk note.
