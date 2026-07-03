# PRD RESEARCH REPORT
## What Version 1 Should Become — Findings of the Product Architecture Tribunal

**Date:** 2026-07-02
**Governing authority:** `OIP_CONSTITUTION.md` (Tier 0, assumed correct until implementation evidence proves otherwise)
**Question under investigation:** *What is the smallest software product capable of validating the Constitution?*
**Method:** Gall's Law applied adversarially. Every capability below survived an attempt to delete it.

---

## 1. EXECUTIVE SUMMARY

The Constitution, decomposed and stress-tested, implies one product — not many. Its center of gravity is the Record (Articles 5–13); its riskiest untested claim is that organizational memory can be captured **as a byproduct of work** (Article 14) while still containing **the why** (Article 11). Those two articles are in genuine tension: rationale is not exhaust — writing "why" is labor, and four decades of knowledge management plus the documented decay curve of Architecture Decision Records prove that labor doesn't get paid under schedule pressure.

**Version 1 is therefore a single instrument: a decision record for small software teams that nearly writes itself.** At the moment a decision becomes real (a merged change, a concluded thread), the system assembles the surrounding context, drafts the decision entry — what, why, alternatives rejected, who — and a human confirms it in under a minute. The Record lives as open plain-text files in the team's own repository, indexed locally. Its recall surface answers one question better than anything on earth: *"why is this the way it is?"* — with citations.

Everything else the Constitution describes — charters, trust ladders, execution, automation, agents, process observation at scale — is **deliberately absent from V1**, because the Constitution itself orders it so: understanding before automation (Article 34), autonomy earned per-domain from the bottom rung (Article 22), complex systems grown from simple systems that work (FP-9).

**The six-week verdict** (expanded in §20): build the capture→record→recall loop for one team (your own), on files + an embedded index, with one capture surface that meets developers where decisions actually conclude. Do not build sync services, web apps, charter engines, integrations, dashboards, or anything that acts. The single hypothesis V1 exists to test: **near-zero-friction capture of decision rationale sustains itself without mandate.** If that fails, the Constitution has a broken load-bearing article and no amount of platform matters.

---

## 2. CONSTITUTION VALIDATION

### Stage A/B findings — what the Constitution actually commits us to

**Explicit goals extracted:** durable organizational memory (Title II); byproduct capture (Art. 14); provenance and attribution (Art. 9, 19); honesty registers (Art. 10); rationale as first-class (Art. 11); zero-AI viability (Art. 32); exit as files-level reality (Art. 6, 12); minimal core / open edge (Art. 41–46); understanding before automation (Art. 34–35).

**Implicit goals discovered under decomposition:**
- The Record must accrue value *before* it is large (otherwise nobody feeds it) — a cold-start requirement the Constitution never states.
- "Capture at the point of work" implies the product must live **inside existing tools' exhaust streams**, not beside them — an integration posture, stated nowhere explicitly.
- The four-constituency trust model (§9.4) implies **consent architecture** from day one, even in a 5-person team.

**Separation of concerns:**

| Category | Contents |
|---|---|
| **Platform** (5–10 yr) | Record semantics, provenance model, trust ladder, charters, projections |
| **Product** (V1) | Decision capture + recall for software teams |
| **Infrastructure** (never own) | Storage engines, models, sync transports, execution engines |
| **Tooling** | Capture surfaces (hooks, CLI, chat commands) — disposable, replaceable |
| **Research** | Registers UX, cold-start economics, unit-of-work question (Q1) |
| **Unknowns** | Carried in §19 |

### Stage B — stress-test results per constitutional commitment

| Constitutional commitment | Stress-test result |
|---|---|
| Record-first, engines-second (Art. 5) | **Survives.** Independently corroborated by V1-scale evidence: Obsidian's files-over-app and SQLite's Library-of-Congress-grade longevity show record-centric products win trust precisely at small scale. |
| Byproduct capture (Art. 14) | **Survives as principle, unproven as mechanism.** ADR evidence confirms the failure mode it guards against ("the discipline of writing good ADRs… collapses under schedule pressure unless there is a system that reduces the friction of capture"). V1's entire purpose is to test the mechanism. |
| Record the why (Art. 11) | **Survives, but only via T1 resolution** (§4). Unaided, it contradicts Art. 14. |
| Honesty registers (Art. 10) | **Weakened.** No evidence any team distinguishes observation/description/intention in daily tooling; risk of epistemic bureaucracy. Demoted to V1 experiment E4, minimal form. |
| Trust ladder & charters (Art. 20–26) | **Untestable in V1** — no machine actor *acts* in V1. Not weakened; simply dormant. V1 exercises only the bottom rungs (observe, suggest) and full attribution (Art. 19, 29). |
| Declarative reconciliation (Art. 37) | **Unfalsifiable at V1 scale.** No execution exists. Flagged: first constitutional article with zero near-term evidence path. |
| Zero-AI test (Art. 32) | **Survives with a documented strain** (T4, §4). |
| Exit rights as strategy (Art. 49) | **Commercially unproven (constitution's own Q3)** but *architecturally free* in V1: plain files in the customer's repo make exit a `git clone`. V1 gets the principle at zero cost. |
| Minimal core (Art. 41–42) | **Survives and bites immediately:** it is the justification for most of §8's "Must NOT exist" list. |

**No contradiction requiring constitutional amendment was found.** Two articles (10, 37) are on notice pending evidence. The tension log follows in §4.

---

## 3. HIDDEN CONSTITUTIONAL ASSUMPTIONS

Assumptions the Constitution makes without stating, now surfaced:

1. **Demand exists.** The Constitution proves organizations *lose* memory; it nowhere proves they will *adopt a product* to keep it. KM history is evidence of failed supply, not of demand. Mitigating signal: ADRs are the one memory practice engineers adopted **voluntarily and bottom-up** — organic demand exists in exactly one niche. That niche is therefore the beachhead. *(Assumption converted to experiment E1/E2.)*
2. **The why is capturable at all.** Art. 11 assumes decision rationale can be externalized. Polanyi (FP-4) warns some of it can't. V1's bet is narrower than the Constitution's: the *stated, confirmable* portion of rationale is enough to be valuable. *(Hypothesis.)*
3. **The Record can be additive.** The Constitution implies the Record coexists with the tool sprawl it compensates for. If adoption requires migration or replacement of any existing tool, V1 dies on arrival. V1 must be purely additive. *(Design constraint.)*
4. **Value precedes mass.** The Record must be worth consulting at 30 entries, not 30,000. *(Cold-start — experiment E6.)*
5. **A buyer exists for a bottom-up artifact.** Deferred: V1 optimizes learning, not revenue (Tier 3 mandate), but the assumption is logged in §19.
6. **"Organization" is homogeneous.** It isn't. V1 must pick software teams without letting that niche's shape leak into Record semantics (guarded by tension T3 below).

---

## 4. CONTRADICTIONS & TENSIONS

**T1 — The load-bearing tension: Art. 11 (record the why) vs Art. 14 (capture must not tax).**
Rationale is generative work, not exhaust. Unresolved, these two articles reproduce the exact ADR failure curve ("five ADRs in the first month, then silence"). **Resolution hypothesis, and the core of the product:** the machine drafts the why from observed context (the diff, the thread, the linked artifacts); the human's cost collapses from *authoring* to *confirming*. If confirmation-cost capture sustains itself, both articles stand. If not, Art. 14 stands and Art. 11 must be weakened by amendment. This is V1's reason to exist. *(Experiment E1.)*

**T2 — Art. 10 (three honesty registers) vs simplicity.** Full register discipline in a capture UI is bureaucracy nobody asked for. V1 implements the *property* with two lightweight distinctions: every entry separates **linked evidence** (observation) from **stated rationale** (description), and machine-drafted text is visibly marked (Art. 29). Intention appears only as explicit "we plan to" language. Whether even this much survives contact with users is experiment E4.

**T3 — Rejected-idea R10 (never design around one context) vs Tier 3 (pick the earliest customer).** Resolved by a boundary rule: **niche may shape capture surfaces and vocabulary of examples; niche may never shape Record semantics.** Entries are (decision, rationale, actors, evidence links, time, register marks) — nothing software-specific. Capture surfaces (git hooks, chat commands) are declared disposable tooling.

**T4 — Art. 32 (zero-AI viability) vs T1's resolution (AI-drafted capture).** If capture only works with AI, is the platform AI-dependent? Ruling: Art. 32 requires the platform to *function* — record, recall, export — without AI, and it does (capture degrades to manual entry, i.e., to classic ADR practice). But honesty requires stating: **without AI, V1's advantage over a folder of markdown ADRs approaches zero.** The differentiating mechanism is AI-shaped even though the Record is not. Logged, not hidden.

**T5 — Art. 16 (never surveil persons) vs observing work exhaust.** Even commit-and-thread observation is observation of people's output. V1 posture: capture is **invoked or confirmed by the person whose work it is** — never silent, never managerial. This also keeps V1 clear of the social-license unknown (constitutional Q2) until there's evidence.

No irreconcilable contradiction found. T1 is the hill the product either takes or dies on — by design.

---

## 5. PRODUCT IDENTITY

**V1 is a team memory instrument: the decision record that writes itself and answers "why."**

What it is: a small, opinionated, local-first tool that turns the moment a decision concludes into a durable, attributed, citable Record entry — and turns "why is X like this?" from an archaeology expedition into a sixty-second answer with sources.

What it is not (each rejection constitutionally grounded):
- **Not a wiki or notes app** — capture is event-shaped and confirmed, not freeform authoring (Art. 14; the KM graveyard).
- **Not a tracker** — Jira/Linear own tasks; V1 owns *why*, the layer they all discard. Additive, not competitive (Assumption 3).
- **Not a copilot** — intelligence drafts and recalls; it never becomes the interface to the Record (Art. 28) and never acts (Art. 34).
- **Not a platform yet** — V1 is the organism, not the ecosystem (Gall's Law; CURRENT_TRUTH_STATE).

Identity precedents from Tier 1 evidence: **Linear** won a giant's market with an opinionated, narrow, fast tool ("won on a philosophy, not a feature list"); **Obsidian** won trust with files-over-app; **Git** proved a well-designed record of *changes with rationale attached* (commit messages) becomes indispensable infrastructure. V1 sits at the intersection: *Linear's opinionation, Obsidian's ownership model, Git's subject matter — applied to decisions.*

**Working name for this report:** the Record. (Naming is deferred; identity is not.)

---

## 6. CORE USER

**The founding engineer / tech lead of a 5–50 person software organization** — chosen by elimination, not preference:

| Criterion (Tier 3) | Why software teams win |
|---|---|
| Proven organic demand for decision memory | Only niche that adopted a memory practice voluntarily (ADRs) |
| Decisions conclude in observable, linkable artifacts | Merges, PRs, threads — context assembly is feasible |
| Tolerates CLI/keyboard-first V1 surfaces | Linear evidence: developers reward speed over polish |
| Shortest path to daily usage | Decisions conclude multiple times per week in any active team |
| Founder can dogfood from day one | Tier 2: the first user is the builder — feedback latency ≈ zero |
| Pain is acute and nameable | "Why is this like this?" burns real hours in onboarding, re-litigation, and post-departure archaeology (40%-in-6-months evidence) |

The **person who feels the pain** is whoever must answer for old decisions: the tech lead onboarding engineer #6, the founder after engineer #3 leaves. The **person who pays the capture cost** is whoever just made a decision. V1 succeeds only if the second person's cost is near zero (T1) and the first person's payoff is visible weekly (Art. 15: the observed must be served).

---

## 7. CORE WORKFLOW

One loop. Everything else is decoration.

```
DECIDE            A decision concludes in the flow of work
                  (PR merged, thread resolved, "let's do B" said aloud)
   ↓
CAPTURE           One invocation at that moment (hook prompt, CLI, chat command).
                  The system assembles surrounding context and DRAFTS the entry:
                  what was decided · why · what was rejected · who · evidence links
   ↓
CONFIRM           The decider edits/approves in ≤60 seconds.
                  Machine-drafted text stays marked as machine-drafted (Art. 29).
                  Entry appends to the Record — plain files in the team's own repo.
   ↓  (days–years pass)
RECALL            Anyone asks: "why do we ___?"  /  "what did we decide about ___?"
                  Answer in seconds, WITH CITATIONS to entries and linked artifacts,
                  with uncertainty stated when the Record is silent (Art. 13, 30).
   ↓
COMPOUND          Each recall that prevents a re-litigated decision or an
                  archaeology session is the value event. Count them.
```

**Primary outcome:** the team stops re-deciding decided things and stops losing the why at departure.
**Primary success metric:** **weekly confirmed captures per team (habit)** and **weekly recall events (value)** — sustained past week 4 without any mandate. Numbers in §13.

---

## 8. VERSION ONE BOUNDARY

### Must Exist (each with constitutional justification)

| Capability | Justification |
|---|---|
| Append-only Record of decision entries, as documented plain-text files in the team's repo | Art. 5–7, 12; exit (Art. 6) becomes trivially true; Obsidian/SQLite longevity evidence |
| Provenance on every entry: author, confirmer, machine-drafted marks, evidence links, time | Art. 9, 19, 29 |
| One near-zero-friction capture surface with AI drafting + human confirmation | Art. 14 + Art. 11 via T1 — the core hypothesis |
| Manual capture path (no AI) | Art. 32; degradation floor = ADR practice |
| Recall: full-text + semantic query with **citations** and stated silence | Art. 13, 30, 33; the value event |
| Local-first operation; the index rebuildable from files at any time | Art. 6, 47; Tier 2 solo-maintainability |
| Consent-shaped capture (invoked/confirmed by the person whose work it is) | Art. 15–16; T5 |

### Should Exist (build only after the loop demonstrably runs)

- Second capture surface (whichever the dogfood team reaches for and misses — chat command or editor).
- A minimal read view (TUI or single local page) for browsing/onboarding — *reading* the Record must not require the CLI.
- "Related prior decisions" shown at capture time (cheap first taste of the suggest rung, Art. 22 — still zero autonomy).

### Could Exist (permitted, unscheduled)

- Git-remote-based team sync (adopting Git as transport, not building sync).
- Import of existing ADR folders (adoption bridge for the beachhead niche).

### Must NOT Exist (each with reasoning)

| Excluded | Reasoning |
|---|---|
| Workflow/execution engine, automation of any kind | Art. 34 (understanding first); R9; nothing in V1 has earned action |
| Agents, charters engine, trust-ladder machinery | No machine actor acts in V1; building governance for actors that don't exist is a V5 problem in V1 (FORBIDDEN_ASSUMPTIONS) |
| Hosted multi-tenant service, accounts, SSO, RBAC | Enterprise optimization ban; local-first + repo permissions suffice for 5–50 |
| Integrations marketplace / breadth of connectors | Art. 41; one surface proves the loop or nothing does |
| Dashboards, analytics, process mining | Observation-at-scale is post-trust (Q2); also surveillance-adjacent (Art. 16) |
| Visual builder, mobile, real-time collaboration | Interface seasons (Art. 48); zero constitutional pull at V1 |
| Custom database / sync protocol / model | Never rebuild mature infrastructure (Tier 1 mandate; FP-8) |
| Org-wide rollout features | Gall's Law; BPR evidence — one team is the organism |

---

## 9. CAPABILITY MAPPING

| Constitutional principle | Visible V1 feature | Invisible V1 infrastructure | Deferred capability |
|---|---|---|---|
| Record durability (Art. 5–8) | "Your decisions are files in your repo" | Append-only entry format, correction-by-append | Governed erasure workflow (Art. 8) |
| Provenance (Art. 9, 19, 29) | Byline + machine-draft marks on every entry | Attribution model, evidence linking | Cryptographic tamper-evidence |
| Byproduct capture (Art. 14) | Draft-and-confirm capture | Context assembler around decision moments | Passive decision-moment detection |
| Rationale first-class (Art. 11) | The why is the entry's body, never optional | Draft prompt structure | Rationale-quality feedback |
| Recall with honesty (Art. 13, 30, 33) | Cited answers; explicit "the Record is silent on this" | Local FTS + embedding index (rebuildable) | Cross-team recall; proactive surfacing |
| Zero-AI viability (Art. 32) | Manual entry + search always work | AI strictly at the edge, swappable | — |
| Registers (Art. 10) | Evidence links vs stated rationale, visibly distinct | Register fields in format | Full register discipline if E4 supports it |
| Trust ladder (Art. 20–26) | *(dormant — nothing acts)* | Attribution groundwork only | Charters, when anything acts (V3) |
| Exit (Art. 6, 49) | `git clone` **is** exit; documented format | Format spec kept human-readable | Import/export ecosystem |
| Minimal core (Art. 41–46) | Small tool, few commands | Format/core vs surface/edge split | Extension interface, after two genuine external use cases |

Rejected for weak constitutional support: gamification of capture (violates Art. 15's spirit), team "memory health scores" (Art. 16 adjacency), auto-publishing digests (acts without standing).

---

## 10. PLATFORM MODULE ARCHITECTURE

Module boundaries are chosen as **replaceability seams** (Art. 47, 51). Six modules; each answerable by one engineer; each replaceable without touching the Record.

```
┌────────────────────────────────────────────────────────────┐
│  CAPTURE SURFACES (disposable tooling: hook · CLI · chat)  │
└───────────────┬────────────────────────────────────────────┘
                ▼
┌───────────────────────────┐    ┌───────────────────────────┐
│  CONTEXT ASSEMBLER        │───▶│  DRAFTING INTELLIGENCE     │
│  gathers artifacts around │    │  (rented, swappable,       │
│  the decision moment      │    │   optional — Art. 27/32)   │
└───────────────┬───────────┘    └───────────┬───────────────┘
                ▼                            ▼
┌────────────────────────────────────────────────────────────┐
│  THE RECORD  — plain-text entries, documented format,      │
│  append-only, provenance-bearing. THE ONLY MODULE WHOSE    │
│  DESIGN IS ALLOWED TO BE EXPENSIVE. (Art. 5–13)            │
└───────────────┬────────────────────────────────────────────┘
                ▼
┌───────────────────────────┐    ┌───────────────────────────┐
│  INDEX (derived, dispos-  │───▶│  RECALL — query, cited     │
│  able, rebuildable from   │    │  answers, stated silence   │
│  files at any moment)     │    │  (Art. 13, 30, 33)         │
└───────────────────────────┘    └───────────────────────────┘
```

Dependency rule: **everything depends on the Record's format; the Record's format depends on nothing.** The index is cache, never truth. Intelligence touches the Record only through the same read/append interface every other consumer uses (Art. 28, 43 — no private doors, enforced from commit one).

---

## 11. BUILD VS BUY MATRIX

| Subsystem | Verdict | Reasoning |
|---|---|---|
| Record entry format & semantics | **BUILD** | The only constitutional IP; requires ownership (Art. 5). Spend the design budget here. |
| Capture UX (draft-confirm loop) | **BUILD** | The T1 experiment itself; no prior art does confirmation-cost capture. |
| Context assembly | **BUILD (thin)** | Product-specific glue over existing artifact APIs. |
| File storage | **ADOPT: plain text on disk** | Obsidian precedent; Art. 12 decade-reader test satisfied by construction. |
| Index/search | **ADOPT: embedded database (SQLite-class) + its FTS** | Library-of-Congress longevity endorsement; zero-ops for solo maintainer; disposable anyway. |
| Versioning/sync/backup | **ADOPT: Git** | The team already runs it; sync, history, permissions, and exit inherited free. Build no transport. |
| Drafting/recall intelligence | **INTEGRATE: rented model APIs behind one internal seam** | Art. 27, 47; provider-swap must be a config change. Never build or fine-tune models in V1. |
| Chat/editor/VCS integration points | **INTEGRATE (one, then a second)** | Disposable surfaces by declaration. |
| Execution engine | **NONE in V1; ADOPT in V3 if execution is earned** | R9; Temporal-class engines are mature — rebuilding one would be constitutional vandalism. |
| Identity | **REUSE: VCS/chat identity** | 5–50-person teams already have attributable identities; building auth is enterprise optimization. |
| UI framework, web app | **DEFER** | CLI + files first (Linear evidence: the niche rewards speed, forgives absence of chrome). |

---

## 12. TECHNICAL RISK MATRIX

Ranked by expected long-term damage (Tier 4 lens: irreversibility × likelihood).

| # | Risk | Class | Damage | Mitigation |
|---|---|---|---|---|
| 1 | **Record format designed wrong** (too software-specific, too rigid, or too clever) — the one nearly-irreversible artifact | Architectural | Severe | Format versioned from day 1; niche-neutrality rule (T3); decade-reader review (E5); keep it embarrassingly simple |
| 2 | **Draft quality below confirmation threshold** — humans rewrite instead of confirm → capture tax returns → ADR death spiral | Technical/Adoption | Severe (kills T1) | Rich context assembly beats clever prompting; measure edit-distance-to-confirm; manual path as floor |
| 3 | **Capture moment mis-located** (hook fires when decisions aren't concluding) | Product | High | Dogfood tuning weeks 1–2; make invocation manual before making it ambient |
| 4 | **Cold start: recall has nothing to say for weeks** | Adoption | High | ADR-folder import; "related decisions at capture time" creates value from entry #10, not #1000 |
| 5 | **AI dependency creep** into the core (violating Art. 32 silently) | Architectural | Medium-high | Zero-AI test in CI of the design: quarterly run with intelligence disabled (E3) |
| 6 | **Registers UX confuses users** (T2) | Product | Medium | Ship minimal two-field form; E4 decides expansion or retreat |
| 7 | Index/library obsolescence | Technical | Low | Index is disposable by construction |
| 8 | Solo-maintainer burnout via surface sprawl | Maintenance | Medium | Hard cap: two capture surfaces until V1.5 |

---

## 13. COMMERCIAL VALIDATION STRATEGY

Optimize for learning, not revenue (Tier 3 mandate). Three concentric rings:

**Ring 0 — Dogfood (weeks 1–6).** The builder's own project keeps its Record in itself from the first week (the tool records its own design decisions — self-hosting the way Git hosted its own source). Failure here is disqualifying: a memory tool its own builder won't feed is dead.

**Ring 1 — Design partners (weeks 6–16).** Five to ten software teams (5–50 people), free, high-touch, selected for *already attempting* ADRs or decision logs (evidence of felt pain, per the beachhead logic in §6).

**Ring 2 — Quiet public availability** only after Ring 1 metrics clear.

**Validation metrics (value events, not vanity):**

| Metric | Threshold | Meaning |
|---|---|---|
| Confirmed captures / team / week, week 4+ without prompting | ≥ 5 | The habit exists — T1 resolved in favor of Art. 11+14 |
| Median confirm time | ≤ 60s | Capture is confirmation-cost, not authoring-cost |
| Recall events / team / week | ≥ 3 | The Record is consulted, not just fed |
| "Saved re-litigation / archaeology" incidents (qualitative log) | ≥ 1/week/team | The pain is the one we predicted |
| Teams still active at week 12 | ≥ 60% | Past the novelty cliff that killed ADR corpora |

**The disqualifying pattern (named in advance):** the *silent record* — capture continues (politeness, novelty) but recall never happens. That means we built a diary, not memory. If recall won't come to the users, V1.5's proactive surfacing is the response — and if that fails too, the demand assumption (§3.1) is falsified and the Constitution's problem statement needs re-examination against reality.

---

## 14. IMPLEMENTATION EXPERIMENTS

Documentation replaced with experiments, per mandate. Each unlocks a decision.

**E1 — Confirmation-cost capture (THE experiment).**
Hypothesis: AI-drafted, human-confirmed capture sustains ≥5 entries/wk/team past week 4 with ≤60s median confirmation. Evidence: capture telemetry + edit distance. Failure condition: reversion to authoring (heavy edits) or silence by week 4. Unlocks: T1 resolution → Art. 11/14 both stand → V1 identity confirmed. On failure: amend Art. 11 (weaken "first-class" to "best-effort") and reconsider the product's center.

**E2 — Recall value.**
Hypothesis: teams ask the Record ≥3 why-questions/week and act on answers. Evidence: query logs + weekly qualitative check. Failure: silent-record pattern. Unlocks: value-event confirmation; pricing conversations become permissible.

**E3 — Zero-AI degradation.**
Hypothesis: with intelligence disabled, record/recall/export remain fully functional and manual capture ≈ ADR-practice effort. Evidence: scripted quarterly run. Failure: any core function requires a model. Unlocks: Art. 32 compliance certificate; also measures honestly how much of the product's advantage is AI-shaped (T4).

**E4 — Registers in the small.**
Hypothesis: users understand evidence-links-vs-stated-rationale without instruction and don't rebel. Evidence: confirm-time deltas, misuse rate, interviews. Failure: field ignored or resented. Unlocks: expand toward full Art. 10 discipline, or formally minimize it.

**E5 — Decade-reader round-trip.**
Hypothesis: a competent engineer with no access to the tool reconstructs a team's decision history from the raw files alone in <1 hour. Evidence: staged trial with an outsider. Failure: format requires the app to be intelligible. Unlocks: Art. 12 compliance; format freeze candidate.

**E6 — Cold-start floor.**
Hypothesis: recall becomes subjectively useful before 50 entries when seeded with an ADR import. Evidence: partner interviews at entry #25 and #50. Unlocks: onboarding design; whether import is Must-level for Ring 1.

**E7 — Git as team transport.**
Hypothesis: repo-based sharing covers 5–50-person team needs without any built sync. Evidence: partner friction reports. Failure: merge/discovery pain dominates feedback. Unlocks: whether sync stays adopted (Git) or becomes V2's first infrastructure decision.

---

## 15. DEFERRED DECISIONS

Deliberately unmade, with the trigger that will make them:

| Decision | Deferred until |
|---|---|
| Unit-of-work expansion beyond decisions (commitments? cases?) — constitutional Q1 | Two genuine recurring use cases appear in partner Records (never before — abstraction rule) |
| Charters/trust-ladder machinery | The first capability that *acts* (V3) |
| Sync beyond Git; hosted option | E7 failure, or first partner who can't run repos |
| Pricing & packaging | E2 passes; not before |
| Second vertical (agencies, research groups, ops teams) | V1 metrics stable in beachhead; Record semantics audit confirms zero software-shape leaked (T3) |
| Protocol-vs-product identity — constitutional Q8 | External parties independently write tools against the format (the signal Git got) |
| Proactive surfacing (suggest rung) | Silent-record signal, or V1.5 by default |
| Erasure workflow (Art. 8) | First real deletion request — handled manually and recorded until then |

---

## 16. EVOLUTION ROADMAP

Each stage emerges from evidence produced by the previous one — never from ambition.

**V1 — Memory (the organism).** Decision capture + recall, one team, files + index, two surfaces max. *Validates: Art. 5–15, 32; T1.*

**V1.5 — Memory that speaks up.** Proactive surfacing at the moment of relevance: "this PR touches decision D-041 (2026): *we rejected this approach because…*". Still zero autonomy — the machine ascends only to the **suggest** rung, per-domain (Art. 22). *Validates: suggest-rung trust mechanics; the compounding claim (§15.4 of the Constitution) in miniature.*

**V2 — Memory across boundaries.** Commitments join decisions as entry kinds (if Q1 evidence says so); cross-team recall; the format opens for external consumers (Art. 43 externalized). First real observation expansion — still consent-shaped. *Validates: registers at scale, Q2 social license, cold-start economics at org level.*

**V3 — Chartered action.** The first capability that *does* something: executing small, reversible, chartered acts grounded in the Record (the bottom of Title VI becomes live). Execution substrate **adopted, not built** (§11). Charters and the full trust ladder are implemented now — because now something exists to govern. *Validates: Art. 20–26, 34–40.*

**V4+ — The platform.** Multiple intelligences over one Record (Art. 28, 31); extension ecosystem (Art. 41–46); the exit-rights commercial bet (Art. 49, Q3) faces its real test at renewal scale.

The Constitution's full shape is reached only at V4 — **and that is the design**: every article dormant in V1 has a named stage where it wakes and gets its evidence.

---

## 17. PRD BLUEPRINT

Structure of the eventual PRD — not the PRD itself. It should be short, evidence-linked, and shrink over time as code becomes truth.

1. **Problem & user** (one page; imports §3, §6 of this report by reference)
2. **The loop** (§7's diagram with acceptance criteria per stage)
3. **Record format specification** — the one lovingly detailed section; doubles as the public format doc (E5's test artifact)
4. **Capture surface #1 spec** — acceptance criteria in behavior terms ("from decision moment to confirmed entry in ≤60s median")
5. **Recall spec** — including the *stated-silence* behavior and citation requirements
6. **Non-goals** (§8's Must-NOT list, verbatim — the PRD's most protective section)
7. **Experiment plan** (§14 imported; each experiment has an owner, a metric, and a kill/continue rule)
8. **Decision records** — the project's own Record entries, kept **in the product itself** from week one (self-hosting; replaces classic ADR appendix)
9. **Milestones** — three: *Loop runs (dogfood)* → *Loop sustains (week-4 metrics)* → *Loop spreads (Ring 1)*
10. **Unknown tracker** (§19, updated as evidence lands; an unknown converted silently is a process defect)

Acceptance criteria style: observable behavior + metric + constitutional article it evidences. Diagrams: the loop (§7) and module seams (§10) only. Anything longer than ~10 pages violates the token-hygiene of product truth.

---

## 18. ARCHITECTURAL REGRET ANALYSIS

Five-year regret projection per major V1 decision:

| Decision | Regret if wrong | Reversibility | Verdict |
|---|---|---|---|
| **Plain-text Record in customer's repo** | Low — worst case: migrate files, which open formats make survivable by design | High | Zero-regret; proceed |
| **Record format semantics** | **Highest regret surface in the product.** A format that leaks software-team shape blocks every future vertical; one that's too abstract burdens V1 | Low (formats fossilize once external parties consume them) | Spend disproportionate design care; version it; delay freezing until E5 + Ring 1 |
| Embedded index (SQLite-class) | Near zero — rebuildable cache | Total | Proceed without ceremony |
| Git as transport | Medium — couples early team UX to dev tooling; non-dev verticals can't follow | Medium (transport sits at a seam) | Accept for beachhead; E7 monitors; seam keeps exit open |
| Rented intelligence behind one seam | Low if the seam holds; high if provider assumptions leak inward | High while seam is honest | Enforce seam by review; provider-swap drill twice a year |
| Decisions as the only V1 entry kind | Medium — if commitments are the truer unit (Q1), early Records skew | Medium (append new kinds; never migrate old) | Accept; additive evolution is the append-only philosophy applied to the schema itself |
| CLI-first, no web app | Low — surfaces are declared disposable | High | Accept; Linear precedent says the niche forgives it |
| No revenue mechanics in V1 | Low at 6–16 weeks; medium if Ring 1 stretches past two quarters | High | Accept with a calendar tripwire |

Aggregate: V1 as specified has **one** high-regret artifact (the format) and deliberately concentrates design effort there. Everything else is built to be thrown away without tears — which is what Gall's Law looks like in practice.

---

## 19. UNKNOWNS REQUIRING EVIDENCE

Carried forward from the Constitution (Q1–Q8) plus new unknowns surfaced by this investigation:

| # | Unknown | Evidence path |
|---|---|---|
| U1 (=Q1) | Fundamental unit: decision, commitment, or case? | Partner Record contents at V1.5 |
| U2 (=Q2) | Social license for observation beyond invoked capture | Not testable until V2; V1 stays consent-invoked |
| U3 (=Q3) | Exit-rights retention economics | Renewal-scale only (V4); architecturally prepaid now |
| U4 | **Demand: will teams sustain feeding a memory tool?** (§3.1) | E1/E2 — the whole of V1 |
| U5 | Confirmation-cost capture achievable with current context quality | E1 edit-distance data |
| U6 | Cold-start floor (entries until value) | E6 |
| U7 | Registers: honest epistemics vs. UX bureaucracy | E4 |
| U8 (=Q8) | Product or protocol | External-consumer signal, V2+ |
| U9 | How much of V1's advantage is AI-shaped (T4's honest question) | E3 deltas |
| U10 | Buyer identity for a bottom-up memory artifact | Ring 1 conversations; deferred with pricing |

---

## 20. FINAL VERDICT

The Constitution survives its first adversarial decomposition intact: no amendment is required, two articles (10, 37) are on notice for evidence, and one load-bearing tension (T1: *the why is not exhaust*) is promoted from hidden flaw to **the founding experiment of the product**. The smallest software product capable of validating the Constitution is not a platform, an engine, or an assistant — it is a **decision record that writes itself and answers "why,"** for one small software team at a time, stored as the team's own files, useful (degraded) with the AI turned off, and constitutionally incapable of acting on anything.

### The six-week answer

> **"If only six weeks of engineering time existed, what should be built first, what should deliberately not be built, and why?"**

**Build first (weeks 1–6):**
1. **The Record format** — plain-text decision entries with provenance, evidence links, and machine-draft marks; documented well enough that a stranger could parse it (this is the only artifact whose design deserves to be slow).
2. **One capture surface** with the draft-and-confirm loop: invoke at a decision moment → context assembled → entry drafted → confirmed in under a minute. Plus the manual path.
3. **Recall** — ask "why," get a cited answer or an honest "the Record is silent."
4. **The index** as a disposable, rebuildable cache.
5. **Dogfood from week one**: the tool's own design decisions live in its own Record.

**Deliberately do not build:** execution or automation of any kind; agents, charters, or trust machinery; a hosted service, accounts, or sync infrastructure (the repo is the sync); a web application; more than one integration; dashboards or analytics; import pipelines; pricing. Every one of these is either governance for actors that don't yet exist, infrastructure the world already provides, or a Version-5 problem cosplaying as a Version-1 requirement.

**Why:** the Constitution's entire edifice rests on one unproven mechanism — that organizational memory can be captured **at confirmation cost rather than authoring cost**. Forty years of knowledge management and the ADR decay curve say every prior attempt died exactly here. Six weeks buys the cleanest possible test of that single hypothesis, on a Record format durable enough to survive being wrong about everything else. If the loop sustains itself, every later stage — surfacing, commitments, chartered action, the platform — has an organism to grow from. If it doesn't, we will have spent six weeks learning that the Constitution's Article 11 or 14 needs amendment *before* a platform was mortgaged on them — which is precisely what a first version is for.

**Maximizes:** learning (one falsifiable bet, instrumented), leverage (all mature infrastructure rented, all design budget on the one irreversible artifact), simplicity (six modules, two of them disposable caches and surfaces), constitutional integrity (17 articles exercised, zero violated, dormant ones staged), and long-term optionality (everything except the format can be replaced without regret).

Ship the loop. Count the recalls. Let the platform earn its own existence one confirmed decision at a time.

---

## APPENDIX — NEW EVIDENCE CITED IN THIS REPORT

(Beyond the constitutional appendix.) ADR adoption/decay: [developersvoice.com on effective ADRs](https://developersvoice.com/blog/architecture/effective-adrs-guide-for-software-architects/), [hidekazu-konishi.com on ADR operations](https://hidekazu-konishi.com/entry/architecture_decision_records_templates_and_operations.html), [Catio 2026 ADR guide](https://www.catio.tech/blog/architecture-decision-record). File/embedded-storage longevity: [SQLite as application file format](https://sqlite.org/appfileformat.html), [Appropriate uses for SQLite](https://sqlite.org/whentouse.html). Opinionated-tool philosophy: [Linear vs Jira analyses](https://tech-insider.org/linear-vs-jira-2026/), [Efficient App comparison](https://efficient.app/compare/linear-vs-jira). Files-over-app: [Obsidian data storage](https://obsidian.md/help/data-storage), [SitePoint Obsidian guide](https://www.sitepoint.com/obsidian-beginner-guide/).

*— End of report —*
