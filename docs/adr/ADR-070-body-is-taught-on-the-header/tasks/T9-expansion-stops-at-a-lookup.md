# Task ADR-070-T9: The expansion walk stops at a lookup-only `command -v`

**Depends-on:** T8
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the lookup stop in the wrapper walk
**Consumes:** T8's wrapper walk
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a lookup expands nothing`

## Goal

The fifth Codex review of PR #222 found that T8's walk to `env` crossed `command -v`, a lookup that runs nothing, so `command -v env -S 'mrw --help'` scored a `--help` that never ran: the help pass, which reads the unstripped segment, saw the manufactured word. The fix: the walk stops at `command -v` and `-V`, leaving the segment as written. Pin it; the 16 replay trials score as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | `expand_split_string` |
| `scripts/test_blind_score.py` | edit | one test |

## Ordered Steps

1. [S1] Add the test; confirm RED. [proof: mutation]
2. [S2] Fix the walk; GREEN; the twenty T3–T8 tests stay green; re-scoring the 16 replay trials moves nothing. [proof: mutation]
   Mutant: the lookup stop dropped.

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T9.out \
  && grep -qE "^test_expansion_stops_at_a_lookup_only_wrapper \(.*\) \.\.\. ok$" /tmp/adr070-T9.out \
  && grep -q '^OK' /tmp/adr070-T9.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_expansion_stops_at_a_lookup_only_wrapper` | `scripts/test_blind_score.py` | `command -v env -S 'mrw --help'` meets; `command env -S 'mrw --help'` voids | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `score` runs on every trial |
| 3 — the caller can discover it | the violation names the command |
| 4 — it is used | reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · 2517b6a* · exit 1 · `set -o pipefail …` · acceptance-sha256:85dd40a04c35776e57bafc134454be75464aed8b14d2712ad7cd07ba5641ecb9 · ms:82 · test-lock-sha256:2ddfb10bb3bbab050a0def76688ba50ba613caa251d47efb7019a8b641122f2c · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jb21tYW5kX3ZfaXNfYV9sb29rdXBfbm90X2FfY2FsbAk0ZjY2NzlmYzI2OGM5OTJmOGMzYWM3ZmI0YjFlNTU0Mzk2NDkyNWIwYzk1YWZlNzYwZjM0YjQ2ZWUwOGNhZDQ5CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZW52X21yd19pc19jb3VudGVkCTNjM2NlZTM5YjE1OTExYjM1MzZhMGFiZDk5NTZlNWY1NmNhZTBlYTJiYTkxMzRjNmIzYjFkYjkxYWM0MjQ5MDgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX2lzX2V4cGFuZGVkX2JlZm9yZV9oZWxwX2FuZF9vcHRpb25zCTNlZmYyNzA1M2IxNjYyYmEzZWYxMmVlYmZlNGNkNGNlODdmNjE4MTQ1NjIxYzFhZjNmN2QwYjg2NWEzMzQ3NzgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX3J1bnNfaXRzX29wZXJhbmQJOWFiMGI2NmQ3N2E3MzMwN2ZkNzkwNjgzMmI1OGY2OGFjNmQ4ZTFkYWQ1ZTg5YTY4YjZmMDAyMmFhZjY2OTkwZApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2VzY2FwZWRfYmFja3NsYXNoX2RvZXNfbm90X21hbnVmYWN0dXJlX2Ffc3Vic3RpdHV0aW9uCTIwNDczNzZhY2Y2ZDE4MTA4YzkxYjY2NzgyMDI3ZWNlODE0OGI3Y2I0NTU2OGE2NjI2MzYwNzMzYWRhMzRiODEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lc2NhcGVkX2hlcmVkb2Nfc3Vic3RpdHV0aW9uX2lzX2xpdGVyYWwJOWYwNWEyNmEwZmZkNTE5Mzc2M2M3YTEwN2UxZDVkZWI5N2FkZjU0YTRiMTE4NGZkOWUyZDk1NzBkOWExMzIyMApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2V4cGFuZGFibGVfaGVyZWRvY19zdWJzdGl0dXRpb25faXNfc2Nhbm5lZAlmMjk2NjY1OTNkOTBhOTUzNTQzZDIwNGVmZWJmYmZjZTA3MTg0ZWI5OWUyOGU2YmI1YWQ2YWI2OTAzMmJlMDg4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZXhwYW5zaW9uX3N0b3BzX2F0X2FfbG9va3VwX29ubHlfd3JhcHBlcgk5NzVkYTI0ZjAxYjRkNTBjZTkyMGJlOWI2MWNjNTNkYmY5ZmU0ZjBmMzQ1NjlmYWRjZmI1Y2JhNTEwZDUxOGM2CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfaGVyZWRvY19ib2R5X2lzX25vdF9jb21tYW5kcwlkMTc4NjU1MTY3MGUyOWRhZmY1MWZlNTMwZDUxNDdjOGEzM2I1YmE1YTY5Mjc4MzE3OWRjOTgzOTZmMDQ5ZTRiCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfbWlzdHlwZWRfYW5zd2VyX2RvZXNfbm90X2NyYXNoCTQ1ZWM2YjQxZjMxMjMzNjYyZmJmZDE5MTEyNThlNzVkYTE1OGU2MWJiZGYwMzdkMmQyNjY2OTc3ZWFjZDAyYWEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9ub25fb2JqZWN0X2ZpbmFsX2ZlbmNlX2lzX25vdF9za2lwcGVkCThlYjc2ZDBiNDk3OWQ5Y2FjZDBhYWE2OTNmNTBlZDJjMWRiM2MyYzcwNmI0YjI0YTJjYWFhOTRlYmYxMmM1N2MKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9xdW90ZWRfYmFubmVkX3dvcmRfaXNfbm90X2FfdmlvbGF0aW9uCTNmZTYzMTRiZWIxYThhMGVmNjkzNmM2YWJmNzY2NjA4NmZmZjU4NDVkNTRkYTdiMjJkOGY2OGUwNTVjMzk3YTIKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9zcGxpdF9zdHJpbmdfaXNfZXhwYW5kZWRfYmVoaW5kX2Ffd3JhcHBlcgliNWI3ODllNDFhOTIwYWI5MmZhYjhmNzk2MzJhNDA2NzJmODY3ZjNjZmRkNjZjZmJmYjZiNjkzMjExZmE3ZTI4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfdGhlX2VzY2FwZV9tYXNrX2tlZXBzX2Ffc3Vic3RpdHV0aW9uX2ludGVyaW9yCTMyMGU2ODI1NTA1YzFmNGE5ZjRhMjdhODA4NGQxYThmNDM2OThhN2E0N2E0ZDgwZjI0Njg4ZDMyYzgyNzkyYzgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF90aGVfbWFza19rZWVwc19pdHNfbGVuZ3RoCTk2YzkxM2Q5Yzg2OGIzMWY4ZjQ5NTQwMDIwOTZlNWNkOGFhMmQzY2VjZmIxNmFlNzQ4NGFiOWVlNzlhNjU2NWQKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX2hlbHBfaXNfYmFubmVkCWNmZTg1MWQ3ZmUyNzljYmYyZTU1YzgxYzY4ZDY0OWJmNjViZjFiNTQxNWYxYjM1MzJjOTUyNDVlMmQ1YTJiMTMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX29wdGlvbl9vcGVyYW5kc19hcmVfc2tpcHBlZAlhMzQ0MzU4M2EzMGEzMmY5YzBjNDE0ZTA5ZmY5MDc4NTQ4NTUyYjEwNTkzZDczYTJiZDY2MmYzNmRlMjU1ZjIwCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfemVyb19tcndfY2FsbHNfY2Fubm90X21lZXQJNTZmMDYzODEyZDYyOGRmMzQyODA0ZmFiODg5NTg5YTY5NTQyY2E2NmE5MDAzYWE2YmYxMTA2ZjBhMzYyMDUzMw
  ```
  --- last 10 line(s) of stdout (of 37 after folding 38 raw)
      self.assertEqual(bs.score(d, t)["verdict"], "MEETS")
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 'VOID' != 'MEETS'
  - VOID
  + MEETS
    (x2)
  ----------------------------------------------------------------------
  Ran 21 tests in 0.023s
  
  FAILED (failures=1)
  ```
- 2026-09-25 · 2517b6a* · exit 0 · `set -o pipefail …` · acceptance-sha256:85dd40a04c35776e57bafc134454be75464aed8b14d2712ad7cd07ba5641ecb9 · ms:108
- 2026-09-25 · 2517b6a* · exit 0 · `set -o pipefail …` · acceptance-sha256:85dd40a04c35776e57bafc134454be75464aed8b14d2712ad7cd07ba5641ecb9 · ms:113

## Mutation Log
(empty until execute)
- 2026-09-25 · 2517b6a* · mutant killed · exit 1 · `scripts/blind-score.py` · the lookup stop is dropped: command -v env -S mrw --help scores a --help that never ran · acceptance-sha256:85dd40a04c35776e57bafc134454be75464aed8b14d2712ad7cd07ba5641ecb9 · covers:a lookup expands nothing

## Invariants

- The 16 replay trials score as before.
- The criterion is unchanged.

## Risks

- `builtin` and `exec` have no lookup mode; `type` and `which` are not wrappers the scorer knows, so `which env` never reaches the walk.

## Out of Scope

- Parsing every shell construct (permanent: boundary: the scorer models what readings 01–04 transcripts contain, pinned by tests)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
