//go:build race

package main

// raceScale widens wall-clock liveness budgets in system_test.go for the
// -race build, where race-detector instrumentation overhead makes 1x budgets
// timing-marginal under full-suite contention.
const raceScale = 6
