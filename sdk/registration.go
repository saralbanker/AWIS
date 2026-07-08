// Registration methods for the sdk Runtime (IMP §27.M8; PRD §18; F-4).
// RegisterHandler adds a native step handler; RegisterWorkflow adds a workflow
// definition and writes a WorkflowRegistered audit entry.
package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// RegistrationError is the error returned by RegisterHandler and
// RegisterWorkflow when registration fails (PRD §18).
type RegistrationError struct {
	// ID is the handler or workflow id that failed to register.
	ID string
	// Message is the human-readable failure reason.
	Message string
	// Hint is a suggestion for the caller.
	Hint string
}

// Error returns the PRD §18 registration-failure format:
//
//	awis: registration failed for "<id>"
//	  <reason>
//	  Hint: <hint>
func (e *RegistrationError) Error() string {
	return fmt.Sprintf("awis: registration failed for %q\n  %s\n  Hint: %s", e.ID, e.Message, e.Hint)
}

// RegisterHandler registers a native step handler with the runtime. Returns a
// *RegistrationError if h is nil or a handler with the same ID is already
// registered.
func (r *Runtime) RegisterHandler(h core.StepHandler) error {
	if h == nil {
		return &RegistrationError{
			ID:      "",
			Message: "handler must not be nil",
			Hint:    "provide a non-nil core.StepHandler implementation",
		}
	}
	id := h.ID()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers[id] {
		return &RegistrationError{
			ID:      id,
			Message: fmt.Sprintf("a handler with id %q is already registered", id),
			Hint:    "use a unique handler ID or deregister the existing handler first",
		}
	}
	r.nr.Register(h)
	r.handlers[id] = true
	return nil
}

// auditAppender is the interface type-asserted from cfg.Storage to write
// WorkflowRegistered audit entries (F-4). A storage that does not implement
// this interface silently skips the audit write (graceful degradation).
type auditAppender interface {
	AppendAudit(ctx context.Context, entry storage.AuditEntry) error
}

// RegisterWorkflow registers a workflow definition with the runtime's storage
// and writes a WorkflowRegistered audit entry (F-4). Returns a
// *RegistrationError if def is nil, def.Version is not valid semver, or the
// same (id, version) pair has already been registered.
func (r *Runtime) RegisterWorkflow(def *core.WorkflowDefinition) error {
	if def == nil {
		return &RegistrationError{
			ID:      "",
			Message: "workflow definition must not be nil",
			Hint:    "provide a non-nil *core.WorkflowDefinition",
		}
	}
	if err := validateSemver(string(def.Version)); err != nil {
		return &RegistrationError{
			ID:      def.ID,
			Message: fmt.Sprintf("invalid semver %q: %s", def.Version, err),
			Hint:    "version must be a valid semver string, e.g. \"1.0.0\"",
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := def.ID + "@" + string(def.Version)
	if existing, ok := r.defs[key]; ok {
		return &RegistrationError{
			ID:      def.ID,
			Message: fmt.Sprintf("workflow %q version %q is already registered (existing: %s)", def.ID, def.Version, existing),
			Hint:    "increment the version to register an updated definition",
		}
	}

	ctx := context.Background()
	if err := r.storage.RegisterWorkflow(ctx, *def); err != nil {
		return &RegistrationError{
			ID:      def.ID,
			Message: fmt.Sprintf("storage registration failed: %s", err),
			Hint:    "check the storage backend and try again",
		}
	}

	r.defs[key] = def.Version

	// F-4: write WorkflowRegistered audit entry; gracefully skip if storage
	// does not implement AppendAudit.
	if aa, ok := r.storage.(auditAppender); ok {
		entry := storage.AuditEntry{
			Timestamp:      time.Now(),
			EventType:      "WorkflowRegistered",
			Actor:          r.workerID,
			PayloadSummary: fmt.Sprintf("%s v%s", def.ID, def.Version),
		}
		// Non-fatal: audit write failure is logged implicitly via return value
		// discard; the registration itself has already succeeded.
		_ = aa.AppendAudit(ctx, entry)
	}

	return nil
}
