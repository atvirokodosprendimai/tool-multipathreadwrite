"""Tests for bench_stage9.py: ADR-206 Decision 7, on run directories built in temporary directories to hit each rule."""
import itertools
import contextlib
import hashlib
import io
import json
import os
import re
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import bench_stage9 as stage9  # noqa: E402

ROOT = Path(__file__).resolve().parent.parent
MODEL = "claude-opus-5-5"
SONNET = "claude-sonnet-5-5"
M_HASH, N_HASH, I_HASH = "a" * 64, "b" * 64, "d" * 64
INIT = "stub-init"
FIGURE = {"a": 14.0, "m": 14.0, "n": 7.0}
CLASSES = {c: [f"{c}-{i:02d}" for i in range(12)] for c in ("single-site", "batch", "follow-on", "control")}
REFUSED_ROW = "FAIL f.go 12 replace (plan line 1): line 12 of f.go has no recorded showing in this version, and a line number alone cannot say which line you mean."
OTHER_ROW = "FAIL f.go 3 replace (plan line 2): anchor \"x\" not in line 3"


def receipt(*rows, ending="NOTHING WRITTEN"):
    """The text receipt of a write: its FAIL rows and then its summary line."""
    failed = sum(r.startswith("FAIL") for r in rows)
    return "\n".join([*rows, f"{len(rows)} hunk(s), 1 file(s), {failed} failed, 0 advisories — {ending}"])


REFUSED = receipt(REFUSED_ROW)
APPLIED = "ok   f.go 1 replace  -1 +1\n1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied"
EVENT = {"path": "f.go", "op": "replace", "start": 4, "end": 4, "first_unshown": 4, "sha_before": "e" * 64}


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
           "max_turns": 60, "keep_diff": True, "bench_revision": "rev1", "max_spend_usd": FIGURE[arm]}
    if arm != "a":
        doc.update(mrw_sha256=M_HASH if arm == "m" else N_HASH, mrw_instructions_sha256=I_HASH)
    if arm == "n":
        doc["keep_scratch_file"] = "stage9-exposed.jsonl"
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
        self.pilot.write_text(f"m sha256: {M_HASH}\nn sha256: {N_HASH}\ninstructions sha256: {I_HASH}\ninit model {MODEL}: {INIT}\ninit model {SONNET}: {INIT}\n", encoding="utf-8")
        self.plans_file = Path(self.tmp.name) / "plans.json"

    def arm(self, arm, specs, model=MODEL, run=None):
        """specs: {task: dict(solved, outside, cost, exclusion, is_error, over_budget, status, kept, log, retry, steps)}."""
        d = self.root / model / arm
        d.mkdir(parents=True, exist_ok=True)
        doc = run if run is not None else run_json(arm, model, list(specs))
        (d / "run.json").write_text(json.dumps(doc), encoding="utf-8")
        for task, s in specs.items():
            config = "A" if arm == "a" else "B"
            suffix = s["suffix"] if "suffix" in s else f".r{s['retry']}" if s.get("retry") else ""
            stem = f"{task}.{config}.{model}.1{suffix}"
            rec = {"attempt": stem, "task": task, "config": config, "model": model, "repeat": 1, "num_turns": 5, "tool_calls": 7, "wall_ms": 900,
                   "is_error": s.get("is_error", False), "total_cost_usd": s.get("cost", 0.1), "status": s.get("status", "done"),
                   "usage": {"input_tokens": 1, "output_tokens": 2, "cache_read_input_tokens": 3, "cache_creation_input_tokens": 4},
                   "oracle": {"solved": s.get("solved", True)}}
            if doc.get("mrw_sha256"):  # the runner writes the run's own build into each receipt (its readsBack checks it)
                rec["mrw_sha256"] = s.get("mrw", doc["mrw_sha256"])
            if not s.get("exclusion"):
                rec["init_model"] = s.get("init_model", INIT)
                text = transcript(*s.get("steps", [("true", "")]))
                (d / f"{stem}.transcript.jsonl").write_text(text, encoding="utf-8")
                rec["transcript_sha256"] = s.get("transcript_sha256", hashlib.sha256(text.encode("utf-8")).hexdigest())
            if s.get("after"):
                rec["mrw_sha256_after"] = s["after"]
            if s.get("slipped"):
                rec["slipped_writes"] = ["f.go"]
            for key in s.get("drop", ()):
                rec.pop(key, None)
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
                (d / f"{stem}.kept").write_text("".join(json.dumps(EVENT) + "\n" for _ in range(s["log"])), encoding="utf-8")
            if "logtext" in s:
                (d / f"{stem}.kept").write_text(s["logtext"], encoding="utf-8")
        return d

    def derive_plans(self):
        """The registered plans of what is in the tree: each run directory as its own run.json describes it."""
        plans = []
        for model_dir in sorted(p for p in self.root.iterdir() if p.is_dir()):
            for d in sorted(p for p in model_dir.iterdir() if p.is_dir()):
                run = json.loads((d / "run.json").read_text(encoding="utf-8"))
                arm = d.name if d.name in stage9.ARMS else "a"
                plans.append({"model": model_dir.name, "arm": arm, "directory": f"{model_dir.name}/{d.name}", "configs": run["configs"],
                              "max_spend_usd": run.get("max_spend_usd"), "tasks": run.get("tasks")})
        return plans

    def plans(self, plans=None):
        self.plans_file.write_text(json.dumps(self.derive_plans() if plans is None else plans), encoding="utf-8")
        return self.plans_file

    def count(self, pilot=True, plans=None):
        return stage9.count(self.root, self.root, self.classes, self.pilot if pilot else None, self.plans(plans))

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
        self.assertEqual(stage9.spend(w.root), "stage9 spend usd 0.25 missing 0")

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

    def test_only_the_runners_exact_ids_make_the_chain(self):
        w = World(self)
        tasks = ["follow-on-00", "follow-on-01", "follow-on-02"]
        run = run_json("m", MODEL, tasks)
        # 00 is fine, so any other receipt of it is stray, whatever its suffix looks like; 01 was excluded as a start error but
        # its .r1 is missing, so the .r2 and a .r01 do not stand in for it; 02's .r1 is there and counts.
        w.arm("m", {"follow-on-00": {}, "follow-on-01": dict(exclusion="start-error"), "follow-on-02": dict(exclusion="start-error")}, run=run)
        w.arm("m", {"follow-on-00": dict(suffix=".r01"), "follow-on-01": dict(suffix=".r2"), "follow-on-02": dict(retry=1)}, run=run)
        w.arm("m", {"follow-on-00": dict(suffix=".r10"), "follow-on-01": dict(suffix=".r01"), "follow-on-02": dict(suffix=".rx")}, run=run)
        got = w.attempts("m")
        stem = f".B.{MODEL}.1"
        self.assertEqual((got["follow-on-00"]["attempt"], got["follow-on-00"]["stray"]), ("follow-on-00" + stem, ["follow-on-00" + stem + s for s in (".r01", ".r10")]))
        self.assertEqual((got["follow-on-01"]["outcome"], got["follow-on-01"]["stray"]), ("missing", ["follow-on-01" + stem + s for s in (".r01", ".r2")]))
        self.assertEqual((got["follow-on-02"]["attempt"], got["follow-on-02"]["stray"]), ("follow-on-02" + stem + ".r1", ["follow-on-02" + stem + ".rx"]))

    def test_safe_is_the_runners_spelling_of_a_model_in_an_id(self):
        self.assertEqual(stage9.safe("a/b c:d-1.5"), "a_b_c_d-1.5")

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
        self.assertEqual(stage9.spend(w.root), "stage9 spend usd 0.25 missing 1")

    def test_spend_is_exactly_one_line_of_six_fields_over_every_receipt(self):
        w = World(self)
        w.arm("a", {"single-site-00": dict(cost=0.1), "single-site-01": dict(exclusion="infrastructure", cost=0.3)})
        w.arm("m", {"follow-on-00": dict(cost=0.2)}, model="claude-sonnet-5-5")
        code, out, _ = cli("spend", "--receipts", str(w.root))
        self.assertEqual((code, out), (0, "stage9 spend usd 0.6 missing 0\n"))
        self.assertEqual(len(out.split()), 6)

    def spend_of(self, *costs):
        """The spend line for receipts whose cost is each given JSON number text, written as is."""
        w = World(self)
        tasks = {f"follow-on-{i:02d}": {} for i in range(len(costs))}
        d = w.arm("m", tasks)
        for i, text in enumerate(costs):
            path = d / f"follow-on-{i:02d}.B.{MODEL}.1.receipt.json"
            path.write_text(re.sub(r'"total_cost_usd": [^,}]+', f'"total_cost_usd": {text}', path.read_text(encoding="utf-8")), encoding="utf-8")
        return stage9.spend(w.root)

    def test_the_sum_is_exact_decimal_of_the_receipts_own_numbers(self):
        self.assertEqual(self.spend_of("0.1", "0.2"), "stage9 spend usd 0.3 missing 0")
        self.assertEqual(self.spend_of("0.1234567890123456789", "0.2"), "stage9 spend usd 0.3234567890123456789 missing 0")
        self.assertEqual(self.spend_of("1e-07", "10"), "stage9 spend usd 10.0000001 missing 0")
        self.assertEqual(self.spend_of("40", "60"), "stage9 spend usd 100 missing 0")
        self.assertEqual(self.spend_of("0.0", "null"), "stage9 spend usd 0 missing 1")

    def test_cost_reconciles_to_the_stops_receipts(self):
        w = World(self)
        tasks = [f"follow-on-0{i}" for i in range(4)]
        run = run_json("m", MODEL, tasks, max_spend_usd=0.8)
        w.arm("m", {"follow-on-00": dict(exclusion="start-error", cost=0.05), "follow-on-01": dict(exclusion="infrastructure", cost=0.3),
                    "follow-on-02": dict(cost=0.1), "follow-on-03": dict(cost=None)}, run=run)
        w.arm("m", {"follow-on-00": dict(retry=1, cost=0.2), "follow-on-02": dict(retry=1, cost=0.4)}, run=run)
        sp = w.count()["models"][MODEL]["m"]["spend"]
        self.assertEqual({k: (sp[k]["usd"], sp[k]["missing"], sp[k]["receipts"]) for k in ("terminal_graded", "terminal_excluded", "superseded", "stray", "total")},
                         {"terminal_graded": ("0.3", 1, 3), "terminal_excluded": ("0.3", 0, 1), "superseded": ("0.05", 0, 1), "stray": ("0.4", 0, 1), "total": ("1.05", 1, 6)})
        # A receipt nested below the run directory is refused, not summed.
        nested = w.root / MODEL / "m" / "sub"
        nested.mkdir()
        (nested / f"follow-on-03.B.{MODEL}.1.receipt.json").write_text(json.dumps({"task": "follow-on-03", "total_cost_usd": 5.0}), encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "below .* and its side/"):
            w.count()
        with self.assertRaisesRegex(stage9.Refusal, "below .* and its side/"):
            stage9.spend(w.root)

    def test_the_figure_is_compared_in_rounded_millionths_as_the_runner_does(self):
        for costs, reached in (((0.1, 0.7), True), ((0.7999996,), True), ((0.7999994,), False), ((0.79,), False)):
            w = World(self)
            w.arm("m", {f"follow-on-0{i}": dict(cost=c) for i, c in enumerate(costs)}, run=run_json("m", MODEL, [f"follow-on-0{i}" for i in range(len(costs))], max_spend_usd=0.8))
            sp = w.count()["models"][MODEL]["m"]["spend"]
            self.assertEqual(sp["figure_reached"], reached, costs)
            self.assertIn("reached: " + ("yes" if reached else "no"), stage9.report(w.root, w.count()))

    def test_paired_and_unpaired_spend_are_printed_under_insufficient_too(self):
        w = World(self)
        w.arm("m", {t: dict(cost=0.1) for t in CLASSES["follow-on"]})
        w.arm("n", {t: dict(cost=0.2 if i < 9 else None) if i < 10 else dict(exclusion="infrastructure") for i, t in enumerate(CLASSES["follow-on"])})
        text = stage9.report(w.root, w.count())
        lines = text.splitlines()
        head = [ln for ln in lines if "m vs n (follow-on):" in ln][0]
        self.assertIn("INSUFFICIENT", head)
        self.assertIn("spend m: paired 1 USD (0 missing), unpaired 0.2 USD (0 missing), terminal graded in follow-on 1.2 USD (0 missing)", text)
        self.assertIn("spend n: paired 1.8 USD (1 missing), unpaired 0 USD (0 missing), terminal graded in follow-on 1.8 USD (1 missing)", text)

    def test_a_cost_that_is_not_a_number_is_refused(self):
        with self.assertRaisesRegex(stage9.Refusal, "not a number"):
            self.spend_of('"0.1"')


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
    def test_m_against_a_is_read_on_every_class(self):
        w = World(self)
        w.arm("a", {**tasks_of("single-site"), **tasks_of("batch"), **tasks_of("follow-on"), **tasks_of("control")})
        w.arm("m", {**tasks_of("single-site"), **tasks_of("batch"), **tasks_of("follow-on"), **tasks_of("control")})
        w.arm("n", {**tasks_of("follow-on"), **tasks_of("control")})
        text = stage9.report(w.root, w.count())
        for want in ("m vs a (single-site):", "m vs a (batch):", "m vs a (follow-on):", "m vs a (control):", "m vs n (follow-on):", "m vs n (control):"):
            self.assertIn(want, text)
        self.assertNotIn("m vs n (single-site)", text)
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

    def test_in_n_a_log_line_that_is_not_an_event_is_unknown_not_refused(self):
        w = World(self)
        w.arm("n", {"follow-on-00": dict(logtext="not json\n")})
        got = w.attempts("n")["follow-on-00"]
        self.assertEqual((got["exposure"], got["exposed"], got["hunks"]), ("unknown", None, 0))

    def test_in_m_it_is_a_refusal_in_the_output_of_a_write(self):
        w = World(self)
        second = receipt(REFUSED_ROW, REFUSED_ROW.replace("line 12", "line 13"))
        w.arm("m", {
            "follow-on-00": dict(steps=[("mrw write plan.txt", REFUSED)]),
            "follow-on-01": dict(steps=[("mrw write plan.txt", APPLIED)]),
            "follow-on-02": dict(steps=[("mrw write --dry-run plan.txt", REFUSED), ("mrw -C d write -", second)]),
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
        self.assertEqual((got["write_calls"], got["write_results_read"], got["exposure"]), (2, 1, "unknown"))

    def test_only_one_simple_write_command_is_attributed(self):
        single = [
            "mrw write p", "command mrw write p", "env X=1 mrw write p", "./bin/mrw write p", "/usr/local/bin/mrw -C d write -",
            '"$MRW" write p', "$MRW --root d write p", "cd d && mrw write p 2>&1", "mrw write --terse - <<'PLAN'\n@@ f 1 replace\nmrw instructions > y\nPLAN",
            "mrw write - <<PLAN\nx\nPLAN\n", "cd d && mrw write - <<'EOF-1'\nbody\nEOF-1", "mrw write - <<-'E_N-D'\nx\nE_N-D",
            "mrw write - <<'PLAN' 2>&1\nx\nPLAN", "mrw write - <<'PLAN'\nPLAN", "cat plan | mrw write -", "cat a b | mrw write - 2>&1",
            "printf '%s\\n' '@@ f 3 replace' 'x' | mrw write -", "echo '@@ f 3 insert-after' | mrw write --terse -",
            "printf '%s\\n' 'if a && b > c; {' | mrw write -", "cd d && printf x | mrw write -",
            "mrw write p 'a;b|c>d'", "printf 'a|b' | printf y | mrw write -",
        ]
        unattributable = [
            "mrw write p >/dev/null 2>&1; echo Finished", "mrw write p > out.txt", "mrw write p >out.txt", "mrw write p &> out.txt",
            "mrw write p 2>/dev/null", "mrw write p >> log", "mrw write p 2>&1 | wc -l", "mrw write p | tee x", "mrw write p 3>&1",
            "mrw write p2>&1", "mrw write p < plan.txt", "mrw instructions; mrw write p", "mrw skill && mrw write p", "mrw --instructions; mrw write p",
            "false && mrw write p", "mrw write p && echo ok", "mrw write p &", "mrw write p & echo x", "mrw write p; echo done", "mrw write p || true",
            "cd d; mrw write p", "cd d && mrw read f && mrw write p",
            "mrw read x; mrw write p", "printf 'mrw write p'", 'echo "mrw write"', "'mrw' write p", "out=$(mrw write p); echo $out",
            "`mrw write p`", "(mrw write p)", "time mrw write p", "mrw write p\necho hi", "mrw write - <<'PLAN'\nx\nPLAN\nmrw read f",
            "mrw write - <<'PLAN'\nx\nPLAN\nmrw write p2\nPLAN", "mrw write - <<'PLAN' > out\nx\nPLAN", "printf x | mrw write - | wc -l",
            "printf x | tee f | mrw write -", 'printf "$(cat f)" | mrw write -', "printf x | mrw write - > out", "printf x > f | mrw write -",
            "printf x | mrw write - 2>&1 | head", "cat | mrw write -", "mrw write p | tee x", "printf x || mrw write -", "printf `cat f` | mrw write -",
        ]
        for command in single:
            self.assertEqual(stage9.classify_command(command), "single", command)
        for command in unattributable:
            self.assertEqual(stage9.classify_command(command), "unattributable", command)
        for command in ("mrw read f.go", "mrw instructions", "echo mrw writer", "mrw read f.go > out.txt", "mrw read --grep 'func write' ."):
            self.assertIsNone(stage9.classify_command(command), command)

    def test_instructions_beside_a_write_and_an_unattributable_write_are_not_read(self):
        w = World(self)
        w.arm("m", {
            "follow-on-00": dict(steps=[("mrw instructions; mrw write p >/dev/null 2>&1", "A refusal reads: " + REFUSED)]),
            "follow-on-01": dict(steps=[("mrw write p >/dev/null 2>&1; echo Finished", "Finished")]),
            "follow-on-02": dict(steps=[("command mrw write p", REFUSED), ("./bin/mrw write p", REFUSED), ("$MRW write p", APPLIED),
                                        ("mrw write p 2>&1 | head", REFUSED), ("mrw write p", ""), ("false && mrw write p", REFUSED)]),
        })
        got = w.attempts("m")
        fields = ("exposed", "hunks", "write_calls", "write_results_read", "write_unattributable", "flagged")
        self.assertEqual([tuple(got[t][f] for f in fields) for t in sorted(got)],
                         [(None, 0, 1, 0, 1, False), (None, 0, 1, 0, 1, False), (True, 2, 6, 3, 3, True)])

    def test_in_n_a_scratch_file_not_retained_is_unknown_not_zero(self):
        w = World(self)
        w.arm("n", {"follow-on-00": dict(note="scratch: not kept: the scratch directory is a link or a file, not a directory"),
                    "follow-on-01": dict(log=1), "follow-on-02": dict(note="diff: not kept: too big")})
        got = w.attempts("n")
        self.assertEqual([(got[t]["exposure_unknown"], got[t]["exposed"], got[t]["hunks"]) for t in sorted(got)],
                         [(True, None, 0), (False, True, 1), (False, False, 0)])
        text = stage9.report(w.root, w.count())
        self.assertIn("exposure n: exposed 1, not seen 1, unknown 1 (in no rate's denominator)", text)

    def test_arm_a_has_no_exposure(self):
        w = World(self)
        w.arm("a", {"single-site-00": {}})
        got = w.attempts("a")["single-site-00"]
        self.assertEqual((got["exposed"], got["hunks"]), (None, None))

    def test_the_phrase_is_the_one_internal_apply_writes(self):
        text = (ROOT / "internal" / "apply" / "apply.go").read_text(encoding="utf-8")
        self.assertIn("has no recorded showing in this version", text)
        self.assertTrue(re.search(stage9.EXPOSED.pattern, "line 12 of f.go has no recorded showing in this version"))

    def test_n_exposure_states(self):
        w = World(self)
        bad_events = {
            "extra key": {**EVENT, "x": 1}, "missing key": {k: v for k, v in EVENT.items() if k != "end"}, "empty path": {**EVENT, "path": ""},
            "path not a string": {**EVENT, "path": 3}, "op": {**EVENT, "op": "create"}, "start 0": {**EVENT, "start": 0},
            "start after first_unshown": {**EVENT, "start": 5}, "first_unshown after end": {**EVENT, "end": 3}, "end before start": {**EVENT, "start": 5, "first_unshown": 5, "end": 4},
            "a float": {**EVENT, "start": 4.0}, "a bool": {**EVENT, "start": True}, "a string": {**EVENT, "end": "4"},
            "short sha": {**EVENT, "sha_before": "e" * 63}, "upper-case sha": {**EVENT, "sha_before": "E" * 64}, "sha not hex": {**EVENT, "sha_before": "g" * 64},
        }
        specs = {
            "follow-on-00": dict(log=1),
            "follow-on-01": {},
            "follow-on-02": dict(note="scratch: not kept: it is not a regular file"),
            "follow-on-03": dict(steps=[("mrw write p", APPLIED)]),
            "follow-on-04": dict(steps=[("mrw write p", APPLIED + "\nmrw: stage9 log: open /x/stage9-exposed.jsonl: permission denied")]),
            "follow-on-05": dict(steps=[("mrw write p > out.txt", "")]),
            "follow-on-06": dict(log=1, steps=[("mrw write p", APPLIED + "\nmrw: stage9 log: no room")]),
            "follow-on-07": dict(log=1, note="scratch: x not kept"),
            "follow-on-08": dict(log=1, steps=[("mrw write p 2>&1 | head", "")]),
            "follow-on-09": dict(logtext=json.dumps(EVENT) + "\nnot an event\n"),
        }
        for i, (why, doc) in enumerate(bad_events.items()):
            specs[(CLASSES["batch"] + CLASSES["single-site"])[i]] = dict(logtext=json.dumps(doc) + "\n")
        w.arm("n", specs)
        got = w.attempts("n")
        want = {"follow-on-00": ("exposed", 1, False), "follow-on-01": ("not seen", 0, False), "follow-on-02": ("unknown", 0, False),
                "follow-on-03": ("not seen", 0, False), "follow-on-04": ("unknown", 0, False), "follow-on-05": ("unknown", 0, False),
                "follow-on-06": ("exposed", 1, True), "follow-on-07": ("exposed", 1, True), "follow-on-08": ("exposed", 1, True),
                "follow-on-09": ("exposed", 1, True)}
        for t, (state, hunks, flagged) in want.items():
            self.assertEqual((got[t]["exposure"], got[t]["hunks"], got[t]["flagged"]), (state, hunks, flagged), t)
        for i, why in enumerate(bad_events):
            self.assertEqual(got[(CLASSES["batch"] + CLASSES["single-site"])[i]]["exposure"], "unknown", why)
        self.assertEqual((got["follow-on-00"]["exposed"], got["follow-on-01"]["exposed"], got["follow-on-02"]["exposed"]), (True, False, None))

    def test_m_exposure_states(self):
        w = World(self)
        json_dry = json.dumps({"dry_run": True, "hunks": [{"status": "failed", "reason": "line 3 of f.go has no recorded showing in this version, and ..."}]})
        json_lone = json.dumps({"dry_run": False, "hunks": [{"status": "ok", "reason": ""}, {"status": "failed", "reason": "line 3 of f.go has no recorded showing in this version"}]})
        json_other = json.dumps({"dry_run": False, "hunks": [{"status": "failed", "reason": "anchor miss"}, {"status": "failed", "reason": "line 3 of f.go has no recorded showing in this version"}]})
        imitation = receipt("ok   f.go 1 replace  -1 +1", ending="applied").replace("ok   f.go 1 replace", "note: line 3 of f.go has no recorded showing in this version\nok   f.go 1 replace")
        w.arm("m", {
            "follow-on-00": dict(steps=[("mrw write p", REFUSED)]),
            "follow-on-01": dict(steps=[("mrw write --json p", json_dry), ("mrw write --json q", json_lone), ("mrw write --json r", json_other)]),
            "follow-on-02": dict(steps=[("mrw write p", imitation)]),
            "follow-on-03": dict(steps=[("mrw write p", "[output truncated]")]),
            "follow-on-04": dict(steps=[("mrw write p", REFUSED + "\n[output truncated]")]),
            "follow-on-05": dict(steps=[("mrw write p", APPLIED)]),
            "follow-on-06": dict(steps=[("mrw write p", REFUSED.rsplit("\n", 1)[0])]),
            "follow-on-07": dict(steps=[("mrw write --dry-run p", REFUSED), ("mrw write p", receipt(REFUSED_ROW, OTHER_ROW)), ("mrw write p", REFUSED)]),
            "follow-on-08": dict(steps=[("mrw write p", REFUSED + "\nmrw: the plan from standard input is kept at /x/y.plan"), ("mrw write p 2>&1 | tee x", REFUSED)]),
            "follow-on-09": dict(steps=[("mrw write p", REFUSED.replace("\n", "\n[output truncated]\n", 1))]),
        })
        got = w.attempts("m")
        self.assertEqual((got["follow-on-00"]["exposure"], got["follow-on-00"]["hunks"], got["follow-on-00"]["refusals"]), ("exposed", 1, {"dry_run": 0, "lone": 1, "beside": 0}), "a real text refusal")
        self.assertEqual((got["follow-on-01"]["exposure"], got["follow-on-01"]["refusals"]), ("exposed", {"dry_run": 1, "lone": 1, "beside": 1}), "the JSON receipts")
        self.assertEqual((got["follow-on-02"]["exposure"], got["follow-on-02"]["hunks"]), ("not seen", 0), "a quoted imitation is not a refusal")
        self.assertEqual((got["follow-on-03"]["exposure"], got["follow-on-03"]["hunks"], got["follow-on-03"]["write_unattributable"]), ("unknown", 0, 1), "a truncation-only result")
        self.assertEqual((got["follow-on-04"]["exposure"], got["follow-on-04"]["hunks"], got["follow-on-04"]["refusals"]["lone"]), ("unknown", 0, 0), "a clipped result is never a lone failure")
        self.assertEqual(got["follow-on-05"]["exposure"], "not seen", "a complete result with no refusal")
        self.assertEqual(got["follow-on-06"]["exposure"], "unknown", "a result that does not end in its summary line is unattributable")
        self.assertEqual((got["follow-on-07"]["exposure"], got["follow-on-07"]["hunks"], got["follow-on-07"]["refusals"]), ("exposed", 3, {"dry_run": 1, "lone": 1, "beside": 1}), "the three kinds")
        self.assertEqual((got["follow-on-08"]["exposure"], got["follow-on-08"]["hunks"], got["follow-on-08"]["flagged"]), ("exposed", 1, True), "stderr lines after the summary are apart; a pipe is unattributable")
        self.assertEqual((got["follow-on-09"]["exposure"], got["follow-on-09"]["hunks"]), ("unknown", 0), "a truncation marker inside a receipt that still ends in its summary")

    def test_mention_sees_flags_before_write(self):
        for command in ("mrw --dry-run write p", "mrw --terse write -", "mrw -C d --dry-run write p", "mrw --root=d --terse write p", "$MRW --dry-run write p",
                        '"$MRW" --terse write p', "./bin/mrw --force write p", "mrw -C d --root e write p"):
            self.assertTrue(stage9.MENTION.search(command), command)
            self.assertEqual(stage9.classify_command(command), "single", command)
        for command in ("mrw read --grep write f.go", "mrw instructions", "mrw --help", "echo write"):
            self.assertIsNone(stage9.classify_command(command), command)
        w = World(self)
        w.arm("m", {"follow-on-00": dict(steps=[("mrw --dry-run write p", REFUSED)])})
        got = w.attempts("m")["follow-on-00"]
        self.assertEqual((got["exposure"], got["hunks"], got["write_calls"]), ("exposed", 1, 1))

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

    def test_count_requires_the_registered_plans(self):
        def plans(model=MODEL, tasks=2):
            return [{"model": model, "arm": a, "directory": f"{model}/{a}", "configs": ["A" if a == "a" else "B"], "max_spend_usd": FIGURE[a], "tasks": CLASSES["follow-on"][:tasks]}
                    for a in ("n", "m", "a")]

        def world(**over):
            w = World(self)
            for a in ("n", "m", "a"):
                w.arm(a, {t: {} for t in CLASSES["follow-on"][:2]}, run=run_json(a, MODEL, CLASSES["follow-on"][:2], **over.get(a, {})))
            return w

        w = world()
        self.assertEqual(sorted(w.count(plans=plans())["models"][MODEL]), ["a", "m", "n"])
        w = world()
        w.arm("m", {t: {} for t in CLASSES["follow-on"][:2]}, model="claude-extra", run=run_json("m", "claude-extra", CLASSES["follow-on"][:2]))
        with self.assertRaisesRegex(stage9.Refusal, "claude-extra/m, which no registered plan names"):
            w.count(plans=plans())
        w = world()
        with self.assertRaisesRegex(stage9.Refusal, f"{SONNET}/m, which .* does not hold"):
            w.count(plans=plans() + [{**plans(SONNET)[1]}])
        w = world(m={"max_spend_usd": 15.0})
        with self.assertRaisesRegex(stage9.Refusal, "max_spend_usd is 15.0, the registered plan's is 14.0"):
            w.count(plans=plans())
        w = world()
        with self.assertRaisesRegex(stage9.Refusal, "tasks are not the registered plan's"):
            w.count(plans=[*plans()[:1], {**plans()[1], "tasks": CLASSES["follow-on"][:3]}, plans()[2]])
        # The smoke's plans, a probe included, pass the smoke's directories, and the probe is checked but not counted.
        w = world()
        w.arm("a", {t: {} for t in CLASSES["follow-on"][:2]}, run=run_json("a", MODEL, CLASSES["follow-on"][:2], max_spend_usd=0.01))
        probe = Path(w.root) / MODEL / "a-probe"
        probe.mkdir()
        (probe / "run.json").write_text(json.dumps(run_json("a", MODEL, CLASSES["follow-on"][:2], max_spend_usd=0.01)), encoding="utf-8")
        smoke = [*plans()[:2], {**plans()[2], "max_spend_usd": 0.01}, {**plans()[2], "directory": f"{MODEL}/a-probe", "max_spend_usd": 0.01, "probe": True}]
        w.arm("m", {t: {} for t in CLASSES["follow-on"][:2]}, run=run_json("m", MODEL, CLASSES["follow-on"][:2], max_spend_usd=14.0))
        counted = w.count(plans=smoke)
        self.assertEqual(sorted(counted["models"][MODEL]), ["a", "m", "n"])

    def test_count_refuses_swapped_transcripts(self):
        w = World(self)
        for arm in ("m", "n"):
            w.arm(arm, {t: dict(steps=[(f"echo {arm}", arm)]) for t in CLASSES["follow-on"][:2]})
        self.assertEqual(sorted(w.count()["models"][MODEL]), ["m", "n"])
        name = f"follow-on-00.B.{MODEL}.1.transcript.jsonl"
        m_path, n_path = w.root / MODEL / "m" / name, w.root / MODEL / "n" / name
        m_text, n_text = m_path.read_text(encoding="utf-8"), n_path.read_text(encoding="utf-8")
        self.assertNotEqual(m_text, n_text)
        m_path.write_text(n_text, encoding="utf-8")
        n_path.write_text(m_text, encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "its transcript hashes to"):
            w.count()
        m_path.write_text(m_text, encoding="utf-8")
        n_path.write_text(n_text, encoding="utf-8")
        w = World(self)
        w.arm("m", {"follow-on-00": dict(drop=["transcript_sha256"])})
        with self.assertRaisesRegex(stage9.Refusal, "no transcript_sha256"):
            w.count()

    def test_count_binds_each_receipt_to_its_runs_build(self):
        w = World(self)
        for arm in ("m", "n"):
            w.arm(arm, {t: {} for t in CLASSES["follow-on"][:2]})
        self.assertEqual(sorted(w.count()["models"][MODEL]), ["m", "n"])

        def receipt_path(arm):
            return w.root / MODEL / arm / f"follow-on-00.B.{MODEL}.1.receipt.json"

        # a swapped pair: M's receipt of the task is N's, and N's is M's, each with its transcript
        docs = {arm: json.loads(receipt_path(arm).read_text(encoding="utf-8")) for arm in ("m", "n")}
        for arm, other in (("m", "n"), ("n", "m")):
            receipt_path(arm).write_text(json.dumps(docs[other]), encoding="utf-8")
        with self.assertRaisesRegex(stage9.Refusal, "its receipt's mrw_sha256 is"):
            w.count()
        for arm in ("m", "n"):
            receipt_path(arm).write_text(json.dumps(docs[arm]), encoding="utf-8")
        w.count()
        # a build the run never had, and none at all
        for change in ({"mrw_sha256": "f" * 64}, {"mrw_sha256": ""}):
            receipt_path("m").write_text(json.dumps({**docs["m"], **change}), encoding="utf-8")
            with self.assertRaisesRegex(stage9.Refusal, "its receipt's mrw_sha256 is"):
                w.count()

    def test_count_refuses_an_unobserved_init_model(self):
        for spec in (dict(drop=["init_model"]), dict(init_model="another-model")):
            w = World(self)
            w.arm("m", {"follow-on-00": spec})
            with self.assertRaisesRegex(stage9.Refusal, "init_model"):
                w.count()
        w = World(self)
        w.arm("m", {"follow-on-00": dict(exclusion="infrastructure", drop=["init_model"])})
        self.assertEqual(w.attempts("m")["follow-on-00"]["outcome"], "missing")

    def test_instruction_hash_is_absent_in_a_and_shared_by_m_and_n(self):
        def world(**over):
            w = World(self)
            w.arm("a", tasks_of("single-site", 2), run=run_json("a", MODEL, CLASSES["single-site"][:2], **over.get("a", {})))
            w.arm("m", tasks_of("follow-on", 2), run=run_json("m", MODEL, CLASSES["follow-on"][:2], **over.get("m", {})))
            w.arm("n", tasks_of("follow-on", 2), run=run_json("n", MODEL, CLASSES["follow-on"][:2], **over.get("n", {})))
            return w

        self.assertEqual(sorted(world().count()["models"][MODEL]), ["a", "m", "n"])
        with self.assertRaisesRegex(stage9.Refusal, "mrw_instructions_sha256"):
            world(n={"mrw_instructions_sha256": "9" * 64}).count()
        with self.assertRaisesRegex(stage9.Refusal, "arm a carries mrw_instructions_sha256"):
            world(a={"mrw_instructions_sha256": I_HASH}).count()
        with self.assertRaisesRegex(stage9.Refusal, "mrw_instructions_sha256"):
            world(m={"mrw_instructions_sha256": "9" * 64}, n={"mrw_instructions_sha256": "9" * 64}).count()
        m = world(m={"mrw_instructions_sha256": None})
        with self.assertRaisesRegex(stage9.Refusal, "mrw_instructions_sha256"):
            m.count()

    def test_a_changed_mrw_is_missing_and_listed(self):
        w = World(self)
        w.arm("m", {"follow-on-00": dict(after="f" * 64, cost=0.3), "follow-on-01": dict(cost=0.1)})
        got = w.attempts("m")
        self.assertEqual((got["follow-on-00"]["outcome"], got["follow-on-00"]["mrw_after_differs"]), ("missing", True))
        self.assertEqual(got["follow-on-01"]["outcome"], "correct")
        text = stage9.report(w.root, w.count())
        self.assertIn(f"follow-on-00.B.{MODEL}.1", [ln for ln in text.splitlines() if "bin/mrw changed during the attempt" in ln][0])
        arm = w.count()["models"][MODEL]["m"]["spend"]
        self.assertEqual((arm["terminal_excluded"]["usd"], arm["terminal_graded"]["usd"]), ("0.3", "0.1"), "its spend stays in the reconciliation")

    def test_the_cli_exits_2_with_the_reason(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"))
        w.arm("n", tasks_of("follow-on"), run=run_json("n", MODEL, CLASSES["follow-on"], mrw_sha256=M_HASH))
        code, out, err = cli("count", "--receipts", str(w.root), "--transcripts", str(w.root), "--classes", str(w.classes), "--pilot", str(w.pilot), "--plans", str(w.plans()))
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

    def test_a_repeated_key_line_is_refused_valid_or_not(self):
        w = World(self)
        w.arm("m", tasks_of("follow-on"))
        for text, reason in ((f"m sha256: zzz\nm sha256: {M_HASH}\nn sha256: {N_HASH}\n", "`m sha256:` 2 times"),
                             (f"m sha256: {M_HASH}\nn sha256: {N_HASH}\nn sha256: nope\n", "`n sha256:` 2 times"),
                             (f"m sha256: nope\nn sha256: {N_HASH}\n", "not `m sha256: `")):
            w.pilot.write_text(text, encoding="utf-8")
            with self.assertRaisesRegex(stage9.Refusal, reason):
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
        self.assertIn("m: planned 12 started 12 graded 12 missing 0; correct 12 outside 0 other 0; claimed 12; attempts with slipped writes 0", text)
        self.assertIn("n: planned 12 started 12 graded 12 missing 0; correct 11 outside 1 other 0; claimed 12; attempts with slipped writes 0", text)
        self.assertIn("exposure m: exposed 2, not seen 10, unknown 0 (in no rate's denominator); flagged 0 (hunks a lower bound); hunks 2; "
                      "refusals on a dry run 0, the plan's only failed hunk 2, beside another failed hunk 0; write results read 2 of 2 write calls (0 unattributable)", text)
        self.assertIn("exposure n: exposed 2, not seen 10, unknown 0 (in no rate's denominator); flagged 0 (hunks a lower bound); hunks 3", text)
        self.assertIn("unattributable in m:", text)
        self.assertIn("not a measured zero", text)
        self.assertIn("One repeat leaves each task one observation per arm", text)
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
        code, out, _ = cli("count", "--receipts", str(w.root), "--transcripts", str(w.root), "--classes", str(w.classes), "--pilot", str(w.pilot), "--plans", str(w.plans()))
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

    def test_an_arm_or_model_on_one_side_only_is_refused(self):
        w = self.build()
        counts = w.count()
        del counts["models"][MODEL]["n"]
        with self.assertRaisesRegex(stage9.Refusal, "n only in the receipts"):
            stage9.report(w.root, counts)
        counts = w.count()
        counts["models"]["claude-sonnet-5-5"] = counts["models"][MODEL]
        with self.assertRaisesRegex(stage9.Refusal, "claude-sonnet-5-5/m only in the counts"):
            stage9.report(w.root, counts)
        counts = w.count()
        w.arm("a", tasks_of("single-site"))
        with self.assertRaisesRegex(stage9.Refusal, "a only in the receipts"):
            stage9.report(w.root, counts)


# ---- ADR-211 T3: the gates, the review and the registration -----------------------------------------------------

T0 = "2026-11-20T10:00:00Z"
CANARY_OPUS = "claude-opus-5-5"


def mark_started(directory, when_text=T0, stems=None):
    """The .started marks the runner writes beside each receipt, and the recorded exit status of the step."""
    for path in sorted(Path(directory).rglob("*.receipt.json")):
        stem = path.name[:-len(".receipt.json")]
        if stems is None or stem in stems:
            (path.parent / f"{stem}.started").write_text(when_text + "\n", encoding="utf-8")
    Path(str(directory) + ".exit").write_text("0\n", encoding="utf-8")


def canary_doc(model=CANARY_OPUS, arm="m", **over):
    doc = {"pass": True, "cli_version": "2.9.9", "model": model, "started": T0, "problems": {}, "total_cost_usd": {"A": 0.1, "B": 0.2},
           "init_models": {"A": INIT, "B": INIT}, "max_spend_usd": stage9.CANARY_FIGURES[model],
           "mrw_sha256": N_HASH if arm == "n" else M_HASH, "mrw_instructions_sha256": I_HASH}
    doc.update(over)
    return doc


class Gate(unittest.TestCase):
    def world(self, tasks=2, **over):
        w = World(self)
        names = CLASSES["follow-on"][:tasks]
        d = w.arm("m", {t: {} for t in names}, run=run_json("m", MODEL, names, **over.pop("run", {})))
        w.gates = Path(w.tmp.name) / "gates.txt"
        return w, d

    def gate(self, w, d, clear_pred="2026-11-20T09:00:00Z", **kw):
        """Gate the m run after its predecessor in the pilot order, n, whose clearance the gates file holds unless clear_pred is None."""
        kw.setdefault("after", f"{MODEL} n")
        if clear_pred and not w.gates.exists():
            w.gates.write_text(f"gate {MODEL} n cleared {clear_pred}\n", encoding="utf-8")
        return stage9.gate("run", MODEL, "m", d, w.plans(kw.pop("plans", None)), w.pilot, w.gates, "pilot", **kw)

    def test_the_gate_passes_a_cleared_step_and_refuses_each_broken_one(self):
        w, d = self.world()
        mark_started(d)
        self.assertEqual(self.gate(w, d, now="2026-11-20T11:00:00Z"), f"gate {MODEL} m cleared 2026-11-20T11:00:00Z")

        def refused(why, mutate, regex, **kw):
            w, d = self.world(**kw.pop("world", {}))
            mark_started(d)
            mutate(w, d)
            with self.assertRaisesRegex(stage9.Refusal, regex, msg=why):
                self.gate(w, d, **kw)

        def receipt_of(d, task):
            return d / f"{task}.B.{MODEL}.1.receipt.json"

        def edit(path, **change):
            doc = json.loads(path.read_text(encoding="utf-8"))
            doc.update(change)
            path.write_text(json.dumps(doc), encoding="utf-8")

        t0, t1 = CLASSES["follow-on"][:2]
        refused("an exit 3 on a complete run whose last attempt crossed its figure", lambda w, d: (Path(str(d) + ".exit").write_text("3\n"), edit(receipt_of(d, t1), total_cost_usd=14.0)), "exited '3'")
        refused("no recorded exit", lambda w, d: Path(str(d) + ".exit").unlink(), "no recorded exit")
        refused("a plan that is not the registered one", lambda w, d: None, "max_spend_usd is 14.0, the registered plan's is 7.0",
                plans=[{"model": MODEL, "arm": "m", "directory": f"{MODEL}/m", "configs": ["B"], "max_spend_usd": 7.0, "tasks": CLASSES["follow-on"][:2]}])
        refused("a .started with no receipt", lambda w, d: (d / f"{t1}.B.{MODEL}.1.r1.started").write_text(T0), "started and never given a receipt")
        refused("a receipt with no .started", lambda w, d: (d / f"{t0}.B.{MODEL}.1.started").unlink(), "no .started")
        refused("a stray receipt", lambda w, d: ((d / f"{t0}.B.{MODEL}.1.r1.receipt.json").write_text(json.dumps({"task": t0, "attempt": f"{t0}.B.{MODEL}.1.r1"})), (d / f"{t0}.B.{MODEL}.1.r1.started").write_text(T0)),
                "reached by no planned attempt")
        refused("a missing owed replacement", lambda w, d: edit(receipt_of(d, t0), exclusion="start-error"), "owed and missing")
        refused("a kept diff that is not its receipt's", lambda w, d: ((d / f"{t0}.B.{MODEL}.1.diff").write_text("x"), edit(receipt_of(d, t0), kept_diff_sha256="0" * 64)), "not the one its receipt hashes")
        refused("a kept file its receipt does not name", lambda w, d: (d / f"{t0}.B.{MODEL}.1.kept").write_text("x"), "a .kept its receipt does not name")
        refused("spend at the figure", lambda w, d: (edit(receipt_of(d, t0), total_cost_usd=7.0), edit(receipt_of(d, t1), total_cost_usd=7.0)), "reached it|the figure", world={"run": {"max_spend_usd": 14.0}})
        refused("known spend a millionth under the figure that rounds to it", lambda w, d: (edit(receipt_of(d, t0), total_cost_usd=0.7999996), edit(receipt_of(d, t1), total_cost_usd=0.0)), "the figure", world={"run": {"max_spend_usd": 0.8}})
        t2 = CLASSES["follow-on"][2]
        refused("three missing costs", lambda w, d: [edit(receipt_of(d, t), total_cost_usd=None) for t in (t0, t1, t2)], "3 receipts carry no cost", world={"tasks": 3})
        refused("the earliest .started before the predecessor's clearance", lambda w, d: w.gates.write_text(f"gate {MODEL} n cleared 2026-11-20T12:00:00Z\n"), "before", after=f"{MODEL} n")
        refused("a predecessor that never cleared", lambda w, d: None, "no recorded clearance", clear_pred=None, after=f"{MODEL} n")
        refused("a later step that names no predecessor", lambda w, d: None, "its predecessor in the pilot order is claude-opus-5-5 n, not none", after=None)
        refused("a later step that names another predecessor", lambda w, d: None, "is claude-opus-5-5 n, not claude-opus-5-5 a", after=f"{MODEL} a")
        refused("a clearance recorded before the step started", lambda w, d: w.gates.write_text(f"gate {MODEL} n cleared 2026-11-20T09:00:00Z\ngate {MODEL} m cleared 2026-11-20T09:30:00Z\n"), "before the step started")
        refused("a receipt swapped with another task's", lambda w, d: (lambda a, b: (receipt_of(d, t0).write_text(b), receipt_of(d, t1).write_text(a)))(receipt_of(d, t0).read_text(), receipt_of(d, t1).read_text()),
                "not the receipt of this attempt")
        refused("a receipt of another model", lambda w, d: edit(receipt_of(d, t0), model="another"), "its model is 'another'")
        refused("a receipt of another run's build", lambda w, d: edit(receipt_of(d, t0), mrw_sha256="f" * 64), "its mrw_sha256")
        refused("a graded receipt whose init_model is not the observed one", lambda w, d: edit(receipt_of(d, t0), init_model="other"), "init_model")
        refused("a receipt nested below the run directory", lambda w, d: ((d / "sub").mkdir(), (d / "sub" / "z.receipt.json").write_text("{}")), "below")
        # A replayed step is judged by the clearance already recorded, never the time of the replay.
        w, d = self.world()
        mark_started(d)
        w.gates.write_text(f"gate {MODEL} n cleared 2026-11-20T09:00:00Z\ngate {MODEL} m cleared 2026-11-20T10:30:00Z\n")
        self.assertEqual(self.gate(w, d, now="2027-01-01T00:00:00Z"), f"gate {MODEL} m cleared 2026-11-20T10:30:00Z")
        w.gates.write_text(f"gate {MODEL} n cleared 2026-11-20T09:00:00Z\n")
        self.assertIn("cleared", self.gate(w, d, after=f"{MODEL} n", now="2026-11-20T10:30:00Z"))

    def test_a_canary_is_gated_by_its_figure_its_costs_and_the_model_it_reported(self):
        tmp = Path(tempfile.mkdtemp())
        self.addCleanup(lambda: __import__("shutil").rmtree(tmp, ignore_errors=True))
        pilot = tmp / "pilot.md"
        pilot.write_text(f"init model {CANARY_OPUS}: {INIT}\nm sha256: {M_HASH}\nn sha256: {N_HASH}\ninstructions sha256: {I_HASH}\n", encoding="utf-8")
        gates = tmp / "gates.txt"

        gates.write_text(f"gate {CANARY_OPUS} canary-n cleared 2026-11-20T09:30:00Z\n", encoding="utf-8")

        def run(m_doc, n_doc=None, after=f"{CANARY_OPUS} canary-n", **kw):
            for arm, doc in (("m", m_doc), ("n", n_doc)):
                if doc is not None:
                    (tmp / f"canary-{CANARY_OPUS}-{arm}.json").write_text(json.dumps(doc), encoding="utf-8")
                    (tmp / f"canary-{CANARY_OPUS}-{arm}.exit").write_text("0\n", encoding="utf-8")
            return stage9.gate("canary", CANARY_OPUS, "m", tmp / f"canary-{CANARY_OPUS}-m.json", None, pilot, gates, "smoke", after=after, now="2026-11-20T11:00:00Z")

        self.assertEqual(run(canary_doc()), f"gate {CANARY_OPUS} canary-m cleared 2026-11-20T11:00:00Z")
        for why, doc, regex in (("a canary that did not pass", canary_doc(**{"pass": False}), "did not pass"),
                                ("a null cost", canary_doc(total_cost_usd={"A": None, "B": 0.1}), "null cost"),
                                ("no init_models", canary_doc(init_models=None), "no init_models"),
                                ("no max_spend_usd", {k: v for k, v in canary_doc().items() if k != "max_spend_usd"}, "max_spend_usd"),
                                ("another figure", canary_doc(max_spend_usd=0.5), "max_spend_usd"),
                                ("costs at the figure", canary_doc(total_cost_usd={"A": 0.4, "B": 0.35}), "reached its figure"),
                                ("a cost for A only", canary_doc(total_cost_usd={"A": 0.1}), "no cost for configuration B"),
                                ("an init model for A only", canary_doc(init_models={"A": INIT}), "no init model for configuration B"),
                                ("another observed model", canary_doc(init_models={"A": "another", "B": "another"}), "observed")):
            with self.assertRaisesRegex(stage9.Refusal, regex, msg=why):
                run(doc)
        with self.assertRaisesRegex(stage9.Refusal, "different models"):
            run(canary_doc(), canary_doc(arm="n", init_models={"A": INIT, "B": "x"}))
        (tmp / f"canary-{CANARY_OPUS}-n.json").unlink()
        gates.write_text(f"gate {CANARY_OPUS} canary-n cleared 2026-11-20T12:00:00Z\n")
        with self.assertRaisesRegex(stage9.Refusal, "before"):
            run(canary_doc(), after=f"{CANARY_OPUS} canary-n")


class Review(unittest.TestCase):
    T = "follow-on-00"
    STEM = f"{T}.B.{MODEL}.1"

    def world(self):
        w = World(self)
        w.arm("m", {self.T: dict(steps=[("mrw write plan.txt", REFUSED)], solved=False), "follow-on-01": {}})
        n = w.arm("n", {self.T: dict(log=1, outside=["f.go"]), "follow-on-01": {}})
        (n / f"{self.STEM}.diff").write_text("-old\n+new\n", encoding="utf-8")
        return w

    def rows(self, w, **change):
        """RESULTS.md as a producer writes it from the receipts; change maps a (arm) to a replacement block."""
        blocks = {
            "m": f"### {MODEL} m {self.STEM}\nadjudication: unassessable\nreason: none\nlimits: one attempt, one read\ndiff: unassessable: no kept diff\n",
            "n": f"### {MODEL} n {self.STEM}\nadjudication: an edit to a line the task did not ask for\nreason: outside: f.go\nlimits: one attempt, one read\ndiff:\n```\n-old\n+new\n```\n",
        }
        blocks.update(change)
        return "# RESULTS\n\n" + "\n".join(blocks.values())

    def review(self, w, text):
        results = Path(w.tmp.name) / "RESULTS.md"
        results.write_text(text, encoding="utf-8")
        return stage9.review(results, w.count(), w.root)

    def test_the_review_rows_cover_every_exposed_attempt(self):
        w = self.world()
        self.assertEqual(self.review(w, self.rows(w)), "review 2 row(s) for 2 exposed attempt(s)")

        def refused(regex, text):
            with self.assertRaisesRegex(stage9.Refusal, regex):
                self.review(w, text)

        good = self.rows(w)
        refused("no row for the exposed attempt", self.rows(w).split("### ")[0] + "### " + good.split("### ")[2])
        refused("no row for the exposed attempt", "# RESULTS\n\n### " + good.split("### ")[1])
        refused("not an exposed attempt", good + f"\n### {MODEL} n follow-on-01.B.{MODEL}.1\nadjudication: no such edit\nreason: none\nlimits: x\ndiff: unassessable: none\n")
        # a row keyed by attempt alone, where M and N share the stem
        refused("no row for the exposed attempt|is none of|Refusal", good.replace(f"### {MODEL} m ", "### ").replace(f"### {MODEL} n ", "### "))
        refused("adjudication", good.replace("adjudication: no such edit", "adjudication: x").replace("adjudication: an edit to a line the task did not ask for", "adjudication: the agent meant well"))
        refused("adjudication", good.replace("adjudication: unassessable\n", ""))
        refused("reason", good.replace("reason: outside: f.go", "reason: none"))
        refused("limits", good.replace("limits: one attempt, one read\ndiff:\n```", "limits: \ndiff:\n```"))
        refused("byte for byte", good.replace("+new", "+newer"))
        refused("unassessable", good.replace("diff: unassessable: no kept diff", "diff: x"))
        # a row of another attempt's diff: the M row quoting N's diff
        refused("diff", good.replace("diff: unassessable: no kept diff", "diff:\n```\n-old\n+new\n```"))
        # byte for byte: a blank line or a trailing space the row adds is not the kept diff
        refused("byte for byte", good.replace("+new\n```", "+new\n\n```"))
        refused("byte for byte", good.replace("+new\n```", "+new \n```"))
        # an empty RESULTS.md, and counts of nothing against run directories that hold exposed attempts
        refused("is empty", "")
        refused("is empty", "\n  \n")
        results = Path(w.tmp.name) / "RESULTS.md"
        results.write_text(good, encoding="utf-8")
        for why, counts in (("no models", {"models": {}}), ("a model with no arms", {"models": {MODEL: {}}}), ("no n arm", {"models": {MODEL: {"m": w.count()["models"][MODEL]["m"]}}})):
            with self.assertRaisesRegex(stage9.Refusal, "counts.json holds no", msg=why):
                stage9.review(results, counts, w.root)


class Registration(unittest.TestCase):
    ENTRY = ("## Pre-registered: cost per correctly completed edit task (Stage 9)\n\nOne gate, M against A, per direction: on single-site or on batch, on both models,\n"
             "the interval of the spend-per-correct ratio excludes 1 and lies on the same side of 1. On opposite sides\n"
             "the models are reported as disagreeing and nothing is registered.\n\n")

    def check(self, text):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "BACKLOG.md"
            path.write_text(text, encoding="utf-8")
            return stage9.registration(path)

    def test_the_registration_check_reads_only_the_stage9_entry(self):
        self.assertIn("none withdrawn", self.check("## Earlier\n\nx\n\n" + self.ENTRY + "## From ADR-206\n\nthe interval excludes 1 for M against N\n"))
        for why, text in (("the gate removed", self.ENTRY.replace("on the same side of 1", "elsewhere")),
                          ("M against N restored", self.ENTRY + "or the interval excludes 1 for M against N.\n"),
                          ("M against N restored across a wrapped line", self.ENTRY + "or the interval excludes 1 for M\nagainst N.\n"),
                          ("the exposed-and-outside gate restored", self.ENTRY + "or an exposed attempt ended `outside` and its diff shows an edit.\n"),
                          ("the floor restored", self.ENTRY + "If fewer than 3 of the 12 follow-on tasks have an exposed attempt, the hazard was not provoked.\n"),
                          ):
            with self.assertRaisesRegex(stage9.Refusal, "withdrawn|holds no gate", msg=why):
                self.check(text)
        with self.assertRaisesRegex(stage9.Refusal, "holds no gate"):  # `same side of 1` only in another entry
            self.check(self.ENTRY.replace("on the same side of 1", "elsewhere") + "## From ADR-211\n\nthe same side of 1 on the same side of 1\n")
        for why, text in (("M against A not named", self.ENTRY.replace("M against A", "M against B")),
                          ("a single direction", self.ENTRY.replace("on single-site or on batch", "on single-site")),
                          ("one model only", self.ENTRY.replace("on both models", "on one model")),
                          ("the interval not required to exclude 1", self.ENTRY.replace("excludes 1 and lies", "lies")),
                          ("disagreement registering something", self.ENTRY.replace("nothing is registered", "a stage is registered")),
                          ("an entry with only its heading", "## Pre-registered: cost per correctly completed edit task (Stage 9)\n\non the same side of 1\n")):
            with self.assertRaisesRegex(stage9.Refusal, "lacks", msg=why):
                self.check(text)
        with self.assertRaisesRegex(stage9.Refusal, "no Stage 9 entry"):
            self.check("## Other\n\nx\n")


class Project(unittest.TestCase):
    def test_the_smoke_projects_each_arm_under_its_figure(self):
        w = World(self)
        w.arm("m", {t: dict(cost=0.30) for t in CLASSES["follow-on"][:2]})
        def plan(figure, n=24):
            return [{"model": MODEL, "arm": "m", "directory": f"{MODEL}/m", "configs": ["B"], "max_spend_usd": figure, "tasks": CLASSES["follow-on"][:2] * (n // 2)},
                    {"model": MODEL, "arm": "a", "directory": f"{MODEL}/a-probe", "configs": ["A"], "max_spend_usd": 0.01, "tasks": ["x"], "probe": True}]

        lines, over = stage9.project(w.root, w.plans(plan(14.0)))
        self.assertEqual((lines, over), ([f"project {MODEL} m 7.2000 of 14.0"], False))
        # 24 attempts at 0.30 is 7.20: a figure of 7.2 is reached, as the runner's `>=` reaches it
        lines, over = stage9.project(w.root, w.plans(plan(7.2)))
        self.assertTrue(over and "REACHES THE FIGURE" in lines[0])
        code, out, _ = cli("project", "--receipts", str(w.root), "--plans", str(w.plans(plan(7.0))))
        self.assertEqual(code, 1)
        w.arm("m", {t: dict(cost=None) for t in CLASSES["follow-on"][:2]})
        with self.assertRaisesRegex(stage9.Refusal, "no cost"):
            stage9.project(w.root, w.plans(plan(14.0)))
        with self.assertRaisesRegex(stage9.Refusal, "no smoke root"):
            stage9.project(w.root / "absent", w.plans(plan(14.0)))
        with self.assertRaisesRegex(stage9.Refusal, "no registered run to project"):
            stage9.project(w.root, w.plans([plan(14.0)[1]]))  # a plan of the probe alone projects nothing, and says so


FENCE = re.compile(r"^## Acceptance\n\n```bash\n(.*?)\n```", re.S | re.M)
POPUS, PSONNET = MODEL, SONNET
STEPS = [(POPUS, "n"), (POPUS, "m"), (POPUS, "a"), (PSONNET, "n"), (PSONNET, "m"), (PSONNET, "a")]
STOPS = {(POPUS, "a"): 14, (POPUS, "m"): 14, (POPUS, "n"): 7, (PSONNET, "a"): 6, (PSONNET, "m"): 6, (PSONNET, "n"): 3}
SMOKE_TASKS = [CLASSES["follow-on"][0], CLASSES["control"][0]]
HOUR = "2026-11-20T{:02d}:{:02d}:00Z"


class FinalReview(unittest.TestCase):
    """The final review of #751: what the gate and count must refuse or accept that the first pass did not."""

    REAL = json.loads((ROOT / "bench" / "stage-9" / "plans.json").read_text(encoding="utf-8"))

    def n_world(self, tasks, **run):
        w = World(self)
        d = w.arm("n", {t: {} for t in tasks}, run=run_json("n", MODEL, sorted(tasks), **run))
        w.gates = Path(w.tmp.name) / "gates.txt"
        return w, d

    def gate_n(self, w, d, plan):
        return stage9.gate("run", MODEL, "n", d, w.plans([plan]), w.pilot, w.gates, "pilot", after=None, now="2026-11-20T11:00:00Z")

    def test_a_run_is_the_registered_tasks_in_the_order_bench_run_sorts_them(self):
        plan = next(p for p in self.REAL if p["directory"] == f"{MODEL}/n")
        self.assertNotEqual(plan["tasks"], sorted(plan["tasks"]), "the registered order is not the sorted one")
        w, d = self.n_world(plan["tasks"])
        mark_started(d)
        self.assertIn("cleared", self.gate_n(w, d, plan))
        counts = stage9.count(w.root, w.root, ROOT / "bench" / "stage-9" / "classes.json", w.pilot, w.plans([plan]))
        self.assertEqual(len(counts["models"][MODEL]["n"]["tasks"]), len(plan["tasks"]))
        # but the same tasks, not others, and none twice
        for why, tasks in (("a task missing", sorted(plan["tasks"])[1:]), ("a task twice", sorted(plan["tasks"]) + [plan["tasks"][0]]),
                           ("another task", sorted(plan["tasks"])[1:] + ["x-other"])):
            w, d = self.n_world(tasks)
            mark_started(d)
            with self.assertRaises(stage9.Refusal, msg=why):
                self.gate_n(w, d, plan)

    def test_an_n_run_is_the_n_build_with_its_scratch_file_kept(self):
        tasks = CLASSES["follow-on"][:2]
        plan = {"model": MODEL, "arm": "n", "directory": f"{MODEL}/n", "configs": ["B"], "max_spend_usd": 7.0, "tasks": tasks}
        w, d = self.n_world(tasks)
        mark_started(d)
        self.assertIn("cleared", self.gate_n(w, d, plan))
        for why, run, regex in (("N run with the M binary", {"mrw_sha256": M_HASH}, "n sha256"),
                                ("no keep_scratch_file", {"keep_scratch_file": None}, "keep_scratch_file"),
                                ("another instruction file", {"mrw_instructions_sha256": "e" * 64}, "instructions sha256"),
                                ("a keep_state_file beside it", {"keep_state_file": "mrw/x"}, "keep_state_file")):
            w, d = self.n_world(tasks, **run)
            mark_started(d)
            for p in d.glob("*.receipt.json"):  # the receipts carry the run's hash, so the receipt-to-run check holds
                doc = json.loads(p.read_text(encoding="utf-8"))
                doc["mrw_sha256"] = run.get("mrw_sha256", doc.get("mrw_sha256", ""))
                p.write_text(json.dumps(doc), encoding="utf-8")
            with self.assertRaisesRegex(stage9.Refusal, regex, msg=why):
                self.gate_n(w, d, plan)

    def test_a_canary_carries_its_arms_build_and_the_instructions(self):
        tmp = Path(tempfile.mkdtemp())
        self.addCleanup(lambda: __import__("shutil").rmtree(tmp, ignore_errors=True))
        pilot = tmp / "pilot.md"
        pilot.write_text(f"init model {CANARY_OPUS}: {INIT}\nm sha256: {M_HASH}\nn sha256: {N_HASH}\ninstructions sha256: {I_HASH}\n", encoding="utf-8")
        gates = tmp / "gates.txt"

        def gate(arm, doc):
            (tmp / f"canary-{CANARY_OPUS}-{arm}.json").write_text(json.dumps(doc), encoding="utf-8")
            (tmp / f"canary-{CANARY_OPUS}-{arm}.exit").write_text("0\n", encoding="utf-8")
            return stage9.gate("canary", CANARY_OPUS, arm, tmp / f"canary-{CANARY_OPUS}-{arm}.json", None, pilot, gates, "smoke",
                               after=None if arm == "n" else f"{CANARY_OPUS} canary-n", now="2026-11-20T11:00:00Z")

        self.assertIn("cleared", gate("n", canary_doc(arm="n")))
        gates.write_text(f"gate {CANARY_OPUS} canary-n cleared 2026-11-20T09:30:00Z\n", encoding="utf-8")
        self.assertIn("cleared", gate("m", canary_doc(arm="m")))
        with self.assertRaisesRegex(stage9.Refusal, "n sha256"):
            gate("n", canary_doc(arm="n", mrw_sha256=M_HASH))
        with self.assertRaisesRegex(stage9.Refusal, "m sha256"):
            gate("m", canary_doc(arm="m", mrw_sha256=N_HASH))
        with self.assertRaisesRegex(stage9.Refusal, "instructions sha256"):
            gate("m", canary_doc(arm="m", mrw_instructions_sha256="e" * 64))

    def test_a_receipt_moved_to_another_attempts_filenames_is_refused_by_count_and_the_gate(self):
        tasks = CLASSES["follow-on"][:2]
        plan = {"model": MODEL, "arm": "m", "directory": f"{MODEL}/m", "configs": ["B"], "max_spend_usd": 14.0, "tasks": tasks}
        for key, value in (("repeat", 2), ("attempt", f"{tasks[0]}.B.{MODEL}.2")):
            w = World(self)
            d = w.arm("m", {t: {} for t in tasks}, run=run_json("m", MODEL, tasks))
            w.gates = Path(w.tmp.name) / "gates.txt"
            mark_started(d)
            path = d / f"{tasks[0]}.B.{MODEL}.1.receipt.json"
            doc = json.loads(path.read_text(encoding="utf-8"))
            doc[key] = value
            path.write_text(json.dumps(doc), encoding="utf-8")
            with self.assertRaisesRegex(stage9.Refusal, key, msg=f"count, {key}"):
                w.count(plans=[plan])
            with self.assertRaisesRegex(stage9.Refusal, key, msg=f"gate, {key}"):
                stage9.gate("run", MODEL, "m", d, w.plans([plan]), w.pilot, w.gates, "pilot", after=f"{MODEL} n", now="2026-11-20T11:00:00Z")

    def test_a_kept_log_the_receipt_names_and_the_directory_lost_is_not_not_seen(self):
        w = World(self)
        w.arm("n", {"follow-on-00": dict(log=1), "follow-on-01": {}})
        path = next(w.root.glob(f"*/n/follow-on-01.B.{MODEL}.1.receipt.json"))
        doc = json.loads(path.read_text(encoding="utf-8"))
        doc["kept_scratch_file_sha256"] = "a" * 64  # the receipt says a log was captured; there is none beside it
        path.write_text(json.dumps(doc), encoding="utf-8")
        got = w.attempts("n")
        self.assertEqual((got["follow-on-01"]["exposure"], got["follow-on-01"]["exposed"]), ("unknown", None))

    def test_a_write_with_a_quoted_flag_value_is_a_write_and_a_heredoc_flag_is_text(self):
        spaced = 'mrw --root "/tmp/my repo" write plan.txt'
        self.assertEqual(stage9.classify_command(spaced), "single")
        self.assertEqual(stage9.classify_command("mrw -C '/tmp/my repo' write -"), "single")
        self.assertEqual(stage9.classify_command("mrw --root=/tmp/a write plan.txt"), "single")
        self.assertEqual(stage9.classify_command('mrw --root "/tmp/a;b" write plan.txt; rm x'), "unattributable")
        w = World(self)
        w.arm("m", {"follow-on-00": dict(steps=[(spaced, REFUSED)])})
        got = w.attempts("m")["follow-on-00"]
        self.assertEqual((got["write_calls"], got["write_results_read"], got["exposed"]), (1, 1, True))
        heredoc = "mrw write - <<'PLAN'\n@@ f.go 3 replace\n  --dry-run is a flag of mrw write\nPLAN"
        self.assertEqual(stage9.refusals_of(heredoc, REFUSED)[1], "lone")
        self.assertEqual(stage9.refusals_of("mrw write --dry-run plan.txt", REFUSED)[1], "dry_run")
        self.assertEqual(stage9.refusals_of("mrw --root '/tmp/my repo' write --dry-run -", REFUSED)[1], "dry_run")
        self.assertEqual(stage9.refusals_of("printf '%s' '--dry-run' | mrw write -", REFUSED)[1], "lone")


def sha_of(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


class Fixture:
    """A repository root the paid tasks' fences can run in, with a Stage 9 directory the way the producers leave it: the
    receipts the way World writes them (the way the runner does), their .started and .exit files, the gates the gate verb
    printed, the canaries, the rehearsal and RESULTS.md. The real script, helper and Go packages are the ones the fences run."""

    def __init__(self, test, projection_over=False):
        self.test = test
        self.tmp = Path(tempfile.mkdtemp())
        test.addCleanup(lambda: __import__("shutil").rmtree(self.tmp, ignore_errors=True))
        self.root = self.tmp / "fx"
        self.R = self.root / "docs" / "bench" / "stage-9"
        self.b = self.root / "bench" / "stage-9"
        (self.root / "scripts").mkdir(parents=True)
        self.b.mkdir(parents=True)
        self.R.mkdir(parents=True)
        for name in ("bench_stage9.py", "bench_mechanism.py"):
            (self.root / "scripts" / name).write_bytes((ROOT / "scripts" / name).read_bytes())
        for name in ("cmd", "internal", "go.mod", "go.sum"):
            os.symlink(ROOT / name, self.root / name)
        everything = [t for c in CLASSES.values() for t in c]
        two = CLASSES["follow-on"] + CLASSES["control"]
        (self.b / "classes.json").write_text(json.dumps(CLASSES), encoding="utf-8")
        (self.b / "instructions.txt").write_text("instructions\n", encoding="utf-8")
        (self.b / "builds.md").write_text(f"m sha256: {M_HASH}\nn sha256: {N_HASH}\n", encoding="utf-8")
        self.plans = [{"model": m, "arm": a, "directory": f"{m}/{a}", "configs": ["A" if a == "a" else "B"], "max_spend_usd": STOPS[m, a],
                       "tasks": two if a == "n" else everything} for m, a in STEPS]
        smoke_figure = {POPUS: 0.8, PSONNET: 0.4}
        self.smoke = [{"model": m, "arm": a, "directory": f"{m}/{a}", "configs": ["A" if a == "a" else "B"], "max_spend_usd": smoke_figure[m], "tasks": SMOKE_TASKS} for m, a in STEPS]
        self.smoke.append({"model": PSONNET, "arm": "a", "directory": f"{PSONNET}/a-probe", "configs": ["A"], "max_spend_usd": 0.01, "tasks": SMOKE_TASKS, "probe": True})
        (self.b / "plans.json").write_text(json.dumps(self.plans), encoding="utf-8")
        (self.b / "smoke-plans.json").write_text(json.dumps(self.smoke), encoding="utf-8")
        self.pilot()
        self.canaries()
        self.smoke_runs(projection_over)
        self.rehearsal()
        self.pilot_runs()

    def pilot(self):
        keys = {"tag": "v1.0.0", "claude code": "2.9.9", "bench revision": "rev1", "m sha256": M_HASH, "n sha256": N_HASH,
                "instructions sha256": sha_of(self.b / "instructions.txt"), "classes sha256": sha_of(self.b / "classes.json"),
                "script sha256": sha_of(self.root / "scripts" / "bench_stage9.py"), "helper sha256": sha_of(self.root / "scripts" / "bench_mechanism.py"),
                "status line sha256": "c" * 64, "plans sha256": sha_of(self.b / "plans.json"), "smoke plans sha256": sha_of(self.b / "smoke-plans.json"),
                "tasks sha256": "e" * 64, "budget seconds": "1200", "max turns": "60", "seed": "206", "smoke transcripts": str(self.R / "smoke"),
                "rehearsal": "docs/bench/stage-9/rehearsal.md", f"init model {POPUS}": INIT, f"init model {PSONNET}": INIT}
        keys.update({f"stop {m} {a}": str(STOPS[m, a]) for m, a in STEPS})
        (self.R / "pilot.md").write_text("".join(f"{k}: {v}\n" for k, v in keys.items()), encoding="utf-8")

    def rj(self, arm, model, tasks, figure):
        """run.json of a fixture run: the frozen instructions are the committed file's."""
        over = {} if arm == "a" else {"mrw_instructions_sha256": sha_of(self.b / "instructions.txt")}
        return run_json(arm, model, tasks, max_spend_usd=figure, **over)

    def rehearsal(self):
        v = dict(line.split(": ", 1) for line in (self.R / "pilot.md").read_text(encoding="utf-8").splitlines())
        text = "date: 2026-11-20\nrevision: rev1\n" + "".join(f"{k}: {v[k]}\n" for k in ("plans sha256", "smoke plans sha256", "tasks sha256", "instructions sha256"))
        text += "".join(f"check {c}: exit 0\n" for c in ("instructions frozen", "build test", "oracle test", "bench build", "stub canaries", "stub plans", "count", "n event counted", "paid fences"))
        text += "".join(f"check {c} negative: exit 1\n" for c in ("n event counted", "stub plans", "build test"))
        (self.R / "rehearsal.md").write_text(text, encoding="utf-8")

    def gate(self, kind, m, a, target, plans, gates, sequence, after, now):
        line = stage9.gate(kind, m, a, target, plans, self.R / "pilot.md", gates, sequence, after=after, now=now)
        with open(gates, "a", encoding="utf-8") as fh:
            fh.write(line + "\n")

    def canaries(self):
        gates = self.R / "smoke" / "gates.txt"
        gates.parent.mkdir(parents=True, exist_ok=True)
        order = [(POPUS, "n"), (POPUS, "m"), (PSONNET, "n"), (PSONNET, "m")]
        for k, (m, a) in enumerate(order):
            doc = canary_doc(m, started=HOUR.format(10 + k // 3, (20 * k) % 60), mrw_sha256=N_HASH if a == "n" else M_HASH,
                             mrw_instructions_sha256=sha_of(self.b / "instructions.txt"), total_cost_usd={"A": 0.05, "B": 0.05})
            (self.R / f"canary-{m}-{a}.json").write_text(json.dumps(doc), encoding="utf-8")
            (self.R / f"canary-{m}-{a}.exit").write_text("0\n", encoding="utf-8")
        prev = None
        for k, (m, a) in enumerate(order):
            self.gate("canary", m, a, self.R / f"canary-{m}-{a}.json", None, gates, "smoke", prev, HOUR.format(10 + k // 3, (20 * k + 10) % 60))
            prev = f"{m} canary-{a}"
        self.prev_after_canaries = prev

    def edit(self, path, **change):
        doc = json.loads(Path(path).read_text(encoding="utf-8"))
        doc.update(change)
        Path(path).write_text(json.dumps(doc), encoding="utf-8")

    def finish(self, d, started, exit_status="0"):
        """The files the runner leaves beside a receipt: its kept diff, the hash of each, .started and the recorded exit."""
        for r in sorted(Path(d).glob("*.receipt.json")):
            stem = r.name[:-len(".receipt.json")]
            rec = json.loads(r.read_text(encoding="utf-8"))
            (Path(d) / f"{stem}.diff").write_text(f"diff {stem}\n", encoding="utf-8")
            rec["kept_diff_sha256"] = sha_of(Path(d) / f"{stem}.diff")
            rec["mrw_sha256"] = json.loads((Path(d) / "run.json").read_text(encoding="utf-8")).get("mrw_sha256", "")
            if (Path(d) / f"{stem}.kept").exists():
                rec["kept_scratch_file_sha256"] = sha_of(Path(d) / f"{stem}.kept")
            r.write_text(json.dumps(rec), encoding="utf-8")
            (Path(d) / f"{stem}.started").write_text(started + "\n", encoding="utf-8")
        Path(str(d) + ".exit").write_text(exit_status + "\n", encoding="utf-8")

    def smoke_runs(self, projection_over):
        w = World(self.test)
        w.root = self.R / "smoke"
        gates = self.R / "smoke" / "gates.txt"
        probe = w.arm("a", {SMOKE_TASKS[0]: dict(cost=0.02)}, model=PSONNET, run=self.rj("a", PSONNET, SMOKE_TASKS, 0.01))
        probe = probe.rename(probe.parent / "a-probe")
        self.finish(probe, HOUR.format(9, 0), "3")
        Path(str(probe) + ".stderr").write_text("bench: recorded spend has reached --max-spend\n", encoding="utf-8")
        prev = self.prev_after_canaries
        figure = {POPUS: 0.8, PSONNET: 0.4}
        for k, (m, a) in enumerate(STEPS):
            specs = {SMOKE_TASKS[0]: dict(cost=0.10, log=1 if a == "n" else 0), SMOKE_TASKS[1]: dict(cost=0.05)}
            if projection_over and (m, a) == (POPUS, "n"):
                # Under its smoke figure ($0.80), so the smoke's own plan clears it; at the registered pilot's 24 attempts
                # it is $8.40 against $7: only a projection against the pilot's plan sees it.
                specs = {SMOKE_TASKS[0]: dict(cost=0.35), SMOKE_TASKS[1]: dict(cost=0.35)}
            d = w.arm(a, specs, model=m, run=self.rj(a, m, SMOKE_TASKS, figure[m]))
            start = HOUR.format(14 + k, 0)
            self.finish(d, start)
            self.gate("run", m, a, d, self.b / "smoke-plans.json", gates, "smoke", prev, HOUR.format(14 + k, 30))
            prev = f"{m} {a}"

    def pilot_runs(self):
        w = World(self.test)
        w.root = self.R / "pilot"
        gates = self.R / "pilot" / "gates.txt"
        gates.parent.mkdir(parents=True, exist_ok=True)
        prev = None
        for k, (m, a) in enumerate(STEPS):
            tasks = CLASSES["follow-on"] + CLASSES["control"] if a == "n" else [t for c in CLASSES.values() for t in c]
            specs = {t: dict(cost=0.05) for t in tasks}
            if a == "n":  # both models' N runs hold an exposed attempt, so a row can be taken from the wrong one
                specs[tasks[0]] = dict(cost=0.05, log=1, outside=["f.go"])
            d = w.arm(a, specs, model=m, run=self.rj(a, m, tasks, STOPS[m, a]))
            self.finish(d, HOUR.format(15 + k, 0))
            self.gate("run", m, a, d, self.b / "plans.json", gates, "pilot", prev, HOUR.format(15 + k, 30))
            prev = f"{m} {a}"
        pilot = self.R / "pilot.md"
        code, counts, err = cli("count", "--receipts", str(self.R / "pilot"), "--transcripts", str(self.R / "pilot"), "--pilot", str(pilot),
                                "--classes", str(self.b / "classes.json"), "--plans", str(self.b / "plans.json"))
        self.test.assertEqual(code, 0, err)
        (self.R / "counts.json").write_text(counts, encoding="utf-8")
        code, report, err = cli("report", "--receipts", str(self.R / "pilot"), "--counts", str(self.R / "counts.json"))
        self.test.assertEqual(code, 0, err)
        (self.R / "report.txt").write_text(report, encoding="utf-8")
        doc = json.loads(counts)
        rows = []
        for model, arms in doc["models"].items():
            for arm in ("m", "n"):
                for entry in arms[arm]["tasks"].values():
                    if entry["exposure"] == "exposed":
                        stem = entry["attempt"]
                        diff = (self.R / "pilot" / model / arm / f"{stem}.diff").read_text(encoding="utf-8").rstrip("\n")
                        rows.append(f"### {model} {arm} {stem}\nadjudication: no such edit\nreason: outside: f.go\nlimits: one attempt\ndiff:\n```\n{diff}\n```\n")
        (self.R / "RESULTS.md").write_text(f"# RESULTS\n\ntranscripts: {self.R / 'pilot'}\n\n" + "\n".join(rows), encoding="utf-8")

    def run(self, task):
        text = (ROOT / "docs" / "adr" / "ADR-206" / "tasks" / task).read_text(encoding="utf-8")
        fence = FENCE.search(text).group(1)
        env = dict(os.environ, STAGE9_DIR=str(self.R), TMPDIR=str(self.tmp))
        done = subprocess.run(["bash", "-c", fence], cwd=self.root, env=env, capture_output=True, text=True)
        return done.returncode, done.stdout + done.stderr


class PaidFences(unittest.TestCase):
    def test_the_paid_fences_pass_a_fixture_and_fail_each_broken_one(self):
        fx = Fixture(self)
        code, out = fx.run("T5-canaries-and-smoke.md")
        self.assertEqual(code, 0, out)
        code, out = fx.run("T6-the-counted-pilot.md")
        self.assertEqual(code, 0, out)
        cases = []

        def t5(why):
            def deco(fn):
                cases.append((why, "T5-canaries-and-smoke.md", fn, {}))
                return fn
            return deco

        def t6(why):
            def deco(fn):
                cases.append((why, "T6-the-counted-pilot.md", fn, {}))
                return fn
            return deco

        S, P = lambda fx: fx.R / "smoke", lambda fx: fx.R / "pilot"
        # T5
        @t5("a rehearsal of another revision")
        def _(fx): (fx.R / "rehearsal.md").write_text((fx.R / "rehearsal.md").read_text().replace("revision: rev1", "revision: rev2"))
        @t5("a rehearsal of another day")
        def _(fx): (fx.R / "rehearsal.md").write_text((fx.R / "rehearsal.md").read_text().replace("2026-11-20", "2026-11-19"))
        @t5("a rehearsal that is a trial run's record")
        def _(fx): (fx.R / "rehearsal.md").write_text((fx.R / "rehearsal.md").read_text() + "subset: 1 task(s) of each class of each plan: a trial, not the rehearsal\n")
        @t5("a rehearsal with a failed check")
        def _(fx): (fx.R / "rehearsal.md").write_text((fx.R / "rehearsal.md").read_text().replace("check count: exit 0", "check count: exit 1"))
        @t5("a rehearsal whose negative did not fail")
        def _(fx): (fx.R / "rehearsal.md").write_text((fx.R / "rehearsal.md").read_text().replace("check n event counted negative: exit 1", "check n event counted negative: exit 0"))
        @t5("a probe whose exit was not 3")
        def _(fx): Path(str(S(fx) / PSONNET / "a-probe") + ".exit").write_text("2\n")
        @t5("a probe with no stop line")
        def _(fx): Path(str(S(fx) / PSONNET / "a-probe") + ".stderr").write_text("bench: failed\n")
        @t5("an N smoke run without keep_scratch_file")
        def _(fx): fx.edit(S(fx) / POPUS / "n" / "run.json", keep_scratch_file=None)
        @t5("a canary of another observed model")
        def _(fx): fx.edit(fx.R / f"canary-{POPUS}-m.json", init_models={"A": "another", "B": "another"})
        @t5("a canary that does not report init_models")
        def _(fx): fx.edit(fx.R / f"canary-{POPUS}-n.json", init_models=None)
        @t5("a smoke run recorded as not exiting 0")
        def _(fx): Path(str(S(fx) / POPUS / "m") + ".exit").write_text("3\n")
        @t5("a step whose start precedes its predecessor's clearance")
        def _(fx): [p.write_text("2026-11-20T01:00:00Z\n") for p in (S(fx) / POPUS / "m").glob("*.started")]
        @t5("a clearance missing from the gates")
        def _(fx): (S(fx) / "gates.txt").write_text("".join(line for line in (S(fx) / "gates.txt").read_text().splitlines(True) if f"{POPUS} a " not in line))
        @t5("plans that are not the pinned ones")
        def _(fx): (fx.b / "smoke-plans.json").write_text((fx.b / "smoke-plans.json").read_text() + " ")
        # T6
        @t6("an N run without keep_scratch_file")
        def _(fx): fx.edit(P(fx) / POPUS / "n" / "run.json", keep_scratch_file=None)
        @t6("a run whose exit was 3")
        def _(fx): Path(str(P(fx) / PSONNET / "m") + ".exit").write_text("3\n")
        @t6("a step started before its predecessor cleared")
        def _(fx): [p.write_text("2026-11-20T01:00:00Z\n") for p in (P(fx) / POPUS / "a").glob("*.started")]
        @t6("a stray receipt")
        def _(fx): (P(fx) / POPUS / "m" / "zz.receipt.json").write_text("{}")
        @t6("an exposed attempt with no review row")
        def _(fx): (fx.R / "RESULTS.md").write_text("# RESULTS\n\ntranscripts: " + str(P(fx)) + "\n")
        @t6("a review row quoting another attempt's diff")
        def _(fx):
            text = (fx.R / "RESULTS.md").read_text()
            first, second = re.findall(r"^diff:\n```\n(.*?)\n```$", text, re.S | re.M)[:2]
            (fx.R / "RESULTS.md").write_text(text.replace(first, "\0").replace(second, first).replace("\0", second))
        @t6("a review row that alters its own diff")
        def _(fx): (fx.R / "RESULTS.md").write_text((fx.R / "RESULTS.md").read_text().replace("diff ", "diff x"))
        @t6("known spend at the figure")
        def _(fx): [fx.edit(p, total_cost_usd=0.5) for p in (P(fx) / PSONNET / "n").glob("*.receipt.json")]
        @t6("a graded receipt whose init_model is not the observed one")
        def _(fx): fx.edit(next((P(fx) / POPUS / "a").glob("*.receipt.json")), init_model="x")

        for why, task, mutate, _ in cases:
            fx2 = Fixture(self)
            mutate(fx2)
            code, out = fx2.run(task)
            self.assertNotEqual(code, 0, f"{task} stayed green on: {why}")

    def test_the_smoke_projection_over_a_figure_fails_the_smoke_fence(self):
        fx = Fixture(self, projection_over=True)
        code, out = fx.run("T5-canaries-and-smoke.md")
        self.assertNotEqual(code, 0, out)
        self.assertIn("REACHES THE FIGURE", out)


class RehearsalFence(unittest.TestCase):
    """ADR-211 T2's fence, extracted from its task file and run over a generated repository: a complete rehearsal record passes
    and each way of breaking it fails. The fence's own first clause runs this module, so the inner run answers through a
    python3 shim that prints the line the clause greps for."""

    FILES = ("bench/stage-9/instructions.txt", "bench/stage-9/plans.json", "bench/stage-9/smoke-plans.json", "bench/stage-9/classes.json",
             "scripts/bench_stage9.py", "scripts/bench_mechanism.py", "bench/stage-9/rehearse.sh", "bench/stage-9/stub-agent")
    CHECKS = ("instructions frozen", "build test", "oracle test", "bench build", "stub canaries", "stub plans", "count", "n event counted", "paid fences")
    NEGATIVES = ("n event counted", "stub plans", "build test")
    ALIASES = (("plans", "bench/stage-9/plans.json"), ("smoke plans", "bench/stage-9/smoke-plans.json"), ("instructions", "bench/stage-9/instructions.txt"))
    IDS = ("s9-one", "s9-two", "s9-three")
    TASK = ROOT / "docs" / "adr" / "ADR-211" / "tasks" / "T2-the-free-rehearsal.md"

    def git(self, *argv):
        env = dict(os.environ, GIT_CONFIG_GLOBAL="/dev/null", GIT_CONFIG_SYSTEM="/dev/null")
        subprocess.run(["git", "-c", "user.name=t", "-c", "user.email=t@t", *argv], cwd=self.repo, env=env, check=True, capture_output=True)

    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.addCleanup(lambda: __import__("shutil").rmtree(self.tmp, ignore_errors=True))
        self.repo = self.tmp / "repo"
        for name in self.FILES:
            path = self.repo / name
            path.parent.mkdir(parents=True, exist_ok=True)
            real = ROOT / name
            path.write_bytes(real.read_bytes() if name.endswith(("rehearse.sh", "stub-agent", ".py")) else f"{name}\n".encode())
        (self.repo / "bench/stage-9/classes.json").write_text(json.dumps({"single-site": [self.IDS[0]], "follow-on": list(self.IDS[1:])}), encoding="utf-8")
        (self.repo / "bench/stage-9/builds.md").write_text(f"m sha256: {M_HASH}\nn sha256: {N_HASH}\n", encoding="utf-8")
        for i in self.IDS:
            (self.repo / "bench/tasks" / i).mkdir(parents=True)
            (self.repo / "bench/tasks" / i / "task.json").write_text(f'{{"id": "{i}"}}\n', encoding="utf-8")
            (self.repo / "bench/tasks" / i / "reference.patch").write_text(f"patch {i}\n", encoding="utf-8")
        self.git("init", "-q")
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "inputs")
        self.rev = subprocess.run(["git", "rev-parse", "HEAD"], cwd=self.repo, capture_output=True, text=True, check=True).stdout.strip()
        self.git("update-ref", "refs/remotes/origin/main", self.rev)
        shim = self.tmp / "shim"
        shim.mkdir()
        (shim / "python3").write_text(
            f'#!/bin/sh\nif [ "$1 $2 $3" = "-m unittest -v" ]; then\n  echo "test_the_rehearsal_fence_refuses_a_missing_or_wrong_digest (scripts.test_bench_stage9.RehearsalFence) ... ok"\n  exit 0\nfi\n'
            f'exec "{sys.executable}" "$@"\n', encoding="utf-8")
        (shim / "python3").chmod(0o755)
        self.shim = shim
        self.lines = self.complete()

    def sha(self, name):
        return sha_of(self.repo / name)

    def tasks_digest(self, ids=None):
        if ids is None:
            ids = subprocess.run(["jq", "-r", "[.[] | .[]] | .[]", "bench/stage-9/classes.json"], cwd=self.repo, capture_output=True, text=True, check=True).stdout.split()
        body = b"".join((self.repo / "bench/tasks" / i / n).read_bytes() for i in ids for n in ("task.json", "reference.patch"))
        return hashlib.sha256(body).hexdigest()

    def complete(self, ids=None):
        lines = ["date: 2026-11-20", f"revision: {self.rev}"]
        lines += [f"{name} sha256: {self.sha(name)}" for name in self.FILES]
        lines += [f"m sha256: {M_HASH}", f"n sha256: {N_HASH}", f"tasks sha256: {self.tasks_digest(ids)}"]
        lines += [f"{k} sha256: {self.sha(f)}" for k, f in self.ALIASES]
        lines += [f"check {c}: exit 0" for c in self.CHECKS] + [f"check {c} negative: exit 1" for c in self.NEGATIVES]
        return lines

    def run_fence(self, lines=None, record=True, name="rehearsal.md"):
        record_path = self.tmp / name
        if record:
            record_path.write_text("\n".join(self.lines if lines is None else lines) + "\n", encoding="utf-8")
        elif record_path.exists():
            record_path.unlink()
        fence = FENCE.search(self.TASK.read_text(encoding="utf-8")).group(1)
        env = dict(os.environ, REHEARSAL=str(record_path), PATH=f"{self.shim}:{os.environ['PATH']}")
        done = subprocess.run(["bash", "-c", fence], cwd=self.repo, env=env, capture_output=True, text=True)
        return done.returncode, done.stdout + done.stderr

    def test_the_rehearsal_fence_refuses_a_missing_or_wrong_digest(self):
        code, out = self.run_fence()
        self.assertEqual(code, 0, out)
        code, _ = self.run_fence(record=False)
        self.assertNotEqual(code, 0, "no record at all")
        broken = []
        for key in ("m sha256", "n sha256", "tasks sha256"):
            line = next(ln for ln in self.lines if ln.startswith(key + ": "))
            broken += [(f"{key} missing", [ln for ln in self.lines if ln != line]),
                       (f"{key} twice", [*self.lines, line]),
                       (f"{key} wrong", [ln.replace(line, key + ": " + "0" * 64) for ln in self.lines])]
        for name in self.FILES:
            line = next(ln for ln in self.lines if ln.startswith(name + " sha256: "))
            broken += [(f"{name} digest missing", [ln for ln in self.lines if ln != line]),
                       (f"{name} digest twice", [*self.lines, line]),
                       (f"{name} digest wrong", [ln.replace(line, name + " sha256: " + "0" * 64) for ln in self.lines])]
        for c in self.CHECKS:
            line = f"check {c}: exit 0"
            broken += [(f"check {c} missing", [ln for ln in self.lines if ln != line]), (f"check {c} failed", [ln.replace(line, f"check {c}: exit 1") for ln in self.lines])]
        for c in self.NEGATIVES:
            line = f"check {c} negative: exit 1"
            broken += [(f"negative {c} recorded as exit 0", [ln.replace(line, f"check {c} negative: exit 0") for ln in self.lines]),
                       (f"negative {c} missing", [ln for ln in self.lines if ln != line])]
        broken += [("no date", [ln for ln in self.lines if not ln.startswith("date: ")]),
                   ("a revision no commit has", [ln.replace(self.rev, "1" * 40) for ln in self.lines]),
                   ("a thirteenth check", [*self.lines, "check extra: exit 0"]),
                   ("a trial run's record", [*self.lines, "subset: 1 task(s) of each class of each plan: a trial, not the rehearsal"])]
        for why, lines in broken:
            code, out = self.run_fence(lines)
            self.assertNotEqual(code, 0, f"the fence stayed green on: {why}\n{out}")

    def test_the_rehearsal_fence_goes_red_when_an_input_changes_after_the_rehearsal(self):
        for name in self.FILES:
            with self.subTest(name):
                path = self.repo / name
                before = path.read_bytes()
                path.write_bytes(before + b"\n")
                try:
                    code, out = self.run_fence()
                    self.assertNotEqual(code, 0, out)
                finally:
                    path.write_bytes(before)
    def test_the_rehearsal_fence_reads_the_working_tree_and_the_alias_digests(self):
        # an input changed and not committed, its digest brought up to date: only the working tree says so
        stub = self.repo / "bench/stage-9/stub-agent"
        before = stub.read_bytes()
        stub.write_bytes(before + b"\n")
        code, out = self.run_fence(self.complete())
        stub.write_bytes(before)
        self.assertNotEqual(code, 0, out)
        extra = self.repo / "bench/stage-9/extra.txt"  # an untracked file under the inputs
        extra.write_text("x\n", encoding="utf-8")
        code, out = self.run_fence()
        extra.unlink()
        self.assertNotEqual(code, 0, out)
        for k, f in self.ALIASES:
            line = f"{k} sha256: {self.sha(f)}"
            for why, lines in ((f"{k} missing", [ln for ln in self.lines if ln != line]), (f"{k} twice", [*self.lines, line]),
                               (f"{k} wrong", [ln.replace(line, f"{k} sha256: " + "0" * 64) for ln in self.lines])):
                code, out = self.run_fence(lines)
                self.assertNotEqual(code, 0, f"the fence stayed green on: {why}\n{out}")

    def test_the_rehearsal_fence_takes_a_record_path_with_a_space(self):
        code, out = self.run_fence(name="a record.md")
        self.assertEqual(code, 0, out)

    def test_the_rehearsal_fence_refuses_a_task_that_is_not_there(self):
        # cat fails on the missing id and the hash of what it did print is the old one: the failure must not hide in the pipeline
        classes = self.repo / "bench/stage-9/classes.json"
        classes.write_text(json.dumps({"single-site": ["s9-ghost", self.IDS[0]], "follow-on": list(self.IDS[1:])}), encoding="utf-8")
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "a task that is not there")
        self.rev = subprocess.run(["git", "rev-parse", "HEAD"], cwd=self.repo, capture_output=True, text=True, check=True).stdout.strip()
        self.git("update-ref", "refs/remotes/origin/main", self.rev)
        lines = [ln for ln in self.complete(self.IDS)]
        code, out = self.run_fence(lines)
        self.assertNotEqual(code, 0, out)



    def commit_change(self, name, body):
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(body)
        self.git("add", "-A")
        self.git("commit", "-q", "-m", f"change {name}")

    def test_the_rehearsal_fence_reads_the_commits_after_the_revision(self):
        # a commit outside what the rehearsal ran leaves the record good; one to an input, with its digest brought up to date, does not
        self.commit_change("docs/other.md", b"prose\n")
        self.assertEqual(self.run_fence(self.complete())[0], 0)
        stub = self.repo / "bench/stage-9/stub-agent"
        self.commit_change("bench/stage-9/stub-agent", stub.read_bytes() + b"\n")
        code, out = self.run_fence(self.complete())
        self.assertNotEqual(code, 0, out)

    def test_the_rehearsal_fence_reads_the_engine_against_origin_main(self):
        self.commit_change("internal/plan/plan_test.go", b"package plan\n")
        self.assertEqual(self.run_fence()[0], 0, "a test file of an engine package is not an engine change")
        self.commit_change("internal/plan/plan.go", b"package plan\n")
        code, out = self.run_fence()
        self.assertNotEqual(code, 0, out)

class RehearseScript(unittest.TestCase):
    """What rehearse.sh refuses before it starts anything: each in a throwaway repository holding only the script and the stub."""

    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.addCleanup(lambda: __import__("shutil").rmtree(self.tmp, ignore_errors=True))
        self.repo = self.tmp / "repo"
        (self.repo / "bench/stage-9").mkdir(parents=True)
        (self.tmp / "t").mkdir()
        for name in ("rehearse.sh", "stub-agent"):
            (self.repo / "bench/stage-9" / name).write_bytes((ROOT / "bench/stage-9" / name).read_bytes())
            (self.repo / "bench/stage-9" / name).chmod(0o755)
        (self.repo / "bench/stage-9/plans.json").write_text("[]\n", encoding="utf-8")
        (self.repo / "bench/stage-9/classes.json").write_text("{}\n", encoding="utf-8")
        for argv in (["init", "-q"], ["add", "-A"], ["commit", "-q", "-m", "x"]):
            subprocess.run(["git", "-c", "user.name=t", "-c", "user.email=t@t", *argv], cwd=self.repo, check=True, capture_output=True,
                           env=dict(os.environ, GIT_CONFIG_GLOBAL="/dev/null", GIT_CONFIG_SYSTEM="/dev/null"))

    def run_script(self, **env):
        done = subprocess.run(["bash", "bench/stage-9/rehearse.sh", str(self.tmp / "out.md")], cwd=self.repo, capture_output=True, text=True,
                              env=dict(os.environ, TMPDIR=str(self.tmp / "t"), **env))
        return done.returncode, done.stderr

    def test_a_subset_is_a_positive_integer_with_no_leading_zero(self):
        for bad in ("0", "01", "x", "1.5", "-1", " 1"):
            code, err = self.run_script(REHEARSE_SUBSET=bad)
            self.assertEqual(code, 2, bad)
            self.assertIn("positive integer", err, bad)

    def test_a_relative_scratch_is_resolved_before_it_is_used(self):
        (self.repo / "scratch").mkdir()
        (self.repo / "scratch/old").write_text("x\n", encoding="utf-8")
        code, err = self.run_script(REHEARSE_SUBSET="1", REHEARSE_SCRATCH="scratch")
        self.assertEqual(code, 2, err)
        self.assertIn(str((self.repo / "scratch").resolve()), err)

    def test_a_stub_that_differs_from_head_is_refused_even_in_a_trial(self):
        stub = self.repo / "bench/stage-9/stub-agent"
        stub.write_bytes(stub.read_bytes() + b"# changed\n")
        code, err = self.run_script(REHEARSE_SUBSET="1")
        self.assertEqual(code, 2, err)
        self.assertIn("uncommitted change", err)
        stub.unlink()
        code, err = self.run_script(REHEARSE_SUBSET="1")
        self.assertEqual(code, 2, err)
        self.assertIn("uncommitted change", err)  # a deleted stub differs from HEAD too

if __name__ == "__main__":
    unittest.main()
