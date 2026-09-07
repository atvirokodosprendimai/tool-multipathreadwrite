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
# FIVE shapes are measured, and the ones where mrw LOSES are in on purpose:
# a benchmark that shows only the favourable shape is marketing. Shapes A, B and
# D each carry a losing byte comparison against a windowed read, C loses against
# both baselines, and E shows what the loss actually is once the payload is not
# a single line.
set -euo pipefail

cd "$(dirname "$0")/.."
# Build a FRESH binary unless the caller named one. Reusing ./bin/mrw meant the
# stamp printed below could name a commit while the NUMBERS came from whatever
# binary happened to be lying there: a ./bin/mrw a day old measured as though it
# were HEAD, and nothing said so. contract.sh builds its own for the same reason
# — a shared mutable artifact is the bug, not the sharing.
# ⚠ ONE DISPOSABLE AREA, ONE TRAP. Everything this script creates lives under
# $SCRATCH — the binary it builds, shape E's fixture, and mrw's own per-root
# STATE. That last one is why XDG_STATE_HOME is pinned: mrw keeps a directory
# per root it has ever seen, outside the tree by ADR-004, and shape E hands it a
# fresh root on every run. Unpinned, each run left an orphan behind for ever.
# Measured 2026-09-07 on this machine: 22,613 such directories, 240 MB, all from
# one day of running the scripts. Found by the Codex review of #136.
SCRATCH=$(mktemp -d)
cleanup() { rm -rf "$SCRATCH"; }
# ⚠ A CAUGHT SIGNAL DOES NOT STOP BASH. `trap cleanup EXIT INT TERM` runs the
# handler and then RESUMES the script — with its binary, its fixture and its
# state directory already deleted. With an externally supplied $MRW it can even
# run to completion and exit 0, publishing a measurement that was interrupted.
# So the signal handlers EXIT, and the EXIT trap is the single owner of the
# removal. Codex, third review of #136.
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
export XDG_STATE_HOME="$SCRATCH/state"
mkdir -p "$XDG_STATE_HOME"
if [ -n "${MRW:-}" ]; then
  MRW=$(cd "$(dirname "$MRW")" && pwd)/$(basename "$MRW")
  MRW_EXTERNAL=1
else
  MRW="$SCRATCH/mrw"
  go build -o "$MRW" ./cmd/mrw
fi

rule() { printf '%s\n' "------------------------------------------------------------"; }

# round1 divides and rounds to one decimal.
#
# ⚠ ROUND, NEVER TRUNCATE, and the distinction is not cosmetic. This was
# `bc scale=1`, which truncates: 1.29 printed as 1.2 and 1.53 as 1.5. On a LOSS
# ratio that understates mrw's OWN weakness, which is the one direction this
# repository may not err in — and it is how the numbers on a page came to
# disagree with the same numbers recomputed by hand, with the script looking
# like the authority. Found 2026-09-07 by a reader who recomputed a published
# ratio and got 1.29 where the script said 1.2. awk's printf rounds.
#
# It also removes `bc`, which was this script's only user of it and is absent
# from Alpine and most slim images. awk is already what contract.sh uses.
# ⚠ AND LC_ALL=C, WHICH IS NOT A TIDINESS FLAG. awk's printf honours the
# locale's decimal separator, so a correctly ROUNDING awk prints "1,3" under
# lt_LT, de_DE or fr_FR. That broke the self-check below into a false failure
# blaming truncation that was not happening — the gate added to catch a silent
# defect became a loud wrong one, on the maintainer's own locale. It also
# matters for the output itself: these figures are quoted verbatim in README.md
# and copied onto a web page, so the separator must not depend on who ran the
# script. Found by the Codex review of #136.
round1() { LC_ALL=C awk -v a="$1" -v b="$2" 'BEGIN { printf "%.1f", a / b }'; }

# fewer() renders a CALL comparison. Equal counts are "the same", not
# "1.0x fewer" — a ratio of one is not a reduction, and shape C prints exactly
# that for its whole-file row. Codex, #136.
fewer() {
  if [ "$1" = "$2" ]; then echo "the same"
  else echo "$(round1 "$1" "$2")x fewer"; fi
}


# ratio and ratio2 report a comparison, and report a LOSS as a loss: "0.8x less
# input" is a sentence that hides which way it went.
#
# ⚠ THEY LIVE AT FILE SCOPE, not inside measure(). `ratio` used to be defined in
# measure()'s body, which in bash leaks it globally once measure has run — so
# shape E called it successfully only because shape A ran first. A function that
# works because of the order its callers happen to appear in is a defect waiting
# for someone to reorder the file.
ratio() {
  if [ "$2" -le "$1" ]; then echo "$(round1 "$1" "$2")x LESS"
  else echo "$(round1 "$2" "$1")x MORE"; fi
}

# Two decimals, for the one place a THIRD digit is the finding: shape E exists to
# show that the overhead does not move across two orders of magnitude, and
# "1.1x, 1.1x, 1.1x" cannot distinguish flat from slowly drifting.
round2() { LC_ALL=C awk -v a="$1" -v b="$2" 'BEGIN { printf "%.2f", a / b }'; }
ratio2() {
  if [ "$2" -le "$1" ]; then echo "$(round2 "$1" "$2")x LESS"
  else echo "$(round2 "$2" "$1")x MORE"; fi
}
# A SELF-CHECK, because the defect it replaces was invisible: a truncated ratio
# is a plausible number, and nobody recomputes a plausible number. 1.29 must
# print as 1.3. Put back `bc scale=1` and this exits 2 before a single figure
# is published, which is the only way a silent arithmetic regression announces
# itself in a script whose whole output is numbers.
[ "$(round1 1290 1000)" = "1.3" ] || {
  echo "measure.sh: round1 did not print 1.3 for 1290/1000 — it printed '$(round1 1290 1000)'." >&2
  echo "  a '1.2' means it is TRUNCATING and every ratio below would understate;" >&2
  echo "  a '1,3' means the decimal separator is not pinned and LC_ALL=C was lost." >&2
  exit 2
}

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

  rule
  printf '%s\n' "$label"
  printf '  %d site(s) across %d file(s)\n\n' "$nsites" "$nfiles"
  printf '  %-38s %10s %10s   %s\n' ""                          "baseline" "mrw" ""
  printf '  %-38s %10s %10s   %s\n' "bytes, vs reading files WHOLE" \
    "$whole" "$ranged" "$(ratio "$whole" "$ranged") input"
  printf '  %-38s %10s %10s   %s\n' "bytes, vs a WINDOWED read" \
    "$windowed" "$ranged" "$(ratio "$windowed" "$ranged") input"
  printf '  %-38s %10s %10s   %s\n' "calls, whole-file (reads+edits)" \
    "$wholecalls" "$mrwcalls" "$(fewer "$wholecalls" "$mrwcalls")"
  printf '  %-38s %10s %10s   %s\n' "calls, windowed (search+reads+edits)" \
    "$windowcalls" "$mrwcalls" "$(fewer "$windowcalls" "$mrwcalls")"
}

# ⚠ PORCELAIN, NOT `git diff --quiet`. That misses STAGED and UNTRACKED changes,
# so a tree with an untracked .go file — which `go build` sees and shape D's
# `git ls-files` does not — was stamped with a bare commit and could not be
# reproduced from it. Codex, second review of #136.
# ⚠ AND THE STAMP SAYS WHOSE BINARY IT IS. $MRW may name a binary built from
# another tree — a release, a colleague's build — and then the commit below
# describes the FIXTURES and the file list, not the code that produced the
# bytes. Codex, third review of #136.
echo "mrw measurement — $(git rev-parse --short HEAD)$([ -n "$(git status --porcelain)" ] && echo ' (DIRTY TREE — these numbers are not reproducible from that commit)')$([ -n "${MRW_EXTERNAL:-}" ] && echo " (binary supplied via \$MRW: $MRW — NOT built from this tree)")"

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

# Shape E: ONE file, three spans. The other four shapes vary the task; this one
# holds the task still and varies how much of a file it needs, which is the only
# way to see what the overhead actually IS.
#
# It answers two questions the ratio columns cannot. First, mrw's cost against a
# windowed read is a FLAT ~13% — the line-number gutter, which is what makes a
# served line addressable by a later write — and it does not grow with the span.
# Second, the SAVING is not a property of file size at all: it is the part of the
# file you were never going to look at, so it collapses as the span approaches
# the whole file.
#
# ⚠ THIS EXISTS BECAUSE README.md QUOTED THESE NUMBERS WITH NO WAY TO REPRODUCE
# THEM, in the section whose own opening line is "a number nobody can reproduce
# is a claim, not a measurement". Measuring it by hand and pasting the result was
# the same defect one level up. Found by the Codex review of #136.
#
# The fixture is synthetic and uniform on purpose — 20,000 identical 53-byte
# lines — so the gutter is isolated from any variation in line length. awk
# generates it rather than python3, which this script does not otherwise need.
spans() {
  local dir big whole n w m
  dir=$(mktemp -d "$SCRATCH/span-XXXXXX")
  big="$dir/big.go"
  awk 'BEGIN { for (i = 1; i <= 20000; i++) printf "\tif err := doSomething(ctx, item%05d); err != nil {\n", i }' > "$big"
  whole=$(wc -c < "$big" | tr -d ' ')

  rule
  printf '%s\n' "E. One 1.06 MB file, three spans — the overhead, and where the saving comes from"
  printf '  1 file, %d lines, %d bytes\n\n' 20000 "$whole"
  printf '  %-16s %12s %12s %12s   %s\n' "span" "whole file" "windowed" "mrw" ""
  for n in 100 2000 20000; do
    w=$(sed -n "1,${n}p" "$big" | wc -c | tr -d ' ')
    m=$("$MRW" -C "$dir" read "big.go:1-$n" | wc -c | tr -d ' ')
    printf '  %-16s %12s %12s %12s   %s vs whole, %s vs windowed\n' \
      "$n lines" "$whole" "$w" "$m" "$(ratio2 "$whole" "$m")" "$(ratio2 "$w" "$m")"
  done
  rm -rf "$dir"   # and the EXIT trap covers an abnormal exit before this line
}
spans

rule
echo "Round trips are the floor: mrw is 2 calls for any N. Bytes depend on how"
echo "much of each file the task needs — measure YOUR shape before quoting a number."
