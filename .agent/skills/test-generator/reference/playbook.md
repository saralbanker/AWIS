# Test Generator — Playbook

## Arrange-Act-Assert

```typescript
it('should_return_discount_when_order_exceeds_100', () => {
  // Arrange
  const order = createOrder({ total: 150 });
  const calculator = new DiscountCalculator();

  // Act
  const discount = calculator.apply(order);

  // Assert
  expect(discount).toEqual({ percent: 10, amount: 15 });
});
```

## Boundary value analysis, worked case
For a function accepting `1-100`: test `0` (just below min, should reject),
`1` (min, should accept), `50` (typical middle), `100` (max, should
accept), `101` (just above max, should reject), and `null`/`undefined`/
`NaN` (should reject as invalid type).

## Test pyramid ratio
Roughly 70% unit / 20% integration / 10% E2E. Unit tests mock all external
dependencies; integration tests use real components to validate a
contract (e.g. a real test database, not a mock); E2E tests stay under
~20 for most apps and cover only flows where a break would be a real
incident (login, checkout, core data creation).

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Tests pass, bugs still ship | tests don't cover real usage paths | add a test from every bug report, not just new code |
| Unit suite is slow (>30s) | integration/E2E creep, or unit tests touch real I/O | push logic into pure functions, mock at the I/O boundary |
| Mock drifts from real implementation | nothing validates they still match | add one contract test calling the real implementation with the same inputs |
| Flaky test | shared state between tests, or async timing | add proper `beforeEach` cleanup, `await` correctly, never rely on test order |
