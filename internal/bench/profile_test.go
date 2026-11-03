package bench

import "testing"

// TestTheProfileSwitchesOffEveryBuiltinPlugin: the built-in plugins the
// stage's Claude Code loads on its own are switched off by name. 2.1.317's
// init line listed cc-plugin-privacy@builtin, which the 2.1.302 list lacked,
// and the stage7 canary failed on it (ADR-192 T3 S2).
func TestTheProfileSwitchesOffEveryBuiltinPlugin(t *testing.T) {
	s, err := profileSettings(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	enabled, _ := s["enabledPlugins"].(map[string]any)
	for _, p := range []string{"cc-plugin-agents-md@builtin", "cc-plugin-telemetry@builtin", "cc-plugin-plugin-authoring@builtin", "cc-plugin-privacy@builtin"} {
		if enabled[p] != false {
			t.Errorf("%s is not switched off: %v", p, enabled)
		}
	}
}
