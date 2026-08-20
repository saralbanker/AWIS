package plugin

// transport_test.go — Golden wire conformance tests (M12-C3).
//
// Verifies that the request/response structs marshalled by transport.go match
// the canonical golden files in testdata/protocol/ (TDS-05 §11).
//
// TRACEABILITY: T8 golden wire conformance.

import (
	"encoding/json"
	"os"
	"testing"
)

// goldenPath returns the absolute path to a golden file.
func goldenPath(t *testing.T, name string) string {
	t.Helper()
	return "testdata/protocol/" + name
}

// readGolden reads a golden JSON file and unmarshals it into a map.
func readGolden(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(goldenPath(t, name))
	if err != nil {
		t.Fatalf("read golden %q: %v", name, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal golden %q: %v", name, err)
	}
	return m
}

// marshalAndUnmarshal marshals v to JSON and then unmarshals it into a map.
func marshalAndUnmarshal(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

// TestGoldenHandshakeRequest verifies the handshake request shape matches
// the golden file (TDS-05 §2, §11).
func TestGoldenHandshakeRequest(t *testing.T) {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "handshake",
		Params:  handshakeParams{ProtocolVersion: "awis-plugin/1"},
	}
	got := marshalAndUnmarshal(t, req)
	want := readGolden(t, "handshake-request.json")
	assertMapsEqual(t, "handshake-request", want, got)
}

// TestGoldenHandshakeResponse verifies the handshake response is parseable and
// matches the golden shape (TDS-05 §2, §11).
func TestGoldenHandshakeResponse(t *testing.T) {
	want := readGolden(t, "handshake-response.json")

	// Parse the golden into rpcResponse.
	data, err := os.ReadFile(goldenPath(t, "handshake-response.json"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal rpcResponse: %v", err)
	}
	got := marshalAndUnmarshal(t, resp)
	assertMapsEqual(t, "handshake-response", want, got)
}

// TestGoldenExecuteRequest verifies the execute request shape (TDS-05 §3, §11).
func TestGoldenExecuteRequest(t *testing.T) {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      "req-2",
		Method:  "execute",
		Params: executeParams{
			Capability: "git.context.assemble",
			StepID:     "assemble-context",
			Inputs: map[string]any{
				"repo_path": "/path/to/repo",
				"ref":       "abc123",
			},
			TimeoutMS: 30000,
		},
	}
	got := marshalAndUnmarshal(t, req)
	want := readGolden(t, "execute-request.json")
	assertMapsEqual(t, "execute-request", want, got)
}

// TestGoldenExecuteResponse verifies the execute success response shape (TDS-05 §3, §11).
func TestGoldenExecuteResponse(t *testing.T) {
	want := readGolden(t, "execute-response.json")
	data, err := os.ReadFile(goldenPath(t, "execute-response.json"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := marshalAndUnmarshal(t, resp)
	assertMapsEqual(t, "execute-response", want, got)
}

// TestGoldenExecuteError verifies the execute error response shape (TDS-05 §3, §11).
func TestGoldenExecuteError(t *testing.T) {
	want := readGolden(t, "execute-error.json")
	data, err := os.ReadFile(goldenPath(t, "execute-error.json"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := marshalAndUnmarshal(t, resp)
	assertMapsEqual(t, "execute-error", want, got)
}

// TestGoldenShutdownNotification verifies the shutdown notification shape (TDS-05 §4, §11).
func TestGoldenShutdownNotification(t *testing.T) {
	// Shutdown is a notification: no id field.
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  "shutdown",
	}
	got := marshalAndUnmarshal(t, req)
	want := readGolden(t, "shutdown-notification.json")
	assertMapsEqual(t, "shutdown-notification", want, got)
}

// TestGoldenBadJSONRPC verifies the bad-jsonrpc.json golden is parseable
// and missing the jsonrpc field (TDS-05 §11 protocol_error case).
func TestGoldenBadJSONRPC(t *testing.T) {
	data, err := os.ReadFile(goldenPath(t, "bad-jsonrpc.json"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// The bad-jsonrpc.json file has no "jsonrpc" field; parsed JSONRPC must be "".
	if resp.JSONRPC != "" {
		t.Errorf("expected empty JSONRPC for bad-jsonrpc.json, got %q", resp.JSONRPC)
	}
}

// assertMapsEqual does a structural deep-equality check on two JSON-decoded maps.
// We compare via re-serialisation to canonical JSON to avoid float/int mismatches.
func assertMapsEqual(t *testing.T, label string, want, got map[string]any) {
	t.Helper()
	wantB, _ := json.Marshal(want)
	gotB, _ := json.Marshal(got)
	if string(wantB) != string(gotB) {
		t.Errorf("%s: wire mismatch:\n  want: %s\n   got: %s", label, wantB, gotB)
	}
}
