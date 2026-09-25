# Task ADR-070-T4: The scorer reads wrapper modes, wrapper help and expandable heredocs

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `strip_wrappers`, `WRAPPER_OPERANDS`, raw-segment help detection, `substitutions` over unquoted heredoc bodies
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a lookup is not a call`, `wrapper operands are skipped`, `wrapper help is banned`, `expandable heredocs are scanned`

## Goal

The Codex review of v1.25.0 found four misreads in the T3 scorer. `command -v mrw` counted as a call, and `command -v cat` voided a run, though both only look a name up. `env -u SOME_VAR mrw` counted `SOME_VAR` as the command. `env --help` was stripped before the help check saw it. And an UNQUOTED heredoc's `$(cat f)` was discarded with the body, though the shell runs it. Fix each one, and pin each with a synthetic transcript. Then re-score readings 03 and 04.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | `strip_wrappers`, `WRAPPER_OPERANDS`, `substitutions`, `segments(raw=)`, help on raw segments |
| `scripts/test_blind_score.py` | edit | four tests |

## Ordered Steps

1. [S1] Add the four tests; confirm each RED. [proof: mutation]
2. [S2] Fix the scorer; GREEN; the nine T3 tests stay green; re-scoring the 16 replay trials moves nothing. [proof: mutation]
   Mutants: the `command -v` return dropped; the operand skip dropped; help back on the stripped segment; the expandable-body scan dropped.

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T4.out \
  && missing=$(for t in test_command_v_is_a_lookup_not_a_call test_wrapper_option_operands_are_skipped test_wrapper_help_is_banned test_expandable_heredoc_substitution_is_scanned; do grep -qE "^$t \(.*\) \.\.\. ok$" /tmp/adr070-T4.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^OK' /tmp/adr070-T4.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_command_v_is_a_lookup_not_a_call` | `scripts/test_blind_score.py` | `command -v` runs nothing | — | S1, S2 |
| `test_wrapper_option_operands_are_skipped` | `scripts/test_blind_score.py` | `env -u VAR`, `xargs -n 1`, `xargs -I {}` reach the real command | — | S1, S2 |
| `test_wrapper_help_is_banned` | `scripts/test_blind_score.py` | `env --help` voids a run | — | S1, S2 |
| `test_expandable_heredoc_substitution_is_scanned` | `scripts/test_blind_score.py` | an unquoted heredoc's `$(cat f)` voids; a quoted one does not | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests |
| 2 — something selects it | `score` runs on every trial |
| 3 — the caller can discover it | the violation names the command |
| 4 — it is used | reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · 5165f0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · ms:121 · test-lock-sha256:6582eadb0c56efdbc653eedc178a15c4957f849a103981946549e334e1ae7812 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jb21tYW5kX3ZfaXNfYV9sb29rdXBfbm90X2FfY2FsbAk0ZjY2NzlmYzI2OGM5OTJmOGMzYWM3ZmI0YjFlNTU0Mzk2NDkyNWIwYzk1YWZlNzYwZjM0YjQ2ZWUwOGNhZDQ5CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZW52X21yd19pc19jb3VudGVkCTNjM2NlZTM5YjE1OTExYjM1MzZhMGFiZDk5NTZlNWY1NmNhZTBlYTJiYTkxMzRjNmIzYjFkYjkxYWM0MjQ5MDgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9leHBhbmRhYmxlX2hlcmVkb2Nfc3Vic3RpdHV0aW9uX2lzX3NjYW5uZWQJZjI5NjY2NTkzZDkwYTk1MzU0M2QyMDRlZmViZmJmY2UwNzE4NGViOTllMjhlNmJiNWFkNmFiNjkwMzJiZTA4OApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2hlcmVkb2NfYm9keV9pc19ub3RfY29tbWFuZHMJZDE3ODY1NTE2NzBlMjlkYWZmNTFmZTUzMGQ1MTQ3YzhhMzNiNWJhNWE2OTI3ODMxNzlkYzk4Mzk2ZjA0OWU0Ygpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X21pc3R5cGVkX2Fuc3dlcl9kb2VzX25vdF9jcmFzaAk0NWVjNmI0MWYzMTIzMzY2MmZiZmQxOTExMjU4ZTc1ZGExNThlNjFiYmRmMDM3ZDJkMjY2Njk3N2VhY2QwMmFhCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3Rfbm9uX29iamVjdF9maW5hbF9mZW5jZV9pc19ub3Rfc2tpcHBlZAk4ZWI3NmQwYjQ5NzlkOWNhY2QwYWFhNjkzZjUwZWQyYzFkYjNjMmM3MDZiNGIyNGEyY2FhYTk0ZWJmMTJjNTdjCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfcXVvdGVkX2Jhbm5lZF93b3JkX2lzX25vdF9hX3Zpb2xhdGlvbgkzZmU2MzE0YmViMWE4YTBlZjY5MzZjNmFiZjc2NjYwODZmZmY1ODQ1ZDU0ZGE3YjIyZDhmNjhlMDU1YzM5N2EyCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3Rfd3JhcHBlcl9oZWxwX2lzX2Jhbm5lZAljZmU4NTFkN2ZlMjc5Y2JmMmU1NWM4MWM2OGQ2NDliZjY1YmYxYjU0MTVmMWIzNTMyYzk1MjQ1ZTJkNWEyYjEzCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3Rfd3JhcHBlcl9vcHRpb25fb3BlcmFuZHNfYXJlX3NraXBwZWQJYTM0NDM1ODNhMzBhMzJmOWMwYzQxNGUwOWZmOTA3ODU0ODU1MmIxMDU5M2Q3M2EyYmQ2NjJmMzZkZTI1NWYyMApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X3plcm9fbXJ3X2NhbGxzX2Nhbm5vdF9tZWV0CTU2ZjA2MzgxMmQ2MjhkZjM0MjgwNGZhYjg4OTU4OWE2OTU0MmNhNjZhOTAwM2FhNmJmMTEwNmYwYTM2MjA1MzM
  ```
  --- last 10 line(s) of stdout (of 65 after folding 67 raw)
  Traceback (most recent call last):
    File "~/GolandProjects/tool-multipathreadwrite/scripts/test_blind_score.py", line 103, in test_wrapper_option_operands_are_skipped
      self.assertEqual(bs.score(d, t)["mrw_calls"], 2)
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 0 != 2
  
  ----------------------------------------------------------------------
  Ran 13 tests in 0.011s
  
  FAILED (failures=4)
  ```
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · ms:103
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · ms:89
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · ms:80
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · ms:87
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · ms:190

## Mutation Log
(empty until execute)
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `scripts/blind-score.py` · command -v counts as a call again: its test goes red · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · covers:a lookup is not a call
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `scripts/blind-score.py` · wrapper operands are read as commands again: its test goes red · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · covers:wrapper operands are skipped
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `scripts/blind-score.py` · help is checked on the stripped segment again: env --help passes and its test goes red · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · covers:wrapper help is banned
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `scripts/blind-score.py` · unquoted heredoc bodies are not scanned: its test goes red · acceptance-sha256:b349164241a100ca7e38d8b90d26917447f17ed3239d5f5d7b20954af7727c7b · covers:expandable heredocs are scanned

## Invariants

- The 16 replay trials score as before.
- The criterion is unchanged.

## Risks

- A wrapper this list does not know passes its operand as the command word; extend `WRAPPER_OPERANDS` when a transcript shows one.

## Out of Scope

- Parsing every shell construct (permanent: boundary: the scorer models what readings 01–04 transcripts contain, pinned by tests)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
