# AWIS — Documentation Coverage and Divergence Analysis

**Scope:** documentation coverage of the existing codebase. Not system health, not release
readiness, not bug discovery.
**Repository state analysed:** `/mnt/data/rj/AWIS`, branch `engine-hardening`, HEAD `8a87f70`,
plus the uncommitted working tree (which contains load-bearing production code — see §3.1).
**Date of analysis:** 2026-09-17.
**Evidence hierarchy applied:** Code > Docs > Tests > Reports. Every finding below is anchored
to Tier 1 (source) or to a mechanically reproducible Tier 2 fact (file existence, git dates,
`docs-lint` output). Tier 4 documents are cited only as corroboration, never as authority.

---

## 0. Summary of what the evidence shows

Three findings carry the analysis. Each is stated here in one line and evidenced in full below.

1. **The canonical `docs/` routing contract has been self-contradicting since day one of
   implementation.** `docs/06-reference/README.md` declares that all normative TDS specs live at
   `docs/06-reference/<NAME>.md` and that "a missing TDS here means its milestone has not
   executed." The directory has contained nothing but that README since it was created
   (2026-07-03). All six produced TDS specs were written to `docs/<NAME>.md` instead — three of
   them on **the same day**, in the same commit series. This is a documentation-system defect
   that predates the GUI by two months (§6.1).

2. **Two entire engineering programs ran outside the documentation system that was built to
   govern them.** The milestone ledger's last entry is 2026-08-21 (M17, `C-VERIFY`, never
   closed). Between 2026-08-26 and 2026-09-07, thirty commits of engine hardening plus an entire
   HTTP API, server binary and web GUI were produced. Neither program opened a milestone module,
   neither updated a TDS, and the second produced no committed code at all (§3.1, §9).

3. **The GUI is the *best*-documented recent surface, not the source of the divergence.** All
   five implemented GUI screens have a written screen spec; five of the seven implemented API
   routes have a written route contract. What the GUI program did *not* do is re-enter the
   canonical authority chain — its 38 documents are invisible to every `docs/` router and index,
   and are not in git (§10).

The undocumented surface is narrow and enumerable: the HTTP API as a *normative* contract, the
`awis-server` binary, `internal/buildinfo`, the web build pipeline, migration `0007`, the SDK
public surface, and the behavioural changes made by the engine-hardening program. That list is
§12.

A fourth finding qualifies the first three: **where specifications do exist, they are accurate
at the level they were verified at, and drift at the level below it.** The six normative TDS
specs enumerate exactly the right event types, YAML keys, expression operators, JSON-RPC
methods and CLI commands — full parity, both directions. One level finer, `CLI_CONTRACT.md`
specifies flags on `start`, `rebuild-state` and `export` that do not exist and that produce
exit-2 parse errors if used as written (§4.5). Coverage at this repository is not uniformly
poor; it is *unevenly verified*.

---

## 1. What exists in code

Inventory of the implemented surface at HEAD plus working tree.

### 1.1 Go packages (265 `.go` files, 2 modules)

| Area | Packages |
|---|---|
| Public SDK | `sdk`, `sdk/testing` |
| Core domain | `internal/core` |
| Engine | `internal/engine`, `internal/signal` |
| Storage | `internal/storage`, `internal/storage/storagetest` |
| Expressions | `internal/expr`, `internal/expr/corpus` |
| DSL | `internal/dsl` |
| Validation | `internal/validate` |
| Intelligence | `internal/intelligence`, `.../adapters/anthropic`, `.../adapters/null`, `.../porttest` |
| Runners | `internal/runner/native`, `internal/runner/subprocess`, `internal/runner/intelligence` |
| Plugins | `internal/plugin` |
| **HTTP API** | **`internal/api`** *(untracked)* |
| **Build metadata** | **`internal/buildinfo`** *(untracked)* |
| Fixtures/examples | `internal/fixtures`, `internal/examples`, `examples/hello_workflow` |
| Binaries | `cmd/awis`, `cmd/awis/scaffold/handlers`, **`cmd/awis-server`** *(untracked)* |
| Application | `apps/oip` (separate module `github.com/awis/oip`) |
| Tests | `test/integration` (binary-level tier), 129 `_test.go` files |

### 1.2 CLI surface — `cmd/awis`

21 top-level commands: `version`, `start`, `stop`, `status`, `submit`, `signal`, `cancel`,
`trace`, `workflow`, `plugin`, `config`, `init`, `history`, `logs`, `metrics`, `recall`,
`replay`, `audit`, `rebuild-state`, `export`, `prune-events`.
Subcommands: `workflow {validate,list,show}`, `config {show,set,validate,edit}`,
`plugin {install,list,status,remove}`.
Global flags: `--data-dir` (default `./.awis/`), `--json`, `--namespace` (default `default`).

### 1.3 HTTP API — `internal/api` (untracked)

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/healthz` | `internal/api/healthz.go` |
| GET | `/api/v1/info` | `internal/api/info.go` |
| GET | `/api/v1/workflows` | `internal/api/workflows.go` |
| GET | `/api/v1/workflows/{id}/{version}` | `internal/api/workflows.go` |
| GET | `/api/v1/instances` | `internal/api/instances.go` |
| GET | `/api/v1/instances/{id}` | `internal/api/instances.go` |
| GET | `/api/v1/instances/{id}/events` | `internal/api/events.go` |

Registered at `internal/api/router.go:33-41`. Query parameters: `namespace`, `status`,
`definition_id`, `limit`, `offset` (instances); `from`, `limit` (events).
Read-only; there is no mutation route.

### 1.4 Server binary — `cmd/awis-server` (untracked)

Flags (`cmd/awis-server/main.go:51-55`): `--db` (default `./awis-server.db`), `--addr`
(default `127.0.0.1:8090`), `--static-dir`. Serves an embedded frontend build
(`cmd/awis-server/static/{index.html,bundle.js,style.css}`) via `static.go`, or from disk when
`--static-dir` is set.

### 1.5 Web GUI — `web/` (untracked)

TypeScript + `lit-html`, built with `esbuild` (`web/build.mjs`, `web/package.json`).
Five screens: `instanceList`, `instanceDetail`, `eventTimeline`, `workflowList`,
`workflowDetail`. Support modules: `apiClient.ts`, `router.ts`, `types.ts`, `ui.ts`,
`eventFormatters.ts`, `onboarding.ts`.

### 1.6 Storage

7 migrations: `0001_core_execution`, `0002_domain_events`, `0003_signals`, `0004_audit`,
`0005_plugins`, `0006_recall_fts`, `0007_cancellation_intent`.
Tables: `execution_events`, `workflow_definitions`, `step_results_cache`,
`workflow_instances`, `step_claims`, `domain_events`, `signal_inbox`, `wait_records`,
`audit_log`, `plugins`, `plugin_capabilities`, `execution_events_fts`, `schema_version`.

### 1.7 Domain constants

- **12 event types** (`internal/core/event.go:40-62`).
- **5 step types** (`internal/core/step.go:44-52`): `native`, `subprocess`, `plugin`,
  `intelligence`, `signal`.
- **9 instance statuses** (`internal/core/instance.go:38-56`).
- **4 trigger types** (`internal/core/workflow.go:67-73`): `manual`, `schedule`, `event`,
  `webhook`.

### 1.8 Intelligence

Adapters: `anthropic` (models `claude-haiku-4-5-20251001` fast, `claude-sonnet-5` quality;
capabilities draft/synthesize/classify; no embed), `null`.
Port method set: `Draft`, `Synthesize`, `Classify`, `IsAvailable`, `Capabilities`,
`ProviderName`, `Usage`.

### 1.9 Configuration surface

Environment variables read anywhere in the tree: `ANTHROPIC_API_KEY`, `EDITOR`, `VISUAL`,
`PATH`. Config file: `config.yaml` (project root or `.awis/config.yaml`). Data directory
`./.awis/` holding `runtime.db`, `awis.pid`, logs.

---

## 2. What exists in docs

271 Markdown files under `docs/`. **212 are tracked in git; 59 are not.**

| Directory | Files | Tracked | Role |
|---|---|---|---|
| `docs/` (root specs) | 11 | yes | Normative TDS specs + router README |
| `00-foundation` … `04-planning` | 5 | yes | Pointer READMEs into the root canonical corpus |
| `05-implementation` | 174 | yes | Milestone modules M00–M18, `STATE.md`, `V-COMMON.md` |
| `06-reference` | 1 | yes | **README only — the directory is empty of content** |
| `07-indices` | 5 | yes | Traceability matrix, dependency index, cross-reference index, canonical map |
| `08-engine-hardening` | 5 | yes | Narrative reports for the hardening program |
| `09-gui-planning` | 38 | **no** | GUI/API planning program |
| `10-release-candidate-audit` | 6 | **no** | Adversarial RC audit |
| `11-intelligence-architecture` | 8 | **no** | Provider-agnostic investigation |
| `12-intelligence-architecture-decision` | 7 | **no** | Architecture decision exercise |
| `edr` | 11 | yes | Engineering decision records EDR-001…011 |

The normative spec set (`docs/*.md`) is: `CLI.md` (873 lines), `CLI_CONTRACT.md` (1497),
`DSL.md` (249), `EVENTLOG_FORMAT.md` (154), `EXPRESSION_GRAMMARS.md` (151),
`PLUGIN_GUIDE.md` (240), `PLUGIN_PROTOCOL.md` (455), `PROVIDERS.md` (145),
`SUBPROCESS_PROTOCOL.md` (260), `WORKFLOW_SCHEMA.md` (131), `README.md` (26).

Separately, the canonical corpus that `docs/` routes *to* lives at repository root
(`OIP_CONSTITUTION.md`, `AWIS_ARCHITECTURE_FINALIZATION.md`, `AWIS_ARCHITECTURE_BLUEPRINT.md`,
`AWIS_PRD.md`, `IMPLEMENTATION_MASTER_PLAN.md` + its verification report,
`IMPLEMENTATION_KNOWLEDGE_BASE.md`). All were last modified 2026-07-02/03; `AWIS_EEOS.md`
2026-07-08. **None has been substantively touched since.**

---

## 3. What exists in code but not in docs

### 3.1 Production code that is not in git at all

| Path | Files | Git status |
|---|---|---|
| `internal/api/` | 12 | untracked |
| `internal/buildinfo/` | 1 | untracked |
| `cmd/awis-server/` | 5 + `static/` | untracked |
| `web/` | 12 TS sources + build config | untracked |

`git ls-files` returns zero entries for each. This is a documentation fact as much as a version
control one: git history — the only durable record the milestone system relies on for
`HANDOFF` actuals and `TRACEABILITY` commit SHAs — contains no account of this code whatsoever.

### 3.2 Capability surfaces with no normative documentation

| Surface | Code evidence | Documentation status |
|---|---|---|
| HTTP API v1 (7 routes) | `internal/api/router.go:33-41` | **No normative spec.** No `docs/HTTP_API.md`; `06-reference` empty. Covered only by planning-grade, untracked `docs/09-gui-planning/GUI_PHASE1_API_MATRIX.md` (5 of 7 routes; `healthz` and `info` appear in other 09 documents but in no route contract). |
| `awis-server` binary and its 3 flags | `cmd/awis-server/main.go:51-55` | **None of any grade.** `--static-dir` appears in 09-gui-planning as a *card to build*; `--db`, `--addr`, and the `127.0.0.1:8090` default appear in no document that specifies them. |
| Static asset serving / embedded frontend | `cmd/awis-server/static.go` | **None.** |
| Web GUI build pipeline (`esbuild`, `lit-html`, `web/build.mjs`) | `web/package.json`, `web/build.mjs` | **None.** `cmd/awis-server/main.go:55` cites `web/README` as the reference — **that file does not exist.** |
| `internal/buildinfo` version stamping | `internal/buildinfo/buildinfo.go` | **None.** |
| Migration `0007_cancellation_intent` | `internal/storage/migrations/0007_cancellation_intent.sql` | Not in `EVENTLOG_FORMAT.md` (TDS-01) or any normative spec; named only in untracked audit narrative. |
| SDK public surface | `sdk/*.go` (17 source files, ~60 exported identifiers) | **No reference document exists.** `docs/SDK.md` absent; the only account is the M08 milestone module and Go doc comments. `sdk.` appears in exactly two normative specs, incidentally. |
| Binary-level integration tier (`test/integration`, `make integration`, `make contract`, `make bench`, `make release-dry`) | `Makefile`, `test/integration/*.go` | Named in `docs/08-engine-hardening/README.md` narrative only; in no milestone module or spec. `make oip-isolation` appears in **zero** documents. |
| Engine behavioural changes from the hardening program — restart hydration, OCC version read from durable storage, bounded instance read model, schema-version ceiling on `RebuildState`, namespace predicate on `export`, allowlist-based config masking, `retrySched` staging | 30 commits `87d2d9d`…`8a87f70` | Recorded **only** as narrative in `docs/08-engine-hardening/*`. Greps for `hydrat`, `allowlist`, `retrySched`, `integration tier` return hits in 08 and nowhere else in the canonical spec set. No TDS was amended. |
| `onboarding.ts` GUI screen | `web/src/onboarding.ts` | Not in `GUI_PHASE1_SCREEN_SPEC.md` (which specs 5 screens + one evaluated-and-deferred aggregate); appears only in the `GUI_BETA_*` forward-planning documents. |

### 3.3 Surfaces that ARE fully documented (coverage is not uniformly bad)

These were checked and matched, and should not be re-remediated:

- **CLI command set:** all 29 leaf commands and subcommands have a `CLI_CONTRACT.md` §4
  contract section and a `CLI.md` reference section. Parity is complete in both directions at
  the command level. (At the *flag* level it is not — see §4.5.)
- **Event types:** all 12 constants in `internal/core/event.go` are enumerated in
  `docs/EVENTLOG_FORMAT.md`; the doc lists no event type that code lacks.
- **Workflow schema:** every top-level YAML key parsed by `internal/dsl/dsl.go` is present in
  `docs/WORKFLOW_SCHEMA.md`, and vice versa.
- **Plugin protocol:** the three JSON-RPC methods implemented in `internal/plugin/transport.go`
  (`handshake`, `execute`, `shutdown`) are exactly the three specified in `PLUGIN_PROTOCOL.md`.
- **Subprocess protocol:** every JSON envelope field emitted by
  `internal/runner/subprocess/subprocess.go` is specified in `SUBPROCESS_PROTOCOL.md`.
- **GUI screens:** all 5 implemented screens have a written screen spec
  (`GUI_PHASE1_SCREEN_SPEC.md` §§1-5).

---

## 4. What exists in docs but not in code

Two distinct categories, which must not be conflated.

### 4.1 Documents that assert a present-tense fact that is false

| Claim | Location | Code reality |
|---|---|---|
| "TDS files … live here", `docs/06-reference/<NAME>.md`; "A missing TDS here means its milestone has not executed." | `docs/06-reference/README.md` | The directory has contained only that README since 2026-07-03. All six produced TDS files live at `docs/<NAME>.md`. By the document's own stated rule it asserts that M01, M11, M12 and M14 never ran. |
| `awis plugin install` "is planned for M14 (F-5)" | `docs/PLUGIN_GUIDE.md:217` | Implemented in M14-C4, commit `51c2575` (2026-07-10), and fully contract-specified in `CLI_CONTRACT.md:655`. |
| `fast` → `claude-haiku-4-5` | `docs/PROVIDERS.md:72` | Code pins `claude-haiku-4-5-20251001` (`internal/intelligence/adapters/anthropic/anthropic.go:34`). |
| `archive/ENGINEERING_ARCHITECTURE_BLUEPRINT.md` | `docs/07-indices/canonical-reference-map.md:13` | `archive/` contains only `README.md`; the file is at repository root. Dangling pointer inside the canonical index. |

### 4.2 Documents describing work deliberately not yet built

These are forward plans, correctly labelled as such, and are **not** divergence:

- `docs/11-intelligence-architecture/` (8 files) and `docs/12-intelligence-architecture-decision/`
  (7 files) — both state explicitly "no code was written or modified."
- The `state_changes` migration, referenced across 10 GUI-planning documents, absent from
  `internal/storage/migrations/`.
- SSE / WebSocket streaming — specified in the GUI architecture, no implementation
  (`text/event-stream`, `EventSource`, `websocket` all return zero hits in source).
- `docs/V1_BENCHMARKS.md` — an M18 deliverable; M18 has never been materialized beyond its
  README, so the absence is consistent.

### 4.3 A genuinely missing promised document

`docs/ADR_INDEX.md` is specified in `IMPLEMENTATION_MASTER_PLAN.md:399` ("the 15 ADRs live in
the frozen Blueprint; `docs/ADR_INDEX.md` points at them"). It does not exist and never has.

### 4.4 A partially-present type

`TriggerTypeWebhook` is defined in `internal/core/workflow.go:73`, re-exported at
`sdk/trigger.go:21`, and **accepted by the validator** at `internal/validate/validate.go:500`.
There is no dispatch path for it. `AWIS_PRD.md:406` classifies webhook triggers as
`Future (V2)` (FR-WE-15). A V2-scoped feature is therefore half-present in the V1 type system
and accepted by V1 validation, and no document records that state.

### 4.5 CLI flag-level divergence — the normative CLI contract specifies flags that do not exist

Command-level parity is complete (§3.3), but the flag contracts inside `CLI_CONTRACT.md` §4
diverge from the registered flag sets. Because `cmd/awis` parses strictly (`newFlagSet` +
`mustParse`), each documented-but-unregistered flag is not merely absent — supplying it as
documented is a parse error, exit 2.

| Command | `CLI_CONTRACT.md` specifies | Code registers | Verdict |
|---|---|---|---|
| `start` | `--config=<path>` (`:203`); synopsis `awis start [--config=<path>]` (`:194`) | `--tick`, `--namespace` only (`cmd/awis/start.go:65-66`). `start.go:57-59` states `--config` is "ignored/unsupported flag noted in usage only". | Documented flag absent; two registered flags absent from the synopsis |
| `rebuild-state` | `--namespace=<ns>` (`:1279`); synopsis `[--namespace=<ns>]` (`:1273`) | `--all` only (`cmd/awis/rebuild.go:34`), and `--all` is **required** | Documented flag absent; required flag undocumented |
| `export` | `--format`, `--namespace`, `--from`, `--to` (`:1316-1319`) | `--out`, `--namespace` only (`cmd/awis/export.go:37-38`) | Three documented flags absent; `--out` undocumented |
| `status --watch` | "re-print every 5s" (`CLI.md:124`; TDS-07 concurs) | 1s (`cmd/awis/status.go:81`, `time.Sleep(1 * time.Second)`) | Interval mismatch |

The `status --watch` case is distinctive: `cmd/awis/status.go:7-8` **documents its own
deviation in a source comment** — *"per SPEC §2: '1s re-poll'; TDS-07 says 5s re-print for
`--watch`; we implement per SPEC §2 … pin"*. The divergence was known and recorded at the point
of implementation, in the code, and never propagated back to either document.

### 4.6 The two CLI documents contradict each other

`CLI.md` and `CLI_CONTRACT.md` are independently maintained accounts of the same surface, and
they disagree where the contract is wrong:

| Command | `CLI.md` | `CLI_CONTRACT.md` |
|---|---|---|
| `rebuild-state` | `awis rebuild-state --all` (`:721`) — **matches code** | `awis rebuild-state [--namespace=<ns>]` (`:1273`) — does not |
| `export` | `[--out=<dir>] [--namespace=<ns>]` (`:748`) — **matches code** | `[--format=json] [--namespace] [--from] [--to]` (`:1310`) — does not |
| `start` | documents `--config` (`:57`) — does not match code | documents `--config` (`:203`) — does not match code |

`CLI.md` is the *guide*; `CLI_CONTRACT.md` is TDS-07, the *normative contract*. On two of three
disputed commands the non-normative document is the accurate one.

---

## 5. What documentation is duplicated

### 5.1 Byte-identical duplicates between repository root and `docs/`

Verified with `diff -q`:

| File | Root copy | `docs/10-release-candidate-audit/` copy |
|---|---|---|
| `DOCUMENT_DRIFT_REPORT.md` | untracked | identical |
| `ENGINE_READINESS_SCORECARD.md` | untracked | identical |
| `PHASE2_BLOCKERS.md` | untracked | identical |
| `RELEASE_CANDIDATE_AUDIT.md` | untracked | identical |

Four documents exist twice, in two locations, with no statement anywhere of which copy is
authoritative.

### 5.2 Filename collision across five scopes

Five distinct documents are all named `FINAL_VERDICT.md`, with five different MD5 hashes:
repository root, `docs/09-gui-planning/`, `docs/10-release-candidate-audit/`,
`docs/11-intelligence-architecture/`, `docs/12-intelligence-architecture-decision/`.
Additionally the root holds `FINAL_RELEASE_VERDICT.md` and `FINAL_EXECUTIVE_SUMMARY.md`. A
citation of "FINAL_VERDICT.md" is unresolvable without a path.

### 5.3 Root-level report accumulation

30 report/audit/register/verdict documents sit at repository root, 29 of them untracked, dated
2026-09-05 through 2026-09-17. They coexist with the 16 canonical documents from 2026-07-02/03
in the same flat directory, with no naming or directory convention separating specification
from commentary.

---

## 6. What documentation is contradictory

### 6.1 The routing contract contradicts the repository, and has since day one

This is the single most consequential contradiction found.

- `docs/README.md` (the root router): "`06-reference/` — normative TDS specs (created BY
  milestones; IMP §12 `docs/X.md` paths resolve here)".
- `docs/06-reference/README.md`: names seven TDS files and states they resolve to
  `docs/06-reference/<NAME>.md`, then: *"Empty until the creating milestone runs. A missing TDS
  here means its milestone has not executed."*
- **Reality:** `docs/06-reference/` has contained exactly one file — that README — since it was
  created. `EVENTLOG_FORMAT.md`, `WORKFLOW_SCHEMA.md` and `EXPRESSION_GRAMMARS.md` were all
  created on **2026-07-03 at `docs/` root** (git `--diff-filter=A`), the same day the README
  declaring otherwise was written. `SUBPROCESS_PROTOCOL.md`, `PLUGIN_PROTOCOL.md` and
  `CLI_CONTRACT.md` followed on 2026-07-10 at the same wrong location.

The routing rule and the repository disagreed from the first implementation day and were never
reconciled in either direction.

### 6.2 The root router does not acknowledge half the documentation tree

`docs/README.md`'s "Tree" section enumerates `00-foundation` through `07-indices` plus
`../archive/`. Grepping `docs/README.md`, `docs/00-foundation/README.md`,
`docs/02-architecture/README.md` and all five files in `docs/07-indices/` for the strings
`08-engine`, `09-gui`, `10-release`, `11-intelligence`, `12-intelligence`, `internal/api`,
`awis-server` and `web/` returns **zero matches**. 64 documents and four production packages
are invisible to every router and index the documentation system defines.

### 6.3 The PRD does not authorize the API or the GUI for this release

`AWIS_PRD.md` states:
- `:2342` — "Server mode (HTTP API) | V2 when second application is added"
- `:2337` — "Web dashboard / visual designer | Data model must stabilize first (V3)"
- `:116` — "web dashboard is V3"
- `:174` — "NG-4 — Built-in authentication. … Authentication is V3."
- `:406` — webhook trigger: "Future (V2)"

`internal/api/` and `web/` are implemented. Grepping all 38 files of `docs/09-gui-planning/`
for a line relating `V2`/`V3` to PRD scope or authorization returns **zero matches**. The GUI
program neither cited nor contested the PRD's version scoping; it does not reference it at all.
The canonical corpus has not been amended.

### 6.4 Two live, unreconciled release verdicts

- `docs/10-release-candidate-audit/README.md:23` (in `docs/`, though untracked):
  **"Status: REJECTED FOR BETA RELEASE."**
- `FINAL_RELEASE_VERDICT.md` (repository root, 2026-09-05): **"VERDICT: CONDITIONAL PASS"**,
  itself declaring that it *"Supersedes: `FINAL_VERDICT.md` (2026-09-05, PASS WITH RISKS)"*.

Three verdicts, three conclusions, no cross-reference from the `docs/` copy to either newer one.

### 6.5 The milestone ledger contradicts the tree it describes

`docs/05-implementation/STATE.md` records M10–M16 as `PHASE: E-MERGE (blocked on founder)` and
M17 as `C-VERIFY` with `M17-V1` **FAIL** on two independent runs and row 8 explicitly
unadjudicated. The corresponding work is nevertheless present on the branch and has been built
upon by two subsequent programs. The ledger's own `NEXT` instruction — "dispatch a fresh,
independent M17-V1 re-run" — has not been executed in the 27 days since it was written.

### 6.6 The documentation system's own lint gate is red

`bash scripts/docs-lint.sh` exits 1:

```
docs-lint: docs/05-implementation/M15-oip-on-awis/ materialized but files != 7-file contract
docs-lint: docs/05-implementation/M16-anthropic-adapter/ materialized but files != 7-file contract
docs-lint: docs/05-implementation/M17-full-cli-init/ materialized but files != 7-file contract
```

M16 has 3 of the 7 contract files (missing `HANDOFF.md`, `TRACEABILITY.md`, `DEPENDENCY_MAP.md`,
`AI_EXECUTION_CONTEXT.md`); M17 has 5; M15 has 8 (two extra). The contract is stated in
`docs/05-implementation/README.md` and mechanically checked — and the check has been failing
without effect.

---

## 7. What documentation is orphaned

**Definition used:** reachable from no router, index, or parent document in the canonical tree.

| Orphan | Files | Evidence |
|---|---|---|
| `docs/08-engine-hardening/` | 5 | Referenced by no router or index (§6.2) |
| `docs/09-gui-planning/` | 38 | Same; additionally untracked |
| `docs/10-release-candidate-audit/` | 6 | Same; additionally untracked |
| `docs/11-intelligence-architecture/` | 8 | Same; additionally untracked |
| `docs/12-intelligence-architecture-decision/` | 7 | Same; additionally untracked |
| **Total** | **64** | |

Each of these directories has an internal `README.md` that indexes its own contents competently.
None of them is indexed *from above*. They are self-consistent islands.

`docs/06-reference/` is a distinct kind of orphan: a registered, routed-to directory that is
empty, and whose README's own rule makes its emptiness mean something false (§6.1).

Within the tracked tree, 33 files are referenced by filename from no other document — 31
execution cards under `docs/05-implementation/M*/cards/` and `docs/edr/edr-008`, `edr-009`.
The cards are reachable by card ID (e.g. `M14-C1` appears in `STATE.md`) and the EDRs by EDR ID,
so these are weak orphans only — navigable, but not linked.

---

## 8. What documentation is stale

Ordered by how badly the staleness misleads a reader.

| Document | Stale claim | Contradicting code |
|---|---|---|
| `docs/09-gui-planning/DASHBOARD_GATE_STATUS.md` | Cards `E-G4-4/5` (workflow + instance routes) and `E-G4-6` (event route) "⬜ Not started" | All three are implemented: `internal/api/workflows.go`, `instances.go`, `events.go` |
| `docs/09-gui-planning/GUI_PHASE1_API_MATRIX.md:92` | "No static-file serving in `cmd/awis-server` — `GET /` 404s today … **Blocks shipping**" | `cmd/awis-server/static.go` implements it, with `static_test.go` |
| `docs/09-gui-planning/GUI_PHASE1_BACKLOG.md:14` | `--static-dir` flag is the "**Blocking** … highest-leverage item in this entire document" | Implemented at `cmd/awis-server/main.go:55` |
| `docs/10-release-candidate-audit/README.md:23` | "REJECTED FOR BETA RELEASE", citing SEC-02 (volatile cancellation intent) among the blockers | SEC-02's fix shipped as migration `0007_cancellation_intent.sql` ("RC-2") in `8a87f70`, after the audit was written. SEC-04 (subprocess env leak) is likewise closed by `buildSubprocessEnv` at `internal/runner/subprocess/subprocess.go:332`. |
| `docs/PLUGIN_GUIDE.md:217` | `awis plugin install` "is planned for M14" | Shipped 2026-07-10 (`51c2575`) — stale for 69 days |
| `docs/PROVIDERS.md:72` | Model pin `claude-haiku-4-5` | `claude-haiku-4-5-20251001` |
| `docs/CLI_CONTRACT.md:203,1279,1316-1319` | `start --config`, `rebuild-state --namespace`, `export --format/--from/--to` | None registered; each is a strict-parse error (§4.5) |
| `docs/CLI.md:124` | `--watch` re-prints every 5s | 1s (`cmd/awis/status.go:81`) |
| `docs/05-implementation/STATE.md` | M17 `C-VERIFY`, V1 re-run "not dispatched this session" | Two subsequent programs have shipped on top of M17; no re-run in 27 days |
| `docs/06-reference/README.md` | "Empty until the creating milestone runs" | Six of seven TDS milestones ran (§6.1) |
| `cmd/awis-server/main.go:55` (code→doc) | Cites `web/README` | No such file |

---

## 9. The evidence-backed source of documentation divergence

The timeline, taken from `git log` dates for tracked material and filesystem mtimes for
untracked material, resolves this question directly.

| Window | Code activity | Documentation activity | Governing system |
|---|---|---|---|
| 2026-07-02 → 07-03 | — | Canonical corpus authored (PRD, Blueprint, Finalization, Constitution, IMP, IKB); `docs/` IKB tree created (`441f017`) | — |
| 2026-07-03 → 08-21 | M00–M17 implemented | 174 milestone-module files; 6 TDS specs written **to the wrong directory** from day one | EEOS / milestone modules |
| 2026-08-26 → 09-05 | **30 commits**: B-0…B-31 defect closure, security fixes, binary-level integration tier built from nothing, migration `0007` | **5 narrative files** in `docs/08-engine-hardening/` (2 commits). Zero milestone modules. Zero TDS amendments. | **none** |
| 2026-08-30 → 09-07 | **HTTP API, server binary, `buildinfo`, full web GUI** — all uncommitted | **59 untracked files** across `docs/09`–`docs/12` | **none** |
| 2026-09-05 → 09-17 | — | 30 root-level reports, 29 untracked | **none** |

Three structural facts follow:

1. **Divergence began 2026-07-03**, on the first implementation day, when TDS specs were written
   to `docs/` root while `docs/06-reference/README.md` — authored the same day — declared they
   would be written elsewhere. Nothing detected this, because `scripts/docs-lint.sh` checks only
   the shape of `docs/05-implementation/M*/` and the existence of `STATE.md` and `V-COMMON.md`.
   It has no check that a TDS exists, that it is current, or that it is where the router says.

2. **The milestone system stopped, and nothing replaced it.** The last milestone commit is
   `7146214` (2026-08-21). M17 was never closed: `M17-V1` failed twice under independent
   verification, row 8 was never adjudicated, and M18 was never materialized. Every subsequent
   program — engine hardening, then GUI/API — ran with no milestone module, no
   `IMPLEMENTATION_SPEC`, no `VALIDATION_CHECKLIST`, no `HANDOFF`, and no `TRACEABILITY` row.
   Those programs produced documentation, and good documentation of its kind, but it was
   *narrative* (what we did and why) rather than *normative* (what the system now is). No TDS
   was amended by either program.

3. **The documentation system has no mechanism for admitting new documentation.**
   `docs/08` through `docs/12` were created by writing directories. Nothing registers them;
   `docs/README.md`'s tree, `docs/07-indices/canonical-reference-map.md`,
   `cross-reference-index.md`, `dependency-index.md` and `traceability-matrix.md` are all silent
   about their existence. The system had a loading protocol
   (`AI_EXECUTION_CONTEXT` → `HANDOFF` → `06-reference`) and a supersession rule, but no
   registration rule and no amendment rule.

**Answer: E — a combination**, with the components separable and unequally weighted:

- **C (weaknesses in the original documentation system) — the enabling cause and the earliest.**
  Onset 2026-07-03. The routing defect, the absence of a TDS-currency check, the absence of a
  registration mechanism, and a lint gate that has been red without consequence.
- **A (capabilities implemented without documentation) — the largest volume.** Concentrated
  entirely in the two post-2026-08-26 programs: the HTTP API, server, `buildinfo`, web build,
  migration `0007`, and every behavioural change in §3.2.
- **D (GUI planning documentation) — a contributing cause, and a comparatively minor one.**
  See §10.
- **B (documentation created without implementation) — mostly benign.** `docs/11`, `docs/12`
  and the GUI forward plans declare their own non-implementation. The only genuine instances
  are `docs/ADR_INDEX.md` (§4.3) and the half-present webhook trigger type (§4.4).

---

## 10. Is GUI planning the primary cause, a contributing cause, or not a cause?

**A contributing cause. Not the primary cause. The evidence points the opposite way from the
intuitive reading.**

Against GUI planning being the primary cause:

1. **Divergence predates it by two months.** The `06-reference` routing defect dates to
   2026-07-03 (§6.1); GUI planning begins 2026-08-30.
2. **The GUI is the best-documented recent surface in the repository.** All 5 implemented
   screens have a written screen spec. 5 of 7 API routes have a route contract with query
   parameters. The engine-hardening program, by contrast, changed engine semantics in 30 commits
   and produced no specification at all.
3. **The undocumented backlog is not GUI-shaped.** The SDK public surface, the integration test
   tier, migration `0007`, restart hydration, the OCC read path, the schema-version ceiling and
   the config-masking allowlist all predate or are independent of the GUI program.

Where GUI planning *did* contribute:

1. **Volume without registration.** 38 documents entered `docs/` with no index entry, no
   authority position, and no git commit — the largest single contribution to the orphaned mass
   in §7.
2. **It bypassed the authority chain rather than amending it.** The PRD scopes the HTTP API to
   V2 and the dashboard to V3 (§6.3). The GUI program built both without citing, contesting, or
   amending that scoping — zero references across 38 files. The canonical corpus was left
   asserting a V1 scope the repository no longer matches.
3. **It generated the most stale documentation.** Three of the five worst staleness findings in
   §8 are GUI-planning documents describing as "not started" or "blocking" work that has since
   shipped.

The precise formulation the evidence supports: GUI planning did not *cause* the divergence; it
**inherited a documentation system with no mechanism for absorbing it, and then operated at a
volume that made the absence of that mechanism visible.** The same is true of the
engine-hardening program, which produced less documentation and diverged more.

---

## 11. Did divergence originate from weaknesses in the original documentation system?

**Yes — partly, and demonstrably. This is the earliest identifiable cause.** Five specific
weaknesses, each evidenced:

1. **A routing rule with no enforcement.** `docs/06-reference/README.md` declares where TDS
   specs live. Six specs were written elsewhere, three of them the same day. Nothing detected
   it in 76 days.

2. **A self-invalidating semantic.** The same README states "a missing TDS here means its
   milestone has not executed." Because the specs went elsewhere, the directory permanently
   asserts a falsehood about four executed milestones. A rule that cannot fail safely.

3. **A lint gate with the wrong scope.** `scripts/docs-lint.sh` checks four things: milestone
   README existence, the 7-file module contract, `archive/` citation hygiene, and the existence
   of `STATE.md`/`V-COMMON.md`. It checks nothing about the TDS layer, nothing about index
   currency, nothing about whether code has documentation. And it is currently **failing** on
   three milestones (§6.6) with no consequence — the gate exists and does not gate.

4. **No registration mechanism.** The system defines a loading protocol, an authority order, and
   a supersession rule. It defines no way to *add* a documentation domain. Consequently five
   directories and 64 files exist inside `docs/` that no part of `docs/` knows about.

5. **No amendment path from code change back to specification.** The milestone contract routes
   TDS creation through milestones only. Once milestones stopped, there was no defined way for a
   behavioural change to reach a spec. The engine-hardening program is the proof: it changed
   restart semantics, cancellation durability, read-model bounds and secret masking, wrote four
   thorough narrative reports about doing so, and amended zero specifications — because the
   system provides no route for that.

Weakness (1) and (2) are *original* defects, present on 2026-07-03. Weaknesses (3), (4) and (5)
are *omissions* that became load-bearing only once work moved outside the milestone structure on
2026-08-26.

---

## 12. Documentation that must be created before further development

The minimum set, derived from §3.2 and §6 only. Each item exists because a capability is
present in Tier 1 code with no Tier 2 representation, or because a canonical document currently
states something false.

**Tier A — normative specifications for implemented capability that has none**

1. **HTTP API contract** — the 7 routes of `internal/api/router.go:33-41`: method, path, query
   parameters, response schema, error envelope, pagination semantics, status codes. Equivalent
   in grade to `CLI_CONTRACT.md`, which is the existing model for this.
2. **`awis-server` operational contract** — the three flags, their defaults
   (`./awis-server.db`, `127.0.0.1:8090`), static-asset resolution order (embedded vs
   `--static-dir`), lifecycle and shutdown behaviour.
3. **SDK public surface reference** — the ~60 exported identifiers in `sdk/`. This is the
   boundary `apps/oip` is mechanically constrained to consume; it is the only public API of the
   platform and has no reference document.
4. **Web GUI build and deployment note** — the `esbuild`/`lit-html` pipeline, how
   `cmd/awis-server/static/` is produced, and the dev loop. The code already cites a
   `web/README` that must be written to exist.

**Tier B — amendments that make existing canonical documents true**

5. **`EVENTLOG_FORMAT.md` / storage spec** — add migration `0007` and the
   `cancellation_reason` / `cancellation_compensate` columns.
6. **Engine semantics amendment** — fold the behavioural changes of the 30 hardening commits
   (restart hydration, durable OCC version read, bounded instance read model, `RebuildState`
   schema-version ceiling, namespace predicate on `export`, allowlist config masking) out of
   `docs/08-engine-hardening/` narrative and into the normative layer.
7. **`docs/06-reference/README.md`** — reconcile with reality: either move the six TDS files or
   restate the path mapping. Until this is done the canonical index asserts that four executed
   milestones did not execute.
8. **`docs/README.md` and `docs/07-indices/*`** — register `08` through `12`, and correct the
   dangling `archive/ENGINEERING_ARCHITECTURE_BLUEPRINT.md` pointer.
9. **`docs/CLI_CONTRACT.md` §4 flag reconciliation** — `start`, `rebuild-state` and `export`
   (§4.5). This is the normative CLI contract, and following it as written produces exit-2
   parse errors on three commands. Also settle the `--watch` interval, which code, `CLI.md` and
   TDS-07 currently answer three ways (§4.5, §4.6).
10. **`docs/PLUGIN_GUIDE.md:217`** and **`docs/PROVIDERS.md:72`** — correct the two false
    present-tense claims.
11. **`docs/ADR_INDEX.md`** — create it, or remove the IMP §399 promise.

**Tier C — the scope question the corpus cannot currently answer**

12. **A recorded position on PRD version scoping.** `AWIS_PRD.md` scopes the HTTP API to V2 and
    the web dashboard to V3; both are implemented. No document in the repository — canonical,
    planning, or audit — records a decision to change that. Whatever the answer is, the corpus
    must state it, because every downstream document inherits the ambiguity.

---

## Method, coverage and limits of this analysis

**Established mechanically** (fully reproducible): all file inventories and counts; git
tracked/untracked status; git creation and last-modified dates; `docs-lint.sh` output;
byte-identical duplicate detection (`diff -q`, `md5sum`); route, command, migration, event-type,
step-type and YAML-key extraction from source; presence/absence greps across `docs/`.

**Established by direct comparison**: CLI commands and flags ↔ `CLI_CONTRACT.md`/`CLI.md`;
event types and envelope fields ↔ `EVENTLOG_FORMAT.md`; DSL keys and enums ↔
`WORKFLOW_SCHEMA.md`/`DSL.md`; expression operators and scopes ↔ `EXPRESSION_GRAMMARS.md`;
plugin JSON-RPC methods, envelope fields and error codes ↔ `PLUGIN_PROTOCOL.md`; subprocess
envelope and error codes ↔ `SUBPROCESS_PROTOCOL.md`; GUI screens ↔
`GUI_PHASE1_SCREEN_SPEC.md`; API routes ↔ `GUI_PHASE1_API_MATRIX.md`; provider set, port
methods and model IDs ↔ `PROVIDERS.md`.

**Candidate findings examined and discarded** (recorded so they are not re-raised):

- `WORKFLOW_SCHEMA.md`'s appendix `WorkflowInstance` fields (`instance_id`, `definition_id`,
  `definition_version`, `status`, `current_steps`, `variables`, `started_at`, `updated_at`,
  `completed_at`) are **implemented** — at `internal/core/instance.go:10-28`, not in
  `internal/dsl/`. Not a gap.
- `EVENTLOG_FORMAT.md`'s optional `StepCompleted` usage triple (`adapter`, `model`,
  `tokens_used`, ADJ-8) is **implemented** at `internal/engine/emit.go:37-48`. Not a gap.
- `retry.retryable_errors` is parsed (`internal/dsl/dsl.go:77`) and **is** specified in the
  normative TDS-02 (`WORKFLOW_SCHEMA.md:48,106-108`); it is merely not itemized in the
  non-normative `DSL.md` field table. A guide-layer omission, not divergence.
- `oip.db` in `docs/07-indices/cross-reference-index.md:10` is prose describing the OIP/runtime
  FTS ownership split, not a file-path pointer. Not a dangling reference.

**Limits, stated explicitly:**

- Within the normative specs, the checks were structural: enumerated keys, fields, operators,
  methods, error codes, commands and flags. Prose semantics were not exhaustively verified —
  §3.3 establishes that the six specs cover the right surfaces and enumerate the right symbols;
  it does not establish that every sentence of `CLI_CONTRACT.md`'s 1497 lines is accurate. §4.5
  shows what that class of check finds when it is run: the flag-level pass over the same
  document that passed the command-level pass surfaced four divergences.
- `docs/05-implementation/`'s 174 milestone-module files were assessed for contract completeness
  (via `docs-lint`) and for ledger currency (via `STATE.md`), not read individually.
- Tier 4 root reports were read selectively for corroboration. Where one is cited (§6.4), the
  underlying claim was re-verified against Tier 1 or Tier 2 before use.
- The four untracked code packages were analysed from the working tree. If that tree is ever
  reset, every §3 finding about them becomes moot in the way that matters most.
