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


def main():
    """Run every case and print the mismatches."""
    for case in (test_cost, test_sites, test_shell, test_plan, test_plan_edges, test_tags, test_compaction,
                 test_changed, test_changed_edges, test_json_and_quiet, test_window):
        case()
    for b in BAD:
        print("FAIL " + b)
    print("%d check(s), %d failed" % (CHECKS[0], len(BAD)))
    return 1 if BAD else 0


if __name__ == "__main__":
    sys.exit(main())
