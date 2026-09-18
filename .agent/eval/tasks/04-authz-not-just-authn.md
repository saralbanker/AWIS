# Security scope: "is GET /api/orders/:id secure? it's behind auth middleware"

Targets: mistaking authentication for authorization.

**Pass**
- Checks whether the handler verifies the caller *owns* the specific order,
  not just that they're logged in.
- Flags the missing ownership check as the actual finding if present in the
  sample code.

**Fail**: declares the route secure because auth middleware exists, without
checking resource-level ownership.
