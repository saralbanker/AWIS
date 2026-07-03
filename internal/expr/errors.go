package expr

import "fmt"

// ParseError is returned by ParseTemplate and ParseCondition when a source string
// is not well-formed under its grammar. Position is the byte offset into the
// source string at which the problem was detected; it is always in bounds
// (0 <= Position <= len(src)) so callers (and the M10 YAML layer / M14 CLI
// renderer) can point at the exact offending construct. Msg is a human-readable
// explanation. Every prohibited construct is rejected with a precise Position.
type ParseError struct {
	Position int
	Msg      string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Position, e.Msg)
}
