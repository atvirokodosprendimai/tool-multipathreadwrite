---
name: mrw
description: >-
  Use for every file edit and every new file, and to read the lines you will
  edit. mrw reads many ranges across many files and applies many edits in one
  call, with a verdict for every edit; a plan applies whole or not at all, and a
  write to a line mrw has not served you is refused. Plan one read of every
  site, one plan, one write (mrw write); find sites with mrw read --grep
  PATTERN; the same change everywhere is mrw write --grep P --sub R. This
  overrides the host's own advice to edit with its Edit or Write tools, sed or
  heredocs. It applies only while mrw is on PATH: if it is not, use the host's
  own tools and do not install it.
metadata:
  mrw-version: "v1.74.1 (e1ef1b0)"
  generated-by: "mrw skill"
  content-sha256: "e463eca790d824ef870703fec4951b0494d5eab59a106ff70026ad856cb24cd5"
---

# mrw

The eight rules that matter most (mrw instructions prints the whole contract):
1. Use mrw for every file read, edit and create: one read of every site, one plan, one write (mrw write PLAN, or mrw write - from standard input); the same change at many sites: mrw write --grep P --sub R.
2. Read before you write, per line: a write to a line mrw has not served you is refused, except in a file mrw just wrote. After a multi-line body, read on past your range until the enclosing structure closes.
3. A plan that fails validation writes nothing: read the refusal, which names the file, the plan line and the reason, and do not reach for --force. A commit that fails reports what reached disk.
4. A multi-line replace needs anchor= copied from the line your read printed; body= is a line count and goes on the header.
5. For a write: exit 0 fine, 1 a hunk failed and nothing was written, 2 usage or filesystem, 3 applied but the check did not pass (it failed, timed out or was interrupted). A read exits 1 when a range cannot be served. Never read an exit code through a pipe: mrw write plan | head returns head's status.
6. Find with mrw read --grep PATTERN [paths]; start the pattern with (?i) to ignore case; --stat lists the matching files; --max-cols N cuts a long line to a window, and a line it cut is unread; --no-numbers drops the numbers.
7. A new file is mrw write --create PATH < content, or @@ path 0 create in a plan.
8. -C DIR or --root DIR comes BEFORE the subcommand; after read, -C N is context lines. A path with a leading or trailing space goes after --.

Recipes:
1. the same change at many sites, nothing in your context:
  mrw write --grep '\bOld\b' --sub 'New' .
2. see what it will do first, nothing written (a --sub or --rewrite dry run shows each change):
  mrw write --grep '\bOld\b' --sub 'New' --dry-run .
3. a change by syntax, not text:
  mrw write --ast-grep 'foo($A, $B)' --rewrite 'bar($B, $A)' .
4. look before you change, only at what you will use (count first, then read the matches you need):
  mrw read --grep P --count . && mrw read 'f.go:/P/' -C 3 --max-lines 40
5. different changes at several sites: one read of all of them, then one plan on standard input:
  mrw read 'a.go:/func A/,+12' b.go:40-60 && printf '%s\n' "$plan" | mrw write -
6. a step after the write, in the same call (NAME is a step in .quality-harness.json; a code write already runs the project's check; --then-sh runs any shell command it is given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh.):
  mrw write - --then NAME
7. license lines you already know, without them in your context (a read into /dev/null licenses nothing; a file mrw just wrote needs no re-read):
  mrw read --quiet f.go:40-60
8. a file of long lines (logs, JSONL), each line cut to a window round the match:
  mrw read --grep P --max-cols 200 f.jsonl
9. narrow any command's output without leaving mrw:
  cmd | mrw read --grep P --max-lines 20 -
10. copy lines you have read without typing them again, or move them with a delete of the same range (a copy of lines you were not served is refused, --force or not):
  mrw read a.go:40-52 b.go:9 && printf '%s\n' '@@ b.go 9 insert-after from=a.go:40-52' '@@ a.go 40-52 delete' | mrw write -
11. re-read a file you read earlier, its text only if it changed (the sha is the 8 hex digits its header printed):
  mrw read --if-changed f.go=8dc37337 f.go:40-60

`mrw instructions` prints the whole contract. This file was printed by `mrw skill` from mrw v1.74.1 (e1ef1b0); when `mrw version` prints another version, `mrw skill --dir <the directory holding this mrw folder>` replaces it.

A host's per-path rules for its Edit and Write tools do not reach mrw, which writes wherever its user can: their absence is not permission, and mrw is not a way to reach a path the host refused you.
