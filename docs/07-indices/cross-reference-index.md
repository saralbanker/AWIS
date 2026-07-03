# Cross-Reference Index — topic → canonical coordinates
| Topic | Where specified |
|---|---|
| Six core structures | Blueprint §6; types frozen at M01 (`internal/core`, F-1) |
| EventLog / event types / envelope | Blueprint §9 → TDS-01 (M01) |
| StateStore projection + rebuild | Blueprint §9; IMP §26 (rollback); M03 |
| Expression grammars + prohibited lists | Finalization Blocker 2 → TDS-03 (M01) |
| Signal atomicity (single txn, delivered_at guard, version lock) | Finalization Blocker 3; IMP IR-3; M07/G2 |
| Cancellation semantics (graceful, --compensate opt-in) | Finalization Blocker 4; M06 |
| FTS ownership split (oip.db vs runtime.db) | Finalization Blocker 5; CONTRA-4; migrations 0004/0005 |
| Compensation vs append-only | Finalization Blocker 6; M06/M15 |
| Parallel via fan-out transitions (no Parallel StepType) | Finalization Blocker 7 |
| Plugin protocol (JSON-RPC 2.0, lifecycle FSM, ≤3 restarts) | Blueprint §11 → TDS-05 (M12 day 1) |
| Subprocess protocol | Blueprint §25 → TDS-04 (M11 day 1) |
| CLI↔runtime interaction model | **TDS-07 (M14 day 1) per F-3**; Blueprint §28 socket vs SQLite-WAL; §9 "no concurrent writers" refinement if latter |
| OIP Record format (frontmatter, corrects:, append-only) | Constitution Title II; Blueprint §10/§18 → TDS-06 (M15, human sign-off) |
| Trigger subsystem | **F-2**: M06 scan + domain_events (TTL 7d); M08 TriggerAPI; M17 cron |
| Migration sequence 0001–0005 | IMP §14 (renumbered per F-4) |
| Repo structure / two modules / QG-4 mechanics | IMP §5 |
| Model allocation | IMP §28 |
| Risks IR-1..7 + CONTRA-1..4 | IMP §25 |
| Secrets (env-only; masked in config show; never in logs/EventLog/export) | PRD §30 + security NFRs |
| AWIS-E1 zero-AI gate | Constitution Art. 32; live from M06, permanent |
