#!/usr/bin/env bash
# Measure what mrw actually saves, on this repository, right now.
#
# The README quotes this script's output. Re-run it rather than trusting the
# number: a measurement nobody can reproduce is a claim.
#
#   ./scripts/measure.sh
#
# METHOD, and its biases stated up front.
#
# The task shape is "touch N sites across M files", which is where the two
# harness primitives cost the most: Edit applies one replacement per call and
# needs the file read first, so N sites in M files is M Reads plus N Edits.
#
# "bytes in" is what lands in the agent's context.
#   * Edit/Write path: the RAW size of each whole file. This UNDERSTATES it —
#     the real Read tool prefixes every line with a number, so the true figure
#     is larger. Understating the side being argued against is deliberate.
#   * mrw path: the ACTUAL bytes mrw prints, headers and line numbers included.
#     Nothing is excluded to flatter it.
#
# What is NOT measured, and must not be claimed: output tokens. The agent emits
# an edit plan for mrw, and old_string/new_string pairs for Edit. Those are the
# same order of magnitude, so this is an INPUT-side and round-trip result, not
# a total-cost one.
#
# Two shapes are measured on purpose. The second is the case mrw barely helps
# with, because a benchmark that only shows the favourable shape is marketing.
set -euo pipefail

cd "$(dirname "$0")/.."
MRW=${MRW:-./bin/mrw}
[ -x "$MRW" ] || go build -o "$MRW" ./cmd/mrw

rule() { printf '%s\n' "------------------------------------------------------------"; }

# measure <label> <sites-as-mrw-specs...>
# Files are derived from the specs, so both paths always cover the same set.
measure() {
  local label=$1; shift
  local specs=("$@")
  local files=() f whole=0

  for s in "${specs[@]}"; do
    f=${s%%:*}
    case " ${files[*]} " in *" $f "*) ;; *) files+=("$f") ;; esac
  done
  for f in "${files[@]}"; do
    whole=$(( whole + $(wc -c < "$f") ))
  done

  local ranged
  ranged=$("$MRW" read "${specs[@]}" | wc -c | tr -d ' ')

  local nsites=${#specs[@]} nfiles=${#files[@]}
  local editcalls=$(( nfiles + nsites ))   # M Reads + one Edit per site
  local mrwcalls=2                         # one read, one write

  rule
  printf '%s\n' "$label"
  printf '  %d site(s) across %d file(s)\n\n' "$nsites" "$nfiles"
  printf '  %-14s %12s %12s\n' ""            "Read+Edit"  "mrw"
  printf '  %-14s %12s %12s\n' "bytes in"    "$whole"     "$ranged"
  printf '  %-14s %12s %12s\n' "tool calls"  "$editcalls" "$mrwcalls"
  # Report a loss as a loss. "0.8x less input" is a sentence that hides which
  # way the comparison went.
  local verdict
  if [ "$ranged" -le "$whole" ]; then
    verdict="$(echo "scale=1; $whole / $ranged" | bc)x LESS input"
  else
    verdict="$(echo "scale=1; $ranged / $whole" | bc)x MORE input"
  fi
  printf '\n  %s, %sx fewer round trips\n' "$verdict" \
    "$(echo "scale=1; $editcalls / $mrwcalls" | bc)"
}

echo "mrw measurement — $(git rev-parse --short HEAD)$(git diff --quiet || echo ' (dirty tree)')"

# Shape A: scattered sites in large files. The case mrw is built for.
measure "A. Scattered sites, large files" \
  'cmd/mrw/main.go:/^const \(/,/^\)/' \
  'internal/apply/apply.go:/READ BEFORE MODIFY/,/^	}$/' \
  'internal/read/read.go:/^func merge/,/^}/' \
  'internal/check/check.go:/^func command/,/^}/'

# Shape B: the same shape as A with half the sites — which is what isolates the
# two axes, since the round-trip saving halves with N while the byte saving does
# not. It was chosen in 2026-08-31 as "most of each small file wanted"; seen.go
# and iter.go outgrew that description as features landed, and the label follows
# what the script now measures rather than what it was picked for.
measure "B. Two sites, mid-sized files" \
  'internal/seen/seen.go:/^func Record/,/^}/' \
  'internal/iter/iter.go:/^func Load/,/^}/'

# Shape C: the case where mrw LOSES, and it belongs in the same output as the
# case where it wins. One site, one file, the whole file needed: the round trips
# are identical, and mrw prints MORE bytes than the file holds because it adds a
# header and a line number per line. Use Read + Edit here. The tool is for
# scattered sites, not for every edit.
measure "C. One site, whole small file needed (mrw loses)" \
  'internal/seen/seen.go'

rule
echo "Round trips are the floor: mrw is 2 calls for any N. Bytes depend on how"
echo "much of each file the task needs — measure YOUR shape before quoting a number."
