# Comparison

This is a dated comparison of six file-edit guarantees. It records the scan below;
it is not a current product ranking or a measurement of task cost or speed.

Scan dated **2026-09-12**, mrw **v1.13.0** (`d7e39bd`).
The mrw side was re-read against **v1.21.0** (`fb40ec1`); at that point it had gained
`--format=apply_patch` / `search_replace` ([ADR-051](adr/ADR-051-foreign-plan-grammars-compile-to-plan-hunks.md)),
`--ast-grep` on read ([ADR-058](adr/ADR-058-structural-find-shells-out-to-ast-grep.md)),
and a default project check on non-prose writes
([ADR-054](adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree.md)).
A `{files}`- or `{dirs}`-only `scoped_check` still runs when `packages()` cannot map
([ADR-061](adr/ADR-061-files-scope-does-not-need-packages.md)); `{packages}`-only
still falls back. None of those add a seventh guarantee. The grid below is the
scan's.
The Claude Edit entries were re-checked on **2026-11-18** against Claude Code 2.1.326
and its [tools reference](https://code.claude.com/docs/en/tools-reference.md), and a row
for shell edits was added then. Nothing else in the grid was re-scanned.

Interface-call and byte measurements live in [measure.md](measure.md); agent turns, cost and
task outcomes live in [bench/RESULTS.md](bench/RESULTS.md). For current mrw behavior, use
the [README](../README.md) and `mrw instructions`.

## The six

yes = ships it · half = file-level, host-decided, or rejects-only · — = absent.
Source: vendor docs and git-apply(1), 2026-09-12. Not a live bake-off.

| Surface | Atomic plan | Per-line license | Unread refuse | Page / fitting ack | One-call N files | skip, not ok |
|---|---|---|---|---|---|---|
| **mrw** | yes | yes | yes | yes | yes | yes |
| git apply | yes | — | — | — | yes | half |
| Codex apply_patch | half | — | — | — | yes | half |
| Claude / Cursor Edit | — | half | half ¹ | — | — | — |
| Aider | — | — | — | — | yes | — |
| Continue MultiEdit | half | — | — | — | — | — |
| MCP fs / Desktop Commander | — | — | — | — | — | — |
| Shell edits (`sed -i`, a heredoc, a script) | — | — | — | — | yes | — |

¹ Claude Code only, and by model since the 2026-09-12 scan. The tools reference says Opus 4.6,
Haiku 4.5 and older models must read a file before editing it; that newer models may edit an
unread file when reading it would need no permission prompt and the Read tool is available; and that a
shell command it recognises (`cat`, `grep`, `head` and the like) on one file, with no pipe or
redirect, counts as the read when it printed the file's text.

## Closest four

1. **git apply** — closest on the contract. Default abort restores the tree.
   `--reject` is the leak. One patch, many files. Already on PATH. Missing
   license, unread refuse, ack, skip-not-ok. `--3way` can leave conflict
   markers.
2. **Codex apply_patch** — closest on the job. One multi-file document, no line
   numbers, fuzzy, trained into the models. OpenAI leaves atomicity to the
   harness. Codex-rs applies sequentially; a failed write can already have
   mutated the target (`delta.exact=false`). `--format=apply_patch` compiles
   that grammar into a native plan so we do not copy the sequential leak.
3. **Claude Edit / Write** (Cursor `StrReplace` is the same shape) — closest on
   license instinct. Unique `old_string` or refuse, unless `replace_all`. Per-file atomic. Not a plan.
   MultiEdit was removed in Claude Code 2.0. Its read rule is file-level and now
   depends on the model (note 1). Probed 2026-11-18 in Claude Code 2.1.326, auto
   mode, a Claude 5 model, on scratch files: an Edit of a file never opened with
   Read applied; a Write over an existing unread file applied; and after the shell
   inserted a line above the target, the Edit still applied, with a note that the
   file had changed on disk, where the 2026-09-12 scan recorded a stale-mtime
   refusal. The reference documents that last behaviour: the edit can apply when
   `old_string` still matches exactly once and the file could be read without a
   prompt. Addressing by string is why an inserted line does not move its target; a
   line number does move. By default mrw refuses a hunk to a file whose content is
   not what it recorded (`--force` and a quoted SEARCH/REPLACE hunk aside), and a
   multi-line `replace` must carry `anchor=` (ADR-035). After a write of its own that
   records the file as wholly licensed, a hunk that names a line by number alone is
   refused on a line no read has shown since (ADR-203); `anchor=`, a pattern address or a read that shows the line lands it.
4. **Shell edits** — not in the 2026-09-12 scan, and what an agent with only a
   shell tool uses. `sed -i` over many files is one call. Nothing is licensed,
   nothing is refused for being unread, and a `sed -i` substitution that matches
   nothing exits 0 having changed no text: the write that changes nothing, unseen. A
   `cat > file <<'EOF'` replaces the whole file. Claude Code's
   [checkpoints](https://code.claude.com/docs/en/checkpointing.md) do not track a
   file only shell commands changed, which is true of an mrw write as well.

## They win. Do not copy them.

| They win at | Why | What we do |
|---|---|---|
| Speed and streaming | Morph / Relace / Cursor Instant Apply merge at ~10k tok/s | **Wait.** [ADR-049](adr/ADR-049-streaming-apply-waits-for-a-size-that-hurts.md): stream when a real file hurts. Measured 14 MB / 200k lines in 0.15 s. The product is turns, not tok/s. |
| Language-aware **write** | Serena / Comby rewrite syntax | **Don't.** [ADR-048](adr/ADR-048-mrw-models-no-target-syntax.md): no target-syntax parser. `--ast-grep` finds on read. Wrap-tails: `--echo-pad`, `--strict-balance`, default `--check`. |
| Fuzzy, no line numbers | apply_patch / Aider / Desktop Commander fallback | **Don't.** Steal the grammar ([ADR-051](adr/ADR-051-foreign-plan-grammars-compile-to-plan-hunks.md)), keep exact apply. |
| IDE UX and adoption | Cursor / Claude / Cline sit where the human already works | **Not this binary.** Teach (`mrw instructions`, the skill). Do not grow an IDE. |
| git already on PATH | every machine has it | Same as row 1. We are not a `patch(1)`. A git patch is not an apply_patch. |
| One-file one-edit | `StrReplace` is shorter on bytes | **Use mrw anyway.** Agents batch one read + one write or they never come back. The 3+ threshold trained them out ([ADR-062](adr/ADR-062-instructions-teach-always-and-a-plan.md)). |

Named, not healed: PATH/skill skew
([ADR-041](adr/ADR-041-path-binary-and-skill-version-skew-is-named-not-healed.md)),
two MCP tools ([ADR-044](adr/ADR-044-mcp-cargo-stays-two-tools.md)), Windows
state stays XDG ([ADR-050](adr/ADR-050-windows-state-stays-xdg-until-windows-is-exercised.md)),
`scripts/contract.sh` is Linux-only, MSYS rewrites regex addresses.

## Measured improvements

Copying Instant Apply, a YAML/Blade checker, or a third MCP tool would make mrw
*worse at the six* and is not in this list. Each row is an experiment: a metric,
a baseline, and a reading that kills it. Each row uses its stated metric; interface-call
counts do not establish model turns, task cost or solving. ADR-049's wall-clock size probe
is separate from the scripted call-and-byte measurements.

| # | Experiment | Metric | Baseline | Goes red if | Cost |
|---|---|---|---|---|---|
| F | **done 2026-09-15.** `measure.sh` shape F: `--grep` vs `rg`+windowed. Calls 225→2. vs windowed **6.3× MORE**; vs rg+windowed **1.5× MORE**. ast-grep skipped (not on PATH). Did not quote a byte win. | calls stayed 2; bytes vs rg+windows | A–D charged search as +1 call, 0 bytes | mrw calls ≠ 2, or a byte win vs the windowed read | cheap: bash, this tree |
| S | **done 2026-09-15.** Second ADR-009 reading: 4/343 = 1.2% `refused_parse`. Criterion held. Not "the format improved". [model-benches.md](model-benches.md) | `refused_parse` / plans | 2026-09-04: 1/68 = 1.5% | rate **> 5%** of the population that produced it | cheap: `mrw stats` |
| C | Re-run the served-size curve at v1.21.0 | planted-line hit rate vs served bytes (ADR-020) | [model-benches.md](model-benches.md) is **v1.15.0 / `31422d8`**. Do not mix `--ast-grep` into the first re-run (confound). | hit rate drops vs the published table | **costly**: model calls. Ask before running. |
| P | `--echo-pad 3` tax on A and D | extra bytes vs pad 0 | Shape E gutter is 13% to 15% with pad 0 (`fe51cdab`, 2026-11-18). Pad default is 0 (ADR-052). | we default pad without publishing this tax | cheap: one `measure.sh` variant |
| K | Size that hurts (ADR-049 gate) | apply of 5 hunks: seconds and RSS | 2026-08-31: 14 MB / 200k lines, 0.15 s. **Not** a `measure.sh` row. | we stream without naming the size that hurt | **costly**: large fixtures. Pre-register the knee before looking. |
| G | apply_patch / search_replace corpus | compile+apply success on documents models already emit | Contract §82/§84: unread sibling writes nothing. No field corpus yet. | a legal Codex/Aider document fails to compile, or compiles then a sequential apply would have mutated | cheap once corpus exists; collecting it is the work |

**F** and **S** ran 2026-09-15. **C** and **K** only with a budget. **P** before any
default-pad discussion. **G** when we have documents, not fixtures we wrote
ourselves.

Parked on purpose (not experiments): MCP cargo (ADR-044), a syntax parser
(ADR-048), auto-healing PATH (ADR-041), git-patch ingest, default
`--strict-balance` (ADR-055/056: campaign prices false positives first).

Far class, same shelf, wrong job: Aider partial-apply is the design (issue
#462); Morph/Relace probabilistic merge; Serena LSP rename; MCP filesystem is
a disk API; Cline checkpoints after the fact; `patch` writes `.rej` and
continues.

Sources: [OpenAI apply_patch](https://developers.openai.com/api/docs/guides/tools-apply-patch),
[git-apply(1)](https://git-scm.com/docs/git-apply),
[Aider edit formats](https://aider.chat/docs/more/edit-formats.html),
Claude Code file-ops, Morph/Relace, Serena / Desktop Commander READMEs,
Continue MultiEdit. mrw: README Status and `AGENTS.md` rules 1–6 at the scan,
plus the ADRs named above at v1.21.0.
