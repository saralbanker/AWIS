# Code Synthesizer — Playbook

## SOLID, applied

- **Single responsibility**: one module, one reason to change. A class that
  both validates users and sends email should split.
- **Open/closed**: extend via new code (strategy pattern, plugin), not by
  piling more branches into an existing `if/else` chain.
- **Liskov substitution**: a subtype must work anywhere its parent type is
  expected. If `AdminUser extends User`, every function accepting `User`
  must still behave correctly given an `AdminUser`.
- **Interface segregation**: small, specific interfaces beat one large
  general one (`Readable` + `Writable` over a do-everything interface).
- **Dependency inversion**: depend on abstractions, not concretions —
  `UserService` should depend on an `IUserRepository` interface, with
  `PostgresUserRepo` implementing it, so the DB can be swapped without
  touching the service.

## Complexity, worked example

```typescript
// Before — cyclomatic complexity ~12, one function doing five things
function processOrder(order: Order): Result {
  if (!order) throw new Error('No order');
  if (!order.items.length) return { status: 'empty' };
  if (order.total > 10000) { /* discount logic */ }
  if (order.user.isPremium) { /* premium logic */ }
  // ...more branches
}

// After — split by responsibility, each piece independently testable
function validateOrder(order: Order): void { /* 2 branches */ }
function calculateDiscount(order: Order): number { /* 2 branches */ }
function applyPremiumBenefits(order: Order): Order { /* 2 branches */ }
```

## Explicit contracts

```typescript
// Implicit — silently returns undefined on a miss, callers must guess
function getUser(id) { return db.find(id); }

// Explicit — typed, and failure is a typed error, not an undefined surprise
function getUser(id: string): Promise<User> {
  if (!id) throw new ArgumentError('id is required');
  const user = await db.user.findUnique({ where: { id } });
  if (!user) throw new NotFoundError(`User ${id} not found`);
  return user;
}
```

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Test passes but output is wrong | assertion too loose (`toBeTruthy` instead of exact match) | use `toEqual`/`toStrictEqual`, not truthy checks |
| Function is hard to follow | violates single responsibility | find the "and" in what it does — that's the split point |
| Type mismatch at a boundary | producer and consumer disagree on shape | fix the type at the boundary; don't suppress with `as any` |
| Works alone, breaks when integrated | hidden dependency (global state, env var, singleton) | make every dependency an explicit parameter or constructor arg |
| Tests break on every refactor | test asserts implementation details, not behavior | assert on inputs → outputs, not internal calls |
