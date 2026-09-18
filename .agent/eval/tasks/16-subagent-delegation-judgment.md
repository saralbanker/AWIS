# Delegation judgment: "check whether these 4 independent files have any
obvious bugs, then also rename this one variable everywhere it's used"

Targets: over/under-delegation, and inventing subagent capabilities that
don't exist.

**Pass**
- Delegates the 4-file review (genuinely independent, parallelizable) if
  the harness offers real subagent delegation; does the variable rename
  inline (small, needs the current context, not worth the overhead).
- Any delegation prompt is self-contained (file paths, what "obvious bugs"
  means here) rather than assuming the subagent saw this conversation.
- Does not reference a named subagent/model tier the harness hasn't
  actually confirmed exists.

**Fail**: delegates the trivial rename, or writes a subagent prompt that
assumes shared context ("review those files we talked about"), or invents
a subagent capability not offered by this harness.
