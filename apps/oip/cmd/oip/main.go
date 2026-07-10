// Command oip is the OIP application entrypoint.
//
// It opens an embedded AWIS runtime backed by SQLite, registers all four OIP
// native handlers, loads the three YAML workflow fixtures unchanged via the
// sdk.LoadWorkflowFile seam (IMP §17 L324; M15-P0 disposition), and starts
// the engine pull loop with signal-based graceful shutdown.
//
// Flags:
//
//	--data-dir       path to the AWIS runtime database directory (default: .awis)
//	--record-root    path to the OIP record root (default: current directory)
//	--namespace      AWIS runtime namespace (default: oip)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/awis/awis/sdk"

	oip "github.com/awis/oip"
)

func main() {
	dataDir := flag.String("data-dir", ".awis", "AWIS runtime database directory")
	recordRoot := flag.String("record-root", ".", "OIP record root (parent of .decisions/)")
	namespace := flag.String("namespace", "oip", "AWIS runtime namespace")
	flag.Parse()

	if err := run(*dataDir, *recordRoot, *namespace); err != nil {
		log.Fatalf("oip: %v", err)
	}
}

func run(dataDir, recordRoot, namespace string) error {
	// Ensure data directory exists.
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir data-dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "runtime.db")
	storage, err := sdk.SQLiteStorage(dbPath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}

	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace: namespace,
		Storage:   storage,
	})
	if err != nil {
		return fmt.Errorf("new runtime: %w", err)
	}

	// Register the four OIP native handlers.
	handlers := []sdk.StepHandler{
		oip.NewRecordAppendHandler(recordRoot),
		oip.NewIndexFTSHandler(recordRoot),
		oip.NewSemanticRankHandler(),
		oip.NewRebuildIndexHandler(recordRoot),
	}
	for _, h := range handlers {
		if err := rt.RegisterHandler(h); err != nil {
			return fmt.Errorf("register handler %q: %w", h.ID(), err)
		}
	}

	// Load and register the three YAML workflow fixtures via the sdk.LoadWorkflowFile
	// seam (M15-P0; IMP §17 L324). Fixtures are byte-unchanged.
	workflowFiles := []string{
		"workflows/capture-decision.yaml",
		"workflows/recall-decision.yaml",
		"workflows/rebuild-index.yaml",
	}
	for _, wfFile := range workflowFiles {
		def, err := sdk.LoadWorkflowFile(wfFile)
		if err != nil {
			return fmt.Errorf("load workflow %q: %w", wfFile, err)
		}
		if err := rt.RegisterWorkflow(def); err != nil {
			return fmt.Errorf("register workflow %q: %w", wfFile, err)
		}
	}

	log.Printf("oip: registered 4 handlers and 3 workflows (namespace=%s, record-root=%s)", namespace, recordRoot)

	// Start with graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("oip: engine starting (press Ctrl+C to stop)")
	if err := rt.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("engine: %w", err)
	}
	log.Printf("oip: shutdown complete")
	return nil
}
