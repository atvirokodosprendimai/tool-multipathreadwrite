"""Self-test for scan.py on a synthetic transcript whose answers are known.

Run: python3 -I docs/bench/copy-study/test_scan.py   (exit 0 green, 1 red)
"""
import json
import os
import sys
import tempfile

sys.dont_write_bytecode = True                   # no __pycache__ in the tree
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import scan  # noqa: E402

A = ["    value_alpha_%02d = compute_something(%d)" % (i, i) for i in range(1, 21)]
A[14:17] = ["  }", "  )", "  }"]                 # lines 15-17: trivial
A[17:19] = ["short_a()", "short_b()"]            # lines 18-19: a run under 40 non-ws chars
B = ["func line_%02d() { return %d }" % (i, i) for i in range(1, 11)]
A2 = A[:5] + ["    changed_line_six = 0"] + A[6:]  # a.py after line 6 changed
E = ["epsilon line number %d of five" % i for i in range(1, 6)]


def read_result(i, path, lines):
    return tool_result(i, "\n".join("%6d\u2192%s" % (n, t) for n, t in enumerate(lines, 1)),
                       {"file": {"filePath": path}})


def tool_use(i, name, inp):
    return {"type": "assistant", "message": {"id": "msg_%d" % i, "role": "assistant",
            "content": [{"type": "tool_use", "id": "t%d" % i, "name": name, "input": inp}],
            "usage": {"output_tokens": 1000}}}


def tool_result(i, text, extra=None):
    o = {"type": "user", "message": {"role": "user", "content": [
        {"type": "tool_result", "tool_use_id": "t%d" % i, "content": text}]}}
    if extra:
        o["toolUseResult"] = extra
    return o


def transcript():
    read_a = "\n".join("%6d→%s" % (n, t) for n, t in enumerate(A, 1))
    mrw_b = "==> b.go  10L  300B  sha 0\n@@ 1-10\n" + "\n".join("%5d| %s" % (n, t) for n, t in enumerate(B, 1))
    kept = ["keep_one = 1  # kept", "keep_two = 2  # kept", "keep_three = 3  # kept"]
    new1 = kept + A[4:10] + ["brand_new_line_one()", "brand_new_line_two()"]
    plan = "\n".join(["@@ b.go 3-5 replace anchor=\"x\"", B[2], B[3], "func changed() {}"] +
                     ["        " + t for t in A[10:13]])
    return [
        tool_use(1, "Read", {"file_path": "/x/proj/a.py"}),
        tool_result(1, read_a, {"file": {"filePath": "/x/proj/a.py"}}),
        tool_use(2, "Bash", {"command": "mrw read b.go:1-10"}),
        tool_result(2, mrw_b),
        # copy of A 5-10 inside a body with kept and new lines
        tool_use(3, "Edit", {"file_path": "/x/proj/c.py", "old_string": "\n".join(kept + ["old_gone()"]),
                             "new_string": "\n".join(new1)}),
        # trivial lines and a too-short run: never copyable
        tool_use(4, "Edit", {"file_path": "/x/proj/c.py", "old_string": "zzz",
                             "new_string": "\n".join(A[14:19])}),
        # whole body = A 1-4
        tool_use(5, "Edit", {"file_path": "/x/proj/c.py", "old_string": "placeholder",
                             "new_string": "\n".join(A[0:4])}),
        # mrw plan via heredoc: 2 kept lines, 1 new, A 11-13 re-indented (normalised copy only)
        tool_use(6, "Bash", {"command": "cat > p.plan <<'EOF'\n" + plan + "\nEOF\nmrw write p.plan"}),
        # the same tool_use repeated in the transcript: counted once
        tool_use(5, "Edit", {"file_path": "/x/proj/c.py", "old_string": "placeholder",
                             "new_string": "\n".join(A[0:4])}),
        # an mrw create of a file already read: KEPT, not a copy
        tool_use(7, "Bash", {"command": "mrw write - <<'EOF'\n@@ /x/proj/a.py 0 create\n" +
                             "\n".join(A[0:4]) + "\nEOF"}),
        # a whole-file copy whose content ends in a newline: still ONE whole-body copy
        tool_use(8, "Write", {"file_path": "/x/proj/d.py", "content": "\n".join(A[0:4]) + "\n"}),
        # a.py re-read after line 6 changed (and after t7 wrote it, so nothing is redundant)
        tool_use(9, "Read", {"file_path": "/x/proj/a.py"}),
        read_result(9, "/x/proj/a.py", A2),
        # old lines 6-8: line 6 is stale, so no run of 3 may start on it
        tool_use(10, "Edit", {"file_path": "/x/proj/f.py", "old_string": "q", "new_string": "\n".join(A[5:8])}),
        # e.txt read twice with no write between: the second read is wholly redundant
        tool_use(11, "Read", {"file_path": "/x/proj/e.txt"}),
        read_result(11, "/x/proj/e.txt", E),
        tool_use(12, "Read", {"file_path": "/x/proj/e.txt"}),
        read_result(12, "/x/proj/e.txt", E),
    ], dict(kept=kept, new1=new1, plan=plan)


def main():
    lines, k = transcript()
    with tempfile.TemporaryDirectory() as root:
        os.makedirs(os.path.join(root, "proj1"))
        with open(os.path.join(root, "proj1", "s.jsonl"), "w") as fh:
            for o in lines:
                fh.write(json.dumps(o, separators=(",", ":")) + "\n")
            fh.write(" " * 11000 + "\n")              # over the 10 KB floor
        res = scan.main(["--root", root, "--out", os.path.join(root, "r.json")])
    c = scan.chars
    al = res["all"]
    want = {
        "write units": (al["write_units"], 7),
        "kept chars": (al["kept_chars"], c(k["kept"]) + c([B[2], B[3]]) + c(A[0:4])),
        "exact k3 runs": (al["exact_k3"]["copyable_runs"], 3),
        "exact k3 chars": (al["exact_k3"]["copyable_chars"], c(A[4:10]) + 2 * c(A[0:4])),
        "exact k10 runs": (al["exact_k10"]["copyable_runs"], 0),
        "norm k3 runs": (al["norm_k3"]["copyable_runs"], 4),
        "whole-body units": (al["exact_k3"]["whole_body_units"], 2),
        "whole-body saving": (al["exact_k3"]["whole_body_saving_chars"], 2 * (c(A[0:4]) - scan.HEADER_COST)),
        "split saving": (al["exact_k3"]["split_hunk_saving_chars"],
                         c(A[4:10]) - 2 * scan.HEADER_COST + 2 * (c(A[0:4]) - scan.HEADER_COST)),
        "mrw_plan units": (res["by_family"]["mrw_plan"]["write_units"], 2),
        "output tokens": (res["corpus"]["output_tokens_reported"], 12000),
        "histogram": (res["run_length_histogram_exact_k3"], {"3-4": 2, "5-9": 1}),
        "served chars": (res.get("read_redundancy", {}).get("served_chars"), c(A) + c(B) + c(A2) + 2 * c(E)),
        "redundant chars": (res.get("read_redundancy", {}).get("redundant_chars"), c(E)),
    }
    bad = [f"{name}: got {got!r}, want {exp!r}" for name, (got, exp) in want.items() if got != exp]
    for b in bad:
        print("FAIL", b)
    print("PASS" if not bad else f"{len(bad)} failed")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
