// Package plugin implements the AWIS plugin subsystem (M12; IMP §27.M12;
// TDS-05 docs/PLUGIN_PROTOCOL.md).
//
// M12-C1: manifest parsing (manifest.go).
// M12-C2: plugin registry storage (internal/storage/plugins.go).
// M12-C3: NDJSON JSON-RPC 2.0 transport (transport.go), CE-pinned lifecycle
// FSM manager (manager.go), PluginRunner for core.StepTypePlugin (runner.go),
// and sdk/runtime.go wiring.
package plugin
