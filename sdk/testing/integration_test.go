// §19 integration suite — one test per workflow shape, executed on the
// WorkflowTestHarness (IMP §19 integration row; IMP §27.M9 DoD).
// Engine-level precursors (M06/M07 suites) remain untouched; this suite is the
// §19 integration layer going forward (CI-blocking from M09).
package awistesting

import (
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/fixtures"
)

// harnessFor creates a Harness with all handlers from a fixture pre-registered.
func harnessFor(t *testing.T, handlers []core.StepHandler) *Harness {
	t.Helper()
	opts := make([]Option, len(handlers))
	for i, h := range handlers {
		opts[i] = WithStepHandler(h)
	}
	return NewHarness(t, opts...)
}

// assertTerminal calls t.Fatal if the instance's current status is not want.
func assertTerminal(t *testing.T, h *Harness, id core.InstanceID, want core.InstanceStatus) {
	t.Helper()
	st, err := h.rt.Status(h.ctx, id)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Status != want {
		t.Fatalf("status = %q, want %q", st.Status, want)
	}
}

// ── §19 shape 1: Linear ───────────────────────────────────────────────────────

func TestIntegration_Linear(t *testing.T) {
	def, handlers := fixtures.Linear()
	h := harnessFor(t, handlers)

	res, err := h.Run(def, map[string]any{"input": "v"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	assertTerminal(t, h, res.InstanceID, core.InstanceStatusCompleted)
	if got := h.GetOutput(res.InstanceID, "c"); got == nil {
		t.Fatal("expected output key 'c' from final step")
	}
}

// ── §19 shape 2: FanOutJoin ───────────────────────────────────────────────────

func TestIntegration_FanOutJoin(t *testing.T) {
	def, handlers := fixtures.FanOutJoin()
	h := harnessFor(t, handlers)

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	assertTerminal(t, h, res.InstanceID, core.InstanceStatusCompleted)
	if got := h.GetOutput(res.InstanceID, "join"); got == nil {
		t.Fatal("expected output key 'join' from join step")
	}
}

// ── §19 shape 3: RetryExhaustionFallback ─────────────────────────────────────

func TestIntegration_RetryExhaustionFallback(t *testing.T) {
	def, handlers := fixtures.RetryExhaustionFallback()
	h := harnessFor(t, handlers)

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	assertTerminal(t, h, res.InstanceID, core.InstanceStatusCompleted)
	out := h.GetOutput(res.InstanceID, "recovery")
	if out == nil {
		t.Fatal("expected output key 'recovery' from the recovery step")
	}
	m, ok := out.(map[string]any)
	if !ok || m["recovered"] != true {
		t.Fatalf("recovery output = %v, want {recovered: true}", out)
	}
}

// ── §19 shape 4: Compensation ─────────────────────────────────────────────────

func TestIntegration_Compensation(t *testing.T) {
	def, handlers := fixtures.Compensation()
	h := harnessFor(t, handlers)

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// step2 fails → CompensationPlan runs → Compensated.
	assertTerminal(t, h, res.InstanceID, core.InstanceStatusCompensated)
}

// ── §19 shape 5: Cancellation (± compensate) ─────────────────────────────────

func TestIntegration_Cancellation(t *testing.T) {
	t.Run("without_compensate", func(t *testing.T) {
		def, handlers := fixtures.Cancellation()
		h := harnessFor(t, handlers)

		res, err := h.Run(def, nil)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		// Instance is Waiting; cancel it.
		if err := h.rt.Cancel(h.ctx, res.InstanceID, "test-cancel"); err != nil {
			t.Fatalf("Cancel: %v", err)
		}
		h.WaitForCompletion(res.InstanceID, time.Second)
		assertTerminal(t, h, res.InstanceID, core.InstanceStatusCancelled)
	})

	t.Run("with_compensate_plan", func(t *testing.T) {
		def, handlers := fixtures.CancellationWithCompensate()
		h := harnessFor(t, handlers)

		res, err := h.Run(def, nil)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		// Instance ran step1 then parked at the wait step; cancel without compensate.
		if err := h.rt.Cancel(h.ctx, res.InstanceID, "test-cancel"); err != nil {
			t.Fatalf("Cancel: %v", err)
		}
		h.WaitForCompletion(res.InstanceID, time.Second)
		// Plain Cancel (compensate=false) → Cancelled even when plan is defined.
		assertTerminal(t, h, res.InstanceID, core.InstanceStatusCancelled)
	})
}

// ── §19 shape 6: WaitSignal ───────────────────────────────────────────────────

func TestIntegration_WaitSignal(t *testing.T) {
	def, handlers := fixtures.WaitSignal()
	h := harnessFor(t, handlers)

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Workflow parked at wait step.
	st, _ := h.rt.Status(h.ctx, res.InstanceID)
	if st.Status != core.InstanceStatusWaiting {
		t.Fatalf("expected Waiting before signal; got %q", st.Status)
	}

	h.Signal(res.InstanceID, "approve", map[string]any{"approved": true})
	assertTerminal(t, h, res.InstanceID, core.InstanceStatusCompleted)
	if got := h.GetOutput(res.InstanceID, "after"); got == nil {
		t.Fatal("expected output key 'after' from the post-signal step")
	}
}

// ── §19 shape 7: TimeoutAction ────────────────────────────────────────────────

func TestIntegration_TimeoutAction(t *testing.T) {
	def, handlers := fixtures.TimeoutAction()
	h := harnessFor(t, handlers)

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Wait step parked; tick until the 5ms deterministic-clock timeout fires.
	h.WaitForCompletion(res.InstanceID, time.Second)
	// timeout_action=fail → WorkflowFailed.
	assertTerminal(t, h, res.InstanceID, core.InstanceStatusFailed)
}

// ── Determinism proof ─────────────────────────────────────────────────────────

// TestDeterminism_SequenceRepeatable runs the Linear fixture twice in independent
// harnesses and asserts the event-type/step-id sequences are byte-identical
// (IMP §19 test-data policy; Blueprint §27 "Deterministic Execution Mode").
func TestDeterminism_SequenceRepeatable(t *testing.T) {
	type pair struct {
		etype  core.EventType
		stepID string
	}

	run := func() []pair {
		def, handlers := fixtures.Linear()
		h := harnessFor(t, handlers)
		res, err := h.Run(def, nil)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		evs, err := h.ReadEvents(res.InstanceID)
		if err != nil {
			t.Fatalf("ReadEvents: %v", err)
		}
		out := make([]pair, len(evs))
		for i, ev := range evs {
			out[i] = pair{ev.EventType, ev.StepID}
		}
		return out
	}

	seq1 := run()
	seq2 := run()

	if len(seq1) != len(seq2) {
		t.Fatalf("event count: run1=%d, run2=%d — sequences diverged", len(seq1), len(seq2))
	}
	for i := range seq1 {
		if seq1[i] != seq2[i] {
			t.Fatalf("event[%d] mismatch: run1=%+v, run2=%+v", i, seq1[i], seq2[i])
		}
	}
	if len(seq1) == 0 {
		t.Fatal("no events recorded — fixture did not execute")
	}
}
