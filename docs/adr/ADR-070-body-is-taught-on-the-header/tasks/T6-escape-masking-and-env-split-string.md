# Task ADR-070-T6: The scorer masks an escaped pair without joining its neighbours, and reads `env -S` as a command line

**Depends-on:** T5
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the escape mask in `segments`; the `-S` branch of `strip_wrappers`
**Consumes:** T5's escape handling, T4's `strip_wrappers`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an escaped pair is masked, not deleted`, `env -S runs its operand`

## Goal

The second Codex review of PR #222 found two misreads. T5 deleted an escaped pair, which joined its neighbours: `$\\(cat f)`, a literal `$` and an escaped backslash, became `$(cat f)` and voided a run with correct answers. And `env -S` / `--split-string` was read as an option with an operand, so `env -S cat go.mod` scored `go.mod` as the command word while `cat` ran. The fix: the pair is replaced by a space, and the `-S` operand is split into words and read as the command line it is. Pin both; the 16 replay trials score as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | the mask in `segments`; `strip_wrappers`; `WRAPPER_OPERANDS` |
| `scripts/test_blind_score.py` | edit | two tests |

## Ordered Steps

1. [S1] Add the two tests; confirm each RED. [proof: mutation]
2. [S2] Fix the scorer; GREEN; the fourteen T3–T5 tests stay green; re-scoring the 16 replay trials moves nothing. [proof: mutation]
   Mutants: the mask deletes again; the `-S` branch dropped.

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T6.out \
  && grep -qE "^test_escaped_backslash_does_not_manufacture_a_substitution \(.*\) \.\.\. ok$" /tmp/adr070-T6.out \
  && grep -qE "^test_env_split_string_runs_its_operand \(.*\) \.\.\. ok$" /tmp/adr070-T6.out \
  && grep -q '^OK' /tmp/adr070-T6.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_escaped_backslash_does_not_manufacture_a_substitution` | `scripts/test_blind_score.py` | `$\\(cat f)` runs nothing | — | S1, S2 |
| `test_env_split_string_runs_its_operand` | `scripts/test_blind_score.py` | `env -S cat f` voids; `env -S 'mrw read a'` counts | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | `score` runs on every trial |
| 3 — the caller can discover it | the violation names the command |
| 4 — it is used | reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · ms:110
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · ms:96
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · ms:91
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · ms:100
- 2026-09-25 · 50becfb* · exit 1 · `set -o pipefail …` · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · ms:77 · test-lock-sha256:3ddc2719b9711c1ca0e6b9e2410f434c054e6bb560bff8324832b58a663e81eb · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jb21tYW5kX3ZfaXNfYV9sb29rdXBfbm90X2FfY2FsbAk0ZjY2NzlmYzI2OGM5OTJmOGMzYWM3ZmI0YjFlNTU0Mzk2NDkyNWIwYzk1YWZlNzYwZjM0YjQ2ZWUwOGNhZDQ5CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZW52X21yd19pc19jb3VudGVkCTNjM2NlZTM5YjE1OTExYjM1MzZhMGFiZDk5NTZlNWY1NmNhZTBlYTJiYTkxMzRjNmIzYjFkYjkxYWM0MjQ5MDgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX3J1bnNfaXRzX29wZXJhbmQJOWFiMGI2NmQ3N2E3MzMwN2ZkNzkwNjgzMmI1OGY2OGFjNmQ4ZTFkYWQ1ZTg5YTY4YjZmMDAyMmFhZjY2OTkwZApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2VzY2FwZWRfYmFja3NsYXNoX2RvZXNfbm90X21hbnVmYWN0dXJlX2Ffc3Vic3RpdHV0aW9uCTIwNDczNzZhY2Y2ZDE4MTA4YzkxYjY2NzgyMDI3ZWNlODE0OGI3Y2I0NTU2OGE2NjI2MzYwNzMzYWRhMzRiODEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lc2NhcGVkX2hlcmVkb2Nfc3Vic3RpdHV0aW9uX2lzX2xpdGVyYWwJOWYwNWEyNmEwZmZkNTE5Mzc2M2M3YTEwN2UxZDVkZWI5N2FkZjU0YTRiMTE4NGZkOWUyZDk1NzBkOWExMzIyMApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2V4cGFuZGFibGVfaGVyZWRvY19zdWJzdGl0dXRpb25faXNfc2Nhbm5lZAlmMjk2NjY1OTNkOTBhOTUzNTQzZDIwNGVmZWJmYmZjZTA3MTg0ZWI5OWUyOGU2YmI1YWQ2YWI2OTAzMmJlMDg4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfaGVyZWRvY19ib2R5X2lzX25vdF9jb21tYW5kcwlkMTc4NjU1MTY3MGUyOWRhZmY1MWZlNTMwZDUxNDdjOGEzM2I1YmE1YTY5Mjc4MzE3OWRjOTgzOTZmMDQ5ZTRiCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfbWlzdHlwZWRfYW5zd2VyX2RvZXNfbm90X2NyYXNoCTQ1ZWM2YjQxZjMxMjMzNjYyZmJmZDE5MTEyNThlNzVkYTE1OGU2MWJiZGYwMzdkMmQyNjY2OTc3ZWFjZDAyYWEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9ub25fb2JqZWN0X2ZpbmFsX2ZlbmNlX2lzX25vdF9za2lwcGVkCThlYjc2ZDBiNDk3OWQ5Y2FjZDBhYWE2OTNmNTBlZDJjMWRiM2MyYzcwNmI0YjI0YTJjYWFhOTRlYmYxMmM1N2MKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9xdW90ZWRfYmFubmVkX3dvcmRfaXNfbm90X2FfdmlvbGF0aW9uCTNmZTYzMTRiZWIxYThhMGVmNjkzNmM2YWJmNzY2NjA4NmZmZjU4NDVkNTRkYTdiMjJkOGY2OGUwNTVjMzk3YTIKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX2hlbHBfaXNfYmFubmVkCWNmZTg1MWQ3ZmUyNzljYmYyZTU1YzgxYzY4ZDY0OWJmNjViZjFiNTQxNWYxYjM1MzJjOTUyNDVlMmQ1YTJiMTMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX29wdGlvbl9vcGVyYW5kc19hcmVfc2tpcHBlZAlhMzQ0MzU4M2EzMGEzMmY5YzBjNDE0ZTA5ZmY5MDc4NTQ4NTUyYjEwNTkzZDczYTJiZDY2MmYzNmRlMjU1ZjIwCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfemVyb19tcndfY2FsbHNfY2Fubm90X21lZXQJNTZmMDYzODEyZDYyOGRmMzQyODA0ZmFiODg5NTg5YTY5NTQyY2E2NmE5MDAzYWE2YmYxMTA2ZjBhMzYyMDUzMw
  ```
  --- last 10 line(s) of stdout (of 41 after folding 42 raw)
      self.assertEqual(bs.score(d, t)["verdict"], "MEETS")
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 'VOID' != 'MEETS'
  - VOID
  + MEETS
    (x2)
  ----------------------------------------------------------------------
  Ran 16 tests in 0.011s
  
  FAILED (failures=2)
  ```
- 2026-09-25 · 50becfb* · exit 0 · `set -o pipefail …` · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · ms:102

## Mutation Log
(empty until execute)
- 2026-09-25 · 50becfb* · mutant killed · exit 1 · `scripts/blind-score.py` · the pair is deleted again: dollar, escaped backslash, paren joins into a substitution · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · covers:an escaped pair is masked, not deleted
- 2026-09-25 · 50becfb* · mutant killed · exit 1 · `scripts/blind-score.py` · -S is an ordinary option again: env -S cat f scores f as the command · acceptance-sha256:0d50fc5f4b5cfdd17c1d6efbceae5d376460d0706913716c22ae4ac2da5bc3e3 · covers:env -S runs its operand

## Invariants

- The 16 replay trials score as before.
- The criterion is unchanged.

## Risks

- `env -S` with `${VAR}` expansion inside the string is split as literal words; no reading transcript contains one.

## Out of Scope

- Parsing every shell construct (permanent: boundary: the scorer models what readings 01–04 transcripts contain, pinned by tests)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
