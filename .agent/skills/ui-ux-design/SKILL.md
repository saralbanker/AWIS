---
name: ui-ux-design
description: >
  Use when choosing a visual style, color palette, layout pattern, or
  stack-specific UI convention for a real interface — not a rule about what
  to avoid, a lookup tool over curated design reference data.
last_verified: 2026-08-30
kill_condition: revisit if the BM25 search script or CSV schema in scripts/ and data/ changes shape.
---

# UI/UX Design

A search tool over curated design-reference data (style archetypes, color
systems, layout patterns, per-stack conventions), not a set of static
rules. Query it instead of guessing — the data doesn't rot the way a
"never use purple" rule would, because it's descriptive reference material,
not asserted taste.

## When to use
Picking a visual style, color palette, chart type, landing-page layout, or
a stack-specific UI convention (React, Vue, SwiftUI, Flutter, shadcn,
etc.) for a real screen or component.

## Anti-trigger
This is a static/layout style lookup — it has no data on motion, animation
timing, 3D transforms, or generative visual effects. For a request that's
primarily about animation or motion design, querying this for a color
palette is fine as one input, but don't treat its results as covering the
actual creative/technical challenge — that part is unaided judgment, not
something this skill has reference data for.

## Procedure
1. **Search the relevant domain** rather than guessing from memory:
   ```bash
   python .agent/skills/ui-ux-design/scripts/search.py "minimalist dashboard" --domain style
   python .agent/skills/ui-ux-design/scripts/search.py "primary button colors" --domain color
   python .agent/skills/ui-ux-design/scripts/search.py "form validation" --stack react
   ```
   Domains: `style, prompt, color, chart, landing, product, ux, typography,
   icons, react, web`. Stacks: `html-tailwind, react, nextjs, vue, nuxtjs,
   nuxt-ui, svelte, swiftui, react-native, flutter, shadcn, jetpack-compose`.
2. **Derive the choice from the project's existing palette/patterns first**;
   fall back to the search results only when the project has no established
   convention yet.
3. **For a whole new project**, generate a coherent design system instead of
   picking pieces ad hoc:
   ```bash
   python .agent/skills/ui-ux-design/scripts/search.py "SaaS dashboard" --design-system -p "ProjectName"
   ```
   Add `--persist` to write it to `design-system/MASTER.md`, and `--page
   "dashboard"` to also create a page-specific override — later work should
   check the page override first, then fall back to `MASTER.md`.
4. **Justify color/layout choices by contrast ratio and semantic fit**, not
   by trend — a choice that works because it satisfies WCAG contrast and
   matches the product's tone survives a taste cycle; a choice that's "in
   style this year" doesn't.

## Avoid
Applying a style pattern from a search result verbatim without checking it
fits the project's existing conventions; treating a "best for" note in the
data as a hard rule for every context; skipping the search and asserting a
color/pattern preference from general impression.

## Checklist
- [ ] Queried the relevant domain/stack rather than guessing
- [ ] Choice matches or intentionally extends the project's existing palette/patterns
- [ ] Color/contrast choices meet WCAG AA at minimum
- [ ] For a new project: a persisted design system exists so later pages stay consistent

Data lives in `data/*.csv` (general) and `data/stacks/*.csv` (per-framework);
the search engine is in `scripts/core.py` (BM25 + domain config),
`scripts/design_system.py` (design-system generation/persistence), and
`scripts/search.py` (CLI entry point) — read those only if the CLI's
behavior needs to be understood or extended, not before every use.
