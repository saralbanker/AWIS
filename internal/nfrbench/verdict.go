package nfrbench

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Basis names the statistic a Bound is judged on.
type Basis string

// The statistics a bound may be judged on. A PRD threshold with no stated
// percentile is judged on Max: the strictest reading of "< X" over the samples.
const (
	BasisP95 Basis = "P95"
	BasisP99 Basis = "P99"
	BasisMax Basis = "max"
)

// Bound is one threshold from the PRD or CONTRA-2. Strict selects "<" (true)
// or "<=" (false), matching the operator the PRD text uses. A binding bound
// decides the NFR verdict; an informational bound is reported but cannot fail
// it (the CONTRA-2 engineering target).
type Bound struct {
	Label   string
	Limit   time.Duration
	Strict  bool
	Basis   Basis
	Binding bool
}

// Check is one Bound judged against a Summary.
type Check struct {
	Bound    Bound
	Measured time.Duration
	Pass     bool
}

// Evaluate judges b against s on b.Basis.
func (b Bound) Evaluate(s Summary) (Check, error) {
	var m time.Duration
	switch b.Basis {
	case BasisP95:
		m = s.P95
	case BasisP99:
		m = s.P99
	case BasisMax:
		m = s.Max
	default:
		return Check{}, fmt.Errorf("nfrbench: bound %q has unknown basis %q", b.Label, b.Basis)
	}
	pass := m <= b.Limit
	if b.Strict {
		pass = m < b.Limit
	}
	return Check{Bound: b, Measured: m, Pass: pass}, nil
}

// Result is one NFR benchmark's report: the measured distribution, the fixture
// it ran against, and every bound it was judged against.
type Result struct {
	NFR     string
	Name    string
	Fixture string
	Summary Summary
	Checks  []Check
	Pass    bool
}

// NewResult judges bounds against s. Pass is true if and only if every binding
// bound passes. It is an error to supply no bounds, or no binding bound.
func NewResult(nfr, name, fixture string, s Summary, bounds ...Bound) (Result, error) {
	if len(bounds) == 0 {
		return Result{}, errors.New("nfrbench: no bounds supplied")
	}
	res := Result{NFR: nfr, Name: name, Fixture: fixture, Summary: s, Pass: true}
	binding := 0
	for _, bd := range bounds {
		c, err := bd.Evaluate(s)
		if err != nil {
			return Result{}, err
		}
		res.Checks = append(res.Checks, c)
		if bd.Binding {
			binding++
			if !c.Pass {
				res.Pass = false
			}
		}
	}
	if binding == 0 {
		return Result{}, errors.New("nfrbench: no binding bound supplied")
	}
	return res, nil
}

// Line renders the single result line that `make bench` prints for this NFR.
// It carries the NFR id, the fixture, the measured percentiles and sample
// count, one clause per threshold with its operator, basis and verdict, and a
// final RESULT token that reflects binding bounds only.
func (r Result) Line() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s | fixture: %s | measured N=%d: P50=%s P95=%s P99=%s max=%s",
		r.NFR, r.Name, r.Fixture, r.Summary.N,
		fmtDur(r.Summary.P50), fmtDur(r.Summary.P95), fmtDur(r.Summary.P99), fmtDur(r.Summary.Max))
	for _, c := range r.Checks {
		kind := "binding"
		if !c.Bound.Binding {
			kind = "informational"
		}
		op := "<="
		if c.Bound.Strict {
			op = "<"
		}
		verdict := "MISS"
		if c.Pass {
			verdict = "PASS"
		}
		fmt.Fprintf(&b, " | %s [%s] %s %s on %s: measured %s %s",
			c.Bound.Label, kind, op, c.Bound.Limit, c.Bound.Basis, fmtDur(c.Measured), verdict)
	}
	result := "MISS"
	if r.Pass {
		result = "PASS"
	}
	fmt.Fprintf(&b, " | RESULT: %s", result)
	return b.String()
}

// Report prints res as one line on stdout, without the testing package's
// file:line prefix, so `make bench` output can be read and grepped directly.
// It also records the percentiles as benchmark metrics in milliseconds. A MISS
// is reported, never failed here: it is a measured result for C4, not a run
// failure.
func Report(b *testing.B, res Result) {
	b.Helper()
	_, _ = fmt.Fprintln(os.Stdout, res.Line())
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	b.ReportMetric(ms(res.Summary.P50), "p50-ms")
	b.ReportMetric(ms(res.Summary.P95), "p95-ms")
	b.ReportMetric(ms(res.Summary.P99), "p99-ms")
	b.ReportMetric(ms(res.Summary.Max), "max-ms")
	b.ReportMetric(float64(res.Summary.N), "samples")
}
