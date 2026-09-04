# Git: every change goes through a pull request

**Never commit while `main` is checked out and never push to `main`.** M, 2026-09-04: *"we always
have to open a PR, not the push to main."* Branch protection enforces it — a PR required, force-push
and deletion refused, `enforce_admins` on (checked with `gh api …/branches/main/protection` on
2026-09-04) — so a direct push is refused rather than half-applied.

- Before any commit: `git branch --show-current`. On `main`, branch first.
- After pushing, confirm the remote moved: `git rev-parse --short HEAD origin/<branch>` must agree.
  A push that moved nothing prints success like one that did; the one time it happened here
  (2026-09-04, a commit made on `main` by mistake) only the missing CI run for the SHA gave it away.
- `CONTRIBUTING.md` owns commit shape. Attribution trailers as the session instructs.

## Stacked branches, and the merge mode that breaks them

`main` takes squash merges. Once a parent branch merges, a plain `git rebase main` of its child
replays the parent's commits too — the squashed commit shares no history with them. Either
`git rebase --onto main <parent-head> <child>`, or cherry-pick the child's commits onto a fresh
branch from `main` and open that (#82 → #83 took the second).

And `gh pr merge --delete-branch` on a parent **auto-closes the child PR** whose base was that
branch. Either merge stacks bottom-up without `--delete-branch`, or do not stack.

## After a merge

Pull `main`, delete the merged branch, and re-run `go test ./...` and `./scripts/contract.sh` on the
MERGED result. The PR's green tick was for a different tree.
