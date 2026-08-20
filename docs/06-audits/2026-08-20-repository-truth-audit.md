# AWIS — Repository Truth Audit
**Date:** 2026-08-20 · **Scope:** repository / EEOS / CI truth · **Method:** independent re-derivation from git, GitHub Actions API, and executed gates. No prior report trusted.
**Audit tree:** `main` @ `98350f6`

---

## 1. Executive Summary

The repository's **code** is in better shape than its **records**. `main` @ `98350f6` is genuinely
green on Linux and macOS — I confirmed this at job-step granularity, including a lint step that
really executed. Everything merged into main is lint-clean today.

Everything *outside* main is unproven. The decisive finding of this audit:

> **GitHub Actions has run exactly 3 times in this repository's entire history, and all 3 runs were
> on `main`.** No milestone branch — m15, m16, m17, m17-c3-wip — has ever been executed by CI.

Every `PASS` verdict recorded for M10–M16 was therefore self-reported from a local machine where,
as the repository itself admits, the lint gate could not run. This is not a hypothesis: the
stabilization commit `be13cf9` states it in its own message, and I reproduced the failing gate.

Three of the previous audit's five headline claims survive re-verification; two do not:

| Claim | Verdict |
|---|---|
| R2 — V1 cards contain PASS ticks for lint that never ran | **CONFIRMED** (two independent proofs) |
| R3 — golangci-lint cannot execute | **CONFIRMED but mis-scoped** — local only, not CI; ~90-second fix |
| R4 — two stabilization lines, "zero patch equivalence" | **PARTIALLY FALSE** — 6 of 8 files identical, 2 differ, 19 files exist only on the branch |
| M16 — rebaselines onto main clean, all gates pass | **FALSE AS STATED** — `make lint` fails; the evidence list omitted the one gate that fails |
| M17 — two root causes for C3 | **CONFIRMED but INCOMPLETE** — both real; **two further failure classes** were missed |

The single highest-value next action is **rebaselining M16 onto main**, because it is the only
milestone with zero founder dependencies, and I have proven end-to-end that it reaches
`verify: ALL GATES PASSED` — but only after a lint fix the previous plan did not include.

---

## 2. Verified Facts

Each fact below was produced by a command run during this audit.

### F-1 — Branch topology matches the briefing exactly
`git ls-remote --heads origin`:

| Branch | SHA | Claimed | Match |
|---|---|---|---|
| main | `98350f6` | 98350f6 | ✅ |
| m15-oip-on-awis | `09ca0cb` | 09ca0cb | ✅ |
| m16-anthropic-adapter | `89a6cb8` | 89a6cb8 | ✅ |
| m17-full-cli-init | `49ce489` | 49ce489 | ✅ |
| m17-c3-wip | `153772c` | 153772c | ✅ |

All four milestone branches share merge-base `827a084` (M14 D-CLOSE) and are **10–19 commits behind main**.
`m17-c3-wip` is exactly **one commit** ahead of `m17-full-cli-init` (`153772c`, self-labelled
"does NOT build green — do not merge"). It is intact and was not modified by this audit.

### F-2 — main is genuinely green (run `32339388733`, all 3 jobs `success`)
- `Verify (ubuntu-latest)` — 13 steps, all success; **step 8 `golangci-lint` ran and passed** (06:26:02→06:26:18)
- `Verify (macos-latest)` — all success
- `EEOS docs-lint` — success

The macOS leg is real, not assumed. Both platform claims in the briefing hold.

### F-3 — CI has never run on any milestone branch ⚠️ *primary finding*
`actions_list(ci.yml)` returns `total_count: 3`:

| Run | Branch | SHA | Conclusion |
|---|---|---|---|
| 1 | main | dcc6f24 | **failure** (version goldens) |
| 2 | main | d5ac004 | success |
| 3 | main | 98350f6 | success |

`ci.yml` exists on every milestone branch (since M00) and triggers on `push: branches: ["**"]`,
so the absence is not a configuration gap in the branch files — those branches simply have never
been pushed under an active Actions configuration. The consequence is what matters: **no milestone
branch has any CI evidence whatsoever.**

This is independently corroborated inside the repository. `be13cf9`'s commit message:

> "Drift had grown monotonically 7 -> 22 files across M10..M17 because gofmt is a CI-only gate and **CI has never executed**."

### F-4 — R3: lint is broken locally, functional in CI
```
$ golangci-lint --version   → 2.5.0, built with go1.25.1
$ go version                → go1.26.4       (go.mod: go 1.26, toolchain go1.26.4)
$ golangci-lint run ./...   → "the Go language version (go1.25) used to build golangci-lint
                               is lower than the targeted Go version (1.26.4)"
$ echo $?                   → 3      (make lint → 2)
```
**Two corrections to R3 as stated:**
1. This is **not a fake-green**. It exits non-zero, so `make verify` hard-fails. A verifier who ran `make verify` could not have seen a pass.
2. It is **not a standing blocker**. CI installs `golangci-lint@v2.12.2` (built with a newer Go) and lints successfully. I installed the same pin locally in ~90 seconds and used it for every lint result in this report.

R3's real significance is narrower and worse than stated: it explains how a *human or agent* on a
local machine came to record lint PASS ticks without lint output.

### F-5 — R2: the false PASS ticks are proven
`V-COMMON.md` §Procedure step 3 is unambiguous:
> "**Core suite:** `make build` · `make test` · `make lint` · `make race` — each ✅/❌."
> "verdict `PASS` requires every row ✅."

Both cards tick it:
- `M15/VALIDATION_CHECKLIST.md`: `[x] V-COMMON all ✅ (build/test/lint/race/e1; clean tree)`
- `M16/VALIDATION_CHECKLIST.md`: `[x] V-COMMON all ✅ + pytest regression`

I ran the real linter against M16's content (§F-6): **3 issues**. The tick is false.
`be13cf9` admits the same conclusion in its own words:
> "This is the defect that made the **M16-V1 `[x] V-COMMON all` tick false**: V-COMMON step 3 requires `make lint`."

**R2 = CONFIRMED**, by my own execution *and* by the repository's own record.

### F-6 — M16 rebaseline: the claim is false, but the milestone is salvageable
Procedure: worktree at `origin/main`; applied `git diff m15..m16` excluding `STATE.md`; ran gates with `golangci-lint@v2.12.2`.

| Gate | Result |
|---|---|
| `git apply --check` | ✅ OK (60,664-byte patch, clean) |
| `make gofmt-check` | ✅ clean |
| `make vet` | ✅ |
| **`make lint`** | ❌ **FAIL — 3 issues** |
| after S3 fix → `make lint` | ✅ 0 issues (both modules) |
| **`make verify` (full)** | ✅ **`verify: ALL GATES PASSED`, exit 0** |
| `docs-lint` | ❌ FAIL (3 of 7 contract files) |

The 3 failures are exactly the S3 defects, still present at M16's HEAD:
```
anthropic.go:259:23  errcheck     defer resp.Body.Close()
anthropic.go:339:3   staticcheck  QF1012 WriteString(fmt.Sprintf(...))
anthropic.go:342:3   staticcheck  QF1012 WriteString(fmt.Sprintf(...))
```
The fix exists only on `m17-full-cli-init` (in `be13cf9`), *downstream* of M16. The previous
audit's evidence list — gofmt, build, vet, tests, oip-isolation — **omits lint**, which is precisely
the gate that fails. The conclusion ("M16 can be rebaselined directly onto main") is right; the
evidence offered for it was incomplete in the one place it mattered.

**Independence confirmed:** M16's delta vs M15 touches only `docs/05-implementation/STATE.md` in
common — a ledger file. M16 is genuinely independent of M15.

### F-7 — M17: both claims true, and two more defects found
Reproduced on `m17-c3-wip` before any change:
```
--- FAIL: TestScaffoldEmbedFilesExist
      scaffold embed: "scaffold/.gitignore" not found
--- FAIL: TestEmbeddedWorkflowsByteIdentical  (3 subtests)
      open ../../../../examples/workflows/hello-world.yaml: no such file or directory
```
- **Issue 1 CONFIRMED** — `cmd/awis/init.go:27` is `//go:embed scaffold`. `cmd/awis/scaffold/.gitignore` exists, and plain `go:embed` excludes dot-files. `all:scaffold` is the correct directive.
- **Issue 2 CONFIRMED** — `init_test.go:51,55,59` use `../../../../examples/workflows/`. `cmd/awis` is two levels deep; `../../` is correct.

I applied both fixes: those two tests pass and `go build ./...` succeeds. **But four tests still fail**, in two classes the previous audit did not report:

- **Issue 3 (NEW) — missing golden files.** `TestInitGoldenTxt` / `TestInitGoldenJSON` fail: `cmd/awis/testdata/golden/init.txt` and `init.json` do not exist. C3 is genuinely unfinished, not merely mis-typed.
- **Issue 4 (NEW, self-resolving) — stale version goldens.** `TestVersionHuman` / `TestVersionJSON` fail (`want go1.26.5, got go1.26.4`). This is the bug main already fixed in `d5ac004`; a rebaseline onto main inherits the fix.

**M17's root-cause analysis was 2 of 4 complete.** Acting on it as written would have produced a red branch.

### F-8 — R4: "zero patch equivalence" is not accurate
`a53e810` (main) = 8 files. `be13cf9` (m17) = 27 files. Every one of a53e810's 8 files also appears in be13cf9 (a53e810 ⊂ be13cf9 by filename), but content diverges:

| Files | Status |
|---|---|
| 6 Go files (engine tests, fixtures, sdk/testing/mock) | **byte-identical** — true no-ops on rebaseline |
| `.github/workflows/ci.yml`, `Makefile` | **DIFFER** — main's are strictly newer (a53e810 → `535e3f0` fetch-depth 0 → `98350f6` macOS matrix) |
| 19 files only in be13cf9 | S3 lint fix, S5 apps/oip module isolation, gofmt for M10–M17 files |

`a53e810`'s own message confirms the relationship rather than equivalence: *"Brings main to the same
verification standard as the stabilization work on m17-full-cli-init (be13cf9), **scoped to what
exists on main today**."*

**Should branch stabilization be dropped during rebaseline? Yes — but not as a no-op.** The 6 gofmt
files are already on main; ci.yml/Makefile on main supersede the branch versions. However the 19
remaining files carry work that must be **re-applied per milestone**, because those files arrive
*with* their milestones:
- **S3** (anthropic.go lint) belongs to **M16** — it is the fix F-6 proves is required.
- **S5** (apps/oip `require`+`replace`) belongs to **M15** — main's Makefile comment confirms `oip-isolation` "passes trivially on main today (apps/oip is still a stub)".

Dropping `be13cf9` wholesale without re-deriving S3 and S5 into their owning milestones would
silently reintroduce both defects.

### F-9 — M15's P2 engine change is real and touches frozen surfaces
`git diff m14..m15 -- internal/engine/emit.go internal/storage/rebuild.go` → **+15/−1 across 2 files.**
Both hunks inject a sentinel on `StepFallbackActivated`:
```go
if _, exists := inst.Variables[pid]; !exists {
        inst.Variables[pid] = map[string]any{}   // emit.go — live path
}
```
with the identical guard in `rebuild.go`'s `projectInstance` — the replay path.

**Engineering assessment:** the change is *symmetric* across live and rebuild projection, which is
the correct discipline for an EventLog system — an asymmetric patch here would corrupt state on
replay. That part is right. The risk is the semantics: injecting an empty map means expressions
referencing a fallback-activated step's outputs now resolve to zero-values instead of failing, and
that behaviour change applies to **every workflow in the system**, not only OIP. `G3_BRIEF.md`
discloses this as the fallback-join sentinel and correctly routes it to founder G3.

---

## 3. Contradictions Found

| # | Contradiction | Evidence |
|---|---|---|
| C-1 | `STATE.md` on main says M10–M14 are `PHASE: E-MERGE (blocked on founder)`; they are **merged** | `f6aa755` "M10-M14" is in main's history |
| C-2 | `STATE.md` `DONE-MILESTONES: M00…M09` — omits M10–M14 | same |
| C-3 | main's `STATE.md` says M15 `PHASE: A-INIT (in progress)`; the m15 branch ledger says `E-MERGE`, all cards DONE | main vs `origin/m15:STATE.md` |
| C-4 | M15/M16 checklists tick `V-COMMON all ✅` incl. lint; lint could not run and fails on M16 content | §F-5, §F-6 |
| C-5 | M16-V1 `PASS all rows at 3b12689`; 3 lint issues exist at that SHA | §F-6 |
| C-6 | Prior audit: M16 rebaseline "gofmt/build/vet/tests/oip-isolation OK" — lint absent from the list | §F-6 |
| C-7 | Prior audit: M17 has 2 root causes; there are **4** | §F-7 |
| C-8 | Prior audit: "zero patch equivalence" between stabilization lines; 2 of 8 shared files differ and 19 files are branch-only | §F-8 |
| C-9 | Every ledger entry attributes D-CLOSE reviews to "Fable"; Fable is unavailable | §8 |
| C-10 | `STATE.md` is tracked per-branch, so four divergent ledgers exist for one project | four branch copies differ |

---

## 4. Milestone Assessment

Percentages are evidence-weighted, not impressionistic.

| Milestone | Impl | Verification | EEOS | Merge-ready | Evidence |
|---|---|---|---|---|---|
| **M15** | **95%** | **20%** | **40%** | **15%** | All cards DONE (P0/C1/C2/C3/C3r); V1 verdict rests on a false lint tick; 6/7 contract files + 2 illegal extras; hard founder gate |
| **M16** | **100%** | **35%** | **30%** | **80%** | C1 complete; V1 verdict disproven; 3/7 contract files; **but proven `verify: ALL GATES PASSED` after 1 fix** |
| **M17** | **60%** | **0%** | **30%** | **10%** | C1+C2 DONE; C3 `DISPATCHED`, WIP does not build; V1 never ran; 3/7 contract files |
| **M18** | **0%** | **0%** | **100%** | **0%** | README-only = "Partitioned — materialize at entry"; **legally compliant** (docs-lint exempts non-materialized dirs) |

### Module contract compliance (7-file)

| Module | Present | Missing | Illegal extras |
|---|---|---|---|
| **M15** | AI_EXECUTION_CONTEXT, HANDOFF, IMPLEMENTATION_SPEC, README, TRACEABILITY, VALIDATION_CHECKLIST (6/7) | **DEPENDENCY_MAP.md** | **BOUNDARY_EVIDENCE.md, G3_BRIEF.md** |
| **M16** | IMPLEMENTATION_SPEC, README, VALIDATION_CHECKLIST (3/7) | AI_EXECUTION_CONTEXT, DEPENDENCY_MAP, HANDOFF, TRACEABILITY | none |
| **M17** | IMPLEMENTATION_SPEC, README, VALIDATION_CHECKLIST (3/7) | AI_EXECUTION_CONTEXT, DEPENDENCY_MAP, HANDOFF, TRACEABILITY | none |
| **M18** | README (not materialized) | — *(compliant by design)* | none |

`cards/` subdirectories are explicitly permitted and are **not** violations
(`docs/05-implementation/README.md:28–33`).

---

## 5. Risk Register

| ID | Risk | Sev | Evidence | Mitigation |
|---|---|---|---|---|
| **R-1** | M15 P2 alters EventLog projection semantics on frozen surfaces; affects expression resolution for **all** workflows | **HIGH** | §F-9 | Opus 5 adversarial review of both projection paths + founder G3; add a replay-equivalence test asserting live == rebuild |
| **R-2** | Entire M10–M16 verification history is unproven — no CI, no lint | **HIGH** | §F-3, §F-5 | Treat all branch V1 verdicts as void; re-verify on real CI. *Mitigating:* main itself is proven green |
| **R-3** | `docs-lint` is an active merge guard and **will** go red for M15/M16/M17 | **MED** | §F-6, §4 | Materialize contract files before merge (intended behaviour per ci.yml comment) |
| **R-4** | `m17-c3-wip` is the sole copy of C3 work, branch-only | **MED** | one commit, `153772c` | Tag it (`preserve/m17-c3-wip`) before any M17 work; never force-push |
| **R-5** | Four divergent `STATE.md` ledgers; main's is materially false | **MED** | §3 C-1..C-3, C-10 | Reconcile on main first; treat main's ledger as the only authority |
| **R-6** | golangci-lint version drift recurs silently | **LOW** | §F-4 | Pin the version in the Makefile, matching ci.yml's `@v2.12.2` |
| **R-7** | Dropping `be13cf9` naively loses S3+S5 | **MED** | §F-8 | Re-derive S3 into M16 and S5 into M15 explicitly |

---

## 6. Recommended Next Action

> ### Rebaseline **M16** onto `main` — with the S3 lint fix included.

This is not inherited from the previous recommendation; it is the conclusion the evidence forces,
and it differs from that recommendation in one decisive respect (the lint fix). Why it wins:

1. **It is the only milestone with zero founder blockers.** M15 needs G3 + TDS-06 (human-only). M17 needs C3 finished. M16 needs neither.
2. **It is proven, not projected.** I executed the whole rebaseline: `verify: ALL GATES PASSED`, exit 0 (§F-6).
3. **It permanently de-stacks the chain.** M16 ∩ M15 = `STATE.md` alone. Rebaselining removes M16 from M15's founder-gated critical path forever.
4. **It buys CI truth as a side effect.** main's `ci.yml` triggers on `branches: ["**"]`, so pushing the rebaselined branch produces **the first CI run on a milestone branch in this project's history** — converting M16's verdict from self-reported to machine-proven, and validating the pipeline for M17 and M15 after it.
5. **It is small and reversible.** One 60KB patch, one 3-line lint fix, four doc files.

**Do this in the same sitting (not an action, but the long pole):** send the founder the G3 /
TDS-06 brief **now**. It is the highest-latency item on the path to M18 and it is gated on a human,
not on us. Starting it on day 0 in parallel is worth more than any sequencing choice we make.

---

## 7. Recommended Branch Strategy

**Preserve, never rewrite.**
- `m17-c3-wip` — **do not delete, rebase, or force-push.** Tag `preserve/m17-c3-wip` → `153772c` first. Recover C3 by `git diff`, not by moving the branch.
- Keep `m15/m16/m17` as-is; they become historical records once rebaselined.

**Create fresh branches from `origin/main`** — do not rebase the old ones in place:

| New branch | From | Contents |
|---|---|---|
| `m16-rb-on-main` | `origin/main` | M16 delta (minus STATE.md) + S3 lint fix + 4 contract docs |
| `m17-rb-on-main` | post-M16 main | M17 C1+C2 + C3 recovered from wip + 4 defect fixes + 4 contract docs |
| `m15-rb-on-main` | post-M17 main | M15 delta + S5 oip isolation + DEPENDENCY_MAP + extras resolved |

**Drop `be13cf9` during rebaseline — deliberately, with S3/S5 re-derived** into M16 and M15
respectively (§F-8). Its gofmt content is already on main; its ci.yml/Makefile are superseded.

---

## 8. EEOS Compliance Findings

**Ledger.** `STATE.md` on main is **not truthful** (§3 C-1..C-3). EEOS §3.3 says "no phase or card
transition is real until it is written here" — the ledger has fallen behind reality in the one
direction that rule exists to prevent. Four divergent per-branch copies compound it. Reconcile main's
copy and treat it as the sole authority.

**docs-lint — what it actually enforces.** Four mechanical rules: (1) every `MXX-*` dir has a
README; (2) a *materialized* dir (>1 `.md`) has **exactly** the 7 contract files, with `cards/` the
only permitted subdirectory; (3) nothing under `docs/` cites the superseded-documents directory
(the supersession rule, IKB §1) except three named registrars;
(4) `STATE.md` and `V-COMMON.md` exist. It is a pure content check — no code, no build.

**Who fails:** M15, M16, M17 (§4 table). M18 passes because rule 2 fires only on materialization —
a README-only partition is compliant by design. main passes today, which is why its docs-lint job is
green while acting as a live merge guard.

**Do M15's `BOUNDARY_EVIDENCE.md` and `G3_BRIEF.md` violate the contract?** **Yes.** Rule 2 demands
*exactly* the 7 files, and precedent is unanimous: all 15 merged modules (M00–M14) carry exactly 7
`.md` files with zero extras.

**Is there a better resolution than delete / move / amend? Yes — fold, and fix two problems with one
move.** M15 is simultaneously *missing* `DEPENDENCY_MAP.md` and *carrying* `BOUNDARY_EVIDENCE.md`.
These are the same subject: `BOUNDARY_EVIDENCE.md` is the QG-4 proof that the platform diff is empty
outside `apps/oip/` + docs + disclosed seams — i.e. a statement of what M15 depends on and what it is
forbidden to touch. M14's `DEPENDENCY_MAP.md` is exactly that genre (Upstream / Downstream / branch
discipline).

1. **`BOUNDARY_EVIDENCE.md` → `DEPENDENCY_MAP.md`**, with the boundary proof as its "Platform boundary
   (QG-4)" section. Removes one extra *and* supplies the missing file. No content lost, no deletion,
   no amendment.
2. **`G3_BRIEF.md` → `cards/M15-G3.md`.** `cards/` is already the sanctioned home for gate artifacts
   and is explicitly exempt (`README.md:28–33`). A founder gate brief *is* an execution card. Cite it
   from `VALIDATION_CHECKLIST.md`'s existing G3 row.

This is strictly better than the three options offered: deletion destroys founder-gate evidence;
a bare move to `cards/` leaves `DEPENDENCY_MAP.md` still missing; an EEOS §13 amendment spends
constitutional capital to legalise an accident and weakens a rule that 15 modules have honoured.
No amendment is required.

**Model allocation — translating "Fable".** The ledger attributes every D-CLOSE semantic review to
Fable (unavailable). Mapping, consistent with `CLAUDE.md`'s frozen policy (Opus not for implementation;
architecture review and merge approval only):

| Fable's role | Now | Why |
|---|---|---|
| D-CLOSE semantic / architectural drift review | **Opus 5** (`awis-core-engineer`) | Judgement work on frozen surfaces — exactly the reserved-review carve-out. **R-1 (M15 P2) requires this.** |
| "Upward substitution" of Opus-designated cards | **Opus 5**, review-only | Correctness-critical engine/EventLog reasoning |
| Card execution (S3 fix, docs, C3 goldens) | **Sonnet 5** (`awis-builder`, `awis-scribe`) | Mechanical, well-specified — the policy's designated implementer |
| Independent verification | **Sonnet 5** (`awis-verifier`) on a clean worktree | Must not be the author; binary evidence table only |
| Repo search / symbol lookup | **Haiku 4.5** | Per the frozen policy |

The policy's intent survives intact: Opus judges, Sonnet builds, and no agent verifies its own work.

---

## 9. Path To M18

M18 depends on M15 + M16 + M17. Blockers, separated as requested:

**Technical (we can fix):**
- T-1 — M16 carries 3 lint defects (§F-6). *Fix: 3 lines.*
- T-2 — M17-C3 incomplete: `go:embed all:`, test path depth, **missing init goldens**, stale version goldens (§F-7).
- T-3 — S5 apps/oip module isolation must land with M15 (§F-8).
- T-4 — M15 P2 needs a replay-equivalence test (R-1).

**Process (we can fix):**
- P-1 — 12 missing contract files across M15/M16/M17 + M15's 2 extras (§4).
- P-2 — `STATE.md` reconciliation (§3).
- P-3 — M15/M16 V1 verdicts are void; re-verify on real CI (R-2).
- P-4 — pin golangci-lint in the Makefile (R-6).

**Founder (only the founder can clear):**
- **G-1 — G3 platform-boundary verdict (QG-4).** *Long pole — start now.*
- **G-2 — TDS-06 OIP Record format sign-off.**
- G-3 — squash-merge of M15, M16, M17 (E-MERGE is human-only under EEOS).

### Execution order

| # | Step | Owner | Blocker cleared |
|---|---|---|---|
| **0a** | **Send founder the G3 + TDS-06 brief** (parallel, day 0) | human | G-1, G-2 *(latency starts now)* |
| **0b** | Tag `preserve/m17-c3-wip`; pin golangci-lint in Makefile | Sonnet 5 | R-4, P-4 |
| **1** | `STATE.md` reconciliation on main | Sonnet 5 (`awis-scribe`) | P-2 |
| **2** | **M16-RB** → `m16-rb-on-main` + S3 fix | Sonnet 5 (`awis-builder`) | T-1 |
| **3** | M16 contract docs (4 files) | Sonnet 5 (`awis-scribe`) | P-1 |
| **4** | **M16-V2** — push; first real branch CI | `awis-verifier` | P-3 |
| **5** | Founder squash-merge M16 | human | G-3 |
| **6** | **M17-RB** — recover C3 from wip; fix all 4 defects; generate init goldens | Sonnet 5 | T-2 |
| **7** | M17 contract docs + M17-V1 on CI | Sonnet 5 / verifier | P-1, P-3 |
| **8** | Founder squash-merge M17 | human | G-3 |
| **9** | **M15-RB** + S5 + `DEPENDENCY_MAP` fold + `cards/M15-G3.md` | Sonnet 5 | T-3, P-1 |
| **10** | **M15 P2 adversarial review** + replay-equivalence test | **Opus 5** | T-4, R-1 |
| **11** | M15-V2 on CI; founder G3 verdict lands | verifier / human | P-3, G-1, G-2 |
| **12** | Founder squash-merge M15 | human | G-3 |
| **13** | **M18 A-INIT** — materialize 7 contract files | Sonnet 5 | — |

M16 before M17 before M15 inverts the numeric order deliberately: it front-loads the two milestones
with no founder dependency while G-1/G-2 are in the founder's queue, so human latency overlaps
machine work instead of following it.

---

## 10. Confidence Levels

| Finding | Confidence | Basis |
|---|---|---|
| Branch SHAs match briefing | **Certain** | `git ls-remote` |
| main green on Linux + macOS incl. lint | **Certain** | Actions API, job-step level |
| **CI never ran on any milestone branch** | **Certain** | `total_count: 3`, all `head_branch: main` |
| R2 — false lint PASS ticks | **Certain** | Executed lint + repo's own `be13cf9` message |
| R3 — lint broken locally, works in CI | **Certain** | Exit 3 locally; CI step 8 passed; fixed by pinned install |
| M16 rebaseline fails lint, passes after fix | **Certain** | `verify: ALL GATES PASSED`, exit 0 |
| M17 issues 1 & 2 confirmed | **Certain** | Reproduced, then fixed, then re-ran |
| M17 issues 3 & 4 (new) | **Certain** | Missing files confirmed by `ls`; golden mismatch observed |
| R4 — not patch-equivalent | **Certain** | Per-file content diff of all 8 |
| STATE.md untruthful | **Certain** | Ledger text vs `f6aa755` |
| M15 P2 changes frozen surfaces | **Certain** | Diff read in full |
| M15 P2 is *risky* (expression semantics) | **High** | Code reading; not yet dynamically tested — this is precisely what R-1 asks Opus 5 to settle |
| M15 impl 95% complete | **High** | Branch ledger + card SHAs; not independently re-verified end-to-end |
| M16 is the highest-value next action | **High** | Proven green + zero founder deps; assumes founder merge cadence is not the binding constraint |
| Fold-into-DEPENDENCY_MAP is the best EEOS fix | **High** | Contract text + 15/15 precedent; founder may still prefer an amendment |
| *Why* CI never ran on branches | **Low** | Effect is certain; the cause (Actions enablement timing) is inferred and does not affect any recommendation |

---

*Audit performed by re-derivation from primary sources: git object store, GitHub Actions API, and
executed build/lint/test gates. No conclusion rests on a prior report.*
