# Traceability Matrix — requirement families → milestones
Source: IMP §8 (capability breakdown) + §27, amended F-1..F-5. Full per-requirement detail lives in each milestone's TRACEABILITY.md at materialization.

| Requirement family | Capability | Milestone(s) |
|---|---|---|
| FR-ST-01..06 | Durable event recording, migrations, rebuild | M02, M03 |
| FR-SE-*, FR-WE-07..11 | Deterministic execution, retry/fallback/compensation/cancellation | M06 |
| Trigger subsystem (SCAN_TRIGGERABLE, DomainEvents, `domain_events` TTL 7d) | **F-2** | M06 (+SDK intake M08, cron M17) |
| FR-WE-05/06, FR-SE-12/13 | WAIT/signal, atomic delivery | M07 (Gate G2) |
| NFR-S-05, FR-OB-08 | audit_log + append API | **M07 per F-4** (write sites M7/M8/M12) |
| FR-SDK-* | Go workflow definition, public surface | M01 (types, F-1), M08 |
| FR-SDK-06/07/08, QG-5 | Deterministic testing harness | M09 |
| FR-WD-* | YAML DSL + validation | M05 (grammars/validator), M10 (parser) |
| FR-SE-02, FR-SDK-10 | Polyglot subprocess steps, awis-step lib | M11 |
| FR-PS-* | Plugin system, JSON-RPC, awis-plugin lib | M12, M13; **minimal `plugin install/list` at M14 per F-5** |
| FR-RM-*, FR-OB-* | CLI operate/debug | M14 (core, TDS-07 day 1 per **F-3**), M17 (full) |
| FR-IL-02/05..09 | Anthropic adapter | M16 (seam at M04) |
| FR-IL-11 | Recall FTS over execution history | M17 (migration 0005) |
| QG-3, QG-4 | OIP on AWIS, boundary verdict | M15 (Gate G3; TDS-06 human sign-off) |
| QG-1..5, NFR benchmarks | Release | M18 (Gate G4) |

**Contradiction dispositions:** CONTRA-1 (module path→edr-001/M00), CONTRA-2 (10ms target/50ms gate→M18 bench), CONTRA-3 (no Embed; FTS pass-through→M15/M16), CONTRA-4 (audit/FTS additive migrations→M07/M17).
