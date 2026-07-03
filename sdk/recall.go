package sdk

import "github.com/awis/awis/internal/core"

// RecallAPI is the application-facing read surface over execution history:
// QueryHistory returns matching records, ReplayInstance replays a completed
// instance for debugging, and StepStats returns aggregate step statistics.
type RecallAPI = core.RecallAPI

// HistoryQuery selects execution history for QueryHistory. Its shape is completed
// at M08; it is not part of the G1 format freeze (sdk surface mutable until M08).
type HistoryQuery = core.HistoryQuery

// ExecutionRecord is a summarized history record returned by the RecallAPI. Its
// shape is completed at M08; it is not part of the G1 format freeze (sdk surface
// mutable until M08).
type ExecutionRecord = core.ExecutionRecord

// ReplayTrace is the trace produced by ReplayInstance. Its shape is completed at
// M08; it is not part of the G1 format freeze (sdk surface mutable until M08).
type ReplayTrace = core.ReplayTrace

// StepStatistics are aggregate statistics for a step. Its shape is completed at
// M08; it is not part of the G1 format freeze (sdk surface mutable until M08).
type StepStatistics = core.StepStatistics
