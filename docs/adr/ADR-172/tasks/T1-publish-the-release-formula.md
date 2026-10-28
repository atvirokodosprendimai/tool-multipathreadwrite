# Task ADR-172-T1: Publish only a trusted release formula and verify it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** Formula/mrw.rb in the org tap
**Consumes:** none
**Data dependency:** needs GitHub Actions and the tap PUSH_TOKEN for post-merge integration sign-off; the automated fence is hermetic
**Proof map:** v1

## Goal

A successful current release reaches Homebrew through an attested formula and a harmless retry.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `.github/workflows/brew-tap.yml` | edit | admit tag runs and manual repair, select the trusted publisher |
| `scripts/brew_tap.py` | add | authorize, render, validate, push and read back |
| `scripts/test_brew_tap.py` | add | real local Git remote plus deterministic GitHub/renderer fixtures |
| `.github/workflows/ci.yml` | edit | run the new regression suite |
| `.quality-harness.json` | edit | replace the grep-only step with the regression suite |
| `UPDATE.md` | edit | document Homebrew verification and manual repair |

## Ordered Steps

1. [S1] Write the asserting tests and record their red before implementing the publisher. [proof: acceptance]
2. [S2] Replace the main-only trigger with trusted-run checks and strict manual repair. [proof: mutation]
3. [S3] Pin the renderer, bind each URL/checksum to its OS/CPU and require complete current assets. [proof: mutation]
4. [S4] Publish through the credential helper, make retries harmless and compare served bytes. [proof: mutation]
5. [S5] After merge, dispatch the current tag once; record the Actions run, tap commit,
   four URL/checksum pairs and served version in the sign-off. The README stays partial
   until this succeeds. [proof: human: inspect the actual downstream Actions run and tap formula]

## Acceptance

```bash
python3 -m unittest discover -s scripts -p test_brew_tap.py -v \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal cmd/mrw
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_publish_and_retry` | `scripts/test_brew_tap.py` | publish and retry | — | S4 |
| `test_manual_replay_requires_release_ci` | `scripts/test_brew_tap.py` | manual replay requires release ci | — | S2 |
| `test_untrusted_runs_refused` | `scripts/test_brew_tap.py` | untrusted runs refused | — | S2 |
| `test_incomplete_or_older_release_refused` | `scripts/test_brew_tap.py` | incomplete or older release refused | — | S3 |
| `test_manifest_and_formula_pairs_refused` | `scripts/test_brew_tap.py` | manifest and formula pairs refused | — | S3 |
| `test_target_outside_os_scope_is_refused` | `scripts/test_brew_tap.py` | targets must remain within their real OS scope | — | S3 |
| `test_subprocess_stderr_is_bounded_and_redacted` | `scripts/test_brew_tap.py` | useful, bounded credential-safe failure details | — | S4 |
| `test_push_failure_not_success` | `scripts/test_brew_tap.py` | push failure not success | — | S4 |
| `test_readback_must_match` | `scripts/test_brew_tap.py` | readback must match | — | S4 |
| `test_api_failure_refused` | `scripts/test_brew_tap.py` | api failure refused | — | S4 |
| `test_workflow_reaches_publisher` | `scripts/test_brew_tap.py` | workflow reaches publisher | — | S2 |
| `test_ci_and_harness_select_tests` | `scripts/test_brew_tap.py` | ci and harness select tests | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | publisher tests execute the coordinator with a real local Git remote |
| 2 — something selects it | workflow and ci wiring tests fail if the caller or test step is removed |
| 3 — the caller can discover it | workflow_dispatch tag input and UPDATE.md repair instructions |
| 4 — it is used | post-merge Actions/tap sign-off; no end-user telemetry |

## Mutation Log
- 2026-10-28 · a683b3be* · mutant killed · exit 1 · `.github/workflows/brew-tap.yml` · S2: a main-only filter excludes the tag trigger · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f
- 2026-10-28 · a683b3be* · mutant killed · exit 1 · `scripts/brew_tap.py` · S2: a fork run is accepted · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f
- 2026-10-28 · a683b3be* · mutant killed · exit 1 · `scripts/brew_tap.py` · S3: an incorrect target checksum is accepted · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f
- 2026-10-28 · a683b3be* · mutant killed · exit 1 · `scripts/brew_tap.py` · S4: stale served bytes are reported as published · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f
- 2026-10-28 · a683b3be* · mutant killed · exit 1 · `scripts/brew_tap.py` · S3: restoring the old validator admits a target outside its OS block · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f
- 2026-10-28 · a683b3be* · mutant killed · exit 1 · `scripts/brew_tap.py` · S4: suppressing subprocess stderr hides the rejecting remote diagnostic · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f

## Invariants

- A failed downstream job does not alter the release.
- Only the latest stable tag with matching successful ci can be pushed.
- No token appears in the Git remote URL or subprocess arguments.

## Risks

- GitHub metadata and renderer doubles do not model authentication or live API semantics:
  the real-manifest render and post-merge Actions sign-off close those integration gaps.
- A latest-release change immediately after the final check can race the push;
  non-force publication and a later current-release retry remain the recovery.

## Stop Condition

Do not claim completion if the actual downstream job cannot publish and read back the current formula.
Do not replace credentials or change another tap formula to bypass a failed push.

## Out of Scope

- Other tap formulas (permanent: boundary: this publisher owns Formula/mrw.rb only).

## Verification Log
- 2026-10-28 · a683b3be* · exit 1 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1064 · test-lock-sha256:89647e86ff381c57cb87437b28759377fa1ba868e7291b9270a396be984a03ab · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfYXBpX2ZhaWx1cmVfcmVmdXNlZAk2Zjc1NzZkZTE0ZWRmYzYwMGJlNjAxOGNmZTg4ZmQ2ODlhNzczOTQ3YmIxMTMwZDI0YWM3M2E3ZmNmMWYzZTQ4CmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfY2lfYW5kX2hhcm5lc3Nfc2VsZWN0X3Rlc3RzCTQyYzYxNmE3ZTZkMTE4MjhhMjdhYjQyZWY1ZTgxMWU0ZDU3NDlhNTI0ZTQwNjUwYTI5NjcyZDYzYTk0ZDI4NTgKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9pbmNvbXBsZXRlX29yX29sZGVyX3JlbGVhc2VfcmVmdXNlZAlhM2Q0ODBiZWUxZGVhZWQ4ODMzZTc2M2UzYTc1M2MwNjFmNmQxNTdmNmY3OThmNmUyYjY4ZmZhM2RiMmM4ZDY1CmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfbWFuaWZlc3RfYW5kX2Zvcm11bGFfcGFpcnNfcmVmdXNlZAk3ZTQ4ZDJkOTU1N2ZjMWViODQyZjRkNzEzNGFiY2M3NjcwYTMwYTk2YTM4NzAwYzYxMTcwMzRkZjE3YTU0OTgwCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfbWFudWFsX3JlcGxheV9yZXF1aXJlc19yZWxlYXNlX2NpCWEzYmYzNzVkNThiNDMwYWFlNjk0MzY4OTBmYzQ4ZjdjNDU2Y2VjZTQ0MmI1ZTJmYWJkNDkxODk3Y2IxYWExMjAKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9wdWJsaXNoX2FuZF9yZXRyeQliYzlmN2E4YjQ3ZjA5MGIzMTMxYzI5N2M3ZjIyNjliOTA3MTYxMThiMTMxZjhkNGQ3NjI4YzRiYTk0MzkyNDg5CmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfcHVzaF9mYWlsdXJlX25vdF9zdWNjZXNzCWM1MDkzZTVkYjljYTVkYzkyYTlmNDMxNmQwMGVkYjMyYWQ5OTc2MGEzNTA3ZTE0YWM4NmMzNTc4ZTUzZTcxZGYKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9yZWFkYmFja19tdXN0X21hdGNoCTNmYjZhZmE4YmE2ODNjYWVkNjQ1MGJiOTZkOTdhODlkOGQzYTk4YzJhYWIwMDYxODg5YTFkNmViMjMyNzA2NzQKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF91bnRydXN0ZWRfcnVuc19yZWZ1c2VkCTQyNzRiMDZjYzMzNWY1YjFjNjg5MTZjMzI3ODgwOTk2ZDM3ODA2M2MyNTY4ZTcxOGM4NDY0MDRmMDExOWRlY2UKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF93b3JrZmxvd19yZWFjaGVzX3B1Ymxpc2hlcgk1ZDIyNDFmYmVkNjg0YjFmNzQzZGZmMGRhN2I1ZWMwODNlMWMyYzYxZTJkMzFmMzQzZWMzMzdmYWNiYzdjOTNk
  ```
  --- last 10 line(s) of stderr (of 60 after folding 60 raw)
  Traceback (most recent call last):
    File "/private/tmp/mrw-cleanup.HLGaZQ/pr528/scripts/test_brew_tap.py", line 204, in test_workflow_reaches_publisher
      self.assertNotIn("branches:", trigger)
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 'branches:' unexpectedly found in "name: brew-tap\n\n# ADR-172: render the Homebrew formula from the release manifest and push it\n# to the org tap, so `brew install atvirokodosprendimai/tap/mrw` (and\n# `brew upgrade`) always serve the released archives. Runs only after a\n# strict version tag's ci run succeeds AND the Release has assets — the tap\n# is a downstream of the Release, not a gate on it; a failed push must not\n# un-publish a tagged release.\n#\n# The generator is vendored from the skills repo at run time (ADR-172: one\n# copy of the convention; this repo keeps no renderer to drift). The formula\n# is generated, never hand-edited: its shas come from SHA256SUMS.txt, the\n# manifest a curl-based installer verifies against.\n\non:\n  workflow_run:\n    workflows: [ci]\n    types: [completed]\n    branches: [main]\n\n"

  ----------------------------------------------------------------------
  Ran 10 tests in 0.924s

  FAILED (failures=5)
  ```
- 2026-10-28 · a683b3be* · exit 0 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1463
- 2026-10-28 · a683b3be* · exit 0 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1410
- 2026-10-28 · a683b3be* · exit 0 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1694
- 2026-10-28 · a683b3be* · exit 0 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1707
- 2026-10-28 · a683b3be* · exit 1 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1817
  ```
  --- last 10 line(s) of stderr (of 35 after folding 35 raw)
  Traceback (most recent call last):
    File "/private/tmp/mrw-cleanup.HLGaZQ/pr528/scripts/test_brew_tap.py", line 226, in test_target_outside_os_scope_is_refused
      self.assertEqual(self.invoke("--run-id", "123"), 1)
      \~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: 0 != 1

  ----------------------------------------------------------------------
  Ran 12 tests in 1.711s

  FAILED (failures=2)
  ```
- 2026-10-28 · a683b3be* · exit 0 · `adr-verify --relock` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:0 · test-lock-sha256:8c23a5b67364592650eb700640ce602a851e579c53d01cbb28b840459e806aa3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfYXBpX2ZhaWx1cmVfcmVmdXNlZAk2Zjc1NzZkZTE0ZWRmYzYwMGJlNjAxOGNmZTg4ZmQ2ODlhNzczOTQ3YmIxMTMwZDI0YWM3M2E3ZmNmMWYzZTQ4CmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfY2lfYW5kX2hhcm5lc3Nfc2VsZWN0X3Rlc3RzCTQyYzYxNmE3ZTZkMTE4MjhhMjdhYjQyZWY1ZTgxMWU0ZDU3NDlhNTI0ZTQwNjUwYTI5NjcyZDYzYTk0ZDI4NTgKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9pbmNvbXBsZXRlX29yX29sZGVyX3JlbGVhc2VfcmVmdXNlZAlhM2Q0ODBiZWUxZGVhZWQ4ODMzZTc2M2UzYTc1M2MwNjFmNmQxNTdmNmY3OThmNmUyYjY4ZmZhM2RiMmM4ZDY1CmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfbWFuaWZlc3RfYW5kX2Zvcm11bGFfcGFpcnNfcmVmdXNlZAk3ZTQ4ZDJkOTU1N2ZjMWViODQyZjRkNzEzNGFiY2M3NjcwYTMwYTk2YTM4NzAwYzYxMTcwMzRkZjE3YTU0OTgwCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfbWFudWFsX3JlcGxheV9yZXF1aXJlc19yZWxlYXNlX2NpCWEzYmYzNzVkNThiNDMwYWFlNjk0MzY4OTBmYzQ4ZjdjNDU2Y2VjZTQ0MmI1ZTJmYWJkNDkxODk3Y2IxYWExMjAKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9wdWJsaXNoX2FuZF9yZXRyeQliYzlmN2E4YjQ3ZjA5MGIzMTMxYzI5N2M3ZjIyNjliOTA3MTYxMThiMTMxZjhkNGQ3NjI4YzRiYTk0MzkyNDg5CmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfcHVzaF9mYWlsdXJlX25vdF9zdWNjZXNzCWM1MDkzZTVkYjljYTVkYzkyYTlmNDMxNmQwMGVkYjMyYWQ5OTc2MGEzNTA3ZTE0YWM4NmMzNTc4ZTUzZTcxZGYKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9yZWFkYmFja19tdXN0X21hdGNoCTNmYjZhZmE4YmE2ODNjYWVkNjQ1MGJiOTZkOTdhODlkOGQzYTk4YzJhYWIwMDYxODg5YTFkNmViMjMyNzA2NzQKYm9keQlzY3JpcHRzL3Rlc3RfYnJld190YXAucHkJdGVzdF9zdWJwcm9jZXNzX3N0ZGVycl9pc19ib3VuZGVkX2FuZF9yZWRhY3RlZAkwMDk1ODM2ODM5Y2IzY2NhZTk1OGVmMGU3Mjg4NzU2YWM4MWZkNGIxNWRkMmViZWI1MjVhMzUyMzQ3YjRjNzBjCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfdGFyZ2V0X291dHNpZGVfb3Nfc2NvcGVfaXNfcmVmdXNlZAkwOWVkZTc3ZTRkMjUwYzIwZjY4OWFlNDE0Mzg4ZTRjZTkzODQzNTc0Yjk1ZjI0YzI3NWQxMGU3ZWRlMDI0YTVhCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3RfdW50cnVzdGVkX3J1bnNfcmVmdXNlZAk0Mjc0YjA2Y2MzMzVmNWIxYzY4OTE2YzMyNzg4MDk5NmQzNzgwNjNjMjU2OGU3MThjODQ2NDA0ZjAxMTlkZWNlCmJvZHkJc2NyaXB0cy90ZXN0X2JyZXdfdGFwLnB5CXRlc3Rfd29ya2Zsb3dfcmVhY2hlc19wdWJsaXNoZXIJNWQyMjQxZmJlZDY4NGIxZjc0M2RmZjBkYTdiNWVjMDgzZTFjMmM2MWUyZDMxZjM0M2VjMzM3ZmFjYmM3YzkzZA · test-lock-kind:relock
- 2026-10-28 · a683b3be* · exit 0 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1664
- 2026-10-28 · a683b3be* · exit 0 · `python3 -m unittest discover -s scripts -p test_brew_tap.py -v \ …` · acceptance-sha256:72c246de9859bb721e3d838b985b3bf1561fef6600ab7ba3f38dba08bd22307f · ms:1668
