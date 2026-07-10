// Package subprocess implements the AWIS subprocess runner (TDS-04).
//
// It satisfies engine.Runner for steps of type core.StepTypeSubprocess: the
// runner spawns one child process per step execution, writes a JSON request
// envelope to the child's stdin, reads a JSON response envelope from stdout,
// and maps every outcome (success, application error, spawn failure, timeout,
// protocol violation) to core.StepResult or *core.StepError per the error
// mapping table in docs/SUBPROCESS_PROTOCOL.md §7.
//
// Wire format, field names, error codes, and timeout behaviour are specified in
// docs/SUBPROCESS_PROTOCOL.md (TDS-04). Golden protocol files shared with the
// Python pytest suite live under testdata/protocol/.
package subprocess
