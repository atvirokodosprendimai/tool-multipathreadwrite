package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// ProfileStage5 is ADR-162's Stage 5 profile. It is the stage's, not a
// configuration's: every configuration of every block runs under it, whichever
// configurations a block still runs, so a comparison that closes cannot change
// the other one's arms (ADR-162 Decision 1).
const ProfileStage5 = "stage5"

// pluginName is the plugin configuration P loads (quality-harness ADR-106).
const pluginName = "quality-harness"

// builtinSkills are the skills Claude Code 2.1.302 still lists with bundled
// skills disabled; read from an uncounted probe on 2026-10-17 (ADR-162 T1 S1).
// The canary holds A and B to exactly these, and P to these and the plugin's.
var builtinSkills = []string{"design", "doctor"}

// builtinPlugins are the plugins Claude Code 2.1.302 loads on its own; the
// probe saw all three switched off by enabledPlugins.
var builtinPlugins = []string{"cc-plugin-agents-md@builtin", "cc-plugin-telemetry@builtin", "cc-plugin-plugin-authoring@builtin"}

// PluginRef names the plugin P ran: the tag its checkout is at, and the commit.
type PluginRef struct {
	Tag    string `json:"tag"`
	Commit string `json:"commit"`
}

// profiled is c as the profile runs it: offered Skill beside its own tools.
func profiled(c Config) Config {
	c.Tools = append(slices.Clone(c.Tools), "Skill")
	return c
}

// profileSettings are the settings keys that keep the config directory's
// account-synced skills and plugins, the bundled skills and the built-in
// plugins out of every configuration. syncClaudeAiPlugins is not used: it
// moves synced plugins into the config directory's trash. Each synced plugin is
// switched off by name instead, read from its manifest; the canary refuses any
// that still loads.
func profileSettings(configDir string) (map[string]any, error) {
	enabled := map[string]any{}
	for _, b := range builtinPlugins {
		enabled[b] = false
	}
	dirs, err := filepath.Glob(filepath.Join(configDir, "plugins", "synced", "*", "*"))
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		name := filepath.Base(d)
		if b, err := os.ReadFile(filepath.Join(d, ".claude-plugin", "plugin.json")); err == nil {
			var m struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(b, &m) == nil && m.Name != "" {
				name = m.Name
			}
		}
		enabled[name+"@synced"] = false
	}
	return map[string]any{"syncClaudeAiSkills": false, "disableBundledSkills": true, "enabledPlugins": enabled}, nil
}

// pluginRef reads the plugin checkout's tag and commit. A checkout with
// uncommitted changes, or not at a tag, is refused: the receipt could not say
// what ran.
func pluginRef(dir string) (*PluginRef, error) {
	st, err := gitOut(dir, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("--plugin-dir %s: %w", dir, err)
	}
	if strings.TrimSpace(st) != "" {
		return nil, fmt.Errorf("--plugin-dir %s has uncommitted changes: a receipt could not name what ran", dir)
	}
	tag, err := gitOut(dir, "describe", "--tags", "--exact-match")
	if err != nil {
		return nil, fmt.Errorf("--plugin-dir %s is not at a tag: %w", dir, err)
	}
	commit, err := gitOut(dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	return &PluginRef{Tag: strings.TrimSpace(tag), Commit: strings.TrimSpace(commit)}, nil
}

// resolveProfile fills what the profile needs before anything runs: the
// plugin's identity when a plugin is given, and node, which the plugin's hooks
// run. Without the profile it does nothing.
func resolveProfile(o *RunOptions) error {
	if o.Profile == "" {
		return nil
	}
	if o.PluginDir != "" {
		abs, err := filepath.Abs(o.PluginDir)
		if err != nil {
			return err
		}
		o.PluginDir = abs
		if o.plugin, err = pluginRef(abs); err != nil {
			return err
		}
	}
	if o.Node == "" {
		p, err := exec.LookPath("node")
		if err != nil {
			return errors.New("the stage5 profile needs node, which the plugin's hooks run: put it on PATH or name it with --node")
		}
		o.Node = p
	}
	// Absolute first: the link toolPath makes is read from the attempt's
	// scratch, where a relative target would point elsewhere (the review of T1).
	abs, err := filepath.Abs(o.Node)
	if err != nil {
		return err
	}
	o.Node = abs
	if real, err := filepath.EvalSymlinks(o.Node); err == nil {
		o.Node = real
	}
	if info, err := os.Stat(o.Node); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("--node %s is not an executable file", o.Node)
	}
	return nil
}

// writeCodexStub writes the program CODEX_CLI_PATH names under the profile: it
// refuses, so a plugin skill that would launch an external model reaches none
// (ADR-162 Decision 1).
func writeCodexStub(scratch string) (string, error) {
	p := filepath.Join(scratch, "codex-stub")
	script := "#!/bin/sh\necho 'external models are not part of this stage (ADR-162)' >&2\nexit 2\n"
	return p, os.WriteFile(p, []byte(script), 0o755)
}
