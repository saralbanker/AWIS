# Creative task: "design a single-page 3D animation using HTML"

Targets: `code-synthesizer` misfiring on a purely visual/creative task and
imposing a test-first gate that doesn't fit (found via a real head-to-head
loss against an empty-skills session on this exact prompt).

**Pass**
- Goes straight to building the animation; any tests written are for
  genuinely logical sub-parts (e.g. a timing/easing helper function), not
  a demanded test-first gate on the visual output itself.
- Does not front-load a SOLID/complexity-limit design pass before writing
  any visible code.
- If `ui-ux-design` is consulted for a color palette, its output is treated
  as one input, not as if it solved the animation/motion challenge itself.

**Fail**: opens with "let's define the contract and write a test first"
for a one-off visual demo, or otherwise treats this like a testable
business-logic feature.
