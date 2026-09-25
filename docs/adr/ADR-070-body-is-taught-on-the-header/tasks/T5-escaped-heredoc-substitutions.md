# Task ADR-070-T5: The scorer honours a backslash escape in an unquoted heredoc body

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** escape-aware scan of unquoted heredoc bodies in `segments`
**Consumes:** T4's `strip_heredocs`, `substitutions`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an escaped substitution is literal`

## Goal

The Codex review of PR #222 found that T4's scan of an unquoted heredoc body read `\$(cat f)` as a substitution, though the shell keeps it literal: a transcript with correct answers scored `VOID: banned: command cat`. Escaped backticks have the same defect. The fix: a backslash before `$`, `` ` `` or `\` in an unquoted body is honoured before substitutions are read, so `\\$(cat f)` (an escaped backslash, then a real substitution) is still scanned. Pin both with a synthetic transcript; the 16 replay trials score as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | the body scan in `segments` |
| `scripts/test_blind_score.py` | edit | one test |

## Ordered Steps

1. [S1] Add the test; confirm RED. [proof: mutation]
2. [S2] Fix the scan; GREEN; the thirteen T3/T4 tests stay green; re-scoring the 16 replay trials moves nothing. [proof: mutation]
   Mutant: the escape stripping dropped.

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T5.out \
  && grep -qE "^test_escaped_heredoc_substitution_is_literal \(.*\) \.\.\. ok$" /tmp/adr070-T5.out \
  && grep -q '^OK' /tmp/adr070-T5.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_escaped_heredoc_substitution_is_literal` | `scripts/test_blind_score.py` | `\$(…)` and `` \` `` are literal; `\\$(…)` is scanned | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `score` runs on every trial |
| 3 — the caller can discover it | the violation names the command |
| 4 — it is used | reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · f2661ae* · exit 1 · `set -o pipefail …` · acceptance-sha256:eefe71abf9f1d5778b89d6132d889e85985386886d4d44191ef43b7ab5fb312d · ms:75 · test-lock-sha256:6177c03459b180b89ec3a819139932e050e4c508045d6e2e216f902820391354 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jb21tYW5kX3ZfaXNfYV9sb29rdXBfbm90X2FfY2FsbAk0ZjY2NzlmYzI2OGM5OTJmOGMzYWM3ZmI0YjFlNTU0Mzk2NDkyNWIwYzk1YWZlNzYwZjM0YjQ2ZWUwOGNhZDQ5CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZW52X21yd19pc19jb3VudGVkCTNjM2NlZTM5YjE1OTExYjM1MzZhMGFiZDk5NTZlNWY1NmNhZTBlYTJiYTkxMzRjNmIzYjFkYjkxYWM0MjQ5MDgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lc2NhcGVkX2hlcmVkb2Nfc3Vic3RpdHV0aW9uX2lzX2xpdGVyYWwJOWYwNWEyNmEwZmZkNTE5Mzc2M2M3YTEwN2UxZDVkZWI5N2FkZjU0YTRiMTE4NGZkOWUyZDk1NzBkOWExMzIyMApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2V4cGFuZGFibGVfaGVyZWRvY19zdWJzdGl0dXRpb25faXNfc2Nhbm5lZAlmMjk2NjY1OTNkOTBhOTUzNTQzZDIwNGVmZWJmYmZjZTA3MTg0ZWI5OWUyOGU2YmI1YWQ2YWI2OTAzMmJlMDg4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfaGVyZWRvY19ib2R5X2lzX25vdF9jb21tYW5kcwlkMTc4NjU1MTY3MGUyOWRhZmY1MWZlNTMwZDUxNDdjOGEzM2I1YmE1YTY5Mjc4MzE3OWRjOTgzOTZmMDQ5ZTRiCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfbWlzdHlwZWRfYW5zd2VyX2RvZXNfbm90X2NyYXNoCTQ1ZWM2YjQxZjMxMjMzNjYyZmJmZDE5MTEyNThlNzVkYTE1OGU2MWJiZGYwMzdkMmQyNjY2OTc3ZWFjZDAyYWEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9ub25fb2JqZWN0X2ZpbmFsX2ZlbmNlX2lzX25vdF9za2lwcGVkCThlYjc2ZDBiNDk3OWQ5Y2FjZDBhYWE2OTNmNTBlZDJjMWRiM2MyYzcwNmI0YjI0YTJjYWFhOTRlYmYxMmM1N2MKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9xdW90ZWRfYmFubmVkX3dvcmRfaXNfbm90X2FfdmlvbGF0aW9uCTNmZTYzMTRiZWIxYThhMGVmNjkzNmM2YWJmNzY2NjA4NmZmZjU4NDVkNTRkYTdiMjJkOGY2OGUwNTVjMzk3YTIKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX2hlbHBfaXNfYmFubmVkCWNmZTg1MWQ3ZmUyNzljYmYyZTU1YzgxYzY4ZDY0OWJmNjViZjFiNTQxNWYxYjM1MzJjOTUyNDVlMmQ1YTJiMTMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX29wdGlvbl9vcGVyYW5kc19hcmVfc2tpcHBlZAlhMzQ0MzU4M2EzMGEzMmY5YzBjNDE0ZTA5ZmY5MDc4NTQ4NTUyYjEwNTkzZDczYTJiZDY2MmYzNmRlMjU1ZjIwCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfemVyb19tcndfY2FsbHNfY2Fubm90X21lZXQJNTZmMDYzODEyZDYyOGRmMzQyODA0ZmFiODg5NTg5YTY5NTQyY2E2NmE5MDAzYWE2YmYxMTA2ZjBhMzYyMDUzMw
  ```
  --- last 10 line(s) of stdout (of 30 after folding 31 raw)
      self.assertEqual(bs.score(d, t)["verdict"], "MEETS")
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 'VOID' != 'MEETS'
  - VOID
  + MEETS
    (x2)
  ----------------------------------------------------------------------
  Ran 14 tests in 0.012s
  
  FAILED (failures=1)
  ```
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:eefe71abf9f1d5778b89d6132d889e85985386886d4d44191ef43b7ab5fb312d · ms:91
- 2026-09-25 · f2661ae* · exit 0 · `set -o pipefail …` · acceptance-sha256:eefe71abf9f1d5778b89d6132d889e85985386886d4d44191ef43b7ab5fb312d · ms:121

## Mutation Log
(empty until execute)
- 2026-09-25 · f2661ae* · mutant killed · exit 1 · `scripts/blind-score.py` · the escape is ignored again: a literal $(cat f) voids the run · acceptance-sha256:eefe71abf9f1d5778b89d6132d889e85985386886d4d44191ef43b7ab5fb312d · covers:an escaped substitution is literal

## Invariants

- The 16 replay trials score as before.
- The criterion is unchanged.

## Risks

- A backslash-newline inside an unquoted body joins lines; the scan treats it as two lines. No reading transcript contains one.

## Out of Scope

- Parsing every shell construct (permanent: boundary: the scorer models what readings 01–04 transcripts contain, pinned by tests)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
