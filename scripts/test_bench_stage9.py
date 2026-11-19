"""Tests for bench_stage9.py: ADR-206 Decision 7, on run directories built in temporary directories to hit each rule."""
import itertools
import contextlib
import io
import json
import re
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import bench_stage9 as stage9  # noqa: E402

ROOT = Path(__file__).resolve().parent.parent
MODEL = "claude-opus-5-5"
M_HASH, N_HASH = "a" * 64, "b" * 64
CLASSES = {c: [f"{c}-{i:02d}" for i in range(12)] for c in ("single-site", "batch", "follow-on", "control")}
REFUSED = ("mrw: plan has 1 error(s), nothing was written\n"
           "f.go: hunk 1: line 12 of f.go has no recorded showing in this version, and a line number alone cannot say which line you mean.")
APPLIED = "ok   f.go 1 replace  -1 +1\n1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied"


def transcript(*steps):
    """A stream-json transcript: steps are (command, result text)."""
    rows = [{"type": "system", "subtype": "init"}]
    for i, (command, text) in enumerate(steps):
        rows.append({"type": "assistant", "message": {"content": [{"type": "tool_use", "id": str(i), "name": "Bash", "input": {"command": command}}]}})
        rows.append({"type": "user", "message": {"content": [{"type": "tool_result", "tool_use_id": str(i), "content": [{"type": "text", "text": text}]}]}})
    rows.append({"type": "result", "subtype": "success"})
    return "\n".join(json.dumps(r) for r in rows)


def run_json(arm, model=MODEL, tasks=(), **over):
    doc = {"stage": "stage9", "tasks": list(tasks), "models": [model], "configs": ["A" if arm == "a" else "B"], "repeats": 1, "seed": 206,
           "cli_version": "2.9.9", "mrw_version": "dev", "containment": "bypass", "status_line_sha256": "c" * 64, "budget_seconds": 1200,
           "max_turns": 60, "keep_diff": True, "bench_revision": "rev1"}
    if arm != "a":
        doc.update(mrw_sha256=M_HASH if arm == "m" else N_HASH, mrw_instructions_sha256="d" * 64, max_spend_usd=7.0 if arm == "m" else 3.0)
    if arm == "n":
        doc["keep_state_file"] = "mrw/stage9-exposed.jsonl"
    doc.update(over)
    return doc


class World:
    """A temporary receipts-and-transcripts tree: DIR/<model>/<arm>/ with run.json, receipts, kept logs and transcripts."""

    def __init__(self, test):
        self.tmp = tempfile.TemporaryDirectory()
        test.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "receipts"
        self.classes = Path(self.tmp.name) / "classes.json"
        self.classes.write_text(json.dumps(CLASSES), encoding="utf-8")
        self.pilot = Path(self.tmp.name) / "pilot.md"
        self.pilot.write_text(f"m sha256: {M_HASH}\nn sha256: {N_HASH}\n", encoding="utf-8")

    def arm(self, arm, specs, model=MODEL, run=None):
        """specs: {task: dict(solved, outside, cost, exclusion, is_error, over_budget, status, kept, log, retry, steps)}."""
        d = self.root / model / arm
        d.mkdir(parents=True, exist_ok=True)
        (d / "run.json").write_text(json.dumps(run if run is not None else run_json(arm, model, list(specs))), encoding="utf-8")
        for task, s in specs.items():
            config = "A" if arm == "a" else "B"
            stem = f"{task}.{config}.{model}.1" + (f".r{s['retry']}" if s.get("retry") else "")
            rec = {"attempt": stem, "task": task, "config": config, "model": model, "repeat": 1, "num_turns": 5, "tool_calls": 7, "wall_ms": 900,
                   "is_error": s.get("is_error", False), "total_cost_usd": s.get("cost", 0.1), "status": s.get("status", "done"),
                   "usage": {"input_tokens": 1, "output_tokens": 2, "cache_read_input_tokens": 3, "cache_creation_input_tokens": 4},
                   "oracle": {"solved": s.get("solved", True)}}
            if s.get("outside"):
                rec["oracle"]["outside"] = s["outside"]
            if s.get("over_budget"):
                rec["over_budget"] = True
            if s.get("exclusion"):
                rec["exclusion"] = s["exclusion"]
            if s.get("note"):
                rec["kept_note"] = s["note"]
            (d / f"{stem}.receipt.json").write_text(json.dumps(rec), encoding="utf-8")
            if s.get("log"):
                (d / f"{stem}.kept").write_text("".join(json.dumps({"path": "f.go", "op": "replace", "start": 4, "end": 4, "first_unshown": 4, "sha_before": "e" * 64}) + "\n" for _ in range(s["log"])), encoding="utf-8")
            if not s.get("exclusion"):
                (d / f"{stem}.transcript.jsonl").write_text(transcript(*s.get("steps", [("true", "")])), encoding="utf-8")
        return d

    def count(self, pilot=True):
        return stage9.count(self.root, self.root, self.classes, self.pilot if pilot else None)

    def attempts(self, arm, model=MODEL, **kw):
        return self.count(**kw)["models"][model][arm]["tasks"]


def tasks_of(cls, n=12, **s):
    return {t: dict(s) for t in CLASSES[cls][:n]}


def cli(*argv):
    out, err = io.StringIO(), io.StringIO()
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
        code = stage9.main(list(argv))
    return code, out.getvalue(), err.getvalue()


class Classification(unittest.TestCase):
    def test_each_outcome_and_the_partition(self):
        w = World(self)
        specs = {
            "follow-on-00": {},
            "follow-on-01": dict(solved=False, outside=["f.go:9"]),
            "follow-on-02": dict(solved=False),
            "follow-on-03": dict(exclusion="infrastructure"),
            "follow-on-04": dict(is_error=True),
            "follow-on-05": dict(over_budget=True),
            "follow-on-06": dict(is_error=True, solved=True, outside=["x"]),
            "follow-on-07": dict(exclusion="setup", solved=True),
        }
        w.arm("m", specs)
        got = {t: a["outcome"] for t, a in w.attempts("m").items() if t in specs}
        self.assertEqual(got, {"follow-on-00": "correct", "follow-on-01": "outside", "follow-on-02": "other", "follow-on-03": "missing",
                               "follow-on-04": "other", "follow-on-05": "other", "follow-on-06": "outside", "follow-on-07": "missing"})

    def test_a_planned_task_with_no_receipt_is_missing_and_not_started(self):
        w = World(self)
        d = w.arm("m", {"follow-on-00": {}})
        run = json.loads((d / "run.json").read_text(encoding="utf-8"))
        run["tasks"] = ["follow-on-00", "follow-on-01"]
        (d / "run.json").write_text(json.dumps(run), encoding="utf-8")
        got = w.attempts("m")["follow-on-01"]
        self.assertEqual((got["outcome"], got["started"]), ("missing", False))

    def test_claimed_is_status_done(self):
        w = World(self)
        w.arm("m", {"follow-on-00": dict(status="done"), "follow-on-01": dict(status="blocked"), "follow-on-02": dict(status="")})
        got = {t: a["claimed"] for t, a in w.attempts("m").items()}
        self.assertEqual(got, {"follow-on-00": True, "follow-on-01": False, "follow-on-02": False})

    def test_a_replacement_counts_and_the_excluded_first_try_does_not(self):
        w = World(self)
        d = w.arm("m", {"follow-on-00": dict(exclusion="start-error", cost=0.05)})
        w.arm("m", {"follow-on-00": dict(retry=1, cost=0.2)})
        (d / "run.json").write_text(json.dumps(run_json("m", MODEL, ["follow-on-00"])), encoding="utf-8")
        got = w.attempts("m")["follow-on-00"]
        self.assertEqual((got["outcome"], got["spend"], got["attempt"]), ("correct", 0.2, f"follow-on-00.B.{MODEL}.1.r1"))
        self.assertEqual(stage9.spend(w.root), "stage9 spend usd 0.250000 missing 0")

    def test_the_chain_follows_the_producers_rule_and_names_the_strays(self):
        w = World(self)
        tasks = [f"follow-on-0{i}" for i in range(5)]
        run = run_json("m", MODEL, tasks)
        w.arm("m", {"follow-on-00": dict(exclusion="setup"), "follow-on-01": dict(exclusion="start-error"),
                    "follow-on-02": dict(exclusion="infrastructure"), "follow-on-03": {}, "follow-on-04": dict(exclusion="setup")}, run=run)
        w.arm("m", {"follow-on-00": dict(retry=1, cost=0.2), "follow-on-01": dict(retry=1, exclusion="start-error"), "follow-on-02": dict(retry=1),
                    "follow-on-03": dict(retry=1), "follow-on-04": dict(retry=1, exclusion="setup")}, run=run)
        w.arm("m", {"follow-on-00": dict(retry=2), "follow-on-01": dict(retry=2, cost=0.3), "follow-on-04": dict(retry=2)}, run=run)
        got = w.attempts("m")
        stem = f".B.{MODEL}.1"
        want = {"follow-on-00": ("correct", ".r1", [".r2"]), "follow-on-01": ("correct", ".r2", []), "follow-on-02": ("missing", "", [".r1"]),
                "follow-on-03": ("correct", "", [".r1"]), "follow-on-04": ("missing", ".r1", [".r2"])}
        for task, (outcome, suffix, stray) in want.items():
            self.assertEqual((got[task]["outcome"], got[task]["attempt"], got[task]["stray"]),
                             (outcome, task + stem + suffix, [task + stem + s for s in stray]), task)
        self.assertIn(f"stray receipts ignored (m): follow-on-00.B.{MODEL}.1.r2, follow-on-02.B.{MODEL}.1.r1", stage9.report(w.root, w.count()))

    def test_the_numbers_beside_the_outcome(self):
        w = World(self)
        w.arm("m", {"follow-on-00": {}})
        got = w.attempts("m")["follow-on-00"]
        self.assertEqual((got["num_turns"], got["tool_calls"], got["wall_ms"], got["spend"]), (5, 7, 900, 0.1))
        self.assertEqual(got["tokens"], {"input": 1, "output": 2, "cache_read": 3, "cache_write": 4})
        self.assertRegex(got["transcript_sha256"], r"^[0-9a-f]{64}$")


class Spend(unittest.TestCase):
    def test_a_null_cost_is_missing_never_zero(self):
        w = World(self)
        w.arm("m", {"follow-on-00": dict(cost=None), "follow-on-01": dict(cost=0.25)})
        self.assertIsNone(w.attempts("m")["follow-on-00"]["spend"])
        self.assertEqual(stage9.spend(w.root), "stage9 spend usd 0.250000 missing 1")

    def test_spend_is_exactly_one_line_of_six_fields_over_every_receipt(self):
        w = World(self)
        w.arm("a", {"single-site-00": dict(cost=0.1), "single-site-01": dict(exclusion="infrastructure", cost=0.3)})
        w.arm("m", {"follow-on-00": dict(cost=0.2)}, model="claude-sonnet-5-5")
        code, out, _ = cli("spend", "--receipts", str(w.root))
        self.assertEqual((code, out), (0, "stage9 spend usd 0.600000 missing 0\n"))
        self.assertEqual(len(out.split()), 6)


class Insufficient(unittest.TestCase):
    def pair(self, x, y, cls="follow-on", n=12, a="m", b="n"):
        w = World(self)
        w.arm(a, {t: dict(s) for t, s in x(CLASSES[cls][:n]).items()})
        w.arm(b, {t: dict(s) for t, s in y(CLASSES[cls][:n]).items()})
        return w

    def line(self, w, a="m", b="n", cls="follow-on"):
        text = stage9.report(w.root, w.count())
        found = [ln for ln in text.splitlines() if ln.strip().startswith(f"{a} vs {b} ({cls}):")]
        self.assertEqual(len(found), 1, text)
        return found[0].strip()

    def test_under_ten_paired(self):
        w = self.pair(lambda ts: {t: {} for t in ts}, lambda ts: {t: {} for t in ts[:9]})
        self.assertRegex(self.line(w), r"INSUFFICIENT: 9 of 12 tasks paired, under 10")

    def test_ten_paired_is_enough(self):
        w = self.pair(lambda ts: {t: {} for t in ts}, lambda ts: {t: {} for t in ts[:10]})
        self.assertNotIn("INSUFFICIENT", self.line(w))

    def test_a_missing_task_in_either_arm_is_not_paired(self):
        w = self.pair(lambda ts: {t: {} for t in ts}, lambda ts: {t: dict(exclusion="infrastructure") if i < 3 else {} for i, t in enumerate(ts)})
        self.assertRegex(self.line(w), r"INSUFFICIENT: 9 of 12 tasks paired")

    def test_a_paired_cost_missing(self):
        w = self.pair(lambda ts: {t: dict(cost=None) if i == 0 else {} for i, t in enumerate(ts)}, lambda ts: {t: {} for t in ts})
        self.assertRegex(self.line(w), r"INSUFFICIENT: 1 paired cost\(s\) missing")

    def test_zero_correct_in_an_arm(self):
        w = self.pair(lambda ts: {t: dict(solved=False) for t in ts}, lambda ts: {t: {} for t in ts})
        self.assertRegex(self.line(w), r"INSUFFICIENT: no correct attempt in arm m")
        w = self.pair(lambda ts: {t: {} for t in ts}, lambda ts: {t: dict(solved=False) for t in ts})
        self.assertRegex(self.line(w), r"INSUFFICIENT: no correct attempt in arm n")

    def test_over_five_percent_of_resamples_dropped(self):
        w = self.pair(lambda ts: {t: dict(solved=i == 0) for i, t in enumerate(ts)}, lambda ts: {t: {} for t in ts})
        line = self.line(w)
        self.assertRegex(line, r"INSUFFICIENT: \d+ of 4000 resamples dropped, over 5%")
        self.assertNotIn("ratio", line)

    def test_a_ratio_is_printed_when_nothing_is_insufficient(self):
        w = self.pair(lambda ts: {t: dict(cost=0.1) for t in ts}, lambda ts: {t: dict(cost=0.2) for t in ts})
        line = self.line(w)
        self.assertIn("paired 12; correct m 12 n 12; spend per correct m 0.1 n 0.2; ratio m/n 0.5 [95% 0.5, 0.5]; excludes 1: yes; resamples dropped 0 of 4000", line)


    def test_the_five_percent_boundary_is_200_of_4000(self):
        w = self.pair(lambda ts: {t: {} for t in ts}, lambda ts: {t: {} for t in ts})
        with mock.patch.object(stage9, "bootstrap", return_value=(0.9, 1.1, 200)):
            self.assertIn("ratio m/n", self.line(w))
        with mock.patch.object(stage9, "bootstrap", return_value=(0.9, 1.1, 201)):
            self.assertRegex(self.line(w), r"INSUFFICIENT: 201 of 4000 resamples dropped")

    def test_excludes_one_is_judged_on_the_unrounded_ends(self):
        w = self.pair(lambda ts: {t: {} for t in ts}, lambda ts: {t: {} for t in ts})
        for ends, want in (((1.0000001, 1.5), "yes"), ((0.5, 0.9999999), "yes"), ((0.9999999, 1.0000001), "no"), ((1.0, 1.5), "no"), ((0.5, 1.0), "no")):
            with mock.patch.object(stage9, "bootstrap", return_value=(*ends, 0)):
                self.assertIn(f"excludes 1: {want}", self.line(w), ends)
    def test_m_against_a_is_read_on_single_site_and_batch_only(self):
        w = World(self)
        w.arm("a", {**tasks_of("single-site"), **tasks_of("batch"), **tasks_of("follow-on")})
        w.arm("m", {**tasks_of("single-site"), **tasks_of("batch"), **tasks_of("follow-on"), **tasks_of("control")})
        w.arm("n", {**tasks_of("follow-on"), **tasks_of("control")})
        text = stage9.report(w.root, w.count())
        for want in ("m vs a (single-site):", "m vs a (batch):", "m vs n (follow-on):", "m vs n (control):"):
            self.assertIn(want, text)
        self.assertNotIn("m vs a (follow-on)", text)
        self.assertNotIn("m vs n (batch)", text)

    def test_a_missing_arm_is_insufficient_not_a_crash(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"))
        self.assertRegex(self.line(w), r"INSUFFICIENT: arm n has no run")


class Bootstrap(unittest.TestCase):
    ROWS = [(0.10 + 0.03 * i, 1, 0.20 + 0.05 * (i % 4), 1 if i % 5 else 0) for i in range(12)]

    def test_the_same_input_gives_the_same_interval(self):
        self.assertEqual(stage9.bootstrap(self.ROWS), stage9.bootstrap(self.ROWS))

    def test_the_seed_is_206_and_the_interval_is_pinned(self):
        # Pinned from this implementation at seed 206; a clock seed or another seed moves it.
        low, high, dropped = stage9.bootstrap(self.ROWS)
        self.assertEqual((round(low, 4), round(high, 4), dropped), PINNED)

    def test_the_resample_count_and_seed_are_the_registered_ones(self):
        self.assertEqual((stage9.RESAMPLES, stage9.SEED, stage9.MAX_DROPPED_FRACTION), (4000, 206, 0.05))

    def test_three_tasks_against_the_exact_distribution_of_all_27_resamples(self):
        # Independent of the script's draws: every resample of 3 paired tasks, each 1/27 likely. All are correct, so the
        # statistic is the ratio of the resample's two spend sums.
        rows = [(0.1, 1, 0.3, 1), (0.2, 1, 0.2, 1), (0.4, 1, 0.1, 1)]
        dist = sorted(sum(rows[j][0] for j in i) / sum(rows[j][2] for j in i) for i in itertools.product(range(3), repeat=3))
        low, high, dropped = stage9.bootstrap(rows)
        self.assertEqual(dropped, 0)
        # The 2.5th and 97.5th percentiles of that distribution are its ends, each 1/27 (3.7%) likely: 4000 draws put the
        # 100th and the 3900th sorted value in them, and a 6th or 94th percentile one value further in.
        self.assertGreater(dist[1], dist[0] + 1e-9)
        self.assertGreater(dist[26], dist[25] + 1e-9)
        self.assertAlmostEqual(low, dist[0], places=9)
        self.assertAlmostEqual(high, dist[26], places=9)
        point = (sum(r[0] for r in rows) / 3) / (sum(r[2] for r in rows) / 3)
        self.assertLessEqual(low, point)
        self.assertLessEqual(point, high)
        # Swapping the arms inverts the interval: the same seed draws the same resamples.
        swapped = [(r[2], r[3], r[0], r[1]) for r in rows]
        slow, shigh, _ = stage9.bootstrap(swapped)
        self.assertAlmostEqual(slow, 1 / high, places=9)
        self.assertAlmostEqual(shigh, 1 / low, places=9)

    def test_a_resample_with_an_undefined_or_zero_figure_is_dropped_and_counted(self):
        rows = [(0.1, 1, 0.1, 1), (0.1, 0, 0.1, 1)]
        low, high, dropped = stage9.bootstrap(rows)
        self.assertGreater(dropped, 0)
        self.assertLess(dropped, stage9.RESAMPLES)
        self.assertLessEqual(low, high)
        _, _, dropped = stage9.bootstrap([(0.0, 1, 0.1, 1)])
        self.assertEqual(dropped, stage9.RESAMPLES)


PINNED = (0.4405, 0.9928, 0)


class Exposure(unittest.TestCase):
    def test_in_n_it_is_a_line_in_the_kept_log(self):
        w = World(self)
        w.arm("n", {"follow-on-00": dict(log=2), "follow-on-01": {}, "follow-on-02": dict(log=1)})
        got = w.attempts("n")
        self.assertEqual([(got[t]["exposed"], got[t]["hunks"]) for t in sorted(got)], [(True, 2), (False, 0), (True, 1)])

    def test_in_n_a_log_that_is_not_json_is_refused(self):
        w = World(self)
        d = w.arm("n", {"follow-on-00": {}})
        (d / f"follow-on-00.B.{MODEL}.1.kept").write_text("not json\n", encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "kept log"):
            w.count()

    def test_in_m_it_is_a_refusal_in_the_output_of_a_write(self):
        w = World(self)
        w.arm("m", {
            "follow-on-00": dict(steps=[("mrw write plan.txt", REFUSED)]),
            "follow-on-01": dict(steps=[("mrw write plan.txt", APPLIED)]),
            "follow-on-02": dict(steps=[("mrw write --dry-run plan.txt", REFUSED), ("mrw -C d write -", REFUSED + "\n" + REFUSED)]),
        })
        got = w.attempts("m")
        self.assertEqual([(got[t]["exposed"], got[t]["hunks"], got[t]["write_results_read"]) for t in sorted(got)], [(True, 1, 1), (False, 0, 1), (True, 3, 2)])

    def test_the_text_of_mrw_instructions_alone_does_not_count(self):
        w = World(self)
        w.arm("m", {"follow-on-00": dict(steps=[("mrw instructions", "A refusal reads: " + REFUSED), ("mrw read f.go", REFUSED)])})
        got = w.attempts("m")["follow-on-00"]
        self.assertEqual((got["exposed"], got["hunks"], got["write_results_read"]), (False, 0, 0))

    def test_a_write_result_the_host_cut_is_not_read_and_is_said(self):
        w = World(self)
        w.arm("m", {"follow-on-00": dict(steps=[("mrw write plan.txt > out.txt", ""), ("mrw write plan.txt", APPLIED)])})
        got = w.attempts("m")["follow-on-00"]
        self.assertEqual((got["write_calls"], got["write_results_read"], got["exposed"]), (2, 1, False))


    def test_what_counts_as_the_output_of_a_write(self):
        cases = [
            ("mrw write p", "read"), ("command mrw write p", "read"), ("env X=1 mrw write p", "read"), ("./bin/mrw write p", "read"),
            ("/usr/local/bin/mrw -C d write -", "read"), ('"$MRW" write p', "read"), ("$MRW --root d write p", "read"),
            ("cd d && mrw write p 2>&1", "read"), ("mrw write - <<'PLAN'\n@@ f 1 replace\nmrw instructions > y\nPLAN", "read"),
            ("mrw write p >/dev/null 2>&1; echo Finished", "redirected"), ("mrw write p > out.txt", "redirected"),
            ("mrw write p &> out.txt", "redirected"), ("mrw write p 2>/dev/null", "redirected"), ("mrw write p >> log", "redirected"),
            ("mrw instructions; mrw write p", "mixed"), ("mrw --instructions; mrw write p", "mixed"), ("mrw skill && mrw write p", "mixed"),
            ("mrw instructions; mrw write p >/dev/null 2>&1", "mixed"), ("command mrw instructions | head -3; ./bin/mrw write p", "mixed"),
            ("mrw read f.go", None), ("mrw instructions", None), ("echo mrw writer", None), ("mrw read f.go > out.txt", None),
        ]
        for command, want in cases:
            self.assertEqual(stage9.classify_command(command), want, command)

    def test_instructions_beside_a_write_and_a_redirected_write_are_not_read(self):
        w = World(self)
        w.arm("m", {
            "follow-on-00": dict(steps=[("mrw instructions; mrw write p >/dev/null 2>&1", "A refusal reads: " + REFUSED)]),
            "follow-on-01": dict(steps=[("mrw write p >/dev/null 2>&1; echo Finished", "Finished")]),
            "follow-on-02": dict(steps=[("command mrw write p", REFUSED), ("./bin/mrw write p", REFUSED), ("$MRW write p", APPLIED)]),
        })
        got = w.attempts("m")
        fields = ("exposed", "hunks", "write_calls", "write_results_read", "write_not_read", "write_mixed")
        self.assertEqual([tuple(got[t][f] for f in fields) for t in sorted(got)],
                         [(False, 0, 1, 0, 0, 1), (False, 0, 1, 0, 1, 0), (True, 2, 3, 3, 0, 0)])

    def test_in_n_a_state_file_not_retained_is_unknown_not_zero(self):
        w = World(self)
        w.arm("n", {"follow-on-00": dict(note="state: not kept: the state home is a link or a file, not a directory"),
                    "follow-on-01": dict(log=1), "follow-on-02": dict(note="diff: not kept: too big")})
        got = w.attempts("n")
        self.assertEqual([(got[t]["exposure_unknown"], got[t]["exposed"], got[t]["hunks"]) for t in sorted(got)],
                         [(True, None, None), (False, True, 1), (False, False, 0)])
        text = stage9.report(w.root, w.count())
        self.assertIn("exposed n: 1 of 2 graded attempts; hunks 1; exposure unknown 1", text)
    def test_arm_a_has_no_exposure(self):
        w = World(self)
        w.arm("a", {"single-site-00": {}})
        got = w.attempts("a")["single-site-00"]
        self.assertEqual((got["exposed"], got["hunks"]), (None, None))

    def test_the_phrase_is_the_one_internal_apply_writes(self):
        text = (ROOT / "internal" / "apply" / "apply.go").read_text(encoding="utf-8")
        self.assertIn("has no recorded showing in this version", text)
        self.assertTrue(re.search(stage9.EXPOSED.pattern, "line 12 of f.go has no recorded showing in this version"))


class Refusals(unittest.TestCase):
    def full(self, w):
        w.arm("a", tasks_of("single-site"))
        w.arm("m", tasks_of("follow-on"))
        w.arm("n", tasks_of("follow-on"))

    def test_a_consistent_set_is_counted_and_may_differ_where_allowed(self):
        w = World(self)
        self.full(w)
        self.assertEqual(sorted(w.count()["models"][MODEL]), ["a", "m", "n"])

    def test_a_plan_that_differs_is_refused_naming_the_key(self):
        for key, value in (("seed", 7), ("max_turns", 30), ("bench_revision", "other"), ("cli_version", "3.0.0"), ("status_line_sha256", "f" * 64)):
            w = World(self)
            w.arm("a", tasks_of("single-site"))
            w.arm("m", tasks_of("follow-on"), run=run_json("m", MODEL, CLASSES["follow-on"], **{key: value}))
            with self.assertRaisesRegex(stage9.Refusal, key):
                w.count()

    def test_equal_m_and_n_hashes_are_refused(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"))
        w.arm("n", tasks_of("follow-on"), run=run_json("n", MODEL, CLASSES["follow-on"], mrw_sha256=M_HASH))
        with self.assertRaisesRegex(stage9.Refusal, "same mrw hash"):
            w.count()

    def test_a_hash_that_is_not_the_arms_in_pilot_md_is_refused(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"), run=run_json("m", MODEL, CLASSES["follow-on"], mrw_sha256="9" * 64))
        with self.assertRaisesRegex(stage9.Refusal, "pilot.md"):
            w.count()
        w = World(self)
        w.arm("n", tasks_of("follow-on"), run=run_json("n", MODEL, CLASSES["follow-on"], mrw_sha256=M_HASH))
        with self.assertRaisesRegex(stage9.Refusal, "pilot.md"):
            w.count()

    def test_a_run_of_the_wrong_model_or_configuration_is_refused(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"), run=run_json("m", "claude-sonnet-5-5", CLASSES["follow-on"]))
        with self.assertRaisesRegex(stage9.Refusal, "models"):
            w.count()
        w = World(self)
        w.arm("m", tasks_of("follow-on"), run=run_json("m", MODEL, CLASSES["follow-on"], configs=["A"]))
        with self.assertRaisesRegex(stage9.Refusal, "configs"):
            w.count()

    def test_a_task_in_no_class_is_refused(self):
        w = World(self)
        w.arm("m", {"stray-task": {}})
        with self.assertRaisesRegex(stage9.Refusal, "stray-task"):
            w.count()

    def test_the_cli_exits_2_with_the_reason(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"))
        w.arm("n", tasks_of("follow-on"), run=run_json("n", MODEL, CLASSES["follow-on"], mrw_sha256=M_HASH))
        code, out, err = cli("count", "--receipts", str(w.root), "--transcripts", str(w.root), "--classes", str(w.classes), "--pilot", str(w.pilot))
        self.assertEqual((code, out), (2, ""))
        self.assertIn("same mrw hash", err)

    def test_count_requires_a_pilot_file_that_names_each_hash_once(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"))
        with self.assertRaisesRegex(stage9.Refusal, "needs --pilot"):
            w.count(pilot=False)
        w.pilot.write_text(f"m sha256: {M_HASH}\nm sha256: {M_HASH}\nn sha256: {N_HASH}\n", encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "`m sha256:` 2 times"):
            w.count()
        w.pilot.write_text(f"m sha256: {M_HASH}\n", encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "`n sha256:` 0 times"):
            w.count()


class Report(unittest.TestCase):
    def build(self, exposed_outside=True):
        w = World(self)
        w.arm("m", {t: dict(steps=[("mrw write p", REFUSED)]) if i < 2 else {} for i, t in enumerate(CLASSES["follow-on"])})
        n = {t: {} for t in CLASSES["follow-on"]}
        n[CLASSES["follow-on"][0]] = dict(log=2, solved=False, outside=["f.go:3"]) if exposed_outside else dict(log=2)
        n[CLASSES["follow-on"][1]] = dict(log=1)
        w.arm("n", n)
        return w

    def test_counts_per_arm_and_the_exposed_lists(self):
        w = self.build()
        text = stage9.report(w.root, w.count())
        self.assertIn(f"model {MODEL}", text)
        self.assertIn("m: planned 12 started 12 graded 12 missing 0; correct 12 outside 0 other 0; claimed 12", text)
        self.assertIn("n: planned 12 started 12 graded 12 missing 0; correct 11 outside 1 other 0; claimed 12", text)
        self.assertIn("exposed m: 2 of 12 graded attempts; hunks 2; write results read 2 of 2 write calls", text)
        self.assertIn("exposed n: 2 of 12 graded attempts; hunks 3", text)
        self.assertIn("not attributable in m:", text)
        self.assertIn(f"n exposed and outside: follow-on-00.B.{MODEL}.1", text)

    def test_no_exposed_outside_attempt_is_said_as_none(self):
        w = self.build(exposed_outside=False)
        self.assertIn("n exposed and outside: none", stage9.report(w.root, w.count()))

    def test_the_report_is_byte_identical_twice_and_has_no_verdict_words(self):
        w = self.build()
        first = stage9.report(w.root, w.count())
        self.assertEqual(first, stage9.report(w.root, w.count()))
        self.assertNotRegex(first, r"(?i)\b(ship|pass|fail|wins?|better|worse|significant)\b")

    def test_both_models_are_printed(self):
        w = self.build()
        w.arm("m", tasks_of("follow-on"), model="claude-sonnet-5-5")
        text = stage9.report(w.root, w.count())
        self.assertIn("model claude-opus-5-5", text)
        self.assertIn("model claude-sonnet-5-5", text)

    def test_count_then_report_through_the_cli(self):
        w = self.build()
        code, out, _ = cli("count", "--receipts", str(w.root), "--transcripts", str(w.root), "--classes", str(w.classes), "--pilot", str(w.pilot))
        self.assertEqual(code, 0)
        counts = Path(w.tmp.name) / "counts.json"
        counts.write_text(out, encoding="utf-8")
        code, text, _ = cli("report", "--receipts", str(w.root), "--counts", str(counts))
        self.assertEqual(code, 0)
        self.assertEqual(text, stage9.report(w.root, json.loads(out)))
        self.assertIn("m vs n (follow-on):", text)

    def test_a_counts_file_of_other_runs_is_refused(self):
        w = self.build()
        counts = w.count()
        counts["models"][MODEL]["m"]["run_sha256"] = "0" * 64
        with self.assertRaisesRegex(stage9.Refusal, "counts file"):
            stage9.report(w.root, counts)

    def test_a_run_json_changed_after_counting_is_refused_even_where_it_may_differ(self):
        w = self.build()
        counts = w.count()
        path = w.root / MODEL / "n" / "run.json"
        path.write_text(json.dumps({**json.loads(path.read_text(encoding="utf-8")), "max_spend_usd": 99.0}), encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "counts file"):
            stage9.report(w.root, counts)


if __name__ == "__main__":
    unittest.main()
