# Conditional rereads in OpenCode

When an agent revisits two files, how much data does it receive if it supplies the hashes from its previous read? On 2026-11-26 we called the OpenCode plugin's `mrw_read.execute` against a real `mrw mcp` child. Both calls requested the same ranges. The conditional call also passed `ifChanged`, a map from file paths to the eight hex digits printed in their earlier headers.

| Files changed since the first read | Ordinary response, bytes | Conditional response, bytes | Extra request bytes | Net bytes saved on the reread |
| --- | ---: | ---: | ---: | ---: |
| Neither fixture file | 13,186 | 329 | 56 | 12,801 |
| Only the short notes file | 13,207 | 748 | 56 | 12,403 |
| Both fixture files | 13,330 | 13,330 | 56 | −56 |
| Actual adapter change, two source ranges | 11,394 | 3,152 | 97 | 8,145 |

The ordinary and conditional responses include the full plugin output and receipt. Extra request bytes are the difference in UTF-8 length between the two compact JSON argument objects. Net savings subtract that request overhead from the response reduction. The initial read is common to both approaches.

For the fixture, `src.ts` had 200 LF-terminated lines, each `export const metricN = N // observed inventory count`, with decimal N from 0 through 199. `notes.md` began with `Review the inventory totals.` and a newline. The first change appended `Count confirmed.` to the notes. The second appended `export const verified = true` to the source, each with a final newline. Each reread used the hashes from the initial read. Every unchanged file returned a notice and every changed file returned its lines and checkpoints.

For the repository case, start with the files at `6ab15973` and read `internal/mcp/mcp.go:600-664` and `cmd/opencode/mrw-plugin/src/index.ts:150-175`. Then replace only the adapter with the version that introduces `ifChanged`, and reread the same ranges. The conditional response served the changed adapter lines and a notice for the unchanged MCP source.
