# Task ADR-092-T6: Help shows a step value as the receipt does

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `TestHelpDoesNotEchoARawStepValue`
**Consumes:** `shown` (T5)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `help shows a step value quoted`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

Close the Codex review of PR #269 (xhigh, 2026-09-28, head 6d82bbc): T5 quoted step names and
commands in the receipt, but `stepFlag.String()` returned the raw values, and urfave prints it as the
flag's default in `--help` — so `mrw check --then $'x\e[2Jy' --help` wrote the escape to the
terminal. The same review's other two findings are not code under a fence: §174's machine-wide
pgrep/pkill (now matched by this run's fixture path) and AGENTS.md's Windows claim (now Unix only).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `stepFlag.String` shows each value |
| `cmd/mrw/then092t6_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestHelpDoesNotEchoARawStepValue`. [proof: mutation]
2. [S2] `String()` maps `shown` over the values. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestHelpDoesNotEchoARawStepValue' -v 2>&1 | tee /tmp/adr092-T6.out \
  && grep -qE '^--- PASS: TestHelpDoesNotEchoARawStepValue \(' /tmp/adr092-T6.out \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 8ecb059 -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestHelpDoesNotEchoARawStepValue` | `cmd/mrw/then092t6_test.go` | `check --help` and `write --help` given a `--then` and a `--then-sh` holding ESC print no raw ESC | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `stepFlag.String` |
| 2 — something selects it | urfave's help renderer |
| 3 — the caller can discover it | `--help` itself |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #269 |

## Mutation Log
(empty until execute)
- 2026-09-28 · 6d82bbc* · mutant killed · exit 1 · `cmd/mrw/main.go` · help shows the raw value · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · covers:help shows a step value quoted
- 2026-09-28 · 6d82bbc* · mutant killed · exit 1 · `cmd/mrw/main.go` · main.go is not gofmt-clean · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · covers:the tree is gofmt-clean
- 2026-09-28 · 6d82bbc* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · covers:no other engine package changes

## Invariants

- `Values()` stays raw: the padded-argument differential compares what the parser delivered.

## Risks

- None beyond the record's.

## Out of Scope

- An ADR-069 padded-value refusal quoting a value with control bytes inside single quotes (permanent: boundary: that refusal is ADR-069's, for every flag, and prints the fix the shell needs)

## Stop Condition

The fence exits 0.

## Verification Log
(empty until execute)
- 2026-09-28 · 6d82bbc* · exit 1 · `set -o pipefail …` · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · ms:788 · test-lock-sha256:1e971b5898531ee2a83542c48dcaa03633037c6f023230ba0e137c3af01a1f56 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGhlbjA5MnQ2X3Rlc3QuZ28JVGVzdEhlbHBEb2VzTm90RWNob0FSYXdTdGVwVmFsdWUJOWU4Zjc3NWViNWFmMDAwY2M0YThhZTExOGI2NGVkYmJkN2JhODY5ODQxZjI1NWE4MmI3ZjY1YWExZDkxY2I1YQ
  ```
  --- last 9 line(s) of stdout
  === RUN   TestHelpDoesNotEchoARawStepValue
      then092t6_test.go:20: check --help echoes a raw control byte:
          "NAME:\n   mrw check - run the project's check, scoped to the working set or to the given paths\n\nUSAGE:\n   mrw check [options] [PATH...]\n\nDESCRIPTION:\n   The command comes from .quality-harness.json (\"check\" for the whole project,\n   \"scoped_check\" for a narrow run, with {packages} and {files} expanded). With no\n   such file, a Go project falls back to \"go test ./...\" — and the output says the\n   command was INFERRED, because an inferred check can be red on a tree you never\n   touched, which is a finding about the machine and not about your change.\n\nOPTIONS:\n   --json         emit the result as JSON\n   --full         run the whole-project check, ignoring any scope\n   --then NAME    after a landed write and a passing check, run the step NAME declared in .quality-harness.json \"steps\" (repeatable; runs in command-line order with --then-sh; the first that does not pass stops the rest) (default: x\x1b[2Jy)\n   --then-sh CMD  like --then, but run CMD with sh -c as given (repeatable). --then-sh runs any shell command it is given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh. (default: true \x1b[31m)\n   --help, -h     show help\n\nGLOBAL OPTIONS:\n   --root DIR, -C DIR  resolve every path relative to DIR (default: \".\")\n"
      then092t6_test.go:20: write --help echoes a raw control byte:
          "NAME:\n   mrw write - apply an edit plan across one or more files; a plan that fails validation writes nothing\n\nUSAGE:\n   mrw write [options] [PLAN|-]\n\nDESCRIPTION:\n   A plan is a sequence of hunks:\n\n     @@ <path> <addr> <op> [sha=… lines=… anchor=… body=…]\n     <body lines>\n\n   Ops are replace, insert-after, insert-before, delete, create, unlink and rename.\n   unlink and rename take address - (a hyphen, no line number). unlink's body is\n   empty. rename's destination is the one-line body (not a to= key). Other\n   addresses are 1-based and inclusive, and every one of them resolves against\n   the ORIGINAL file — so several hunks in one file need no offset arithmetic.\n\n   The optional guards are what make a batch safe to trust: sha= pins the whole\n   file, lines= asserts how many lines the range covers, anchor= requires a\n   substring in the range's first line. If any hunk fails validation, every hunk is\n   reported and NOTHING is written. A filesystem failure while committing can leave\n   some files written: the receipt says PARTIALLY APPLIED and names them.\n\n   A value with spaces can be double-quoted (anchor=\"func openTestStore\"),\n   single-quoted (anchor='func openTestStore'), or — for anchor= only — left\n   unquoted until the next key= (anchor=func openTestStore body=1).\n   body= is a line count, not a character count. Python str splits characters;\n   do not use len(body) as body=.\n   body=@path loads the body from a root-relative file (an empty file is the\n   same as body=0). An unquoted anchor= that contains a double quote is refused;\n   write it as anchor=\"…\". A leftover body= names the declared count and how\n   many extra lines sat before the next @@.\n   lines= is a guard on how many lines the ADDRESS covers, and is not body=.\n\n   The checkout is named by global -C DIR or --root DIR before the subcommand\n   (mrw -C repo write plan). After read, -C is context lines, not a checkout.\n\n   Prefer authoring the plan with your harness's own file tool and passing its\n   path, rather than piping it in: a plan on disk is a reviewable artifact and is\n   visible to whatever hooks watch file writes.\n\n   --format=apply_patch compiles a Codex apply_patch document (*** Begin Patch)\n   into the native plan above, then Parse and Apply run unchanged. A git patch\n   is not an apply_patch; --format=git is usage.\n   --format=search_replace compiles an Aider SEARCH/REPLACE document\n   (<<<<<<< SEARCH / ======= / >>>>>>> REPLACE) the same way. Default --format is plan.\n\n   --echo-pad N prints N lines after an applied body so a surviving closer is\n   visible. Default 0. It is not a checker: a closer in the pad does not fail\n   the hunk. Negative is usage.\n\n   After a successful apply the project's check runs by default when a written\n   path is not prose (.md .markdown .txt .rst .adoc) and a check exists — declared\n   in .quality-harness.json or inferred from go.mod. A markdown-only plan does\n   not spawn it; a tree with no check stays exit 0. --no-check opts out. --check\n   is a demand: it runs on prose too, and is exit 2 when nothing can run. A\n   failing check is exit 3 and the tree is NOT reverted.\n\n   A non-prose hunk whose {} () [] nets differ between the replaced lines and\n   the body prints a balance row under ok. It does not fail the hunk and it is\n   not a checker: braces in strings miscount, and a balanced insert landing\n   inside a function is invisible to it — the check catches that, the row does\n   not.\n\n   The summary line counts those rows — \"N hunk(s), M file(s), F failed, A\n   advisories — applied\", zero included — and the JSON receipt carries\n   \"advisories\". When three of your last ten landed writes carried one, the\n   receipt prints a pattern line — read past the range before the next one.\n   --strict-balance is opt-in: it refuses a single-line replace whose line's\n   {} () [] do not balance and whose body does not match them (the wrap-tail\n   shape) as a failed hunk — exit 1, nothing is written. Off by default; braces\n   inside strings count, so drop it for that plan. Both JSON receipts (--json\n   and mrw_write) carry \"pattern\" {advisory_writes, window, fires} on every\n   write, and mrw stats prints a \"strict-balance pricing\" block as writes land —\n   how many the flag would have refused, and whether those writes then broke,\n   held or went unchecked.\n\nOPTIONS:\n   --dry-run, -n     validate and report, write nothing\n   --force           edit files mrw has not read, or that changed since it last saw them (the escape hatch, not the habit)\n   --json            emit the receipt as JSON (for hooks and quality gates)\n   --quiet           print only failures and the summary line\n   --check           demand the project's check after a successful write, even on a prose-only plan (exit 2 if none can run)\n   --no-check        do not run the project's check after this write (the default runs it when a written path is not prose)\n   --format string   plan (default), apply_patch (Codex *** Begin Patch; a git patch is not one), or search_replace (Aider SEARCH/REPLACE) (default: \"plan\")\n   --echo-pad int    print N lines after an applied body (opt-in pad; default 0; not a checker) (default: 0)\n   --strict-balance  refuse a single-line replace whose line's {} () [] do not balance and whose body does not match them (the wrap-tail shape); exit 1, nothing written. Off by default\n   --then NAME       after a landed write and a passing check, run the step NAME declared in .quality-harness.json \"steps\" (repeatable; runs in command-line order with --then-sh; the first that does not pass stops the rest) (default: x\x1b[2Jy)\n   --then-sh CMD     like --then, but run CMD with sh -c as given (repeatable). --then-sh runs any shell command it is given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh. (default: true \x1b[31m)\n   --help, -h        show help\n\nGLOBAL OPTIONS:\n   --root DIR, -C DIR  resolve every path relative to DIR (default: \".\")\n"
  --- FAIL: TestHelpDoesNotEchoARawStepValue (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.305s
  FAIL
  ```
- 2026-09-28 · 6d82bbc* · exit 0 · `set -o pipefail …` · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · ms:818
- 2026-09-28 · 6d82bbc* · exit 0 · `set -o pipefail …` · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · ms:363
- 2026-09-28 · 6d82bbc* · exit 0 · `set -o pipefail …` · acceptance-sha256:45d04e1b8ebc06a06dc75e31ba16b5bd61f7b9f2288c57480469bd5edd19af19 · ms:351
