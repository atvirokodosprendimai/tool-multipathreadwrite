---
name: release
description: >-
  Cut an mrw release: gates on merged main, break campaign diffed against the
  previous tag, a strict vX.Y.Z tag on the MERGE commit with the shape the last
  tags use, CI publishes, then ONE Status PR naming the tagged SHA, and reinstall
  the local CLI from the tag. Use when M says "release", "cut", "tag it", or
  after a served-path change merges. CONTRIBUTING §Releasing owns the tag rule
  and the Status-names-the-tagged-SHA rule; this is the drill around them,
  learned cutting v1.15.0–v1.16.1.
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

```sh
go build -o bin/mrw ./cmd/mrw && go test ./...; echo TEST:$?
./scripts/contract.sh > /tmp/c.out 2>&1; echo CONTRACT:$?; tail -1 /tmp/c.out
```

The PR's green tick was for a different tree.

## 2. Break campaign, and DIFF it

```sh
MRW=$PWD/bin/mrw bash scripts/break-campaign.sh | grep -v 'campaign dir' > docs/break/campaign-vX.Y.Z.txt
diff <(grep -o '^\[[^]]*\] exit=[0-9]*' docs/break/campaign-<prev>.txt) \
     <(grep -o '^\[[^]]*\] exit=[0-9]*' docs/break/campaign-vX.Y.Z.txt) && echo NO-EXIT-DIFF
```

47 probes as of v1.16.1. An exit-code diff is a finding to explain in the tag
message or a reason not to tag; "identical to <prev>" is the sentence the Status
line carries. The campaign file is committed in the Status PR (step 5), not
before the tag — the tag is code, the campaign is evidence about it.

## 3. Tag the MERGE commit with the shape the last tags use

```sh
git tag -a vX.Y.Z <merge-sha> -m "vX.Y.Z — <one line: what a caller sees differently>

<ADR-NNN. The failure it exists to prevent, in two or three sentences.>

<What changed on the served path, flag by flag. Exit codes.>

Contract §NN–§MM. Tagged at <merge-sha> (#PR). Do not retag v<prev> or the
README status squash."
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

## 7. Write it down

`am_kg_add(tool-multipathreadwrite, released, vX.Y.Z@<sha>)` and a diary line.
The centralised `mrw` skill mirrors AGENTS.md: if the release changed what
AGENTS.md teaches, `am_update_skill("mrw")` with the provenance pin moved to
this tag — a skill one release behind teaches a plan the binary now rejects
(v8 was two behind).

## What went wrong before, so it does not again

- `git pull` on `main` refused to fast-forward (v1.16.0): a never-landed local
  commit. Reset to origin, do not merge.
- `git rev-parse HEAD origin/<branch>` right after a push said "Needed a single
  revision" — the tracking ref was not fetched yet. `git ls-remote` is the check.
- `gh pr merge --delete-branch` on a parent auto-closes a stacked child PR.
  Status PRs are never stacked; cut them from `main` after the tag.
- Reading an exit code through `| head` returns head's status. The v1.16.0
  field verification did exactly this within an hour of writing the rule.
