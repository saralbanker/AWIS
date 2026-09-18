---
name: subagent-protocol
description: >
  Use when deciding whether part of a task should be delegated to a
  subagent versus done inline, or when writing a new subagent/agent
  definition. Not a replacement for the harness's own delegation
  mechanism — teaches how to use it well.
last_verified: 2026-09-03
kill_condition: revisit if a harness's actual delegation model (context sharing, parallel dispatch) changes shape enough that the handoff-contract advice below no longer matches how it works.
---

# Subagent Protocol

How to decide when delegation is worth it, and how to write a subagent
definition or handoff prompt that actually works — independent of which
harness's delegation mechanism is in play. This kit does not hand-roll its
own orchestration layer or invent subagent names/capabilities a harness
doesn't provide; an earlier version of this kit did exactly that
(`orchestrator.md`, deleted) and it hallucinated harness internals that
didn't exist. Use the harness's real mechanism; this skill is about using
it well.

## When to use
Deciding whether to spawn a subagent for part of a task, or authoring a new
subagent definition.

## Procedure
1. **Delegate only when it earns its overhead.** Good reasons: the work is
   genuinely independent (its raw output isn't something you need to keep
   in your own context), it can run in parallel with other work, or it
   needs a different tool-scope or model tier than the current context. A
   one-shot lookup or anything that needs your running context isn't worth
   spawning for — do it inline.
2. **Scope each subagent to one clear role, least-privilege tools.** A
   subagent that can do everything isn't meaningfully different from doing
   it yourself, and it carries more risk (broader blast radius, harder to
   review). Read-only roles (review, verification) get read-only tools.
3. **Write the handoff as a self-contained brief, not a pointer.** Check
   whether this harness's delegation shares context with the parent or
   starts fresh (most do the latter) — a fresh subagent has zero memory of
   this conversation. State what it needs: file paths, decisions already
   made, what's explicitly out of scope. "Based on what we discussed"
   means nothing to a subagent that never saw the discussion.
4. **State the return contract up front**: what should come back (a
   decision plus evidence, not a raw transcript) and how it'll be used.
   A vague ask produces an oversized report that costs more to read than
   doing the work yourself would have.
5. **Batch independent subagents in parallel; sequence dependent ones.**
   Don't parallelize speculatively — coordination and merge cost is real,
   and most tasks don't actually decompose into independent pieces.
6. **Verify a subagent's summary against its actual output before trusting
   it.** A report describes what the subagent intended to do, not a
   guarantee of what happened — check the real diff/file/result for
   anything that matters.

## Avoid
Inventing a subagent name or capability the harness doesn't actually
provide (the single worst failure mode in this kit's history — see
`reference/playbook.md` for what went wrong); delegating trivial work that
costs more in coordination than it saves; granting a subagent broader tool
access than its one job needs; writing a handoff prompt that assumes
shared context it doesn't have.

## Checklist
- [ ] Delegation earns its overhead — not busywork split into pieces
- [ ] Subagent has one clear role and least-privilege tools
- [ ] Handoff prompt is self-contained, no unstated shared context
- [ ] Return contract stated (what comes back, not just "look into X")
- [ ] Subagent's report checked against its real output before being trusted

See `reference/playbook.md` for the context-sharing model across harnesses
and a worked handoff-prompt example, and `reference/templates/` for two
ready-to-use subagent definitions (least-privilege, single-role, matching
every principle above).
