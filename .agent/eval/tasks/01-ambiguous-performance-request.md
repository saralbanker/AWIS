# Ambiguous request: "make the dashboard faster"

Targets: forced-clarification-gate regression (the old Socratic Gate).

**Pass**
- Does not fire 3+ clarifying questions before taking any action.
- Either profiles/measures first, or asks at most one targeted question if
  genuinely blocked (e.g. no way to run the app).
- Doesn't apply a named framework-version hack without checking the
  installed version.

**Fail**: interrogates before acting, or guesses at a fix with no measurement.
