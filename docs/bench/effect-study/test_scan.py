"""Self-test for scan.py on synthetic transcripts whose answers are known.

Run: python3 -I docs/bench/effect-study/test_scan.py   (exit 0 green, 1 red)
Each case is its own small transcript, so a count has one cause.
"""
import json
import os
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True                   # no __pycache__ in the tree
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import scan  # noqa: E402

N = [0]
BAD = []
CHECKS = [0]
U = {"input_tokens": 100, "output_tokens": 200, "cache_read_input_tokens": 10000,
     "cache_creation_input_tokens": 400, "cache_creation": {"ephemeral_1h_input_tokens": 300}}
OK1 = "1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied\n"


def stamp():
    """Return the next timestamp, one second after the last, in November 2026."""
    N[0] += 1
    return "2026-11-01T%02d:%02d:%02dZ" % (N[0] // 3600, N[0] // 60 % 60, N[0] % 60)


def use(mid, tid, name, inp, usage=None):
    """One assistant entry holding one tool call."""
    msg = {"id": mid, "content": [{"type": "tool_use", "id": tid, "name": name, "input": inp}]}
    if usage:
        msg["usage"] = usage
    return {"type": "assistant", "timestamp": stamp(), "message": msg}


def res(tid, text):
    """One user entry holding one tool result."""
    return {"type": "user", "timestamp": stamp(),
            "message": {"content": [{"type": "tool_result", "tool_use_id": tid, "content": text}]}}


def bash(mid, tid, cmd, out, usage=None):
    """A shell call and its result."""
    return [use(mid, tid, "Bash", {"command": cmd}, usage), res(tid, out)]


def compact():
    """A compaction boundary."""
    return {"type": "system", "subtype": "compact_boundary", "timestamp": stamp()}


def writes(prefix, n):
    """n mrw writes of one hunk each."""
    out = []
    for i in range(n):
        out += bash("%s%d" % (prefix, i), "t%s%d" % (prefix, i), "mrw write --terse p%d.txt" % i, OK1)
    return out


def edits(prefix, n):
    """n host Edit calls."""
    return [use("%s%d" % (prefix, i), "t%s%d" % (prefix, i), "Edit", {"file_path": "/r/e%d.go" % i}) for i in range(n)]


def scan_of(entries, **kw):
    """Write entries as one transcript in a fresh folder and scan it."""
    with tempfile.TemporaryDirectory() as d:
        os.mkdir(os.path.join(d, "p"))
        with open(os.path.join(d, "p", "s.jsonl"), "w", encoding="utf-8") as fh:
            for o in entries:
                fh.write((o if isinstance(o, str) else json.dumps(o)) + "\n")
        return scan.run(d, **kw)


def check(name, got, want):
    """Compare one value and remember a mismatch."""
    CHECKS[0] += 1
    if got != want:
        BAD.append("%s = %r, want %r" % (name, got, want))


def refusal(path):
    """The line mrw prints for a file changed since it was read."""
    return "FAIL %s 1 replace: %s changed since mrw last saw it (recorded a, now b)\n" % (path, path)


def served(path):
    """The answer of a one-line read of path."""
    return "==> %s  3L  9B  sha aaaaaaaa\n@@ 1-1\n 1| x\n" % path


def test_cost():
    """E1: usage counts once per message id, on entries with and without a tool call."""
    e = bash("m1", "t1", "mrw read a.go:1-3 b.go:5 b.go:9", "==> a.go  9L\n@@ 1-3\n 1| x\n==> b.go  9L\n@@ 5-5\n@@ 9-9\n", U)
    e.append({"type": "assistant", "timestamp": stamp(), "message": {"id": "m1", "usage": U, "content": []}})
    e.append({"type": "assistant", "timestamp": stamp(),               # usage with no tool call at all
              "message": {"id": "m2", "usage": U, "content": [{"type": "text", "text": "hi"}]}})
    e.append("not json")
    r = scan_of(e)
    check("requests", r["corpus"]["requests"], 2)
    tok = r["E1"]["tokens"]
    check("tokens", (tok["input"], tok["output"], tok["cache_read"], tok["cache_write_1h"], tok["cache_write_5m"]),
          (200, 400, 20000, 600, 200))
    # priced: 200 + 2000 + 2000 + 1200 + 250 = 5650; cache = 3450
    check("cache_share_pct", r["E1"]["cache_share_pct"], round(100 * 3450 / 5650, 3))
    check("three ranges in one read", r["E1"]["sites_per_shell_call_running_mrw"],
          {"n": 1, "mean": 3.0, "median": 3.0, "p90": 3, "share_of_one": 0.0})


def test_sites():
    """E1: sites per shell call and per request; a repeated tool call counts once; a mixed request keeps both."""
    e = bash("m1", "t1", "mrw read a.go:1", served("a.go"))                       # 1 site
    e += bash("m2", "t2", "mrw write p.txt", "3 hunk(s), 2 file(s), 0 failed, 0 advisories — applied\n")  # 3
    e.append(use("m3", "t3", "Read", {"file_path": "/r/c.go"}))
    e.append(use("m3", "t3", "Read", {"file_path": "/r/c.go"}))                   # the same call again
    e.append(use("m3", "t4", "Read", {"file_path": "/r/d.go"}))
    e.append(use("m4", "t5", "Edit", {"file_path": "/r/d.go"}))                   # mixed: 1 host + 1 mrw site
    e += bash("m4", "t6", "mrw read b.go:1", served("b.go"))
    e += bash("m5", "t7", "mrw read --grep write f.go", "==> f.go  3L\n@@ 1-1\n 1| write\n")   # a read, not a write
    e += bash("m6", "t8", 'echo "mrw write p"', "mrw write p\n")                  # no mrw call at all
    r = scan_of(e)
    check("shell calls running mrw", r["E1"]["sites_per_shell_call_running_mrw"],
          {"n": 4, "mean": 1.5, "median": 1.0, "p90": 1, "share_of_one": 75.0})
    check("requests with mrw", r["E1"]["sites_per_request_with_mrw"],
          {"n": 4, "mean": 1.75, "median": 1.5, "p90": 2, "share_of_one": 50.0})
    check("requests with host tools only", r["E1"]["sites_per_request_native_only"],
          {"n": 1, "mean": 2.0, "median": 2.0, "p90": 2, "share_of_one": 0.0})
    check("mrw_write_calls", r["corpus"]["mrw_write_calls"], 1)


def test_plan():
    """E2 (b): hunks, anchors in each spelling, single-line replaces, misses."""
    plan = ("mrw write - <<'EOF'\n"
            '@@ a.go 1-3 replace anchor="func A"\nx\n'
            "@@ a.go 7-9 replace anchor='func B'\nx\n"
            "@@ a.go 12-14 replace anchor=func C body=1\nx\n"
            "@@ b.go 5 replace\ny\n"
            "@@ b.go 6-6 replace\ny\n"
            '@@ "c d.go" 9 insert-after\nz\n'
            "EOF")
    miss = 'FAIL a.go 1-3 replace (plan line 1): anchor "func A" not in line 1: "x"; the anchor is in line 2\n'
    e = bash("m1", "t1", plan, miss) + bash("m2", "t2", "mrw write p.txt", miss)   # the same miss, sent twice
    e.append(use("m3", "t3", "Write", {"file_path": "/tmp/p.txt", "content": "@@ a.go 2 replace\nq\n"}))
    r = scan_of(e)["E2"]
    check("hunks_typed", r["hunks_typed"], 7)
    check("anchored_hunks", r["anchored_hunks"], 3)
    check("anchor_chars", r["anchor_chars"], 18)
    check("bare_single_line_replaces", r["bare_single_line_replaces"], 3)
    check("anchor_miss_lines", r["anchor_miss_lines"], 2)
    check("anchor_misses_distinct", r["anchor_misses_distinct"], 1)


def test_tags():
    """E2 (a): per line, with no phantom line after a final newline."""
    with tempfile.TemporaryDirectory() as repo:
        with open(os.path.join(repo, "t.txt"), "w", encoding="utf-8") as fh:
            fh.write("line13\nline18\nother0\na\na\n")       # sha1 of line13 and line18 both begin "40"
        with open(os.path.join(repo, "u.txt"), "w", encoding="utf-8") as fh:
            fh.write("line13\nline18\nline13\n")             # the middle line collides on both sides: once
        with open(os.path.join(repo, "v.txt"), "w", encoding="utf-8") as fh:
            fh.write("row3\nrow13")                          # both begin "21f"; no final newline
        git = ["git", "-C", repo]
        subprocess.run(git + ["init", "-q"], check=True)
        subprocess.run(git + ["add", "t.txt", "u.txt", "v.txt"], check=True)
        tc = scan.tag_collisions(repo)
    check("lines", tc["lines"], 10)
    check("identical neighbours", tc["lines_with_identical_neighbour"], 2)
    check("different neighbours, same 2 hex", tc["lines_same_2hex_as_different_neighbour"], 7)
    check("different neighbours, same 3 hex", tc["lines_same_3hex_as_different_neighbour"], 2)
    check("2 hex share", tc["lines_same_2hex_as_different_neighbour_pct"], 70.0)


def test_compaction():
    """E3: the window is 30 calls each side, and a neighbouring compaction does not cut it."""
    r = scan_of(writes("w", 35) + [compact()] + edits("n", 35))["E3"]
    check("window before", r["before"], {"native": 0, "calls": 30})
    check("window after", r["after"], {"native": 30, "calls": 30})
    check("paired difference", r["mean_paired_difference_pp"], 100.0)
    r = scan_of(writes("a", 6) + [compact()] + writes("b", 4) + [compact()] + edits("n", 6))["E3"]
    check("both neighbouring compactions pair", r["compactions_paired"], 2)
    check("before, pooled", r["before"], {"native": 0, "calls": 16})         # 6, then 10
    check("after, pooled", r["after"], {"native": 12, "calls": 16})          # 6 of 10, then 6 of 6
    r = scan_of(edits("n", 6) + [compact()] + writes("w", 6))["E3"]
    check("a compaction before any mrw write is left out", (r["compactions_paired"], r["compactions_before_any_mrw_write"]), (0, 1))
    r = scan_of(writes("w", 6) + [compact()] + edits("n", 4))["E3"]
    check("fewer than five on a side is left out", (r["compactions_paired"], r["compactions_too_few_calls"]), (0, 1))


def test_changed():
    """E4: the cause is looked for since mrw last served the path, and only an Edit of it is certain."""
    e = bash("m1", "t1", "mrw read f.go:1", served("f.go"))
    e.append(use("m2", "t2", "Edit", {"file_path": "/r/f.go"}))
    e += bash("m3", "t3", "mrw read f.go:1", "==> f.go  UNREADABLE  open f.go: permission denied\n")   # serves nothing
    e += bash("m4", "t4", "mrw write p.txt", refusal("f.go"))                # own_edit
    e += bash("m5", "t5", "mrw read g.go:1 && gofmt -w .", served("g.go"))
    e += bash("m6", "t6", "mrw write p.txt", refusal("g.go"))                # own_shell: gofmt in the read's command
    e += bash("m7", "t7", "mrw read h.go:1", served("h.go"))
    e.append(use("m8", "t8", "Agent", {"prompt": "x"}))
    e += bash("m9", "t9", "mrw write p.txt", refusal("h.go"))                # own_subagent
    e += bash("m10", "t10", "gofmt -w .", "")                                # before the read: not a cause
    e += bash("m11", "t11", "mrw read i.go:1", served("i.go"))
    e += bash("m12", "t12", "mrw write p.txt", refusal("i.go"))              # nothing_found
    r = scan_of(e)
    check("refusals", r["E4"]["changed_since_refusals"], 4)
    check("causes", (r["E4"]["own_edit"], r["E4"]["own_shell"], r["E4"]["own_subagent"], r["E4"]["nothing_found"]),
          (1, 1, 1, 1))
    check("nothing found per write", r["E4"]["nothing_found_per_mrw_write_pct"], 25.0)
    check("not an own edit per write", r["E4"]["not_own_edit_per_mrw_write_pct"], 75.0)


def test_shell():
    """What counts as a shell call that runs mrw: not quoted text, not a heredoc body; a quoted root does."""
    e = bash("m1", "t1", 'echo "x; mrw write p"', "x; mrw write p\n")
    e += bash("m2", "t2", "cat > note.txt <<'EOF'\nmrw write p\nEOF", "")
    e += bash("m3", "t3", 'mrw --root "/tmp/some dir" write p', OK1)
    e += bash("m4", "t4", 'out="$(mrw read a.go:1)"', "")                         # a read that printed nothing
    e.append(use("m4", "t5", "Read", {"file_path": "/r/c.go"}))
    r = scan_of(e)
    check("only the real write is a write call", r["corpus"]["mrw_write_calls"], 1)
    check("a call of zero sites is a call", r["E1"]["sites_per_shell_call_running_mrw"],
          {"n": 2, "mean": 0.5, "median": 0.5, "p90": 0, "share_of_one": 50.0})
    check("a request whose mrw call served nothing is still an mrw request",
          (r["E1"]["sites_per_request_with_mrw"]["n"], r["E1"]["sites_per_request_native_only"]["n"]), (2, 0))


def test_plan_edges():
    """E2 (b): a pattern address with a space; the same miss text in two files is two misses."""
    plan = "mrw write - <<'EOF'\n@@ f.go /^func A/,/^}/ replace anchor=\"func A\"\nx\nEOF"
    miss = ('FAIL a.go 1 replace (plan line 1): anchor "a" not in line 1: "x"; the anchor is in line 2\n'
            'FAIL b.go 1 replace (plan line 3): anchor "a" not in line 1: "x"; the anchor is in line 2\n')
    r = scan_of(bash("m1", "t1", plan, miss))["E2"]
    check("a spaced pattern address is a hunk", (r["hunks_typed"], r["anchored_hunks"], r["anchor_chars"]), (1, 1, 6))
    check("misses in two files are distinct", (r["anchor_miss_lines"], r["anchor_misses_distinct"]), (2, 2))
    check("miss ratios", (r["anchor_miss_lines_per_anchored_hunk_pct"], r["anchor_misses_distinct_per_anchored_hunk_pct"]),
          (200.0, 200.0))


def test_changed_edges():
    """E4: a dry run and a bare header refresh nothing; a formatter before the read is no cause; spaced paths."""
    dry = "ok   f.go 1 replace  -1 +1\n1 hunk(s), 1 file(s), 0 failed, 0 advisories — dry run, nothing written\n"
    e = bash("m1", "t1", "mrw read f.go:1", served("f.go"))
    e.append(use("m2", "t2", "Edit", {"file_path": "/r/f.go"}))
    e += bash("m3", "t3", "mrw write --dry-run --force p.txt", dry)
    e += bash("m4", "t4", "mrw read --stat f.go", "==> f.go  3L  9B  sha bbbbbbbb\n")
    e += bash("m5", "t5", "mrw write p.txt", refusal("f.go"))                # own_edit, through both
    e += bash("m6", "t6", "gofmt -w . ; mrw read g.go:1", served("g.go"))
    e += bash("m7", "t7", "mrw write p.txt", refusal("g.go"))                # nothing_found: gofmt ran first
    e += bash("m8", "t8", "mrw read 'dir/a b.go:1'", served("dir/a b.go"))
    e.append(use("m9", "t9", "Edit", {"file_path": "/r/dir/a b.go"}))
    e += bash("m10", "t10", "mrw write p.txt", refusal("dir/a b.go"))        # own_edit, path with a space
    r = scan_of(e)["E4"]
    check("edge causes", (r["changed_since_refusals"], r["own_edit"], r["own_shell"], r["nothing_found"]), (3, 2, 0, 1))


def test_json_and_quiet():
    """A --json answer counts as the plain one does, and a quiet write refreshes the paths its plan names."""
    read_doc = json.dumps({"content": served("f.go"), "receipt": {"problems": 0}}, indent=2)
    wrote = json.dumps({"dry_run": False, "applied": True, "failed": 0,
                        "files": [{"path": "g.go", "written": True}],
                        "hunks": [{"path": "g.go", "addr": "1", "op": "replace", "status": "ok"},
                                  {"path": "g.go", "addr": "2", "op": "replace", "status": "ok"}]}, indent=2)
    refused = json.dumps({"dry_run": False, "applied": False, "failed": 1, "files": [{"path": "f.go", "written": False}],
                          "hunks": [{"path": "f.go", "addr": "1", "op": "replace", "status": "fail",
                                     "reason": "f.go changed since mrw last saw it (recorded a, now b)"}]}, indent=2)
    miss = json.dumps({"applied": False, "failed": 1, "files": [], "hunks": [
        {"path": "a.go", "addr": "1-3", "op": "replace", "status": "fail",
         "reason": 'anchor "func A" not in line 1: "x"; the anchor is in line 2'}]}, indent=2)
    e = [use("m1", "t1", "Edit", {"file_path": "/r/f.go"})]                  # before the JSON read: no cause
    e.append(use("m1b", "t1b", "Edit", {"file_path": "/r/g.go"}))           # before the JSON write of g.go: no cause
    e += bash("m2", "t2", "mrw read --json f.go:1", read_doc)
    e += bash("m3", "t3", "mrw write --json p.txt", "mrw: a note on stderr\n" + refused)   # nothing_found
    e += bash("m4", "t4", "mrw write --json p.txt", wrote)                   # two hunks; wrote g.go
    e += bash("m5", "t5", "mrw write p.txt", refusal("g.go"))                # nothing_found: the JSON write refreshed it
    e += bash("m6", "t6", "mrw write --json p.txt", miss)
    e += bash("m7", "t7", "mrw read h.go:1", served("h.go"))
    e.append(use("m8", "t8", "Edit", {"file_path": "/r/h.go"}))
    e += bash("m9", "t9", "mrw write --quiet - <<'EOF'\n@@ h.go 1 replace\nx\nEOF", OK1)   # a quiet write of h.go
    e += bash("m10", "t10", "mrw write p.txt", refusal("h.go"))              # nothing_found: the Edit came before it
    r = scan_of(e)
    check("json causes", (r["E4"]["changed_since_refusals"], r["E4"]["own_edit"], r["E4"]["nothing_found"]), (3, 0, 3))
    check("a json miss is a miss", r["E2"]["anchor_miss_lines"], 1)
    both = "mrw: note\n" + read_doc + "\n" + OK1 + refusal("k.go")
    r2 = scan_of(bash("m1", "t1", "mrw read --json f.go:1 && mrw write p.txt", both))
    check("what follows a json document is kept", (r2["E1"]["sites_per_shell_call_running_mrw"]["mean"],
                                                   r2["E4"]["changed_since_refusals"]), (2.0, 1))
    sites = r["E1"]["sites_per_shell_call_running_mrw"]
    check("json sites", (sites["n"], sites["mean"]), (8, round((1 + 1 + 2 + 0 + 1 + 1 + 1 + 0) / 8, 2)))


def test_window():
    """--since/--until: by the entry's time; undated entries; a transcript with nothing in the window."""
    undated = {"type": "assistant", "message": {"id": "u0", "usage": U, "content": []}}
    naive = {"type": "assistant", "timestamp": "2026-11-02T00:00:00", "message": {"id": "u1", "usage": U, "content": []}}
    dated = {"type": "assistant", "timestamp": "2026-11-03T00:00:00Z", "message": {"id": "u2", "usage": U, "content": []}}
    e = [undated, naive, dated]
    check("no bound counts all", scan_of(e)["corpus"]["requests"], 3)
    r = scan_of(e, since=scan.parse_time("2026-11-01"))
    check("an undated entry is left out under a bound", r["corpus"]["requests"], 2)
    r = scan_of([dated, undated], until=scan.parse_time("2026-12-01"))
    check("an undated entry after a dated one is left out too", r["corpus"]["requests"], 1)
    text_only = {"type": "user", "timestamp": "2026-11-03T00:00:00Z", "message": {"content": "hello"}}
    check("a transcript of one plain entry in the window is counted",
          scan_of([text_only], since=scan.parse_time("2026-11-01"))["corpus"]["transcripts"], 1)
    r = scan_of(e, until=scan.parse_time("2026-11-02T12:00:00Z"))
    check("until is exclusive and a zone-less time is UTC", r["corpus"]["requests"], 1)
    r = scan_of(e, since=scan.parse_time("2027-01-01"))
    check("a transcript with nothing in the window is not counted", (r["corpus"]["transcripts"], r["corpus"]["requests"]), (0, 0))
    check("a skipped folder is not read", scan_of(e, skip=["p" + os.sep])["corpus"]["transcripts"], 0)


def plan_call(tid, plan, out=OK1, cmd="mrw write --terse - <<'EOF'\n%s\nEOF"):
    """A shell call that pipes a typed plan to mrw write, and its answer."""
    return bash("m" + tid, tid, cmd % plan, out)


def shown(tid, path, lines, cmd=None):
    """An mrw read of path that printed the given line numbers."""
    body = "".join(" %d| x\n" % n for n in lines)
    return bash("m" + tid, tid, cmd or "mrw read %s" % path, "==> %s  9L  9B  sha aaaaaaaa\n@@ %d-%d\n%s" % (
        path, lines[0], lines[-1], body))


def e5(entries):
    """Scan entries and return E5's counts that a case compares."""
    r = scan_of(entries)["E5"]
    return (r["landed_plan_calls"], r["refused_calls_a"], r["refused_calls_b"], r["uncertain_calls_a"])


def test_guard():
    """E5: the reproduced failure is refused at both widths; what shows, anchors or names by pattern is not."""
    start = shown("r0", "f.txt", [1, 2, 3, 4, 5]) + plan_call("w0", "@@ f.txt 1 insert-after\nNEW")
    check("a first write to a file is not refused", e5(start), (1, 0, 0, 0))
    check("a stale number after an insert is refused at both widths",
          e5(start + plan_call("w1", "@@ f.txt 4 replace\nD")), (2, 1, 1, 0))
    check("a read of the line between is enough",
          e5(start + shown("r1", "f.txt", [4]) + plan_call("w1", "@@ f.txt 4 replace\nD")), (2, 0, 0, 0))
    check("a read of another line is not",
          e5(start + shown("r1", "f.txt", [5]) + plan_call("w1", "@@ f.txt 4 replace\nD")), (2, 1, 1, 0))
    check("an anchor answers it",
          e5(start + plan_call("w1", "@@ f.txt 4 replace anchor=\"c\"\nD")), (2, 0, 0, 0))
    check("a pattern, the last line and line 0 are not numbers",
          e5(start + plan_call("w1", "@@ f.txt /^c$/ replace\nD\n@@ f.txt $ replace\nE\n@@ f.txt 0 insert-after\nT")),
          (2, 0, 0, 0))
    check("another file is not tracked",
          e5(start + plan_call("w1", "@@ g.txt 4 replace\nD")), (2, 0, 0, 0))
    r = scan_of(start + plan_call("w1", "@@ f.txt 4 delete\n@@ f.txt 2 insert-before\nQ\n@@ f.txt 5,+1 replace\nZ"))["E5"]
    check("each op is counted, a call once", (r["refused_calls_a"], r["refused_hunks_by_op"], r["ceiling_calls"]),
          (1, {"replace": 1, "delete": 1, "insert-after": 0, "insert-before": 1}, 1))


def test_guard_widths():
    """E5: width B keeps a line shown before a write that left it in place; width A does not."""
    read = shown("r0", "f.txt", [1, 2, 3, 4, 5])
    same = read + plan_call("w0", "@@ f.txt 4 replace\nD")
    check("above an equal replace: A refuses, B does not",
          e5(same + plan_call("w1", "@@ f.txt 2 replace\nB")), (2, 1, 0, 0))
    check("below an equal replace: the same",
          e5(same + plan_call("w1", "@@ f.txt 5 replace\nE")), (2, 1, 0, 0))
    check("the replaced line itself was not shown",
          e5(same + plan_call("w1", "@@ f.txt 4 replace\nD2")), (2, 1, 1, 0))
    grew = read + plan_call("w0", "@@ f.txt 3 replace\nC\nC2")
    check("above a replace that grew: B keeps it",
          e5(grew + plan_call("w1", "@@ f.txt 2 replace\nB")), (2, 1, 0, 0))
    check("below a replace that grew: B refuses",
          e5(grew + plan_call("w1", "@@ f.txt 5 replace\nE")), (2, 1, 1, 0))
    after = read + plan_call("w0", "@@ f.txt 3 insert-after\nN")
    check("the line an insert-after names stays; the next moves",
          (e5(after + plan_call("w1", "@@ f.txt 3 replace\nC")), e5(after + plan_call("w2", "@@ f.txt 4 replace\nD"))),
          ((2, 1, 0, 0), (2, 1, 1, 0)))
    before = read + plan_call("w0", "@@ f.txt 3 insert-before\nN")
    check("an insert-before moves the line it names",
          (e5(before + plan_call("w1", "@@ f.txt 2 replace\nB")), e5(before + plan_call("w2", "@@ f.txt 3 replace\nC"))),
          ((2, 1, 0, 0), (2, 1, 1, 0)))
    check("a delete moves from its first line",
          e5(read + plan_call("w0", "@@ f.txt 3 delete") + plan_call("w1", "@@ f.txt 3 replace\nD")), (2, 1, 1, 0))
    check("a delete with its expected body still moves the lines below",
          e5(read + plan_call("w0", "@@ f.txt 3 delete\nc") + plan_call("w1", "@@ f.txt 4 replace\nD")), (2, 1, 1, 0))
    check("a quoted word after the heredoc is not a body line",
          e5(read + plan_call("w0", "@@ f.txt 4 replace\nD", OK1, "cd \"d\" && mrw write - <<'EOF'\n%s\nEOF\necho \"exit $?\"")
             + plan_call("w1", "@@ f.txt 5 replace\nE")), (2, 1, 0, 0))
    check("two heredocs are two plans",
          e5(read + plan_call("w0", "@@ f.txt 4 replace\nD", OK1 + OK1,
                              "mrw write - <<'EOF'\n%s\nEOF\necho x\nmrw write - <<'EOF'\npreamble\n@@ g.txt 0 insert-after\nG\nEOF")
             + plan_call("w1", "@@ f.txt 5 replace\nE")), (2, 1, 0, 0))
    quoted = "printf '%%s\\n' '@@ f.txt 4 replace%s' 'D' | mrw write - && echo 'done'"
    check("the last hunk typed as quoted words has no known body",
          e5(read + plan_call("w0", "", OK1, quoted) + plan_call("w1", "@@ f.txt 2 replace\nB")), (2, 1, 1, 0))
    check("unless it declares one",
          e5(read + plan_call("w0", " body=1", OK1, quoted) + plan_call("w1", "@@ f.txt 2 replace\nB")), (2, 1, 0, 0))
    check("a declared body count is the body",
          e5(read + plan_call("w0", "@@ f.txt 3 replace body=1\nC") + plan_call("w1", "@@ f.txt 5 replace\nE")), (2, 1, 0, 0))
    check("a hunk by pattern moves every line",
          e5(read + plan_call("w0", "@@ f.txt /^c$/ replace\nC") + plan_call("w1", "@@ f.txt 1 replace\nA")), (2, 1, 1, 0))
    check("a body the plan does not hold moves every line",
          e5(read + plan_call("w0", "@@ f.txt 5 insert-after from=g.txt:1-2") + plan_call("w1", "@@ f.txt 1 replace\nA")),
          (2, 1, 1, 0))


def test_guard_state():
    """E5: what counts as landed, shown, uncertain or no longer tracked."""
    start = shown("r0", "f.txt", [1, 2, 3, 4, 5]) + plan_call("w0", "@@ f.txt 1 insert-after\nNEW")
    stale = "@@ f.txt 4 replace\nD"
    check("a dry run and a refused write are not landed",
          e5(start + plan_call("w1", stale, "1 hunk(s), 1 file(s), 0 failed, 0 advisories — dry run, nothing written\n")
             + plan_call("w2", stale, "1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN\n")), (1, 0, 0, 0))
    piped = shown("r1", "f.txt", [1], "mrw read f.txt | head -1")
    check("a piped read makes the file uncertain", e5(start + piped + plan_call("w1", stale)), (2, 0, 0, 1))
    check("what a piped read printed is shown", e5(start + piped + plan_call("w1", "@@ f.txt 1 replace\nA")), (2, 0, 0, 0))
    check("a captured read is piped", e5(start + shown("r1", "f.txt", [1], 'out="$(mrw read f.txt)"')
                                         + plan_call("w1", stale)), (2, 0, 0, 1))
    check("stderr sent away is not a pipe", e5(start + shown("r1", "f.txt", [5], "mrw read f.txt:5 2>/dev/null")
                                               + plan_call("w1", stale)), (2, 1, 1, 0))
    check("a range is refused when any line of it is unshown",
          e5(start + shown("r1", "f.txt", [4]) + plan_call("w1", "@@ f.txt 4-5 delete")), (2, 1, 1, 0))
    check("a second write forgets what was shown after the first",
          e5(start + shown("r1", "f.txt", [4]) + plan_call("w1", "@@ f.txt /^e$/ replace\nE") + plan_call("w2", stale)),
          (3, 1, 1, 0))
    check("a call with a certain refusal is not also uncertain",
          e5(start + plan_call("wg", "@@ g.txt 0 insert-after\nG") + shown("r1", "g.txt", [1], "mrw read g.txt | head -1")
             + plan_call("w1", stale + "\n@@ g.txt 2 replace\nB")), (3, 1, 1, 0))
    check("a piped read of another file leaves this one certain",
          e5(start + shown("r1", "g.txt", [1], "mrw read g.txt | head -1") + plan_call("w1", stale)), (2, 1, 1, 0))
    check("the next write ends the doubt",
          e5(start + piped + plan_call("w1", "@@ f.txt /^c$/ replace\nC") + plan_call("w2", stale)), (3, 1, 1, 0))
    check("the host's edit of the file ends the tracking",
          e5(start + [use("mE", "tE", "Edit", {"file_path": "/r/f.txt"})] + plan_call("w1", stale)), (2, 0, 0, 0))
    check("a read before the write in one command counts",
          e5(start + plan_call("w1", stale, "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 4-4\n 4| c\n" + OK1,
                               "mrw read f.txt:4 && mrw write - <<'EOF'\n%s\nEOF")), (2, 0, 0, 0))
    check("a read after the write in one command shows the new lines",
          e5(start + plan_call("w1", "@@ f.txt /^c$/ replace\nC", OK1 + "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 4-4\n 4| C\n",
                               "mrw write - <<'EOF'\n%s\nEOF\nmrw read f.txt:4") + plan_call("w2", stale)), (3, 0, 0, 0))
    check("a plan typed as quoted words is read",
          e5(start + plan_call("w1", "", OK1, "printf '%%s\\n' '@@ f.txt 4 replace' 'D' | mrw write -%s")), (2, 1, 1, 0))
    made = bash("mc", "tc", "printf 'a\\nb\\n' | mrw write --create n.txt", "created n.txt  0L -> 2L  sha aaaaaaaa\n" + OK1)
    r = scan_of(made + plan_call("w1", "@@ n.txt 2 replace\nB"))["E5"]
    check("a file the session created is counted apart",
          (r["landed_plan_calls"], r["refused_calls_a"], r["refused_hunks_on_created"], r["landed_calls_with_no_typed_plan"]),
          (1, 1, 1, 1))
    filed = bash("mp", "tp", "mrw write --terse plan.txt", "ok   f.txt  1 hunk(s)  -0 +1  sha aaaaaaaa\n" + OK1)
    check("a plan in a file still marks what it wrote",
          e5(shown("r0", "f.txt", [1, 2, 3]) + filed + plan_call("w1", "@@ f.txt 2 replace\nB")), (1, 1, 1, 0))
    two = plan_call("w1", "@@ g.txt 1 insert-after\nN", OK1 + OK1,
                    "mrw write - <<'EOF'\n%s\nEOF\nmrw write - <<'EOF'\n@@ g.txt 3 replace\nC\nEOF")
    check("two writes in one command are counted as chained", scan_of(two)["E5"]["chained_write_commands"], 1)
    r = scan_of(start + piped + plan_call("w1", stale))["E5"]
    check("an uncertain call is counted at each width", (r["uncertain_calls_a"], r["uncertain_calls_b"]), (1, 1))
    r = scan_of(shown("r0", "f.txt", [1, 2, 3, 4, 5]) + plan_call("w0", "@@ f.txt 4 replace\nD") + piped
                + plan_call("w1", "@@ f.txt 2 replace\nB"))["E5"]
    check("and the widths can differ: a line the write left in place is shown at width B",
          (r["uncertain_calls_a"], r["uncertain_calls_b"], r["refused_calls_a"], r["refused_calls_b"]), (1, 0, 0, 0))
    for other in ("mrw read 'g.txt:1' | head -1", "mrw read g.txt | sed -n '1p'"):
        check("a piped read's doubt stays with the files it names: " + other,
              e5(start + bash("mr", "tr", "mrw read f.txt:1; " + other, "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 1-1\n 1| a\n")
                 + plan_call("w1", stale)), (2, 1, 1, 0))
    check("with reads on both sides, the plan in that command is judged uncertain, not refused",
          e5(start + plan_call("w1", stale, "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 4-4\n 4| c\n" + OK1,
                               "mrw read f.txt:4 && mrw write - <<'EOF'\n%s\nEOF\nmrw read g.txt:1")), (2, 0, 0, 1))
    check("a command-shaped line in a plan body is not a command",
          e5(start + plan_call("w1", "@@ f.txt 4 replace\nmrw read f.txt", "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 4-4\n 4| c\n" + OK1,
                               "mrw read f.txt:4 && mrw write - <<'PLAN-1'\n%s\nPLAN-1")), (2, 0, 0, 0))
    check("a key inside another option's value is not that option",
          ([h[4] for h in scan.typed_hunks("@@ h.txt 3 replace anchor='literal body=2 here'\nC\n@@ h.txt 5 replace\nE")],
           e5(start + plan_call("w1", "@@ f.txt 4 replace body=@\"anchor=x\"")),
           [h[4] for h in scan.typed_hunks("@@ h.txt 3 replace body=\"1\"\nC")]),
          ([1, 1], (2, 1, 1, 0), [1]))
    check("a piped read of one file leaves another the command read plainly certain",
          e5(start + bash("mr", "tr", "mrw read f.txt:1; mrw read g.txt | head -1", "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 1-1\n 1| a\n")
             + plan_call("w1", stale)), (2, 1, 1, 0))
    both = bash("mb", "tb", "mrw read f.txt:4 && mrw write - <<'EOF'\n@@ f.txt /^e$/ replace\nE\nEOF\nmrw read f.txt:1",
                "==> f.txt  6L  9B  sha aaaaaaaa\n@@ 4-4\n 4| c\n" + OK1 + "==> f.txt  6L  9B  sha bbbbbbbb\n@@ 1-1\n 1| a\n")
    check("reads on both sides of a write show nothing and leave the file uncertain",
          e5(start + both + plan_call("w2", stale)), (3, 0, 0, 1))
    read5 = shown("r9", "h.txt", [1, 2, 3, 4, 5])
    check("the host's edit forgets what was shown before any write, at width B too",
          e5(read5 + [use("mE", "tE", "Edit", {"file_path": "/r/h.txt"})] + plan_call("w0", "@@ h.txt 5 replace\nE")
             + plan_call("w1", "@@ h.txt 2 replace\nB")), (2, 1, 1, 0))
    check("a replace to the last line moves everything from its first, and nothing above it",
          (e5(read5 + plan_call("w0", "@@ h.txt 3-$ replace anchor=\"c\"\nX") + plan_call("w1", "@@ h.txt 4 replace\nD")),
           e5(read5 + plan_call("w0", "@@ h.txt 3-$ replace anchor=\"c\"\nX") + plan_call("w1", "@@ h.txt 2 replace\nB"))),
          ((2, 1, 1, 0), (2, 1, 0, 0)))
    check("unless its body is unknown, which moves every line",
          e5(read5 + plan_call("w0", "@@ h.txt 3-$ replace anchor=\"c\" from=g.txt:1") + plan_call("w1", "@@ h.txt 2 replace\nB")),
          (2, 1, 1, 0))
    check("a counted body is not read for headers",
          e5(read5 + plan_call("w0", "@@ h.txt 3 replace body=2 raw=true\n@@ h.txt 1 delete\nX") + plan_call("w1", "@@ h.txt 2 replace\nB")),
          (2, 1, 0, 0))
    check("a delimiter with a dash, and tabs a dashed heredoc strips",
          (e5(start + plan_call("w1", stale, OK1, "mrw write - <<'PLAN-1'\n%s\nPLAN-1")),
           e5(start + plan_call("w1", "\t@@ f.txt 4 replace\n\tD", OK1, "mrw write - <<-EOF\n%s\n\tEOF"))),
          ((2, 1, 1, 0), (2, 1, 1, 0)))
    check("a second write in a command is counted as chained whether or not it landed",
          scan_of(plan_call("w1", "@@ g.txt 1 replace\nN", "nothing\n",
                            "mrw write - <<'EOF'\n%s\nEOF\nmrw write - <<'EOF'\n@@ g.txt 3 replace\nC\nEOF"))["E5"]["chained_write_commands"], 1)
    check("the verdict's rules", [scan.guard_verdict(*a) for a in (
        (199, 0, 0, 0, 0), (200, 10, 10, 99, 99), (200, 11, 0, 10, 10), (200, 10, 11, 10, 11), (200, 11, 0, 11, 0),
        (200, 10, 11, 10, 10))],
        ["INSUFFICIENT", "BUILD A", "BUILD B", "STOP", "STOP", "BUILD B"])


def e6(entries):
    """Scan entries and return E6's counts that a case compares."""
    r = scan_of(entries)["E6"]
    return (r["landed_write_calls"], r["path_reverts"], r["near_path_reverts"], r["tree_reverts"])


def sh(tid, cmd):
    """A shell call that runs no mrw, and its empty answer."""
    return bash("m" + tid, tid, cmd, "")


def test_reverts():
    """E6: a shell revert of a file mrw just wrote is counted, near or late; a tree revert apart."""
    wrote = plan_call("w0", "@@ f.txt 1 replace\nA")
    check("a revert of the file just written is a near path revert",
          e6(wrote + sh("g1", "git checkout -- f.txt")), (1, 1, 1, 0))
    check("git restore counts, with ./ and an absolute path",
          (e6(wrote + sh("g1", "git restore ./f.txt")), e6(wrote + sh("g2", "git checkout HEAD -- /r/f.txt"))),
          ((1, 1, 1, 0), (1, 1, 1, 0)))
    check("a revert of a file mrw did not write is nothing",
          e6(wrote + sh("g1", "git checkout -- g.txt")), (1, 0, 0, 0))
    late = wrote + [x for i in range(6) for x in sh("n%d" % i, "ls")] + sh("g1", "git checkout -- f.txt")
    check("a revert more than five calls later is a path revert, not near", e6(late), (1, 1, 0, 0))
    edge = wrote + [x for i in range(4) for x in sh("n%d" % i, "ls")] + sh("g1", "git checkout -- f.txt")
    check("the fifth call after the write is still near", e6(edge), (1, 1, 1, 0))
    for cmd in ("git checkout -b fix f.txt", "git restore --staged f.txt", "git restore -S f.txt",
                "git restore --staged --worktree f.txt", "git checkout-index f.txt", "git checkout main",
                "echo 'git checkout -- f.txt'", "echo \"x; git checkout -- f.txt\"",
                "cat <<'EOF'\ngit checkout -- f.txt\nEOF", "git stash pop", "git stash list", "git reset HEAD f.txt"):
        check("not a revert: " + cmd, e6(wrote + sh("g1", cmd)), (1, 0, 0, 0))
    for cmd in ("git stash", "git stash push -m x", "git reset --hard", "git checkout .", "git checkout -- .",
                "git restore ."):
        check("a tree revert: " + cmd, e6(wrote + sh("g1", cmd)), (1, 0, 0, 1))
    check("a tree revert before any write is nothing", e6(sh("g1", "git stash") + wrote), (1, 0, 0, 0))
    check("a command is counted once, and a reverted file is no longer written",
          e6(wrote + sh("g1", "git checkout -- f.txt && git checkout -- f.txt") + sh("g2", "git checkout -- f.txt")),
          (1, 1, 1, 0))
    check("a later git command in the chain is read, and git -C is git",
          (e6(wrote + sh("g1", "git checkout main && git checkout -- f.txt")),
           e6(wrote + sh("g2", "git -C /r checkout -- f.txt"))), ((1, 1, 1, 0), (1, 1, 1, 0)))
    two = plan_call("w0", "@@ f.txt 1 replace\nA\n@@ g.txt 1 replace\nB", "2 hunk(s), 2 file(s), 0 failed, 0 advisories — applied\n")
    check("two written files reverted in one command are one revert, and both are no longer written",
          e6(two + sh("g1", "git checkout -- f.txt g.txt") + sh("g2", "git checkout -- g.txt")), (1, 1, 1, 0))
    check("a path revert and a tree revert in one command are counted apart",
          e6(wrote + sh("g1", "git checkout -- f.txt; git stash")), (1, 1, 1, 1))
    slow = ([use("mw", "tw", "Bash", {"command": "mrw write --terse - <<'EOF'\n@@ f.txt 1 replace\nA\nEOF"})]
            + [use("mr%d" % i, "tr%d" % i, "Read", {"file_path": "/r/x"}) for i in range(5)] + [res("tw", OK1)])
    check("the window counts from the write's own call, not from when its answer arrived",
          (e6(slow + sh("g1", "git checkout -- f.txt")), e6(slow[:5] + [slow[-1]] + sh("g2", "git checkout -- f.txt"))),
          ((1, 1, 0, 0), (1, 1, 1, 0)))
    refused = plan_call("w0", "@@ f.txt 1 replace\nA", "1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN\n")
    check("a refused write leaves nothing to revert", e6(refused + sh("g1", "git checkout -- f.txt")), (0, 0, 0, 0))
    filed = bash("mp", "tp", "mrw write --terse plan.txt", "ok   f.txt  1 hunk(s)  -0 +1  sha aaaaaaaa\n" + OK1)
    check("a write with no typed plan is a landing, and its file can be reverted",
          e6(filed + sh("g1", "git restore f.txt")), (1, 1, 1, 0))
    check("the verdict's rules", [scan.revert_verdict(*a) for a in (
        (999, 500), (1000, 25), (1000, 24), (6000, 29), (6000, 30), (5000, 25))],
        ["INSUFFICIENT", "RECORD", "STOP", "STOP", "RECORD", "RECORD"])


def main():
    """Run every case and print the mismatches."""
    for case in (test_cost, test_sites, test_shell, test_plan, test_plan_edges, test_tags, test_compaction,
                 test_changed, test_changed_edges, test_json_and_quiet, test_window, test_guard, test_guard_widths,
                 test_guard_state, test_reverts):
        case()
    for b in BAD:
        print("FAIL " + b)
    print("%d check(s), %d failed" % (CHECKS[0], len(BAD)))
    return 1 if BAD else 0


if __name__ == "__main__":
    sys.exit(main())
