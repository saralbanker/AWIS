package nfrbench

import (
	"errors"
	"math"
	"testing"
	"time"
)

// seq returns 1ms..n*ms in reverse order, so tests also exercise sorting.
func seq(n int) []time.Duration {
	out := make([]time.Duration, 0, n)
	for i := n; i >= 1; i-- {
		out = append(out, time.Duration(i)*time.Millisecond)
	}
	return out
}

func TestPercentileNearestRank(t *testing.T) {
	s := seq(100)
	cases := []struct {
		p    float64
		want time.Duration
	}{
		{1, 1 * time.Millisecond},
		{50, 50 * time.Millisecond},
		{95, 95 * time.Millisecond},
		{99, 99 * time.Millisecond},
		{100, 100 * time.Millisecond},
	}
	for _, c := range cases {
		got, err := Percentile(s, c.p)
		if err != nil {
			t.Fatalf("Percentile(p=%v): %v", c.p, err)
		}
		if got != c.want {
			t.Errorf("Percentile(p=%v) = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestPercentileSmallSampleCounts(t *testing.T) {
	// N=2: P50 rank ceil(1)=1 -> 2ms; P95 rank ceil(1.9)=2 -> 10ms.
	s := []time.Duration{10 * time.Millisecond, 2 * time.Millisecond}
	if got, _ := Percentile(s, 50); got != 2*time.Millisecond {
		t.Errorf("P50 of {10ms,2ms} = %v, want 2ms", got)
	}
	if got, _ := Percentile(s, 95); got != 10*time.Millisecond {
		t.Errorf("P95 of {10ms,2ms} = %v, want 10ms", got)
	}
	// N=1: every percentile is the single observed sample.
	one := []time.Duration{7 * time.Millisecond}
	for _, p := range []float64{1, 50, 99, 100} {
		if got, _ := Percentile(one, p); got != 7*time.Millisecond {
			t.Errorf("Percentile(one, %v) = %v, want 7ms", p, got)
		}
	}
}

func TestPercentileDoesNotMutateInput(t *testing.T) {
	s := []time.Duration{3 * time.Millisecond, 1 * time.Millisecond, 2 * time.Millisecond}
	before := append([]time.Duration(nil), s...)
	if _, err := Percentile(s, 50); err != nil {
		t.Fatal(err)
	}
	for i := range s {
		if s[i] != before[i] {
			t.Fatalf("input mutated at %d: got %v, was %v", i, s[i], before[i])
		}
	}
}

func TestPercentileErrors(t *testing.T) {
	if _, err := Percentile(nil, 50); !errors.Is(err, ErrNoSamples) {
		t.Errorf("empty samples: err = %v, want ErrNoSamples", err)
	}
	for _, p := range []float64{0, -1, 100.0001, math.NaN()} {
		if _, err := Percentile(seq(10), p); err == nil {
			t.Errorf("Percentile(p=%v): want error, got nil", p)
		}
	}
}

func TestSummarize(t *testing.T) {
	got, err := Summarize(seq(100))
	if err != nil {
		t.Fatal(err)
	}
	want := Summary{N: 100, P50: 50 * time.Millisecond, P95: 95 * time.Millisecond,
		P99: 99 * time.Millisecond, Max: 100 * time.Millisecond}
	if got != want {
		t.Errorf("Summarize = %+v, want %+v", got, want)
	}
	if _, err := Summarize(nil); !errors.Is(err, ErrNoSamples) {
		t.Errorf("Summarize(nil) err = %v, want ErrNoSamples", err)
	}
}

func TestFmtDurUnits(t *testing.T) {
	if got := fmtDur(1500 * time.Millisecond); got != "1.500s" {
		t.Errorf("fmtDur(1.5s) = %q, want 1.500s", got)
	}
	if got := fmtDur(250 * time.Microsecond); got != "0.250ms" {
		t.Errorf("fmtDur(250us) = %q, want 0.250ms", got)
	}
}
