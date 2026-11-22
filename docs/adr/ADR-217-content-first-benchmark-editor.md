# ADR-217: A benchmark editor takes explicit targets and exact old lines

**Status:** Proposed
**Date:** 2026-11-22
**Owner:** Zy
**Spec:** None — benchmark experiment
**Cross-references:** ADR-001, ADR-029, ADR-190, ADR-193, ADR-203, ADR-205, ADR-206, ADR-211
**Governs:** `bench/content-edit/**`, `scripts/contract.sh`
**Enforced-by:** `bench/content-edit/main_test.go::TestContentEditBatch`
**Served-path change:** A separately built benchmark binary accepts one batch on standard input in a disposable checkout.

## Context

Audit of the November 19 design review and November 20 Grok and Codex critique against main at `724d564e`:

| Finding | Current evidence and disposition |
|---|---|
| Position can outlive the observation | ADR-203 refuses some unshown bare numbers. A later read can license the same number at different content. Experiment with an edit carrying its own expected text. |
| Plan document confused with target, first failed match ends compilation, deletion needs another operation | `internal/ingest/searchreplace.go` gives a better diagnostic but still returns on the first failed block. Stage 8 saw 15 of Sonnet's 17 refusals confuse the document with the target. Test a separate input with explicit fields and an empty replacement for deletion. |
| Shared ledger represents checkout knowledge | `internal/seen/checkout.go` shares observations. Arm X obtains its precondition from the submitted old text. |
| Host integrations and interface cost | Native tools provide host-specific checkpoints. v1's task recipes and computed substitutions already exist. Owner instructions govern adoption. A future comparison should include native tools and charge generated plan text and retries. |
| Explicit check semantics | ADR-205 shipped. Both `cmd/mrw/main.go` and `internal/mcp/tools.go` select `writer.CheckDemand` for an explicit check. |
| Process burden and undo | ADR-204 was withdrawn. Historic test locks, prose and the standing tool policy have separate owners and acceptance criteria in BACKLOG. |
| Measurement and review defects | Issues #699–#706 are closed. ADR-211 records the corrected base and kept-file checks, scoped claim, model and spend gates. The free rehearsal exercised stub attempts. A measured default still needs correctly graded agent runs. |

The six runtime findings have executable regression coverage in `scripts/contract.sh`.

| Issue | What the current check exercises | Contract section |
|---|---|---|
| #699 | A file marker without a path rejects its entire patch. | §353 |
| #700 | An empty external delete body cannot silently remove a line. | §371 |
| #701 | Conflicting ancestor and descendant file targets fail before commit. | §354 |
| #702 | A quiet read reports failure when it cannot record its licence. | §355 |
| #703 | A directory target gets a failure verdict alongside its siblings. | §357 |
| #704 | A multiline replacement in a previously written file needs the line after its span to have been shown. | §359 |

## Decision

1. Build `go build -o /tmp/content-edit ./bench/content-edit`. Run that binary with zero arguments from the disposable checkout. Stdin is a nonempty JSON array. Each edit supplies a root-relative `target`, a nonempty array of exact whole `old` lines and an explicit `replacement` array. An empty replacement deletes the matched lines. An empty string in an array represents one blank line. Line terminators are supplied by the file. The benchmark bounds request and individual target sizes.
2. Locate each old block exactly once in its target's pre-edit contents. Report each missing or ambiguous match with its edit number and target. Any such failure prevents the entire batch. Two edits may address the same file in any order if their old spans are disjoint.
3. Hold the checkout's normal write lock, pin each verified file version, and apply all edits through the existing apply engine in one call. Use only in-memory observations of matched spans. The engine retains its overlap, identity, staging, rollback and sync checks. Report a commit or sync error even if files reached disk.
4. Keep X separate from the release binary and the registered A/M/N arms. For a future X comparison, run M again alongside X, grade the complete resulting diff, and charge failed attempts and output tokens. For a Code Mode comparison, use the same model, prompt and permission settings and verify Code Mode is active on each arm. Include a follow-on task whose second target becomes known only after the first write. Deterministic checks establish mechanics; a model trial measures cost.

For example, run the executable in a disposable checkout containing `notes.txt`. This request changes one line and deletes another in the same write.

```sh
printf '%s\n' 'draft' 'obsolete' 'keep' > notes.txt
/tmp/content-edit <<'JSON'
[
  {"target":"notes.txt","old":["draft"],"replacement":["ready"]},
  {"target":"notes.txt","old":["obsolete"],"replacement":[]}
]
JSON
```

The resulting file contains `ready` and `keep` on separate lines. The request limit is 8 MiB and each input file is limited to 16 MiB.
Every target must have consistent line endings and contain no embedded carriage return. To bound preflight on repeated text, the batch has a 64 MiB comparison budget. Each candidate costs the number of bytes in its old lines plus one per line. The search uses the rarest old line to locate candidates. An exhausted budget rejects the batch.

## Alternatives Considered

- Teach the existing SEARCH/REPLACE syntax as the default. Stage 8 recorded higher output tokens per attempt on both models and a NO SHIP verdict for the tested instruction.
- Introduce a production flag. A benchmark executable permits testing the design before adding another permanent interface.

## Component / Boundary Impact

The disposable checkout is the experiment's write boundary. Rooted target reads and the apply engine supply the existing filesystem checks. Use a separate checkout for each arm so one arm's changes cannot affect the next.

## Wiring & Contract Changes

The prototype reads standard input, writes a completion summary on success and reports failures on standard error. It runs no model or project check. The experiment grader checks the final tree independently. Contract §383 drives the built executable.

## Implementation

[Task T1](ADR-217/tasks/T1-content-edit.md) covers the wire format, engine integration and independent tree checks.

## Consequences

A caller can recover every missing or ambiguous target from one refusal. Repeating long old text still costs output tokens. An ambiguous block needs more distinguishing lines.

## Out of Scope

- Paid trials and a production default (deferred: docs/adr/BACKLOG.md)
- A new persistent ledger or host rewind integration (permanent: boundary: isolated benchmark trials)

## Risks

Changed content between lookup and apply invalidates the pinned digest. Repeated old blocks must refuse even if they would receive identical replacement text. A success count alone cannot tell an exact-reference completion from an unintended edit.

## Rollback

Remove the standalone benchmark editor and its contract row. The released mrw binary has no dependency on either.

## Follow-ups

Use the prototype to price the next paired experiment when its tasks, model access and budget are agreed.
