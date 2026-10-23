# Task ADR-163-T3: the baseline, the surviving pool and the block plan

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `bench baseline`, `baseline.json`, the surviving pool as a selection-shaped file (`bench baseline --surviving FILE`), blocks drawn from it, `export-paired --pair A,W` against it
**Consumes:** the container and the shims (T1), the declared check, `checks.json` and the bound export fields (T2)
**Data dependency:** hermetic tests against a fake `docker`
**Proof map:** v1
**Rests-on:** `a baseline runs the declared check on the unmodified tree, with no model`, `a stopped baseline keeps its rows and resumes without rerunning one`, `a check that never started keeps its reason`, `the surviving pool is the green instances in pool order, frozen`, `a block is a slice of the surviving pool`, `the rows export for W against the surviving pool`, `a capped run exports unresolved with no done claim`

## Goal

`bench baseline` records, per instance, whether its declared check is green on the unmodified tree, and writes the green ones in ADR-162's pool order as the surviving pool. Blocks are 40-instance slices of it, and `export-paired` exports against it (ADR-163 Decisions 4 and 5).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/baseline.go` | add | per instance: pull the image if absent, start the container (T1), run the declared check once, record green or red, exit code and wall time; drop the image as the runner does; `--surviving FILE` writes the green ids in pool order as a selection |
| `internal/bench/runner.go` | edit | `--block K` over a selection whose ids are the surviving pool; the last block may be partial |
| `internal/bench/export.go` | edit | `instance_list_sha256` and `instance_count` from the selection given, its `.selected` ids joined by a newline with no trailing newline, so the surviving pool's digest is what the rows carry; under `stage6` a capped run is written `not done`, unresolved, `limit` true |
| `internal/bench/baseline_test.go` | add | the tests below |
| `cmd/bench/main.go` | edit | `bench baseline`; the usage text |
| `docs/bench/README.md` | edit | the baseline, the surviving pool and its blocks |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. [proof: acceptance]
2. [S2] `bench baseline --stage S --out DIR --selection FILE --checks FILE [--surviving FILE]`: one container per instance, the check on the unmodified tree once, `baseline.json` with `{instance, green, exit_code, wall_ms}` per instance; a check that cannot start is red, with its reason; `--surviving` writes the green ids in the selection's order, with the rule, the source selection's sha and the red count. [proof: mutation]
3. [S3] `bench run --selection selection-adr-163.json --block K` runs instances 40(K−1)+1 to 40K, the last block partial; `export-paired` takes the surviving pool's digest (the scorer's convention) and count from the selection, and under `stage6` writes a capped run as unresolved with no done claim. [proof: mutation]
4. [S4] `export-paired --pair A,W` is tested; the usage text and `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestTheBaselineRecordsGreenRedAndTime|TestABaselineThatStopsKeepsItsRowsAndResumes|TestABaselineWhoseCheckNeverStartedKeepsItsReason|TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder|TestExportPairedTakesAWAgainstTheSurvivingPool|TestACappedRunExportsUnresolvedWithNoDoneClaim)$' 2>&1) \
  && for t in TestTheBaselineRecordsGreenRedAndTime TestABaselineThatStopsKeepsItsRowsAndResumes TestABaselineWhoseCheckNeverStartedKeepsItsReason TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder TestExportPairedTakesAWAgainstTheSurvivingPool TestACappedRunExportsUnresolvedWithNoDoneClaim; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'bench baseline'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheBaselineRecordsGreenRedAndTime` | `internal/bench/baseline_test.go` | a check exiting 0 is green, non-zero red with its exit code, a container that cannot start red with its reason; the wall time is recorded; each instance runs once; no model is invoked | none | S2 |
| `TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder` | `internal/bench/baseline_test.go` | with reds interleaved, the surviving file holds exactly the greens in pool order with the red count and no red id; blocks 1 and 2 are its first and second 40, a last block of fewer is accepted | none | S2, S3 |
| `TestExportPairedTakesAWAgainstTheSurvivingPool` | `internal/bench/baseline_test.go` | `--pair A,W` writes arms A and W; `instance_list_sha256` equals the sha256 of the surviving `.selected` ids joined by a newline with no trailing newline, `instance_count` their number, not ADR-162's 191 | none | S3, S4 |
| `TestACappedRunExportsUnresolvedWithNoDoneClaim` | `internal/bench/baseline_test.go` | under `stage6`, a receipt over its cap with `STATUS: done` and `harness_resolved` true exports `not done`, unresolved, `limit` true; under `stage5` the row is today's | none | S3 |
| `TestABaselineThatStopsKeepsItsRowsAndResumes` | `internal/bench/baseline_test.go` | an image that will not drop after a finished check stops the stage with that row kept in `baseline.partial.json`; a rerun skips it and completes, writing `baseline.json` (the Codex review of #518) | none | S2 |
| `TestABaselineWhoseCheckNeverStartedKeepsItsReason` | `internal/bench/baseline_test.go` | a testbed that does not activate is red, exit 126, with a reason; 125 and 127 the same (the Codex review of #518) | none | S2 |

Planned tests, which this table names in the commit that adds them:

- TestTheBaselineRecordsGreenRedAndTime in internal/bench/baseline_test.go: a check exiting 0 is green, non-zero red, one that cannot start red with its reason; the wall time is recorded; no model is invoked; each instance runs once (steps S2)
- TestTheSurvivingPoolIsTheGreenInstancesInPoolOrder in internal/bench/baseline_test.go: with reds interleaved, the surviving file holds exactly the greens in pool order with the red count; blocks 1 and 2 are its first and second 40, a last block of fewer is accepted (steps S2, S3)
- TestExportPairedTakesAWAgainstTheSurvivingPool in internal/bench/baseline_test.go: `--pair A,W` writes arms A and W; the config line's `instance_list_sha256` equals the sha256 of the surviving `.selected` ids joined by a newline with no trailing newline, and its `instance_count` their number, not ADR-162's 191 (steps S3, S4)
- TestACappedRunExportsUnresolvedWithNoDoneClaim in internal/bench/baseline_test.go: under `stage6`, a receipt over its cap with `STATUS: done` and `harness_resolved` true exports `not done`, unresolved, `limit` true; under `stage5` the row is today's (steps S3)

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `bench baseline` in `cmd/bench/main.go`; `run --selection` with the surviving file |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's baselines and T5's blocks |

## Mutation Log
(empty until execute)
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/baseline.go` · a check that exits 0 is recorded red · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/baseline.go` · the surviving pool holds the red instances · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · 9040f08b* · mutant survived · exit 0 · `internal/bench/baseline.go` · an existing baseline is overwritten · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/baseline.go` · a check that could not start is recorded with exit code 0 · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/baseline.go` · an existing baseline is run again · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/export.go` · a capped stage6 run exports done and resolved · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/export.go` · stage5 capped rows change too · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · 9040f08b* · mutant killed · exit 1 · `internal/bench/baseline.go` · a block holds 41 instances · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · c73aac2b · mutant killed · exit 1 · `internal/bench/export.go` · a capped stage6 run keeps its done claim · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · c73aac2b* · mutant killed · exit 1 · `internal/bench/baseline.go` · the surviving file says no instance was red · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · c73aac2b* · mutant killed · exit 1 · `internal/bench/baseline.go` · a rerun runs a finished instance again · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de
- 2026-10-23 · c73aac2b* · mutant killed · exit 1 · `internal/bench/baseline.go` · finished rows are not saved before the stage stops · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de
- 2026-10-23 · c73aac2b* · mutant killed · exit 1 · `internal/bench/baseline.go` · the partial file stays after a complete run · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de
- 2026-10-23 · c73aac2b* · mutant killed · exit 1 · `internal/bench/baseline.go` · a check that never started loses its reason · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de

## Invariants

- A baseline invokes no model.
- No instance enters the surviving pool, or a block, by any rule but ADR-163 Decision 4's.

## Risks

- A flaky check is green on one run and red on another; one run decides, and T4 reports instances whose check took over a quarter of the budget.

## Stop Condition

The fence exits 0.

## Out of Scope

- Re-running a red baseline to make it green (permanent: boundary: one run decides, so no instance is chosen for an outcome)

## Verification Log
(empty until execute)
- 2026-10-23 · 9040f08b* · exit 1 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:158 · test-lock-sha256:4774ffb28af7a8a7f689fbc25405c3408b76f00c0f6b891c483a0f59c95b2f8a · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0QUNhcHBlZFJ1bkV4cG9ydHNVbnJlc29sdmVkV2l0aE5vRG9uZUNsYWltCTQ0YmQyOGM2MDI4YmNkYWNiZDIxZTA2OTAzNGVjYjM2MzkzZDE2OGMzZDFjZGQ3YTc0MTQ4NzBhOTQwZWFkZTAKYm9keQlpbnRlcm5hbC9iZW5jaC9iYXNlbGluZV90ZXN0LmdvCVRlc3RFeHBvcnRQYWlyZWRUYWtlc0FXQWdhaW5zdFRoZVN1cnZpdmluZ1Bvb2wJN2Y3OGIxYmExNGI1NjkwOWQxY2VhMmUxYThmMjRlOWM3ZWZmOWU1MjUwMzg0YmEzZjYwY2VkZTQ1NjU1OWU5MQpib2R5CWludGVybmFsL2JlbmNoL2Jhc2VsaW5lX3Rlc3QuZ28JVGVzdFRoZUJhc2VsaW5lUmVjb3Jkc0dyZWVuUmVkQW5kVGltZQljNWNjYzFjM2Y1MjhjZGIxMGVhZjE3Y2FmY2Y1ZjA4NTI3MGQ1YzhmMmM0NDc4YmUyOGY2OGVmOTVmOGJmYmViCmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0VGhlU3Vydml2aW5nUG9vbElzVGhlR3JlZW5JbnN0YW5jZXNJblBvb2xPcmRlcglmNzcwOWMyY2IxNjQ0MGEzZWZlMzU1YmYxZTdlYTA1N2U0YTM3MGZhMmQzOTM4Njg0OWNiOTJiMDdmOTcyYzA5
  ```
  ```
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:5974
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:6653
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:6758
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:6743
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:6787
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:7387
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:7271
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:7559
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:7025
- 2026-10-23 · 9040f08b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:15653
- 2026-10-23 · 9040f08b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:0 · test-lock-sha256:2cfab6e9871f618d487672e44106ac8dd2301977247c1ff9a0dd0eb5aa83e31b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0QUNhcHBlZFJ1bkV4cG9ydHNVbnJlc29sdmVkV2l0aE5vRG9uZUNsYWltCTQ0YmQyOGM2MDI4YmNkYWNiZDIxZTA2OTAzNGVjYjM2MzkzZDE2OGMzZDFjZGQ3YTc0MTQ4NzBhOTQwZWFkZTAKYm9keQlpbnRlcm5hbC9iZW5jaC9iYXNlbGluZV90ZXN0LmdvCVRlc3RFeHBvcnRQYWlyZWRUYWtlc0FXQWdhaW5zdFRoZVN1cnZpdmluZ1Bvb2wJN2Y3OGIxYmExNGI1NjkwOWQxY2VhMmUxYThmMjRlOWM3ZWZmOWU1MjUwMzg0YmEzZjYwY2VkZTQ1NjU1OWU5MQpib2R5CWludGVybmFsL2JlbmNoL2Jhc2VsaW5lX3Rlc3QuZ28JVGVzdFRoZUJhc2VsaW5lUmVjb3Jkc0dyZWVuUmVkQW5kVGltZQk2YzNlY2Q4ZDMzZGM3MWFhMTNlMjcwNWUxNGFiMDhkZTBjOTA1N2NmYzZmYTIzMDBjZWE2ZmE1NDVjY2U3ZWE5CmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0VGhlU3Vydml2aW5nUG9vbElzVGhlR3JlZW5JbnN0YW5jZXNJblBvb2xPcmRlcglmNzcwOWMyY2IxNjQ0MGEzZWZlMzU1YmYxZTdlYTA1N2U0YTM3MGZhMmQzOTM4Njg0OWNiOTJiMDdmOTcyYzA5 · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed: TestTheBaselineRecordsGreenRedAndTime's second-baseline case now requires the 'holds a baseline already' refusal and no second run of the check, after the mutant that disabled the refusal survived the first version (it accepted any error). No assertion removed or weakened; approved · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae
- 2026-10-23 · c73aac2b · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:7076
- 2026-10-23 · c73aac2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:caa3c2608e17c7b21ac349eabcdaa58ac1717ae67883ba96349670da3c0f54ae · ms:7076
- 2026-10-23 · c73aac2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de · ms:7511
- 2026-10-23 · c73aac2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de · ms:20392
- 2026-10-23 · c73aac2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de · ms:13041
- 2026-10-23 · c73aac2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de · ms:11089
- 2026-10-23 · c73aac2b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de · ms:8391
- 2026-10-23 · c73aac2b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de · ms:0 · test-lock-sha256:db2ecbb2745b1acf6c9ae05161c3cd18b14e90f0a48e39148f0fe70f61f487ff · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0QUJhc2VsaW5lVGhhdFN0b3BzS2VlcHNJdHNSb3dzQW5kUmVzdW1lcwk3NWQzMzRlZWQyYjc1MTk1MjJjMDZiODdmY2EzYjhjYTVkYThlNTY3YTZjZTYwZGQxNzdiNjljNjYzN2U3OTAwCmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0QUJhc2VsaW5lV2hvc2VDaGVja05ldmVyU3RhcnRlZEtlZXBzSXRzUmVhc29uCTExZTA5M2UwZmZhZDhiZTVkMTZjMzA1Yjk5NTlkNDBlMjRjN2M3YzIyODNlOGFmZjZiYzBhMTE2YWU5MTI1Y2IKYm9keQlpbnRlcm5hbC9iZW5jaC9iYXNlbGluZV90ZXN0LmdvCVRlc3RBQ2FwcGVkUnVuRXhwb3J0c1VucmVzb2x2ZWRXaXRoTm9Eb25lQ2xhaW0JNDRiZDI4YzYwMjhiY2RhY2JkMjFlMDY5MDM0ZWNiMzYzOTNkMTY4YzNkMWNkZDdhNzQxNDg3MGE5NDBlYWRlMApib2R5CWludGVybmFsL2JlbmNoL2Jhc2VsaW5lX3Rlc3QuZ28JVGVzdEV4cG9ydFBhaXJlZFRha2VzQVdBZ2FpbnN0VGhlU3Vydml2aW5nUG9vbAk3Zjc4YjFiYTE0YjU2OTA5ZDFjZWEyZTFhOGYyNGU5YzdlZmY5ZTUyNTAzODRiYTNmNjBjZWRlNDU2NTU5ZTkxCmJvZHkJaW50ZXJuYWwvYmVuY2gvYmFzZWxpbmVfdGVzdC5nbwlUZXN0VGhlQmFzZWxpbmVSZWNvcmRzR3JlZW5SZWRBbmRUaW1lCTZjM2VjZDhkMzNkYzcxYWExM2UyNzA1ZTE0YWIwOGRlMGM5MDU3Y2ZjNmZhMjMwMGNlYTZmYTU0NWNjZTdlYTkKYm9keQlpbnRlcm5hbC9iZW5jaC9iYXNlbGluZV90ZXN0LmdvCVRlc3RUaGVTdXJ2aXZpbmdQb29sSXNUaGVHcmVlbkluc3RhbmNlc0luUG9vbE9yZGVyCWY3NzA5YzJjYjE2NDQwYTNlZmUzNTViZjFlN2VhMDU3ZTRhMzcwZmEyZDM5Mzg2ODQ5Y2I5MmIwN2Y5NzJjMDk · test-lock-kind:replace
- 2026-10-23 · human-observed · relock and fence change 2026-10-23 reviewed, after Codex on #518: baseline.json is a top-level array (T4's reader); finished rows are kept in baseline.partial.json and a rerun skips them; exit 125/126/127 carry the last stderr line as the reason. The fence adds TestABaselineThatStopsKeepsItsRowsAndResumes and TestABaselineWhoseCheckNeverStartedKeepsItsReason (red against the old baseline.go: no partial file; the object format); the fake docker gains fail-rm. T4's jq clauses that need no external file were run by hand on an emitted baseline.json and pass; T4 untouched. No assertion removed or weakened; approved · acceptance-sha256:1cbc2619556a6684fca19a11a443e2103d88270e386d6d540d554ad46b8f72de
