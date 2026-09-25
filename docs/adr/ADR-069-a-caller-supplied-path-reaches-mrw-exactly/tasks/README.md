# ADR-069 Tasks

Implementation tasks for ADR-069: a caller-supplied path reaches mrw exactly as typed, or is refused. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

T1–T4 are independent. T5 amends T1's guard after the Codex review of v1.25.0; T6 to T10 each
amend the one before after a round of the Codex review of PR #222; T11 checks them all against
the parser at random and amends what that found.

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3, T4 | none |
| 2 | T5 | none |
| 3 | T6 | T5 |
| 4 | T7 | T6 |
| 5 | T8 | T7 |
| 6 | T9 | T8 |
| 7 | T10 | T9 |
| 8 | T11 | T10 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | none |
| 5 | T5 | none |
| 6 | T6 | T5 |
| 7 | T7 | T6 |
| 8 | T8 | T7 |
| 9 | T9 | T8 |
| 10 | T10 | T9 |
| 11 | T11 | T10 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | The CLI refuses a positional its parser trimmed; contract §128 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T1-refuse-a-padded-positional.md` fence |
| T2 | `--files-from` and the working set keep a line's trailing space; contract §129 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T2-line-formats-keep-the-path.md` fence |
| T3 | A rename destination keeps its trailing space; contract §130 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T3-rename-destination-keeps-the-path.md` fence |
| T4 | apply_patch and search_replace keep a path's trailing space; contract §131 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T4-foreign-formats-keep-the-path.md` fence |
| T5 | A flag value does not end the guard; an attached padded value is refused; contract §134 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T5-flag-values-and-attached-values.md` fence |
| T6 | A flag name is read trimmed; the whole-argv guard reads values and the terminator; any trailing whitespace refuses an attached value; contract §135 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T6-flag-names-argv-walk-and-any-whitespace.md` fence |
| T7 | Both guards read the flags the parser accepts, ancestors included, and stop where it stops; contract §136 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T7-inherited-flags-and-the-parsers-stop.md` fence |
| T8 | A root `--` does not end the guard below it; a lone `-` ends it; contract §137 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T8-a-root-terminator-and-a-lone-dash.md` fence |
| T9 | The stop token is judged before the guard stops; contract §138 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T9-the-stop-token-is-judged-first.md` fence |
| T10 | A preserved stop token is not judged against a sibling; contract §139 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T10-a-preserved-stop-token-is-not-judged.md` fence |
| T11 | The guards agree with the parser on random argv; an iter verb is not judged | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T11-the-guards-agree-with-the-parser.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No `§NN` row in any Tests table.
- Engine go/no-go: ADR-069 owns `internal/iter` (T2) and `internal/apply` (T3). `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` and `internal/lines` stay byte-identical.
