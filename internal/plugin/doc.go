// Package plugin implements the AWIS plugin subsystem (M12; IMP §27.M12;
// TDS-05 docs/PLUGIN_PROTOCOL.md). This card (M12-C1) provides manifest
// parsing only; transport, FSM, runner, and Python code arrive in C3/C4.
package plugin
