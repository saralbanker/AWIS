// Package nfrbench is the shared measurement arithmetic and result-line format
// for the M18 NFR benchmarks (IMP §27.M18; CONTRA-2; PRD §11 NFR-P-01..09,
// §31, §32). It is test-support only: no production package imports it.
//
// Percentiles are nearest-rank over per-iteration wall-clock samples, never an
// ns/op average. Every verdict names the statistic it was judged on (P95, P99
// or max), so a reviewer can read PASS/MISS against the PRD threshold without
// re-deriving units or basis.
package nfrbench

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// ErrNoSamples is returned when a statistic is requested over zero samples.
var ErrNoSamples = errors.New("nfrbench: no samples")

// Percentile returns the nearest-rank p-th percentile of samples, where
// 0 < p <= 100. The rank is ceil(p*N/100) (1-based), so the result is always an
// observed sample and Percentile(samples, 100) is the maximum. The input slice
// is not modified.
func Percentile(samples []time.Duration, p float64) (time.Duration, error) {
	if len(samples) == 0 {
		return 0, ErrNoSamples
	}
	if !(p > 0 && p <= 100) {
		return 0, fmt.Errorf("nfrbench: percentile %v outside (0, 100]", p)
	}
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	rank := int(math.Ceil(p * float64(len(sorted)) / 100))
	if rank < 1 {
		rank = 1
	}
	return sorted[rank-1], nil
}

// Summary is the distribution of one benchmark's per-iteration samples.
type Summary struct {
	N   int
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
	Max time.Duration
}

// Summarize computes the Summary of samples.
func Summarize(samples []time.Duration) (Summary, error) {
	p50, err := Percentile(samples, 50)
	if err != nil {
		return Summary{}, err
	}
	p95, err := Percentile(samples, 95)
	if err != nil {
		return Summary{}, err
	}
	p99, err := Percentile(samples, 99)
	if err != nil {
		return Summary{}, err
	}
	mx, err := Percentile(samples, 100)
	if err != nil {
		return Summary{}, err
	}
	return Summary{N: len(samples), P50: p50, P95: p95, P99: p99, Max: mx}, nil
}

// fmtDur renders a measured duration with an explicit unit: seconds at or above
// one second, milliseconds otherwise, always with three decimals.
func fmtDur(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
	return fmt.Sprintf("%.3fms", float64(d)/float64(time.Millisecond))
}
