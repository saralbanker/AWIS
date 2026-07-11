# M15 → M16/M17/M18 Handoff
**Status: STAGED — actuals at completion; G3 verdict + TDS-06 sign-off pend the founder.**

## Guaranteed outputs (contract)
- apps/oip: 4 handlers, embedded-runtime entrypoint (sdk-only), oip.db (B5 Option A),
  TDS-06 (DRAFT), rebuild-index; QG-3 green on NullAdapter; boundary evidence + G3 brief.
- sdk.LoadWorkflowFile (P0 seam, disclosed).

## What M16 may assume
- OIP's draft/synthesize intelligence steps upgrade transparently when a real adapter is
  configured (fallback paths already proven).

## What M17/M18 may assume
- `awis init` scaffolding can cite OIP as the reference app; QG-3 artifacts exist for G4.

## Actuals (filled at completion)
- P0: c3fba29 · C1: 80ef917 · C2: 6731297 · C3: 0ef33c8 · C3r: 0e452f8
- V1: PASS all rows (2026-07-11 at 9973a3f); QG-3 8/8; QG-4 diff = exactly P0+P1+P2 seams;
  fixtures byte-identical; e2e 32s green.
- Deviations/adjudications: fixture edit rejected+restored; P1 (RegisterPlugin, ns-scoped
  Submit fallback, --namespace) and P2 (engine fallback-join sentinel — frozen fixture
  exposed real AND-join defect) accepted as disclosed RB-mechanism seams. G3 verdict +
  TDS-06 sign-off PENDING FOUNDER (G3_BRIEF.md).
