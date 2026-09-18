---
name: architecture-analyst
description: >
  Use when mapping an unfamiliar codebase, reviewing system design before adding
  features, or hunting structural problems like tight coupling, circular
  dependencies, or layer violations. Not for runtime bugs (debugging-master) or
  performance (performance-optimizer).
last_verified: 2026-08-30
kill_condition: n/a — coupling/cohesion/layering are structural concepts, not tied to a tool or framework version.
---

# Architecture Analyst

## When to use
Understanding an unfamiliar codebase, reviewing design before a big feature,
or investigating suspected coupling/anti-pattern problems.

## Procedure
1. **Trace from the entry point** (`main`/`index`/`app`/`server`), following
   imports outward — the import graph is the real architecture, not what the
   README claims.
2. **Map layers** as you trace: routes/UI → application/services → domain →
   infrastructure. Dependencies should point inward; domain code importing a
   framework or the DB layer is a violation worth flagging.
3. **Score coupling** per module by counting external imports: 0-3 low, 4-7
   medium, 8+ high. High-coupling modules are your hotspot list.
4. **Check for anti-patterns** by measurable signal, not vibes: a class with
   >10 public methods/fields, a file >500 lines, a circular import, or a
   feature that requires editing 6+ files to change.
5. **Report findings with file:line evidence.** "Reduce coupling" is not a
   finding; "move `formatAddress` from `OrderService` (order.ts:42) to
   `AddressFormatter`" is.

## Avoid
Asserting an anti-pattern without evidence; treating this as a substitute for
`debugging-master` (runtime behavior) or `performance-optimizer` (speed).

## Checklist
- [ ] Entry point traced, layer boundaries identified
- [ ] Coupling scored per module
- [ ] Every anti-pattern claim has file:line evidence
- [ ] Recommendations name a specific file/function, not a generality

See `reference/playbook.md` for detection commands (circular deps, god
objects, layer-violation greps) and the anti-pattern signature table.
