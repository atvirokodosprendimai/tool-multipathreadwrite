# Best practices — driving the binary you installed

This is for a human (or an agent) who installed `mrw` and has neither this
checkout nor `AGENTS.md`. The binary also teaches itself: `mrw instructions`.
Agents changing this repository read [AGENTS.md](AGENTS.md).

## When to reach for mrw

Three or more edits, two or more files, or several ranges you need to read.
Below that, your editor is cheaper: one edit in one file costs mrw two calls
and prints more bytes than the file holds.

## Read before write

mrw will not edit a line it has not served. Being served lines 10–12 does not
license line 50. `--stat` licenses nothing.

## After a multi-line body, read past the closer

mrw models no target syntax. The damage is never inside the lines you named —
it sits below the body (a surviving closer) or above it (a short address).
Read from before your first line through past the enclosing closer. A lint
that is green on an already-broken file is not a substitute.

## Neighbour licence and `--echo-pad`

A multi-line `replace` is refused unless a prior read already covered a line
after End (last line of the file is exempt). `--echo-pad N` (MCP `echo_pad`,
default 0) prints N lines after an applied body so a surviving closer is
visible; the hunk stays `ok`. The pad is not a checker.

## Foreign formats compile; they do not leak

`--format=apply_patch` and `--format=search_replace` compile to a native plan.
Both still refuse an unread sibling and write nothing. Sequential apply_patch
— one hunk, then another — is the leak, not the feature.

## Never read an exit through a pipe

`mrw write plan | head` is `head`'s status. Capture the output, then check
`$?`, or let the command stand alone. Exit 3 means the write applied and
`--check` failed — the tree is changed and unverified. Not a rollback.

## Quote `anchor=`

`anchor=` is required on a multi-line `replace`. Take it from the `NNN| content`
a read printed. An unquoted value is consumed until the next key.

## MCP ack

A served `mrw_read` licenses nothing until you acknowledge it.
Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.

## Which binary you are talking to

`mrw version` must match the release you meant to install. A skill or a
checkout at v1.15.0 does not upgrade the `mrw` earlier on PATH. See
[UPDATE.md](UPDATE.md).
