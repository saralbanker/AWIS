# M06 — AI Execution Context

**Model allocation (IMP §28):** **Opus** — "Highest implementation complexity: concurrency, FSMs,
compensation/cancellation semantics." ALL builder cards run on Opus (no downward substitution,
EDR-004). Verifier: Sonnet.

**Builder cards (Phase A / Phase B per README):**
- M06-C1 (Opus): core shape completions + engine core — emit/projection mirror, submit, tick
  SCAN/CLAIM/DISPATCH/SETTLE, NativeRunner, transition evaluation + fan-out/join (EDR-011),
  idempotency, slog JSON, happy-path fixtures.
- M06-C2 (Opus): IntelligenceRunner + retry/backoff (ADJ-6) + fallback + failure→compensation +
  cancellation FSM (B4 verbatim) + those event-sequence fixtures + forward≡rebuild equivalence.
- M06-C3 (Opus): F-2 triggers (migration 0002, DomainEvent, SCAN_TRIGGERABLE, TTL prune) +
  E1 gate wiring + remaining fixtures.
**Verifier card:** M06-V1 (Sonnet, clean clone).

**Context per card:** this module's 7 files + docs/EVENTLOG_FORMAT.md (TDS-01+ADJ-8) +
docs/WORKFLOW_SCHEMA.md (TDS-02+ADJ-6/7) + docs/edr/edr-005..011 + Finalization B4 excerpt
(embedded in C2's card) + the relevant internal package sources. Never the IMP.

**Frozen constraints binding every card:**
- EDR-007 §2 status map is the FORWARD-PATH SPEC: applyEvent mirrors it exactly; fixtures prove
  forward ≡ rebuild.
- EDR-005: the ENGINE assigns sequence_num; storage only guards monotonicity.
- CONTRA-5: WorkflowCancelled payload = {reason} ONLY. CONTRA-6: claiming never touches status.
- StoragePort's 12 methods are frozen — new storage needs = additive SQLiteStorage methods.
- B4 cancellation semantics verbatim, including the warning string and the event-sequence fixture.
- Constitution Art. 32: E1 tests run with intelligence entirely absent.
- Toolchain: `export PATH=$HOME/toolchains/go/bin:$HOME/toolchains/bin:$PATH`; builders do NOT
  commit; frozen-text gap/conflict ⇒ STOP (CONTRA protocol) — ADJ-6/7/8 + CONTRA-10 are already
  adjudicated; do not invent others silently.
