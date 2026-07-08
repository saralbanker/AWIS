# M06 — Dependency Map

**Upstream (consumes):**
- M02/M03: StoragePort 12/12 (EventLog append w/ monotonicity guard, StateStore optimistic
  upsert + ClaimStep, registry, cache); EDR-005/006/007; RebuildState (equivalence oracle).
- M04: intelligence.Dispatcher + typed FallbackSignal/CapabilityUnavailableError + Usage
  (→ ADJ-8 payload fields); porttest untouched; ValidateChain added here per M04 HANDOFF.
- M05: expr (template resolution of step inputs + condition Eval for transitions and trigger
  filters; Warnings logged per FR-WD-05); validate.Validate at submit (FR-WD-04).

**Downstream (blocks):**
- M07 (signals): replaces the SIGNAL_SCAN stub + signal-step runner_unavailable; takes
  migrations 0003_signals/0004_audit (F-4 renumber); wait_records cleanup on cancel (B4 step 6).
- M08 (SDK): NewRuntime wraps Engine; TriggerAPI intake (F-2) wraps Engine.Ingest; WorkflowRunner
  impl delegates Submit/Cancel/Status/List; latest-version selection at submit.
- M11/M12: register SubprocessRunner/PluginRunner into the engine's runner map.
- M09: harness drives THIS engine with deterministic clock/sources (Config.Clock seam exists).

**Handoff contract:** see HANDOFF.md.
