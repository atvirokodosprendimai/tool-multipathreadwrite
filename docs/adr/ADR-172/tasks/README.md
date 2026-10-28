# ADR-172 Tasks

Implementation tasks for ADR-172. Source of truth: the task file headers.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Publish the trusted release formula | partial | — | `python3 -m unittest discover -s scripts -p test_brew_tap.py -v` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

None — one task.

## Notes

T1 becomes partial after local proof and remains partial until its post-merge
Actions/tap sign-off succeeds.
