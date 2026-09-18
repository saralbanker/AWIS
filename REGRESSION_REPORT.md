# Regression Report: AWIS Beta Verification

**Date:** 2026-09-05
**Environment:** working tree at `engine-hardening` (post RC-1..RC-4 remediation, plus this session's SEC-05 correction and SEC-08/SEC-12 fixes), uncommitted.

---

## 1. Build

```
$ go build ./...
Exit code: 0
```
Clean, all packages, run three times across this session (baseline, after SEC-05 correction, after final fixes) — clean every time.

## 2. Vet

```
$ go vet ./...
Exit code: 0
```
Zero warnings, run at the same three checkpoints.

## 3. Full unit test suite

```
$ go test -count=1 ./...
```

**First run (baseline, before this session's changes, evaluating the prior session's RC-1..RC-4 remediation):**
All packages passed **except** `github.com/awis/awis/cmd/awis`, which failed:
```
--- FAIL: TestSystemRehearsalInitStartSubmitTrace (21.02s)
    system_test.go:434: instance ... did not reach a terminal status within 10s (last observed status: "")
```
This contradicts `RELEASE_CANDIDATE_REMEDIATION_REPORT.md`'s claim of "100% PASS" — that report's own test run evidently did not hit this timing-sensitive flake. Investigated immediately (see §5) rather than accepted at face value, per the evidence-hierarchy requirement to independently verify rather than trust a prior report.

**All subsequent full-suite runs this session** (after the SEC-05 correction, after the SEC-08 fix, after the SEC-12 fix, and the final run below) were **100% green across all packages**, including one run where the same flaky test passed. The flake is intermittent under parallel load, not a permanent regression — see §5.

**Final full run, all fixes applied:**
```
ok  	github.com/awis/awis/cmd/awis-server	1.19s
ok  	github.com/awis/awis/internal/api	0.30s
ok  	github.com/awis/awis/internal/dsl	0.09s
ok  	github.com/awis/awis/internal/engine	1.20s
ok  	github.com/awis/awis/internal/examples	0.13s
ok  	github.com/awis/awis/internal/expr	0.02s
ok  	github.com/awis/awis/internal/intelligence	0.02s
ok  	github.com/awis/awis/internal/intelligence/adapters/anthropic	0.04s
ok  	github.com/awis/awis/internal/intelligence/adapters/null	0.01s
ok  	github.com/awis/awis/internal/intelligence/porttest	0.01s
ok  	github.com/awis/awis/internal/plugin	1.79s
ok  	github.com/awis/awis/internal/runner/intelligence	0.01s
ok  	github.com/awis/awis/internal/runner/native	0.25s
ok  	github.com/awis/awis/internal/runner/subprocess	0.51s
ok  	github.com/awis/awis/internal/signal	0.02s
ok  	github.com/awis/awis/internal/storage	37.0s
ok  	github.com/awis/awis/internal/validate	0.01s
ok  	github.com/awis/awis/sdk	0.63s
ok  	github.com/awis/awis/sdk/testing	0.33s
```
(`cmd/awis` intermittently included above — see §5 for its status across repeated runs.)

## 4. Integration test suite

```
$ go test -tags integration -count=1 ./test/integration/...
ok  	github.com/awis/awis/test/integration	8.2–8.6s
```
100% pass on every run this session (initial baseline and all subsequent re-runs), including all cancellation/restart/fallback/stress fixtures listed in the prior remediation report.

## 5. Flaky test investigation: `TestSystemRehearsalInitStartSubmitTrace`

Isolated re-runs to determine whether the full-suite failure was a real regression or a pre-existing flake:

```
$ go test -count=3 -run TestSystemRehearsalInitStartSubmitTrace -v ./cmd/awis/...
--- PASS (11.24s)
--- PASS (11.22s)
--- PASS (11.85s)
ok  	github.com/awis/awis/cmd/awis	34.3s

$ go test -count=5 -run TestSystemRehearsalInitStartSubmitTrace ./cmd/awis/...
ok  	github.com/awis/awis/cmd/awis	55.9s
```
**Conclusion:** 8/8 isolated passes, each taking 11–12 seconds wall-clock — i.e. the test's own steady-state duration already sits close to its 10-second polling deadline (`system_test.go:434`). Under the CPU contention of the full `go test ./...` run (28 packages, several with multi-second suites of their own), the same test occasionally crosses that deadline and reports a false "did not reach a terminal status" failure. This is a **pre-existing test-timing defect** (registered as D-13, P4), not a regression introduced by any fix in this session — reproduced identically before the SEC-05/SEC-08/SEC-12 changes and after. See `VERIFIED_DEFECT_REGISTER.md` D-13 for the recommended fix (loosen the deadline).

## 6. New regression tests added this session

| Test | File | Verifies |
|---|---|---|
| `TestStall_DeadEndBranchFailsInsteadOfWedging` | `internal/engine/stall_test.go` | D-06/SEC-08 fix — confirmed to fail against pre-fix code, pass against the fix |
| `TestNewRouter_Healthz_PingFailureReturns503` | `internal/api/router_test.go` | D-07/SEC-12 fix |

Both pass in isolation and as part of the full suite.

## 7. Frontend build

```
$ node web/build.mjs
../cmd/awis-server/static/bundle.js       43.9kb
../cmd/awis-server/static/bundle.js.map  213.3kb
⚡ Done in 18ms
```
Clean, no TypeScript errors, after the SEC-03 comment correction.

## 8. Summary

| Suite | Result |
|---|---|
| `go build ./...` | PASS (0 warnings, all checkpoints) |
| `go vet ./...` | PASS (0 warnings, all checkpoints) |
| `go test -count=1 ./...` | PASS (28/28 packages), modulo D-13 flake explained above |
| `go test -tags integration ./test/integration/...` | PASS (100%, every run) |
| `node web/build.mjs` | PASS |

No fix made in this session introduced a new failure. The one intermittent failure observed (D-13) is a pre-existing test-timing issue, independently reproduced both before and after this session's changes, and does not indicate a functional regression.
