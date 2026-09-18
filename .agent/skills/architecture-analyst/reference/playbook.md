# Architecture Analyst — Playbook

## Core concepts

**Coupling vs. cohesion.** Low coupling = modules are independent; changing A
doesn't break B. High cohesion = things that change together live together.
Aim for both.

**Dependency inversion.** High-level modules (business logic) should not
import low-level modules (DB, HTTP) directly — both should depend on an
abstraction.
```
Violation: UserService imports PostgresAdapter directly.
Correct:   UserService depends on an IUserRepository interface;
           PostgresUserRepo implements it.
```

**Layered architecture.** Dependencies point inward: Infrastructure →
Application → Domain. Domain should contain pure business logic with zero
framework imports; if it imports a DB client or HTTP framework, that's a
layer violation.

## Anti-pattern signatures

| Pattern | Detection | Threshold |
|---|---|---|
| God object | public methods + fields | >10 methods AND >10 fields, or file >500 lines |
| Circular dependency | A imports B, B imports A | any occurrence |
| Feature envy | method uses another class's data more than its own | >50% external references |
| Shotgun surgery | files touched per feature change | 6+ files in different modules |
| Data clump | same group of fields passed together | 3+ functions share the same 3+ params |

## Detection commands

Import graph:
```bash
grep -rn "import\|require" src/ --include="*.ts" --include="*.js" | grep -v node_modules | grep -v ".test.\|.spec."
```

Circular dependencies:
```bash
npx madge --circular src/
```

God objects (files over 500 lines):
```bash
find src/ -name "*.ts" -o -name "*.js" | xargs wc -l | sort -rn | head -20
```

Layer violations (does domain/ import infrastructure or a framework?):
```bash
grep -rn "import.*from.*infrastructure\|import.*from.*db\|import.*from.*api" src/domain/
grep -rn "import.*from.*express\|import.*from.*prisma\|import.*from.*axios" src/domain/
```

Shotgun-surgery candidates (files that keep changing together):
```bash
git log --oneline --name-only -50 | grep "src/" | sort | uniq -c | sort -rn | head -20
```

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| `madge` not installed | tool missing | `npx madge --circular src/` auto-installs it |
| Circular chain too deep to read | transitive A→B→C→D→A | `npx madge --image dep-graph.svg src/` for a visual |
| No clear entry point | monorepo, multiple apps | check each package's `package.json` `main`; analyze separately |
| Everything looks coupled | architecture grew organically, was never designed | document current state as-is; propose target architecture as a separate deliverable |
