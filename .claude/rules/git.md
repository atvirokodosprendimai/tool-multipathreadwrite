# Git: every change goes through a pull request

**Never commit while `main` is checked out and never push to `main`.** Branch protection requires a
PR, refuses force-push and deletion, and sets `enforce_admins`, so the owner is bound too. A direct
push is refused rather than half-applied.

- Before any commit: `git branch --show-current`. On `main`, branch first.
- After pushing, confirm the remote moved: `git rev-parse --short HEAD origin/<branch>` must agree.
  A `git push origin <branch> && echo pushed` once reported success on a push that moved nothing,
  and it was caught only because CI listed no run for the SHA.
- One logical change per commit; the message says WHY. Attribution trailers as the session
  instructs.

## Stacked branches, and the merge mode that breaks them

`main` takes squash merges. Once a parent branch merges, its child cannot be rebased onto `main`
— the squashed commit shares no history with the child's. **Cherry-pick the child's commits onto a
fresh branch from `main`** and open that (#82 → #83).

And `gh pr merge --delete-branch` on a parent **auto-closes the child PR** whose base was that
branch. Either merge stacks bottom-up without `--delete-branch`, or do not stack.

## After a merge

Pull `main`, delete the merged branch, and re-run `go test ./...` and `./scripts/contract.sh` on the
MERGED result. The PR's green tick was for a different tree.
