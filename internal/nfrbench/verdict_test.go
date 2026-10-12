package nfrbench

import (
	"strings"
	"testing"
	"time"
)

func TestBoundEvaluateOperatorBoundary(t *testing.T) {
	s := Summary{N: 1, P95: 50 * time.Millisecond, Max: 50 * time.Millisecond}
	// "<= 50ms" accepts exactly 50ms; "< 50ms" rejects it.
	le := Bound{Label: "le", Limit: 50 * time.Millisecond, Basis: BasisP95, Binding: true}
	lt := Bound{Label: "lt", Limit: 50 * time.Millisecond, Strict: true, Basis: BasisP95, Binding: true}
	if c, _ := le.Evaluate(s); !c.Pass {
		t.Error("<=50ms: measured 50ms should pass")
	}
	if c, _ := lt.Evaluate(s); c.Pass {
		t.Error("<50ms: measured 50ms should fail")
	}
	over := Summary{N: 1, P95: 51 * time.Millisecond, Max: 51 * time.Millisecond}
	if c, _ := le.Evaluate(over); c.Pass {
		t.Error("<=50ms: measured 51ms should fail")
	}
}

func TestBoundEvaluateBasisSelection(t *testing.T) {
	s := Summary{N: 100, P95: 10 * time.Millisecond, P99: 20 * time.Millisecond, Max: 30 * time.Millisecond}
	cases := []struct {
		basis Basis
		want  time.Duration
	}{
		{BasisP95, 10 * time.Millisecond},
		{BasisP99, 20 * time.Millisecond},
		{BasisMax, 30 * time.Millisecond},
	}
	for _, c := range cases {
		got, err := Bound{Label: "b", Limit: time.Second, Basis: c.basis}.Evaluate(s)
		if err != nil {
			t.Fatalf("basis %s: %v", c.basis, err)
		}
		if got.Measured != c.want {
			t.Errorf("basis %s measured %v, want %v", c.basis, got.Measured, c.want)
		}
	}
	if _, err := (Bound{Label: "bad", Limit: time.Second, Basis: "P50"}).Evaluate(s); err == nil {
		t.Error("unknown basis: want error, got nil")
	}
}

func TestNewResultBindingDecidesOverall(t *testing.T) {
	s := Summary{N: 10, P95: 12 * time.Millisecond, Max: 12 * time.Millisecond}
	// Informational miss (10ms target) with binding pass (50ms gate): overall PASS.
	res, err := NewResult("NFR-P-02", "x", "f", s,
		Bound{Label: "gate", Limit: 50 * time.Millisecond, Basis: BasisP95, Binding: true},
		Bound{Label: "target", Limit: 10 * time.Millisecond, Strict: true, Basis: BasisP95, Binding: false})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("informational miss must not fail the NFR verdict")
	}
	if res.Checks[1].Pass {
		t.Error("informational check should still record its miss")
	}
	// Binding miss: overall MISS.
	res, err = NewResult("NFR-P-02", "x", "f", s,
		Bound{Label: "gate", Limit: 10 * time.Millisecond, Basis: BasisP95, Binding: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Error("binding miss must fail the NFR verdict")
	}
}

func TestNewResultRejectsBadBounds(t *testing.T) {
	s := Summary{N: 1}
	if _, err := NewResult("NFR-X", "x", "f", s); err == nil {
		t.Error("no bounds: want error")
	}
	if _, err := NewResult("NFR-X", "x", "f", s,
		Bound{Label: "info", Limit: time.Second, Basis: BasisMax, Binding: false}); err == nil {
		t.Error("only informational bounds: want error")
	}
	if _, err := NewResult("NFR-X", "x", "f", s,
		Bound{Label: "bad", Limit: time.Second, Basis: "P50", Binding: true}); err == nil {
		t.Error("unknown basis: want error")
	}
}

func TestResultLineCarriesEveryReviewerField(t *testing.T) {
	s := Summary{N: 500, P50: 1 * time.Millisecond, P95: 2 * time.Millisecond,
		P99: 3 * time.Millisecond, Max: 4 * time.Millisecond}
	res, err := NewResult("NFR-P-02", "native step dispatch latency", "1 native step per instance", s,
		Bound{Label: "ship gate (PRD §32)", Limit: 50 * time.Millisecond, Basis: BasisP95, Binding: true},
		Bound{Label: "engineering target (CONTRA-2)", Limit: 10 * time.Millisecond, Strict: true, Basis: BasisP95})
	if err != nil {
		t.Fatal(err)
	}
	line := res.Line()
	for _, want := range []string{
		"NFR-P-02", "native step dispatch latency", "fixture: 1 native step per instance",
		"measured N=500", "P50=1.000ms", "P95=2.000ms", "P99=3.000ms", "max=4.000ms",
		"ship gate (PRD §32) [binding] <= 50ms on P95: measured 2.000ms PASS",
		"engineering target (CONTRA-2) [informational] < 10ms on P95: measured 2.000ms PASS",
		"RESULT: PASS",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("line missing %q\nline: %s", want, line)
		}
	}
	if strings.Contains(line, "\n") {
		t.Errorf("line must be a single line: %q", line)
	}
}

func TestResultLineReportsMiss(t *testing.T) {
	s := Summary{N: 3, P99: 9 * time.Millisecond, Max: 9 * time.Millisecond}
	res, err := NewResult("NFR-P-01", "tick", "100 instances", s,
		Bound{Label: "NFR-P-01 target", Limit: 5 * time.Millisecond, Strict: true, Basis: BasisP99, Binding: true})
	if err != nil {
		t.Fatal(err)
	}
	line := res.Line()
	if !strings.Contains(line, "measured 9.000ms MISS") || !strings.HasSuffix(line, "RESULT: MISS") {
		t.Errorf("miss not reported as MISS: %s", line)
	}
}
