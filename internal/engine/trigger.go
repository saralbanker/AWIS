package engine

// F-2 triggers: DomainEvent ingestion + the SCAN_TRIGGERABLE stage (T5).
//
// Trigger storage is reached through triggerStore — an engine-internal interface
// the SQLite adapter satisfies with ADDITIVE methods (triggers.go), exactly as
// the cancellation flag is reached through cancellationStore (cancel.go).
// StoragePort's 12 methods stay frozen (Blueprint §20).
//
// Semantics are frozen by EDR-011 §4: a trigger-submitted instance receives
// inputs = DomainEvent.payload VERBATIM; a definition matches an event when its
// trigger.type=="event", config["event"]==ev.EventType, def namespace==ev
// namespace, and (no config["filter"] OR the filter Evals true under the event
// scope). An event is consumed once ≥1 workflow fired; unmatched events survive
// until the 7-day TTL prune; multiple matching definitions all fire.

import (
	"context"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/expr"
)

// domainEventTTL is the retention bound for domain_events (Blueprint §10 L724
// "TTL 7d"). Events older than this are pruned each tick (EDR-011 §4).
const domainEventTTL = 7 * 24 * time.Hour

// triggerStore is the additive slice of the SQLite adapter the engine needs for
// domain-event triggers. It is NOT part of the frozen StoragePort (Blueprint
// §20); the concrete adapter satisfies it (triggers.go). Mirrors the
// cancellationStore additive-interface pattern.
type triggerStore interface {
	InsertDomainEvent(ctx context.Context, ev core.DomainEvent) error
	ListUnconsumedDomainEvents(ctx context.Context, namespace string) ([]core.DomainEvent, error)
	MarkDomainEventConsumed(ctx context.Context, eventID string, at time.Time) error
	PruneExpiredDomainEvents(ctx context.Context, olderThan time.Time) (int64, error)
}

// Ingest records an external domain event so a later tick's SCAN_TRIGGERABLE can
// match it against event triggers (F-2). Validation is minimal: event_id,
// event_type, and namespace must be non-empty. EmittedAt defaults to Clock()
// when zero (determinism rule — IMP §3). The SDK TriggerAPI wrapper is M08.
func (e *Engine) Ingest(ctx context.Context, ev core.DomainEvent) error {
	if ev.EventID == "" || ev.EventType == "" || ev.Namespace == "" {
		return fmt.Errorf("engine: Ingest: event_id, event_type, and namespace must be non-empty")
	}
	store, ok := e.storage.(triggerStore)
	if !ok {
		return fmt.Errorf("engine: Ingest: storage does not support domain-event ingestion")
	}
	if ev.EmittedAt.IsZero() {
		ev.EmittedAt = e.now()
	}
	if err := store.InsertDomainEvent(ctx, ev); err != nil {
		return fmt.Errorf("engine: Ingest %s: %w", ev.EventID, err)
	}
	e.logger.Info("ingest", "event_id", ev.EventID, "event_type", ev.EventType, "namespace", ev.Namespace)
	return nil
}

// scanTriggerable is the SCAN_TRIGGERABLE stage (Tick stage 1, before
// SCAN_RUNNABLE): it prunes expired domain events, then matches every remaining
// unconsumed event against registered event triggers and fires the matches. A
// storage that does not implement triggerStore no-ops the stage (graceful
// degradation, mirroring cancellationStore). Errors are logged and never
// returned: a transient trigger-store failure must not wedge the tick.
func (e *Engine) scanTriggerable(ctx context.Context) {
	store, ok := e.storage.(triggerStore)
	if !ok {
		return
	}

	// TTL prune first: the 7-day retention bound is authoritative, so an event
	// past its TTL never fires (EDR-011 §4 / Blueprint §10 L724).
	if n, err := store.PruneExpiredDomainEvents(ctx, e.now().Add(-domainEventTTL)); err != nil {
		e.logger.Warn("domain-event prune failed", "error", err.Error())
	} else if n > 0 {
		e.logger.Info("domain-event prune", "pruned", n)
	}

	// Empty namespace ⇒ all namespaces: the engine has no namespace enumeration,
	// so it lists across namespaces and resolves each event via ListWorkflows.
	events, err := store.ListUnconsumedDomainEvents(ctx, "")
	if err != nil {
		e.logger.Warn("domain-event scan failed", "error", err.Error())
		return
	}
	for _, ev := range events {
		e.matchAndFire(ctx, store, ev)
	}
}

// matchAndFire fires every definition in ev's namespace whose event trigger
// matches ev, then marks ev consumed iff ≥1 Submit SUCCEEDED (EDR-011 §4). A
// Submit failure on one definition is logged and does not prevent the others
// from firing, and does not by itself consume the event.
func (e *Engine) matchAndFire(ctx context.Context, store triggerStore, ev core.DomainEvent) {
	defs, err := e.storage.ListWorkflows(ctx, ev.Namespace)
	if err != nil {
		e.logger.Warn("trigger scan list workflows failed",
			"event_id", ev.EventID, "namespace", ev.Namespace, "error", err.Error())
		return
	}

	fired := 0
	for _, def := range defs {
		if !e.triggerMatches(def, ev) {
			continue
		}
		e.logger.Info("trigger match", "event_id", ev.EventID, "definition_id", def.ID)
		// inputs = DomainEvent.payload VERBATIM (EDR-011 §4).
		if _, err := e.Submit(ctx, def.ID, def.Version, ev.Payload); err != nil {
			e.logger.Warn("trigger submit failed",
				"event_id", ev.EventID, "definition_id", def.ID, "error", err.Error())
			continue
		}
		fired++
		e.logger.Info("trigger fire", "event_id", ev.EventID, "definition_id", def.ID)
	}

	if fired == 0 {
		return // unmatched (or all Submits failed) ⇒ survive to TTL prune.
	}
	if err := store.MarkDomainEventConsumed(ctx, ev.EventID, e.now()); err != nil {
		e.logger.Warn("trigger mark consumed failed", "event_id", ev.EventID, "error", err.Error())
		return
	}
	e.logger.Info("trigger consume", "event_id", ev.EventID, "fired", fired)
}

// triggerMatches reports whether def has an event trigger matching ev
// (EDR-011 §4). A filter that fails to parse or Eval is treated as a NON-match
// and logged — a bad filter must never wedge the scan. Filters are parsed per
// scan (not cached in defView, whose condition index holds only transitions);
// trigger filters are few and short, so the re-parse cost is negligible.
func (e *Engine) triggerMatches(def core.WorkflowDefinition, ev core.DomainEvent) bool {
	// namespace match (EDR-011 §4). ListWorkflows(ev.Namespace) already scopes to
	// the namespace; this guard keeps the invariant explicit and adapter-agnostic.
	if def.Namespace != ev.Namespace {
		return false
	}
	for _, tr := range def.Triggers {
		if tr.Type != core.TriggerTypeEvent {
			continue
		}
		evName, _ := tr.Config["event"].(string)
		if evName != ev.EventType {
			continue
		}
		filter, _ := tr.Config["filter"].(string)
		if filter == "" {
			return true
		}
		cond, err := expr.ParseCondition(filter)
		if err != nil {
			e.logger.Warn("trigger filter parse failed",
				"event_id", ev.EventID, "definition_id", def.ID, "filter", filter, "error", err.Error())
			continue
		}
		ok, err := cond.Eval(expr.Env{Event: ev.Payload})
		if err != nil {
			e.logger.Warn("trigger filter eval failed",
				"event_id", ev.EventID, "definition_id", def.ID, "error", err.Error())
			continue
		}
		if ok {
			return true
		}
	}
	return false
}
