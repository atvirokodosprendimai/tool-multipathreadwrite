# ADR-176: Renovate updates ADR-147's knip pins in the same pull request

**Status:** Proposed
**Date:** 2026-10-25
**Owner:** Zy
**Cross-references:** ADR-147, ADR-088 (fence-prose), ADR-089 (the plugin's npm jobs)
**Governs:** `renovate.json`
**Enforced-by:** None — Renovate runs hosted, outside this tree's tests; the record's fence (T1) pins the manager's shape and the prose shapes it covers, `scripts/fence-prose.py` re-runs those shape guards on every static run, and the next hosted knip PR is the live end-to-end test
**Invalidates:** none
**Served-path change:** none — no file under `internal/` or `cmd/mrw/` changes; what a caller sees differently is Renovate's pull requests, which stop failing CI

## Context

ADR-147 pins knip as the plugin's exact devDependency and repeats the pin in prose — the Context finding (`knip` (6.38.0)), the install sentence (`knip@6.38.0`), the change table (`knip` 6.38.0, exact), and task T2's acceptance fence, whose `grep -q '"knip": "6.38.0"' cmd/opencode/mrw-plugin/package.json` is re-run by fence-prose (ADR-088 T2) on every static run.

Renovate bumps the devDependency — PR #523 (run 40144353890, 2026-10-25, knip 6.38.0 → 6.43.0) — but knows nothing of the prose. Its PR left every ADR-147 mention at 6.38.0, fence-prose failed the fenced grep, and the PR's one failing step was the static-analysis gate. A renovate PR that cannot pass CI is noise: somebody hand-finishes it, or closes it and hand-rolls the bump.

`postUpgradeTasks` commands look like the fix and are not one here: a command runs only when it matches the host's `allowedCommands`, that list belongs to whoever hosts the app — not to `renovate.json` — and a command it does not allow is skipped, leaving the PR open and half-done under a green face.

## Decision

1. **A regex custom manager in `renovate.json` extracts ADR-147's pin mentions as a second `knip` dependency** (`datasourceTemplate: npm`, `depNameTemplate: knip`, `managerFilePatterns: /^docs[/]adr[/]ADR-147/`), through five `matchStrings`:
   - `` `knip@<version>` `` — the install sentence;
   - `--save-exact knip@<version>` — T2's change-table install command;
   - `` `knip` (<version>) `` — the Context finding;
   - `` `knip` <version>, exact `` — the change tables of both files;
   - `grep -q '"knip": "<version>"'` — T2's fenced grep.

   The extractions group with `package.json`'s into one branch and one pull request. No `postUpgradeTasks`, no commands, nothing for a host to allow or skip. Renovate replaces the version inside the matched span and leaves the surrounding prose byte-identical, so T2's fence keeps its shape and matches the new pin.

2. **Three version-shaped sentences stay prose, and the manager must not match them.** `6.38.0 (published 2026-09-23)` is a historical statement about that release; `knip 6.38.0` / `declares 10 of its 13 dependencies` (split across two lines) speaks about that version's manifest; and `` `npx -y knip@6.38.0` `` is a rejected alternative recorded with the version it was rejected at. The five shapes above miss all three by construction; T1's coverage test proves the miss.

3. **The prose stays the source of truth; fence-prose stays the enforcement.** If a future edit rewords a pin into a shape the manager does not match, the next hosted knip PR fails CI exactly as #523 did. The fix is widening `matchStrings` in the change that adds the shape — T1's fence greps the five shipped shapes, so a sixth needs a clause of its own.

## Consequences

- The next hosted knip bump lands with ADR-147 fully bumped — finding, install sentence, change tables, T2's fence — and CI green without hand finishing.
- A pin shape added to ADR-147 prose must add its `matchStrings` pattern (and a T1 fence clause) in the same change.
- The manager is scoped to ADR-147's path; a record that pins another npm tool needs its own manager in the change that pins it.
