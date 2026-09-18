# AWIS — Final Release Verdict

**Date:** 2026-09-05 · **HEAD:** `engine-hardening` @ `8a87f70`
**Audit:** Adversarial release audit, every conclusion re-derived from Tier 1 evidence.
**Supersedes:** `FINAL_VERDICT.md` (2026-09-05, *PASS WITH RISKS*)

---

# VERDICT: **CONDITIONAL PASS**

**AWIS's engine is Beta-quality. Its release envelope is not.** The conditions below are specific, enumerable, and mostly mechanical — several are minutes of work. None require redesign. Once they are met, this is a defensible public Beta.

---

## Why not PASS WITH RISKS

The prior verdict was reasonable for what it examined and its engine claims held up under re-derivation. It could not be sustained because of one fact no prior report checked:

> **The entire Beta deliverable is not in version control.** `internal/api`, `cmd/awis-server`, `web/`, and `internal/buildinfo` have **zero tracked files**. A clean clone of `HEAD` cannot build `awis-server` — the directory does not exist.

This is not a risk to be accepted. It is a precondition of releasing software. Every "Dashboard-live / GUI MVP / GUI Beta achieved" verdict was issued against code that has never been committed, never been reviewed as a diff, and — because CI checks out the repository — **never once been built or tested by CI**.

"PASS WITH RISKS" would mean the remaining items are informed tradeoffs. Uncommitted code is not a tradeoff.

Compounding: the HTTP API has **no authentication, CORS policy, rate limiting, body limits, or server timeouts** — verified by grep returning zero matches and by live probing. The default loopback bind limits real exposure today, which is why this is not on its own disqualifying. But `--addr` accepts any interface, with no auth, no warning, and no deployment guide saying otherwise.

## Why not REJECTED

Because the engineering underneath is genuinely good, and an audit that punished it for a `git add` would be wrong.

Re-derived from Tier 1, not inherited:

- **Race detector clean across all 20 packages.**
- **`golangci-lint`: 0 issues** across both modules; `gofmt` clean; `go vet` clean.
- **Integration suite passes.**
- **The end-to-end user journey works** — I ran `init → start → submit → status → trace` by hand and the instance completed.
- **The engine-hardening fixes are real**, including ones I tried hard to disprove: `healthz` genuinely pings the database, the Anthropic model IDs are correct and current, crash recovery and cancellation durability are covered by passing tests.
- **No secret was ever committed** — full history scan; all hits are placeholders.
- **No XSS surface** in the GUI — zero `innerHTML`/`unsafeHTML`, lit-html escaping throughout.
- **The SQLite concurrency design is genuinely well-reasoned**, and its rationale comments are accurate.

I probed the engine adversarially and found **no correctness, data-loss, or security defect in it.** The prior audits' engine work was sound. The problems are concentrated in the delivery and operational envelope — a layer nobody audited, because every prior audit was scoped to the engine.

---

## Conditions for Beta release

### Blocking — release cannot proceed

**C1. Commit the deliverable.** `git add internal/api cmd/awis-server web internal/buildinfo`, and correct the `.gitignore` comment that currently asserts — falsely — that `web/` source and `cmd/awis-server/static/` are tracked. *(RA-01)*

**C2. Re-run full verification after C1.** It will be the first verification pass in this project's history that actually covers the API and GUI. Treat its results as new information, not confirmation.

**C3. Add the API and GUI to CI.** Frontend build + `tsc --noEmit` + `make integration` in the `verify` target. *(RA-02, RA-14)*

**C4. Set HTTP server timeouts.** `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`. Two lines; reproduced exposure. *(RA-04)*

**C5. Either enforce the loopback bind or add authentication.** A Beta may ship loopback-only — that is a legitimate scope decision — but then `--addr` must refuse non-loopback values, or warn loudly. Shipping an unauthenticated bind-anywhere flag with no deployment guidance is not acceptable. *(RA-03)*

### Required before public Beta

**C6. Write an install + quickstart README.** One sentence about module boundaries is not an entry point. The scaffolded onboarding is genuinely good and currently unreachable. *(RA-10)*

**C7. Document backup and restore, including the WAL caveat** — that copying `runtime.db` without `-wal`/`-shm` yields a torn backup. Operators will otherwise believe they have backups they do not have.

**C8. Add the migration downgrade guard.** Refuse to open a database whose `schema_version` exceeds the binary's newest migration. One `if`; makes rollback safe. *(RA-09/AD-06)*

**C9. Fix `awis status` / `awis history` performance.** Replace the insertion sort with `sort.Slice` and route both through the existing `ListInstancesPaged`. Currently 4.93s and 5.17s at 20k instances, ~19s of sort alone at 40k — degrading exactly in the incident-response path. Highest value-to-effort item in the audit. *(RA-05)*

**C10. Validate API query parameters.** `?status=TOTALLY_BOGUS` returning `200 {"instances":[],"total":0}` is indistinguishable from a genuinely empty result. This is the same silent-defaults class commit `e75c1f1` claims to have closed — closed in the CLI, still open across the whole API. *(RA-07)*

### Strongly recommended

**C11.** Stop returning internal error strings to unauthenticated clients *(RA-08)*.
**C12.** Drop `bundle.js.map` from the production `go:embed` *(RA-12)*.
**C13.** Stabilize `TestSystemRehearsalInitStartSubmitTrace` — raise its budget or serialize it. It failed on my first clean full-suite run and it gates the only end-to-end user-journey check *(RA-15)*.
**C14.** Separate engine `slog` output from CLI stdout — the test suite already documents a workaround for this.
**C15.** Move `api_keys/apikeys.txt` out of the repository directory. Correctly gitignored and never committed, but a live credential in the working tree is one tarball from exposure.

---

## Answers to the audit questions

**Is AWIS genuinely Beta-ready?** The engine is. The release envelope is not. The gap is C1–C5.

**Can a real external user successfully use it?** Given a working checkout and told what to run — yes, verified by hand. Starting from `git clone` — **no**: no server binary, no GUI, no README worth the name.

**Are any release blockers still present?** Yes: C1 and C3. Both are process blockers, not engine defects.

**Are hidden P0/P1 defects likely?** In the engine, no — I looked hard. In the **API/GUI layer, yes** — it has never been through CI, code review, or a typechecker. RA-03, RA-04, and RA-07 are what one hour of probing found in a layer with zero prior scrutiny. **Expect more.** That expectation is itself an argument for C1–C3: not because the code is bad, but because nobody knows whether it is.

**Real operational maturity?** **Low — developer preview.** Excellent engine and diagnostics; essentially no operational envelope.

**What work remains?** C1–C10, of which C1, C4, C8, and C9 are roughly an afternoon.

---

## Closing note

The strongest signal in this audit is not any single defect. It is the **shape** of what was missed.

Five prior reports examined the engine in real depth and got it right. Not one of them asked whether the software was in version control, whether CI had ever built it, whether the HTTP API had authentication, or whether a new user could install it. The engine was audited repeatedly; the product was never audited once.

`VERIFIED_GEMINI_FINDINGS.md` already contains the correct diagnosis of this pattern, written about a different defect:

> *"an LLM-authored audit and an LLM-authored remediation share the same blind spot."*

That was right, and it generalizes further than its author applied it. Successive passes over the same artifact converge on the same frame and stop seeing past its edges. Fixing that is not more engine audits — it is checking the boring perimeter: is it committed, is it built, is it documented, can someone else run it.

**Recommended next step:** satisfy C1–C3, then re-verify. The first genuinely complete verification run may surface findings this audit could not reach, because until the deliverable is in version control, no tool in the project has ever looked at it.
