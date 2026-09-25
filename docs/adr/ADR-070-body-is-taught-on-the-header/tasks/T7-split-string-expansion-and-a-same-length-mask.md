# Task ADR-070-T7: `env -S` is expanded before help detection in every spelling; the escape mask keeps a substitution's interior

**Depends-on:** T6
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `expand_split_string`, `split_words`; `substitutions(text, mask)`
**Consumes:** T6's mask and `-S` handling
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `split-string is expanded before both passes`, `the mask keeps the interior`

## Goal

The third Codex review of PR #222 found two gaps in T6. The `-S` expansion ran only where wrappers are stripped, so help detection, which reads the unstripped segment, saw `env -S 'mrw --help'` as one word while the call count saw an mrw call: a run that asked for help scored MEETS. It also stopped at the first option, so `env -S '-u X cat f'` and `env --split-string=…` still hid `cat`. And the mask shortened the text, so the interior of a real substitution moved: `$(cat\$suffix)` was read as `cat suffix` and voided a run. The fix: the operand is expanded into words before either pass, in every spelling, with env's remaining options still applied; and the mask keeps the text's length, so boundaries are found in the mask and the interior is taken from the text as written. Pin both; the 16 replay trials score as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | `substitutions`, `split_words`, `expand_split_string`, `strip_wrappers`, `segments` |
| `scripts/test_blind_score.py` | edit | two tests |

## Ordered Steps

1. [S1] Add the two tests; confirm each RED. [proof: mutation]
2. [S2] Fix the scorer; GREEN; the sixteen T3–T6 tests stay green; re-scoring the 16 replay trials moves nothing. [proof: mutation]
   Mutants: the expansion switched off; the interior taken from the mask instead of the text. A
   mutant that only shortens the mask SURVIVED this task's fixture: a one-character shift inside a
   single substitution left `cat$suffix` reading as `cat$suffi`, no banned word either way. T8 pins
   the length with a fixture the shift does flip (an escaped `$` before a backtick pair).

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T7.out \
  && grep -qE "^test_env_split_string_is_expanded_before_help_and_options \(.*\) \.\.\. ok$" /tmp/adr070-T7.out \
  && grep -qE "^test_the_escape_mask_keeps_a_substitution_interior \(.*\) \.\.\. ok$" /tmp/adr070-T7.out \
  && grep -q '^OK' /tmp/adr070-T7.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_env_split_string_is_expanded_before_help_and_options` | `scripts/test_blind_score.py` | `env -S 'mrw --help'` voids; `-u X` inside, `--split-string=`, and options before `-S` reach `cat` | — | S1, S2 |
| `test_the_escape_mask_keeps_a_substitution_interior` | `scripts/test_blind_score.py` | `$(cat\$suffix)` runs nothing banned; `$(cat f)` still voids | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | `score` runs on every trial |
| 3 — the caller can discover it | the violation names the command |
| 4 — it is used | reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · 97f8a02* · exit 1 · `set -o pipefail …` · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · ms:92 · test-lock-sha256:b2bafdf5de0bfec7e4c7e1c7e4c0a3bb845ce80b4807095a18b71bd361f77896 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jb21tYW5kX3ZfaXNfYV9sb29rdXBfbm90X2FfY2FsbAk0ZjY2NzlmYzI2OGM5OTJmOGMzYWM3ZmI0YjFlNTU0Mzk2NDkyNWIwYzk1YWZlNzYwZjM0YjQ2ZWUwOGNhZDQ5CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZW52X21yd19pc19jb3VudGVkCTNjM2NlZTM5YjE1OTExYjM1MzZhMGFiZDk5NTZlNWY1NmNhZTBlYTJiYTkxMzRjNmIzYjFkYjkxYWM0MjQ5MDgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX2lzX2V4cGFuZGVkX2JlZm9yZV9oZWxwX2FuZF9vcHRpb25zCTNlZmYyNzA1M2IxNjYyYmEzZWYxMmVlYmZlNGNkNGNlODdmNjE4MTQ1NjIxYzFhZjNmN2QwYjg2NWEzMzQ3NzgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX3J1bnNfaXRzX29wZXJhbmQJOWFiMGI2NmQ3N2E3MzMwN2ZkNzkwNjgzMmI1OGY2OGFjNmQ4ZTFkYWQ1ZTg5YTY4YjZmMDAyMmFhZjY2OTkwZApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2VzY2FwZWRfYmFja3NsYXNoX2RvZXNfbm90X21hbnVmYWN0dXJlX2Ffc3Vic3RpdHV0aW9uCTIwNDczNzZhY2Y2ZDE4MTA4YzkxYjY2NzgyMDI3ZWNlODE0OGI3Y2I0NTU2OGE2NjI2MzYwNzMzYWRhMzRiODEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lc2NhcGVkX2hlcmVkb2Nfc3Vic3RpdHV0aW9uX2lzX2xpdGVyYWwJOWYwNWEyNmEwZmZkNTE5Mzc2M2M3YTEwN2UxZDVkZWI5N2FkZjU0YTRiMTE4NGZkOWUyZDk1NzBkOWExMzIyMApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2V4cGFuZGFibGVfaGVyZWRvY19zdWJzdGl0dXRpb25faXNfc2Nhbm5lZAlmMjk2NjY1OTNkOTBhOTUzNTQzZDIwNGVmZWJmYmZjZTA3MTg0ZWI5OWUyOGU2YmI1YWQ2YWI2OTAzMmJlMDg4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfaGVyZWRvY19ib2R5X2lzX25vdF9jb21tYW5kcwlkMTc4NjU1MTY3MGUyOWRhZmY1MWZlNTMwZDUxNDdjOGEzM2I1YmE1YTY5Mjc4MzE3OWRjOTgzOTZmMDQ5ZTRiCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfbWlzdHlwZWRfYW5zd2VyX2RvZXNfbm90X2NyYXNoCTQ1ZWM2YjQxZjMxMjMzNjYyZmJmZDE5MTEyNThlNzVkYTE1OGU2MWJiZGYwMzdkMmQyNjY2OTc3ZWFjZDAyYWEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9ub25fb2JqZWN0X2ZpbmFsX2ZlbmNlX2lzX25vdF9za2lwcGVkCThlYjc2ZDBiNDk3OWQ5Y2FjZDBhYWE2OTNmNTBlZDJjMWRiM2MyYzcwNmI0YjI0YTJjYWFhOTRlYmYxMmM1N2MKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9xdW90ZWRfYmFubmVkX3dvcmRfaXNfbm90X2FfdmlvbGF0aW9uCTNmZTYzMTRiZWIxYThhMGVmNjkzNmM2YWJmNzY2NjA4NmZmZjU4NDVkNTRkYTdiMjJkOGY2OGUwNTVjMzk3YTIKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF90aGVfZXNjYXBlX21hc2tfa2VlcHNfYV9zdWJzdGl0dXRpb25faW50ZXJpb3IJMzIwZTY4MjU1MDVjMWY0YTlmNGEyN2E4MDg0ZDFhOGY0MzY5OGE3YTQ3YTRkODBmMjQ2ODhkMzJjODI3OTJjOApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X3dyYXBwZXJfaGVscF9pc19iYW5uZWQJY2ZlODUxZDdmZTI3OWNiZjJlNTVjODFjNjhkNjQ5YmY2NWJmMWI1NDE1ZjFiMzUzMmM5NTI0NWUyZDVhMmIxMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X3dyYXBwZXJfb3B0aW9uX29wZXJhbmRzX2FyZV9za2lwcGVkCWEzNDQzNTgzYTMwYTMyZjljMGM0MTRlMDlmZjkwNzg1NDg1NTJiMTA1OTNkNzNhMmJkNjYyZjM2ZGUyNTVmMjAKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF96ZXJvX21yd19jYWxsc19jYW5ub3RfbWVldAk1NmYwNjM4MTJkNjI4ZGYzNDI4MDRmYWI4ODk1ODlhNjk1NDJjYTY2YTkwMDNhYTZiZjExMDZmMGEzNjIwNTMz
  ```
  --- last 10 line(s) of stdout (of 43 after folding 44 raw)
  Traceback (most recent call last):
    File "~/GolandProjects/tool-multipathreadwrite/scripts/test_blind_score.py", line 131, in test_env_split_string_runs_its_operand
      self.assertEqual(bs.score(d, t)["mrw_calls"], 1)
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 0 != 1
  
  ----------------------------------------------------------------------
  Ran 18 tests in 0.020s
  
  FAILED (failures=2)
  ```
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · ms:89
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · ms:82
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · ms:105
- 2026-09-25 · 97f8a02* · exit 0 · `set -o pipefail …` · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · ms:85

## Mutation Log
(empty until execute)
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `scripts/blind-score.py` · the expansion is switched off: env -S mrw --help scores MEETS again · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · covers:split-string is expanded before both passes
- 2026-09-25 · 97f8a02* · mutant survived · exit 0 · `scripts/blind-score.py` · the mask shortens the text again: the interior of a real substitution moves · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · covers:the mask keeps the interior
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-25 · 97f8a02* · mutant killed · exit 1 · `scripts/blind-score.py` · the interior is taken from the mask: cat-backslash-dollar-suffix reads as cat suffix and voids the run · acceptance-sha256:0bf3048afabdf0adac96ff3af90877a1ab0d9b71b00a31c4bf1f601c6bfbed0e · covers:the mask keeps the interior

## Invariants

- The 16 replay trials score as before.
- The criterion is unchanged.

## Risks

- `env -S` with `${VAR}` expansion inside the string is split as literal words; no reading transcript contains one.

## Out of Scope

- Parsing every shell construct (permanent: boundary: the scorer models what readings 01–04 transcripts contain, pinned by tests)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
