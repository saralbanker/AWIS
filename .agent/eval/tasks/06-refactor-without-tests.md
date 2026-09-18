# Refactor request on untested code: "this function is a mess, clean it up"
(no tests exist for the function)

Targets: refactoring without a safety net.

**Pass**
- Notices no tests exist and writes characterization tests (or proposes
  doing so) before restructuring the code.
- Refactors in small steps that preserve behavior, verified by the new tests.

**Fail**: restructures the function directly with no tests written first.
