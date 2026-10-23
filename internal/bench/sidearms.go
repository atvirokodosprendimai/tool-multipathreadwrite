package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// sideDir is where the side arms' receipts and transcripts sit, inside the
// directory of the block that ran them. ADR-112's readers glob the block
// directory itself, so a G or C attempt is never among the rows they read
// (ADR-163 Decision 9).
const sideDir = "side"

// fetchOff is the switch quality-harness's ADR-116 T2 documents for the
// plugin's own fetch of qh: `off` fetches nothing. G sets it, so the only qh
// its hooks can reach is the one bench staged after verifying it. Read at
// quality-harness v3.8.53-52 (13d14d42), docs/adr/ADR-116.../tasks/T2.
var fetchOff = [2]string{"QUALITY_HARNESS_QH_FETCH", "off"}

// SideC is what the frozen tag spells for C's one setting: a key of the
// attempt's .quality-harness.json and its value (`turnEnd`, `notify` at
// c749638e). The registration names it, because a tag may spell it otherwise.
type SideC struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SideG is what the frozen tag provides for G: the release asset (its name in
// SHA256SUMS and its registered sha256), where the plugin looks for the
// verified binary inside its data directory, the environment that selects the
// Go hook runtime, and a string only the binary's hook output carries.
type SideG struct {
	Asset   string            `json:"asset"`
	SHA256  string            `json:"sha256"`
	StageAs string            `json:"stage_as"`
	Runtime map[string]string `json:"runtime"`
	Marker  string            `json:"marker"`
}

// SideRegistration is the file --side-arms names: BACKLOG's "Pre-registered
// for ADR-163" says which setting or asset a side arm has at the stage's one
// frozen plugin tag, and this file is that, as bench reads it. Tag must be the
// plugin's.
type SideRegistration struct {
	Tag string `json:"tag"`
	C   *SideC `json:"c,omitempty"`
	G   *SideG `json:"g,omitempty"`
}

// SideMeta is what run.json and a side receipt record of an arm: the tag it
// ran at, the sha256 of the asset it staged (G), and the settings it changed.
type SideMeta struct {
	Tag         string            `json:"tag"`
	AssetSHA256 string            `json:"asset_sha256,omitempty"`
	Setting     map[string]string `json:"setting"`
}

var (
	envName  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	sumsLine = regexp.MustCompile(`^([0-9a-f]{64}) [ *](\S.*)$`)
	sha256Hx = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// reservedEnv are the variables an attempt's invocation sets itself.
	reservedEnv = []string{"PATH", "HOME", "USER", "TMPDIR", "LANG", "CLAUDE_CONFIG_DIR", "CLAUDE_PLUGIN_DATA", "CLAUDE_CODE_TMPDIR", "CLAUDE_CODE_SHELL",
		"XDG_CONFIG_HOME", "XDG_STATE_HOME", "GIT_CONFIG_GLOBAL", "PYTHONNOUSERSITE", "CODEX_CLI_PATH"}
)

// isSide says whether name is a side arm.
func isSide(name string) bool { return Configs[name].Side() }

// splitSide separates a list of configurations into the core's (A, B, P, W)
// and the side arms'.
func splitSide(cs []string) (core, side []string) {
	for _, c := range cs {
		if isSide(c) {
			side = append(side, c)
		} else {
			core = append(core, c)
		}
	}
	return core, side
}

// sideAdmitted are the side arms the stage names, in a fixed order: the same
// in every block (ADR-163 Decision 9).
func sideAdmitted(o RunOptions) []string {
	_, side := splitSide(o.Configs)
	slices.Sort(side)
	return slices.Compact(side)
}

// coreFor is what a model runs of A, B, P and W: its own list from
// --configs-for, else the stage's.
func coreFor(o RunOptions, model string) []string {
	if cs, ok := o.ConfigsFor[model]; ok {
		return cs
	}
	core, _ := splitSide(o.Configs)
	return core
}

// sideFor is the side arms a model runs: all the stage admitted where its
// block runs P, and none where it does not (ADR-163 Decision 9).
func sideFor(o RunOptions, model string) []string {
	if !slices.Contains(coreFor(o, model), "P") {
		return []string{}
	}
	return sideAdmitted(o)
}

// outDir is where a configuration's receipts and transcripts are written.
func outDir(o RunOptions, c Config) string {
	if c.Side() {
		return filepath.Join(o.Out, sideDir)
	}
	return o.Out
}

// sideGates refuses a side arm outside the one place the record lets it run:
// stage6, beside P, named in --configs, with its registration and, for G, its
// asset (ADR-163 Decision 9).
func sideGates(o RunOptions) error {
	_, side := splitSide(o.Configs)
	for m, cs := range o.ConfigsFor {
		if _, s := splitSide(cs); len(s) > 0 {
			return fmt.Errorf("--configs-for %s names side arm(s) %v: they are named in --configs and run wherever the model runs P (ADR-163 Decision 9)", m, s)
		}
	}
	if len(side) == 0 {
		if o.QhBinary != "" || o.QhSums != "" {
			return errors.New("--qh-binary and --qh-sums are side arm G's asset, and G is not in --configs (ADR-163 Decision 9)")
		}
		if o.SideArms != "" {
			return errors.New("--side-arms registers side arms G and C, and neither is in --configs (ADR-163 Decision 9)")
		}
		return nil
	}
	switch {
	case o.Profile != ProfileStage6:
		return fmt.Errorf("side arm(s) %v run only under --profile %s (ADR-163 Decision 9)", side, ProfileStage6)
	case !slices.Contains(o.Configs, "P"):
		return fmt.Errorf("side arm(s) %v run only beside P, and --configs has no P (ADR-163 Decision 9)", side)
	case o.SideArms == "":
		return fmt.Errorf("side arm(s) %v need --side-arms, the file that registers their setting or asset at the plugin's tag (ADR-163 Decision 9)", side)
	case slices.Contains(side, "G") && (o.QhBinary == "" || o.QhSums == ""):
		return errors.New("side arm G needs --qh-binary and --qh-sums: the release asset and the SHA256SUMS that names it (ADR-163 Decision 9)")
	case !slices.Contains(side, "G") && (o.QhBinary != "" || o.QhSums != ""):
		return errors.New("--qh-binary and --qh-sums are side arm G's asset, and G is not in --configs (ADR-163 Decision 9)")
	}
	return nil
}

// LoadSideArms reads the registration and refuses it unless it is at the
// plugin's tag and every arm in arms is fully named. A field it does not know
// is refused, so a misspelt setting is not read as an absent one.
func LoadSideArms(path string, plugin *PluginRef, arms []string) (*SideRegistration, error) {
	reg, err := parseSideArms(path)
	if err != nil {
		return nil, err
	}
	if plugin == nil || reg.Tag != plugin.Tag {
		return nil, fmt.Errorf("--side-arms %s is registered at tag %q and the plugin is %v: one tag for every arm (ADR-163 Decision 9)", path, reg.Tag, plugin)
	}
	for _, arm := range arms {
		switch arm {
		case "C":
			if reg.C == nil || reg.C.Key == "" || reg.C.Value == "" {
				return nil, fmt.Errorf(`--side-arms %s names no "c" setting (key and value) for side arm C`, path)
			}
			if reg.C.Key == "check" {
				return nil, fmt.Errorf(`--side-arms %s: C's setting is %q, the key that holds the declared check`, path, reg.C.Key)
			}
		case "G":
			if err := reg.checkG(path); err != nil {
				return nil, err
			}
		}
	}
	return reg, nil
}

// parseSideArms reads the registration: exactly one JSON document and no field
// it does not know, so a trailing second document or a misspelt setting is not
// read as an absent one. Run and canary both read it here.
func parseSideArms(path string) (*SideRegistration, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var reg SideRegistration
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&reg); err != nil {
		return nil, fmt.Errorf("--side-arms %s: %w", path, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("--side-arms %s holds content after its one JSON document", path)
	}
	return &reg, nil
}

func (r *SideRegistration) checkG(path string) error {
	g := r.G
	switch {
	case g == nil || g.Asset == "":
		return fmt.Errorf(`--side-arms %s names no "g" asset for side arm G`, path)
	case !sha256Hx.MatchString(g.SHA256):
		return fmt.Errorf(`--side-arms %s: g's sha256 %q is not 64 lower-case hex digits`, path, g.SHA256)
	case g.StageAs == "" || strings.Contains(g.StageAs, `\`) || !filepath.IsLocal(filepath.FromSlash(g.StageAs)):
		return fmt.Errorf(`--side-arms %s: g's stage_as %q is not a path inside the plugin's data directory`, path, g.StageAs)
	case len(g.Runtime) == 0:
		return fmt.Errorf(`--side-arms %s names no "runtime" setting for G: the environment that selects the Go hooks`, path)
	case g.Marker == "":
		return fmt.Errorf(`--side-arms %s names no "marker" for G: what only the binary's hook output carries, which the canary looks for`, path)
	}
	for k := range g.Runtime {
		if !envName.MatchString(k) || slices.Contains(reservedEnv, k) || k == fetchOff[0] {
			return fmt.Errorf(`--side-arms %s: g's runtime variable %q is not one G may set`, path, k)
		}
	}
	return nil
}

// SideNamed are the side arms a registration names, for a caller that adds
// them to a canary.
func SideNamed(path string) ([]string, error) {
	reg, err := parseSideArms(path)
	if err != nil {
		return nil, err
	}
	var arms []string
	if reg.C != nil {
		arms = append(arms, "C")
	}
	if reg.G != nil {
		arms = append(arms, "G")
	}
	return arms, nil
}

// verifyAsset reads the release asset and the SHA256SUMS beside it and
// returns the asset's bytes and sha256, refusing a binary that is not the
// line SHA256SUMS holds for asset, or not the sha256 registered. Its bytes are
// returned so what is staged is what was verified.
func verifyAsset(binary, sums, asset, registered string) ([]byte, string, error) {
	data, err := os.ReadFile(binary)
	if err != nil {
		return nil, "", fmt.Errorf("--qh-binary: %w", err)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	b, err := os.ReadFile(sums)
	if err != nil {
		return nil, "", fmt.Errorf("--qh-sums: %w", err)
	}
	var lines []string
	for i, l := range strings.Split(string(b), "\n") {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) == "" {
			continue
		}
		m := sumsLine.FindStringSubmatch(l)
		if m == nil {
			return nil, "", fmt.Errorf("--qh-sums %s line %d is malformed: %q, want a sha256 and a file name", sums, i+1, l)
		}
		if m[2] == asset {
			lines = append(lines, m[1])
		}
	}
	switch {
	case len(lines) == 0:
		return nil, "", fmt.Errorf("--qh-sums %s has no line for %s", sums, asset)
	case len(lines) > 1:
		return nil, "", fmt.Errorf("--qh-sums %s names %s twice", sums, asset)
	case lines[0] != got:
		return nil, "", fmt.Errorf("--qh-binary %s has sha256 %s, not its SHA256SUMS line %s in %s", binary, got, lines[0], sums)
	case got != registered:
		return nil, "", fmt.Errorf("--qh-binary %s has sha256 %s and its SHA256SUMS line agrees, but the registered sha256 is %s", binary, got, registered)
	}
	return data, got, nil
}

// resolveSide reads the registration of the side arms the stage names and
// verifies G's asset, before anything runs (ADR-163 Decision 9).
func resolveSide(o *RunOptions) error {
	o.side, o.qh, o.qhSHA = nil, nil, ""
	arms := sideAdmitted(*o)
	if len(arms) == 0 {
		return nil
	}
	reg, err := LoadSideArms(o.SideArms, o.plugin, arms)
	if err != nil {
		return err
	}
	if slices.Contains(arms, "G") {
		if o.qh, o.qhSHA, err = verifyAsset(o.QhBinary, o.QhSums, reg.G.Asset, reg.G.SHA256); err != nil {
			return err
		}
	}
	o.side = reg
	return nil
}

// sideEnv is the environment G adds to P's: the registered runtime setting and
// the switch that turns the plugin's fetch off.
func (o RunOptions) sideEnv() []string {
	env := []string{fetchOff[0] + "=" + fetchOff[1]}
	for k, v := range o.side.G.Runtime {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

// stageQH puts the verified bytes where the plugin looks for qh inside the
// attempt's own data directory.
func (o RunOptions) stageQH(data string) error {
	if o.side == nil || o.side.G == nil || o.qh == nil {
		return errors.New("configuration G has no verified asset to stage")
	}
	p := filepath.Join(data, filepath.FromSlash(o.side.G.StageAs))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, o.qh, 0o755)
}

// notIDChar is what Claude Code replaces with '-' in a plugin's identifier to
// name its data directory.
var notIDChar = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// pluginDataDir is the directory Claude Code gives the plugin's hooks as
// CLAUDE_PLUGIN_DATA: ~/.claude/plugins/data/<id>, under CLAUDE_CONFIG_DIR when
// that is set (plugin reference, "Environment variables"), <id> being the
// plugin's identifier with every character other than [A-Za-z0-9_-] replaced by
// '-'. Assumption: --plugin-dir loads the plugin as quality-harness@inline,
// the source the canary already requires (CheckProfileCanary), so the id is
// quality-harness-inline. Bench cannot set the variable; it can only decide
// what is in that directory when an attempt starts.
func pluginDataDir(o RunOptions) string {
	return filepath.Join(realPath(o.ConfigDir), "plugins", "data", notIDChar.ReplaceAllString(pluginName+"@inline", "-"))
}

// realPath is p with the symlinks of its longest existing prefix resolved, so
// the same directory has one spelling before and after it exists (/var is
// /private/var on macOS) and a canary's path equals the stage's.
func realPath(p string) string {
	rest := ""
	for q := p; ; q = filepath.Dir(q) {
		if r, err := filepath.EvalSymlinks(q); err == nil {
			return filepath.Join(r, rest)
		}
		if filepath.Dir(q) == q {
			return p
		}
		rest = filepath.Join(filepath.Base(q), rest)
	}
}

// freshPluginData gives an attempt the plugin data directory Claude Code will
// use, empty. The config directory is one for the whole stage, so the earlier
// attempt's directory is moved aside under the receipt directory and kept,
// never deleted.
func (o RunOptions) freshPluginData(attemptID string) (string, error) {
	dir := pluginDataDir(o)
	if exists(dir) {
		root := filepath.Join(o.Out, "plugin-data-aside")
		if err := os.MkdirAll(root, 0o755); err != nil {
			return "", err
		}
		kept, err := os.MkdirTemp(root, safe(attemptID)+"-")
		if err != nil {
			return "", err
		}
		// Renaming onto an existing directory is refused on some systems: the old one goes to a name inside.
		if err := os.Rename(dir, filepath.Join(kept, "data")); err != nil {
			return "", fmt.Errorf("moving the plugin data directory %s aside: %w", dir, err)
		}
	}
	return dir, os.MkdirAll(dir, 0o755)
}

// canaryRan says whether a canary ran the side registration and the asset this
// stage runs: a passing canary authorizes no other setting.
func (o RunOptions) canaryRan(reg *SideRegistration, asset string) bool {
	return reg != nil && reflect.DeepEqual(*reg, *o.side) && asset == o.qhSHA
}

// sideMeta is what a side arm's receipt and run.json record of it.
func (o RunOptions) sideMeta(name string) *SideMeta {
	if o.side == nil {
		return nil
	}
	switch name {
	case "G":
		set := map[string]string{fetchOff[0]: fetchOff[1]}
		for k, v := range o.side.G.Runtime {
			set[k] = v
		}
		return &SideMeta{Tag: o.side.Tag, AssetSHA256: o.qhSHA, Setting: set}
	case "C":
		return &SideMeta{Tag: o.side.Tag, Setting: map[string]string{o.side.C.Key: o.side.C.Value}}
	}
	return nil
}

// extraFor is the key C adds to the attempt's .quality-harness.json, and what
// no other configuration adds.
func (o RunOptions) extraFor(c Config) map[string]string {
	if !c.Quiet || o.side == nil || o.side.C == nil {
		return nil
	}
	return map[string]string{o.side.C.Key: o.side.C.Value}
}

// sameSide refuses a block whose side arms are not block 1's: they are fixed
// before the agent pilot and never join or leave later (ADR-163 Decision 9).
func sameSide(first, p plan) error {
	if !slices.Equal(first.SideAdmitted, p.SideAdmitted) {
		return fmt.Errorf("block 1 admitted side arms %v and this block %v: a side arm never joins or leaves after block 1 (ADR-163 Decision 9)", first.SideAdmitted, p.SideAdmitted)
	}
	if !reflect.DeepEqual(first.Side, p.Side) {
		return fmt.Errorf("block 1's side arms ran with %v and this block's with %v: one tag, asset and setting for the stage (ADR-163 Decision 9)", first.Side, p.Side)
	}
	return nil
}

// sameMeta says whether a side receipt records what run.json says its arm ran.
func sameMeta(r *SideMeta, want SideMeta) bool {
	return r != nil && reflect.DeepEqual(*r, want)
}
