# Task ADR-070-T8: `env -S` is expanded behind a wrapper; the mask's length is pinned

**Depends-on:** T7
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** wrapper-aware search for `env` in the expansion; a fixture the mask's length can fail
**Consumes:** T7's expansion and mask
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `expansion is reached behind a wrapper`, `the mask keeps its length`

## Goal

The fourth Codex review of PR #222 found that T7's expansion looked for `env` only at the front of a segment, so `command env -S 'cat f'` hid `cat` again and `command env -S 'mrw read a'` counted no call; and that T7's mask-length claim had no fixture that could fail — a mask one character short left `cat$suffix` reading as `cat$suffi`, no banned word either way. The fix: the expansion walks the wrappers the scorer knows, with their operands, to reach `env`; and an escaped `$` before a backtick pair pins the length, because the short mask extracts `` `cat `` instead of `cat /dev/null`. Both pinned; the 16 replay trials score as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | `expand_split_string` |
| `scripts/test_blind_score.py` | edit | two tests |

## Ordered Steps

1. [S1] Add the two tests; confirm each RED (the second against a one-space mask). [proof: mutation]
2. [S2] Fix the expansion; GREEN; the eighteen T3–T7 tests stay green; re-scoring the 16 replay trials moves nothing. [proof: mutation]
   Mutants: the wrapper walk refused; the mask shortened.

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T8.out \
  && grep -qE "^test_split_string_is_expanded_behind_a_wrapper \(.*\) \.\.\. ok$" /tmp/adr070-T8.out \
  && grep -qE "^test_the_mask_keeps_its_length \(.*\) \.\.\. ok$" /tmp/adr070-T8.out \
  && grep -q '^OK' /tmp/adr070-T8.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_split_string_is_expanded_behind_a_wrapper` | `scripts/test_blind_score.py` | `command env -S 'cat f'` voids; `command env -S 'mrw read a'` counts | — | S1, S2 |
| `test_the_mask_keeps_its_length` | `scripts/test_blind_score.py` | an escaped `$` before a backtick pair still reaches `cat`; escaped backticks do not | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | `score` runs on every trial |
| 3 — the caller can discover it | the violation names the command |
| 4 — it is used | reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · a04ee52* · exit 1 · `set -o pipefail …` · acceptance-sha256:866cbe7b218038afd8b26e88e91f51681fe34862d850aa3d46cfeb9e03935781 · ms:99 · test-lock-sha256:da5447754d91cef0768a3c1643285e8c9975336e281aeece7437779629792cc1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jb21tYW5kX3ZfaXNfYV9sb29rdXBfbm90X2FfY2FsbAk0ZjY2NzlmYzI2OGM5OTJmOGMzYWM3ZmI0YjFlNTU0Mzk2NDkyNWIwYzk1YWZlNzYwZjM0YjQ2ZWUwOGNhZDQ5CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfZW52X21yd19pc19jb3VudGVkCTNjM2NlZTM5YjE1OTExYjM1MzZhMGFiZDk5NTZlNWY1NmNhZTBlYTJiYTkxMzRjNmIzYjFkYjkxYWM0MjQ5MDgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX2lzX2V4cGFuZGVkX2JlZm9yZV9oZWxwX2FuZF9vcHRpb25zCTNlZmYyNzA1M2IxNjYyYmEzZWYxMmVlYmZlNGNkNGNlODdmNjE4MTQ1NjIxYzFhZjNmN2QwYjg2NWEzMzQ3NzgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfc3BsaXRfc3RyaW5nX3J1bnNfaXRzX29wZXJhbmQJOWFiMGI2NmQ3N2E3MzMwN2ZkNzkwNjgzMmI1OGY2OGFjNmQ4ZTFkYWQ1ZTg5YTY4YjZmMDAyMmFhZjY2OTkwZApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2VzY2FwZWRfYmFja3NsYXNoX2RvZXNfbm90X21hbnVmYWN0dXJlX2Ffc3Vic3RpdHV0aW9uCTIwNDczNzZhY2Y2ZDE4MTA4YzkxYjY2NzgyMDI3ZWNlODE0OGI3Y2I0NTU2OGE2NjI2MzYwNzMzYWRhMzRiODEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lc2NhcGVkX2hlcmVkb2Nfc3Vic3RpdHV0aW9uX2lzX2xpdGVyYWwJOWYwNWEyNmEwZmZkNTE5Mzc2M2M3YTEwN2UxZDVkZWI5N2FkZjU0YTRiMTE4NGZkOWUyZDk1NzBkOWExMzIyMApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2V4cGFuZGFibGVfaGVyZWRvY19zdWJzdGl0dXRpb25faXNfc2Nhbm5lZAlmMjk2NjY1OTNkOTBhOTUzNTQzZDIwNGVmZWJmYmZjZTA3MTg0ZWI5OWUyOGU2YmI1YWQ2YWI2OTAzMmJlMDg4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfaGVyZWRvY19ib2R5X2lzX25vdF9jb21tYW5kcwlkMTc4NjU1MTY3MGUyOWRhZmY1MWZlNTMwZDUxNDdjOGEzM2I1YmE1YTY5Mjc4MzE3OWRjOTgzOTZmMDQ5ZTRiCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfbWlzdHlwZWRfYW5zd2VyX2RvZXNfbm90X2NyYXNoCTQ1ZWM2YjQxZjMxMjMzNjYyZmJmZDE5MTEyNThlNzVkYTE1OGU2MWJiZGYwMzdkMmQyNjY2OTc3ZWFjZDAyYWEKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9ub25fb2JqZWN0X2ZpbmFsX2ZlbmNlX2lzX25vdF9za2lwcGVkCThlYjc2ZDBiNDk3OWQ5Y2FjZDBhYWE2OTNmNTBlZDJjMWRiM2MyYzcwNmI0YjI0YTJjYWFhOTRlYmYxMmM1N2MKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9xdW90ZWRfYmFubmVkX3dvcmRfaXNfbm90X2FfdmlvbGF0aW9uCTNmZTYzMTRiZWIxYThhMGVmNjkzNmM2YWJmNzY2NjA4NmZmZjU4NDVkNTRkYTdiMjJkOGY2OGUwNTVjMzk3YTIKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9zcGxpdF9zdHJpbmdfaXNfZXhwYW5kZWRfYmVoaW5kX2Ffd3JhcHBlcgliNWI3ODllNDFhOTIwYWI5MmZhYjhmNzk2MzJhNDA2NzJmODY3ZjNjZmRkNjZjZmJmYjZiNjkzMjExZmE3ZTI4CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfdGhlX2VzY2FwZV9tYXNrX2tlZXBzX2Ffc3Vic3RpdHV0aW9uX2ludGVyaW9yCTMyMGU2ODI1NTA1YzFmNGE5ZjRhMjdhODA4NGQxYThmNDM2OThhN2E0N2E0ZDgwZjI0Njg4ZDMyYzgyNzkyYzgKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF90aGVfbWFza19rZWVwc19pdHNfbGVuZ3RoCTk2YzkxM2Q5Yzg2OGIzMWY4ZjQ5NTQwMDIwOTZlNWNkOGFhMmQzY2VjZmIxNmFlNzQ4NGFiOWVlNzlhNjU2NWQKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX2hlbHBfaXNfYmFubmVkCWNmZTg1MWQ3ZmUyNzljYmYyZTU1YzgxYzY4ZDY0OWJmNjViZjFiNTQxNWYxYjM1MzJjOTUyNDVlMmQ1YTJiMTMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF93cmFwcGVyX29wdGlvbl9vcGVyYW5kc19hcmVfc2tpcHBlZAlhMzQ0MzU4M2EzMGEzMmY5YzBjNDE0ZTA5ZmY5MDc4NTQ4NTUyYjEwNTkzZDczYTJiZDY2MmYzNmRlMjU1ZjIwCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfemVyb19tcndfY2FsbHNfY2Fubm90X21lZXQJNTZmMDYzODEyZDYyOGRmMzQyODA0ZmFiODg5NTg5YTY5NTQyY2E2NmE5MDAzYWE2YmYxMTA2ZjBhMzYyMDUzMw
  ```
  --- last 10 line(s) of stdout (of 47 after folding 49 raw)
      self.assertEqual(bs.score(d, t)["verdict"], "VOID")
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 'MEETS' != 'VOID'
  - MEETS
  + VOID
    (x2)
  ----------------------------------------------------------------------
  Ran 20 tests in 0.026s
  
  FAILED (failures=2)
  ```
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:866cbe7b218038afd8b26e88e91f51681fe34862d850aa3d46cfeb9e03935781 · ms:92
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:866cbe7b218038afd8b26e88e91f51681fe34862d850aa3d46cfeb9e03935781 · ms:86
- 2026-09-25 · a04ee52* · exit 0 · `set -o pipefail …` · acceptance-sha256:866cbe7b218038afd8b26e88e91f51681fe34862d850aa3d46cfeb9e03935781 · ms:105

## Mutation Log
(empty until execute)
- 2026-09-25 · a04ee52* · mutant killed · exit 1 · `scripts/blind-score.py` · the wrapper walk is refused: command env -S cat f hides cat again · acceptance-sha256:866cbe7b218038afd8b26e88e91f51681fe34862d850aa3d46cfeb9e03935781 · covers:expansion is reached behind a wrapper
- 2026-09-25 · a04ee52* · mutant killed · exit 1 · `scripts/blind-score.py` · the mask is one short again: an escaped dollar before backticks extracts backtick-cat instead of cat · acceptance-sha256:866cbe7b218038afd8b26e88e91f51681fe34862d850aa3d46cfeb9e03935781 · covers:the mask keeps its length

## Invariants

- The 16 replay trials score as before.
- The criterion is unchanged.

## Risks

- A wrapper the scorer does not know (`doas`, `chroot`) hides `env` behind it as before; extend `WRAPPERS` when a transcript shows one.

## Out of Scope

- Parsing every shell construct (permanent: boundary: the scorer models what readings 01–04 transcripts contain, pinned by tests)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
