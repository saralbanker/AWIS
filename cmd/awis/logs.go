package main

// logs.go — 'awis logs' command (TDS-07 §3 M17; M17-C1).
//
// logs [--tail N] [--level error|info|debug] [--instance <id>]
// Reads runtime structured log from <data-dir>/awis.log.
// start.go gains a file-sink writing to that path (sanctioned addition).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	commands["logs"] = command{fn: runLogs, summary: "Structured log stream from awis.log"}
}

// logLineJSON represents a parsed structured log line.
type logLineJSON struct {
	Time    string `json:"time,omitempty"`
	Level   string `json:"level,omitempty"`
	Message string `json:"msg,omitempty"`
	// Extra carries any remaining fields from the structured log line.
	Extra map[string]any `json:"-"`
}

// logsOutputJSON is the JSON output for 'awis logs --json'.
type logsOutputJSON struct {
	LogFile string        `json:"log_file"`
	Lines   []interface{} `json:"lines"`
}

func runLogs(args []string) {
	fs := newFlagSet("logs")
	var tail int
	var level string
	var instanceID string
	fs.IntVar(&tail, "tail", 50, "Number of log lines to show (default 50)")
	fs.StringVar(&level, "level", "", "Filter by log level: error|info|debug")
	fs.StringVar(&instanceID, "instance", "", "Filter by instance ID")
	mustParse(fs, args)

	logPath := filepath.Join(globalDataDir, "awis.log")

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			if globalJSON {
				out := logsOutputJSON{LogFile: logPath, Lines: []interface{}{}}
				enc := json.NewEncoder(os.Stdout)
				enc.SetEscapeHTML(false)
				_ = enc.Encode(out)
				return
			}
			fmt.Printf("No log file found at %s\n", logPath)
			fmt.Println()
			fmt.Println("  What: awis.log does not exist yet")
			fmt.Println("  What now: awis start  (the runtime writes awis.log on startup)")
			return
		}
		fail(1, fmt.Sprintf("logs: cannot open %s: %s", logPath, err), logPath, "check file permissions")
	}
	defer func() { _ = f.Close() }()

	// Read all lines, then apply tail + filters.
	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fail(1, fmt.Sprintf("logs: read error: %s", err), logPath, "check file integrity")
	}

	// Apply filters.
	var filtered []string
	for _, line := range allLines {
		if level != "" && !logLineMatchesLevel(line, level) {
			continue
		}
		if instanceID != "" && !strings.Contains(line, instanceID) {
			continue
		}
		filtered = append(filtered, line)
	}

	// Apply tail.
	if tail > 0 && len(filtered) > tail {
		filtered = filtered[len(filtered)-tail:]
	}

	if globalJSON {
		parsed := make([]interface{}, 0, len(filtered))
		for _, line := range filtered {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				// Non-JSON line: emit as raw string.
				parsed = append(parsed, line)
			} else {
				parsed = append(parsed, obj)
			}
		}
		out := logsOutputJSON{LogFile: logPath, Lines: parsed}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// Human output: print each line (structured or raw).
	if len(filtered) == 0 {
		fmt.Printf("No log lines found in %s\n", logPath)
		return
	}
	w := io.Writer(os.Stdout)
	for _, line := range filtered {
		_, _ = fmt.Fprintln(w, line)
	}
}

// logLineMatchesLevel reports whether a structured log line matches the given level.
// Handles JSON log lines with a "level" field, and raw lines containing the level string.
func logLineMatchesLevel(line, level string) bool {
	if strings.Contains(strings.ToLower(line), `"level":"`+strings.ToLower(level)+`"`) {
		return true
	}
	// Fallback: substring match for level= (logfmt) or plain level word.
	return strings.Contains(strings.ToLower(line), "level="+strings.ToLower(level))
}
