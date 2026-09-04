# Reviews: a different lineage, acknowledged at once, merged when clean

**Every review goes through Codex** — `/quality-harness:codex-review` — in addition to whatever
in-process review runs. M, 2026-09-04: *"use codex reviews"*, then *"add this as a standing rule."*
A different lineage with fresh context fails differently: on #72 it found the wrong `hunks.status`
enum independently of the in-repo review that found it too, and on #79 it returned five findings
that all held, among them documentation claims the in-repo review had approved.

⚠ Redirect `< /dev/null` when running `codex exec` from a script: with stdin left on an open pipe, a
backgrounded run has hung here instead of exiting. Defensive, and cheap.

## Acknowledge before fixing

When a review lands on one of our PRs, post ONE short comment first — the head SHA being worked
and what is already verified — before the first fix. Then the substantive reply when the fixes are
pushed and green. Asked for by M on 2026-09-01, so the PR itself shows the bot is working without
anyone having to ask; it also stops the reviewer re-running against a head that is about to move.

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
merge. M, 2026-09-01: *"so if comment nothing from my side, clean - merge it."* Asking after a clean
verdict adds a round trip that decides nothing. Findings, even non-blocking ones, are not clean;
someone else's PR is theirs to merge. Then `git.md`, "After a merge".
