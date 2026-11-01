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
  mrw-version: "dev (de68a89, dirty)"
  generated-by: "mrw skill"
  content-sha256: "b6d26219297127b50023d502b403d4d65e0fc79cffc32432c13b6b107f33440b"
---

# mrw

The eight rules that matter most; read them once a session, and after a compaction only if lost:
1. Use mrw for every file read, edit and create: one read of every site, one plan, one write (mrw write PLANFILE); one change at many sites: mrw write --grep P --sub R.
2. Read before you write, per line: writing a line mrw never served you is refused, except in a file mrw just wrote. After a multi-line body, read on past your range until the enclosing structure closes.
3. A plan that fails validation writes nothing: read the refusal, which names file, plan line and reason; never --force. A commit that fails reports what reached disk.
4. A hunk is @@ PATH N-M replace anchor="TEXT", @@ PATH N insert-after or @@ PATH N-M delete, then its lines; anchor=, required on a multi-line replace, is copied from line N as read; omit body= unless mrw instructions names your case.
5. For a write: exit 0 fine, 1 a hunk failed and nothing was written, 2 usage or filesystem, 3 applied but the check did not pass. A read exits 1 when a range cannot be served; a range your own --max-lines or --max-matches withheld is not one: exits 0 (capped). Never read an exit code through a pipe: head's status replaces it.
6. Find with mrw read --grep PATTERN; start it with (?i) to ignore case; --stat lists the matching files; --max-cols N cuts long lines, leaving them unread; --no-numbers drops numbers.
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
5. different changes at several sites: one read of all of them, then the plan in a file — written with the host's file tool, in a temporary directory you may write — so no shell parser sees its text:
  mrw read 'a.go:/func A/,+12' b.go:40-60 && mrw write "$TMPDIR/plan.txt"
6. a step after the write, in the same call (NAME is a step in .quality-harness.json; a write runs no check unless --check asks for it; --then-sh and mrw run run any shell command they are given, so both are off unless MRW_SHELL=on is in mrw's environment (a step the project declares in .quality-harness.json runs with --then NAME); with them off, allowing mrw still runs the project's declared check and steps and no ad-hoc shell, but that is not containment: mrw writes any file its user can, .quality-harness.json included.):
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
12. run a long command (a build, a test suite, a gate) with its output kept out of your context, one string, instead of cmd | tail, which returns tail's status: a four-line receipt (on Windows a read line per shell, labelled) with the command's own exit code, the log's path and a read of it; read the log by pattern in a second call, so the exit code is not lost to it: mrw read --grep FAIL -C 3 - < 'LOG', LOG being the path the receipt printed (--then-sh and mrw run run any shell command they are given, so both are off unless MRW_SHELL=on is in mrw's environment (a step the project declares in .quality-harness.json runs with --then NAME); with them off, allowing mrw still runs the project's declared check and steps and no ad-hoc shell, but that is not containment: mrw writes any file its user can, .quality-harness.json included.; in Windows PowerShell 5.1 a command holding " reaches mrw as several arguments, so use single quotes inside the command, or pwsh 7):
  mrw run 'go test ./...'
13. a few fields out of a JSON document (cmd is any command that prints one), one line each instead of the lines that hold them (a path address of single-line JSON serves the whole line); the projection licenses nothing, so read the path address (mrw read f.json:$.a.b) before editing the value:
  cmd | mrw read --json-path '$.a,$.b' -
14. a long file's shape before its lines — a .go file's declarations or a .md or .markdown file's headings, each with the lines it spans; it licenses nothing, so read the lines you will edit after it:
  mrw read --outline f.go docs.md
15. matches across many files without the flood: count them, then read the first N (when any remain unshown they are counted, the answer says how many and the read exits 0 saying capped; a match inside a span already shown counts as shown):
  mrw read --grep P --count . && mrw read --grep P --max-matches 40 .
16. a log or a file of near-identical lines, each run shown once (only its first line is licensed: read the folded lines without --fold to edit them):
  mrw read --fold f.log
17. a Go file without its comment runs (the elided lines are not licensed; trailing comments, directives and the cgo preamble stay):
  mrw read --no-comments f.go
18. a setup that reads long lines: set MRW_MAX_COLS=200 in its environment and every read that serves lines cuts them as --max-cols 200 does, unless the call gives --max-cols itself (--max-cols 0 turns it off; a read that serves a line over 400 columns whole, with no width chosen, says so once on standard error):
  mrw read --grep P .
19. a write receipt of one line per file, every failure and advisory still printed in full:
  mrw write --terse plan.txt
20. answer by reference what you were already served, with --if-served PATH=SHA — only in a session alone in its checkout, whose host runs mrw seen --forget on start, resume, clear and compaction, and whose reads were made under --max-chars below the host's output limit, since a line recorded as shown may never have reached you (anywhere else, recipe 11):
  mrw read --if-served f.go=8dc37337 f.go:40-60

`mrw instructions` prints the whole contract. This file was printed by `mrw skill` from mrw dev (de68a89, dirty); when `mrw version` prints another version, `mrw skill --dir <the directory holding this mrw folder>` replaces it.

A host's per-path rules for its Edit and Write tools do not reach mrw, which writes wherever its user can: their absence is not permission, and mrw is not a way to reach a path the host refused you.
