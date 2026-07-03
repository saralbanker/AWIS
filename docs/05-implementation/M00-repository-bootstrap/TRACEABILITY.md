# M00 — Traceability
| Task / artifact | Canonical source | Coordinate |
|---|---|---|
| git init, branch, .gitignore | IMP | §4 item 1 |
| go.work two-module workspace | IMP | §4 item 2, §5 (structure + rationale block) |
| Module path `github.com/awis/awis/sdk` | IMP | §25 CONTRA-1 disposition |
| QG-4 mechanical enforcement via module split | IMP §5; PRD | QG-4; Blueprint P1/P5, FR-SDK-09 |
| Makefile 8 targets | IMP | §4 item 3 |
| CI skeleton, both OSes | IMP | §4 item 4, §21 |
| e1 gate slot visible from day one | IMP §21; Constitution | Article 32 (AWIS-E1, permanent from M6) |
| docs/ + CODEBASE.md + boundary README | IMP | §4 item 5, §22 |
| Toolchain pinning, pyproject stubs | IMP | §4 item 6 |
| EDR: CLI framework + SQLite driver now | IMP | §27.M0 Risk note |
| Pure-Go SQLite (modernc.org/sqlite, no CGO) | Blueprint | §20 (storage stack); PRD NFR-Portability |
| Secret rules bind from first commit | PRD | §30 config/secrets; Constitution security articles |

**Requirements satisfied:** none directly (M00 is substrate). **Requirements enabled:** all.
**Contradictions touched:** CONTRA-1 (recorded in edr-001; disposition applied, not re-decided).
