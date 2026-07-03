# 06 — Reference: Normative Technical Design Specifications

TDS files are CREATED BY milestones (IMP §12) and live here. Path mapping (recorded, not silent):
IMP §12 names paths like `docs/EVENTLOG_FORMAT.md`; in this repository they resolve to
`docs/06-reference/<NAME>.md`. TDS-06 (OIP Record format) lives in `apps/oip/docs/RECORD_FORMAT.md`
because it is OIP-owned (Blocker 5 boundary).

| TDS | File (here) | Created in | Consumed by |
|---|---|---|---|
| TDS-01 | EVENTLOG_FORMAT.md | M01 | M02, M03, M06, M07 |
| TDS-02 | WORKFLOW_SCHEMA.md | M01 | M02, M08, M10 |
| TDS-03 | EXPRESSION_GRAMMARS.md (+ conformance corpus) | M01 | M05, M10 |
| TDS-04 | SUBPROCESS_PROTOCOL.md | M11 | M11, python/awis-step |
| TDS-05 | PLUGIN_PROTOCOL.md | M12 | M12, M13, python/awis-plugin |
| TDS-06 | ../../apps/oip/docs/RECORD_FORMAT.md | M15-A (human sign-off) | M15 |
| TDS-07 | CLI_CONTRACT.md (incl. F-3 CLI↔runtime interaction model) | M14 | M14, M17 |

Empty until the creating milestone runs. A missing TDS here means its milestone has not executed.
