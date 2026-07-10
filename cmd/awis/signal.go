package main

// signal.go — 'awis signal' command (TDS-07 §4; M14-C3 T7).
//
// Delivers a signal to a waiting instance using the M07 signal path
// (direct storage write under WAL+busy-timeout; engine picks up on next tick ≤100ms).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

func init() {
	commands["signal"] = command{fn: runSignal, summary: "Deliver a signal to a waiting instance"}
}

// signalOutput is the JSON schema for 'awis signal --json' (TDS-07 §4).
type signalOutput struct {
	InstanceID string  `json:"instance_id"`
	SignalName string  `json:"signal_name"`
	Delivered  bool    `json:"delivered"`
	NextStep   *string `json:"next_step"`
}

func runSignal(args []string) {
	fs := newFlagSet("signal")
	var payloadFlags multiStringFlag
	fs.Var(&payloadFlags, "payload", "Payload key=value or @file.json (repeatable)")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 2 {
		fail(2, "signal requires <instance-id> and <signal-name> arguments", "", "awis signal <instance-id> <signal-name>")
	}
	instanceID := core.InstanceID(rest[0])
	signalName := rest[1]

	payload, err := parseInputFlags(payloadFlags)
	if err != nil {
		fail(1, fmt.Sprintf("signal: payload parse error: %s", err), "", "use --payload key=value or --payload @file.json")
	}

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("signal: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	// Check instance exists and is waiting.
	ctx := context.Background()
	inst, serr := store.GetInstance(ctx, instanceID)
	if serr != nil {
		fail(1, fmt.Sprintf("instance %q not found", instanceID),
			"--data-dir "+globalDataDir,
			"awis status")
	}
	if inst.Status != core.InstanceStatusWaiting {
		fail(1, fmt.Sprintf("instance %q is not in waiting state (status: %s)", instanceID, inst.Status),
			string(instanceID),
			"awis status  # to see current instance state")
	}

	// Deliver signal via runtime (M07 path).
	rt, rerr := sdk.NewRuntime(sdk.Config{
		Namespace: string(inst.Namespace),
		Storage:   store,
	})
	if rerr != nil {
		fail(1, fmt.Sprintf("signal: cannot create runtime: %s", rerr), "", "")
	}

	if err := rt.Signal(ctx, instanceID, signalName, payload); err != nil {
		fail(1, fmt.Sprintf("signal: delivery failed: %s", err),
			string(instanceID),
			"awis status "+string(instanceID))
	}

	// Determine next step (best-effort; instance may have advanced by now).
	var nextStep *string
	if updated, serr := store.GetInstance(ctx, instanceID); serr == nil {
		if len(updated.CurrentSteps) > 0 {
			s := updated.CurrentSteps[0]
			nextStep = &s
		}
	}

	if globalJSON {
		out := signalOutput{
			InstanceID: string(instanceID),
			SignalName: signalName,
			Delivered:  true,
			NextStep:   nextStep,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// TDS-07 §4 human output:
	// Signal delivered: entry_confirmed → i-d4e5f6
	// Instance resumed; next step: append-to-record
	fmt.Printf("Signal delivered: %s → %s\n", signalName, instanceID)
	if nextStep != nil {
		fmt.Printf("Instance resumed; next step: %s\n", *nextStep)
	} else {
		fmt.Printf("Instance resumed.\n")
	}
}
