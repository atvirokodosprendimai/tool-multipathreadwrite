# Task ADR-162-T1: configuration P and the Stage 5 profile

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configuration P, `bench run --profile stage5 --plugin-dir --node` (the Codex stub written by the profile, not a flag), the canary per configuration, the receipt's and `run.json`'s `plugin: {tag, commit}`
**Consumes:** none
**Data dependency:** hermetic, except S1's one uncounted canary probe on Sonnet (a real P transcript the canary rule is written against)
**Proof map:** v1
**Rests-on:** `P is A plus the plugin`, `the profile is the stage's, not the arms'`, `the canary proves engagement and nothing synced`, `no configuration reaches an external model`, `the usage text names --plugin-dir`

## Goal

`bench run --profile stage5 --configs A,B,P` runs a third configuration that is A with the quality-harness plugin loaded, inside a profile that every configuration and every block of the stage shares, and the canary proves P engaged and nothing else loaded (ADR-162 Decisions 1 and 2).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go` | edit | `Configs["P"]`; the Stage 5 profile in `RunOptions` and `run.json`; `invocation`: `--plugin-dir` for P, and under the profile for every configuration no `--disable-slash-commands`, `Skill` offered, `--node` linked on PATH, `CODEX_CLI_PATH` at the stub, `--include-hook-events`; `plugin: {tag, commit}` |
| `internal/bench/profile.go` | add | the profile's parts in one place: `ProfileStage5`, the built-in skill and plugin names, `profiled`, the exclusion settings, `pluginRef`, `resolveProfile`, and the Codex stub (exits 2 saying external models are not part of this stage) |
| `internal/bench/canary.go` | edit | `CheckProfileCanary`: the init line's skills against the built-in set (plus `quality-harness:*` in P), its plugins, the hook events by configuration, the plugin's own text taken out of P before the mrw and marker checks; `RunCanary` runs every configuration given |
| `internal/bench/transcript.go` | edit | read the init line's skills and plugins, and hook events with their source |
| `internal/bench/runner_test.go` | edit | P's invocation; the profile with P absent from a block |
| `internal/bench/redbase_test.go` | edit | `toolPath` takes the node to link |
| `internal/bench/canary_test.go` | edit | the canary per configuration |
| `cmd/bench/main.go` | edit | `--profile`, `--plugin-dir`, `--node`, `P` in `--configs`, P in the canary when `--plugin-dir` is given; the usage text (what selects them) |
| `docs/bench/README.md` | edit | configuration P and the profile |

## Ordered Steps

1. [S1] Write the four failing tests and see them red. Then one uncounted canary probe, stage `probe`, Sonnet, P with the plugin at its latest tag: record its init line's skills and plugins, its hook events, and whether any account-synced skill or plugin from the config directory loaded; find the setting or layout that keeps those out of every configuration, or stop back to Zy. [proof: human: the probe's init line, hook events, and the synced entries' fate]
2. [S2] `Configs["P"]` is A's tools. The profile, set by `--profile stage5` and written in `run.json`, applies to every configuration a block runs: skills on, `Skill` offered, `--node` on PATH, `CODEX_CLI_PATH` at the stub and no `codex` on PATH, `--include-hook-events`. P adds `--plugin-dir DIR`; the plugin's tag and commit (`git -C DIR describe --tags --exact-match`, `rev-parse HEAD`; a dirty or untagged worktree refused) go into `run.json` and P's receipts. [proof: mutation]
3. [S3] `CheckCanary` takes the configuration and the built-in skill set S1 recorded.
   - A's and B's init skills equal that set, and their init line lists no plugin.
   - P's init skills are that set plus `quality-harness:` skills and nothing else, and its init line lists only that plugin.
   - P shows a hook event whose source is the plugin.
   - In P, `quality-harness:` is allowed in the validated init skills, in `Skill` calls to them, and in the plugin's hook events; "SessionStart hook" and "UserPromptSubmit hook" only in the plugin's hook events. Any other plugin or hook source is refused in every configuration.
   - The mrw check reads the agent's text and tool results.

   [proof: mutation]
4. [S4] Wire the flags and `P` in `cmd/bench/main.go` and its usage text; `docs/bench/README.md` names P, the profile and its limitation (no `Agent`, no external model). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestConfigurationPIsAPlusThePluginAndSkills|TestTheProfileOutlivesP|TestTheCanaryRefusesAPluginThatIsNotEngaged|TestAAndBCarryNoPluginInAStageWithP)$' 2>&1) \
  && for t in TestConfigurationPIsAPlusThePluginAndSkills TestTheProfileOutlivesP TestTheCanaryRefusesAPluginThatIsNotEngaged TestAAndBCarryNoPluginInAStageWithP; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q -- '--plugin-dir'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestConfigurationPIsAPlusThePluginAndSkills` | `internal/bench/runner_test.go` | P's arguments are A's plus `--plugin-dir`, with skills, `Skill`, hook events and the refusing Codex stub; `run.json` and P's receipt carry `{tag, commit}`, A's none; an untagged or dirty plugin, P without the profile and P without `--plugin-dir` are refused | none | S2 |
| `TestTheProfileOutlivesP` | `internal/bench/runner_test.go` | a profiled block running only A and B keeps skills, `Skill`, `node`, the stub and the exclusion settings, with no `codex` on PATH; without the profile A's arguments are today's | none | S2 |
| `TestTheCanaryRefusesAPluginThatIsNotEngaged` | `internal/bench/canary_test.go` | a clean P passes with the plugin's own text naming mrw and the markers; no plugin skill, no session hook, a synced plugin or skill, the marker or mrw in the agent's text, or another tool list fails | none | S3 |
| `TestAAndBCarryNoPluginInAStageWithP` | `internal/bench/runner_test.go` | A's and B's arguments hold no `--plugin-dir`; clean A and B pass; a plugin skill, the plugin, a hook, the marker in A, or the plugin's hook, a synced plugin or skill in B fails | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--profile` and `--configs …,P` in `cmd/bench/main.go`; the fence runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's pilot and T5's stage |

## Mutation Log
(empty until execute)
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/runner.go` · the profile reaches only P · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the profile is the stage's, not the arms'
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/runner.go` · P runs without the plugin · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:P is A plus the plugin
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/profile.go` · a plugin with uncommitted changes is accepted · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:P is A plus the plugin
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/profile.go` · synced skills load · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the profile is the stage's, not the arms'
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/canary.go` · a synced plugin passes the canary · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the canary proves engagement and nothing synced
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/canary.go` · P passes with no plugin hook fired · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the canary proves engagement and nothing synced
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/runner.go` · the Codex path reaches a program that succeeds · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:no configuration reaches an external model
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/canary.go` · the agent's own text passes as an injected skill · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the canary proves engagement and nothing synced
- 2026-10-17 · 1fc00bad* · mutant survived · exit 0 · `internal/bench/runner.go` · a profiled stage starts on an A/B canary · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the canary proves engagement and nothing synced
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/canary.go` · a same-named synced plugin passes · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the canary proves engagement and nothing synced
- 2026-10-17 · 1fc00bad* · mutant killed · exit 1 · `internal/bench/runner.go` · a profiled stage starts on a canary run without the profile · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · covers:the canary proves engagement and nothing synced

## Invariants

- Without `--profile stage5`, A and B get exactly today's arguments, so Stages 1 to 4b stay reproducible.
- The plugin is read from a worktree and never written.

## Risks

- The headless init line may not attribute skills to a plugin; S1's probe settles the shape before S3's rule is written.
- Synced account entries may have no off switch; then S1 stops and Zy decides.

## Stop Condition

The fence exits 0 and S1's probe is recorded; or no layout keeps the synced entries out, or no headless transcript shows a plugin engaged, either of which goes back to Zy.

## Out of Scope

- Offering `Agent`, or allowing an external model, in any configuration (deferred: docs/adr/BACKLOG.md — "From ADR-162")

## Verification Log
(empty until execute)
- 2026-10-17 · 1fc00bad* · exit 1 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:286 · test-lock-sha256:cd8057b39ddf6bef072ca736bebe59e86c2e0fc39cbeb9a765142d5682fe0e2e · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFBbmRCQ2FycnlOb1BsdWdpbkluQVN0YWdlV2l0aFAJOWU2NTUwMzkxNzNiNGZlNzM4OGZjMDc0ZTU3ZjMzODdlZTMwMDQ2NzY2ZTkxMzE4NDkzMTg1MzMxYzU0MjJlOQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBRmFpbGluZ0V4aXRJc0FuRXJyb3JBbmRBbkludGVycnVwdFN0b3BzVGhlU3RhZ2UJMTA0NTZiMzFlZWE2MzY3OWRhMmU0M2FmZjg3ZjRmYzMyOTc4MzRkNjA4NDIwZWE2Njg4Njc1M2FkZjgyYzlkZQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBU3RhZ2VXaXRoQU1pc3NpbmdEZWNsYXJlZFNlY3Rpb25Eb2VzTm90U3RhcnQJNTBhY2IxYjcxY2JkNjA1MGJhNjdjMDcxNGYzZmM2NGQxNzFmYmY4M2Y3NjhjNjRiYTZiNTJkZDE0N2RjNTY0Ngpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBU3VwZXJzZWRlZFRhc2tJc05vdFJ1bkJ5RGVmYXVsdAkzZjA4NDU3N2FhZmFkOTAxZGZiMTEzZGIwZDU0MGY2ZDlkZjQwOWJjYjM1ODhmODQzZDk5MjEyZWJkNzBhMzQzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFuQXR0ZW1wdExlYXZlc05vdGhpbmdSdW5uaW5nCTM4ZDhlNWUyZTk2NTE5Y2UyMTA0ZDkzZDRmMGM1OTExMmZkNDg0ZGQ1ZjQ3NjZkNWRiYjM3OTdhYzlkODY5NmUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QW5BdHRlbXB0UmVhZHNOb25lT2ZUaGVIb3N0c1NoZWxsT3JHaXRTZXR1cAlmYWFlNGE5NWIwYzg4YjQ4NWViMTNjNGZjZDRmZjUyY2ZlMTFlYTE4YjQyOTdjZWMxZDI3ZjBjNDRlNGRhNmNiCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFuRXhjbHVkZWRBdHRlbXB0SXNSZXBsYWNlZEFuZEtlcHQJN2YxOTI2OWIyZTQ4N2Y2MmM2NTJmYzU5ODFjNjgzOTNjNWZiZGEzN2ZmMWQ5M2FmMTVkYTlmOTkxOWQyZTYyNgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RCb3RoQ29uZmlndXJhdGlvbnNLZWVwVGhlU3lzdGVtUHJvbXB0U3RhYmxlCTM4ZjIzYzM2OWEwYjQ5OWVlZGEzNWRmNTdlNDgyM2QyYzhlMjdmZWZlNDg2NDFkM2JlMDUzOGU5YzAzZjc2ZjYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0Q29uZmlndXJhdGlvbkFIYXNOb01yd09uSXRzUGF0aAlmMDc0Yjk1MTQ2YTc5YjhiODcwYTQwYzc2NDQ0YzI0ZDI0ODA3N2RkODYyYmZkYzEzZTU1OGE2YWVhYTBhYThmCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25QSXNBUGx1c1RoZVBsdWdpbkFuZFNraWxscwk1ZmViOTA4MDYxZjczZjExMjBkZWZhODgyODkxN2RmZjE4MDM4YmZlOTZjYjg4YTUwZDY0ZDAzYjM4OThlMjMzCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZUNhbmFyeVJ1bnNCb3RoQ29uZmlndXJhdGlvbnNBbmRSZWNvcmRzSXRzVmVyZGljdAk1ZjYwYTk5MDVhNjkzZTRlM2ViMzRhZmRlZTg5MmU4OTQ0MWRlODZlMTRjMDFkZTlhODVkNDg2NmRiMzgzYjg1CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZUNvbmZpZ0RpcmVjdG9yeUlzSnVkZ2VkQnlJZGVudGl0eQkxZTllZmM3ZDYxYTFiNmNkMDI2MmRiMWM4YTAxODYzZWM1NTAyZTUwOWMzYWQ3NDI5MTk2NjJhNzQ3YzZiYjBkCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVByb2ZpbGVPdXRsaXZlc1AJZDc5MDM5MDQ4YWNlNWE0NDBmMDUzMjQ2MWEwOTYyM2I0MDk2NWU4YTA3NmY0MGExNjdhMTA1YTBmOTZjZWRhMApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVSdW5uZXJSZWNvcmRzQVJlY2VpcHRQZXJBdHRlbXB0RnJvbUFTdHViQ2xhdWRlCTA1MDhiMzM3NmM3NmI4MGRjZDI3NGI1YjUzOWI3ZWFjNGM0NGM0NjQyNmI2MGRjOTFmZDQ5ZWJlYzUzZWY5MDkKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVmdXNlc1RvT3ZlcndyaXRlQW5FYXJsaWVyUnVuc1JlY2VpcHRzCThkYzQ5YTc2ZWMwZDQ1MjYzNDliMDE0OGQxZTYzNTQ0YTYzNzU1NTIwZWI3NTgzZTljNzExZjIzOWMzNzQ0ODcKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVmdXNlc1dpdGhvdXRUaGVHYXRlc1p5SG9sZHMJNWFhMTRkMjZlZTlhM2FiYzMzODY5YzQ0Y2Y5MGRiMDc4YTZlNjU5YjcwM2E2N2M0MTIwODQwOTIxZDU2OTNmOApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVSdW5uZXJSZXN1bWVzV2l0aG91dFJlcGVhdGluZ0FTdGFydGVkQXR0ZW1wdAliMjAwZGM0ZTk4OTM5OGFlNWY3ZmVkYzBiZmJhNjQxNmFjNGM4NDliOWEzZDY4MmQ4NDlmZGIwNDBhMzI3YTI0CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFZlcmlmeU5hbWVzQW5BdHRlbXB0V2l0aE5vUmVjZWlwdAkzNDIwNmRkYjQ2NjAzYzM2MGIwYTkwMzVjMDg4MmIxOTAzNmZlN2M5OTliYzZhNGZlMWFlZmIwMmNhYmRiZGQyCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFZlcmlmeVJlZnVzZXNBUmVjZWlwdFRoYXRJc05vdEl0c0F0dGVtcHQJZmYyOGFiY2I1Yzk2Nzc0NmRhYWZiN2Q3M2U5ZmIwMjJhYjg0ZTk5NThjOWQ4M2YxYjBmMTgyZGZlNzNmMmE4NQ
  ```
  ```
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2317
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2445
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2354
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2293
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2473
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2317
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2342
- 2026-10-17 · human-observed · S1 observed 2026-10-17: uncounted canary probes, stage probe, Sonnet 5.5, CLI 2.1.302 (~/.local/share/claude/versions/2.1.302), config dir ~/.local/share/mrw-bench/claude-config, plugin quality-harness cloned at tag v3.8.39 (79ac6651). With skills on and no exclusion, every configuration loaded the synced plugins figma@synced and cowork-plugin-management@synced, the synced skills anthropic-skills:* and about 30 bundled skills. Settings syncClaudeAiSkills false, disableBundledSkills true and enabledPlugins false for the two synced and the three @builtin plugins left A with skills [design, doctor], no plugin and no hook event, and P with those plus 17 quality-harness:* skills, plugins [quality-harness@inline], and hooks SessionStart x2, UserPromptSubmit, Stop (PreToolUse:Skill on a Skill call). plugins/ in the config dir was unchanged (no trash move). A Skill call shows as tool_use Skill {skill: quality-harness:operating}, a tool_result 'Launching skill: …' and a user text 'Base directory for this skill: <plugin>/skills/<name>'. node must be a real binary (/opt/homebrew/bin/node); nvm's is a shell function
- 2026-10-17 · human-observed · S1 approved 2026-10-17: the probe recorded in the line above is the shape S3's canary rule was written against (built-in skills design and doctor; P lists the plugin and its skills and fires its session hooks; the synced and built-in plugins are switched off by settings); observed, and the rule follows it
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2458
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2618
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2772
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:1896
- 2026-10-17 · 1fc00bad* · exit 0 · `set -o pipefail …` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:2528
- 2026-10-17 · human-observed · relock 2026-10-17 reviewed: TestConfigurationPIsAPlusThePluginAndSkills and TestTheProfileOutlivesP gained assertions from the Codex review of T1, before the commit (a profiled stage refuses a canary run without the profile, without P, or with another plugin; a relative --node reaches the attempt's PATH as an absolute link); no assertion was removed or weakened; approved
- 2026-10-17 · 1fc00bad* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:eb0de459f1acfaf89f9d9f2c71f70634a1da5e3d0db5ce846fcff320d9f42f80 · ms:0 · test-lock-sha256:972022ac255c9d1da7c2e7cd9d75ab4afa702bd19fe87960475ff21cbaa47245 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNhbmFyeUZhaWxzV2hlblRoZVRyYW5zY3JpcHROYW1lc01yd09yVXNlckluc3RydWN0aW9ucwkwN2EzOTVkNDI3ODFjMmY0YjQ5YTY5ZTJiMGJkNTdlNjhkMzRlZDdmNzMxMTYxYzZjMTEwYWUxMzY2ZTA3NzM2CmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNhbmFyeVJlZnVzZXNBUGx1Z2luVGhhdElzTm90RW5nYWdlZAk1Y2NkZWFjODc3YTZhODdkMWZmZDg5OWE4M2Y3NzM2ZWIxNGNjYzM2OTM5NTRiMjE3NmM0MjRlZThkZDU3NjViCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNvbmZpZ0RpcmVjdG9yeUlzTm90QUxlYWsJMWRiNDdkNGY1ZWEzYTI5ZTA3NDVlZDBmNzUyMzIzZGQ4NzhkZmE4NjMwNDA0YjRjMGIxMjU4MTI2YjZkNDYzZApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBQW5kQkNhcnJ5Tm9QbHVnaW5JbkFTdGFnZVdpdGhQCTllNjU1MDM5MTczYjRmZTczODhmYzA3NGU1N2YzMzg3ZWUzMDA0Njc2NmU5MTMxODQ5MzE4NTMzMWM1NDIyZTkKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QUZhaWxpbmdFeGl0SXNBbkVycm9yQW5kQW5JbnRlcnJ1cHRTdG9wc1RoZVN0YWdlCTEwNDU2YjMxZWVhNjM2NzlkYTJlNDNhZmY4N2Y0ZmMzMjk3ODM0ZDYwODQyMGVhNjY4ODY3NTNhZGY4MmM5ZGUKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QVN0YWdlV2l0aEFNaXNzaW5nRGVjbGFyZWRTZWN0aW9uRG9lc05vdFN0YXJ0CTUwYWNiMWI3MWNiZDYwNTBiYTY3YzA3MTRmM2ZjNjRkMTcxZmJmODNmNzY4YzY0YmE2YjUyZGQxNDdkYzU2NDYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0QVN1cGVyc2VkZWRUYXNrSXNOb3RSdW5CeURlZmF1bHQJM2YwODQ1NzdhYWZhZDkwMWRmYjExM2RiMGQ1NDBmNmQ5ZGY0MDliY2IzNTg4Zjg0M2Q5OTIxMmViZDcwYTM0Mwpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkF0dGVtcHRMZWF2ZXNOb3RoaW5nUnVubmluZwkzOGQ4ZTVlMmU5NjUxOWNlMjEwNGQ5M2Q0ZjBjNTkxMTJmZDQ4NGRkNWY0NzY2ZDVkYmIzNzk3YWM5ZDg2OTZlCmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdEFuQXR0ZW1wdFJlYWRzTm9uZU9mVGhlSG9zdHNTaGVsbE9yR2l0U2V0dXAJZmFhZTRhOTViMGM4OGI0ODVlYjEzYzRmY2Q0ZmY1MmNmZTExZWExOGI0Mjk3Y2VjMWQyN2YwYzQ0ZTRkYTZjYgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RBbkV4Y2x1ZGVkQXR0ZW1wdElzUmVwbGFjZWRBbmRLZXB0CTdmMTkyNjliMmU0ODdmNjJjNjUyZmM1OTgxYzY4MzkzYzVmYmRhMzdmZjFkOTNhZjE1ZGE5Zjk5MTlkMmU2MjYKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0Qm90aENvbmZpZ3VyYXRpb25zS2VlcFRoZVN5c3RlbVByb21wdFN0YWJsZQkzOGYyM2MzNjlhMGI0OTllZWRhMzVkZjU3ZTQ4MjNkMmM4ZTI3ZmVmZTQ4NjQxZDNiZTA1MzhlOWMwM2Y3NmY2CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdENvbmZpZ3VyYXRpb25BSGFzTm9NcndPbkl0c1BhdGgJZjA3NGI5NTE0NmE3OWI4Yjg3MGE0MGM3NjQ0NGMyNGQyNDgwNzdkZDg2MmJmZGMxM2U1NThhNmFlYWEwYWE4Zgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RDb25maWd1cmF0aW9uUElzQVBsdXNUaGVQbHVnaW5BbmRTa2lsbHMJYmE1N2Y3ZWEyNDYxZDNlNWM1MWNlODRhZTE0MTg3NDBmOWIzMjE2NWM2NmQ3YTY4NWVmNWVmOTA2NjFjZjNmMgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVDYW5hcnlSdW5zQm90aENvbmZpZ3VyYXRpb25zQW5kUmVjb3Jkc0l0c1ZlcmRpY3QJNWY2MGE5OTA1YTY5M2U0ZTNlYjM0YWZkZWU4OTJlODk0NDFkZTg2ZTE0YzAxZGU5YTg1ZDQ4NjZkYjM4M2I4NQpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVDb25maWdEaXJlY3RvcnlJc0p1ZGdlZEJ5SWRlbnRpdHkJMWU5ZWZjN2Q2MWExYjZjZDAyNjJkYjFjOGEwMTg2M2VjNTUwMmU1MDljM2FkNzQyOTE5NjYyYTc0N2M2YmIwZApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RUaGVQcm9maWxlT3V0bGl2ZXNQCWI4YWYwMzNkMzVmYzZhNjdlYzNlMmFmZjk4ZTU3NGVhYmZkN2VmN2ViYjBmZGQzYjJmNjdmYjAyMGJjNWIxY2EKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVjb3Jkc0FSZWNlaXB0UGVyQXR0ZW1wdEZyb21BU3R1YkNsYXVkZQkwNTA4YjMzNzZjNzZiODBkY2QyNzRiNWI1MzliN2VhYzRjNDRjNDY0MjZiNjBkYzkxZmQ0OWViZWM1M2VmOTA5CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNUb092ZXJ3cml0ZUFuRWFybGllclJ1bnNSZWNlaXB0cwk4ZGM0OWE3NmVjMGQ0NTI2MzQ5YjAxNDhkMWU2MzU0NGE2Mzc1NTUyMGViNzU4M2U5YzcxMWYyMzljMzc0NDg3CmJvZHkJaW50ZXJuYWwvYmVuY2gvcnVubmVyX3Rlc3QuZ28JVGVzdFRoZVJ1bm5lclJlZnVzZXNXaXRob3V0VGhlR2F0ZXNaeUhvbGRzCTVhYTE0ZDI2ZWU5YTNhYmMzMzg2OWM0NGNmOTBkYjA3OGE2ZTY1OWI3MDNhNjdjNDEyMDg0MDkyMWQ1NjkzZjgKYm9keQlpbnRlcm5hbC9iZW5jaC9ydW5uZXJfdGVzdC5nbwlUZXN0VGhlUnVubmVyUmVzdW1lc1dpdGhvdXRSZXBlYXRpbmdBU3RhcnRlZEF0dGVtcHQJYjIwMGRjNGU5ODkzOThhZTVmN2ZlZGMwYmZiYTY0MTZhYzRjODQ5YjlhM2Q2ODJkODQ5ZmRiMDQwYTMyN2EyNApib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlOYW1lc0FuQXR0ZW1wdFdpdGhOb1JlY2VpcHQJMzQyMDZkZGI0NjYwM2MzNjBiMGE5MDM1YzA4ODJiMTkwMzZmZTdjOTk5YmM2YTRmZTFhZWZiMDJjYWJkYmRkMgpib2R5CWludGVybmFsL2JlbmNoL3J1bm5lcl90ZXN0LmdvCVRlc3RWZXJpZnlSZWZ1c2VzQVJlY2VpcHRUaGF0SXNOb3RJdHNBdHRlbXB0CWZmMjhhYmNiNWM5Njc3NDZkYWFmYjdkNzNlOWZiMDIyYWI4NGU5OTU4YzlkODNmMWIwZjE4MmRmZTczZjJhODU · test-lock-kind:replace
- 2026-10-17 · human-observed · approved 2026-10-17: the relock above is reviewed; the two tests only gained assertions (profiled canary matching, an absolute node link), and the fence passes on the relocked tests
