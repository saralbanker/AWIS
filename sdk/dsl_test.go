package sdk_test

import (
	"testing"

	"github.com/awis/awis/internal/dsl"
	"github.com/awis/awis/sdk"
)

const helloWorldPath = "../examples/workflows/hello-world.yaml"

// TestLoadWorkflowFile_DeepEqual verifies that sdk.LoadWorkflowFile returns a
// result deep-equal to dsl.ParseFile for the canonical hello-world fixture.
func TestLoadWorkflowFile_DeepEqual(t *testing.T) {
	want, err := dsl.ParseFile(helloWorldPath)
	if err != nil {
		t.Fatalf("dsl.ParseFile: %v", err)
	}
	got, err := sdk.LoadWorkflowFile(helloWorldPath)
	if err != nil {
		t.Fatalf("sdk.LoadWorkflowFile: %v", err)
	}

	if want.ID != got.ID {
		t.Errorf("ID: want %q got %q", want.ID, got.ID)
	}
	if want.Version != got.Version {
		t.Errorf("Version: want %q got %q", want.Version, got.Version)
	}
	if want.Namespace != got.Namespace {
		t.Errorf("Namespace: want %q got %q", want.Namespace, got.Namespace)
	}
	if want.Name != got.Name {
		t.Errorf("Name: want %q got %q", want.Name, got.Name)
	}
	if len(want.Steps) != len(got.Steps) {
		t.Errorf("Steps len: want %d got %d", len(want.Steps), len(got.Steps))
	}
	for i := range want.Steps {
		if i >= len(got.Steps) {
			break
		}
		ws, gs := want.Steps[i], got.Steps[i]
		if ws.ID != gs.ID {
			t.Errorf("Steps[%d].ID: want %q got %q", i, ws.ID, gs.ID)
		}
		if ws.Handler != gs.Handler {
			t.Errorf("Steps[%d].Handler: want %q got %q", i, ws.Handler, gs.Handler)
		}
	}
}

// TestLoadWorkflowFile_ErrorPath verifies that a nonexistent file yields an error.
func TestLoadWorkflowFile_ErrorPath(t *testing.T) {
	_, err := sdk.LoadWorkflowFile("/nonexistent/path/workflow.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
