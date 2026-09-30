# Stress: `mrw check --json` refusals and exit codes

`stress.py` drives a built `mrw` — never the Go API — and checks nine invariants on every run. It was
written on 2026-09-30 to stress v1.34.0 (ADR-100) and v1.34.1 (ADR-101), and it stays here as a
repeatable check for anyone changing `checkCmd`, `internal/check`'s results, or the depth guard.

```sh
python3 docs/break/check-json/stress.py                          # the mrw on PATH
MRW=$PWD/bin/mrw SEED=7 N=1500 python3 docs/break/check-json/stress.py
```

Unix only: one case makes a temp directory unwritable with `chmod`. Exit 0 means no invariant was
broken; exit 1 prints each distinct violation and writes all of them to a JSON file in the system temp
directory. It writes nothing into the checkout: every child gets its own state directory, and a temp
directory inside the harness's work directory as `TMPDIR`, so failed checks' logs and mrw's 7-day log
prune (ADR-080) stay there; the work directory is removed on exit, including an interrupted one.

## What it checks

The docstring at the top of `stress.py` lists the invariants (I1–I9). In short: one JSON object on
stdout under `--json`; a `check` refusal is `{"error"}` (plus `then`), never with `exit_code`;
`"ran": false` means `exit_code` -1 and `exit_code` 0 means the check ran; a depth `strconv.Atoi` reads
as 8 or more answers first; a refusal before flag parsing prints nothing on stdout; a refusal runs no
check.

It runs three groups: targeted cases (every refusal kind across eight trees — no check, passing,
failing, malformed harness, a timeout, a check killed by a signal, exit 255, an unreadable working set —
with padded, outside, missing, unicode and `--`-separated paths, ten depth values, a missing or
file-shaped `TMPDIR`, `write --check --json`, and ADR-101's step path end to end); 48 refusal and
no-check runs, eight at a time, across two checkouts; and `N` random combinations per `SEED`. Where a
case is certain to refuse, an empty or error-less document is a violation; a result missing `ran` or
`exit_code` is one too. I9 (a refusal ran no check) is judged only where the check touches a marker: the
serial targeted and fuzz runs, not the parallel group, whose runs share a tree. The last lines print how
often each outcome was reached, so a clean run also shows what it covered.

## Results, 2026-09-30 (v1.34.1, `626f25f`)

Four seeds (1340, 7, 424242, 99) with `N=1500`: 2,653 runs each, 10,612 in all, **0 violations**. The
seed-99 coverage included 594 error documents (8 with `then`), 355 pre-parse refusals, 749 depth
refusals, `"ran": false` with -1 on `check` and `write` receipts, -1 for a timed-out and a killed check,
and the step whose log could not be created reporting `could_not_start` with -1.

After the review of #292 tightened the harness — a certain refusal must carry `error`, a result must
carry both `ran` and `exit_code`, depth parsing matches `strconv.Atoi`, and every child gets the
harness's own `TMPDIR` — the same four seeds ran again: 10,612 runs, 0 violations, and the host temp
directory held 2,503 `mrw-check-*.log` files before and after. Each new check was shown to fire on the
shape it exists for (`{}` on a certain refusal, `{"ran": false}` without `exit_code`, a step without
`ran`) and to pass a valid refusal.

The first run reported 57 violations, every one a defect in the harness, not in mrw: a depth past int64
reads as 0 by ADR-095 Decision 3; a `write` refusal is its receipt plus `error` by ADR-072; and `--json`
after `--` is a path. The invariants above are the corrected ones.

Not covered: Windows, the MCP surface (it runs no check), and the Go suite under load.
