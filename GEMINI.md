# Gemini CLI / Antigravity — this project

See `.agent/AGENTS.md` for the full baseline. This file exists only because
harnesses in this family auto-load a root-level `GEMINI.md` the same way
Claude Code auto-loads `CLAUDE.md` — it does not scan `.agent/` on its own,
so this pointer is what makes the rest of the kit actually activate here.
Keep this file short; put real content in `.agent/AGENTS.md` and the
skills, not here.

- Skills in `.agent/skills/` are plain Markdown with YAML frontmatter; treat
  `name` + `description` as the trigger signal and load the rest (body,
  then `reference/`, then `scripts/`) only as needed.
- If this harness offers native subagent or model-tier routing (e.g. a
  fast/cheap tier for quick lookups vs. a stronger tier for hard reasoning),
  use that directly — there is no hand-rolled orchestrator in this kit to
  consult instead. See `.agent/skills/subagent-protocol/SKILL.md` for how
  to design a delegation well regardless of the exact mechanism this
  harness exposes; the `reference/templates/antigravity-gemini/` note there
  is a best-effort starting point, not a verified spec — check this
  harness's current docs for its actual subagent/config format before
  relying on it.
- "Search the codebase" / "run tests" in skill text maps to whatever tools
  this harness exposes — skills intentionally don't hardcode tool names.
