# ADR-179 tasks

Source of truth: the task files. This index is derived from them.

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3, T4, T6, T7, T8 | none |
| 2 | T5 | T1, T2, T4, T7, T8 |
| 3 | T9 | T1, T2, T3, T4, T5, T6, T7, T8 |

Order by measured payoff inside wave 1: T1, T2, T3, T4; then T6, T7, T8. T2 and T8 each stop before implementation if their measurement (S2) is below its BACKLOG bar.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | a long line served whole says so once; `MRW_MAX_COLS` sets the default | none | done |
| T2 | `--outline` prints a Go or Markdown file's structure and licenses nothing | none | done |
| T3 | `--if-served PATH=SHA` answers a block the caller and the ledger both hold by reference, opt-in | none | done |
| T4 | `--fold` collapses runs of identical and digit-only-different lines | none | done |
| T5 | no notice is printed twice in one invocation | T1, T2, T4, T7, T8 | done |
| T6 | `mrw write --terse` prints one line per file and every failure and advisory | none | done |
| T7 | `--max-matches N` caps the matches a whole read shows, and counts the rest | none | done |
| T8 | `--no-comments` elides Go comment blocks, leaving them unlicensed | none | done |
| T9 | the instructions teach the shorter reads and the terse receipt | T1-T8 | done |

**Branch: `adr-179`, from `main` after ADR-177 (#536) and ADR-178 (#541) merged.** This record and ADR-178 change `internal/read`; both hold `internal/check` byte-identical, which ADR-177 changed and has merged.

Status: `pending` | `partial` | `blocked` | `done`.
