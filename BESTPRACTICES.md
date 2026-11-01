# Best practices — driving the binary you installed

This is for callers using an installed `mrw`. [README.md](README.md) is the interface
reference; [agent setup](docs/agents.md) covers host instructions and skills. Agents
changing this repository also follow [AGENTS.md](AGENTS.md).

## When to reach for mrw

Always. Plan the activity as one read of every site, then one plan, then one
write. One edit in one file still goes through mrw; the read ledger is the point.
This is the usage policy, not a claim that every task becomes faster or cheaper.

Read `mrw instructions --core` once at session start. After compaction, re-read the
lines you will edit; re-read the rules only if they are gone from your context.
Use `mrw instructions` for the recipes or when a refusal names a case the core does not.

The same change at many sites is one command:
`mrw write --grep '\bOld\b' --sub 'New' .` (add `--dry-run` to preview it).
Count before reading (`mrw read --grep P --count .`), or license known lines without
printing them (`mrw read --quiet f.go:40-60`). Narrow another command's output with
`cmd | mrw read --grep P --max-lines 20 -`.

## Read before write

A write needs a current read licence for the lines it changes. Being served lines
10–12 does not license line 50. A file mrw just wrote is wholly known;
`--stat` and `--outline` license nothing.

## Prepare the plan

Send a plan in a temporary file written with the host's file tool, then run
`mrw write PLANFILE`. The plan file is input, not a repository edit, so the host's
file tool may write it. This keeps code in the plan out of the host's shell parser.

`anchor=` is required on a multi-line `replace`; `sha=` and `lines=` do not stand in
for it. Quote the anchor and copy its value from the content a read printed.
Omit `body=` for an ordinary inline body; `mrw instructions` names the exceptional
cases that need a count.

## After a multi-line body, read past the closer

mrw models no target syntax or semantics. A wrong range can leave a surviving closer
below the body or unwanted lines above it; a correct range can still hold incorrect code.
Read the result from before the changed range through the enclosing closer, then run
the project's checks.

A multi-line `replace` needs a prior read covering a line after its end (the last line
of the file is exempt). `--echo-pad N` (MCP `echo_pad`, default 0) prints lines after an
applied body. The hunk stays `ok`; the pad is not a checker.

## Check the final tree

A write runs no check unless `--check` asks for it (ADR-187). `--check` runs the check the project declares,
or mrw infers for a Go module; other stacks need an explicit project check; see [README: Write](README.md#write).

For a queue of code edits, put `--check` on the last write, or run `mrw check` after it. Verify a
check covering every queued edit after the last write; a checked last write may have narrower
scope. Follow the [queue recipe](docs/agents.md#queue-code-edits-then-check-the-final-tree),
including its deletion/rename and checker-configuration guidance.

## Keep the command's exit status

`mrw write plan | head` reports `head`'s status. Let the command stand alone, or capture
its output and save its exit status immediately. Exit 3 means the write applied but its
check or step did not pass: the tree remains changed and unverified.

Declared checks and named `--then` steps use project configuration. Ad-hoc `--then-sh`
steps and `mrw run` require `MRW_SHELL=on` in mrw's environment. Enable that deliberately;
it is not required for ordinary reads, writes or declared checks.

## Foreign formats compile to one plan

`--format=apply_patch` and `--format=search_replace` compile to a native plan.
They keep the read guard and whole-plan validation; an unread sibling refuses the plan
before anything is written.

## MCP acknowledgements

Use the CLI when the host has a shell. A host without a shell enables the MCP server
with `mrw mcp --enable` or `MRW_MCP=on`; the tools are off without an explicit opt-in.

Numbered lines served by `mrw_read` need checkpoint acknowledgement before they license a write.
Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
Quiet and plan-making reads license their ranges without printing numbered lines or requiring ack.

## Which binary you are talking to

Check `command -v mrw` and `mrw version`. Updating a skill or a checkout does not replace
another binary earlier on PATH. See [UPDATE.md](UPDATE.md) for release, Homebrew and
`go install` version checks.
