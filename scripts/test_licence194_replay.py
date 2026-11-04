"""Synthetic transcripts with a known W, R and U for scripts/licence194_replay.py (ADR-194 T1)."""
import importlib.util
import json
import os
import tempfile
import unittest

_spec = importlib.util.spec_from_file_location(
    "licence194_replay", os.path.join(os.path.dirname(os.path.abspath(__file__)), "licence194_replay.py"))
lr = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(lr)


def read(path, nl, a, b):
    """What `mrw read path:a-b` prints of an nl-line file: the line after the range included (ADR-161)."""
    end = min(b + 1, nl) if (a, b) != (1, nl) else nl
    body = "".join(f" {n}| line {n}\n" for n in range(a, end + 1))
    return f"==> {path}  {nl}L  99B  sha 0000000a\n@@ {a}-{end}\n{body}"


def whole(path, nl):
    return read(path, nl, 1, nl)


def write(rows, path, before, after, status="applied"):
    """A write receipt: rows like '3 replace -1 +1'."""
    out = "".join(f"ok   {path} {r.split()[0]} {r.split()[1]}  {r.split()[2]} {r.split()[3]}\n" for r in rows)
    out += f"wrote {path}  {before}L -> {after}L  sha 0000000b\n"
    return out + f"{len(rows)} hunk(s), 1 file(s), 0 failed, 0 advisories — {status}\n"


def counts(events):
    rp = lr.Replay()
    for cmd, result in events:
        rp.event(cmd, result)
    kinds = [o[0] for o in rp.outcomes]
    return len(kinds), kinds.count("R"), kinds.count("U")


PLAN = "mrw write - <<'EOF'\n@@ ...\nEOF"


class Replay(unittest.TestCase):
    def test_a_write_outside_a_ranged_read_after_a_write_is_refused(self):
        base = [("mrw read f:3-4", read("f", 12, 3, 4)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(base + [(PLAN, write(["6 replace -1 +1"], "f", 12, 12))]), (2, 1, 0))
        self.assertEqual(counts(base + [(PLAN, write(["5 replace -1 +1"], "f", 12, 12))]), (2, 0, 0))

    def test_a_whole_read_keeps_the_whole_licence(self):
        ev = [("mrw read f", whole("f", 12)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12)),
              (PLAN, write(["6 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(ev), (2, 0, 0))

    def test_an_insert_moves_the_carried_lines(self):
        base = [("mrw read f:10-20", read("f", 30, 10, 20)), (PLAN, write(["12 insert-after -0 +3"], "f", 30, 33))]
        self.assertEqual(counts(base + [(PLAN, write(["24 replace -1 +1"], "f", 33, 33))]), (2, 0, 0))
        self.assertEqual(counts(base + [(PLAN, write(["25 replace -1 +1"], "f", 33, 33))]), (2, 1, 0))

    def test_a_delete_moves_the_carried_lines(self):
        base = [("mrw read f:3-8", read("f", 12, 3, 8)), (PLAN, write(["4-5 delete -2 +0"], "f", 12, 10))]
        self.assertEqual(counts(base + [(PLAN, write(["4 replace -1 +1"], "f", 10, 10))]), (2, 0, 0))
        self.assertEqual(counts(base + [(PLAN, write(["8 replace -1 +1"], "f", 10, 10))]), (2, 1, 0))

    def test_a_created_file_is_whole(self):
        ev = [("mrw write --create f", write(["0 create -0 +5"], "f", 0, 5).replace("wrote", "created")),
              (PLAN, write(["4 replace -1 +1"], "f", 5, 5))]
        self.assertEqual(counts(ev), (2, 0, 0))

    def test_a_search_replace_hunk_is_licensed_by_its_quote(self):
        sr = "mrw write --format=search_replace -"
        self.assertEqual(counts([(sr, write(["3 replace -1 +1"], "f", 12, 12))]), (1, 0, 0))
        # its line after unlicensed under v1.83.0 too, yet it landed: the quote kept it
        self.assertEqual(counts([(sr, write(["3-4 replace -2 +2"], "f", 12, 12))]), (1, 0, 0))
        # its line after covered only by v1.83.0's whole-file licence: the replay cannot tell
        ev = [("mrw read f:3-4", read("f", 12, 3, 4)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12)),
              (sr, write(["6-7 replace -2 +2"], "f", 12, 12))]
        self.assertEqual(counts(ev), (2, 0, 1))

    def test_a_sub_write_and_a_forced_write_license_themselves(self):
        self.assertEqual(counts([("mrw write --grep a --sub b .", write(["3 replace -1 +1"], "f", 12, 12))]), (1, 0, 0))
        forced = [("mrw write --force -", write(["3 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(forced), (1, 0, 0))
        # ADR-052 holds under --force: a forced replace of 3-4 needs line 5, which nothing licenses now
        self.assertEqual(counts(forced + [("mrw write --force -", write(["3-4 replace -2 +2"], "f", 12, 12))]),
                         (2, 1, 0))

    def test_an_unlink_needs_the_whole_file(self):
        ev = [("mrw read f:3-4", read("f", 12, 3, 4)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12)),
              (PLAN, "ok   f - unlink  -0 +0\nremoved f  12L -> 0L  was sha 0000000b\n"
                     "1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied\n")]
        self.assertEqual(counts(ev), (2, 1, 0))

    def test_an_unplaceable_hunk_leaves_the_file_uncertain_until_a_whole_read(self):
        pat = (PLAN, "ok   f /^x/ replace  -1 +1\nwrote f  12L -> 12L  sha 0000000b\n"
                     "1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied\n")
        ev = [pat, (PLAN, write(["3 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(ev), (2, 0, 2))
        ev2 = ev + [("mrw read f:3-4", read("f", 12, 3, 4)), (PLAN, write(["9 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(ev2), (3, 0, 3))
        ev3 = ev2 + [("mrw read f", whole("f", 12)), (PLAN, write(["9 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(ev3), (4, 0, 3))

    def test_a_quiet_or_cut_read_makes_its_file_uncertain(self):
        quiet = [("mrw read --quiet f:3-4", read("f", 12, 3, 4)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(quiet), (1, 0, 1))
        cut = read("f", 30, 10, 20).split("\n")[:5]  # the header, the span, three lines: ends inside 10-21
        piped = [("mrw read f:10-20 | sed -n 1,5p", "\n".join(cut) + "\n"), (PLAN, write(["12 replace -1 +1"], "f", 30, 30))]
        self.assertEqual(counts(piped), (1, 0, 1))

    def test_a_refused_write_is_not_counted(self):
        refused = "FAIL f 6 replace (plan line 1): 6 of f has not been read\n" \
                  "1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN\n"
        self.assertEqual(counts([("mrw read f:3-4", read("f", 12, 3, 4)), (PLAN, refused)]), (0, 0, 0))

    def test_a_terse_receipt_is_unplaced(self):
        terse = "ok f 2 hunk(s) -2 +2 sha 0000000b\n2 hunk(s), 1 file(s), 0 failed, 0 advisories — applied\n"
        self.assertEqual(counts([("mrw write --terse -", terse)]), (2, 0, 2))


    def test_a_read_printed_after_a_write_does_not_license_it(self):
        base = [("mrw read f:3-4", read("f", 12, 3, 4)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12))]
        both = (PLAN + " && mrw read f:6", write(["6 replace -1 +1"], "f", 12, 12) + read("f", 12, 6, 6))
        self.assertEqual(counts(base + [both]), (2, 1, 0))

    def test_a_lost_licence_leaves_the_file_uncertain(self):
        ev = [(PLAN, write(["6 replace -1 +1"], "f", 12, 12)), (PLAN, write(["3 replace -1 +1"], "f", 12, 12))]
        self.assertEqual(counts(ev), (2, 0, 2))

    def test_a_create_clears_an_uncertain_file(self):
        ev = [("mrw read --quiet f", read("f", 12, 1, 12)),
              ("mrw write --create f", write(["0 create -0 +5"], "f", 0, 5).replace("wrote", "created")),
              (PLAN, write(["4 replace -1 +1"], "f", 5, 5))]
        self.assertEqual(counts(ev), (2, 0, 0))

class Verdict(unittest.TestCase):
    def test_the_verdict_is_the_pre_registered_rule(self):
        for args, want in (((100, 6, 0), "STOP"), ((100, 3, 1), "GO"), ((100, 3, 3), "INCONCLUSIVE"),
                           ((100, 0, 21), "INCONCLUSIVE"), ((40, 0, 0), "INCONCLUSIVE"), ((100, 5, 0), "GO")):
            self.assertEqual(lr.verdict(*args), want, args)


class Transcript(unittest.TestCase):
    def test_a_stream_json_transcript_is_read_in_order(self):
        lines = []
        for i, (cmd, result) in enumerate([("mrw read f:3-4", read("f", 12, 3, 4)),
                                           (PLAN, write(["3 replace -1 +1"], "f", 12, 12)),
                                           ("ls", "f"),
                                           (PLAN, write(["6 replace -1 +1"], "f", 12, 12))]):
            lines.append({"type": "assistant", "message": {"content": [
                {"type": "tool_use", "id": f"t{i}", "name": "Bash", "input": {"command": cmd}}]}})
            lines.append({"type": "user", "message": {"content": [
                {"type": "tool_result", "tool_use_id": f"t{i}", "content": [{"type": "text", "text": result}]}]}})
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "x.transcript.jsonl")
            with open(p, "w", encoding="utf-8") as fh:
                fh.write("\n".join(json.dumps(x) for x in lines) + "\n")
            events = lr.transcript_events(p)
        self.assertEqual(len(events), 3)  # `ls` names no mrw
        self.assertEqual(counts(events), (2, 1, 0))


if __name__ == "__main__":
    unittest.main()
