#!/usr/bin/env bash
# Break campaign against mrw built from main. Every probe prints ONE line:
#   [name] exit=N  <what happened>
# and never aborts the script. A hang is caught by a 5s alarm (exit 142).
MRW=${MRW:?}
W=$(mktemp -d); export XDG_STATE_HOME="$W/state"; mkdir -p "$XDG_STATE_HOME"
t() { perl -e 'alarm shift; exec @ARGV' 5 "$@"; }
fresh() { R="$W/r$RANDOM$RANDOM"; mkdir -p "$R"; cd "$R" || exit 9; }
say() { printf '[%s] exit=%s  %s\n' "$1" "$2" "$(printf '%s' "$3" | tr '\n' ' ' | tr -s ' ')"; }  # one line per probe, whatever the tool printed
plan() { printf '%b' "$1" > "$R/p.plan"; }

# ---------- 1. pattern addresses in plans (ADR-013) ----------
fresh; printf 'alpha\nbeta gamma\nbeta\ndelta\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt /beta gamma/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-with-space" $? "$(sed -n 2p f.txt) | $out"
fresh; printf 'a/b\nc\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt /a\\/b/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-escaped-slash" $? "$(sed -n 1p f.txt) | $out"
fresh; printf 'one\ntwo\nthree\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt /three/,/one/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-end-before-start" $? "$(tr '\n' '|' < f.txt) | $out"
fresh; printf 'one\ntwo\nthree\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt /one/,/nomatch/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-end-never-matches" $? "$(tr '\n' '|' < f.txt) | $out"
fresh; printf 'one\ntwo\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt // replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-empty-regex" $? "$(tr '\n' '|' < f.txt) | $out"
plan '@@ f.txt /(/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-invalid-regex" $? "$out"
plan '@@ f.txt /one/ replace\nX\n@@ f.txt 1 replace\nY\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat+numeric-same-line" $? "$(tr '\n' '|' < f.txt) | $out"
fresh; printf 'one\r\ntwo\r\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt /two$/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-dollar-on-crlf" $? "$(od -c f.txt | head -2 | tr -s ' ' | tr '\n' ' ') | $out"
fresh; printf 'one\ntwo\n' > f.txt; t "$MRW" read f.txt:1 >/dev/null
plan '@@ f.txt /two/ replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-resolves-to-unserved-line" $? "$(tr '\n' '|' < f.txt) | $out"
fresh; printf 'one\ntwo\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt /one/ insert-after lines=1\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "pat-insert-after-lines-guard" $? "$(tr '\n' '|' < f.txt) | $out"

# ---------- 2. case-insensitive filesystem: two spellings of one file ----------
fresh; printf 'one\ntwo\nthree\n' > Same.txt; t "$MRW" read Same.txt same.txt >/dev/null 2>&1
plan '@@ Same.txt 1 replace\nX\n@@ same.txt 3 replace\nZ\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "case-two-spellings-one-plan" $? "file=$(tr '\n' '|' < Same.txt) | $out"
fresh; printf 'one\ntwo\n' > Same.txt; t "$MRW" read Same.txt >/dev/null
plan '@@ same.txt 1 replace\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "case-read-A-write-a" $? "file=$(tr '\n' '|' < Same.txt) | $out"

# ---------- 3. symlinks ----------
fresh; mkdir -p "$W/outside"; printf 'secret\n' > "$W/outside/s.txt"; ln -s "$W/outside/s.txt" link.txt
out=$(t "$MRW" read link.txt 2>&1); say "symlink-read-escapes-root?" $? "$out"
plan '@@ link.txt 1 replace\nOWNED\n'; out=$(t "$MRW" write --quiet --force p.plan 2>&1); say "symlink-write-through-force" $? "outside=$(cat "$W/outside/s.txt") islink=$([ -L link.txt ] && echo yes || echo no) | $out"
fresh; printf 'in\n' > real.txt; ln -s real.txt link.txt; t "$MRW" read link.txt >/dev/null 2>&1
plan '@@ link.txt 1 replace\nVIA-LINK\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "symlink-inside-write" $? "real=$(cat real.txt) islink=$([ -L link.txt ] && echo yes || echo no) | $out"
fresh; mkdir sub; ln -s "$W/outside" sub/esc; out=$(t "$MRW" read --grep secret sub/ 2>&1); say "grep-walk-symlink-dir-outside" $? "$out"
fresh; ln -s . loop; printf 'needle\n' > n.txt; out=$(t "$MRW" read --grep needle . 2>&1); say "grep-walk-symlink-loop" $? "$(echo "$out" | grep -c needle) matches, $(echo "$out" | tail -1 | cut -c1-60)"

# ---------- 4. a FIFO and an unreadable dir in a walked tree ----------
fresh; printf 'needle\n' > a.txt; mkfifo pipe; out=$(t "$MRW" read --grep needle . 2>&1); say "grep-walk-fifo" $? "$out"
fresh; printf 'needle\n' > a.txt; mkdir locked; printf 'needle\n' > locked/b.txt; chmod 000 locked; out=$(t "$MRW" read --grep needle . 2>&1); rc=$?; chmod 755 locked; say "grep-walk-unreadable-dir" $rc "$(echo "$out" | grep -ci 'locked\|permission') mentions of the dir | $(echo "$out" | tail -1 | cut -c1-70)"

# ---------- 5. degenerate files ----------
fresh; : > empty.txt; t "$MRW" read empty.txt >/dev/null
plan '@@ empty.txt 1 insert-before\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "empty-insert-before-1" $? "bytes=$(wc -c < empty.txt | tr -d ' ') | $out"
fresh; : > empty.txt; t "$MRW" read empty.txt >/dev/null
plan '@@ empty.txt $ insert-after\nX\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "empty-insert-after-dollar" $? "bytes=$(wc -c < empty.txt | tr -d ' ') content=$(tr '\n' '|' < empty.txt) | $out"
fresh; printf 'no newline at end' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt 1 replace\nstill none?\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "no-trailing-newline-preserved" $? "$(tail -c 1 f.txt | od -c | head -1 | tr -s ' ') | $out"
fresh; printf 'only\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt 1 delete\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "delete-only-line" $? "bytes=$(wc -c < f.txt | tr -d ' ') | $out"
fresh; printf 'a\rb\rc' > cr.txt; out=$(t "$MRW" read cr.txt 2>&1); say "cr-only-file-read" $? "$(echo "$out" | head -1 | cut -c1-40)"
fresh; printf 'bin\0ary\nline2\n' > b.bin; t "$MRW" read b.bin >/dev/null 2>&1
plan '@@ b.bin 2 replace\nL2\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "binary-nul-write" $? "$(od -c b.bin | head -1 | tr -s ' ' | cut -c1-50) | $out"
fresh; head -c 3000000 /dev/zero | tr '\0' 'x' > huge.txt; printf '\nend\n' >> huge.txt; out=$(t "$MRW" read huge.txt:2 2>&1); say "3MB-single-line-read-line-2" $? "$(echo "$out" | tail -1 | cut -c1-30)"

# ---------- 6. overlapping and duplicate hunks ----------
fresh; printf '1\n2\n3\n4\n' > f.txt; t "$MRW" read f.txt >/dev/null
plan '@@ f.txt 1-3 replace\nA\n@@ f.txt 2 replace\nB\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "overlap-range-and-line" $? "$(tr '\n' '|' < f.txt) | $out"
plan '@@ f.txt 2 replace\nB\n@@ f.txt 2 insert-after\nC\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "replace+insert-after-same-line" $? "$(tr '\n' '|' < f.txt) | $out"
plan '@@ f.txt 2 delete\n@@ f.txt 2 replace\nB\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "delete+replace-same-line" $? "$(tr '\n' '|' < f.txt) | $out"

# ---------- 7. header grammar edge ----------
fresh; printf 'x\n' > 'my file.txt'; t "$MRW" read 'my file.txt' >/dev/null
plan '@@ "my file.txt" 1 replace\nY\n'; out=$(t "$MRW" write --quiet p.plan 2>&1); say "quoted-path-with-space" $? "$(cat 'my file.txt') | $out"
fresh; printf 'x\n' > f.txt; t "$MRW" read f.txt >/dev/null
printf '\xef\xbb\xbf@@ f.txt 1 replace\nY\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "plan-with-BOM" $? "$(cat f.txt) | $out"
printf '@@ f.txt 1 replace\r\nY\r\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "plan-with-CRLF" $? "$(od -c f.txt | head -1 | tr -s ' ') | $out"
printf '@@\tf.txt\t1\treplace\nZ\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "plan-tab-separated-header" $? "$(cat f.txt) | $out"
printf '@@ f.txt 1 replace sha=%s\nQ\n' "$(shasum -a 256 f.txt | cut -c1-64 | tr a-f A-F)" > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "sha-guard-uppercase-hex" $? "$(cat f.txt) | $out"
printf '@@ f.txt 1 replace lines=0\nQ\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "lines=0-guard" $? "$(cat f.txt) | $out"
printf '@@ f.txt 1 replace\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "replace-with-empty-body" $? "bytes=$(wc -c < f.txt | tr -d ' ') | $out"
printf '@@ f.txt - create\nnew\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "create-over-existing-file" $? "$(cat f.txt) | $out"
printf '@@ sub/dir/new.txt - create\nnew\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "create-in-missing-dir" $? "$(cat sub/dir/new.txt 2>&1 | head -1) | $out"
printf '@@ ../escape.txt - create\nx\n' > p.plan; out=$(t "$MRW" write --quiet p.plan 2>&1); say "create-outside-root" $? "$([ -e ../escape.txt ] && echo ESCAPED || echo contained) | $out"
printf '@@ f.txt 1 replace\nQ\n' | t "$MRW" write --quiet --dry-run - >/dev/null 2>&1; rc=$?; l=$(t "$MRW" seen 2>/dev/null | grep -c 'f.txt'); say "dry-run-records-nothing-new?" $rc "ledger lines for f.txt=$l (1 from the read is expected)"

# ---------- 8. MCP stdio: a request line over bufio's 64KB default ----------
fresh; printf 'one\ntwo\n' > f.txt
big=$(head -c 120000 /dev/zero | tr '\0' 'x')
req=$(python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ f.txt - create\n"+sys.argv[1]+"\n","dry_run":True}}}))' "$big")
out=$(printf '%s\n' "$req" | t "$MRW" mcp 2>&1); rc=$?; say "mcp-120KB-request-line" $rc "$(echo "$out" | head -c 160 | tr '\n' ' ')"
out=$(printf '[{"jsonrpc":"2.0","id":1,"method":"tools/list"}]\n' | t "$MRW" mcp 2>&1); say "mcp-batch-array" $? "$(echo "$out" | head -c 120 | tr '\n' ' ')"
out=$(printf '{"jsonrpc":"2.0","method":"tools/list"}\n{"jsonrpc":"2.0","id":"s","method":"tools/list"}\n' | t "$MRW" mcp 2>&1); say "mcp-notification-then-string-id" $? "$(echo "$out" | grep -o '"id":"s"' | head -1) responses=$(echo "$out" | grep -c jsonrpc)"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":"f.txt"}}}\n' | t "$MRW" mcp 2>&1); say "mcp-specs-as-string" $? "$(echo "$out" | head -c 140 | tr '\n' ' ')"
out=$(printf 'not json at all\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n' | t "$MRW" mcp 2>&1); say "mcp-garbage-then-valid" $? "responses=$(echo "$out" | grep -c jsonrpc) $(echo "$out" | head -c 100 | tr '\n' ' ')"

# ---------- 9. state and env ----------
fresh; printf 'x\n' > f.txt; out=$(t env -u HOME -u XDG_STATE_HOME "$MRW" read f.txt 2>&1); say "no-HOME-no-XDG" $? "$out"
out=$(XDG_STATE_HOME=/dev/null/nope t "$MRW" read f.txt 2>&1); say "unwritable-state-home" $? "$out"
echo "campaign dir: $W"
