// Package sdk — deterministic mode preset (FR-SDK-07).
package sdk

import (
	"fmt"
	"sync/atomic"
	"time"
)

// deterministicEpoch is the fixed base instant for the deterministic clock.
// A stable non-zero wall time prevents accidental zero-value comparisons in tests.
var deterministicEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// deterministicClockStep is the amount each successive Clock call advances.
const deterministicClockStep = time.Millisecond

// DeterministicMode returns a Config preset for deterministic test execution:
// fixed clock, seeded sequential IDs, manual tick (no poll loop). Caller sets
// Namespace and Storage; Intelligence nil ⇒ NullAdapter (FR-SDK-07).
//
// Clock advances by 1ms per call starting from a fixed epoch so tests that
// compare timestamps see stable, monotonically increasing values.
//
// NewID returns "det-000001", "det-000002", … in the order called. Each call
// to DeterministicMode returns an independent counter, so two Runtimes built
// from separate DeterministicMode() calls produce identical sequences.
//
// TickInterval is set to 24h so the pull loop is never relied upon; tests
// advance execution via Runtime.Tick.
func DeterministicMode() Config {
	var clockCounter int64
	clock := func() time.Time {
		n := atomic.AddInt64(&clockCounter, 1)
		return deterministicEpoch.Add(time.Duration(n-1) * deterministicClockStep)
	}

	var idCounter int64
	newID := func() string {
		n := atomic.AddInt64(&idCounter, 1)
		return fmt.Sprintf("det-%06d", n)
	}

	return Config{
		Clock:        clock,
		NewID:        newID,
		TickInterval: 24 * time.Hour,
	}
}
