#!/usr/bin/env bash
# One run of the ADR-218 pilot (README.md): reset the trusted folder, start one
# Code Mode session in it, send the task's prompt, wait, take the digest.
#   run.sh ARM TASK RUN PROJ OUT MRWBIN
# ARM is switched or control; PROJ the trusted scratch folder; OUT a directory
# for this run's log, screen and state; MRWBIN the directory holding the build.
set -u
arm=$1 task=$2 run=$3 proj=$4 out=$5 bin=$6
here=$(cd "$(dirname "$0")" && pwd)
name="p${arm:0:1}${task}${run}"
claude=${CLAUDE_BIN:-$HOME/.local/bin/claude}
# shellcheck disable=SC2016 # the backticks are README.md's own, matched literally
prompt=$(grep -E "^${task}\. \`" "$here/README.md" | sed -E 's/^[0-9]\. `(.*)`$/\1/')
# PILOT_PROBE_PROMPT replaces the task's prompt for a check of the runner itself; such a run is not counted.
prompt=${PILOT_PROBE_PROMPT:-$prompt}
[ -n "$prompt" ] || { echo "no prompt for task $task"; exit 2; }

rm -rf "$out" "$proj/pkg" && mkdir -p "$out" "$proj/.claude"
python3 -I "$here/fixture.py" make "$proj"
(cd "$proj" && git add -A && git commit -qm "reset" --allow-empty) > /dev/null 2>&1
env=""
[ "$arm" = switched ] && env='"MRW_SHOWN": "off", '
hook="python3 -I $here/hooklog.py"
cat > "$proj/.claude/settings.json" <<JSON
{
  "env": {${env}"PATH": "$bin:$PATH"},
  "permissions": {"allow": ["Bash(mrw:*)"]},
  "hooks": {
    "PreToolUse": [{"matcher": "", "hooks": [{"type": "command", "command": "$hook"}]}],
    "PostToolUse": [{"matcher": "", "hooks": [{"type": "command", "command": "$hook"}]}]
  }
}
JSON

hc run --bg --name "$name" --cwd "$proj" --no-notify -- env "PATH=$bin:$PATH" "CM_HOOK_LOG=$out/hooks.jsonl" \
  CLAUDE_CODE_EXPERIMENTAL_CODE_MODE=1 "XDG_STATE_HOME=$out/state" "$claude" --model claude-fable-5-2 > "$out/start.json" 2>&1
hc wait "$name" --timeout 60 > "$out/wait0.json" 2>&1
started=$(date +%s)
hc send "$name" "$prompt" > /dev/null 2>&1
hc wait "$name" --timeout 300 > "$out/wait1.json" 2>&1
secs=$(( $(date +%s) - started ))
hc read "$name" > "$out/screen.txt" 2>&1
hc kill "$name" > /dev/null 2>&1
hc rm "$name" > /dev/null 2>&1
got=$(python3 -I "$here/fixture.py" sha "$proj")
# shellcheck disable=SC2016 # the backticks are README.md's own, matched literally
want=$(grep -E "^\| ${task} \| \`" "$here/README.md" | sed -E 's/.*`([0-9a-f]{64})`.*/\1/')
verdict=miss
[ "$got" = "$want" ] && verdict=digest-ok
echo "$arm task=$task run=$run $verdict secs=$secs mrw=$("$bin/mrw" version) $(python3 -I "$here/summarise.py" "$arm" "$out/hooks.jsonl")" | tee "$out/verdict.txt"
