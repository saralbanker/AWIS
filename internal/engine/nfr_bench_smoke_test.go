package engine

// nfr_bench_smoke_test.go runs the NFR-P-01/P-02 measurement paths with tiny
// sample counts under `make test`, so their fixture invariants and verdict
// plumbing stay green between bench runs. The real measurements run only under
// `make bench`.

import (
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/nfrbench"
)

func TestNFRSamplesSmoke(t *testing.T) {
	ticks := runTickSamples(t, 3, 5)
	if len(ticks) != 5 {
		t.Fatalf("NFR-P-01 samples: got %d, want 5", len(ticks))
	}
	for i, d := range ticks {
		if d <= 0 {
			t.Errorf("NFR-P-01 sample %d: non-positive duration %v", i, d)
		}
	}

	dispatch, spans := runDispatchSamples(t, 3)
	if len(dispatch) != 3 || len(spans) != 3 {
		t.Fatalf("NFR-P-02 samples: got %d ticks and %d spans, want 3 and 3", len(dispatch), len(spans))
	}
	for i := range spans {
		if spans[i] < 0 {
			t.Errorf("NFR-P-02 span %d negative: %v", i, spans[i])
		}
	}

	sum, err := nfrbench.Summarize(dispatch)
	if err != nil {
		t.Fatal(err)
	}
	res, err := nfrbench.NewResult("NFR-P-02", "smoke", "smoke fixture", sum,
		nfrbench.Bound{Label: "ship gate", Limit: 50 * time.Millisecond, Basis: nfrbench.BasisP95, Binding: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Line(), "RESULT: ") {
		t.Errorf("result line lacks a RESULT token: %s", res.Line())
	}
}

func TestNFRStepSpan(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := func(et core.EventType, step string, at time.Duration) core.ExecutionEvent {
		return core.ExecutionEvent{EventType: et, StepID: step, EmittedAt: base.Add(at)}
	}
	evs := []core.ExecutionEvent{
		ev(core.EventTypeWorkflowStarted, "", 0),
		ev(core.EventTypeStepStarted, "other", 5*time.Millisecond),
		ev(core.EventTypeStepStarted, "n", 10*time.Millisecond),
		ev(core.EventTypeStepCompleted, "other", 12*time.Millisecond),
		ev(core.EventTypeStepCompleted, "n", 13500*time.Microsecond),
	}
	got, err := nfrStepSpan(evs, "n")
	if err != nil {
		t.Fatal(err)
	}
	if got != 3500*time.Microsecond {
		t.Errorf("span(n) = %v, want 3.5ms (unrelated step must be ignored)", got)
	}
	if _, err := nfrStepSpan(evs[:3], "n"); err == nil {
		t.Error("span without StepCompleted: want error, got nil")
	}
	if _, err := nfrStepSpan(evs, "absent"); err == nil {
		t.Error("span for absent step: want error, got nil")
	}
}
