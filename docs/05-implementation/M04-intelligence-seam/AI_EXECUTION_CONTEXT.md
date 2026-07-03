# M04 — AI Execution Context

**Model allocation (IMP §28):** Sonnet — "Interface transcription + straightforward routing tree."
No Opus review row; CE (Fable) reviews all diffs per AEO §4 (EDR-004 mapping).

**Builder cards:** M04-C1 (types + NullAdapter + contract suite), M04-C2 (router + budget + dispatcher).
**Verifier card:** M04-V1 (Sonnet, clean clone, read-only).

**Context to load per card (token hygiene):** this module's 7 files + Blueprint §13–§17 excerpt
embedded in the card + `internal/core/ports.go` + `internal/core/step.go` (IntelReq). Never the IMP.

**Frozen constraints binding every card:**
- Architecture and G1 formats frozen; PRD frozen. No redesign, no scope expansion.
- IntelligencePort method set is frozen verbatim (M01) — adapters implement it identically.
- "The record is silent." is a frozen literal (Blueprint §14, Constitution-adjacent voice).
- CONTRA-3: Embed unavailable in V1; zero-vector null response; no OpenAI delegation code.
- Toolchain: `export PATH=$HOME/toolchains/go/bin:$HOME/toolchains/bin:$PATH`; CGO_DISABLED; no sudo.
- Builders do NOT commit; CE commits after review.
