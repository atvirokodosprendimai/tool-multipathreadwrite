# ADR-008 Tasks

Implementation tasks for ADR-008: A delete says what it removed, and may say
what it expected. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes`
headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

T1 ships alone if T2 is never built — it makes a wrong delete legible, though
only T2 refuses one. With T1 alone the incident replays: the wrong range still
applies, still exits 0, still reports `ok`, and the two extra strings go past in
output the caller was not reading. T1 is the diagnostic; T2 is the refusal.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A delete receipt names the first and last line it removed | done | — | `go test ./internal/apply/ -run 'Delete.*Bounds\|OneLineDelete'` |
| T2 | A delete may say which lines it expects to remove | done | — | `go test ./internal/plan/ ./internal/apply/ ./internal/adversarial/ -run 'DeleteBody\|ExpectedRemoval\|BodylessDelete\|StillRejected'` |

Status: `pending` | `partial` | `blocked` | `done`.
