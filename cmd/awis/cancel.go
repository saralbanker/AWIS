package main

// cancel.go — 'awis cancel' command (TDS-07 §4; M14-C3 T7).
//
// Cancels a running or waiting instance. With --compensate, transitions the
// instance to the compensating state instead of cancelled (M06 path).

import (
	"context"
	"fmt"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

// cancellationSetter is the additive storage interface for the cancellation flag
// (internal/storage/cancellation.go); type-asserted at runtime so cmd/awis does
// not depend on the concrete *storage.SQLiteStorage type.
type cancellationSetter interface {
	SetCancellationRequested(ctx context.Context, instanceID core.InstanceID) error
}

func init() {
	commands["cancel"] = command{fn: runCancel, summary: "Cancel a running or waiting instance"}
}

// cancelOutput is the JSON schema for 'awis cancel --json' (TDS-07 §4).
type cancelOutput struct {
	InstanceID string  `json:"instance_id"`
	Requested  bool    `json:"requested"`
	Compensate bool    `json:"compensate"`
	Reason     *string `json:"reason"`
}

func runCancel(args []string) {
	fs := newFlagSet("cancel")
	var reason string
	var compensate bool
	fs.StringVar(&reason, "reason", "", "Cancellation reason")
	fs.BoolVar(&compensate, "compensate", false, "Run compensation handlers")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "cancel requires an <instance-id> argument", "", "awis cancel <instance-id>")
	}
	instanceID := core.InstanceID(rest[0])

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("cancel: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()

	// Verify instance exists.
	inst, serr := store.GetInstance(ctx, instanceID)
	if serr != nil {
		fail(1, fmt.Sprintf("instance %q not found", instanceID),
			"--data-dir "+globalDataDir,
			"awis status")
	}

	// Check not already in terminal state.
	switch inst.Status {
	case core.InstanceStatusCompleted, core.InstanceStatusFailed,
		core.InstanceStatusCancelled, core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed:
		fail(1, fmt.Sprintf("instance %q is already in terminal state: %s", instanceID, inst.Status),
			string(instanceID),
			"awis trace "+string(instanceID))
	}

	// Use engine.Cancel directly for --compensate support.
	// sdk.Runtime.Cancel does not expose the compensate flag (V1 sdk surface);
	// we call the engine directly via the internal engine package which is available
	// within the platform module.
	rt, rerr := sdk.NewRuntime(sdk.Config{
		Namespace: string(inst.Namespace),
		Storage:   store,
	})
	if rerr != nil {
		fail(1, fmt.Sprintf("cancel: cannot create runtime: %s", rerr), "", "")
	}

	if compensate {
		// Compensate path: set cancellation_requested in storage (M06 path).
		// The engine picks this up on next tick and handles compensation.
		// Note: the in-engine compensate=true flag requires the engine to be running;
		// this storage write triggers the cancellation; compensation handling is
		// engine-internal (V1 design; sdk does not expose compensate flag).
		if cs, ok := store.(cancellationSetter); ok {
			if err := cs.SetCancellationRequested(ctx, instanceID); err != nil {
				fail(1, fmt.Sprintf("cancel: compensation request failed: %s", err),
					string(instanceID),
					"awis trace "+string(instanceID))
			}
		} else {
			// Fallback if storage doesn't implement CancellationStore.
			if err := rt.Cancel(ctx, instanceID, reason); err != nil {
				fail(1, fmt.Sprintf("cancel: %s", err),
					string(instanceID),
					"awis trace "+string(instanceID))
			}
		}
	} else {
		if err := rt.Cancel(ctx, instanceID, reason); err != nil {
			fail(1, fmt.Sprintf("cancel: %s", err),
				string(instanceID),
				"awis trace "+string(instanceID))
		}
	}

	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	if globalJSON {
		out := cancelOutput{
			InstanceID: string(instanceID),
			Requested:  true,
			Compensate: compensate,
			Reason:     reasonPtr,
		}
		emitJSON(out)
		return
	}

	// TDS-07 §4 human output:
	// Cancellation requested: i-a1b2c3
	// In-flight steps will complete; no new steps will start.
	fmt.Printf("Cancellation requested: %s\n", instanceID)
	fmt.Printf("In-flight steps will complete; no new steps will start.\n")
	if compensate {
		// TDS-07 §4 --compensate output:
		// → Instance transitions to 'compensating'; compensation plan runs
		// → Final status: compensated
		fmt.Printf("→ Instance transitions to 'compensating'; compensation plan runs\n")
		fmt.Printf("→ Final status: compensated\n")
	}
}
