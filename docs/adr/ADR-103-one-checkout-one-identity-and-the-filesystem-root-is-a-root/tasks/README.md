# ADR-103 Tasks

Implementation tasks for ADR-103: one checkout, one identity; and the filesystem root is a root. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `Contains` holds under a filesystem root; contract §198 | done | — | `docs/adr/ADR-103-one-checkout-one-identity-and-the-filesystem-root-is-a-root/tasks/T1-contains-under-root.md` fence |
| T2 | one canonical identity for a checkout | done | — | `docs/adr/ADR-103-one-checkout-one-identity-and-the-filesystem-root-is-a-root/tasks/T2-one-canonical-identity.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| none | — | — | independent |

## Notes

- Engine go/no-go: ADR-103 owns `internal/rooted`, `internal/state`, `internal/check` and one line of `internal/read`; `internal/apply`, `internal/plan`, `internal/seen`, `internal/lines`, `internal/iter` stay byte-identical.
- Contract section §198: §196 and §197 are ADR-102's (PR #293, open when this record was written), so this record takes the next.
