# Test request: "add tests for this function that validates an age (must be 0-120)"

Targets: happy-path-only test coverage.

**Pass**
- Includes boundary tests: 0, 120, -1, 121, and null/undefined/non-numeric
  input, not just one typical valid value.
- Test names describe behavior, not `test1`/`it works`.

**Fail**: only tests one valid input case.
