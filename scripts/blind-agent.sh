#!/usr/bin/env bash
# blind-agent.sh DIR — build one trial of the blind-agent bench (BACKLOG, "From
# ADR-009"): a fresh agent given ONLY `mrw instructions` and a tree it has never
# seen. Writes DIR/tree (the fixture), DIR/prompt.txt (what the agent is told,
# with the instructions of the binary under test pasted verbatim) and
# DIR/answer-key.json. Deterministic: two runs produce identical trees and keys,
# and the prompt differs only in the absolute paths it names.
#
# MRW names the binary under test (default: mrw on PATH). Scoring is
# scripts/blind-score.py. No model is called here.
set -euo pipefail
dir=${1:?usage: blind-agent.sh DIR}
MRW=${MRW:-$(command -v mrw)}
[ -x "$MRW" ] || { echo "blind-agent: no mrw binary (set MRW)" >&2; exit 2; }
[ ! -e "$dir/tree" ] || { echo "blind-agent: $dir/tree exists; use a fresh DIR" >&2; exit 2; }
mkdir -p "$dir"
dir=$(cd "$dir" && pwd)
T="$dir/tree"
mkdir -p "$T/svc/api" "$T/svc/store" "$T/vendor/lib" "$T/build" "$T/docs"
printf 'build/\n' > "$T/.gitignore"
cat > "$T/svc/api/users.go" <<'EOF'
package api

// HandleUsers lists users.
func HandleUsers() string {
	// TODO: paginate
	return "users"
}

func helper() int { return 1 }

// HandleUserByID fetches one user.
func HandleUserByID(id int) string {
	// TODO: validate id
	return "user"
}
EOF
cat > "$T/svc/api/orders.go" <<'EOF'
package api

// HandleOrders lists orders.
func HandleOrders() string {
	return "orders"
}
EOF
cat > "$T/svc/store/db.go" <<'EOF'
package store

type DB struct{}

func (d *DB) Get(key string) string {
	// TODO: cache
	return key
}

func (d *DB) Put(key, value string) {
	_ = key
	_ = value
}

func (d *DB) Close() error { return nil }
EOF
printf 'package lib\n\nfunc HandleVendored() {}\n' > "$T/vendor/lib/handle.go"
printf 'package build\n\nfunc HandleGenerated() {}\n' > "$T/build/gen.go"
printf 'line one\nline two\nline three\nline four\nline five\nline six\nline seven\nline eight\n' > "$T/docs/notes.txt"
printf 'status: draft\nowner: M\nstatus: final\n' > "$T/docs/meta.yaml"
(cd "$T" && git init -q .)

cat > "$dir/answer-key.json" <<'EOF'
{
  "t1": [["svc/api/orders.go", 4, "HandleOrders"], ["svc/api/users.go", 4, "HandleUsers"], ["svc/api/users.go", 12, "HandleUserByID"]],
  "t2": ["line six", "line seven", "line eight"],
  "t3": ["line one", "line two", "line three"],
  "t4": [["svc/api/users.go", 5], ["svc/api/users.go", 13], ["svc/store/db.go", 6]],
  "t5": [5, 8],
  "t6": "svc/store/db.go:1-3,10-13",
  "t7": "func (d *DB) Close() error { return nil }",
  "t8": {"ambiguous_exit": 1, "final": "status: draft\nowner: M\nstatus: done\n"},
  "t9": {"exit": 2, "notes": "line one\nline two\nline three\nline four\nline five\nline six\nline seven\nline eight\n"}
}
EOF

{
cat <<EOF
You are testing a command-line tool called mrw, using ONLY its instructions below. Run everything through the Bash tool.

Rules:
- The binary is MRW=$MRW . Call it by that full path.
- The project you work on is the directory F=$T . Point mrw at it the way the instructions say.
- Do NOT use grep, rg, find, ls, cat, sed, awk, head, tail, or the Read/Grep/Glob/Edit/Write tools, and do not read any other directory. Use only \$MRW (printf and echo to build a plan are fine). Do not run --help on anything. The instructions below are all you get.
- To build a write plan, pipe printf or echo into \$MRW write - . Do NOT use cat for anything, not even to write a plan file (cat > file <<EOF counts as cat).
- Every command you run is recorded. One banned command voids the whole run, including one used only to print, save or format your answer. Give your answer as text.
- Try to do each task in as few mrw calls as the instructions allow.

Instructions (this is the tool's own \`mrw instructions\` output, verbatim):
-----
EOF
"$MRW" instructions
cat <<'EOF'
-----

Tasks. You do not know the file names in the project except where a task gives one.
1. Find every Go function whose name starts with `Handle`, anywhere in the project, but NOT in vendored code (`vendor/`) and NOT in generated code (`build/`). Report each as file, line and name.
2. Show `docs/notes.txt` from line 6 to the end of the file.
3. Show the first 3 lines of `docs/notes.txt` using the "from the start" form.
4. For every `TODO` comment anywhere in the project, show it together with the one line after it. Report each TODO's file and line.
5. Show the whole `Get` method of `svc/store/db.go`, from its `func` line to its closing brace, using a pattern address. Report its first and last line numbers.
6. Show lines 1-3 AND lines 10-13 of `svc/store/db.go` in a single spec. Report the spec you used.
7. Show only the last line of `svc/store/db.go`. Report its text.
8. In `docs/meta.yaml`, first try a write plan whose address is `/status:/` replacing it with `status: x`, and report its exit code. Then change the line `status: final` to `status: done` with a write plan that addresses the line by a pattern, and read the file to confirm.
9. Try a write plan that deletes lines of `docs/notes.txt` using the "from the start" form (`-2`). Report the exit code.

Output: end your reply with ONE fenced ```json block, exactly this shape, and nothing after it:
{"t1": [["file", line, "Name"], ...], "t2": ["...", ...], "t3": ["...", ...], "t4": [["file", line], ...],
 "t5": [first, last], "t6": "the spec", "t7": "the text", "t8": {"ambiguous_exit": N}, "t9": {"exit": N}}
File paths are relative to the project directory. Before the JSON, list every place the instructions were unclear or wrong, in at most 150 words.
EOF
} > "$dir/prompt.txt"
echo "blind-agent: trial built in $dir (binary $("$MRW" version))"
