# M06 — Execution Engine Core
**Status:** Partitioned — materialize at entry · **Effort:** 4d · **Window:** Week 2 · **Phases:** A, B (stacked PRs)
**Phase A:** pull loop (SCAN→CLAIM→DISPATCH→SETTLE, 100ms tick), NativeRunner + IntelligenceRunner dispatch, transition evaluation incl. fan-out within max_parallel, idempotency-key check, structured JSON logging.
**Phase B:** RetryPolicy + backoff, fallback activation, compensation (reverse order, own retries, compensation_failed), cancellation FSM verbatim from Finalization Blocker 4, **[F-2] SCAN_TRIGGERABLE + DomainEvent trigger matching + `domain_events` table (TTL 7d, additive migration entry)**.
**Depends on:** M02, M03, M04, M05 · **Blocks:** M07, M08, M11
**Primary sources:** IMP §27.M6; Blueprint §8 (full loop incl. step 2), §10 (DomainEvents); Finalization Blocker 4; Verification F-2 · **Compilation spec:** IKB §4/M06
**Key ACs:** AWIS-E1 zero-AI gate LIVE in CI and green (permanent); `-race` clean; every Blueprint §9 event type emitted; cancellation event sequence matches Finalization fixture.
