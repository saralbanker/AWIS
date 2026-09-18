---
name: system-auditor
description: >
  Use when investigating system-level failures spanning services, config, or
  infrastructure, where the root cause is unknown. Not for a single-function
  bug (debugging-master) or a pure performance question (performance-optimizer).
last_verified: 2026-08-30
kill_condition: n/a — Five Whys and evidence-first incident investigation are methodology, not tool-specific.
---

# System Auditor

## When to use
A production incident, "works locally, breaks in prod," or multiple
components failing at once with no clear single cause.

## Procedure
1. **Collect evidence before forming a hypothesis** — logs, timestamps,
   recent deploys/config changes, current state of dependencies. Forming a
   hypothesis first invites confirmation bias.
2. **Find the first error, not the latest.** Cascading failures produce
   many log lines; the earliest one is usually the actual origin, the rest
   are consequences.
3. **Trace the pipeline forward** from where the request enters the system,
   and find the point where observed state first diverges from expected
   state — that's the investigation focus, not "everything downstream."
4. **Apply Five Whys** from that divergence point, and stop at a
   *structural* cause (missing test, missing monitoring, a bad default) —
   not an accidental one ("someone typo'd a value").
5. **Assess blast radius** before fixing: what else shares the failing
   dependency? Is data integrity at risk?
6. **Deliver two things**: an immediate mitigation (restore service now)
   and a root fix (prevent recurrence) — a report with only one of these
   is incomplete.

## Avoid
Reporting the symptom as the finding ("500 error" is not a root cause);
investigating everything at once instead of the divergence point; a fix
plan with no monitoring/prevention component.

## Checklist
- [ ] Root cause is structural, reached via Five Whys, not asserted
- [ ] Timeline reconstructed from the first error, not the most recent
- [ ] Blast radius assessed — other affected systems named
- [ ] Fix plan has both immediate mitigation and a permanent fix
- [ ] Every conclusion cites evidence (a log line, metric, or observed state)

See `reference/playbook.md` for the Five Whys worked example and
evidence-collection commands.
