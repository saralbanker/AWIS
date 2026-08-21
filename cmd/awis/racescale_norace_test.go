//go:build !race

package main

// raceScale is 1 for non-race builds; see racescale_race_test.go for the
// -race counterpart.
const raceScale = 1
