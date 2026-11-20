#!/usr/bin/env bash
# ADR-211 Decision 10: the free rehearsal of everything the paid Stage 9 steps rely on. It spends nothing: the CLI every
# attempt starts is bench/stage-9/stub-agent, never `claude`. Run from a clean checkout at the revision pilot.md will name:
#   bench/stage-9/rehearse.sh                      writes docs/bench/stage-9/rehearsal.md
#   REHEARSE_SUBSET=K REHEARSE_SCRATCH=DIR bench/stage-9/rehearse.sh OUT
#       a trial run over the first K tasks of each class of each plan; it writes OUT, never the committed record, and the
#       record it writes says `subset:` so no one mistakes it for the rehearsal.
# The record has the date, the revision, one digest line for every input, and one `check NAME: exit CODE` line for each check
# and each negative; the exit code is what the check's command returned (a code the script computes, 1). A check or a negative
# that fails does not stop the run: the record shows it, and the exit status is 1.
set -u
cd "$(git rev-parse --show-toplevel)" || exit 2
repo=$PWD
default=docs/bench/stage-9/rehearsal.md
subset=${REHEARSE_SUBSET:-}
out=${1:-}
if [ -n "$subset" ]; then
	case $subset in '' | *[!0-9]* | 0) echo "rehearse.sh: REHEARSE_SUBSET is a positive count" >&2; exit 2 ;; esac
	[ -n "$out" ] || { echo "rehearse.sh: a subset run needs the file to write: it never writes $default" >&2; exit 2; }
	if [ "$(cd "$(dirname "$out")" 2>/dev/null && pwd -P)/$(basename "$out")" = "$repo/$default" ]; then
		echo "rehearse.sh: a subset run never writes $default" >&2
		exit 2
	fi
else
	out=${out:-$default}
	[ "$out" = "$default" ] || { echo "rehearse.sh: the full rehearsal writes $default" >&2; exit 2; }
	[ -z "$(git status --porcelain)" ] || { echo "rehearse.sh: the tree is not clean: the rehearsal is of a revision" >&2; exit 2; }
fi
for tool in jq shasum go git python3; do
	command -v "$tool" >/dev/null 2>&1 || { echo "rehearse.sh: $tool is not on PATH" >&2; exit 2; }
done
S=${REHEARSE_SCRATCH:-$(mktemp -d)}
mkdir -p "$S"/{builds,trees,canaries,runs,neg,stub,stub-quiet,config} || exit 2
rev=$(git rev-parse HEAD)
stage=bench/stage-9
models=$(jq -r '[.[].model] | unique | .[]' $stage/plans.json)
log() { printf '%s %s\n' "$(date +%H:%M:%S)" "$*" >&2; }
sha() { shasum -a 256 "$1" | cut -d' ' -f1; }
rec=()

# The plans the run uses: the registered file, or with a subset the first K tasks of each class of each plan.
if [ -n "$subset" ]; then
	jq --argjson k "$subset" --slurpfile c $stage/classes.json \
		'map(.tasks as $t | .tasks = [$c[0][] | map(select(. as $x | $t | index($x))) | .[0:$k][]])' $stage/plans.json >"$S/plans.json"
else
	cp $stage/plans.json "$S/plans.json"
fi

# --- the stub, with the instructions of every follow-on and control task beside it, and the same stub with none
install -m 755 $stage/stub-agent "$S/stub/stub-agent"
install -m 755 $stage/stub-agent "$S/stub-quiet/stub-agent"
for id in $(jq -r '[."follow-on", .control] | add | .[]' $stage/classes.json); do
	jq -r '.instruction | gsub("\n"; " ")' "bench/tasks/$id/task.json"
done >"$S/stub/stub-exposed.txt"
: >"$S/stub-quiet/stub-exposed.txt"

# --- the two builds and the stand-in pilot.md the gate and count read
# shellcheck disable=SC2016 # the backticks are builds.md's own
tag=$(sed -n 's/^- Tag: `\(v[^`]*\)`.*/\1/p' $stage/builds.md | head -n 1)
want_m=$(sed -n 's/^m sha256: //p' $stage/builds.md)
want_n=$(sed -n 's/^n sha256: //p' $stage/builds.md)
built=1
log "building $tag"
sh $stage/build.sh "$tag" "$S/builds" >"$S/build.out" 2>&1 || built=0
if [ $built = 1 ]; then
	[ "$(sha "$S/builds/mrw-m")" = "$want_m" ] && [ "$(sha "$S/builds/mrw-n")" = "$want_n" ] || built=0
fi
{
	echo "m sha256: $want_m"
	echo "n sha256: $want_n"
	echo "instructions sha256: $(sha $stage/instructions.txt)"
	for m in $models; do echo "init model $m: $m"; done
} >"$S/pilot.md"

# go_test NAME [DIR]: the named test passed, and was not skipped.
go_test() {
	local name=$1 dir=${2:-$repo} c
	(cd "$dir" && go test ./internal/bench -run "^$name\$" -count=1 -v) >"$S/$name.out" 2>&1
	c=$?
	if [ $c -eq 0 ] && { ! grep -q "^--- PASS: $name " "$S/$name.out" || grep -q SKIP "$S/$name.out"; }; then c=1; fi
	return $c
}

# run_of MODEL ARM DIR PLANS CANARY STUBDIR [TASKS...]: one bench run, its exit status recorded beside it.
run_of() {
	local model=$1 arm=$2 dir=$3 plans=$4 canary=$5 stubdir=$6 cfg=B figure c
	shift 6
	[ "$arm" = a ] && cfg=A
	figure=$(jq -r --arg d "$model/$arm" '.[] | select(.directory == $d) | .max_spend_usd' "$plans")
	local args=(run --stage stage9 --out "$dir/$model/$arm" --model "$model" --claude "$stubdir/stub-agent" --config-dir "$S/config"
		--containment bypass --canary "$canary" --scratch "$S/trees" --configs "$cfg" --repeats 1 --seed 206 --status-line --keep-diff
		--max-turns 60 --max-spend "$figure")
	[ "$arm" = a ] || args+=(--mrw "$S/builds/mrw-$arm" --mrw-sha256 "$(sed -n "s/^$arm sha256: //p" "$S/pilot.md")" --mrw-instructions "$stage/instructions.txt")
	[ "$arm" != n ] || args+=(--keep-scratch-file stage9-exposed.jsonl)
	local id
	for id in "$@"; do args+=(--task "$id"); done
	mkdir -p "$dir/$model"
	"$S/bench" "${args[@]}" >"$dir/$model/$arm.out" 2>"$dir/$model/$arm.stderr"
	c=$?
	echo "$c" >"$dir/$model/$arm.exit"
	return $c
}

tasks_of() { jq -r --arg d "$2/$3" '.[] | select(.directory == $d) | .tasks[]' "$1"; }

# n_event COUNTS: every follow-on and control attempt of arms M and N is `exposed`, and no arm A attempt is.
n_event() {
	jq -e --slurpfile k $stage/classes.json '
		([$k[0]["follow-on"], $k[0].control] | add) as $two
		| [.models | to_entries[] | .key as $m | .value | to_entries[] | .key as $arm | .value.tasks | to_entries[] | {m: $m, arm: $arm, id: .key, e: .value.exposure}] as $all
		| if ([$all[] | select(.arm == "n")] | length) == 0 then "no arm n attempt was counted" | error else . end
		| [$all[] | select((.arm == "a" and .e == "exposed") or (.arm != "a" and ((.id as $i | $two | index($i)) != null) and .e != "exposed"))] as $bad
		| if ($bad | length) == 0 then true else ($bad | map("\(.m) \(.arm) \(.id) is \(.e), not as the stub makes it") | join("; ") | error) end' "$1" >/dev/null
}

# step NAME CODE: records one check.
step() { rec+=("check $1: exit $2"); log "check $1: exit $2"; }

# 1. instructions frozen
if [ $built = 1 ]; then "$S/builds/mrw-m" instructions >"$S/instructions.m" 2>/dev/null && cmp -s "$S/instructions.m" $stage/instructions.txt; c=$?; else c=1; fi
step 'instructions frozen' $c
# 2, 3. the two tests that skip when they cannot run
go_test TestArmNDiffersFromMInOneRefusal; step 'build test' $?
go_test TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit; step 'oracle test' $?
# 4. the bench build: from the clean tree, stamped with its revision
go build -o "$S/bench" ./cmd/bench >"$S/bench-build.out" 2>&1
c=$?; bench_ok=$c # whether the binary exists: the stamp check below judges the tree, and a trial in a modified one still runs the rest
if [ $c -eq 0 ]; then
	info=$(go version -m "$S/bench")
	grep -q "vcs.revision=$rev\$" <<<"$info" && grep -q 'vcs.modified=false$' <<<"$info" || c=1
fi
step 'bench build' $c

# 5. the four stub canaries, each gated before the next
c=0
if [ $built = 1 ] && [ $bench_ok -eq 0 ]; then
	prev=''
	for m in $models; do
		figure=$(python3 -B -c 'import sys; sys.path.insert(0, "scripts"); import bench_stage9 as s; print(s.CANARY_FIGURES[sys.argv[1]])' "$m")
		for a in n m; do
			name=canary-$m-$a
			"$S/bench" canary --out "$S/canaries/$name" --model "$m" --claude "$S/stub/stub-agent" --mrw "$S/builds/mrw-$a" \
				--mrw-sha256 "$(sed -n "s/^$a sha256: //p" "$S/pilot.md")" --mrw-instructions $stage/instructions.txt \
				--config-dir "$S/config" --containment bypass --scratch "$S/trees" --max-spend "$figure" >"$S/canaries/$name.out" 2>&1
			e=$?
			echo "$e" >"$S/canaries/$name.exit"
			[ -f "$S/canaries/$name/canary.json" ] && cp "$S/canaries/$name/canary.json" "$S/canaries/$name.json"
			if [ $e -ne 0 ]; then c=$e; break 2; fi
			g=$(python3 -B scripts/bench_stage9.py gate canary "$m" "$a" --target "$S/canaries/$name.json" --pilot "$S/pilot.md" \
				--gates "$S/canaries/gates.txt" --sequence smoke ${prev:+--after "$prev"} 2>"$S/canaries/$name.gate") || { c=$?; break 2; }
			echo "$g" >>"$S/canaries/gates.txt"
			prev="$m canary-$a"
		done
	done
else
	c=1
fi
step 'stub canaries' $c

# 6. the six runs of the plans in order, the stub the CLI, each gated before the next
c=0
if [ $built = 1 ] && [ $bench_ok -eq 0 ]; then
	prev=''
	for step_ in $(jq -r '.[] | "\(.model):\(.arm)"' "$S/plans.json"); do
		m=${step_%%:*}; a=${step_##*:}
		canary_arm=$a; [ "$a" = a ] && canary_arm=m
		ids=(); while IFS= read -r line; do ids+=("$line"); done < <(tasks_of "$S/plans.json" "$m" "$a")
		log "run $m $a: ${#ids[@]} attempts"
		run_of "$m" "$a" "$S/runs" "$S/plans.json" "$S/canaries/canary-$m-$canary_arm.json" "$S/stub" "${ids[@]}" || { c=$?; break; }
		g=$(python3 -B scripts/bench_stage9.py gate run "$m" "$a" --target "$S/runs/$m/$a" --plans "$S/plans.json" --pilot "$S/pilot.md" \
			--gates "$S/runs/gates.txt" --sequence pilot ${prev:+--after "$prev"} 2>"$S/runs/$m/$a.gate") || { c=$?; break; }
		echo "$g" >>"$S/runs/gates.txt"
		"$S/bench" verify --stage stage9 --tasks "${#ids[@]}" --repeats 1 "$S/runs/$m/$a" >"$S/runs/$m/$a.verify" 2>&1 || { c=$?; break; }
		prev="$m $a"
	done
else
	c=1
fi
step 'stub plans' $c
plans_ok=$c

# 7. count and report on the stub's output
c=1
if [ $plans_ok -eq 0 ]; then
	python3 -B scripts/bench_stage9.py count --receipts "$S/runs" --transcripts "$S/runs" --pilot "$S/pilot.md" --classes $stage/classes.json \
		--plans "$S/plans.json" >"$S/counts.json" 2>"$S/count.err" \
		&& python3 -B scripts/bench_stage9.py report --receipts "$S/runs" --counts "$S/counts.json" >"$S/report.txt" 2>"$S/report.err"
	c=$?
fi
step count $c
count_ok=$c

# 8. N's event is counted
c=1
[ $count_ok -eq 0 ] && { n_event "$S/counts.json" 2>"$S/n-event.err"; c=$?; }
step 'n event counted' $c

# 9. the paid steps' fences
python3 -B -m unittest scripts.test_bench_stage9.PaidFences.test_the_paid_fences_pass_a_fixture_and_fail_each_broken_one >"$S/fences.out" 2>&1
step 'paid fences' $?

# --- the negatives: each must fail, and for its own reason
negative() { # NAME CODE REASON-PATTERN FILE: the check's code, or 0 when it failed for another reason, which the fence reads as red
	local c=$2
	[ "$c" -ne 0 ] && ! grep -q -- "$3" "$4" && c=0
	rec+=("check $1 negative: exit $c"); log "check $1 negative: exit $c"
}
# n event counted, the stub making no write: one arm N attempt of the quiet stub, counted
c=1
if [ $built = 1 ] && [ $bench_ok -eq 0 ]; then
	m=$(jq -r '.[] | select(.arm == "n") | .model' "$S/plans.json" | head -n 1)
	jq --arg m "$m" 'map(select(.model == $m and .arm == "n") | .tasks |= .[0:1])' "$S/plans.json" >"$S/neg/plans.json"
	ids=(); while IFS= read -r line; do ids+=("$line"); done < <(tasks_of "$S/neg/plans.json" "$m" n)
	if run_of "$m" n "$S/neg" "$S/neg/plans.json" "$S/canaries/canary-$m-n.json" "$S/stub-quiet" "${ids[@]}" \
		&& python3 -B scripts/bench_stage9.py count --receipts "$S/neg" --transcripts "$S/neg" --pilot "$S/pilot.md" --classes $stage/classes.json \
			--plans "$S/neg/plans.json" >"$S/neg/counts.json" 2>"$S/neg/count.err"; then
		n_event "$S/neg/counts.json" 2>"$S/neg/n-event.err"; c=$?
	fi
fi
negative 'n event counted' $c 'not as the stub makes it' "$S/neg/n-event.err"
# stub plans, one plan's task list cut to one id: the gate refuses the full run it is shown
c=1
if [ $plans_ok -eq 0 ]; then
	first=$(jq -r '.[0] | "\(.model) \(.arm)"' "$S/plans.json")
	jq '.[0].tasks |= .[0:1]' "$S/plans.json" >"$S/neg/cut-plans.json"
	# shellcheck disable=SC2086
	set -- $first
	python3 -B scripts/bench_stage9.py gate run "$1" "$2" --target "$S/runs/$1/$2" --plans "$S/neg/cut-plans.json" --pilot "$S/pilot.md" \
		--gates "$S/neg/cut-gates.txt" --sequence pilot >/dev/null 2>"$S/neg/cut.err"
	c=$?
fi
negative 'stub plans' $c "tasks are not the registered plan's" "$S/neg/cut.err"
# build test, the pinned tag hidden: a clone with no tags, in which the test skips
rm -rf "$S/notags"
c=1
if git clone -q --no-tags --no-hardlinks "$repo" "$S/notags" 2>"$S/neg/clone.err"; then
	git -C "$S/notags" checkout -q "$rev" 2>>"$S/neg/clone.err" && ! git -C "$S/notags" tag | grep -q . && { go_test TestArmNDiffersFromMInOneRefusal "$S/notags"; c=$?; }
fi
negative 'build test' $c 'SKIP' "$S/TestArmNDiffersFromMInOneRefusal.out"
rm -rf "$S/notags"

# --- the record
{
	echo "date: $(date +%Y-%m-%d)"
	echo "revision: $rev"
	[ -z "$subset" ] || echo "subset: $subset task(s) of each class of each plan: a trial, not the rehearsal"
	for f in $stage/instructions.txt $stage/plans.json $stage/smoke-plans.json $stage/classes.json scripts/bench_stage9.py scripts/bench_mechanism.py \
		$stage/rehearse.sh $stage/stub-agent; do
		echo "$f sha256: $(sha "$f")"
	done
	echo "m sha256: $want_m"
	echo "n sha256: $want_n"
	# the aliases ADR-206 T5's fence reads
	echo "plans sha256: $(sha $stage/plans.json)"
	echo "smoke plans sha256: $(sha $stage/smoke-plans.json)"
	echo "instructions sha256: $(sha $stage/instructions.txt)"
	echo "tasks sha256: $(for id in $(jq -r '[.[] | .[]] | .[]' $stage/classes.json); do cat "bench/tasks/$id/task.json" "bench/tasks/$id/reference.patch"; done | shasum -a 256 | cut -d' ' -f1)"
	printf '%s\n' "${rec[@]}"
} >"$out.part" && mv "$out.part" "$out"
log "wrote $out; scratch $S"
pos_bad=$(grep -E '^check [^:]*: exit [1-9]' "$out" | grep -vc ' negative: ' || true)
neg_bad=$(grep -cE '^check .* negative: exit 0$' "$out" || true)
[ "$pos_bad" -eq 0 ] && [ "$neg_bad" -eq 0 ]
