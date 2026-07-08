# CLAUDE.md — Agent System Bridge

> **AWIS EEOS override (canonical, founder-approved 2026-07-08 — AWIS_EEOS.md §15 D4):**
> in any session whose task references EEOS, a milestone (MXX), or Execution Cards,
> `/EEOS.md` is the P0 protocol; skip the `.agents/` loading protocol, request routing,
> and skill announcements entirely. The AWIS canonical corpus and the AEO govern; the
> generic rules below apply only to non-EEOS work in this workspace.

This file connects Claude Code to the `.agent` / `.agents` folder so the same agent/skill/rules system works here as in other IDEs (Cursor, Windsurf, Copilot, etc.).

---

## Folder Layout

```
.agents/          ← canonical folder (ag-kit + winter-box merged)
  agent/          ← agent definitions (.md persona files)
  skills/         ← modular skill packs (each has SKILL.md index)
  rules/          ← workspace rules (always-on protocols)
  memory/         ← persistent agent memory
  workflows/      ← multi-step workflow definitions
  scripts/        ← utility scripts
.agent → .agents  ← symlink for IDE compatibility
```

---

## Loading Order (Priority)

1. **P0 — Rules** (`.agents/rules/`): Always-on. Load ALL rule files before any task.
2. **P1 — Agent** (`.agents/agent/`): Load the most relevant agent `.md` for the current task.
3. **P2 — Skills** (`.agents/skills/`): Load specific skill sections on demand — read `SKILL.md` index first, then only the relevant sections.

---

## Always-On Rules

Read these rule files at the start of every session:

- `.agents/rules/core-protocol.md` — mandatory agent & skill loading protocol
- `.agents/rules/universal-rules.md` — universal coding standards
- `.agents/rules/output-quality-contract.md` — output quality requirements
- `.agents/rules/token-hygiene.md` — context efficiency rules
- `.agents/rules/request-routing.md` — how to route requests to agents/skills
- `.agents/rules/skill-routing-protocol.md` — skill selection logic
- `.agents/rules/code-rules.md` — code quality rules
- `.agents/rules/global-engineering-standards.md` — engineering standards

---

## Skill Announcement (Required)

Before applying any skill, announce it:

```
📚 Using skill: @[skill-name]...
```

---

## Invoking Agents

Reference agents by name from `.agents/agent/`. Example:

- `@frontend-specialist` → `.agents/agent/frontend-specialist.md`
- `@debugger` → `.agents/agent/debugger.md`
- `@orchestrator` → `.agents/agent/orchestrator.md`

## Available Agents

backend-specialist, code-archaeologist, database-architect, debugger, devops-engineer,
documentation-writer, explorer-agent, frontend-specialist, game-developer, mobile-developer,
orchestrator, penetration-tester, performance-optimizer, product-manager, product-owner,
project-planner, qa-automation-engineer, quality-enforcer, security-auditor, senior-engineer,
seo-specialist, system-investigator, test-engineer

## Available Skills

api-patterns, app-builder, architecture, architecture-analyst, bash-linux, batch-operations,
behavioral-modes, brainstorming, clean-code, code-review-checklist, code-review-graph,
code-synthesizer, context-compression, coordinator-mode, database-design, debugging-master,
dependency-analyzer, deployment-procedures, design-spec, documentation-templates,
documentation-writer, frontend-architecture, frontend-design, game-development, geo-fundamentals,
i18n-localization, intelligent-routing, lint-and-validate, mcp-builder, memory-system,
mobile-design, nextjs-react-expert, nodejs-best-practices, parallel-agents,
performance-optimizer, performance-profiling, plan-writing, powershell-windows,
python-patterns, red-team-tactics, refactoring-specialist, research-engine, rust-pro,
security-auditor, seo-fundamentals, server-management, simplify-code, skill-forge,
skillify, systematic-debugging, system-auditor, tailwind-patterns, task-planner,
tdd-workflow, test-generator, testing-patterns, verify-changes, vulnerability-scanner,
webapp-testing, web-design-guidelines



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
