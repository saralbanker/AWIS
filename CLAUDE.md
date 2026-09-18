# CLAUDE.md — Agent System Bridge

> **AWIS EEOS override (canonical, founder-approved 2026-07-08 — AWIS_EEOS.md §15 D4):**
> in any session whose task references EEOS, a milestone (MXX), or Execution Cards,
> `/EEOS.md` is the P0 protocol; skip the `.agent/` loading protocol, request routing,
> and skill announcements entirely. The AWIS canonical corpus and the AEO govern; the
> generic rules below apply only to non-EEOS work in this workspace.

---

# Claude Code — this project

See `.agent/AGENTS.md` for the full baseline. This file exists only because
Claude Code auto-loads a root-level `CLAUDE.md` — it does not scan `.agent/`
on its own, so this pointer is what makes the rest of the kit actually
activate in Claude Code. Keep this file short; put real content in
`.agent/AGENTS.md` and the skills, not here.

- Skills in `.agent/skills/` map directly onto Claude Code's native Skill
  mechanism: `name` + `description` drive discovery, the body loads on
  trigger, `reference/` and `scripts/` load lazily. No translation needed.
- Subagents ship with this kit in `.claude/agents/` (Claude Code's native
  subagent location): `code-reviewer` (read-only review pass) and
  `verification-runner` (runs tests/lint/build, reports real output). See
  `.agent/skills/subagent-protocol/SKILL.md` before writing new ones.
- "Search the codebase" / "run tests" in skill text maps to whatever tools
  this session actually exposes (Grep, Bash, etc.) — skills intentionally
  don't hardcode tool names so they survive tool renames.

---

# Model Allocation Policy (Frozen)

The implementation phase of AWIS Baseline V1 uses a fixed model allocation.

## Allowed

- Sonnet
- Haiku

## Forbidden

- Opus
- Fable (except human-requested architecture reviews)

Implementation, testing, verification, refactoring, migrations, documentation updates, and repository changes MUST be performed using Sonnet.

Repository search, symbol lookup, dependency discovery, and lightweight documentation tasks MAY use Haiku.

Opus MUST NOT be selected automatically.

Fable MUST remain dormant unless explicitly invoked by the human for architectural reasoning or merge approval.

This policy is frozen for the remainder of Baseline V1.
