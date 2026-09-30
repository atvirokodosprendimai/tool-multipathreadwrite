# Task ADR-100-T1: Every `mrw check --json` refusal is one document

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `refuse` defined before every refusal in `checkCmd`'s Action, and the one exit for all of them; contract §193 and §187's no-step row; the BACKLOG entry "From ADR-100"
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every refusal is a document`, `without --json nothing changes`, `a contract row drives the binary`, `no engine package changes`

## Goal

Under `mrw check --json`, an argument with edge whitespace, `--full` with a PATH, an unreadable working set,
a refused scope, and a check whose log cannot be created with no step asked each print exactly one JSON
document on stdout, `{"error": "<message>"}`, with the same message on stderr, exit 2. Without `--json`
each prints nothing on stdout, as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `checkCmd`: `then` and `refuse` move to the top of the Action; `refusePaddedArgs` (`:2004`), the `--full` refusal (`:2013`), `iter.Load` (`:2018`) and `check.Run`'s error (`:2065-2069`) return through `refuse` |
| `cmd/mrw/json100_test.go` | add | the test below |
| `cmd/mrw/thenrefusal_test.go` | edit | `:112` pinned "no step asked" as no document; it now asserts one, and the comment at `:81-83` says so |
| `scripts/contract.sh` | edit | §193; §187's no-step row (`:7792-7793`) printed no document and now expects one holding the error, with no `then` and no `exit_code` |
| `docs/adr/BACKLOG.md` | edit | "From ADR-100": `write`'s pre-plan refusals and `stats`'s, under `--json` |

**What selects it:** `checkCmd`'s Action is the only reader of `--json` for `check`; `refuse` is the one
place that reads it for a refusal.

## Ordered Steps

1. [S1] Write the failing test `TestEveryCheckRefusalIsAJSONDocumentUnderJSON` and flip `thenrefusal_test.go:112`;
   confirm both RED on `7537718`. [proof: mutation]
2. [S2] Route the five refusals through `refuse`. [proof: mutation] Mutants: the `--full` refusal returns
   `cli.Exit` directly again; `check.Run`'s no-step branch returns `cli.Exit` directly again.
3. [S3] Contract §193, driving `$MRW`: `check --json ../outside`, `check --json nosuchdir` and
   `check --json --full x.go` each exit 2 with one document whose `error` names the cause and no
   `exit_code`; paired, the same without `--json` prints nothing on stdout; and the boundary,
   `check --json --then-sh='true '` refused in `main`, still prints nothing on stdout. §187's no-step row
   expects the document.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §193's rows printed — the fence greps the section, since the whole contract takes minutes]
4. [S4] BACKLOG "From ADR-100" names the two deferred surfaces with their sites. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestEveryCheckRefusalIsAJSONDocumentUnderJSON|TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestEveryCheckRefusalIsAJSONDocumentUnderJSON \(' "$out" \
  && grep -q '^# 193\. ' scripts/contract.sh \
  && grep -q '^## From ADR-100 ' docs/adr/BACKLOG.md \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal | grep '^??')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryCheckRefusalIsAJSONDocumentUnderJSON` | `cmd/mrw/json100_test.go` | through `runSplit`, each of `check --json 'x '`, `check --json --full x.go`, `check --json` over an unreadable working set, `check --json ../outside` and `check --json nosuchdir` exits 2 with stdout decoding as exactly one object whose `error` equals the returned error and which has no `exit_code`; each without `--json` exits 2 with empty stdout; and in a tree with no check, `check --json --full` exits 2 with exactly one document, `"ran": false` | — | S1, S2 |
| `TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps` | `cmd/mrw/thenrefusal_test.go` | (edited) the log refusal with no step asked is now a document carrying `error` and no `then` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `refuse` in `checkCmd` |
| 2 — something selects it | `checkCmd`'s Action; the tests run `rootCommand`, §193 the built binary |
| 3 — the caller can discover it | the document on stdout, the message on stderr, exit 2 |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 audit |

## Mutation Log
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the --full refusal bypasses refuse again: plain text under --json · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refused-scope and no-step branch returns plain text again · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the edge-whitespace refusal bypasses refuse · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · without --json a refusal also prints on stdout · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · covers:without --json nothing changes
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · covers:no engine package changes
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the --full refusal bypasses refuse again: plain text under --json · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refused-scope and no-step branch returns plain text again · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the edge-whitespace refusal bypasses refuse · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · without --json a refusal also prints on stdout · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · covers:without --json nothing changes
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `cmd/mrw/main.go` · the no-check exit prints a second document after the receipt · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · covers:every refusal is a document
- 2026-09-30 · 7537718* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · covers:no engine package changes

## Invariants

- Without `--json` every refusal's message, stream and exit code is unchanged.
- The `then` block appears only where ADR-092 already named steps not_run.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The depth refusal's turn — T2.
- `write` and `stats` (deferred: `docs/adr/BACKLOG.md` "From ADR-100")
- A refusal before any command parses its flags (permanent: boundary: ADR-100 Out of Scope; §193 pins it)

## Verification Log
- 2026-09-30 · 7537718* · exit 1 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:319 · test-lock-sha256:3b8a98f97d44beb570fb5105c881fef1d1c8ac903efd41cfd616f7ddc6b03b06 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanNvbjEwMF90ZXN0LmdvCVRlc3RFdmVyeUNoZWNrUmVmdXNhbElzQUpTT05Eb2N1bWVudFVuZGVySlNPTgk2OGFjNTAyNDVlOGVjMWU2YmIzOGI3Y2E2MGYwMjY2MDFjMDlmNjQ1ZGZmYzg0MDBlODFmYmE0NDRiNGIyYzNiCmJvZHkJY21kL21ydy9qc29uMTAwX3Rlc3QuZ28JVGVzdFRoZURlcHRoTGltaXRBbnN3ZXJzRXZlcnlDaGVja0Zvcm1GaXJzdAk2YzA2OGE4YzMyMjljYTEwZTUxZjA4YTJkYjRmNmY0ZDJiZWFhNjZkNmNmOTM3MGU2ZDZjNzE3NTYxNzMwOTA5CmJvZHkJY21kL21ydy90aGVucmVmdXNhbF90ZXN0LmdvCVRlc3RBQ2hlY2tXaG9zZUxvZ0Nhbm5vdEJlQ3JlYXRlZFN0aWxsTmFtZXNJdHNTdGVwcwkxZTUzYWU3ZGIxMmQ3NDc5NjE4ZjY4YmE5NGMyYjE5ZmYzYjM1ZDViYWEwZjNjOGU0ZWUwOWQ0M2FiNDg0ZGQ0CmJvZHkJY21kL21ydy90aGVucmVmdXNhbF90ZXN0LmdvCVRlc3RBTGVkZ2VyRmFpbHVyZVN0aWxsTmFtZXNUaGVTdGVwc05vdFJ1bgk4ZDRiNzU3ZTk1ZTJjZDlhMmVjNmE2NjgzZDJhYmI4MDM1MjFjZTIyMTE0Nzg1OGQ1ZDE2MWQ2ODhjNjg3YjY3
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
      json100_test.go:75: a path outside the root: stdout is not one document holding only the refusal "../outside resolves to /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestEveryCheckRefusalIsAJSONDocumentUnderJSON1074707356, which is outside the root /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestEveryCheckRefusalIsAJSONDocumentUnderJSON1074707356/002: check it with --root pointed where you mean":
      json100_test.go:75: a path not there: stdout is not one document holding only the refusal "nosuchdir is not there, so mrw cannot honour that scope: name a path that exists or run the whole-project check":
  --- FAIL: TestEveryCheckRefusalIsAJSONDocumentUnderJSON (0.00s)
  === RUN   TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps
      thenrefusal_test.go:114: no step asked: exit 2, want 2 and a document with an error and no then block:
          open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps2474876108/003/gone/mrw-check-1883512394.log: no such file or directory
  --- FAIL: TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.103s
  FAIL
  ```
- 2026-09-30 · 7537718* · exit 1 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:770
  ```
  --- last 6 line(s) of stdout
  === RUN   TestEveryCheckRefusalIsAJSONDocumentUnderJSON
  --- PASS: TestEveryCheckRefusalIsAJSONDocumentUnderJSON (0.00s)
  === RUN   TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps
  --- PASS: TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.206s
  ```
- 2026-09-30 · 7537718* · exit 1 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:302
  ```
  --- last 6 line(s) of stdout
  === RUN   TestEveryCheckRefusalIsAJSONDocumentUnderJSON
  --- PASS: TestEveryCheckRefusalIsAJSONDocumentUnderJSON (0.00s)
  === RUN   TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps
  --- PASS: TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.097s
  ```
- 2026-09-30 · 7537718* · exit 1 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:310
  ```
  --- last 6 line(s) of stdout
  === RUN   TestEveryCheckRefusalIsAJSONDocumentUnderJSON
  --- PASS: TestEveryCheckRefusalIsAJSONDocumentUnderJSON (0.00s)
  === RUN   TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps
  --- PASS: TestACheckWhoseLogCannotBeCreatedStillNamesItsSteps (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.116s
  ```
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:577
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:499
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:507
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:545
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c68a94211efa7a0902dab70836b5f76bfb96196db4a21a78f68b8c7ebd7e98 · ms:509
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:743
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:349
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:362
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:411
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:396
- 2026-09-30 · 7537718* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:525
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped on this branch (T1 and T2 applied, base 7537718), exit 0 'contract holds', with §193 printed: check --json ../outside, nosuchdir and --full a.go each one document holding the error and no exit_code, each plain form nothing on stdout, and the pre-parse boundary (--then-sh='true ') nothing on stdout; §187's no-step row PASS as one document with no then block
- 2026-09-30 · human-observed · relock 2026-09-30 (Codex cold review of the record, P2 'every refusal conflicts with the receipt emitted later'): after the red run, TestEveryCheckRefusalIsAJSONDocumentUnderJSON gained one assertion — in a tree with no check, check --json --full exits 2 with exactly one document, ran false — pinning the boundary the record now draws; every earlier assertion kept, and the new one killed the mutant that prints a second document after the receipt
- 2026-09-30 · 7537718* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:0 · test-lock-sha256:fb83d4aa450e373fbcb73da18898b9942b64a79a1040f0ead7e01a57a4825aa6 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanNvbjEwMF90ZXN0LmdvCVRlc3RFdmVyeUNoZWNrUmVmdXNhbElzQUpTT05Eb2N1bWVudFVuZGVySlNPTgk2YjJiMTEyMjJlMTY3OGM2NWMyNWIzMjM1NDgwYzE0NmMxOGFjODQ5MTVkMzQ3NjlhMjAyOWEyZDhmYzI3ZjMxCmJvZHkJY21kL21ydy9qc29uMTAwX3Rlc3QuZ28JVGVzdFRoZURlcHRoTGltaXRBbnN3ZXJzRXZlcnlDaGVja0Zvcm1GaXJzdAk2YzA2OGE4YzMyMjljYTEwZTUxZjA4YTJkYjRmNmY0ZDJiZWFhNjZkNmNmOTM3MGU2ZDZjNzE3NTYxNzMwOTA5CmJvZHkJY21kL21ydy90aGVucmVmdXNhbF90ZXN0LmdvCVRlc3RBQ2hlY2tXaG9zZUxvZ0Nhbm5vdEJlQ3JlYXRlZFN0aWxsTmFtZXNJdHNTdGVwcwkxZTUzYWU3ZGIxMmQ3NDc5NjE4ZjY4YmE5NGMyYjE5ZmYzYjM1ZDViYWEwZjNjOGU0ZWUwOWQ0M2FiNDg0ZGQ0CmJvZHkJY21kL21ydy90aGVucmVmdXNhbF90ZXN0LmdvCVRlc3RBTGVkZ2VyRmFpbHVyZVN0aWxsTmFtZXNUaGVTdGVwc05vdFJ1bgk4ZDRiNzU3ZTk1ZTJjZDlhMmVjNmE2NjgzZDJhYmI4MDM1MjFjZTIyMTE0Nzg1OGQ1ZDE2MWQ2ODhjNjg3YjY3 · test-lock-kind:replace
- 2026-09-30 · human-observed · Codex review of #288 (xhigh, REQUEST CHANGES, 3 P2s, all confirmed): §187's no-step row and §193's rows now require exactly one object whose only key is error via jq -se (probed: a missing brace, a trailing ], two run-together documents and an exit_code key are each refused); oneDocument now requires a second Decode to meet io.EOF, since More does not check end of input, pinned by TestOneDocumentRefusesAnythingAfterTheObject; README and AGENTS name the pre-parse refusals as the exception. No locked test body changed
- 2026-09-30 · d283dc2* · exit 0 · `set -o pipefail …` · acceptance-sha256:ecc4fa66a73436ac2e3e8cd61e338a0e4c06603da35bfbab18705a30a3edb802 · ms:382
