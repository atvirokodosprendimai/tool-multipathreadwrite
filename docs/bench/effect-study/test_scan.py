"""Self-test for scan.py on synthetic transcripts whose answers are known.

Run: python3 -I docs/bench/effect-study/test_scan.py   (exit 0 green, 1 red)
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


def stamp():
    """Return the next timestamp, one second after the last."""
    N[0] += 1
    return "2026-11-01T00:%02d:%02dZ" % (N[0] // 60, N[0] % 60)


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


def session():
    """Build one transcript: every number the asserts below name is planted here."""
    u = {"input_tokens": 100, "output_tokens": 200, "cache_read_input_tokens": 10000,
         "cache_creation_input_tokens": 400, "cache_creation": {"ephemeral_1h_input_tokens": 300}}
    e = []
    # m1: one mrw read of three ranges. The usage repeats on a second line of the same message.
    e += bash("m1", "t1", "mrw read a.go:1-3 b.go:5 b.go:9", "==> a.go  9L\n@@ 1-3\n 1| x\n==> b.go  9L\n@@ 5-5\n@@ 9-9\n", u)
    e.append({"type": "assistant", "timestamp": stamp(), "message": {"id": "m1", "usage": u, "content": []}})
    # m2: a plan of three hunks: one anchored, one bare single-line replace, one insert.
    plan = ('mrw write - <<\'EOF\'\n@@ a.go 1-3 replace anchor="func A"\nx\n@@ b.go 5 replace\ny\n'
            '@@ b.go 9 insert-after\nz\nEOF')
    e += bash("m2", "t2", plan, "ok a.go\n3 hunk(s), 2 file(s), 0 failed, 0 advisories — applied\n", u)
    # m3: a request of two host reads, no mrw.
    e.append(use("m3", "t3", "Read", {"file_path": "/r/c.go"}, u))
    e.append(use("m3", "t4", "Read", {"file_path": "/r/d.go"}))
    # five mrw writes, a compaction, then five host edits. Before: 0 of 5. After, within the window:
    # these five, the Edit of f.go and four mrw writes further down, so 6 of 10.
    for i in range(5):
        e += bash("w%d" % i, "tw%d" % i, "mrw write --terse p%d.txt" % i,
                  "1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied\n")
    e.append({"type": "system", "subtype": "compact_boundary", "timestamp": stamp()})
    for i in range(5):
        e.append(use("n%d" % i, "tn%d" % i, "Edit", {"file_path": "/r/e%d.go" % i}))
    # an anchor miss.
    e += bash("m4", "t5", "mrw write p.txt", 'FAIL a.go 1-3 replace: anchor "func A" not in line 1: "x"\n')
    # three changed-since refusals: after the session's own Edit, after its own gofmt, and after nothing.
    e += bash("m5", "t6", "mrw read f.go:1", "==> f.go  3L\n@@ 1-1\n 1| x\n")
    e.append(use("m6", "t7", "Edit", {"file_path": "/r/f.go"}))
    e += bash("m7", "t8", "mrw write p.txt", "FAIL f.go 1 replace: f.go changed since mrw last saw it (recorded a, now b)\n")
    e += bash("m8", "t9", "mrw read g.go:1", "==> g.go  3L\n@@ 1-1\n 1| x\n")
    e += bash("m9", "t10", "gofmt -w .", "")
    e += bash("m10", "t11", "mrw write p.txt", "FAIL g.go 1 replace: g.go changed since mrw last saw it (recorded a, now b)\n")
    e += bash("m11", "t12", "mrw read h.go:1", "==> h.go  3L\n@@ 1-1\n 1| x\n")
    e += bash("m12", "t13", "mrw write p.txt", "FAIL h.go 1 replace: h.go changed since mrw last saw it (recorded a, now b)\n")
    return e


def check(name, got, want, bad):
    """Compare one number and remember a mismatch."""
    if got != want:
        bad.append("%s = %r, want %r" % (name, got, want))


def main():
    """Scan the synthetic corpus and compare every planted number."""
    bad = []
    with tempfile.TemporaryDirectory() as d:
        os.mkdir(os.path.join(d, "p"))
        with open(os.path.join(d, "p", "s.jsonl"), "w", encoding="utf-8") as fh:
            for o in session():
                fh.write(json.dumps(o) + "\n")
            fh.write("not json\n")
        repo = os.path.join(d, "repo")
        os.mkdir(repo)
        with open(os.path.join(repo, "t.txt"), "w", encoding="utf-8") as fh:
            fh.write("a\na\nb\nc")                      # pairs: a/a identical, a/b, b/c
        git = ["git", "-C", repo, "-c", "user.name=t", "-c", "user.email=t@example.invalid"]
        subprocess.run(git + ["init", "-q"], check=True)
        subprocess.run(git + ["add", "t.txt"], check=True)
        r = scan.run(d, repo=repo)
        late = scan.run(d, since=scan.parse_time("2027-01-01"))
        skipped = scan.run(d, skip=["p" + os.sep])

    check("transcripts", r["corpus"]["transcripts"], 1, bad)
    check("requests", r["corpus"]["requests"], 3, bad)          # m1, m2, m3 carry usage; m1's repeat counts once
    tok = r["E1"]["tokens"]
    check("input", tok["input"], 300, bad)
    check("output", tok["output"], 600, bad)
    check("cache_read", tok["cache_read"], 30000, bad)
    check("cache_write_1h", tok["cache_write_1h"], 900, bad)
    check("cache_write_5m", tok["cache_write_5m"], 300, bad)
    # priced: 300 + 3000 + 3000 + 1800 + 375 = 8475; cache = 5175
    check("cache_share_pct", r["E1"]["cache_share_pct"], round(100 * 5175 / 8475, 3), bad)
    check("sites_per_mrw_call", r["E1"]["sites_per_mrw_call"]["n"], 10, bad)     # 4 reads + m2 + 5 writes
    check("sites_per_mrw_call p90", r["E1"]["sites_per_mrw_call"]["p90"], 3, bad)
    check("native-only requests", r["E1"]["sites_per_request_native_only"], {
        "n": 7, "mean": round(8 / 7, 2), "median": 1, "p90": 1, "share_of_one": round(100 * 6 / 7, 3)}, bad)
    check("hunks_typed", r["E2"]["hunks_typed"], 3, bad)
    check("anchored_hunks", r["E2"]["anchored_hunks"], 1, bad)
    check("bare_single_line_replaces", r["E2"]["bare_single_line_replaces"], 1, bad)
    check("anchor_chars", r["E2"]["anchor_chars"], len("func A"), bad)
    check("anchor_misses", r["E2"]["anchor_misses"], 1, bad)
    tc = r["E2"]["tag_collisions"]
    check("pairs", tc["pairs"], 3, bad)
    check("identical_pairs", tc["identical_pairs"], 1, bad)
    check("compactions_paired", r["E3"]["compactions_paired"], 1, bad)
    check("native_share_before_pct", r["E3"]["native_share_before_pct"], 0.0, bad)
    check("native_share_after_pct", r["E3"]["native_share_after_pct"], 60.0, bad)
    check("mean_paired_difference_pp", r["E3"]["mean_paired_difference_pp"], 60.0, bad)
    check("mrw_write_calls", r["corpus"]["mrw_write_calls"], 10, bad)             # m2, five, m4, m7, m10, m12
    check("changed_since_refusals", r["E4"]["changed_since_refusals"], 3, bad)
    check("own_edit", r["E4"]["own_edit"], 1, bad)
    check("own_shell", r["E4"]["own_shell"], 1, bad)
    check("nothing_found", r["E4"]["nothing_found"], 1, bad)
    check("a window after the corpus is empty", late["corpus"]["requests"], 0, bad)
    check("a skipped folder is not read", skipped["corpus"]["transcripts"], 0, bad)
    for b in bad:
        print("FAIL " + b)
    print("%d check(s), %d failed" % (30, len(bad)))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
