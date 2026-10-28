"""Regression checks for the registered reference-cost counter."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "docs/bench/output-study/measure_repeat.py"


def block(rows, sha="12345678"):
    body = "".join(f"{n:5d}| {text}\n" for n, text in rows)
    return f"==> a file.go  100L  8100B  sha {sha}\n" + body


def reference(spans, sha="12345678"):
    return f"==> a file.go  100L  8100B  sha {sha}\n-- same as served: lines {spans} at sha {sha}\n"


class ReferenceCosts(unittest.TestCase):
    def measure(self, results):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "session.jsonl"
            messages = []
            for i, (stamp, text) in enumerate(results):
                for content in [
                    {"type": "tool_use", "id": str(i), "name": "Bash"},
                    {"type": "tool_result", "tool_use_id": str(i), "content": text},
                ]:
                    messages.append({"timestamp": stamp, "message": {"content": [content]}})
            path.write_text("".join(json.dumps(m) + "\n" for m in messages), encoding="utf-8")
            p = subprocess.run([sys.executable, "-I", str(SCRIPT), "--since", "2026-10-28T00:00:00Z",
                                "--until", "2026-10-29T00:00:00Z", "--glob", str(path)],
                               check=True, capture_output=True, text=True, timeout=10)
            return json.loads(p.stdout)

    def test_a_small_reference_credits_only_its_lines(self):
        rows = [(n, "x" * 80) for n in range(1, 101)]
        d = self.measure([("2026-10-28T01:00:00Z", block(rows)),
                          ("2026-10-28T01:01:00Z", reference("1-1"))])
        self.assertEqual(d["reference_stands_for_chars"], len(block(rows[:1]).split("\n", 1)[1]))

    def test_a_reference_can_use_several_reads_including_before_the_window(self):
        d = self.measure([("2026-10-27T23:59:00Z", block([(1, "one")])),
                          ("2026-10-28T00:01:00Z", block([(2, "two"), (3, "three")])),
                          ("2026-10-28T00:02:00Z", reference("1-2, 3-3"))])
        self.assertEqual(d["reference_stands_for_chars"], sum(len(f"{n:5d}| {s}\n") for n,s in [(1,"one"),(2,"two"),(3,"three")]))
        self.assertEqual(d.get("reference_unmatched_count", 0), 0)

    def test_an_unseen_line_or_different_sha_is_unmatched(self):
        for spans, sha in [("1-2", "12345678"), ("1-1", "abcdef01")]:
            with self.subTest(spans=spans, sha=sha):
                d = self.measure([("2026-10-28T00:01:00Z", block([(1,"one")])),
                                  ("2026-10-28T00:02:00Z", reference(spans, sha))])
                self.assertEqual(d.get("reference_stands_for_chars",0), 0)
                self.assertEqual(d["reference_unmatched_count"], 1)

    def test_a_column_cut_line_cannot_supply_a_reference(self):
        d = self.measure([("2026-10-28T00:01:00Z", block([(1, "x…  [cols 1-1 of 500]")])),
                          ("2026-10-28T00:02:00Z", reference("1-1"))])
        self.assertEqual(d.get("reference_stands_for_chars", 0), 0)
        self.assertEqual(d["reference_unmatched_count"], 1)

    def test_a_window_never_borrows_a_future_read(self):
        # Transcript file order does not override the observation timestamps.
        d = self.measure([("2026-10-29T01:00:00Z", block([(1,"one")])),
                          ("2026-10-28T01:00:00Z", reference("1-1"))])
        self.assertEqual(d["reference_unmatched_count"], 1)


if __name__ == "__main__":
    unittest.main()
