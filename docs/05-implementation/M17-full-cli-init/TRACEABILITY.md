# M17 — Traceability
| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | Migration `0006_recall_fts` (FTS5 over execution_events payloads, additive) | CONTRA-4; IMP §14 (renumbered 0006 per F-4 sequence, disposition in M12 TRACEABILITY) | C1 |
| T2 | `RecallStore` additive interface (StoragePort untouched; audit.go pattern) | IMP §27.M17 | C1 |
| T3 | `history [workflow-id] [--limit]` | PRD §20 | C1 |
| T4 | `logs [--tail N]` (+ start.go file-sink addition) | IMP §27.M17 | C1 |
| T5 | `metrics` (counts/durations from events) | PRD §20 | C1 |
| T6 | `recall <query> [--synthesize]` (Null adapter empty-state, M16 adapter when key present) | PRD §21 | C1 |
| T7 | `replay <instance-id>` (dry-run, no state writes) | PRD Should-Have | C1 |
| T8 | `audit [--limit]` | — | C1 |
| T9 | `export [--out dir]` | PRD §25 | C1 |
| T10 | `prune-events --dry-run` (destructive prune deferred post-V1) | F-2 (7d TTL) | C1 |
| T11 | `config show\|set\|validate\|edit` (+ `ConfigChanged` audit row, closes F-4 write-site set) | NFR-S-01; F-4 | C2 |
| T12 | `plugin status\|remove` | F-5 | C2 |
| T13 | `rebuild-state <instance-id>\|--all` | M03 rebuild path | C2 |
| T14 | Cron trigger scanner in `start` (stdlib 5-field parser; fake-clock tests; engine untouched) | F-2; FR-WE-14 | C2 |
| T15 | `awis init` (go:embed scaffold: config.yaml, .gitignore, 3 M10 example workflows byte-identical, handlers/example_handler.go, README_AWIS.md; `--force` guard) | FR-RM-01 | C3 |
| T16 | init→start→submit→trace rehearsal system test (real binary, not `-short`) | IMP §27.M17 Val; QG-1 path (measured loosely here, formal at M18) | C3 |
| T17 | `docs/CLI.md` completed for the full command tree | IMP §27.M17 DoD | C1–C3 |
| T18 | TDS-07 (`docs/CLI_CONTRACT.md`) section per command | TDS-07 discipline | C1–C3 |

## Notes / dispositions
- Cron scanner lives in `cmd/awis` start loop, not the engine (Non-scope, IMPLEMENTATION_SPEC.md)
  — engine semantics stay frozen.
- `awis init`'s embedded workflow examples must be byte-identical to `examples/workflows/*.yaml`
  (M10 sources) — proven by `TestEmbeddedWorkflowsByteIdentical`, not merely "similar."
- `//go:embed scaffold` (no `all:` prefix) silently excludes dotfiles — `scaffold/.gitignore`
  would be dropped from the binary without `all:scaffold`. Guarded by
  `TestScaffoldGitignoreEmbedded` so this cannot regress silently again.

## Execution record (appended during B-BUILD/C-VERIFY)
- C1 (a342394, 2026-07-11): migration 0006 + RecallStore + 8 read-path commands (history, logs,
  metrics, recall, replay, audit, export, prune-events --dry-run) + goldens + TDS-07 sections.
  Gates green.
- C2 (744d3d0, 2026-07-11): config show/set/validate/edit + ConfigChanged audit + plugin
  status/remove + rebuild-state + cron trigger scanner (fake-clock tests). Gates green.
- C3 (afe526d, 2026-08-21): **P2 salvage re-dispatch.** STATE had shown C3 `DISPATCHED` with no
  commit on `m17-full-cli-init` — a prior run died before finishing or committing. A 9-file WIP
  survived on preserved branch `m17-c3-wip` (`cmd/awis/init.go`, `init_test.go`,
  `scaffold/{.gitignore,config.yaml,README_AWIS.md,handlers/example_handler.go,
  workflows/{hello-world,with-signal,with-intelligence}.yaml}`), explicitly marked "does NOT
  build green — do not merge." Re-dispatched to awis-builder with the P2 preamble: audited the
  WIP against this card's OUTPUTS/ACCEPTANCE, salvaged what passed, and fixed 4 real defects
  found in it plus 2 outstanding gaps:
  1. `//go:embed scaffold` → `//go:embed all:scaffold` (dotfile `.gitignore` was silently
     excluded without the `all:` prefix; reproduced the exclusion before fixing; added
     `TestScaffoldGitignoreEmbedded` as a regression guard).
  2. `TestEmbeddedWorkflowsByteIdentical` source paths were `../../../../examples/workflows/…`
     (4 levels up) — 2 levels too deep from `cmd/awis/init_test.go` (repo root is 2 levels up).
     Fixed to `../../examples/workflows/…`.
  3. No init goldens existed — added `cmd/awis/testdata/golden/init.{txt,json}` following the
     `history`/`config_show` golden-harness pattern.
  4. No rehearsal system test existed — added `TestSystemRehearsalInitStartSubmitTrace` to
     `cmd/awis/system_test.go`: builds the real binary, runs `init` into a temp dir,
     `start`s against it, `submit`s (with `--namespace=examples`, see deviation below),
     polls `status --json`, and asserts on `trace` output. Not skipped under `-short`.
  5. `docs/CLI.md` completed for the full tree (all C1/C2 commands + `init`), not just the
     init section.
  6. `docs/CLI_CONTRACT.md` gained its TDS-07 `### init` section.
  `m17-c3-wip` was left untouched (not merged, not deleted) per instruction.
  **Independent re-verification this session** (not just the builder's self-report): `make
  verify` full green at afe526d (gofmt-check, vet, lint 0 issues, oip-isolation, build, test,
  race, e1); `TestSystemRehearsalInitStartSubmitTrace` re-run directly, confirmed un-skipped
  (11.01s) and passing; `go.mod`/`go.sum` diff vs branch tip (49ce489) is empty; embed
  directive and test-path fixes confirmed by direct file inspection; docs sections confirmed
  present by grep (`docs/CLI.md` `## init` at L807, `docs/CLI_CONTRACT.md` `### init` at L725).
  **Deviation:** the scaffolded `hello-world.yaml` carries `namespace: examples`, and
  `sdk.Runtime.Submit`'s cross-process lookup filters by the caller's `--namespace` with no
  cross-namespace default fallback — the rehearsal test (and `init`'s docs/CLI_CONTRACT.md
  notes) require `--namespace=examples` on submit. Documented, not a defect.
  **Deviation:** the rehearsal test's workflow instance legitimately reaches terminal status
  `failed` (`handler_not_found`) because the generic `awis` binary ships no compiled-in
  application handlers — this is correct: the rehearsal proves CLI wiring end-to-end
  (init→start→submit→trace all function against a fresh scaffold), not business-handler
  execution, consistent with the CE pin's "measured loosely here, formally at M18" framing.
  **Found, out of C3's scope, not fixed:** `trace --json` silently swallows a JSON-encode
  error when an event's `Payload` is a non-nil zero-length `json.RawMessage`, producing empty
  stdout at exit 0. The rehearsal test routes its polling through `status --json` instead
  (card explicitly permits "status+trace") and asserts on `trace` in human mode, which is
  unaffected. Flagged here for M17-V1 to assess/route, or M18 otherwise — not part of this
  card's acceptance criteria.
- V1: not yet dispatched (module still needs its C-VERIFY pass; out of this session's scope).
