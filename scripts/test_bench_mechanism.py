"""Tests for bench_mechanism.py: each frozen rule of HOOK-PREREGISTRATION.md, on synthetic transcripts."""
import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import bench_mechanism as mech  # noqa: E402

CWD = "/private/var/t/tree"


def transcript(*steps):
    """A stream-json transcript: steps are (tool, input, result text)."""
    rows = [{"type": "system", "subtype": "init", "cwd": CWD}]
    for i, (name, given, result) in enumerate(steps):
        rows.append({"type": "assistant", "message": {"content": [{"type": "tool_use", "id": str(i), "name": name, "input": given}]}})
        rows.append({"type": "user", "message": {"content": [{"type": "tool_result", "tool_use_id": str(i), "content": [{"type": "text", "text": result}]}]}})
    return "\n".join(json.dumps(r) for r in rows)


def host_read(path):
    return ("Read", {"file_path": "/var/t/tree/" + path}, "1\tx")


def bash(command, result="ok"):
    return ("Bash", {"command": command}, result)


APPLIED = "ok   a.go  1 hunk(s)\n1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied"
REFUSED = "FAIL a.go 3 replace: 3 of a.go has not been read\n1 hunk(s), 1 file(s), 1 failed — NOTHING WRITTEN"
PLAN = "mrw write - <<'EOF'\n@@ a.go 3 replace\nx\nEOF"


class Counts(unittest.TestCase):
    def test_an_mrw_read_after_a_host_read_overlaps(self):
        c = mech.classify(transcript(host_read("a.go"), bash("mrw read a.go:1-9 'a.go:/x/' b.go")))
        self.assertEqual((c["host_reads"], c["overlapping_reads"]), (1, 1))

    def test_an_mrw_read_with_no_host_read_before_it_does_not(self):
        c = mech.classify(transcript(bash("mrw read a.go"), host_read("a.go")))
        self.assertEqual(c["overlapping_reads"], 0)

    def test_a_landed_write_ends_the_overlap_and_a_refused_one_does_not(self):
        c = mech.classify(transcript(host_read("a.go"), bash(PLAN, APPLIED), bash("mrw read a.go")))
        self.assertEqual(c["overlapping_reads"], 0)
        c = mech.classify(transcript(host_read("a.go"), bash(PLAN, REFUSED), bash("mrw read a.go")))
        self.assertEqual(c["overlapping_reads"], 1)

    def test_a_write_after_only_a_host_read_is_hook_licensed(self):
        c = mech.classify(transcript(host_read("a.go"), bash(PLAN, APPLIED)))
        self.assertEqual(c["hook_licensed_writes"], 1)
        c = mech.classify(transcript(host_read("a.go"), bash("mrw read a.go"), bash(PLAN, APPLIED)))
        self.assertEqual(c["hook_licensed_writes"], 0)

    def test_a_write_refused_unread_after_a_host_read_is_counted(self):
        c = mech.classify(transcript(host_read("a.go"), bash(PLAN, REFUSED)))
        self.assertEqual((c["refused_unread_writes"], c["hook_licensed_writes"]), (1, 0))

    def test_a_call_whose_files_cannot_be_named_is_unknown_and_in_no_count(self):
        for command in ("mrw read --grep x .", "mrw read $F", "mrw --root d read a.go", "mrw write plan.txt",
                        "mrw write --grep a --sub b .", "cat l | mrw read --files-from -"):
            c = mech.classify(transcript(host_read("a.go"), bash(command, APPLIED)))
            self.assertEqual((c["unknown"], c["overlapping_reads"], c["hook_licensed_writes"]), (1, 0, 0), command)

    def test_a_dollar_address_is_a_file_not_a_variable(self):
        c = mech.classify(transcript(host_read("a.go"), bash("mrw read 'a.go:$' 'a.go:/x$/'")))
        self.assertEqual((c["unknown"], c["overlapping_reads"]), (0, 1))

    def test_one_command_naming_a_file_twice_counts_once_and_other_tools_are_not_mrw(self):
        c = mech.classify(transcript(host_read("a.go"), bash("mrw read a.go:1 a.go:5 && grep x a.go | head")))
        self.assertEqual(c["overlapping_reads"], 1)

    def test_a_chained_read_then_write_in_one_command(self):
        c = mech.classify(transcript(host_read("a.go"), bash("mrw read a.go:3 && " + PLAN, APPLIED)))
        self.assertEqual((c["overlapping_reads"], c["hook_licensed_writes"]), (1, 0))

    def test_a_create_names_its_file(self):
        c = mech.classify(transcript(host_read("a.go"), bash("mrw write --create a.go <<'EOF'\nx\nEOF", APPLIED)))
        self.assertEqual(c["hook_licensed_writes"], 1)


if __name__ == "__main__":
    unittest.main()
