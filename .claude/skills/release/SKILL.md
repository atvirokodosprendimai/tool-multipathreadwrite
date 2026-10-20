---
name: release
description: >-
  Cut an mrw release: gates on merged main, break campaign diffed against the
  previous tag, a strict vX.Y.Z tag on the MERGE commit with the shape the last
  tags use, CI publishes, then ONE Status PR naming the tagged SHA, and reinstall
  the local CLI from the tag. Use when Zy (M in older notes) says "release", "cut", "tag it", or
  after a served-path change merges. CONTRIBUTING §Releasing owns the tag rule
  and the Status-names-the-tagged-SHA rule; this is the drill around them.
---

# release — the drill around CONTRIBUTING §Releasing

CONTRIBUTING says two things and both are load-bearing: push a strict `vX.Y.Z`
tag, and the README **Status** line names the **tagged** SHA, never a later
squash. Everything below is the order that satisfies both without a second tag.

## 0. Preconditions

- `git branch --show-current` is `main`, `git status --short` empty, and `main`
  is at the merge commit you are about to tag (`git pull --ff-only origin main`;
  if it will not fast-forward, local `main` carries a stale commit — `git reset
  --hard origin/main`, the way v1.16.0 had to).
- The change that motivates the tag is a **served-path** change (a flag, a
  receipt field, an exit code, a refusal, a stats line). A docs-only merge is
  not a release.

## 1. Gates on the MERGED tree, stand-alone, never piped

Every gate in `AGENTS.md` §Build and check, each on its own, exit code read directly:

```sh
go build -o bin/mrw ./cmd/mrw && go test -timeout 30m ./...; echo TEST:$?
go test -race -timeout 30m ./...; echo RACE:$?
./scripts/contract.sh > /tmp/c.out 2>&1; echo CONTRACT:$?; tail -1 /tmp/c.out
./scripts/static.sh > /tmp/s.out 2>&1; echo STATIC:$?
(cd cmd/opencode/mrw-plugin && npm ci && npm run build && npm test); echo PLUGIN:$?
```

`static.sh` is the dead-code scan the release needs (deadcode, staticcheck U1000): put its tools,
versions and finding count in the tag message, zero included. `-timeout 30m` because `internal/bench`
alone took 504 s on a loaded machine (2026-10-20), near `go test`'s 10-minute default.
The PR's green tick was for a different tree.

## 2. Break campaign, and DIFF it

```sh
MRW=$PWD/bin/mrw bash scripts/break-campaign.sh > /tmp/campaign.raw; echo CAMPAIGN:$?
grep -v 'campaign dir' /tmp/campaign.raw > docs/break/campaign-vX.Y.Z.txt
diff <(grep -o '^\[[^]]*\] exit=[0-9]*' docs/break/campaign-<prev>.txt) \
     <(grep -o '^\[[^]]*\] exit=[0-9]*' docs/break/campaign-vX.Y.Z.txt) && echo NO-EXIT-DIFF
```

59 probes as of v1.32.0. Read CAMPAIGN's status on its own line: piping the script straight into
`grep` reports grep's status, not the campaign's. An exit-code diff is a finding to explain in the tag
message or a reason not to tag; "identical to <prev>" is the sentence the Status
line carries. The campaign file is committed in the Status PR (step 5), not
before the tag — the tag is code, the campaign is evidence about it.

## 3. Tag the MERGE commit with the shape the last tags use

```sh
git tag -a vX.Y.Z <merge-sha> -m "vX.Y.Z — <one line: what a caller sees differently>

<ADR-NNN. The failure it exists to prevent, in two or three sentences.>

<What changed on the served path, flag by flag. Exit codes.>

Contract §NN–§MM. Dead code: deadcode and staticcheck U1000 (static.sh), N findings.
Tagged at <merge-sha> (#PR). Do not retag v<prev> or the README status squash."
git push origin vX.Y.Z
git ls-remote --tags origin vX.Y.Z | cut -c1-7     # the TAG object's sha, not the commit's
```

Read `git show v<prev> -s --format=%b` first and keep the shape. `-X
main.version=$GITHUB_REF_NAME` in CI stamps the tag; a prerelease suffix builds
nothing (the `check` job re-matches with a regex).

## 4. Let CI publish, and confirm

`gh run list --branch vX.Y.Z` → `build` (5 targets) then `release`. Confirm
`gh release view vX.Y.Z --json assets -q '.assets|length'` is **11** (5 raw, 5
archives, `SHA256SUMS.txt`). Do not push anything else while it runs.

## 5. ONE Status PR, after the tag exists

On a branch: README Status → `**Status: stable at vX.Y.Z (date), the tag cut
from \`<merge-sha>\`.** Break campaign for it: [docs/break/campaign-vX.Y.Z.txt],
N probes, exit codes identical to v<prev>.`; CONTRIBUTING's `(\`v<prev>\` is
\`<sha>\`)` example moves with it; add the campaign file. Squash-merge on
`CLEAN`. This commit is NOT tagged — that is the whole point of the rule.

## 6. Reinstall the local CLI from the tag, then smoke it

```sh
git -c advice.detachedHead=false checkout vX.Y.Z
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=vX.Y.Z" -o /tmp/mrw ./cmd/mrw
install -m 0755 /tmp/mrw ~/.local/bin/mrw && git checkout main && mrw version
```

Same flags CI uses. Then one write that exercises the change through the
installed binary — not `bin/mrw` — and read the receipt.

Between releases, after a merge, install from `main` instead, stamped so `mrw version` names the commit:
`go build -ldflags "-X main.version=v<next>-dev+$(git rev-parse --short=8 HEAD)" -o "$(command -v mrw)" ./cmd/mrw`.
This is the one place that says how the local CLI is installed; `~/.claude/CLAUDE.md` points here.

## 7. Write it down

memoryd `fact`, space `tool-multipathreadwrite`: subject `tool-multipathreadwrite.releases`, predicate
`released`, object `vX.Y.Z@<sha>`; and an `open-thread` naming what is left. If the release changed
what AGENTS.md teaches, update the `mrw` skill in memoryd's `craft` space (`update_skill`, its one
store) with the provenance pin moved to this tag — a skill one release behind teaches a plan the
binary now rejects (v8 was two behind) — then mirror it to `.claude/skills/mrw/SKILL.md`.

## What went wrong before, so it does not again

- `git pull` on `main` refused to fast-forward (v1.16.0): a never-landed local
  commit. Reset to origin, do not merge.
- `git rev-parse --short HEAD origin/<branch>` answered "Needed a single revision", and it was read
  as a tracking ref not fetched yet. It fails every time: `--short` verifies ONE revision (found by
  the review of #282). `git ls-remote` asks the remote itself and is the check.
- `gh pr merge --delete-branch` on a parent auto-closes a stacked child PR.
  Status PRs are never stacked; cut them from `main` after the tag.
- Reading an exit code through `| head` returns head's status. The v1.16.0
  field verification did exactly this within an hour of writing the rule.
