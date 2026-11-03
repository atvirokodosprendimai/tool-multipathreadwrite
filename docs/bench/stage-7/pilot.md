# Stage 7: the re-probe, the canary, the preflight and the uncounted pilot (ADR-192 T3)

Run 2026-11-03 on this Mac, before any counted attempt. Nothing here is counted: the receipts' stage begins `pilot`
and none sits in a block.

cli: 2.1.317 (Claude Code)
mrw: v1.83.0 (3a97083), the release asset, checksum verified against `SHA256SUMS.txt`
sentence: In this session, read files with the Read tool, not with mrw read, cat or grep: the lines a Read shows you count as read for mrw, so a plan may then edit them with no mrw read. Edit and create files only with mrw write.
sentence sha256: 20c9a6c319bb995d57995ee653d38ed7939c1f75a71c91ce23485f8720b50d0d

## S2: the re-probe

The first stage7 canary on 2.1.317 failed A, B and H alike on `plugins [cc-plugin-privacy], want []`: a built-in
plugin the binary carries (`cc-plugin-privacy@builtin` in its init line) that 2.1.302's probe did not list. The
skills were 2.1.302's, `design` and `doctor`. `builtinPlugins` in `internal/bench/profile.go` now switches it off
by name with the other three (`TestTheProfileSwitchesOffEveryBuiltinPlugin`).

## S3: the canary

`canary-claude-sonnet-5-5.json` and `canary-claude-opus-5-5.json`: stage7, A, B and H, pass, no problems, run with
the frozen sentence. H's init tools held `Read`, its transcript a `PostToolUse:Read` hook event (the name T1 assumed),
`hook.fired` one line per Read and `hook.stderr` empty.

## S4: the preflight

A scratch git checkout, Sonnet, CLI 2.1.317, mrw v1.83.0 on PATH, a fresh `XDG_STATE_HOME`, the stage's hook command
(`'<mrw>' hook read 2>>hook.stderr; echo fired >>hook.fired`, matcher `Read`). The agent Read `notes.txt`, then ran
`printf '@@ notes.txt 2 replace\ngamma\n' | mrw write -` with no `mrw read` between.

- With the hook: `1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied`; line 2 became `gamma`; one hook run,
  empty stderr.
- With the hook removed: `notes.txt 2 replace (plan line 1): notes.txt has not been read … NOTHING WRITTEN`; the
  file unchanged.

Cost $0.24 and $0.22.

## S5: the pilot

H alone on the first 2 instances of `bench/swe-bench/selection.json` (`matplotlib__matplotlib-22719`,
`astropy__astropy-13033`; neither is in the 191-instance pool), both models, one attempt each.

| Round | Sentence | Attempts with a host Read | host_reads | hook_runs | hook stderr bytes | solved | cost |
|---|---|---|---|---|---|---|---|
| 1 (`pilot-1/`) | "In this session you may also read files with the Read tool: …" (ADR-192 as accepted) | 0 of 4 | 0 | 0 | 0 | 3 of 4 | $0.33 |
| 2 (`pilot-2/`) | the frozen sentence above | 4 of 4 | 9 | 9 | 0 | 3 of 4 | $0.47 |

In round 1 every read was `mrw read` through Bash, as the first rule of `mrw instructions` says ("Use mrw for every
file read"), which "you may also" did not override. Round 2's sentence tells the agent to read with the Read tool; both
models did, every Read ran the hook, and the hook wrote nothing to standard error. `scripts/bench_mechanism.py` counts
0 hook-licensed writes in round 2 (8 writes unknown): its proxy recognises a plan only in some shapes, and zero is not
evidence either way (ADR-192 Decision 4); the preflight is the evidence that the hook licenses. Two instances say little
about uptake over 191: the uptake gate in block 1 is the counted check.
