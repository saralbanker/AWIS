# Winter-Box Architecture (2026 rebuild)

A lean, cross-tool `.agent/` kit: works with Claude Code, Antigravity/Gemini
CLI, or any harness that reads a `.agent/` directory in a repo.

## Why this shape

Earlier versions of this kit shipped 23 persona "agents," 11 slash-command
workflows, 5 rule files (including a forced clarification gate), and ~50
skills — much of it framework-version-pinned, unenforced invented
frontmatter, or negative-constraint bans that fight modern models more than
they help. This rebuild removed all of that in favor of a few things that
actually change outcomes: a short, high-agency baseline; skills that encode
real, non-obvious procedure instead of persona flavor; real progressive
disclosure so token cost stays flat; and governance (`last_verified` /
`kill_condition`) so decay gets caught instead of accumulating silently.

## Directory structure

```plaintext
CLAUDE.md                   # root pointer — what makes Claude Code auto-load this kit
GEMINI.md                   # root pointer — same, for Gemini CLI / Antigravity
.claude/agents/              # (installed by `init`) Claude Code subagent templates
.agent/
├── AGENTS.md              # canonical baseline — read this first
├── skills/
│   └── <skill>/
│       ├── SKILL.md         # name + description (discovery) + <100-line body
│       ├── reference/       # loaded only when SKILL.md says to
│       ├── data/             # (ui-ux-design only) curated CSV reference data
│       └── scripts/          # executable helpers, where one exists
├── eval/
│   ├── README.md            # how to use the task set
│   └── tasks/                # ~17 prompts + pass/fail rubrics targeting specific regressions
└── scripts/
    ├── audit_staleness.py    # flags skills missing/overdue on last_verified
    ├── auto_preview.py       # start/stop/status a local dev server
    └── session_manager.py    # detect project stack, file counts, feature dirs
```

`CLAUDE.md`/`GEMINI.md` live at the project root, not inside `.agent/`,
because that's the only place each harness actually auto-loads them from —
a harness-specific adapter nested inside `.agent/` is invisible to a
harness that doesn't already know to look there. Both are short pointers
into `.agent/AGENTS.md`; real content stays in one place.

## Skills (15)

| Skill | Focus |
|---|---|
| `task-planner` | DAG task decomposition, dependencies, acceptance criteria |
| `code-synthesizer` | Test-first implementation of new code with a testable contract |
| `debugging-master` | Evidence-first root-cause diagnosis |
| `architecture-analyst` | Codebase mapping, coupling, anti-patterns |
| `system-auditor` | Multi-service incident investigation |
| `test-generator` | Boundary-value test suites, AAA structure |
| `security-auditor` | OWASP-pattern review, secrets, injection |
| `performance-optimizer` | Measure-first profiling and optimization |
| `refactoring-specialist` | Behavior-preserving cleanup, Fowler's taxonomy |
| `research-engine` | Sourced, synthesized technical comparisons |
| `dependency-analyzer` | CVE/license/currency audits |
| `documentation-writer` | Audience-first docs |
| `skill-forge` | Meta-skill: forges new skills in this kit's own lean format |
| `ui-ux-design` | Searchable design-reference data (styles, color, per-stack conventions) |
| `subagent-protocol` | When/how to delegate to a subagent; ships 2 example Claude Code subagent templates |

Every `SKILL.md` follows the same shape: `name` + `description` frontmatter
for discovery, `last_verified` + `kill_condition` for governance, and a
body under ~100 lines (when to use → procedure → what to avoid →
checklist). Anything longer — command references, code samples, edge
cases — lives in `reference/`, loaded only when the skill body points to
it, not on every activation. Several skills also carry an explicit
**anti-trigger** section — a note on when *not* to engage, added where a
skill's description alone risked firing on the wrong task (e.g.
`code-synthesizer`'s test-first gate is a bad fit for a purely visual/
creative deliverable, so it says so).

## Governance

Run `python .agent/scripts/audit_staleness.py` periodically — it flags any
skill missing `last_verified`/`kill_condition` or overdue for review
(default: 180 days). This is separate from `skill-forge`'s own
`ttl_watcher.py`, which expires *forged* skills unused for 30+ days — the
audit script is calendar-based review hygiene, the watcher is usage-based
cleanup.

Before a release, walk through `.agent/eval/tasks/*` — each targets one
specific failure mode this rebuild fixed (forced clarification gates,
guessed fixes, stale framework advice, scope-creep refactors, persona
theater, and so on). A change that turns a passing task into a failing one
is a regression, not a style choice.
