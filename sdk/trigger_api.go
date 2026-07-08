// TriggerAPI is the F-2 sdk intake surface: it lets application code submit
// domain events that the engine's SCAN_TRIGGERABLE stage matches against
// registered event triggers (Blueprint §12; IMP §27.M8 T9; F-2).
package sdk

import (
	"context"

	"github.com/awis/awis/internal/core"
)

// engineIngester is the minimal interface the TriggerAPI requires from the
// underlying engine (F-2 card spec). The *engine.Engine satisfies it via Ingest.
type engineIngester interface {
	Ingest(ctx context.Context, ev core.DomainEvent) error
}

// TriggerAPI exposes domain-event submission to application code. Obtain
// an instance via Runtime.Triggers().
type TriggerAPI struct {
	engine engineIngester
}

// Triggers returns a TriggerAPI backed by this Runtime's engine.
func (r *Runtime) Triggers() *TriggerAPI {
	return &TriggerAPI{engine: r.eng}
}

// SubmitEvent persists a domain event into the domain_events table.
// Delegates to engine.Ingest, which validates that EventID, EventType, and
// Namespace are non-empty and records the event for SCAN_TRIGGERABLE.
func (t *TriggerAPI) SubmitEvent(ctx context.Context, ev core.DomainEvent) error {
	return t.engine.Ingest(ctx, ev)
}
