# Counting matches and reading long lines in OpenCode

On 2026-11-26, two existing MCP options were tested through the OpenCode plugin. The first hypothesis was that a count could answer a 200-line search in one call with at least 80% fewer returned bytes. The second was that a width limit could show the match on a line larger than the plugin's 40,000-character result ceiling. Both held in the fixtures below.

## Results

Each cell is one call to the built plugin's `mrw_read.execute` with its normal checkout context and a real `mrw mcp` child. Bytes are the UTF-8 length of the returned `output`, including its receipt.

| Question | Existing grep response | With the forwarded option |
| --- | --- | --- |
| How many of 200 inventory lines match? | 16,362 bytes, 200 numbered lines | `count: true` returned 272 bytes and the total 200 |
| How many numbered sections are in the repository's contract script? | 445 bytes, a file index with no count | `count: true` returned 299 bytes and the total 370 |
| What surrounds the match in a 72,008-character line? | 433 bytes, a file index with no line content | `maxCols: 100` returned 442 bytes including the match, its surrounding text and its column range |

The first search saved 16,090 response bytes (98.3%). The third used nine additional bytes and answered the question. In the contract-script case, `rg -c '^# [0-9]+\. ' scripts/contract.sh` independently returned `370` and a newline, four bytes. This was a comparison of tool responses. Model tokens, billing, end-to-end agent success and user time were unmeasured.

## Repeating the cases

The comparison used macOS arm64, Node v26.13.1, Go 1.27.4 and ripgrep 15.2.0. The baseline plugin and Go binary were built from `167b9e50`. The contract-script fixture was taken from `6cf8d53c`. The Go sources and baseline plugin source are identical between those revisions. Build the plugin and binary as described in [the repository checks](../../AGENTS.md). The [plugin smoke tests](../../cmd/opencode/mrw-plugin/test/smoke.test.mjs) show how to create an isolated checkout and call its registered tools.

Use a fresh state directory for each experiment. Pass the single file in `specs`, along with the indicated `grep`. On the baseline omit both new options. On the candidate add the option from the table. Measure `Buffer.byteLength(result.output, "utf8")` and inspect the answer.

- **Inventory**. Name the file `fixture.txt`. Generate 200 LF-terminated lines numbered 000 through 199. Each starts with `MATCH `, then the three digits, then ` inventory record with calibration details and processing state`. Search for `MATCH`.
- **Contract**. Run from the checkout root with `specs: ["scripts/contract.sh"]` and the regexp shown above. The numbered sections change as the repository evolves.
- **Long line**. Name the file `log.txt`. Concatenate `"ab".repeat(18000)`, `" NEEDLE "`, `"yz".repeat(18000)` and a newline. Search for `NEEDLE`. The shortened line carries no checkpoint. The smoke test also attempts an edit to verify the licence boundary.

## Why these comparisons

OpenCode's own tool guide uses a search followed by a targeted read as a coding workflow. Anthropic recommends evaluating whether the tool returns the information an agent needs and bounding large results. Aider makes a related economy on the write side by sending only the changed sections in its search/replace format. These informed the two questions above. The measurements cover the OpenCode adapter only. [OpenCode tools](https://opencode.ai/docs/tools/) · [Anthropic on evaluating agent tools](https://www.anthropic.com/research/writing-tools-for-agents) · [Aider edit formats](https://aider.chat/docs/more/edit-formats.html) · [ripgrep's benchmark methodology](https://burntsushi.net/ripgrep/)
