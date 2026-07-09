// QG-5 acceptance test: a complete workflow including WAIT/signal delivery runs
// in a single Go test, asserts wall clock < 1s, zero external dependencies
// (in-memory SQLite only). (PRD QG-5; IMP §27.M9 Val)
package awistesting

import (
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/fixtures"
)

func TestQG5_WaitSignalWithin1s(t *testing.T) {
	start := time.Now()

	def, handlers := fixtures.WaitSignal()
	h := harnessFor(t, handlers)

	// Submit; engine ticks synchronously until the WAIT step parks.
	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("QG-5 Run: %v", err)
	}

	st, _ := h.rt.Status(h.ctx, res.InstanceID)
	if st.Status != core.InstanceStatusWaiting {
		t.Fatalf("QG-5: expected Waiting before signal; got %q", st.Status)
	}

	// Deliver signal; engine ticks until terminal.
	h.Signal(res.InstanceID, "approve", map[string]any{"approved": true})

	if st, _ = h.rt.Status(h.ctx, res.InstanceID); st.Status != core.InstanceStatusCompleted {
		t.Fatalf("QG-5: status = %q, want completed", st.Status)
	}

	elapsed := time.Since(start)
	if elapsed >= time.Second {
		t.Fatalf("QG-5 wall clock %v ≥ 1s (must complete in < 1s)", elapsed)
	}
}
