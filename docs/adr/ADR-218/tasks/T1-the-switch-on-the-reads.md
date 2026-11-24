# Task ADR-218-T1: the switch on the reads

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `shownOff` (`cmd/mrw/shown218.go`), the one place `MRW_SHOWN` is read; the switched recording of `mrw read` and of `mrw hook read`; the read's stderr line; contract §385
**Consumes:** `mcp.LicensedNotShown` (`internal/mcp/licensed167.go`); `recordedAs` (`cmd/mrw/quiet167.go`)
**Proof map:** v1
**Rests-on:** `the two named tests passed with every named case and none skipped`, `contract §385 exists, names the switch and the script parses`, `the engine packages this task does not touch are unchanged`

## Goal

ADR-218 Decisions 1, 2, 3 and 6, on the recording side.

- **The read.** `MRW_SHOWN=off` makes a content read of the CLI record what `--quiet` records, print what it prints today, and say so in one stderr line when it served a line and the ledger was written. `on` or unset is today's behaviour; another value is ignored with a note.
- **The hook.** Under the switch `mrw hook read` records the lines a host Read returned as licensed and not shown, whatever the payload's `tool_use_id`. Without it, a payload with a `repl_` id records a showing as any other does.
- **Nothing detected.** `CLAUDE_CODE_EXPERIMENTAL_CODE_MODE` set, with `MRW_SHOWN` unset, changes nothing.
- **The boundary of Decision 1.** A showing recorded before the switched read, for the same version, is kept; after `mrw seen --forget` a switched read leaves none.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/shown218.go` | new | `shownOff`, the note for an unknown value, the stderr line's text |
| `cmd/mrw/main.go` | edit | the read's recording choice and its stderr line |
| `cmd/mrw/hookread191.go` | edit | the hook's recording choice |
| `scripts/contract.sh` | edit | §385 |
| `cmd/mrw/shown218_test.go` | new | the tests |

What selects it: the `read` action (`cmd/mrw/main.go:1681`, the one place a content read is recorded) and `hookRead` (`cmd/mrw/hookread191.go:140`, the one place the hook records). The tests run both commands in-process with the environment set, and §385 runs the built binary, so a switch that is parsed and never consulted fails both.

## Ordered Steps

1. [S1] §385 and the two tests first, RED on `main`: a read under `MRW_SHOWN=off` there records a showing. The read test's cases, each a named subtest: `read_bytes_equal`, `ledger_no_showing`, `stderr_line`, `on_is_plain`, `unset_is_plain`, `unknown_value_noted`, `earlier_showing_kept`, `forget_then_none`, `stat_says_nothing`, `if_changed_text_shows_nothing`, `code_mode_variable_not_read`. The hook test's: `switched_no_showing`, `switched_repl_id_no_showing`, `plain_shows`, `plain_repl_id_shows`.
2. [S2] `shownOff` and the read. Mutants, each killed: the switch never consulted by the read (`ledger_no_showing`); `on` treated as `off` (`on_is_plain`); an unknown value treated as `off` (`unknown_value_noted`); the Code Mode variable read (`code_mode_variable_not_read`). [proof: mutation]
3. [S3] The hook. Mutants: the switch never consulted by the hook (`switched_no_showing`); a `repl_` id refused without the switch (`plain_repl_id_shows`). [proof: mutation]
4. [S4] `bash -n scripts/contract.sh`, and `./scripts/contract.sh` with §385 passing. [proof: acceptance]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./cmd/mrw -run '^(TestAReadUnderShownOffLicensesAndShowsNothing|TestAHostReadUnderShownOffLicensesAndShowsNothing)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 2 ] && ! grep -q -- '--- SKIP' "$out" \
  && for c in TestAReadUnderShownOffLicensesAndShowsNothing/read_bytes_equal TestAReadUnderShownOffLicensesAndShowsNothing/ledger_no_showing TestAReadUnderShownOffLicensesAndShowsNothing/stderr_line TestAReadUnderShownOffLicensesAndShowsNothing/on_is_plain TestAReadUnderShownOffLicensesAndShowsNothing/unset_is_plain TestAReadUnderShownOffLicensesAndShowsNothing/unknown_value_noted TestAReadUnderShownOffLicensesAndShowsNothing/earlier_showing_kept TestAReadUnderShownOffLicensesAndShowsNothing/forget_then_none TestAReadUnderShownOffLicensesAndShowsNothing/stat_says_nothing TestAReadUnderShownOffLicensesAndShowsNothing/if_changed_text_shows_nothing TestAReadUnderShownOffLicensesAndShowsNothing/code_mode_variable_not_read TestAHostReadUnderShownOffLicensesAndShowsNothing/switched_no_showing TestAHostReadUnderShownOffLicensesAndShowsNothing/switched_repl_id_no_showing TestAHostReadUnderShownOffLicensesAndShowsNothing/plain_shows TestAHostReadUnderShownOffLicensesAndShowsNothing/plain_repl_id_shows; do grep -q "^    --- PASS: $c " "$out" || exit 1; done \
  && grep -q 'MRW_SHOWN' <(sed -n '/^# 385\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) && bash -n scripts/contract.sh \
  && base="$(git merge-base HEAD origin/main)" && git diff --quiet "$base" -- internal/read internal/plan internal/seen internal/check internal/state && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state)" ]
```

What each clause holds. Both tests pass, each of the fifteen named cases passes, none skipped. §385 exists, names the switch, and the script parses; that the row RAN is `./scripts/contract.sh` (S4), which the fence does not run. The engine packages this task does not touch are unchanged, test files included.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadUnderShownOffLicensesAndShowsNothing` | `cmd/mrw/shown218_test.go` | the eleven read cases, the ledger read back through `seen` | none | S1, S2 |
| `TestAHostReadUnderShownOffLicensesAndShowsNothing` | `cmd/mrw/shown218_test.go` | the four hook cases, on the payload ADR-191 captured, with and without a `repl_` id | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §385 |
| 2 — something selects it | the `read` action and `hookRead` |
| 3 — the caller can discover it | the stderr line on every switched read; T2's words |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #790 and the pilot |

## Mutation Log
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the switch never consulted by the read (ledger_no_showing) · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: on no longer read as plain (on_is_plain) · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: an unknown value treated as off (unknown_value_noted) · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: the Code Mode variable read (code_mode_variable_not_read) · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/hookread191.go` · S3: the switch never consulted by the hook (switched_no_showing) · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/hookread191.go` · S3: a repl_ id refused without the switch (plain_repl_id_shows) · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the switch never consulted by the read (ledger_no_showing) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: on no longer read as plain (on_is_plain) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: an unknown value treated as off (unknown_value_noted) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: the Code Mode variable read (code_mode_variable_not_read) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the note printed for a read that served no line (stat_says_nothing) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/hookread191.go` · S3: the switch never consulted by the hook (switched_no_showing) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/hookread191.go` · S3: a repl_ id refused without the switch (plain_repl_id_shows) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · 65764867* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the note printed though the ledger kept nothing (stderr_line, a newer ledger) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · 65764867* · mutant survived · exit 0 · `cmd/mrw/shown218.go` · S2: a file served no line recorded licence-only (stat_says_nothing) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-24 · 65764867* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the switch consulted by a read that served no line (stat_says_nothing, --json-path) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba
- 2026-11-24 · 65764867* · mutant killed · exit 1 · `cmd/mrw/shown218.go` · S2: a file served no line recorded licence-only (stat_says_nothing, a missed range beside a served one) · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba

## Verification Log
- 2026-11-24 · c0e9f381* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:586 · test-lock-sha256:929195a2296ffa053ed5d46fbfc0c716ea70fe88568eb134974832f0bf756fa5 · test-lock-file:docs/adr/locks/929195a2296ffa053ed5d46fbfc0c716ea70fe88568eb134974832f0bf756fa5.lock
  ```
  ```
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:2145
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:2182
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:2199
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:2208
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:2221
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:7f7288d05a08d3c27dc683a3d75598f7ba925e0d88515439564ff3fa43e18b55 · ms:2304
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:870d179281e9aafcb1423556d1da2d245ca32b21a1e7d236a35f5d19a6336dc4 · ms:622
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2459
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2378
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2118
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2173
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2331
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2719
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:3133
- 2026-11-24 · c0e9f381* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:9 · test-lock-sha256:dca53b6aa99560b6c391e365004f686781e48d9a0324dc410f1ba55a5c5c6386 · test-lock-file:docs/adr/locks/dca53b6aa99560b6c391e365004f686781e48d9a0324dc410f1ba55a5c5c6386.lock · test-lock-kind:replace
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:1384
- 2026-11-24 · 65764867* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:10 · test-lock-sha256:d44e444bbbe56216da6cf529b063f692d7ffb85cc06a05e97e868e522f4a882f · test-lock-file:docs/adr/locks/d44e444bbbe56216da6cf529b063f692d7ffb85cc06a05e97e868e522f4a882f.lock · test-lock-kind:replace
- 2026-11-24 · 65764867* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2236
- 2026-11-24 · 65764867* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2224
- 2026-11-24 · 65764867* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2181
- 2026-11-24 · 65764867* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:10 · test-lock-sha256:4b0366f2b66cdffa88d49e023b1eac4c7bff3302feceb7ebd9a2285b5e53944b · test-lock-file:docs/adr/locks/4b0366f2b66cdffa88d49e023b1eac4c7bff3302feceb7ebd9a2285b5e53944b.lock · test-lock-kind:replace
- 2026-11-24 · 65764867* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:06b1adb98f52ec9d98c4f06cac94ecfef93988e3840bf876dee91faf29a4c4ba · ms:2220

## Invariants

- Without the switch a read and the hook behave as on `main`.
- A switched read prints the same standard output as a plain one.
- `mrw_read`, `--quiet`, `--sub`, `--rewrite` and a read to the null device are unchanged.
- No receipt key is added; no exit code changes.

## Risks

- A host sets the variable for mrw and not for the hook process: measured for Claude Code's `env` setting, which reaches both (`docs/break/code-mode-790/env1.jsonl`).

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 with §385 passing and not skipped; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- The write side and the words (deferred: docs/adr/ADR-218/tasks/T2-the-switch-on-the-write-and-the-words.md)
- The switch on `mrw_read` (permanent: boundary: MCP's showing is an acknowledgement, ADR-108; ADR-218 Out of Scope)
