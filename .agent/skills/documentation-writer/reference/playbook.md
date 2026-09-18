# Documentation Writer — Playbook

## README, section order
A README is a funnel — most readers only need the first few sections:

```markdown
# Project Name          — what this is, one sentence
Brief description        — the problem it solves, 2-3 sentences
## Installation          — exact copy-paste commands
## Usage                 — minimal working example, 10-20 lines
## API Reference          — full public surface (power users only)
## Contributing           — dev setup, how to submit changes
## License                — SPDX identifier
```

## API doc, required fields per function/endpoint
Description, every parameter (with constraints), the return type, every
`throws`/error case, and at least one runnable example:

```typescript
/**
 * Calculate shipping cost based on weight and destination.
 * Uses tiered pricing: domestic flat rate, international by weight bracket.
 *
 * @param weight - Package weight in kg. Must be > 0 and <= 50.
 * @param destination - ISO 3166-1 alpha-2 country code (e.g. "US").
 * @returns Shipping cost as a Money object.
 * @throws {ArgumentError} If weight is <= 0 or > 50.
 * @example
 * const cost = calculateShippingCost(2.5, 'DE');
 * // => { amount: 15.99, currency: 'USD' }
 */
```

## Comment philosophy, with examples

```typescript
// Bad — restates the code
i += 1; // increment i by 1

// Bad — obvious from the function name
async function getUser(id: string) { // gets a user by id

// Good — explains a non-obvious constraint
const MAX_RETRIES = 3; // circuit breaker trips per PCI-DSS requirement

// Good — explains why this approach, not another
// setTimeout instead of setInterval: setInterval doesn't account for
// execution time, which causes drift under load.
```

## C4 model, for architecture docs
Zoom levels, most projects only need the first two: **Context** (the system
plus external actors/systems), **Container** (major deployment units — web
app, API, DB, cache), **Component** (modules within a container), **Code**
(classes/functions — usually unnecessary detail).

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Docs stale within weeks | written separately from code, no update trigger | move into the repo; require doc updates in the same PR as the code change |
| API docs have no examples | written from the code's perspective, not the caller's | write the example first, then fill in the parameter docs around it |
| Examples silently broken | code changed, doc wasn't updated with it | treat doc examples as tests where feasible; otherwise spot-check before merging |
