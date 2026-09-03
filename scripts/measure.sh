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
# Build a FRESH binary unless the caller named one. Reusing ./bin/mrw meant the
# stamp printed below could name a commit while the NUMBERS came from whatever
# binary happened to be lying there: a ./bin/mrw a day old measured as though it
# were HEAD, and nothing said so. contract.sh builds its own for the same reason
# — a shared mutable artifact is the bug, not the sharing.
if [ -n "${MRW:-}" ]; then
  MRW=$(cd "$(dirname "$MRW")" && pwd)/$(basename "$MRW")
else
  MRWDIR=$(mktemp -d)
  trap 'rm -rf "$MRWDIR"' EXIT
  MRW="$MRWDIR/mrw"
  go build -o "$MRW" ./cmd/mrw
fi

rule() { printf '%s\n' "------------------------------------------------------------"; }

# measure <label> <sites-as-mrw-specs...>
# Files are derived from the specs, so both paths always cover the same set.
measure() {
  local label=$1; shift
  local specs=("$@")
  local files=() f whole=0

  for s in "${specs[@]}"; do
    f=${s%%:*}
    # ${files[*]-} , not ${files[*]} : under `set -u` bash 3.2 — still the
    # default /bin/bash on macOS — treats an EMPTY array's expansion as unbound
    # and aborts. files IS empty on the first spec, every time, so this script
    # could not run at all on the platform it is developed on. bash 4.4+ allows
    # the bare form, which is why CI never saw it.
    case " ${files[*]-} " in *" $f "*) ;; *) files+=("$f") ;; esac
  done
  for f in "${files[@]}"; do
    whole=$(( whole + $(wc -c < "$f") ))
  done

  local ranged out
  out=$("$MRW" read "${specs[@]}")
  ranged=$("$MRW" read "${specs[@]}" | wc -c | tr -d ' ')

  # The SECOND baseline, and the one that decides whether the byte claim means
  # anything: Read takes offset/limit, so an agent that already knows the line
  # ranges reads only those lines. That is not hypothetical, it is the
  # documented interface. Sum the raw bytes of exactly the spans mrw served —
  # parsed from mrw's own `==>` and `@@` lines so the two sides cover the same
  # content by construction rather than by assertion.
  local windowed=0 cur="" ln a b
  while IFS= read -r ln; do
    case $ln in
      '==> '*) cur=${ln#'==> '}; cur=${cur%% *} ;;
      '@@ '*)  a=${ln#'@@ '}; b=${a#*-}; a=${a%%-*}
               [ -n "$cur" ] && windowed=$(( windowed + $(sed -n "${a},${b}p" "$cur" | wc -c) )) ;;
    esac
  done <<EOF
$out
EOF

  local nsites=${#specs[@]} nfiles=${#files[@]}
  # Two Read strategies, two call counts. Reading files WHOLE needs no search —
  # the file reveals the site. Reading WINDOWS presupposes knowing where the
  # window is, which costs a search first. mrw needs neither: the specs here are
  # regexes, so the finding happens inside the read.
  local wholecalls=$(( nfiles + nsites ))       # M Reads + one Edit per site
  local windowcalls=$(( 1 + nfiles + nsites ))  # + the search that windowing needs
  local mrwcalls=2                              # one read, one write

  # Report a loss as a loss. "0.8x less input" is a sentence that hides which
  # way the comparison went.
  ratio() {
    if [ "$2" -le "$1" ]; then echo "$(echo "scale=1; $1 / $2" | bc)x LESS"
    else echo "$(echo "scale=1; $2 / $1" | bc)x MORE"; fi
  }

  rule
  printf '%s\n' "$label"
  printf '  %d site(s) across %d file(s)\n\n' "$nsites" "$nfiles"
  printf '  %-38s %10s %10s   %s\n' ""                          "baseline" "mrw" ""
  printf '  %-38s %10s %10s   %s\n' "bytes, vs reading files WHOLE" \
    "$whole" "$ranged" "$(ratio "$whole" "$ranged") input"
  printf '  %-38s %10s %10s   %s\n' "bytes, vs a WINDOWED read" \
    "$windowed" "$ranged" "$(ratio "$windowed" "$ranged") input"
  printf '  %-38s %10s %10s   %s\n' "calls, whole-file (reads+edits)" \
    "$wholecalls" "$mrwcalls" "$(echo "scale=1; $wholecalls / $mrwcalls" | bc)x fewer"
  printf '  %-38s %10s %10s   %s\n' "calls, windowed (search+reads+edits)" \
    "$windowcalls" "$mrwcalls" "$(echo "scale=1; $windowcalls / $mrwcalls" | bc)x fewer"
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

# Shape D: the shape the tool is actually FOR, at the scale it is actually met.
# One site in every Go file in the repository — the codebase-wide rename, the
# added build tag, the changed import. The file list comes from git rather than
# being written out here, so this row grows with the repository instead of
# quietly measuring a subset somebody typed once.
#
# READ THIS ROW FOR THE CALLS, NOT THE BYTES. mrw loses the byte comparison
# against a windowed read here, badly and by construction: each site is one
# line, and mrw adds a header per file and a number per line, so it is paying
# overhead on the smallest possible payload. The calls column is the whole
# point — M reads plus N edits, versus 2, for any N.
mapfile -t GOFILES < <(git ls-files '*.go') 2>/dev/null || {
  # bash 3.2 — still the default /bin/bash on macOS — has no mapfile.
  GOFILES=()
  while IFS= read -r f; do GOFILES+=("$f"); done < <(git ls-files '*.go')
}
DSPECS=()
for f in "${GOFILES[@]}"; do DSPECS+=("$f:/^package /"); done
measure "D. One site in every Go file — the shape mrw is for" "${DSPECS[@]}"

rule
echo "Round trips are the floor: mrw is 2 calls for any N. Bytes depend on how"
echo "much of each file the task needs — measure YOUR shape before quoting a number."
