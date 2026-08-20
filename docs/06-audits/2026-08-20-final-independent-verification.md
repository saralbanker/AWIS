# Final Independent Verification — the claimed shortest path to M18 A-INIT

**Date:** 2026-08-20 · **Scope:** the claimed A-INIT path only, not a repository re-audit.
**Method:** EXEC > QUOTE > TRACE > INTERP. Every prior conclusion was treated as a hypothesis
and re-derived. Measurements were taken on clean worktrees of `origin/m17-full-cli-init`
(49ce489) and `origin/m17-c3-wip` (153772c).

**Headline:** the path is directionally right and its two most-quoted claims survive execution.
It is nevertheless **NOT VALIDATED**, because M17-C3's real scope is roughly three times the
preserved WIP, and two checklist rows that gate M17-V1 are currently unsatisfiable for reasons
that have nothing to do with C3's scaffold.

---

## Deliverable 1 — A-INIT Path Validation

| # | Step | Evidence | Conf. | Hidden dependency | Blocker class |
|---|---|---|---|---|---|
| 1 | M17-C3 is incomplete on `m17-full-cli-init` | EXEC — no `init.go`, no `scaffold/` in `cmd/awis/` | Certain | — | TRUE, open |
| 2 | `m17-c3-wip` ∥ `m17-full-cli-init` are siblings | EXEC — both diverge at `a29076b` (1 vs 2 commits) | Certain | — | Confirmed |
| 3 | Version-golden is the only failing test | EXEC — `make verify` reaches `test`; only `TestVersionHuman`/`TestVersionJSON` fail | Certain | Toolchain-environmental (golden pins go1.26.5) | TRUE, trivial |
| 4 | The fix already exists | EXEC — `d5ac004` cherry-picks onto the M17 lineage with **zero conflicts** | Certain | — | Solved |
| 5 | Porting the WIP yields green gates | EXEC — port 9 files + `all:scaffold` + test-path + `-update` + `d5ac004` → **`verify: ALL GATES PASSED` (exit 0)** | Certain | — | Measured |
| 6 | C3's *documentation* deliverable | EXEC — `m17-c3-wip` contains **no docs changes**; `docs/CLI.md` and `docs/CLI_CONTRACT.md` are still M14-era | Certain | **Covers C1/C2's commands, not just init** | **TRUE, omitted** |
| 7 | C3's rehearsal system test | EXEC — `grep -r rehearsal` hits only module docs; no test file exists | Certain | Card ACCEPTANCE + checklist row 5 | **TRUE, omitted** |
| 8 | M17-V1 can then pass | TRACE — checklist rows 3 and 6 fail today for C1/C2 reasons | High | See §3 | **Blocked by 6 & 7** |
| 9 | M17 D-CLOSE | QUOTE — EEOS D-CLOSE = semantic review + **HANDOFF actuals** + **PR + evidence** | High | M17 module lacks 4 of 7 contract files; **zero PRs have ever existed in this repo** | **TRUE, open** |
| 10 | M18 A-INIT opens from M17 D-CLOSE | QUOTE — EEOS D-CLOSE fuses next A-INIT at a non-gated boundary; IMP §"Gates G2–G4 … do not idle the pipeline"; M17 carries no gate | High | — | **Correct** |
| 11 | M15/M16 merges do not gate M18 A-INIT | QUOTE + EXEC — branch stacking already carries M15+M16 code; G3 does not idle the pipeline | High | Does not extend to M18 *completion*: G3/TDS-06 gate the OIP dogfood week | **Correct, for A-INIT only** |

### Verdict per claim under test

- **"M17-C3 is the only remaining B-BUILD item"** — true as stated about *cards*, misleading about
  *work*. C3's card charters `docs/CLI.md + CLI_CONTRACT.md (complete them)`, and neither has been
  touched since M14 even though C1 and C2 shipped ~13 commands.
- **"The version-golden mismatch is the only currently observed failing test"** — **CONFIRMED and
  strengthened**: it is the only failing row in the entire `verify` chain. `gofmt`, `vet`, `lint`
  (0 issues, both modules, golangci-lint v2.12.2), `oip-isolation`, `build`, `e1` all pass, and
  `pytest` is 41/41 green.
- **"P2 does not fully solve the workflow hang; the same pattern exists for StepFallbackActivated
  and StepFailed(retrying=false)"** — **half wrong as written.** See §3.

---

## Deliverable 2 — Missing Evidence Register

| # | Assumption | Status |
|---|---|---|
| E1 | Any milestone branch has CI evidence | **REFUTED.** 8 CI runs exist in total: 3 on `main` (run 1 **failed**), 5 on two audit branches. **Zero runs on m10–m17.** Every milestone PASS is self-reported. |
| E2 | M17-V1 will be verified on a clean tree with real gates | **UNSUPPORTED.** No CI has run on this branch; V-COMMON row 1 also requires a **pytest regression** that CI explicitly does not wire. |
| E3 | Intermediate merge states are green | **REFUTED.** `main` ↔ `m17-full-cli-init` trial merge **conflicts** in `.github/workflows/ci.yml` and `Makefile`. |
| E4 | `m17-c3-wip` "does NOT build green" (its own commit message) | **REFUTED.** `go build ./...` exits 0. Only its *tests* failed, for two shallow reasons. |
| E5 | `main`'s STATE.md reflects the repository | **REFUTED.** It says M10–M14 await founder merge; `f6aa755` merged them. A live ledger-repo contradiction. |
| E6 | M16 is genuinely DONE | **UNSUPPORTED.** Its checklist ticks `V-COMMON all ✅` but no CI ran, and its module is missing 4 of 7 contract files, which D-CLOSE is supposed to produce. |
| E7 | M17-V1's card is well-formed | **PARTLY REFUTED.** Of 11 V1 cards (M07→M17), **only M16-V1 and M17-V1 omit the V-COMMON citation** that EEOS rule 5 and V-COMMON's own header require. Substance survives via checklist row 1; the card-level omission is a real deviation and a regression that begins exactly at M16. |
| E8 | The G4/QG-1..5 "clean macOS + Linux" requirement is reachable from the stack | **UNSUPPORTED.** The macOS matrix leg exists only on `main` (98350f6) and not on the M17 stack. |

---

## Deliverable 3 — Deferred Work Validation

| Item | Ruling | Basis |
|---|---|---|
| **EDR-007 amendment** | **Correctly deferred** for A-INIT — but **misscoped as briefed** | Exposure is latent: **no shipped workflow uses `on_error`** (EXEC grep over all 6 YAMLs). But see §3 — the brief names the wrong event. |
| **docs-lint** | **Partly incorrectly deferred** | M16/M17 are genuinely missing files (creatable). **M15 is different**: it fails because `BOUNDARY_EVIDENCE.md` and `G3_BRIEF.md` are *extra*, yet both are **G3-mandated evidence** (IMP: "the boundary demo … is Gate G3's evidence"). The 7-file contract cannot express them. That is an amendment/relocation **decision**, not cleanup. |
| **STATE.md reconciliation** | **INCORRECTLY deferred** | EEOS phase table: *"ledger contradicts repo → STOP — Fable wake; never 'fix' STATE to match a guess."* `main`'s STATE is false **now**. This is a protocol STOP condition, not backlog. |
| **Contract files** | **INCORRECTLY deferred for M17** | `HANDOFF.md`/`TRACEABILITY.md` are **D-CLOSE outputs**, and M18 A-INIT opens *from* D-CLOSE. They are on the path, not behind it. |
| **Merge preparation** | **Correctly deferred** | E-MERGE is a separate human phase; A-INIT does not require it. |

---

## Deliverable 4 — Critical Path Reconstruction (from first principles)

Rebuilt from EEOS's phase machine, not from any prior plan. Everything below happens on
`m17-full-cli-init` — the branch the immutable M17-V1 card already names.

**Phase B-BUILD — close M17-C3** (re-dispatch with P1+**P2 salvage preamble**; STATE shows C3
`DISPATCHED`, so this consumes the single permitted re-dispatch — a second death is `STOPPED` + CE audit)

1. Port the 9 preserved files from `m17-c3-wip`.
2. `//go:embed scaffold` → `//go:embed all:scaffold` (Go excludes dotfiles; `scaffold/.gitignore` is silently dropped).
3. Fix the golden source path in `init_test.go`: `../../../../examples/workflows/` → `../../`.
4. `go test ./cmd/awis/ -run TestInitGolden -update` to create `init.{txt,json}`.
5. Cherry-pick `d5ac004` (version goldens; clean).
6. **Write the `init → start → submit → trace` rehearsal system test** (does not exist).
7. **Write the TDS-07 §4 per-command contracts for every M17 command** — `history, logs, metrics, recall, replay, audit, plugin status, plugin remove, config show|set|validate|edit, rebuild-state, export, prune-events, init` — and flip §3's "M17 — Planned Commands (not yet implemented)".
8. **Rewrite `docs/CLI.md`** for the complete PRD §15 tree; delete its "Upcoming Commands (M17) … not yet implemented" section.
9. Commit once; STATE → C3 DONE.

Steps 1–5 are mechanical and **measured green**. Steps 6–8 are the real work and are **untouched**.

**Phase C-VERIFY**
10. Materialize M17's 4 missing contract files (`AI_EXECUTION_CONTEXT`, `DEPENDENCY_MAP`, `HANDOFF`, `TRACEABILITY`) — HANDOFF finalizes at D-CLOSE, the rest are A-INIT debt.
11. Dispatch M17-V1 on a clean tree. Note its card omits the V-COMMON citation; checklist row 1 carries it, and the verifier must run `build/test/lint/race/e1` **plus pytest**.
12. All-✅ → STATE → D.

**Phase D-CLOSE** *(this is where M18 A-INIT is born)*
13. Fable semantic review of the full M17 diff.
14. HANDOFF actuals.
15. **Open the PR + evidence.** No PR has ever existed in this repository; this mechanism is unexercised.
16. Non-gated boundary ⇒ **run M18 A-INIT**: branch, materialize the M18 module, write all cards READY, STATE → B.

**Off the A-INIT path, but required before M18 can *finish*:** founder G3 verdict + TDS-06 sign-off
(gates the OIP dogfood week), the three squash merges, the `ci.yml`/`Makefile` merge conflicts, the
macOS matrix leg, and the EDR-007 amendment.

---

## Deliverable 5 — Final Verdict

### NOT VALIDATED

*(≈430 words)*

The path's **architecture** is sound and I confirm its two load-bearing claims. M18 A-INIT does
open from M17 D-CLOSE, needs no merge and no founder: EEOS fuses the next A-INIT into D-CLOSE at a
non-gated boundary, M17 carries no gate, and IMP states Gates G2–G4 "do not idle the pipeline."
Branch stacking already carries M15+M16 code, so those merges genuinely do not gate A-INIT. And
the version-golden claim is not merely true but understated — it is the only red row in the whole
`verify` chain, and `d5ac004` cherry-picks cleanly.

It fails validation on **scope**, in one specific and consequential way.

`m17-c3-wip` preserves a scaffold: 9 files, 755 lines. Ported with two one-line fixes, generated
goldens and that cherry-pick, it reaches `verify: ALL GATES PASSED` — I measured this rather than
projecting it. That result is exactly what makes the plan dangerous, because green gates here do
**not** mean C3 is done. The M17-C3 card also owns `docs/CLI.md + CLI_CONTRACT.md (complete them)`
and an `init → start → submit → trace` rehearsal system test. **Neither exists.** TDS-07 — the
*normative* CLI contract — still headers its M17 section "Planned Commands (not yet implemented)"
and carries per-command contracts for M14 commands only, while `docs/CLI.md` still tells readers
the M17 commands "are not described here." C1 and C2 are marked DONE having shipped roughly
thirteen commands whose normative contracts were never written. M17's VALIDATION_CHECKLIST row 3
demands a TDS-07 section for **every** one of them. That row is unsatisfiable today, and C3 — not
C1 or C2 — is chartered to fix it. So "M17-C3 is the only remaining B-BUILD item" is true about
cards and misleading about effort: the preserved WIP is the smaller half.

Two further corrections. **The brief's engine claim is half wrong.** I reproduced the stall and
its mirror: `StepFallbackActivated` **is** fixed — P2's sentinel works, and that probe completes.
The live hole is `StepFailed(retrying=false)` reached via the **`on_error`** route, the one of
three terminal routes that writes no sentinel. The failed step then stays permanently in
`activatableFor`, which blocks tick.go's completion check outright — it strands joins *and* hangs
join-free workflows. Amending EDR-007 for `StepFallbackActivated`, as the brief's bullet implies,
would patch a working row and leave the defect live. Exposure is latent (no shipped workflow uses
`on_error`), so deferral is right; the *scope* is not.

Second, **`main`'s STATE.md is false now** — it says M10–M14 await merge; `f6aa755` merged them.
EEOS makes that a STOP, not backlog.

**Conditions to upgrade to VALIDATED:** re-scope C3 to include TDS-07 + CLI.md + the rehearsal
test; correct the EDR-007 amendment to `StepFailed(retrying=false)`/`on_error`; reconcile `main`'s
ledger; and treat M17's four contract files as D-CLOSE work, not deferred cleanup.
