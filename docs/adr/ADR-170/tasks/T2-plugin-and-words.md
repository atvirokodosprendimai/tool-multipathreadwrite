# Task ADR-170-T2: the opencode plugin enables it, and every place that teaches `mrw mcp` says so

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the plugin's `MRW_MCP=on`; the words in `mrw mcp --help`, README, AGENTS.md, `docs/agents.md`, CLAUDE.md
**Consumes:** `mrw mcp --enable` / `MRW_MCP=on` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the binary builds`, `the plugin builds and its smoke test passes with MCP off in its environment`, `the plugin sets MRW_MCP`, `the help says it`, `README says it`, `AGENTS.md says it`, `docs/agents.md says it`

## Goal

Decisions 3 and 4: the opencode plugin keeps working against any binary by setting `MRW_MCP=on` for its child; every place a caller reads about `mrw mcp` says the tools are off until enabled, and a shell-less registration example carries `--enable`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/opencode/mrw-plugin/src/index.ts` | edit | the `mcp` call's child environment gains `MRW_MCP=on` (`:80`, `run` at `:49`) |
| `cmd/opencode/mrw-plugin/test/smoke.test.mjs` | edit | the fixture sets `MRW_MCP=off` in its own environment, so only the plugin's setting can turn the tools on; binary selection is unchanged (`:14-16`, `:33-40`) |
| `cmd/mrw/main.go` | edit | `mrw mcp`'s Description |
| `README.md` | edit | the MCP section and its registration examples; the opencode "Or over MCP alone" recipe (`:615`); the release migration line |
| `AGENTS.md` | edit | "If you are reaching mrw over MCP" |
| `docs/agents.md` | edit | `:114`, which says a host with no shell runs `mrw mcp` |
| `CLAUDE.md` | edit | the note on this machine's deliberate registration: it needs the switch, the owner's to add |

## Ordered Steps

1. [S1] Set `MRW_MCP=off` in the smoke fixture's environment and see the smoke test RED against a binary built from T1 (the plugin's calls are refused). [proof: acceptance]
2. [S2] The plugin sets `MRW_MCP=on` for its child; the help, README, AGENTS.md, `docs/agents.md`, CLAUDE.md. Mutant: the plugin's `MRW_MCP` dropped. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go build -o bin/mrw ./cmd/mrw \
  && (cd cmd/opencode/mrw-plugin && npm ci --silent && npm run build --silent && npm test --silent) \
  && grep -qF 'MRW_MCP' cmd/opencode/mrw-plugin/src/index.ts \
  && h=$(./bin/mrw mcp --help 2>&1) && grep -qF -- '--enable' <<<"$h" \
  && grep -qF 'mrw mcp --enable' README.md \
  && grep -qF 'mrw mcp --enable' AGENTS.md \
  && grep -qF 'mrw mcp --enable' docs/agents.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the plugin's smoke test (`npm test`) is the test; it is not a Go test | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the plugin sets the variable |
| 2 — something selects it | opencode loads the plugin |
| 3 — the caller can discover it | README, AGENTS.md, `docs/agents.md`, `mrw mcp --help` |
| 4 — it is used | opencode sessions |

## Mutation Log
(empty until execute)
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `cmd/opencode/mrw-plugin/src/index.ts` · the plugin does not enable the tools for its child · acceptance-sha256:f09909a5b61a2695e60f87d0db7eab12a25e41424c44f84e6ffe4270508985fc

## Invariants

- The plugin's behaviour is unchanged apart from the variable.

## Risks

- None beyond ADR-170's.

## Stop Condition

The fence exits 0.

## Out of Scope

- Migrating this machine's own registration (permanent: boundary: it lives in the owner's account config outside this repository; CLAUDE.md names it)

## Verification Log
(empty until execute)
- 2026-10-23 · b5f2b277* · exit 1 · `set -o pipefail …` · acceptance-sha256:f09909a5b61a2695e60f87d0db7eab12a25e41424c44f84e6ffe4270508985fc · ms:9924 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  --- last 10 line(s) of stdout (of 70 after folding 70 raw)
    ...
  1..10
  # tests 10
  # suites 0
  # pass 10
  # fail 0
  # cancelled 0
  # skipped 0
  # todo 0
  # duration_ms 788.621416
  ```
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:f09909a5b61a2695e60f87d0db7eab12a25e41424c44f84e6ffe4270508985fc · ms:7390
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:f09909a5b61a2695e60f87d0db7eab12a25e41424c44f84e6ffe4270508985fc · ms:3726
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:f09909a5b61a2695e60f87d0db7eab12a25e41424c44f84e6ffe4270508985fc · ms:3321
- 2026-10-23 · b5f2b277* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f09909a5b61a2695e60f87d0db7eab12a25e41424c44f84e6ffe4270508985fc · ms:0 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz · test-lock-kind:replace
