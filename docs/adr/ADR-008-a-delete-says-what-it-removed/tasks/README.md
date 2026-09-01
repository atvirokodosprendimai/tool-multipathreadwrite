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

T1 ships alone if T2 is never built: the receipt bounds are what the incident
actually needed, and the expected body is the stronger, opt-in form.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A delete receipt names the first and last line it removed | pending | — | `go test ./internal/apply/ -run 'TestDeleteRecordsItsBounds'` |
| T2 | A delete may say which lines it expects to remove | pending | — | `go test ./internal/plan/ ./internal/apply/ ./internal/adversarial/ -run 'DeleteBody\|ExpectedRemoval'` |

Status: `pending` | `partial` | `blocked` | `done`.
