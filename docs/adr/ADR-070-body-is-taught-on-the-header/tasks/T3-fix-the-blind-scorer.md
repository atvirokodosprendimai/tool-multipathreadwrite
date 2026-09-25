# Task ADR-070-T3: The blind scorer reads every BACKLOG shape; minimum one mrw call

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** a scorer reading 05 can trust, and its unit tests
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the scorer reads every recorded shape`, `a zero-call run cannot meet`

## Goal

`scripts/blind-score.py` misreads the shapes BACKLOG `:301-319` lists. Fix each, pin each with a synthetic transcript in `scripts/test_blind_score.py` (one named test per shape), and add a minimum of one mrw call to `meets`. Before reading 05's first trial, pre-register that criterion change in BACKLOG. Re-score readings 03 and 04 with the fixed scorer, and add a line beginning `Re-scored with the ADR-070 scorer` to each result file saying whether any verdict moved.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/blind-score.py` | edit | each shape; `meets` requires ≥1 mrw call |
| `scripts/test_blind_score.py` | new | one synthetic transcript per shape |
| `docs/adr/BACKLOG.md` | edit | pre-registration; strike the scorer row with receipts |

## Ordered Steps

1. [S1] Write one unittest per shape with a synthetic stream-json transcript; confirm each RED. [proof: mutation]
   Shapes: `\`-continued newline; heredoc body lines; `command cat`; `cat` in `$(…)`; `env mrw`; a banned word quoted after `;`; a non-object final JSON fence; a wrongly typed answer; zero mrw calls with 8+ correct answers.
2. [S2] Fix the scorer; GREEN. [proof: mutation]
   Mutants: the heredoc skip removed; the ≥1-call clause removed.
3. [S3] Re-score readings 03 and 04; record the verdicts in their result files. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v 2>&1 | tee /tmp/adr070-T3.out \
  && missing=$(for t in test_backslash_newline_is_one_command test_heredoc_body_is_not_commands test_command_cat_is_banned test_cat_in_substitution_is_banned test_env_mrw_is_counted test_quoted_banned_word_is_not_a_violation test_non_object_final_fence_is_not_skipped test_mistyped_answer_does_not_crash test_zero_mrw_calls_cannot_meet; do grep -qE "^$t \(.*\) \.\.\. ok$" /tmp/adr070-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^OK' /tmp/adr070-T3.out \
  && grep -q 'at least one mrw call' docs/adr/BACKLOG.md \
  && grep -q '^Re-scored with the ADR-070 scorer' docs/blind/blind-03-result.md \
  && grep -q '^Re-scored with the ADR-070 scorer' docs/blind/blind-04-result.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_backslash_newline_is_one_command test_heredoc_body_is_not_commands test_command_cat_is_banned test_cat_in_substitution_is_banned test_env_mrw_is_counted test_quoted_banned_word_is_not_a_violation test_non_object_final_fence_is_not_skipped test_mistyped_answer_does_not_crash test_zero_mrw_calls_cannot_meet` | `scripts/test_blind_score.py` | one BACKLOG shape, pinned by a synthetic transcript | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unittest module |
| 2 — something selects it | reading 05 runs `blind-score.py` per trial |
| 3 — the caller can discover it | the scorer's JSON names each violation |
| 4 — it is used | readings 03/04 re-scored; reading 05 |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:0001b6b3a8393b8fbd69b12bb6a02c33ff5929fe8c3da622f52558f505b87c0a · ms:116 · test-lock-sha256:0bfe91359edaeb4951c5c10b3ce587d64801312d9d9c4f10f0337e849779cca9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2JhY2tzbGFzaF9uZXdsaW5lX2lzX29uZV9jb21tYW5kCWU4NTFmYjRiZDRjNTliZmI5MzJhNWY3NmM4NGY2ZWE4OGJlODdiNDFlZDNjY2VlMjU0ZWZhMzJjZTQxMDMzYmMKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZAkwMTA0NzhmMGJkNmQzOWVkZjE5OTFmNTU5NWUwZjY2YTA2OWFlNGRhODhiODVhZGNlMWMzNzVhMzFmYWJkZGY3CmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfY29tbWFuZF9jYXRfaXNfYmFubmVkCWRlYTBkNzQwMDI3YjIwZmY5M2YwM2Q4N2JjZjNmMjZhZjE2ZTVhNzhjODhhMmNiM2JiY2UyOTRmOGQ5ODRmN2YKYm9keQlzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9lbnZfbXJ3X2lzX2NvdW50ZWQJM2MzY2VlMzliMTU5MTFiMzUzNmEwYWJkOTk1NmU1ZjU2Y2FlMGVhMmJhOTEzNGM2YjNiMWRiOTFhYzQyNDkwOApib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X2hlcmVkb2NfYm9keV9pc19ub3RfY29tbWFuZHMJZDE3ODY1NTE2NzBlMjlkYWZmNTFmZTUzMGQ1MTQ3YzhhMzNiNWJhNWE2OTI3ODMxNzlkYzk4Mzk2ZjA0OWU0Ygpib2R5CXNjcmlwdHMvdGVzdF9ibGluZF9zY29yZS5weQl0ZXN0X21pc3R5cGVkX2Fuc3dlcl9kb2VzX25vdF9jcmFzaAk0NWVjNmI0MWYzMTIzMzY2MmZiZmQxOTExMjU4ZTc1ZGExNThlNjFiYmRmMDM3ZDJkMjY2Njk3N2VhY2QwMmFhCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3Rfbm9uX29iamVjdF9maW5hbF9mZW5jZV9pc19ub3Rfc2tpcHBlZAk4ZWI3NmQwYjQ5NzlkOWNhY2QwYWFhNjkzZjUwZWQyYzFkYjNjMmM3MDZiNGIyNGEyY2FhYTk0ZWJmMTJjNTdjCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfcXVvdGVkX2Jhbm5lZF93b3JkX2lzX25vdF9hX3Zpb2xhdGlvbgkzZmU2MzE0YmViMWE4YTBlZjY5MzZjNmFiZjc2NjYwODZmZmY1ODQ1ZDU0ZGE3YjIyZDhmNjhlMDU1YzM5N2EyCmJvZHkJc2NyaXB0cy90ZXN0X2JsaW5kX3Njb3JlLnB5CXRlc3RfemVyb19tcndfY2FsbHNfY2Fubm90X21lZXQJNTZmMDYzODEyZDYyOGRmMzQyODA0ZmFiODg5NTg5YTY5NTQyY2E2NmE5MDAzYWE2YmYxMTA2ZjBhMzYyMDUzMwp1bnByb3ZlbglzY3JpcHRzL3Rlc3RfYmxpbmRfc2NvcmUucHkJdGVzdF9iYWNrc2xhc2hfbmV3bGluZV9pc19vbmVfY29tbWFuZCB0ZXN0X2hlcmVkb2NfYm9keV9pc19ub3RfY29tbWFuZHMgdGVzdF9jb21tYW5kX2NhdF9pc19iYW5uZWQgdGVzdF9jYXRfaW5fc3Vic3RpdHV0aW9uX2lzX2Jhbm5lZCB0ZXN0X2Vudl9tcndfaXNfY291bnRlZCB0ZXN0X3F1b3RlZF9iYW5uZWRfd29yZF9pc19ub3RfYV92aW9sYXRpb24gdGVzdF9ub25fb2JqZWN0X2ZpbmFsX2ZlbmNlX2lzX25vdF9za2lwcGVkIHRlc3RfbWlzdHlwZWRfYW5zd2VyX2RvZXNfbm90X2NyYXNoIHRlc3RfemVyb19tcndfY2FsbHNfY2Fubm90X21lZXQ
  ```
  --- last 10 line(s) of stdout (of 113 after folding 113 raw)
  First extra element 2:
  'ls'
  
  - ['mrw', '@@', 'ls', 'grep', 'EOF', 'mrw']
  + ['mrw', 'mrw']
  
  ----------------------------------------------------------------------
  Ran 9 tests in 0.008s
  
  FAILED (failures=2, errors=7)
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0001b6b3a8393b8fbd69b12bb6a02c33ff5929fe8c3da622f52558f505b87c0a · ms:95
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0001b6b3a8393b8fbd69b12bb6a02c33ff5929fe8c3da622f52558f505b87c0a · ms:84

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `scripts/blind-score.py` · heredoc bodies are parsed as commands again: its test goes red · acceptance-sha256:0001b6b3a8393b8fbd69b12bb6a02c33ff5929fe8c3da622f52558f505b87c0a · covers:the scorer reads every recorded shape
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `scripts/blind-score.py` · the one-call minimum is removed: the zero-call test goes red · acceptance-sha256:0001b6b3a8393b8fbd69b12bb6a02c33ff5929fe8c3da622f52558f505b87c0a · covers:a zero-call run cannot meet

## Invariants

- A transcript none of the shapes touch scores exactly as before.
- The criterion otherwise stays ≥8/9 and ≤20 calls.

## Risks

- Re-scoring moves a past verdict: recorded in that reading's result file, not hidden.

## Out of Scope

- Running reading 05 (deferred: docs/adr/BACKLOG.md)

## Stop Condition

Stop if the change needs a plan-grammar change or an exit-code change.
