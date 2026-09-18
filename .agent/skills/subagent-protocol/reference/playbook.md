# Subagent Protocol — Playbook

## What went wrong before (why this skill exists)

An earlier version of this kit shipped `orchestrator.md`, a persona that
referenced specific fake subagents — "Explore (Haiku)," "Plan (Sonnet)" —
and a hand-rolled decision tree for picking between them. Neither of those
subagents existed in the harnesses this kit targets; the routing table was
guessing at internals it had no way to verify and had no mechanism to
notice when it was wrong. The fix isn't a better routing table — it's not
hand-rolling one at all. Use whatever delegation mechanism the current
harness actually documents, and treat this playbook as guidance for using
it well, not a replacement for it.

## Context-sharing model, and why it matters

Delegation mechanisms generally come in two shapes:

- **Fresh delegate**: starts with zero memory of the parent conversation.
  Everything it needs — file paths, prior decisions, constraints, what's
  out of scope — must be in the prompt you write. This is the more common
  case; assume it by default.
- **Context-inheriting delegate** (where a harness offers one): starts with
  the parent's full conversation so far. Prompts to this kind are
  *directives* ("do X"), not re-explanations of context it already has —
  padding it with background it already saw wastes tokens and can read as
  condescending to no one but still costs real tokens.

Check which kind you have before writing the prompt. Using a fresh-delegate
prompt style ("here's everything you need...") on a context-inheriting
delegate is wasteful but harmless; using a context-inheriting style ("you
know why we're doing this...") on a fresh delegate produces a confused,
under-specified result.

## A worked handoff prompt, bad vs. good

```
Bad (assumes shared context, vague return contract):
"Look into why the auth thing is broken and fix it."

Good (self-contained, states the return contract):
"The login endpoint at src/routes/auth.ts returns 500 for valid
credentials as of the change in commit a3f21. Reproduce it, find the
root cause (not just where the error surfaces), and report: the root
cause with file:line evidence, the fix you'd apply, and whether you
need explicit approval before touching anything outside auth.ts.
Don't fix it yet — report first."
```

The good version works whether the delegate is fresh or context-inheriting,
because it doesn't depend on shared memory to be understood — it's
correct either way, and that portability is the entire point.

## Parallel dispatch

Only batch subagents whose work is genuinely independent — each one should
be answerable without knowing what the others find. If subagent B's prompt
would change based on subagent A's result, they're sequential, not
parallel, no matter how tempting it is to fire them all at once.

## Templates

`reference/templates/claude-code/` — two subagent definitions in Claude
Code's real, current `.claude/agents/*.md` format (verified against this
session's own environment): `code-reviewer.md` (read-only review pass) and
`verification-runner.md` (runs tests/lint/build, reports real output, does
not attempt fixes). Both are least-privilege by design — read-only roles
get read-only tools, matching this skill's own advice.

`reference/templates/antigravity-gemini/` — a best-effort starting point
for the same two roles under Gemini CLI / Antigravity conventions. This one
carries real uncertainty: unlike the Claude Code templates, this wasn't
verified against that harness's current documentation firsthand. Treat it
as a starting shape, not a spec — check the harness's own docs for the
current subagent/config format before relying on it in production.
