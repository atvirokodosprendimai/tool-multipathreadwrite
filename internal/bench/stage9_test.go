package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// pinRig is the runner tests' fixture task run by a fake agent that does what
// ADR-206 T1's rows need to see: it records the bytes of the mrw on its PATH,
// keeps the prompt file it was given and the settings it was started with,
// and, when a test asks by creating a flag file in dir, has that mrw write a
// file under the state home, replaces the file at --mrw, commits its change or
// leaves a file uncommitted. Its cost for the Nth run is line N of dir/costs.
type pinRig struct {
	opts RunOptions
	dir  string // the fake's own record of what it ran and was given
	live string // the file at --mrw
	sha  string // the SHA-256 of live as written
	kept string // the path under the state home the fake mrw writes
}

const (
	pinMrw  = "#!/bin/sh\n[ \"$1\" = version ] && { echo 'mrw pin one'; exit 0; }\nif [ \"$1\" = writestate ]; then mkdir -p \"$XDG_STATE_HOME/mrw\" && echo exposed > \"$XDG_STATE_HOME/mrw/x.jsonl\"; exit 0; fi\necho ran >> \"%s/instructions.ran\"\necho 'the binary instructions'\n"
	evilMrw = "#!/bin/sh\n[ \"$1\" = version ] && { echo 'mrw evil'; exit 0; }\necho 'evil instructions'\n"
)

func hashOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func newPinRig(t *testing.T, configs []string, repeats int) *pinRig {
	t.Helper()
	opts, _ := runOptions(t)
	work := filepath.Dir(opts.Out)
	dir := filepath.Join(work, "fake")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &pinRig{opts: opts, dir: dir, live: filepath.Join(work, "mrw-pinned"), kept: "mrw/x.jsonl"}
	mrw := []byte(fmt.Sprintf(pinMrw, dir))
	r.sha = hashOf(mrw)
	for p, b := range map[string][]byte{r.live: mrw, filepath.Join(dir, "evil"): []byte(evilMrw)} {
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "secret", "mrw"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret", "mrw", "x.jsonl"), []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(opts.Tasks[0].Dir, "reference.patch")
	script := `#!/bin/sh
D="` + dir + `"
prompt=""; settings=""; tools=""; prev=""
for a in "$@"; do
	[ "$a" = "--version" ] && { echo "9.9.9 (Claude Code)"; exit 0; }
	[ "$prev" = "--tools" ] && tools="$a"
	[ "$prev" = "--append-system-prompt-file" ] && prompt="$a"
	[ "$prev" = "--settings" ] && settings="$a"
	prev="$a"
done
n=$(cat "$D/count" 2>/dev/null || echo 0); n=$((n+1)); echo $n > "$D/count"
[ -n "$prompt" ] && cp "$prompt" "$D/prompt.$n"
cp "$settings" "$D/settings.$n"
m=$(command -v mrw || true)
if [ -n "$m" ]; then
	cp "$m" "$D/ranmrw.$n"
	"$m" version > "$D/ranout.$n"
	[ -f "$D/writestate" ] && mrw writestate
	S="$(dirname "$XDG_STATE_HOME")"
	[ -f "$D/scratchlog" ] && echo '{"x":1}' > "$S/stage9-exposed.jsonl"
	[ -f "$D/scratchlink" ] && ln -s "$D/secret/mrw/x.jsonl" "$S/stage9-exposed.jsonl"
	[ -f "$D/scratchdir" ] && mkdir "$S/stage9-exposed.jsonl"

	[ -f "$D/big" ] && dd if=/dev/zero bs=1048576 count=17 > "$XDG_STATE_HOME/mrw/x.jsonl" 2>/dev/null
	[ -f "$D/linkstate" ] && { rm -rf "$XDG_STATE_HOME"; ln -s "$D/secret" "$XDG_STATE_HOME"; }
	[ -f "$D/linkfile" ] && { rm -f "$XDG_STATE_HOME/mrw/x.jsonl"; ln -s "$D/secret/mrw/x.jsonl" "$XDG_STATE_HOME/mrw/x.jsonl"; }
	[ -f "$D/replace" ] && [ "$n" = 1 ] && cp "$D/evil" "` + r.live + `"
	[ -f "$D/tamper" ] && cp -f "$D/evil" "$m"
fi
git apply "$(cat "$D/patch" 2>/dev/null || echo "` + patch + `")" >/dev/null 2>&1
[ -f "$D/forceadd" ] && { printf '*.log\n' >> .git/info/exclude; echo x > forced.log; git add -f forced.log; }
[ -f "$D/commit" ] && git -c user.name=t -c user.email=t@t commit -qam change
[ -f "$D/badindex" ] && echo garbage > .git/index
[ -f "$D/untracked" ] && echo hi > created.txt
[ -f "$D/untrackedp" ] && echo hi > p/created.txt
[ -f "$D/bigdiff" ] && dd if=/dev/zero bs=1048576 count=17 2>/dev/null | tr '\0' a > big.txt
[ -f "$D/hang" ] && sleep 30
[ -f "$D/sigint" ] && { kill -INT $PPID; sleep 5; }

im=stub-model; [ -f "$D/initmodel" ] && im=$(sed -n "${n}p" "$D/initmodel")
case "$im" in "") mf="";; *) mf='"model":"'$im'",';; esac
printf '{"type":"system","subtype":"init",%s"tools":["%s"],"mcp_servers":[]}\n' "$mf" "$(printf '%s' "$tools" | sed 's/,/","/g')"

printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}'
cost=0.25; [ -f "$D/costs" ] && cost=$(sed -n "${n}p" "$D/costs")
case "$cost" in ""|none) field="";; *) field='"total_cost_usd":'$cost',';; esac
printf '{"type":"result","subtype":"success","is_error":false,"num_turns":2,"duration_ms":10,%s"usage":{"input_tokens":100,"output_tokens":20}}\n' "$field"
`
	claude := filepath.Join(dir, "claude")
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	r.opts.Claude, r.opts.Mrw, r.opts.Configs, r.opts.Repeats = claude, r.live, configs, repeats
	r.opts.MrwSHA256 = r.sha
	r.writeCanary(t, "")
	return r
}

// writeCanary makes the canary the rig's runs are judged on: one made with
// the pinned mrw, and with the instruction file whose hash is given.
func (r *pinRig) writeCanary(t *testing.T, instrSHA string) {
	t.Helper()
	body := fmt.Sprintf(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","model":"stub-model","mrw_sha256":%q,"mrw_instructions_sha256":%q}`, r.sha, instrSHA)
	if err := os.WriteFile(r.opts.Canary, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// unpin turns the run into one that pins nothing: it then runs the file at
// --mrw itself, so the file must be executable, which a pinned rig's is not.
func (r *pinRig) unpin(t *testing.T) {
	t.Helper()
	r.opts.MrwSHA256 = ""
	if err := os.Chmod(r.live, 0o755); err != nil {
		t.Fatal(err)
	}
}

// flag makes the flag file that turns one of the fake's abilities on.
func (r *pinRig) flag(t *testing.T, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, name), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r *pinRig) costs(t *testing.T, lines ...string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, "costs"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ran is the SHA-256 of the mrw the fake found on its PATH in each run, in order.
func (r *pinRig) ran(t *testing.T) []string {
	t.Helper()
	var got []string
	for n := 1; ; n++ {
		b, err := os.ReadFile(filepath.Join(r.dir, fmt.Sprintf("ranmrw.%d", n)))
		if err != nil {
			return got
		}
		got = append(got, hashOf(b))
	}
}

func fileMap(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return m
}

func readReceipt(t *testing.T, out, id string) Receipt {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(out, id+".receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var r Receipt
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestARunPinsItsMrwItsInstructionsAndItsSpend(t *testing.T) {
	const b1, b2 = "fix-one.B.stub-model.1", "fix-one.B.stub-model.2"

	t.Run("a wrong hash refuses the run", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		other := hashOf([]byte("another build"))
		rig.opts.MrwSHA256 = other
		// A canary of the wrong hash too, so only the check of --mrw itself can refuse.
		body := fmt.Sprintf(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","model":"stub-model","mrw_sha256":%q}`, other)
		if err := os.WriteFile(rig.opts.Canary, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		err := Run(rig.opts)
		if err == nil || strings.Contains(err.Error(), "canary") || !strings.Contains(err.Error(), other) || !strings.Contains(err.Error(), rig.sha) {
			t.Fatalf("a wrong --mrw-sha256: %v, want a refusal of --mrw naming both hashes", err)
		}
		if _, err := os.Stat(filepath.Join(rig.dir, "count")); err == nil {
			t.Error("an attempt ran after the refusal")
		}
		if _, err := os.Stat(filepath.Join(rig.opts.Out, "run.json")); err == nil {
			t.Error("a run.json was written before the refusal")
		}
	})

	t.Run("the attempt runs the pinned bytes", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 2)
		rig.flag(t, "replace")
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		got := rig.ran(t)
		if len(got) != 2 || got[0] != rig.sha || got[1] != rig.sha {
			t.Errorf("the attempts ran mrw %v, want the hashed bytes %s twice (the file at --mrw was replaced after the first)", got, rig.sha)
		}
		if now, _ := os.ReadFile(rig.live); hashOf(now) == rig.sha {
			t.Error("the fixture did not replace the file at --mrw, so the row proves nothing")
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["mrw_sha256"] != rig.sha || m["mrw_version"] != "mrw pin one" {
			t.Errorf("run.json mrw_sha256 %v, mrw_version %v, want %s and the version of the hashed bytes", m["mrw_sha256"], m["mrw_version"], rig.sha)
		}
		for _, id := range []string{b1, b2} {
			if r := readReceipt(t, rig.opts.Out, id); r.MrwSHA256 != rig.sha {
				t.Errorf("%s: mrw_sha256 %q, want %s", id, r.MrwSHA256, rig.sha)
			}
		}
		for n := 1; n <= 2; n++ {
			if out, err := os.ReadFile(filepath.Join(rig.dir, fmt.Sprintf("ranout.%d", n))); err != nil || string(out) != "mrw pin one\n" {
				t.Errorf("run %d: the PATH mrw printed %q (%v) for `version`, want the pinned bytes' output", n, out, err)
			}
			if pr, err := os.ReadFile(filepath.Join(rig.dir, fmt.Sprintf("prompt.%d", n))); err != nil || !bytes.Contains(pr, []byte("the binary instructions")) {
				t.Errorf("run %d: the prompt is %q (%v), want what the pinned bytes' `instructions` print: the file at --mrw is not executable", n, pr, err)
			}
		}
	})

	t.Run("a run without the pin records no mrw hash", func(t *testing.T) {
		rig := newPinRig(t, []string{"A", "B"}, 1)
		rig.unpin(t)
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["mrw_sha256"] != nil {
			t.Errorf("an unpinned run.json names a hash: %v", m["mrw_sha256"])
		}
		if r := readReceipt(t, rig.opts.Out, b1); r.MrwSHA256 != "" {
			t.Errorf("an unpinned receipt names a hash: %q", r.MrwSHA256)
		}
	})

	t.Run("the instructions are the file's", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		text := []byte("one frozen instruction\nsecond line\n")
		file := filepath.Join(rig.dir, "instructions.txt")
		if err := os.WriteFile(file, text, 0o644); err != nil {
			t.Fatal(err)
		}
		rig.opts.MrwInstructions = file
		rig.writeCanary(t, hashOf(text))
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		prompt, err := os.ReadFile(filepath.Join(rig.dir, "prompt.1"))
		if err != nil || !bytes.HasSuffix(prompt, text) || bytes.Contains(prompt, []byte("the binary instructions")) {
			t.Errorf("the prompt file is %q (%v), want one ending in the file's bytes and nothing of the binary's", prompt, err)
		}
		if _, err := os.Stat(filepath.Join(rig.dir, "instructions.ran")); err == nil {
			t.Error("the binary's instructions were run beside the file")
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["mrw_instructions_sha256"] != hashOf(text) {
			t.Errorf("run.json mrw_instructions_sha256 %v, want %s", m["mrw_instructions_sha256"], hashOf(text))
		}
		// The unpinned run gets the binary's text, and says nothing of a file.
		ctl := newPinRig(t, []string{"B"}, 1)
		if err := Run(ctl.opts); err != nil {
			t.Fatal(err)
		}
		prompt, _ = os.ReadFile(filepath.Join(ctl.dir, "prompt.1"))
		if !bytes.Contains(prompt, []byte("the binary instructions")) {
			t.Errorf("without --mrw-instructions the prompt is %q, want the binary's instructions", prompt)
		}
		if m := fileMap(t, filepath.Join(ctl.opts.Out, "run.json")); m["mrw_instructions_sha256"] != nil {
			t.Errorf("a run with no instructions file names a hash: %v", m["mrw_instructions_sha256"])
		}
	})

	t.Run("the kept file is beside the receipt", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "writestate")
		rig.opts.KeepStateFile = rig.kept
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(rig.opts.Out, b1+".kept"))
		if err != nil || string(got) != "exposed\n" {
			t.Fatalf("the kept file is %q (%v), want what the fake mrw wrote", got, err)
		}
		if r := readReceipt(t, rig.opts.Out, b1); r.KeptStateFileSHA256 != hashOf(got) {
			t.Errorf("kept_state_file_sha256 %q, want %s", r.KeptStateFileSHA256, hashOf(got))
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["keep_state_file"] != rig.kept {
			t.Errorf("run.json keep_state_file %v, want %s", m["keep_state_file"], rig.kept)
		}
		// Not written: no file and no key.
		none := newPinRig(t, []string{"B"}, 1)
		none.opts.KeepStateFile = none.kept
		if err := Run(none.opts); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(none.opts.Out, b1+".kept")); err == nil {
			t.Error("a kept file exists for an attempt that wrote none")
		}
		if b, _ := os.ReadFile(filepath.Join(none.opts.Out, b1+".receipt.json")); bytes.Contains(b, []byte("kept_state_file_sha256")) {
			t.Error("the receipt names a kept file that does not exist")
		}
		// Not asked for: nothing kept even though the file was written.
		off := newPinRig(t, []string{"B"}, 1)
		off.flag(t, "writestate")
		if err := Run(off.opts); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(off.opts.Out, b1+".kept")); err == nil {
			t.Error("a state file was kept without --keep-state-file")
		}
		if m := fileMap(t, filepath.Join(off.opts.Out, "run.json")); m["keep_state_file"] != nil {
			t.Errorf("run.json names a setting nobody gave: %v", m["keep_state_file"])
		}
	})

	t.Run("the run stops on known spend", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 3)
		rig.costs(t, "0.60", "0.60", "0.60")
		rig.opts.MaxSpend = 1.00
		err := Run(rig.opts)
		if !errors.Is(err, ErrSpendStop) || errors.Is(err, ErrDiskFloor) {
			t.Fatalf("a run past its figure: %v, want ErrSpendStop", err)
		}
		if got, floor := ExitCode(err), ExitCode(ErrDiskFloor); got == floor || got == 0 || got == 1 {
			t.Errorf("the spend stop exits %d: it needs a status of its own (disk floor %d, failure 1)", got, floor)
		}
		if n := len(receipts(t, rig.opts.Out)); n != 2 {
			t.Errorf("%d receipts, want the two attempts that fitted the figure before it was reached", n)
		}
		if m, _ := filepath.Glob(filepath.Join(rig.opts.Out, "*.started")); len(m) != 2 {
			t.Errorf("started markers %v, want two: the third attempt must not begin", m)
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["max_spend_usd"] != 1.0 {
			t.Errorf("run.json max_spend_usd %v, want 1", m["max_spend_usd"])
		}
		raised := rig.opts
		raised.MaxSpend = 2.00
		if err := Run(raised); err == nil || errors.Is(err, ErrSpendStop) || !strings.Contains(err.Error(), "another plan") {
			t.Errorf("a raised figure on the same directory: %v, want it refused as another plan", err)
		}
		if n := len(receipts(t, rig.opts.Out)); n != 2 {
			t.Errorf("%d receipts after the refused raise, want 2", n)
		}
		// Under the figure, the same run completes: the stop is the figure's.
		ok := newPinRig(t, []string{"B"}, 3)
		ok.costs(t, "0.20", "0.20", "0.20")
		ok.opts.MaxSpend = 1.00
		if err := Run(ok.opts); err != nil {
			t.Errorf("a run under its figure: %v", err)
		}
	})

	t.Run("a missing cost is not zero", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 4)
		rig.costs(t, "none", "none", "none", "none")
		rig.opts.MaxSpend = 9
		err := Run(rig.opts)
		if !errors.Is(err, ErrSpendStop) || !strings.Contains(err.Error(), "no cost") {
			t.Fatalf("three receipts with no cost: %v, want a spend stop naming the missing costs", err)
		}
		if n := len(receipts(t, rig.opts.Out)); n != 3 {
			t.Errorf("%d receipts, want 3 before the stop", n)
		}
		// A recorded zero is a cost: four of them do not stop the run.
		zero := newPinRig(t, []string{"B"}, 4)
		zero.costs(t, "0", "0", "0", "0")
		zero.opts.MaxSpend = 9
		if err := Run(zero.opts); err != nil {
			t.Errorf("four recorded costs of 0: %v", err)
		}
	})

	t.Run("a receipt with another hash fails resume and verify", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		if err := Run(rig.opts); err != nil {
			t.Fatalf("a resume of an intact run: %v", err)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
			t.Fatalf("an intact run does not verify: %v", err)
		}
		path := filepath.Join(rig.opts.Out, b1+".receipt.json")
		m := fileMap(t, path)
		m["mrw_sha256"] = hashOf([]byte("another build"))
		b, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Run(rig.opts); err == nil || !strings.Contains(err.Error(), b1) {
			t.Errorf("a resume over a receipt with another hash: %v, want it refused, naming %s", err, b1)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), b1) {
			t.Errorf("verify over a receipt with another hash: %v, want it to name %s", err, b1)
		}
	})

	t.Run("an altered kept file fails verify", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "writestate")
		rig.flag(t, "commit")
		rig.opts.KeepStateFile, rig.opts.KeepDiff = rig.kept, true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
			t.Fatalf("an intact run with kept files does not verify: %v", err)
		}
		for _, ext := range []string{".kept", ".diff"} {
			path := filepath.Join(rig.opts.Out, b1+ext)
			orig, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(orig, "tampered\n"...), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), b1) {
				t.Errorf("verify over an altered %s: %v, want it to name %s", ext, err, b1)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), b1) {
				t.Errorf("verify over a removed %s: %v, want it to name %s", ext, err, b1)
			}
			if err := os.WriteFile(path, orig, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
				t.Errorf("verify after %s was put back: %v", ext, err)
			}
		}
	})

	t.Run("the diff is kept", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "commit")
		rig.flag(t, "untracked")
		rig.opts.KeepDiff = true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(rig.opts.Out, b1+".diff"))
		if err != nil {
			t.Fatal(err)
		}
		// The fake committed its change, so only a diff against the base
		// commit holds it; the created file is uncommitted and untracked.
		for _, want := range []string{"-beta", "+gamma", "created.txt", "+hi"} {
			if !strings.Contains(string(got), want) {
				t.Errorf("the kept diff lacks %q:\n%s", want, got)
			}
		}
		if r := readReceipt(t, rig.opts.Out, b1); r.KeptDiffSHA256 != hashOf(got) {
			t.Errorf("kept_diff_sha256 %q, want %s", r.KeptDiffSHA256, hashOf(got))
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["keep_diff"] != true {
			t.Errorf("run.json keep_diff %v, want true", m["keep_diff"])
		}
		off := newPinRig(t, []string{"B"}, 1)
		if err := Run(off.opts); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(off.opts.Out, b1+".diff")); err == nil {
			t.Error("a diff was kept without --keep-diff")
		}
	})

	t.Run("configuration H with the pin runs the pinned bytes in its hook", func(t *testing.T) {
		opts := stage7Options(t, "H")
		data, err := os.ReadFile(opts.Mrw)
		if err != nil {
			t.Fatal(err)
		}
		opts.MrwSHA256 = hashOf(data)
		if err := resolveProfile(&opts); err != nil {
			t.Fatal(err)
		}
		if err := resolvePins(&opts); err != nil {
			t.Fatal(err)
		}
		// Replaced after the pin was taken: the attempt must not follow it.
		if err := os.WriteFile(opts.Mrw, []byte(evilMrw), 0o755); err != nil {
			t.Fatal(err)
		}
		args, _, err := invocation(opts, attempt{ID: "x", Task: opts.Tasks[0], Model: "m", Config: Configs["H"], Repeat: 1}, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(args[slices.Index(args, "--settings")+1])
		if err != nil {
			t.Fatal(err)
		}
		hook := regexp.MustCompile(`'([^']+/bin/mrw)' hook read`).FindSubmatch(b)
		if hook == nil {
			t.Fatalf("settings hold no read hook of bin/mrw: %s", b)
		}
		if fi, err := os.Lstat(string(hook[1])); err != nil || !fi.Mode().IsRegular() {
			t.Fatalf("the hook's mrw %s is not a copy of the bytes: %v %v", hook[1], fi, err)
		}
		if fi, err := os.Lstat(string(hook[1])); err != nil || fi.Mode().Perm() != 0o555 {
			t.Errorf("the staged mrw is %v (%v), want mode 0555", fi, err)
		}
		if got, _ := os.ReadFile(string(hook[1])); hashOf(got) != opts.MrwSHA256 {
			t.Errorf("the read hook runs bytes hashing to %s, want the pinned %s", hashOf(got), opts.MrwSHA256)
		}
	})

	t.Run("a canary with other hashes is refused", func(t *testing.T) {
		rig := newPinRig(t, []string{"A", "B"}, 1)
		instr := filepath.Join(rig.dir, "instructions.txt")
		if err := os.WriteFile(instr, []byte("frozen\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		n := 0
		canary := func(mrw, in, model string) string {
			n++
			p := filepath.Join(rig.dir, fmt.Sprintf("canary-%d.json", n))
			body := fmt.Sprintf(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","model":%q,"mrw_sha256":%q,"mrw_instructions_sha256":%q}`, model, mrw, in)
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			return p
		}
		insSHA := hashOf([]byte("frozen\n"))
		other := hashOf([]byte("another build"))
		run := func(name, canaryFile string, pin, ins bool) error {
			o := rig.opts
			o.Canary, o.Out = canaryFile, filepath.Join(rig.dir, "out-"+name)
			if !pin {
				if err := os.Chmod(rig.live, 0o755); err != nil {
					t.Fatal(err)
				}
				o.MrwSHA256 = ""
			}
			if ins {
				o.MrwInstructions = instr
			}
			return Run(o)
		}
		refused := func(name, why string, err error) {
			t.Helper()
			if err == nil || !strings.Contains(err.Error(), "canary") {
				t.Errorf("%s: %v, want the canary refused", why, err)
			}
		}
		if err := run("same", canary(rig.sha, insSHA, "stub-model"), true, true); err != nil {
			t.Errorf("a canary with this run's hashes: %v", err)
		}
		refused("mrw", "a canary of another mrw", run("mrw", canary(other, insSHA, "stub-model"), true, true))
		refused("ins", "a canary of other instructions", run("ins", canary(rig.sha, other, "stub-model"), true, true))
		// The instruction source counts both ways: a file, or the binary's own.
		refused("fromfile", "a canary made with an instruction file, for a run whose instructions are the binary's", run("fromfile", canary(rig.sha, insSHA, "stub-model"), true, false))
		refused("frombin", "a canary made with the binary's instructions, for a run given a file", run("frombin", canary(rig.sha, "", "stub-model"), true, true))
		refused("model", "a canary of another model", run("model", canary(rig.sha, insSHA, "another-model"), true, true))
		refused("nomodel", "a pinning run on a canary that names no model", run("nomodel", canary(rig.sha, insSHA, ""), true, true))
		noHash := filepath.Join(rig.dir, "canary-nohash.json")
		if err := os.WriteFile(noHash, []byte(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","model":"stub-model"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		refused("old", "a pinning run on a canary that recorded no hash", run("old", noHash, true, false))
		refused("freemodel", "a run that pins nothing, on a canary of another model", run("freemodel", canary(other, insSHA, "another-model"), false, false))
		if err := run("free", canary(other, insSHA, "stub-model"), false, false); err != nil {
			t.Errorf("a run that pins nothing, on a canary of its model and either build: %v", err)
		}
		noModel := filepath.Join(rig.dir, "canary-nomodel.json")
		if err := os.WriteFile(noModel, []byte(`{"pass":true,"cli_version":"9.9.9 (Claude Code)"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		refused("freeold", "a run that pins nothing, on a canary that recorded no model", run("freeold", noModel, false, false))

		two := func(name string, models ...string) error {
			o := rig.opts
			o.Models = []string{"stub-model", "second-model"}
			o.Canary, o.Out, o.MrwInstructions = canary(rig.sha, insSHA, models[0]), filepath.Join(rig.dir, "out-"+name), instr
			for _, m := range models[1:] {
				o.MoreCanaries = append(o.MoreCanaries, canary(rig.sha, insSHA, m))
			}
			return Run(o)
		}
		// A canary authorises the model it ran: two models need a canary of each (the review of #751).
		refused("two", "a pinned run of two models, on a canary of one of them", two("two", "second-model"))
		if err := two("twoboth", "stub-model", "second-model"); err != nil {
			t.Errorf("a pinned run of two models, on a canary of each: %v", err)
		}
		refused("twoother", "a pinned run of two models, on a canary of neither", two("twoother", "third-model"))
		// RunCanary records what it ran.
		co := rig.opts
		co.Out, co.MrwInstructions = filepath.Join(rig.dir, "canary-out"), instr
		got, err := RunCanary(co, co.Tasks[0], "")
		if err != nil || !got.Pass {
			t.Fatalf("the canary: %+v %v", got, err)
		}
		if m := fileMap(t, filepath.Join(co.Out, "canary.json")); m["mrw_sha256"] != rig.sha || m["mrw_instructions_sha256"] != insSHA {
			t.Errorf("canary.json mrw_sha256 %v, mrw_instructions_sha256 %v, want %s and %s", m["mrw_sha256"], m["mrw_instructions_sha256"], rig.sha, insSHA)
		}
		o := rig.opts
		o.Canary, o.Out, o.MrwInstructions = filepath.Join(co.Out, "canary.json"), filepath.Join(rig.dir, "out-recorded"), instr
		if err := Run(o); err != nil {
			t.Errorf("a run on the canary RunCanary wrote for the same pins: %v", err)
		}
	})

	t.Run("a state home that is a link keeps nothing", func(t *testing.T) {
		for _, c := range []struct{ flag, want string }{{"linkstate", "state home"}, {"linkfile", "regular"}} {
			rig := newPinRig(t, []string{"B"}, 1)
			rig.flag(t, "writestate")
			rig.flag(t, c.flag)
			rig.opts.KeepStateFile = rig.kept
			if err := Run(rig.opts); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(rig.opts.Out, b1+".kept")); err == nil {
				t.Errorf("%s: %q was kept through the link", c.flag, got)
			}
			if r := readReceipt(t, rig.opts.Out, b1); r.KeptStateFileSHA256 != "" || !strings.Contains(r.KeptNote, c.want) {
				t.Errorf("%s: the receipt names %q and the note %q, want nothing kept and a note naming %q", c.flag, r.KeptStateFileSHA256, r.KeptNote, c.want)
			}
			if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
				t.Errorf("%s: a refused keep, noted, does not verify: %v", c.flag, err)
			}
		}
	})

	t.Run("an attempt that rewrote its mrw is recorded", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "tamper")
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		if r := readReceipt(t, rig.opts.Out, b1); r.MrwSHA256After != hashOf([]byte(evilMrw)) {
			t.Errorf("mrw_sha256_after %q, want the hash of what bin/mrw became", r.MrwSHA256After)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), b1) || !strings.Contains(err.Error(), "after the attempt") {
			t.Errorf("verify over an attempt that rewrote its mrw: %v, want it to name %s", err, b1)
		}
	})

	t.Run("resume and verify want the kept files their receipts name", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "commit")
		rig.opts.KeepDiff = true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(rig.opts.Out, b1+".diff")
		orig, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(orig, "tampered\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Run(rig.opts); err == nil || !strings.Contains(err.Error(), b1) {
			t.Errorf("a resume over an altered kept diff: %v, want it refused, naming %s", err, b1)
		}
		if err := os.WriteFile(path, orig, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Run(rig.opts); err != nil {
			t.Errorf("a resume over an intact kept diff: %v", err)
		}
		// A receipt that names no diff, and says why not, is not an attempt that kept one.
		rp := filepath.Join(rig.opts.Out, b1+".receipt.json")
		m := fileMap(t, rp)
		delete(m, "kept_diff_sha256")
		b, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(rp, b, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), b1) || !strings.Contains(err.Error(), "no kept diff") {
			t.Errorf("verify over a graded attempt with no diff and no reason: %v, want it to name %s", err, b1)
		}
		if err := Run(rig.opts); err == nil || !strings.Contains(err.Error(), b1) {
			t.Errorf("a resume over a graded attempt with no diff and no reason: %v, want it refused, naming %s", err, b1)
		}
	})

	t.Run("the diff is taken before grading and survives a timeout", func(t *testing.T) {
		plain := newPinRig(t, []string{"B"}, 1)
		plain.flag(t, "commit")
		plain.flag(t, "untracked")
		if err := Run(plain.opts); err != nil {
			t.Fatal(err)
		}
		kept := newPinRig(t, []string{"B"}, 1)
		kept.flag(t, "commit")
		kept.flag(t, "untracked")
		kept.opts.KeepDiff = true
		if err := Run(kept.opts); err != nil {
			t.Fatal(err)
		}
		r0, r1 := readReceipt(t, plain.opts.Out, b1), readReceipt(t, kept.opts.Out, b1)
		if !reflect.DeepEqual(r0.Oracle, r1.Oracle) || !reflect.DeepEqual(r0.Diff, r1.Diff) || len(r1.Oracle.Outside) == 0 {
			t.Errorf("keeping the diff changed the grading: %+v %+v against %+v %+v (want the created file outside in both)", r1.Oracle, r1.Diff, r0.Oracle, r0.Diff)
		}
		hang := newPinRig(t, []string{"B"}, 1)
		hang.flag(t, "commit")
		hang.flag(t, "hang")
		hang.opts.KeepDiff, hang.opts.Budget = true, 3*time.Second
		if err := Run(hang.opts); err != nil {
			t.Fatal(err)
		}
		r := readReceipt(t, hang.opts.Out, b1)
		got, err := os.ReadFile(filepath.Join(hang.opts.Out, b1+".diff"))
		if !r.TimedOut || err != nil || !strings.Contains(string(got), "+gamma") || r.KeptDiffSHA256 != hashOf(got) {
			t.Errorf("a timed-out attempt: timed out %v, diff %q (%v), want its diff kept all the same", r.TimedOut, got, err)
		}
	})

	t.Run("what is kept is bounded", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "writestate")
		rig.flag(t, "big")
		rig.flag(t, "bigdiff")
		rig.opts.KeepStateFile, rig.opts.KeepDiff = rig.kept, true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		for _, ext := range []string{".kept", ".diff"} {
			if fi, err := os.Stat(filepath.Join(rig.opts.Out, b1+ext)); err == nil {
				t.Errorf("a %s file of %d bytes was kept over the bound", ext, fi.Size())
			}
		}
		r := readReceipt(t, rig.opts.Out, b1)
		if r.KeptStateFileSHA256 != "" || r.KeptDiffSHA256 != "" || strings.Count(r.KeptNote, "bound") < 2 {
			t.Errorf("the receipt names %q and %q with note %q, want nothing kept and the bound named twice", r.KeptStateFileSHA256, r.KeptDiffSHA256, r.KeptNote)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
			t.Errorf("an attempt that kept nothing and said why does not verify: %v", err)
		}
	})

	t.Run("the stop counts the last attempt and compares in cents", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 2)
		rig.costs(t, "0.10", "0.70")
		rig.opts.MaxSpend = 0.80
		if err := Run(rig.opts); !errors.Is(err, ErrSpendStop) {
			t.Errorf("0.10 and 0.70 against a figure of 0.80, the last attempt reaching it: %v, want ErrSpendStop", err)
		}
		if n := len(receipts(t, rig.opts.Out)); n != 2 {
			t.Errorf("%d receipts, want both attempts", n)
		}
		under := newPinRig(t, []string{"B"}, 2)
		under.costs(t, "0.10", "0.60")
		under.opts.MaxSpend = 0.80
		if err := Run(under.opts); err != nil {
			t.Errorf("a run that ends under its figure: %v", err)
		}
	})

	t.Run("a path the agent added by force survives the private index", func(t *testing.T) {
		for _, commit := range []bool{false, true} {
			rig := newPinRig(t, []string{"B"}, 1)
			rig.flag(t, "forceadd")
			if commit {
				rig.flag(t, "commit")
			}
			rig.opts.KeepDiff = true
			if err := Run(rig.opts); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(rig.opts.Out, b1+".diff"))
			if err != nil || !strings.Contains(string(got), "forced.log") {
				t.Errorf("committed %v: the kept diff (%v) lacks forced.log, an ignored file the agent added by force:\n%s", commit, err, got)
			}
		}
	})

	t.Run("a receipt needs a note about the diff, not about the state file", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "commit")
		rig.opts.KeepDiff = true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		rp := filepath.Join(rig.opts.Out, b1+".receipt.json")
		m := fileMap(t, rp)
		delete(m, "kept_diff_sha256")
		m["kept_note"] = "state: something else was not kept"
		b, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(rp, b, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(rig.opts.Out, b1+".diff")); err != nil {
			t.Fatal(err)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), b1) {
			t.Errorf("verify over a missing diff with a note about the state file: %v, want it to name %s", err, b1)
		}
		m["kept_note"] = "state: x; diff: not kept: for a reason"
		b, _ = json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(rp, b, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
			t.Errorf("verify over a missing diff with a note about the diff: %v", err)
		}
	})

	t.Run("a capture that fails is a setup exclusion with its own note", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "badindex")
		rig.opts.KeepDiff = true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		r := readReceipt(t, rig.opts.Out, b1)
		if r.Exclusion != ExclusionSetup || !hasNote(r.KeptNote, "diff:") || r.KeptDiffSHA256 != "" {
			t.Errorf("exclusion %q, note %q, want a setup exclusion and a diff note", r.Exclusion, r.KeptNote)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
			t.Errorf("a run whose capture failed, noted, does not verify: %v", err)
		}
	})

	t.Run("a canary whose mrw was rewritten does not pass", func(t *testing.T) {
		rig := newPinRig(t, []string{"A", "B"}, 1)
		rig.flag(t, "tamper")
		co := rig.opts
		co.Out = filepath.Join(rig.dir, "canary-out")
		got, err := RunCanary(co, co.Tasks[0], "")
		if err != nil {
			t.Fatal(err)
		}
		if got.Pass || !strings.Contains(strings.Join(got.Problems["B"], " "), "bin/mrw") {
			t.Errorf("a canary whose attempt rewrote bin/mrw: pass %v, problems %v", got.Pass, got.Problems)
		}
	})

	t.Run("a grader that writes into the tree is not in the kept diff", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		dir := filepath.Join(filepath.Dir(rig.opts.Out), "coupled", "fix-coupled")
		for name, body := range map[string]string{
			"task.json":        `{"id":"fix-coupled","shape":"fixture","repo":"fixture","instruction":"Change active to Active in p/p.go, and keep the tests passing.","require":[{"file":"p/p.go","match":"\"Active\""}],"forbid":[{"file":"p/p.go","match":"\"active\""}]}`,
			"base/go.mod":      "module fix\n\ngo 1.21\n",
			"base/p/p.go":      "package p\n\nfunc Status() string { return \"active\" }\n",
			"base/p/p_test.go": "package p\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\nfunc TestStatus(t *testing.T) {\n\t_ = os.WriteFile(\"../grader-wrote.txt\", []byte(\"x\"), 0o644)\n\t_ = os.WriteFile(os.Getenv(\"PINTEST_MARK\"), nil, 0o644)\n\tif Status() == \"\" {\n\t\tt.Fatal(\"empty\")\n\t}\n}\n",
			"reference.patch":  "diff --git a/p/p.go b/p/p.go\n--- a/p/p.go\n+++ b/p/p.go\n@@ -1,3 +1,3 @@\n package p\n \n-func Status() string { return \"active\" }\n+func Status() string { return \"Active\" }\n",
		} {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		task, err := LoadTask(dir)
		if err != nil {
			t.Fatal(err)
		}
		// A fixture task cannot name a package; the oracle reads these two fields of the struct.
		task.TestPackage, task.AssertingTests = "p", []string{"TestStatus"}
		mark := filepath.Join(rig.dir, "grader-ran")
		t.Setenv("PINTEST_MARK", mark)
		if err := os.WriteFile(filepath.Join(rig.dir, "patch"), []byte(filepath.Join(dir, "reference.patch")), 0o644); err != nil {
			t.Fatal(err)
		}
		rig.opts.Tasks, rig.opts.KeepDiff = []Task{task}, true
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(mark); err != nil {
			t.Fatalf("the grader did not run its test, so the row proves nothing: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(rig.opts.Out, "fix-coupled.B.stub-model.1.diff"))
		if err != nil || !strings.Contains(string(got), `+func Status() string { return "Active" }`) {
			t.Fatalf("the kept diff (%v) lacks the agent's change:\n%s", err, got)
		}
		if strings.Contains(string(got), "grader-wrote") {
			t.Errorf("the kept diff holds what the grader wrote into the tree:\n%s", got)
		}
	})

	t.Run("a run without the flags is unchanged", func(t *testing.T) {
		opts, _ := runOptions(t)
		if err := Run(opts); err != nil {
			t.Fatal(err)
		}
		// What the same run held before ADR-206: taken from the tree at 755ab96f.
		golden := map[string]string{
			"run.json":                            `{"cli_version":"9.9.9 (Claude Code)","configs":["A","B"],"containment":"bypass","models":["stub-model"],"mrw_version":"mrw stub","repeats":1,"seed":156,"stage":"test","tasks":["fix-one"]}`,
			"fix-one.A.stub-model.1.receipt.json": `{"attempt":"fix-one.A.stub-model.1","cli_version":"9.9.9 (Claude Code)","config":"A","diff":{"deletions":1,"files":1,"insertions":1},"exit_code":0,"init_mcp_servers":null,"init_tools":["Bash","Read","Edit","Write","Glob","Grep"],"is_error":false,"model":"stub-model","num_turns":2,"oracle":{"solved":true},"order":1,"repeat":1,"shape":"fixture","silent_mis_edit":false,"source":"own","stage":"test","task":"fix-one","tool_calls":1,"total_cost_usd":0.25,"usage":{"cache_creation_input_tokens":40,"cache_read_input_tokens":300,"input_tokens":100,"output_tokens":20}}`,
			"fix-one.B.stub-model.1.receipt.json": `{"attempt":"fix-one.B.stub-model.1","cli_version":"9.9.9 (Claude Code)","config":"B","diff":{"deletions":1,"files":1,"insertions":1},"exit_code":0,"init_mcp_servers":null,"init_tools":["Bash","Glob"],"is_error":false,"model":"stub-model","mrw_version":"mrw stub","num_turns":2,"oracle":{"solved":true},"order":0,"repeat":1,"shape":"fixture","silent_mis_edit":false,"slipped_writes":["notes.txt"],"source":"own","stage":"test","task":"fix-one","tool_calls":1,"total_cost_usd":0.25,"usage":{"cache_creation_input_tokens":40,"cache_read_input_tokens":300,"input_tokens":100,"output_tokens":20}}`,
		}
		for name, want := range golden {
			got := fileMap(t, filepath.Join(opts.Out, name))
			var wantMap map[string]any
			if err := json.Unmarshal([]byte(want), &wantMap); err != nil {
				t.Fatal(err)
			}
			// ADR-206 adds two keys to every run, and compares nothing else of them;
			// ADR-211 adds the transcript's hash to every receipt that has a transcript.
			extra := "bench_revision"
			if name != "run.json" {
				extra = "init_model"
			}
			if got[extra] == nil {
				t.Errorf("%s lacks %s", name, extra)
			}
			if name != "run.json" && got["init_model"] != "stub-model" {
				t.Errorf("%s init_model %v, want the init line's stub-model", name, got["init_model"])
			}
			if name != "run.json" && got["transcript_sha256"] == nil {
				t.Errorf("%s lacks transcript_sha256", name)
			}
			delete(got, extra)
			delete(got, "transcript_sha256")
			delete(got, "started")
			delete(got, "wall_ms")

			delete(got, "wall_ms")
			if !reflect.DeepEqual(got, wantMap) {
				gb, _ := json.Marshal(got)
				t.Errorf("%s changed without a flag:\n got %s\nwant %s", name, gb, want)
			}
		}
	})
}

// ADR-211 T1: the instrument's own tests. The rig's fake agent also reports the
// init model a file names, makes the scratch file a run keeps, and signals bench.

func (r *pinRig) initModels(t *testing.T, lines ...string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, "initmodel"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r *pinRig) canaryOpts() RunOptions {
	co := r.opts
	co.Out = filepath.Join(r.dir, "canary-out")
	return co
}

func canaryFile(t *testing.T, rig *pinRig) map[string]any {
	t.Helper()
	return fileMap(t, filepath.Join(rig.canaryOpts().Out, "canary.json"))
}

func attemptsRun(rig *pinRig) int {
	b, _ := os.ReadFile(filepath.Join(rig.dir, "count"))
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

// Decision 8: every run takes only a canary of its own model and its own profile.
func TestARunTakesOnlyACanaryOfItsOwnModel(t *testing.T) {
	for _, pinned := range []bool{true, false} {
		name := map[bool]string{true: "pinned", false: "pinning nothing"}[pinned]
		t.Run(name, func(t *testing.T) {
			rig := newPinRig(t, []string{"B"}, 1)
			if !pinned {
				rig.unpin(t)
			}
			canary := func(model, profile string) {
				t.Helper()
				hash := ""
				if pinned {
					hash = rig.sha
				}
				body := fmt.Sprintf(`{"pass":true,"cli_version":"9.9.9 (Claude Code)","model":%q,"profile":%q,"mrw_sha256":%q}`, model, profile, hash)
				if err := os.WriteFile(rig.opts.Canary, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, c := range []struct{ why, model, profile string }{
				{"another model's canary", "another-model", ""},
				{"a canary that names no model", "", ""},
				{"another profile's canary", "stub-model", ProfileStage5},
			} {
				canary(c.model, c.profile)
				o := rig.opts
				o.Out = filepath.Join(rig.dir, "out-"+strings.ReplaceAll(c.why, " ", "-"))
				if err := Run(o); err == nil || !strings.Contains(err.Error(), "canary") {
					t.Errorf("%s: %v, want the canary refused", c.why, err)
				}
				if attemptsRun(rig) != 0 {
					t.Errorf("%s: an attempt ran", c.why)
				}
			}
			canary("stub-model", "")
			if err := Run(rig.opts); err != nil {
				t.Errorf("a canary of this run's model and profile: %v", err)
			}
		})
	}
}

// Decision 8: the canary records the model each configuration's init line named.
func TestACanaryRecordsTheModelEachConfigurationReported(t *testing.T) {
	rig := newPinRig(t, []string{"A", "B"}, 1)
	rig.initModels(t, "reported-model", "reported-model")
	got, err := RunCanary(rig.canaryOpts(), rig.opts.Tasks[0], "")
	if err != nil || !got.Pass {
		t.Fatalf("a canary whose configurations report one model: %+v %v", got, err)
	}
	want := map[string]string{"A": "reported-model", "B": "reported-model"}
	if !reflect.DeepEqual(got.InitModels, want) {
		t.Errorf("init_models %v, want %v (the requested model is stub-model)", got.InitModels, want)
	}
	if m := canaryFile(t, rig); m["model"] != "stub-model" || fmt.Sprint(m["init_models"]) != fmt.Sprint(map[string]any{"A": "reported-model", "B": "reported-model"}) {
		t.Errorf("canary.json model %v init_models %v, want the requested alias beside what was observed", m["model"], m["init_models"])
	}

	for name, lines := range map[string][]string{
		"a configuration whose init line names no model": {"reported-model", ""},
		"configurations that name different models":      {"reported-model", "another-reported-model"},
	} {
		bad := newPinRig(t, []string{"A", "B"}, 1)
		bad.initModels(t, lines...)
		got, err := RunCanary(bad.canaryOpts(), bad.opts.Tasks[0], "")
		if err != nil || got.Pass || !strings.Contains(strings.Join(slices.Concat(got.Problems["A"], got.Problems["B"], got.Problems["init_models"]), " "), "init") {
			t.Errorf("%s: pass %v problems %v (%v), want the canary failed and the init model named", name, got.Pass, got.Problems, err)
		}
	}
}

// Decision 9: a canary stops on the spend its attempts recorded, before each
// configuration's attempt and after the last, and says so in canary.json.
func TestACanaryStopsOnItsSpendAtEitherBoundary(t *testing.T) {
	for _, c := range []struct {
		why   string
		costs []string
		max   float64
		ran   int
	}{
		{"A reaching the figure ends it before B starts", []string{"0.60", "0.10"}, 0.50, 1},
		{"B crossing it on the last attempt", []string{"0.20", "0.40"}, 0.50, 2},
		{"a null cost", []string{"none", "0.10"}, 0.50, 1},
		{"the figure compared in millionths", []string{"0.10", "0.70"}, 0.80, 2},
	} {
		rig := newPinRig(t, []string{"A", "B"}, 1)
		rig.costs(t, c.costs...)
		co := rig.canaryOpts()
		co.MaxSpend = c.max
		got, err := RunCanary(co, co.Tasks[0], "")
		if !errors.Is(err, ErrSpendStop) || ExitCode(err) != 3 {
			t.Errorf("%s: %v, want ErrSpendStop and exit 3", c.why, err)
		}
		if n := attemptsRun(rig); n != c.ran {
			t.Errorf("%s: %d attempts ran, want %d", c.why, n, c.ran)
		}
		m := canaryFile(t, rig)
		if got.Pass || m["pass"] != false || m["max_spend_usd"] != c.max || len(got.Problems["spend"]) == 0 || len(got.Costs) != c.ran {
			t.Errorf("%s: pass %v, canary.json %v, problems %v, costs %v, want pass false, the figure %v, a spend problem and %d costs", c.why, got.Pass, m, got.Problems, got.Costs, c.max, c.ran)
		}
	}
	under := newPinRig(t, []string{"A", "B"}, 1)
	under.costs(t, "0.10", "0.10")
	co := under.canaryOpts()
	co.MaxSpend = 0.50
	got, err := RunCanary(co, co.Tasks[0], "")
	if err != nil || !got.Pass || canaryFile(t, under)["max_spend_usd"] != 0.5 {
		t.Errorf("a canary under its figure: pass %v, %v; want it to pass and record the applied figure", got.Pass, err)
	}
	// An interrupted canary writes canary.json too, under problems["interrupted"].
	sig := newPinRig(t, []string{"A", "B"}, 1)
	sig.flag(t, "sigint")
	got, err = RunCanary(sig.canaryOpts(), sig.opts.Tasks[0], "")
	m := canaryFile(t, sig)
	if err == nil || m["pass"] != false || len(got.Problems["interrupted"]) == 0 {
		t.Errorf("an interrupted canary: %v, canary.json %v, problems %v, want it written pass false under interrupted", err, m, got.Problems)
	}
}

// Decision 7: the transcript's hash is taken from the bytes read back once the
// transcript is closed, and a setup failure before it opens leaves the key out.
func TestAReceiptRecordsItsTranscriptHashAtClose(t *testing.T) {
	rig := newPinRig(t, []string{"A", "B"}, 1)
	if err := Run(rig.opts); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"fix-one.A.stub-model.1", "fix-one.B.stub-model.1"} {
		b, err := os.ReadFile(filepath.Join(rig.opts.Out, id+".transcript.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if r := readReceipt(t, rig.opts.Out, id); r.TranscriptSHA256 != hashOf(b) {
			t.Errorf("%s: transcript_sha256 %q, want the hash %s of the bytes on disk", id, r.TranscriptSHA256, hashOf(b))
		}
	}
	setup := newPinRig(t, []string{"A"}, 1)
	blocker := filepath.Join(setup.dir, "blocker") // a file: no scratch directory can be made under it
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	setup.opts.Scratch = filepath.Join(blocker, "no-directory-here")
	if err := Run(setup.opts); err != nil {
		t.Fatal(err)
	}
	m := fileMap(t, filepath.Join(setup.opts.Out, "fix-one.A.stub-model.1.receipt.json"))
	if m["exclusion"] != ExclusionSetup || m["transcript_sha256"] != nil {
		t.Errorf("a setup failure before the transcript opened: exclusion %v, transcript_sha256 %v, want setup and no key", m["exclusion"], m["transcript_sha256"])
	}
}

// Decision 7: a directory stopped on its spend, re-invoked with its own figure,
// stops again at once: exit 3 and no new .started.
func TestASpendStoppedRunIsNotResumedWithItsFigure(t *testing.T) {
	rig := newPinRig(t, []string{"A", "B"}, 2)
	rig.opts.MaxSpend = 0.50
	err := Run(rig.opts)
	if !errors.Is(err, ErrSpendStop) || ExitCode(err) != 3 {
		t.Fatalf("the first run: %v, want a spend stop", err)
	}
	started := func() int {
		n := 0
		ents, _ := os.ReadDir(rig.opts.Out)
		for _, e := range ents {
			if strings.HasSuffix(e.Name(), ".started") {
				n++
			}
		}
		return n
	}
	before, ran := started(), attemptsRun(rig)
	if before != 2 || ran != 2 {
		t.Fatalf("the first run started %d attempts and ran %d, want 2 and 2 (0.25 each against 0.50)", before, ran)
	}
	err = Run(rig.opts)
	if !errors.Is(err, ErrSpendStop) || ExitCode(err) != 3 {
		t.Errorf("the resume: %v, want exit 3", err)
	}
	if started() != before || attemptsRun(rig) != ran {
		t.Errorf("the resume started %d attempts and ran %d, want none new", started()-before, attemptsRun(rig)-ran)
	}
}

// Decision 4: the runner keeps one file of the attempt's scratch directory beside
// the receipt, before grading and before the scratch goes.
func TestAScratchFileIsKeptBesideTheReceipt(t *testing.T) {
	const id, name = "fix-one.B.stub-model.1", "stage9-exposed.jsonl"
	kept := func(t *testing.T, flags ...string) (*pinRig, Receipt) {
		t.Helper()
		rig := newPinRig(t, []string{"B"}, 1)
		for _, f := range flags {
			rig.flag(t, f)
		}
		rig.opts.KeepScratchFile = name
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		return rig, readReceipt(t, rig.opts.Out, id)
	}
	t.Run("kept, hashed and verified", func(t *testing.T) {
		rig, r := kept(t, "scratchlog")
		got, err := os.ReadFile(filepath.Join(rig.opts.Out, id+".kept"))
		if err != nil || string(got) != "{\"x\":1}\n" || r.KeptScratchFileSHA256 != hashOf(got) || r.KeptNote != "" {
			t.Fatalf("kept %q (%v), receipt hash %q note %q", got, err, r.KeptScratchFileSHA256, r.KeptNote)
		}
		if m := fileMap(t, filepath.Join(rig.opts.Out, "run.json")); m["keep_scratch_file"] != name {
			t.Errorf("run.json keep_scratch_file %v, want %s", m["keep_scratch_file"], name)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
			t.Errorf("verify: %v", err)
		}
		if err := os.WriteFile(filepath.Join(rig.opts.Out, id+".kept"), append(got, "tampered\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Verify(rig.opts.Out, "test", 1, 1); err == nil || !strings.Contains(err.Error(), id) {
			t.Errorf("verify over a changed kept file: %v, want it to name %s", err, id)
		}
		if err := Run(rig.opts); err == nil || !strings.Contains(err.Error(), id) {
			t.Errorf("a resume over a changed kept file: %v, want it refused, naming %s", err, id)
		}
	})
	t.Run("an attempt that left none keeps none and says nothing", func(t *testing.T) {
		rig, r := kept(t)
		if _, err := os.Stat(filepath.Join(rig.opts.Out, id+".kept")); err == nil || r.KeptScratchFileSHA256 != "" || r.KeptNote != "" {
			t.Errorf("a file kept for an attempt that wrote none: hash %q note %q", r.KeptScratchFileSHA256, r.KeptNote)
		}
	})
	t.Run("a link or a directory is a note", func(t *testing.T) {
		for _, flag := range []string{"scratchlink", "scratchdir"} {
			rig, r := kept(t, flag)
			if _, err := os.Stat(filepath.Join(rig.opts.Out, id+".kept")); err == nil || r.KeptScratchFileSHA256 != "" || !hasNote(r.KeptNote, "scratch:") {
				t.Errorf("%s: kept %q, note %q, want nothing kept and a scratch: note", flag, r.KeptScratchFileSHA256, r.KeptNote)
			}
			if err := Verify(rig.opts.Out, "test", 1, 1); err != nil {
				t.Errorf("%s: a refused keep, noted, does not verify: %v", flag, err)
			}
		}
	})
	t.Run("a name with a separator is a note", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.flag(t, "scratchlog")
		rig.opts.KeepScratchFile = "state/" + name
		if err := Run(rig.opts); err != nil {
			t.Fatal(err)
		}
		if r := readReceipt(t, rig.opts.Out, id); r.KeptScratchFileSHA256 != "" || !hasNote(r.KeptNote, "scratch:") {
			t.Errorf("hash %q note %q, want nothing kept and a scratch: note", r.KeptScratchFileSHA256, r.KeptNote)
		}
	})
	t.Run("kept before grading", func(t *testing.T) {
		rig, r := kept(t, "scratchlog", "badindex") // grading fails on the broken index
		if r.Exclusion != ExclusionSetup || r.KeptScratchFileSHA256 == "" {
			t.Errorf("exclusion %q hash %q, want a setup exclusion that still kept the file", r.Exclusion, r.KeptScratchFileSHA256)
		}
		if _, err := os.Stat(filepath.Join(rig.opts.Out, id+".kept")); err != nil {
			t.Errorf("no kept file beside a receipt that grading ended: %v", err)
		}
	})
	t.Run("refused beside --keep-state-file", func(t *testing.T) {
		rig := newPinRig(t, []string{"B"}, 1)
		rig.opts.KeepScratchFile, rig.opts.KeepStateFile = name, rig.kept
		if err := Run(rig.opts); err == nil || !strings.Contains(err.Error(), "keep-scratch-file") || !strings.Contains(err.Error(), "keep-state-file") {
			t.Errorf("both flags: %v, want a refusal naming both", err)
		}
		if attemptsRun(rig) != 0 {
			t.Error("an attempt ran")
		}
	})
}

// Decision 11 (#710): diff statistics and slipped writes are taken before the
// grader runs, the statistics in an index of their own, so a file the grader
// writes is in neither and the tree's index stays the agent's.
func TestAGraderWrittenFileIsNeitherInTheDiffNorASlippedWrite(t *testing.T) {
	rig := newPinRig(t, []string{"B"}, 1)
	rig.flag(t, "untrackedp")
	dir := filepath.Join(filepath.Dir(rig.opts.Out), "coupled", "fix-coupled")
	for name, body := range map[string]string{
		"task.json":        `{"id":"fix-coupled","shape":"fixture","repo":"fixture","instruction":"Change active to Active in p/p.go, and keep the tests passing.","require":[{"file":"p/p.go","match":"\"Active\""}],"forbid":[{"file":"p/p.go","match":"\"active\""}]}`,
		"base/go.mod":      "module fix\n\ngo 1.21\n",
		"base/p/p.go":      "package p\n\nfunc Status() string { return \"active\" }\n",
		"base/p/p_test.go": "package p\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\nfunc TestStatus(t *testing.T) {\n\t_ = os.WriteFile(\"../grader-wrote.txt\", []byte(\"x\"), 0o644)\n\t_ = os.WriteFile(os.Getenv(\"PINTEST_MARK\"), nil, 0o644)\n\tif Status() == \"\" {\n\t\tt.Fatal(\"empty\")\n\t}\n}\n",
		"reference.patch":  "diff --git a/p/p.go b/p/p.go\n--- a/p/p.go\n+++ b/p/p.go\n@@ -1,3 +1,3 @@\n package p\n \n-func Status() string { return \"active\" }\n+func Status() string { return \"Active\" }\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	task, err := LoadTask(dir)
	if err != nil {
		t.Fatal(err)
	}
	task.TestPackage, task.AssertingTests = "p", []string{"TestStatus"}
	mark := filepath.Join(rig.dir, "grader-ran")
	t.Setenv("PINTEST_MARK", mark)
	if err := os.WriteFile(filepath.Join(rig.dir, "patch"), []byte(filepath.Join(dir, "reference.patch")), 0o644); err != nil {
		t.Fatal(err)
	}
	rig.opts.Tasks = []Task{task}
	if err := Run(rig.opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Fatalf("the grader did not run its test, so the row proves nothing: %v", err)
	}
	r := readReceipt(t, rig.opts.Out, "fix-coupled.B.stub-model.1")
	if r.Diff.Files != 2 {
		t.Errorf("the diff counts %d files, want p/p.go and p/created.txt, the agent's two, and not the file the grader wrote", r.Diff.Files)
	}
	if want := []string{"p/created.txt", "p/p.go"}; !slices.Equal(slices.Sorted(slices.Values(r.SlippedWrites)), want) {
		t.Errorf("slipped_writes %v, want %v: grader-wrote.txt is the grader's", r.SlippedWrites, want)
	}
	if !r.Oracle.Solved {
		t.Errorf("the oracle's verdict %+v: the agent's file inside the graded package is not outside it", r.Oracle)
	}

	// The statistics are taken in an index of their own: the tree's index, which
	// the oracle reads, is the one the agent left, and its new file is still untracked.
	tree, base := stagedTree(t)
	idx := filepath.Join(tree, ".git", "index")
	before, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "created.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := diffStat(t.TempDir(), tree, base)
	if err != nil || d.Files != 1 {
		t.Fatalf("diffStat: %+v %v, want the new file counted", d, err)
	}
	after, _ := os.ReadFile(idx)
	if !bytes.Equal(before, after) {
		t.Error("taking the diff statistics changed the tree's index")
	}
	if out, _ := gitOut(tree, "ls-files", "--others", "--exclude-standard"); !strings.Contains(out, "created.txt") {
		t.Errorf("the agent's file is no longer untracked after the statistics: %q", out)
	}
}

// A diff whose numstat is longer than the capped writer keeps (64 KiB) is counted
// whole, or the run fails: totals cut at the cap would be a silent undercount.
func TestDiffStatCountsAnOutputLongerThanTheOutputCap(t *testing.T) {
	tree, base := stagedTree(t)
	const files = 600
	for i := range files {
		name := fmt.Sprintf("%s%04d.txt", strings.Repeat("n", 130), i)
		if err := os.WriteFile(filepath.Join(tree, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d, err := diffStat(t.TempDir(), tree, base)
	if err != nil || d.Files != files || d.Insertions != files {
		t.Fatalf("diffStat: %+v %v, want %d files and %d insertions, from more than %d bytes of numstat", d, err, files, files, outputCap)
	}
}

// stagedTree is a one-commit git tree and that commit.
func stagedTree(t *testing.T) (tree, base string) {
	t.Helper()
	tree = t.TempDir()
	if err := os.WriteFile(filepath.Join(tree, "f.txt"), []byte("f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=b", "-c", "user.email=b@b", "commit", "-q", "-m", "base"}} {
		if err := git(tree, args...); err != nil {
			t.Fatal(err)
		}
	}
	out, err := gitOut(tree, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return tree, strings.TrimSpace(out)
}
