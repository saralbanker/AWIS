package main

// submit.go — 'awis submit' command (TDS-07 §4; M14-C3 T7).
//
// Creates the instance through the same intake path an embedded app uses
// (storage/sdk API; the running engine picks it up on tick — F-3 semantics).
// Prints instance id immediately. With --wait, polls status to terminal.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

func init() {
	commands["submit"] = command{fn: runSubmit, summary: "Submit a workflow instance"}
}

// submitOutput is the JSON schema for 'awis submit --json' (TDS-07 §4).
type submitOutput struct {
	InstanceID      string  `json:"instance_id"`
	WorkflowID      string  `json:"workflow_id"`
	WorkflowVersion string  `json:"workflow_version"`
	Namespace       string  `json:"namespace"`
	Status          string  `json:"status"`
	DurationMs      *int    `json:"duration_ms"`
}

func runSubmit(args []string) {
	fs := newFlagSet("submit")
	var inputFlags multiStringFlag
	var wait bool
	var timeoutStr string
	fs.Var(&inputFlags, "input", "Input key=value or @file.json (repeatable)")
	fs.BoolVar(&wait, "wait", false, "Block until terminal status")
	fs.StringVar(&timeoutStr, "timeout", "1h", "Maximum wait duration (requires --wait)")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "submit requires a workflow-id argument", "", "awis submit <workflow-id>")
	}
	workflowID := rest[0]

	// Parse --input flags into a map.
	inputs, err := parseInputFlags(inputFlags)
	if err != nil {
		fail(1, fmt.Sprintf("submit: input parse error: %s", err), "", "use --input key=value or --input @file.json")
	}

	// Parse --timeout if --wait.
	var timeout time.Duration
	if wait {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			fail(1, fmt.Sprintf("submit: invalid --timeout value %q: %s", timeoutStr, err), "", "use a valid Go duration, e.g. 1h")
		}
	}

	// Open storage.
	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("submit: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	// Submit via the storage-level engine API:
	// sdk.NewRuntime + Submit inserts the instance row exactly as an embedded app would.
	// The running engine picks it up on the next 100ms tick (F-3).
	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace: "default",
		Storage:   store,
	})
	if err != nil {
		fail(1, fmt.Sprintf("submit: cannot create runtime: %s", err), "", "")
	}

	ctx := context.Background()
	instanceID, err := rt.Submit(ctx, workflowID, inputs)
	if err != nil {
		fail(1, fmt.Sprintf("submit: %s", err),
			"--data-dir "+globalDataDir,
			"awis workflow list  # to see registered workflows")
	}

	// Get initial status.
	inst, serr := store.GetInstance(ctx, instanceID)
	var statusStr, versionStr, namespace string
	if serr == nil {
		statusStr = string(inst.Status)
		versionStr = string(inst.DefinitionVersion)
		namespace = inst.Namespace
	} else {
		statusStr = "pending"
	}

	if !wait {
		if globalJSON {
			out := submitOutput{
				InstanceID:      string(instanceID),
				WorkflowID:      workflowID,
				WorkflowVersion: versionStr,
				Namespace:       namespace,
				Status:          statusStr,
				DurationMs:      nil,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(out)
			return
		}
		// TDS-07 §4 human output (submit only):
		// Submitted: capture-decision v1.0.0
		// Instance:  i-a1b2c3
		// Status:    pending → running
		//
		// Monitor:  awis status
		// Debug:    awis trace i-a1b2c3
		fmt.Printf("Submitted: %s v%s\n", workflowID, versionStr)
		fmt.Printf("Instance:  %s\n", instanceID)
		fmt.Printf("Status:    %s → running\n", statusStr)
		fmt.Println()
		fmt.Printf("Monitor:  awis status\n")
		fmt.Printf("Debug:    awis trace %s\n", instanceID)
		return
	}

	// --wait: poll until terminal status.
	startedAt := time.Now()
	deadline := startedAt.Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		inst, serr := store.GetInstance(ctx, instanceID)
		if serr != nil {
			continue
		}
		switch inst.Status {
		case core.InstanceStatusCompleted, core.InstanceStatusCompensated:
			dur := int(time.Since(startedAt).Milliseconds())
			if globalJSON {
				out := submitOutput{
					InstanceID:      string(instanceID),
					WorkflowID:      workflowID,
					WorkflowVersion: string(inst.DefinitionVersion),
					Namespace:       inst.Namespace,
					Status:          string(inst.Status),
					DurationMs:      &dur,
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetEscapeHTML(false)
				_ = enc.Encode(out)
				return
			}
			// TDS-07 §4 human output (--wait, success):
			// Instance:  i-a1b2c3
			// Status:    completed ✓  Duration: 89s
			fmt.Printf("Instance:  %s\n", instanceID)
			fmt.Printf("Status:    %s ✓  Duration: %ds\n", inst.Status, dur/1000)
			return
		case core.InstanceStatusFailed, core.InstanceStatusCancelled, core.InstanceStatusCompensationFailed:
			dur := int(time.Since(startedAt).Milliseconds())
			if globalJSON {
				out := submitOutput{
					InstanceID:      string(instanceID),
					WorkflowID:      workflowID,
					WorkflowVersion: string(inst.DefinitionVersion),
					Namespace:       inst.Namespace,
					Status:          string(inst.Status),
					DurationMs:      &dur,
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetEscapeHTML(false)
				_ = enc.Encode(out)
				os.Exit(1)
			}
			// TDS-07 §4 human output (--wait, failure):
			// Instance:  i-a1b2c3
			// Status:    failed ✗   Step: draft-entry
			//            awis trace i-a1b2c3
			step := ""
			if len(inst.CurrentSteps) > 0 {
				step = inst.CurrentSteps[0]
			}
			fmt.Printf("Instance:  %s\n", instanceID)
			if step != "" {
				fmt.Printf("Status:    %s ✗   Step: %s\n", inst.Status, step)
			} else {
				fmt.Printf("Status:    %s ✗\n", inst.Status)
			}
			fmt.Printf("           awis trace %s\n", instanceID)
			os.Exit(1)
		}
	}

	// Timeout.
	fail(1,
		fmt.Sprintf("submit --wait: timeout after %s (instance %s still %s)", timeout, instanceID, statusStr),
		"--data-dir "+globalDataDir,
		fmt.Sprintf("awis trace %s", instanceID))
}

// multiStringFlag is a flag.Value for repeatable string flags.
type multiStringFlag []string

func (f *multiStringFlag) String() string { return strings.Join(*f, ", ") }
func (f *multiStringFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

// parseInputFlags parses --input flags into a map[string]any.
// Each value is either "key=value" or "@file.json".
func parseInputFlags(flags multiStringFlag) (map[string]any, error) {
	result := make(map[string]any)
	for _, f := range flags {
		if strings.HasPrefix(f, "@") {
			// Read from JSON file.
			data, err := os.ReadFile(f[1:])
			if err != nil {
				return nil, fmt.Errorf("cannot read input file %q: %w", f[1:], err)
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, fmt.Errorf("input file %q is not valid JSON: %w", f[1:], err)
			}
			for k, v := range m {
				result[k] = v
			}
		} else {
			// key=value.
			eq := strings.IndexByte(f, '=')
			if eq < 0 {
				return nil, fmt.Errorf("input %q: expected key=value format", f)
			}
			result[f[:eq]] = f[eq+1:]
		}
	}
	return result, nil
}
