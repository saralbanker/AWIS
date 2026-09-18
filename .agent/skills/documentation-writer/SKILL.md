---
name: documentation-writer
description: >
  Use when writing technical documentation — READMEs, API references,
  architecture docs, or inline comments. Not for implementation
  (code-synthesizer) or design decisions themselves (architecture-analyst).
last_verified: 2026-08-30
kill_condition: n/a — audience-first writing and docs-as-code are durable practices, not tool-specific.
---

# Documentation Writer

## When to use
Writing or updating a README, API reference, architecture doc, or code
comments.

## Procedure
1. **Identify the single audience** for this document (new contributor, API
   consumer, ops/SRE, future-you) before writing a word. One document, one
   audience — mixing them serves nobody well.
2. **Answer what → why → how, in that order.** What it is (one sentence),
   why it exists (the problem it solves), how to use it (a minimal runnable
   example).
3. **Make every example runnable.** Copy-paste it mentally against the
   actual API before including it.
4. **Keep docs in the repo, next to the code**, not in an external wiki —
   anything not versioned with the code it describes goes stale within
   weeks.
5. **Comments explain why, not what.** If deleting a comment wouldn't
   confuse a future reader, delete it.

## Avoid
Writing a README that's a wall of prose with no structure; documenting
implementation details instead of the public contract; comments that
narrate the code line by line.

## Checklist
- [ ] Audience identified before writing
- [ ] What/why stated before how
- [ ] Every example is actually runnable, not aspirational
- [ ] Docs live in the repo, not an external doc tool
- [ ] No "what" comments — only "why" comments where genuinely non-obvious

See `reference/playbook.md` for the README section order, the API-doc
field checklist, and the C4 model for architecture docs.
