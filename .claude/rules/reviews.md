# Reviews: a different lineage, acknowledged at once, merged when clean

**Every review goes through Codex** — `/quality-harness:codex-review` — in addition to whatever
in-process review runs. Standing rule since 2026-09-04. A different lineage with fresh context fails
differently: on this repository it caught a wrong `hunks.status` enum, a `-C` recipe that errored
when followed, and two documentation claims an in-repo review had approved.

⚠ `codex exec` reads stdin even with the prompt as an argument; a backgrounded run hangs on a pipe
that never closes. Redirect `< /dev/null`.

## Acknowledge before fixing

When a review lands on one of our PRs, post ONE short comment first — the head SHA being worked
and what is already verified — before the first fix. Then the substantive reply when the fixes are
pushed and green. Two comments; the first costs nothing and stops the reviewer re-running against a
head that is about to move.

## Reconcile against the source, never by authority

A finding is a lead. Confirm it against the code in front of you before acting, and report back
either way.

**Verify a documentation claim against the IMPLEMENTATION, never against other prose.** `--help`,
README, an ADR and a tag list can all be wrong together. `mrw check` "changes nothing" matched its
help exactly and contradicted `internal/check/check.go`; "every release returns empty
`instructions`" matched an empty `git tag --contains` while the tagged trees said MCP first shipped
in v0.0.19. For a release claim read the tagged tree: `git show <tag>:<path>`.

**Credit a review for what it FOUND.** "Found while addressing the review" is not "found by the
review", and a record that says the second when the first is true is a false citation.

## Merge when clean, without asking

A clean verdict — no blockers AND nothing open from the reviewer's side — on one of our PRs means
merge. Asking after a clean verdict adds a round trip that decides nothing. Findings, even
non-blocking ones, are not clean; someone else's PR is theirs to merge. Then `git.md`, "After a
merge".
