# Reading 13, plan: reading 11's number at 2 KB and 20 KB — is the late-only pattern about position or about preceding text?

**Written and committed before any trial ran.** Readings 10 and 11 stand as collected; nothing
here changes them.

## Why a thirteenth reading

Reading 11 put reading 4's outer number — the served text's row index, `T + 2` on the target's row —
back inside a tool result at 200 KB, and the five misses it produced were the five late cells; the
ten early and middle cells were ten hits. Reading 4's misses on the same cells were spread across
positions. The result named two accounts and tested neither: the miss is about the target's
POSITION in the file, or about how much numbered text PRECEDES the target (about 2,900 rows for a
late target at 200 KB, about 730 for an early one). At 200 KB the two are confounded, because a
late target is also a far one. At 2 KB and 20 KB they come apart: a late target at 20 KB has about
280 rows before it, fewer than an early target at 200 KB, and at 2 KB about 35.

## Cells

Reading 4's fifteen 2,000-byte and fifteen 20,000-byte cells, byte-identical to reading 9's
copies (verified before the first trial), window from line 1. Reading 4's rates on them through
the read arm: 15 of 15 at 2 KB; 12 of 15 at 20 KB with misses at `20000-late-1`, `20000-middle-1`,
`20000-middle-5`. Reading 9's rates through the bare tool-result arm: 15 and 15.

## Client and arm

**Client:** a fresh Haiku subagent per trial (`claude-haiku-4-5`).

**Arm: reading 11's, at these sizes.** Bash and Write only; two listed commands per cell —
`cat task.json`, then one `sed -n '1,Np' served.txt | nl -ba -v 1` (reading 9's single range, which
fits inline at these sizes), so the outer number of every row is its row index from the top and
`T + 2` on the target's row. Compliance as reading 8 pre-registered it: the listed set run,
nothing else, no Read/Grep/Glob before the Write, memory-tool calls after the Write not the trial's,
no spill.

## The two accounts, and what each predicts

- **H-position:** the weaker client takes the outer number for a target late in the file, whatever
  the file's size. Misses concentrate in the late cells at 20 KB and at 2 KB, at `T + 2`.
- **H-preceding-text:** the weaker client takes the outer number after enough numbered rows have
  gone by. No miss at 2 KB; few or none at 20 KB; any that appear are not late-only.
- Both predict every miss at exactly `T + 2`; a miss elsewhere is a third account, reported as such.

## Predictions, recorded before collection

1. The author expects **H-preceding-text**: 15 of 15 at 2 KB, and at 20 KB no more than one miss,
   not confined to late. Stated so it can be wrong; the author's expectation was wrong in readings
   10 and 11.
2. **Any miss sits at exactly `T + 2`.**
3. **Compliance at least 27 of 30.**
4. **Cost within 20% of reading 9's medians at the same sizes** (31,909 and 36,444), with the outer
   numbers' bytes on top.

## What this decides

Late-only misses at 20 KB, or any at 2 KB, mean position is the variable and the served-size curve
through this delivery has a shape that depends on where the target sits, not on how much was
served. No misses at either size, with reading 11's five at 200 KB, mean the amount of numbered text
before the target is the variable — which is a served-size effect, in the one delivery that lays a
plausible second number beside mrw's. Either way the tool-result arm without that number (readings
8 and 9) stays at the ceiling, and the cap does not move.

## What would void this reading

The cells, this plan or the harness changing after the first trial; a client reaching
`answer.json`.
