# ADR-172 Tasks

Implementation tasks for ADR-172. Source of truth: the task file headers.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Publish the trusted release formula | done | — | `python3 -m unittest discover -s scripts -p test_brew_tap.py -v` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

None — one task.

## Notes

T1's post-merge Actions/tap sign-off succeeded for v1.77.1 on 2026-10-28.
The exact run, tap commit, formula digest and all four manifest pairs are recorded in T1.
