# Task ADR-097-T2: README.md and AGENTS.md say what `mrw --instructions` answers

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** the named refusal (`subcommandForFlag`) (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the README clause`, `the AGENTS clause`, `the behaviour it documents holds`, `the subcommand and instructions doc gates hold`

## Goal

The two surfaces that already document `mrw instructions` and its usage errors — README.md `:31-33`
and AGENTS.md `:449-` — say in one clause, inside that paragraph and that bullet, that
`mrw --instructions` (and any other undefined flag that is exactly the name of a subcommand of the
command it was given to) is refused, exit 2, with a message naming the subcommand. `--version` and
`--help` are defined flags, not members of that class, and the clause must not say otherwise.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the `mrw instructions` paragraph (`:31-33`), beside "Extra arguments are usage (exit 2)" for `version` (`:28-29`) |
| `AGENTS.md` | edit | the `mrw instructions` bullet in "The other subcommands" (`:449-`), which the centralised skill mirrors |

**What selects it:** nothing in code; these are the two places a reader looking up `instructions` or
`version` lands. `internal/guide` is left alone (the record's Out of Scope).

## Ordered Steps

1. [S1] Confirm the fence below is RED before the edit: the `mrw instructions` paragraph of README.md
   and bullet of AGENTS.md hold neither `mrw --instructions` nor `exit 2` on `25acdc1`. [proof: mutation]
2. [S2] Add one clause INSIDE each of those two, e.g. "`mrw --instructions` is refused, exit 2, and the
   message names `mrw instructions`; so is any other undefined flag that is exactly the name of a
   subcommand of the command it was given to." Wording may differ; each clause must contain the
   literals `mrw --instructions` and `exit 2`, and must not say `--version` or `--help` is refused.
   Mutants to kill: either clause removed; either clause moved out of its paragraph or bullet. [proof: mutation]

## Acceptance

```bash
set -o pipefail
readme=$(awk '/^`mrw instructions` prints/{f=1} f&&/^$/{exit} f' README.md | tr '\n' ' ' | tr -s ' ') \
  && agents=$(awk '/^- \*\*`mrw instructions`\*\*/{f=1;print;next} f&&/^(- |$)/{exit} f' AGENTS.md | tr '\n' ' ' | tr -s ' ') \
  && printf '%s' "$readme" | grep -qF 'mrw --instructions' \
  && printf '%s' "$readme" | grep -qF 'exit 2' \
  && printf '%s' "$agents" | grep -qF 'mrw --instructions' \
  && printf '%s' "$agents" | grep -qF 'exit 2' \
  && out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAFlagNamedForASubcommandNamesTheSubcommand|TestEverySubcommandReachesTheAgentFacingGuide|TestReadmeAndSkillNameInstructions' -v 2>&1 | tee "$out" \
  && missing=$(for t in TestAFlagNamedForASubcommandNamesTheSubcommand TestEverySubcommandReachesTheAgentFacingGuide TestReadmeAndSkillNameInstructions; do grep -qE "^--- PASS: $t \(" "$out" || echo "$t"; done) \
  && [ -z "$missing" ]
```

The paragraph greps carry the verdict. Each reads only the `mrw instructions` paragraph of README.md
or its bullet in AGENTS.md, so a mention elsewhere in the file — an HTML comment, another section —
cannot satisfy it; on `25acdc1` both extractions are non-empty and hold neither literal (checked
2026-09-29). The test run proves the documented behaviour (T1) and the existing doc gates still hold,
so the clause cannot land on a tree where it is false; its log is a fresh `mktemp` file.
The segments rest on the four declared mechanisms: the README extraction and its two greps are
`the README clause`, the AGENTS extraction and its two are `the AGENTS clause`; the test run, its
PASS loop and the empty check together prove `the behaviour it documents holds` and `the subcommand
and instructions doc gates hold`.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFlagNamedForASubcommandNamesTheSubcommand` | `cmd/mrw/usage097_test.go` | the behaviour the clause documents (T1) | — | — |
| `TestEverySubcommandReachesTheAgentFacingGuide` | `cmd/mrw/agentsdoc_test.go` | AGENTS.md still names every subcommand | — | — |
| `TestReadmeAndSkillNameInstructions` | `cmd/mrw/agentsdoc_test.go` | README.md still names `mrw instructions` | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the clause in README.md and AGENTS.md |
| 2 — something selects it | the fence's two `grep -qF` lines |
| 3 — the caller can discover it | README.md's install notes and AGENTS.md's "The other subcommands" |
| 4 — it is used | nothing measures this yet |

## Mutation Log
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `README.md` · the README clause removed · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · covers:the README clause
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `AGENTS.md` · the AGENTS clause removed · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · covers:the AGENTS clause
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `README.md` · the README clause moved out of the mrw instructions paragraph · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · covers:the README clause
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `AGENTS.md` · the AGENTS clause moved out of the mrw instructions bullet · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · covers:the AGENTS clause
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · the documented behaviour removed: --instructions takes the v1.31.0 wording · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · covers:the behaviour it documents holds
- 2026-09-29 · 3582232* · mutant killed · exit 1 · `cmd/mrw/main.go` · a subcommand AGENTS.md does not name: the doc gate must go red · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · covers:the subcommand and instructions doc gates hold

## Invariants

- No other paragraph of README.md or AGENTS.md changes.
- The doc gates in `cmd/mrw/agentsdoc_test.go` stay green.

## Risks

- The AGENTS.md clause drifts from the centralised skill mirror until it is refreshed; the record's
  Out of Scope names who closes that (`external: agentsmemory`), since this fence cannot see it.

## Stop Condition

Stop and ask if T1 is not `done`, or if T1's sentence changed at acceptance (Decision 5) — the clause
must match what ships.

## Out of Scope

- `internal/guide` (`mrw instructions`' own text) — the record's Out of Scope.
- The code — T1.
- Refreshing the centralised `mrw` skill that mirrors AGENTS.md — the record's Out of Scope, `(external: agentsmemory …)`; not a step here, because this fence cannot see it.

## Verification Log
- 2026-09-29 · 3582232* · exit 1 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:35 · test-lock-sha256:042cf627478cf6d73a5269988818802e19cc881480dd30d6773c4280fa845fc1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdEV2ZXJ5U3ViY29tbWFuZFJlYWNoZXNUaGVBZ2VudEZhY2luZ0d1aWRlCWMzODMyODkzNDAzN2RmMTU5YWFhZDI5ZjdkOTYyMjVmMjY3NGI3NzM5MzQzYjUxNGI2ZjgyZTQ2MDA0ZDllNmMKYm9keQljbWQvbXJ3L2FnZW50c2RvY190ZXN0LmdvCVRlc3RSZWFkbWVBbmRTa2lsbE5hbWVJbnN0cnVjdGlvbnMJMGU2NDkxNjYxZWNkMDk0NzdmMTJjZjAxOWYyZDljNmEyN2Q2ODA5ZGYwZjZkODA2MDRlNWFhYzVjZDcwZWExYwpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZURvZXNOb3RUZWFjaEFMZWRnZXJSYWNlCTRjMDA5YzM4MDhkMzY2OTE2MDI5NGI5MDFlZGVmOGRlMmQzNDdjNGUzYzg3MDEyNDlhYzEyZjA1MTE4N2VlYWIKYm9keQljbWQvbXJ3L2FnZW50c2RvY190ZXN0LmdvCVRlc3RUaGVHYXRlRG9lc05vdEFjY2VwdEFQcmVmaXhPZkFub3RoZXJDb21tYW5kCTc4M2VlYjAyOGY5MGU3ZWE0MDU0NjFlNmZlZDkwYThjZGQyOTg2NTU4ODY0NTFlOGU3YzA2NjhiYjg2MTNhNjUKYm9keQljbWQvbXJ3L3VzYWdlMDk3X3Rlc3QuZ28JVGVzdEFGbGFnTmFtZWRGb3JBU3ViY29tbWFuZE5hbWVzVGhlU3ViY29tbWFuZAk4ZDg3YzhlYjNiNTEyZjI5NWZkYmMzZWFiOTIzMDkyMTM4NmJkYjA4NmM3ZTNiMmQyZTdjZmVjMjBiNTIzNjY4CmJvZHkJY21kL21ydy91c2FnZTA5N190ZXN0LmdvCVRlc3RBblVua25vd25GbGFnTmFtaW5nTm9TdWJjb21tYW5kSXNSZWZ1c2VkQXNCZWZvcmUJYzZiYzA5NTI0NzE5N2YxZDliMmUyNzkyOWQ3NzcxMTQxNjZlMjg3ZTUyN2Q4YWYzZDU3MjZlYjc4MzZlZDY4ZA
  ```
  ```
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:370
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:325
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:322
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:317
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:316
- 2026-09-29 · 3582232* · exit 0 · `set -o pipefail …` · acceptance-sha256:38a973f36404549a4ca519ebadb85a71ca61340726fb9c460e692d9e0d5b9891 · ms:336
