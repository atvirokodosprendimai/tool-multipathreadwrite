# Stress: `mrw write` flag combinations

`stress.py` drives a built `mrw` — never the Go API — through seeded combinations of the write flags, and holds
every run to nine invariants that no single-flag test states together. It was written on 2026-10-01 for B2 in
BACKLOG "From ADR-108" (the Codex design review of v1.37.1: "flags are tested one at a time"), and it stays here as
a repeatable check for anyone changing `writeCmd`, the foreign-format compilers, the balance rules or the receipt.

```sh
python3 docs/break/write-flags/stress.py                          # the mrw on PATH
MRW=$PWD/bin/mrw SEED=7 N=400 python3 docs/break/write-flags/stress.py
```

Unix only. Exit 0 means no invariant was broken; exit 1 writes every violation to a JSON file in the system temp
directory and prints its path. It writes nothing into the checkout: each run gets a fresh tree, its own state
directory and a temp directory inside the harness's work directory, which goes with the run, an interrupted one
included.

## What it varies

Per run, seeded: the target (`a.txt`, prose — no default check, no balance rows; or `x.go`, code — the check runs
by default and a delimiter move prints a balance row); the plan format (native, `--format=apply_patch`,
`--format=search_replace`); the plan shape (one that applies, one that cannot, one against a file that was never
read, and on `x.go` the wrap-tail replace of `func f() {` by `func g()`); the check (none, passing, failing); and
the flags — `--json`, `--dry-run`, `--strict-balance`, `--check` or `--no-check` or neither, `--then-sh` with a
passing or failing step, `--echo-pad`.

## What it checks

The docstring lists the invariants (W1–W9). In short: the exit code is 0–3; under `--json` stdout is one object
whose every key path `docs/receipts.txt` lists for `write` (ADR-111); exit 1 and `--dry-run` leave the tree
byte-identical; a landed write has exactly the planned effect; the receipt carries the plan's one hunk with a valid
status, `ok` when it landed; exit 3 has a check that ran and failed, or a step that failed, as its evidence;
`--strict-balance` refuses the wrap-tail replace and its absence lets it land with a balance advisory; and a landed
code write runs a configured check by default while a prose write does not.

One shape is allowed on purpose: `--check` beside `--dry-run` is a usage refusal, exit 2, made before the plan, and
like a flag the parser rejects it prints nothing on stdout — ADR-072 makes every refusal *after a plan is named* one
document, and this one is not. The exemption covers that combination and nothing else.

## Results

2026-10-01, `mrw` built at `2041f33` (v1.37.2 plus ADR-109–112): seed 1380, 400 runs, 0 violations, 93 distinct
target/format/shape/check/exit cells; seed 7, 400 runs, 0 violations, 93 cells.

The gate was shown to fail, and each check to reach real runs: against a copy of `docs/receipts.txt` without the
`write hunks` keys, 40 runs gave 20 W2 violations; and with each expectation inverted in turn, 120 runs gave 59 W6,
10 W7, 5 W8 and 5 W9 violations; and against two fake binaries, one answering every --json write with `null` and exit 3 and one refusing the non-strict wrap-tail write, 120 runs each gave 82 W2 and 9 W8 violations. The first cut of W8 expected `--strict-balance` to refuse any delimiter move; the
code refuses the wrap-tail shape only (ADR-055, `internal/apply/apply.go:1478`), as the README says, and AGENTS.md,
which said otherwise, was corrected.
