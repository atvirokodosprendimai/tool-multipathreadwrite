# Stress: `mrw write` flag combinations

`stress.py` drives a built `mrw` — never the Go API — through seeded combinations of the write flags, and holds
every run to seven invariants that no single-flag test states together. It was written on 2026-10-01 for B2 in
BACKLOG "From ADR-108" (the Codex design review of v1.37.1: "flags are tested one at a time"), and it stays here as
a repeatable check for anyone changing `writeCmd`, the foreign-format compilers, or the receipt.

```sh
python3 docs/break/write-flags/stress.py                          # the mrw on PATH
MRW=$PWD/bin/mrw SEED=7 N=400 python3 docs/break/write-flags/stress.py
```

Unix only. Exit 0 means no invariant was broken; exit 1 writes every violation to a JSON file in the system temp
directory and prints its path. It writes nothing into the checkout: each run gets a fresh tree, its own state
directory and a temp directory inside the harness's work directory, which goes with the run, an interrupted one
included.

## What it varies

Per run, seeded: the plan format (native, `--format=apply_patch`, `--format=search_replace`); the plan shape (one
that applies, one that cannot, one against a file that was never read); the check (none, passing, failing); and the
flags — `--json`, `--dry-run`, `--strict-balance`, `--check` or `--no-check` or neither, `--then-sh` with a passing
or failing step, `--echo-pad`.

## What it checks

The docstring lists the invariants (W1–W7). In short: the exit code is 0–3; under `--json` stdout is one object
whose every key path `docs/receipts.txt` lists for `write` (ADR-111); exit 1 and `--dry-run` leave the tree
byte-identical; a landed write has exactly the planned effect; every hunk has a verdict; exit 3 means a check or a
step did not pass.

One shape is allowed on purpose: `--check` beside `--dry-run` is a usage refusal, exit 2, made before any receipt,
and like a flag the parser rejects it prints nothing on stdout — ADR-072 makes every refusal *after a plan is named*
one document, and this one is not.

## Results

2026-10-01, `mrw` built at `2041f33` (v1.37.2 plus ADR-109–112): seed 1380, 400 runs, 0 violations, 48 distinct
format/shape/check/exit cells; seed 7, 400 runs, 0 violations, 52 cells. The gate was shown to fail: against a copy
of `docs/receipts.txt` without the `write hunks` keys, 40 runs gave 20 W2 violations.
