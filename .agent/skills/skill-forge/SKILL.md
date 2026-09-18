---
name: skill-forge
description: >
  Use when a recurring, real task has no existing skill covering it and the
  gap is worth encoding permanently — not for one-off questions, and not
  when an existing skill already covers the domain (extend that instead).
last_verified: 2026-08-30
kill_condition: revisit if this kit's skill file format (frontmatter fields, reference/ split) changes — this skill must forge skills matching the current format, not an old one.
---

# Skill Forge

Skill-forge is the meta-skill: it creates other skills, in this kit's own
format. Every skill it forges must follow the same shape as the skills
already in `.agent/skills/` — a short `SKILL.md` plus a `reference/` file
for depth — never the old heavy format (9 frontmatter fields, mandatory
Mermaid diagrams, 500-line bodies) that this kit was rebuilt away from.

## When to use
All of these must be true: no existing skill covers the domain even
partially (check `.agent/skills/` first — extend, don't duplicate); the
task is complex or recurring, not a one-off question; and there's real,
encodable domain knowledge — concrete heuristics or a non-obvious
procedure, not something a frontier model already knows cold.

## Procedure
1. **Check for overlap.** List `.agent/skills/`. If something already
   covers this domain even loosely, extend it instead of forging a
   duplicate.
2. **Research the domain** just enough to extract 3-6 concrete, non-obvious
   decision points or heuristics — not generic advice a model already
   produces unprompted.
3. **Write `SKILL.md`** in this kit's format only:
   - Frontmatter: `name`, `description` (states when to use it and, where
     relevant, when not to), `last_verified` (today's date),
     `kill_condition` (one line — what fact going stale invalidates this).
     No other frontmatter fields — nothing else in this kit reads them.
   - Body, under ~100 lines: `## When to use`, `## Procedure` (5-7 concrete
     steps), `## Avoid` (2-3 real failure modes), `## Checklist` (4-6
     verifiable items). Reference capabilities ("search the codebase"),
     never a specific tool name.
   - Anything longer — code samples, command references, edge cases —
     goes in `reference/playbook.md`, linked from the body's last line.
4. **Validate against the checklist below** before writing the file.
5. **Write the file** to `.agent/skills/<name>/SKILL.md` (+ `reference/`
   if needed), then record it in `.agent/skills/.skill_usage.json` as
   `{ "<name>": "<ISO-8601 timestamp>" }` so the TTL watcher can track it.
6. **Use the new skill immediately** to complete the original request.

## Avoid
Forging a skill for a one-off question; copying the old 9-field/8-section
template from anywhere it still appears in examples; restating knowledge a
current frontier model already has (a whole skill for "use const not var"
is dead weight, not a capability); writing a skill with no
`kill_condition`, which is how a kit quietly rots unnoticed.

## Checklist
- [ ] No existing skill already covers this domain
- [ ] Frontmatter is exactly `name` / `description` / `last_verified` /
      `kill_condition` — nothing invented
- [ ] Body is under ~100 lines; depth lives in `reference/`, not the body
- [ ] No hardcoded tool names, model names, or framework-version pins in
      the body (those belong in `reference/`, dated, if unavoidable)
- [ ] Procedure encodes a real, non-obvious decision — not generic advice
- [ ] Recorded in `.skill_usage.json` after writing

See `reference/annotated-example.md` for a worked good-vs-bad comparison,
and `scripts/ttl_watcher.py` for auto-expiring forged skills unused for 30+
days (core skills listed in `scripts/README.md` are exempt).
