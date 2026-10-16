# Task ADR-161-T4: AGENTS.md and the mrw skill lead with the same recipes

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S (prose and a skill upload)
**Owner:** Zy
**Produces:** the recipes in `AGENTS.md`, `README.md`, `BESTPRACTICES.md`, `.claude/skills/mrw/SKILL.md` and the four agent definitions
**Consumes:** the `Recipes:` block (T3)
**Data dependency:** the centralised skill stores (agentsmemory, memoryd), at release
**Proof map:** v1
**Rests-on:** `AGENTS.md carries recipe 1`

## Goal

Every place an agent learns mrw opens with the recipes the binary prints.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `AGENTS.md` | edit | "Using mrw" opens with the recipes, recipe 1 verbatim |
| `README.md`, `BESTPRACTICES.md` | edit | point at the recipes and carry recipe 1 |
| `.claude/skills/mrw/SKILL.md` | edit | its description teaches recipe 1, not only one read and one plan |
| `.claude/agents/adr-challenger.md`, `go-implementer.md`, `go-reviewer.md`, `scout.md` | edit | the mrw block carries recipe 1 |

## Ordered Steps

1. [S1] See the fence red: AGENTS.md lacks recipe 1. [proof: acceptance]
2. [S2] "Using mrw" opens with the recipes, each command verbatim from `mrw instructions`; the other surfaces of Decision 7 carry recipe 1; the opencode plugin is checked to inherit them. [proof: acceptance]
3. [S3] At the release, the centralised `mrw` skill is updated in memoryd's `craft` and mirrored to agentsmemory, both compared byte for byte with AGENTS.md's section. [proof: human: the executor records both skill versions and the comparison]

## Acceptance

```bash
set -o pipefail
r="mrw write --grep '\bOld\b' --sub 'New' ."
i=$(go run ./cmd/mrw instructions) && grep -qF -- "$r" <<<"$i" \
  && for c in $(grep -n '^  mrw \|^  cmd ' <<<"$i" | cut -d: -f1); do l=$(sed -n "${c}p" <<<"$i" | sed 's/^  //'); grep -qF -- "$l" AGENTS.md || exit 1; done \
  && for f in README.md BESTPRACTICES.md .claude/skills/mrw/SKILL.md .claude/agents/adr-challenger.md .claude/agents/go-implementer.md .claude/agents/go-reviewer.md .claude/agents/scout.md; do grep -qF -- "$r" "$f" || exit 1; done
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence holds AGENTS.md to the binary's recipe 1 | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the AGENTS.md section |
| 2 — something selects it | agents read AGENTS.md first |
| 3 — the caller can discover it | the top of "Using mrw" |
| 4 — it is used | sessions here; the skill elsewhere |

## Mutation Log
(empty until execute)
- 2026-10-16 · 9db0e7ba* · mutant killed · exit 1 · `AGENTS.md` · S2: AGENTS.md no longer carries the binary's recipe 1 verbatim · acceptance-sha256:2f4c4623c2e7e67f8cf6665215169712857dcfe48d201b06ff4193cabd50138e

## Invariants

- AGENTS.md's existing rules stay; the recipes are added above them.

## Risks

- The skill copy drifts; the byte comparison at release catches it.

## Stop Condition

The binary's recipe and AGENTS.md cannot carry the same command line.

## Out of Scope

- Other repositories' instructions (permanent: boundary: they read the skill)

## Verification Log
(empty until execute)
- 2026-10-16 · 9db0e7ba · exit 1 · `set -o pipefail …` · acceptance-sha256:2f4c4623c2e7e67f8cf6665215169712857dcfe48d201b06ff4193cabd50138e · ms:1273 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · 9db0e7ba* · exit 1 · `set -o pipefail …` · acceptance-sha256:2f4c4623c2e7e67f8cf6665215169712857dcfe48d201b06ff4193cabd50138e · ms:297
  ```
  ```
- 2026-10-16 · 9db0e7ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f4c4623c2e7e67f8cf6665215169712857dcfe48d201b06ff4193cabd50138e · ms:202
- 2026-10-16 · 9db0e7ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f4c4623c2e7e67f8cf6665215169712857dcfe48d201b06ff4193cabd50138e · ms:978
- 2026-10-17 · human-observed · S3 observed by the executor, 2026-10-17, at the v1.68.0 release (tag 975c3ba): the centralised mrw skill is version 60 in memoryd's craft space and version 73 in agentsmemory, the two bodies byte-identical (cmp), and each body's 'Using mrw' section is byte-identical to that section of AGENTS.md at v1.68.0, compared programmatically after both stores were read back; the provenance pin names v1.68.0 and a 'What v1.68.0 changed' section leads. agentsmemory takes a description of at most 1024 characters, so its description is the shortened form; memoryd keeps the longer one.
