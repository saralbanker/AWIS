# V-COMMON — Invariant Verification Block (immutable cache, EEOS §15 D5)
Cited by every `MXX-V1` verification card from M07 forward. Written once at EEOS v2.0
materialization (2026-07-08); encodes only frozen DoD/CI items (IMP §19/§20/§24; Constitution
Art. 32). Editing this file is an EEOS §13 amendment.

## Procedure (execute first, verbatim, before any milestone-specific item)
1. **Clean state:** fresh clone or worktree of the milestone branch; else `git status` must be
   empty. Record `HEAD: <sha>` in the report.
2. **Toolchain:** `go version` — record it.
3. **Core suite:** `make build` · `make test` · `make lint` · `make race` — each ✅/❌.
4. **Zero-AI gate (Constitution Art. 32, permanent from M06):** `make e1` — ✅/❌. If the
   verification card that cited this block also omitted any milestone-specific E1 impact item,
   that omission is itself a ❌ finding (AEO §18.3.4).
5. **Suite layers per IMP §19** applicable to the milestone: unit + contract always; integration/
   golden/system only where the V-card lists them.

## Report rules (AEO §18.3, restated by pointer)
Binary table only; a ❌ carries the exact command + failing output lines; flakes are ❌ with the
flake noted; benchmarks report measured vs target (CONTRA-2: 10ms engineering / 50ms gate);
total ≤800 tokens; verdict `PASS` requires every row ✅.
