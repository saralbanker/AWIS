# Antigravity / Gemini CLI subagent templates — best effort, unverified

Unlike the Claude Code templates next to this file, these were **not**
verified against Antigravity's or Gemini CLI's current documentation
firsthand — this kit doesn't have first-hand confirmed knowledge of that
harness's exact subagent/config file format. Treat what follows as a role
brief to adapt into whatever format that harness actually requires today,
not a working config file to copy verbatim. Check the harness's current
docs before relying on this.

## code-reviewer (role brief)

**Mission**: independent, read-only review pass on a diff or set of files.
**Scope**: read/search only — no file-write access, no execution beyond
what's needed to read state (e.g. `git diff`).
**Trigger**: an explicit review request, or before merging a non-trivial
change.
**Output**: findings ranked most-severe first, each with a concrete
failure scenario — not a style opinion stated as a defect. No edits made.

## verification-runner (role brief)

**Mission**: run the project's real test/lint/build commands and report
pass/fail with actual output.
**Scope**: execute commands, read config/output — no source edits.
**Trigger**: after a change is ready to verify, before treating it as done.
**Output**: which commands ran, pass/fail per command, real failure output
for anything that failed. No attempt to fix failures — that's a separate
step for the caller.

## If you formalize these for this harness

Once you've confirmed the actual config format this harness expects,
consider replacing this README with real config files (matching the
Claude Code templates' pattern), and update this skill's
`last_verified`/`kill_condition` to reflect that the format is now
confirmed rather than best-effort.
