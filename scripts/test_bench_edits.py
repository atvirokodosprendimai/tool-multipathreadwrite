"""Tests for bench_edits.py: ADR-193 Decisions 4 and 5, on synthetic transcripts and receipts."""
import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import bench_edits as edits  # noqa: E402
from bench_mechanism import mrw_calls  # noqa: E402

APPLIED = "ok   f.go 1 replace  -1 +1\nwrote f.go\n1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied"
SR_DOC = "f.go\n<<<<<<< SEARCH\nold\nkeep\n=======\nnew\nkeep\n>>>>>>> REPLACE"


def transcript(*steps, result=True):
    """A stream-json transcript: steps are (command, result text)."""
    rows = [{"type": "system", "subtype": "init"}]
    for i, (command, text) in enumerate(steps):
        rows.append({"type": "assistant", "message": {"content": [{"type": "tool_use", "id": str(i), "name": "Bash", "input": {"command": command}}]}})
        rows.append({"type": "user", "message": {"content": [{"type": "tool_result", "tool_use_id": str(i), "content": [{"type": "text", "text": text}]}]}})
    if result:
        rows.append({"type": "result", "subtype": "success"})
    return "\n".join(json.dumps(r) for r in rows)


TRAP = "mrw read absent-file; false && mrw write --format=search_replace F; mrw write plan.txt"
TRAP_RESULT = "mrw: incomplete: absent-file\n" + APPLIED


def counted(command, text):
    return edits.classify(transcript((command, text)))


def arm(n=18, landed=36, refused=0, undecided=0, uptake=0, solved=18, out=100, cost=1.0, tasks=None):
    """n attempts whose totals are the arguments; each unit goes to attempt i % n."""
    recs = []
    for i in range(n):
        recs.append(dict(task=(tasks or INSTANCES)[i], landed=landed // n + (i < landed % n), refused=refused // n + (i < refused % n),
                         undecided=undecided // n + (i < undecided % n), untokenized=0, blocked=0, attributed=int(i < uptake),
                         attributed_landed=0, has_result=True, solved=i < solved, turns=4, out=out, cost=round(cost * 10 ** 9)))
    return recs


INSTANCES = edits.INSTANCES


def judged(b=None, s=None, **kw):
    base_b = dict(landed=36, solved=12, out=200, cost=1.0)
    base_s = dict(landed=36, solved=12, out=100, cost=0.7, uptake=9)
    return edits.decide(edits.evaluate(arm(**dict(base_b, **(b or {}))), arm(**dict(base_s, **(s or {})))), **kw)


def receipt(task, config, model, solved=True, cost=0.1, out=100, exclusion="", is_error=False):
    rec = {"task": task, "config": config, "model": model, "repeat": 1, "num_turns": 5, "total_cost_usd": cost,
           "usage": {"output_tokens": out}, "oracle": {"solved": solved}, "is_error": is_error}
    if exclusion:
        rec["exclusion"] = exclusion
    return rec


def default_attempt(model, arm_name, i):
    return dict(landed=2, refused=0, attributed=1 if arm_name == "S" else 0, solved=True, out=100 if arm_name == "S" else 200, cost=0.07 if arm_name == "S" else 0.1)


def build_pilot(root, attempt=default_attempt, run_edit=None, excl=None):
    """root/receipts/<model>/ and root/transcripts/<model>/ for the registered matrix; excl names the exclusion
    reasons of an arm's base receipt and its replacements, in order ('' is a good one)."""
    root = Path(root)
    for model in edits.MODELS:
        (root / "receipts" / model).mkdir(parents=True)
        (root / "transcripts" / model).mkdir(parents=True)
        run = {"stage": "pilot-8", "models": [model], "configs": ["B", "S"], "repeats": 1, "tasks": list(INSTANCES)}
        if run_edit:
            run_edit(model, run)
        (root / "receipts" / model / "run.json").write_text(json.dumps(run))
        for i, task in enumerate(INSTANCES):
            for name in edits.ARMS:
                reasons = (excl or {}).get((model, task, name), [""])
                for k, reason in enumerate(reasons):
                    stem = f"{task}.{name}.{model}.1" + (f".r{k}" if k else "")
                    a = attempt(model, name, i)
                    (root / "receipts" / model / (stem + ".receipt.json")).write_text(
                        json.dumps(receipt(task, name, model, a["solved"], a["cost"], a["out"], reason, a.get("is_error", False))))
                    if not reason:
                        steps = [(TRAP if j < a.get("trap", 0) else "mrw write --format=search_replace f.go" if j < a["attributed"] else "mrw write plan.txt", TRAP_RESULT if j < a.get("trap", 0) else APPLIED) for j in range(a["landed"])]
                        steps += [("mrw write bad.txt", "mrw: nothing was written") for _ in range(a["refused"])]
                        (root / "transcripts" / model / (stem + ".transcript.jsonl")).write_text(transcript(*steps, result=a.get("result", True)))
    return root / "receipts", root / "transcripts"


def run_verdict(root, extra=(), **kw):
    """Build a pilot, count it, add the extra receipts (counts are made without them), and judge."""
    receipts, transcripts = build_pilot(root, **kw)
    counts = Path(root) / "counts.json"
    counts.write_text(json.dumps(edits.count(receipts, transcripts, edits.ARMS)))
    for model, stem, rec in extra:
        # a receipt, or with None a .started file that has none
        (receipts / model / (stem + (".started" if rec is None else ".receipt.json"))).write_text("" if rec is None else json.dumps(rec))
    return edits.verdict(receipts, counts, transcripts)


class BenchEditsTest(unittest.TestCase):
    def tmp(self):
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        return d.name

    def test_a_write_call_is_refused_only_on_an_mrw_refusal(self):
        self.assertEqual(counted("mrw write P", APPLIED)["landed"], 1)
        self.assertEqual(counted("mrw write P", "PARTIALLY APPLIED\nf.go")["landed"], 1)
        for text in ["mrw: nothing was written", "NOTHING WAS WRITTEN", "plan has 1 error(s)", "search_replace: no match", "apply_patch: bad",
                     "flag provided but not defined: -z", "cannot hold a line break", "is not relative to the root", "plan is empty"]:
            c = counted("mrw write P", text)
            self.assertEqual((c["refused"], c["landed"], c["undecided"]), (1, 0, 0), text)
        c = counted("mrw write P", "mrw: something else went wrong")
        self.assertEqual((c["refused"], c["landed"]), (1, 0))
        self.assertEqual(counted("mrw write P", "ok\nmrw: boom")["refused"], 1)  # the line is not the first
        self.assertEqual(counted("mrw write P", APPLIED + "\nmrw: nothing recorded")["landed"], 1)
        c = counted("mrw write P", "bash: line 3: unexpected EOF")
        self.assertEqual((c["undecided"], c["refused"], c["landed"], c["decided"]), (1, 0, 0, 0))
        none = counted("mrw write --dry-run P", "dry run, nothing written")
        self.assertEqual({k: v for k, v in none.items() if k != "has_result"}, {k: 0 for k in none if k != "has_result"})

    def test_a_bench_hook_block_is_not_a_write(self):
        c = counted("mrw write P", "PreToolUse:Bash hook error: nothing was written, plan has 1 error")
        self.assertEqual((c["blocked"], c["refused"], c["landed"], c["undecided"], c["decided"]), (1, 0, 0, 0, 0))
        c = counted("printf '<<<<<<< SEARCH\n' | mrw write --format=search_replace -", "PreToolUse:Bash hook error")
        self.assertEqual((c["blocked"], c["untokenized"], c["undecided"]), (1, 0, 0))

    def test_uptake_needs_one_attributed_search_replace_write(self):
        for flag in ("--format=search_replace", "--format search_replace"):
            for text, landed in ((APPLIED, 1), ("search_replace: SEARCH matches 2 times", 0)):
                c = counted(f"mrw write {flag} F", text)
                self.assertEqual((c["attributed"], c["attributed_landed"]), (1, landed), (flag, text))
        c = counted("false && mrw write --format=search_replace F", "")
        self.assertEqual((c["attributed"], c["undecided"]), (0, 1))
        c = counted("false && mrw write --format=search_replace F; mrw write P", APPLIED)
        self.assertEqual((c["attributed"], c["landed"]), (0, 1))
        for command in ("mrw write P", "mrw write - <<'EOF'\n@@ f.go 1 replace\nx\nEOF", "mrw write --format=apply_patch F"):
            self.assertEqual(counted(command, APPLIED)["attributed"], 0, command)
        # a read error earlier in the call is not a SEARCH/REPLACE write's verdict (Codex on #612)
        c = counted("mrw read absent-file; false && mrw write --format=search_replace F", "mrw: incomplete: absent-file")
        self.assertEqual((c["attributed"], c["refused"]), (0, 1))
        c = counted(TRAP, TRAP_RESULT)
        self.assertEqual((c["attributed"], c["landed"]), (0, 1))
        c = counted("mrw read F && mrw write --format=search_replace F2", APPLIED)
        self.assertEqual((c["attributed"], c["landed"]), (0, 1))

        def trap(model, name, i):
            a = default_attempt(model, name, i)
            return dict(a, attributed=0, trap=1) if name == "S" and i < 9 else dict(a, attributed=0) if name == "S" else a
        lines = run_verdict(self.tmp(), attempt=trap)
        self.assertIn("0 of 18", [r for r in lines if "(U)" in r][0])
        self.assertEqual(lines[-1], "verdict: NO SHIP")

    def test_a_search_replace_heredoc_is_one_write_call(self):
        filed = "cat > $TMPDIR/sr.txt <<'EOF'\n" + SR_DOC + "\nEOF\nmrw write --format=search_replace $TMPDIR/sr.txt"
        piped = "mrw write --format=search_replace - <<'EOF'\n" + SR_DOC + "\nEOF"
        for command in (filed, piped):
            self.assertIsNotNone(mrw_calls(command)[0], command)
            c = counted(command, APPLIED)
            self.assertEqual((c["landed"], c["attributed"], c["attributed_landed"], c["untokenized"]), (1, 1, 1, 0), command)

    def test_an_untokenized_command_holding_mrw_write_is_undecided(self):
        command = "printf 'f.go\n<<<<<<< SEARCH\nold\n=======\nnew\n>>>>>>> REPLACE\n' | mrw write --format=search_replace -"
        self.assertIsNone(mrw_calls(command)[0])
        c = counted(command, APPLIED)
        self.assertEqual((c["untokenized"], c["undecided"], c["landed"], c["attributed"], c["decided"]), (1, 1, 0, 0, 0))
        d = judged(s=dict(landed=89, undecided=11), b=dict(landed=89))
        self.assertEqual(d["result"], "UNTESTED")

    def test_a_mixed_call_counts_as_refused(self):
        text = "plan has 1 error(s)\n  hunk 1: bad\n" + APPLIED  # the Opus sphinx-doc__sphinx-7748 shape: no \"nothing was written\" line
        c = counted("mrw write A; mrw write B", text)
        self.assertEqual((c["refused"], c["landed"]), (1, 0))

    def test_global_flags_before_write_are_counted(self):
        for command in ("mrw -C DIR write P", "mrw --root DIR write P", "mrw --root=DIR write P"):
            self.assertEqual(counted(command, APPLIED)["landed"], 1, command)
        self.assertEqual(counted("mrw -C DIR read F", APPLIED)["landed"], 0)
        self.assertEqual(counted("mrw -C DIR write --dry-run P", APPLIED)["landed"], 0)

    def test_the_verdict_follows_the_registered_rule(self):
        self.assertEqual(judged()["result"], "PASS")
        for n, uptake, want in ((18, 8, "UNTESTED"), (18, 9, "PASS"), (16, 7, "UNTESTED"), (16, 8, "PASS")):
            d = edits.decide(edits.evaluate(arm(n=n, landed=36, solved=12, out=200), arm(n=n, landed=36, solved=12, out=100, cost=0.7, uptake=uptake)))
            self.assertEqual(d["result"], want, (n, uptake))
        self.assertEqual(judged(b=dict(landed=90), s=dict(landed=90, undecided=10))["result"], "PASS")
        self.assertEqual(judged(b=dict(landed=89), s=dict(landed=89, undecided=11))["result"], "UNTESTED")  # 89%
        self.assertEqual(judged(b=dict(landed=90), s=dict(landed=90, undecided=10))["coverage"]["S"], True)
        self.assertEqual(judged(s=dict(solved=9))["result"], "FAIL")
        self.assertEqual(judged(s=dict(solved=10))["result"], "PASS")
        # (i) at two thirds of B's, with B at 3 refusals; the second branch of (b) is shut by equal output
        eq = dict(out=200)
        self.assertEqual(judged(b=dict(landed=33, refused=3, **eq), s=dict(landed=34, refused=2, **eq))["result"], "PASS")
        self.assertEqual(judged(b=dict(landed=34, refused=2, **eq), s=dict(landed=35, refused=1, **eq))["result"], "FAIL")
        self.assertEqual(judged(b=dict(landed=33, refused=3, **eq), s=dict(landed=33, refused=3, **eq))["result"], "FAIL")
        # (ii) at 0.85 and above it
        self.assertEqual(judged(s=dict(out=170))["result"], "PASS")
        self.assertEqual(judged(s=dict(out=171))["result"], "FAIL")
        # (g) at 1.2 and above it
        self.assertEqual(judged(b=dict(landed=30), s=dict(landed=36))["result"], "PASS")
        self.assertEqual(judged(b=dict(landed=30), s=dict(landed=37))["result"], "FAIL")
        # (c) at 1.10 and above it
        self.assertEqual(judged(s=dict(cost=1.1))["result"], "PASS")
        self.assertEqual(judged(s=dict(cost=1.11))["result"], "FAIL")
        # solved is bench's predicate: Oracle.Solved, not over budget, not an agent error (Codex on #612)
        self.assertEqual(edits.record(dict(has_result=True), dict(oracle={"solved": True}))["solved"], True)
        for flag in ("is_error", "over_budget"):
            self.assertEqual(edits.record(dict(has_result=True), {"oracle": {"solved": True}, flag: True})["solved"], False, flag)

        def errors(model, name, i):
            a = default_attempt(model, name, i)
            return dict(a, is_error=True) if name == "S" and i < 3 else a
        self.assertEqual(run_verdict(self.tmp(), attempt=errors)[-1], "verdict: NO SHIP")

    def test_undefined_measures_fail_conservatively(self):
        self.assertEqual(judged(b=dict(landed=0))["result"], "UNTESTED")
        self.assertEqual(judged(s=dict(landed=0, refused=36))["clauses"]["g"], False)
        self.assertEqual(judged(s=dict(landed=0, refused=36))["result"], "FAIL")
        d = judged(b=dict(landed=36), s=dict(out=170))
        self.assertEqual((d["clauses"]["b1"], d["clauses"]["b2"], d["result"]), (False, True, "PASS"))
        self.assertEqual(judged(s=dict(out=171))["result"], "FAIL")
        for who in ("b", "s"):
            d = judged(**{who: dict(solved=0)})
            self.assertEqual(d["clauses"]["c"], False, who)
        # an empty both-solved subset: no interval, and the verdict is what decide() said
        b, s = arm(landed=36, solved=6, out=200), arm(landed=36, solved=6, out=100, cost=0.7, uptake=9)
        for r in s:
            r["solved"] = 6 <= INSTANCES.index(r["task"]) < 12
        result, rows = edits.model_lines("m", b, s, [])
        self.assertIn("both-solved subset 0, S spend over B's minus one none", "\n".join(rows))
        self.assertEqual(result, edits.decide(edits.evaluate(b, s))["result"])
        # a missing output and cost: S takes the model's largest, B the smallest, never zero
        b, s = arm(landed=36, out=100, cost=1.0), arm(landed=36, out=50, cost=0.5, uptake=18)
        for r in (b[0], s[0]):
            r.update(has_result=False, out=None, cost=None)
        m = edits.evaluate(b, s)
        self.assertEqual((m["S"]["out"], m["B"]["out"]), (17 * 50 + 100, 17 * 100 + 50))
        self.assertEqual((m["S"]["cost"], m["B"]["cost"]), (17 * 500000000 + 10 ** 9, 17 * 10 ** 9 + 500000000))
        # nothing to impute from is undefined, not zero
        alone = arm(n=2, tasks=INSTANCES)
        for r in alone:
            r.update(out=None, cost=None)
        self.assertIsNone(edits.evaluate(alone, alone)["S"]["out"])
        # an S attempt whose transcript has no result takes the model's largest output and cost, through count and verdict
        def lost(model, name, i):
            a = default_attempt(model, name, i)
            return dict(a, result=False, out=0, cost=None) if (model, name, i) == (edits.MODELS[0], "S", 0) else a
        root = self.tmp()
        receipts, transcripts = build_pilot(root, attempt=lost)
        counts = edits.count(receipts, transcripts, edits.ARMS)
        self.assertEqual([a["has_result"] for a in counts["attempts"] if a["id"] == f"{INSTANCES[0]}.S.{edits.MODELS[0]}.1"], [False])
        (Path(root) / "counts.json").write_text(json.dumps(counts))
        row = [r for r in edits.verdict(receipts, Path(root) / "counts.json") if r.startswith("  S:")][0]
        self.assertIn("(ii) output per attempt 105.6", row)  # (17 * 100 + the largest, 200) / 18, not 94.4
        self.assertIn("(iii) spend per success 0.0717", row)  # (17 * 0.07 + the largest, 0.1) / 18, not 0.0661

    def test_the_exclusion_walk_takes_the_final_receipt(self):
        def recs(*reasons):
            return {"t.B.m.1" + (f".r{k}" if k else ""): {"exclusion": r} for k, r in enumerate(reasons)}
        self.assertEqual(edits.walk(recs("setup", ""), "t.B.m.1")[::2], ("t.B.m.1.r1", False))
        self.assertEqual(edits.walk(recs("start-error", "start-error", ""), "t.B.m.1")[::2], ("t.B.m.1.r2", False))
        self.assertEqual(edits.walk(recs("setup", "setup"), "t.B.m.1")[::2], ("t.B.m.1.r1", True))
        self.assertEqual(edits.walk(recs("start-error", "start-error", "start-error"), "t.B.m.1")[::2], ("t.B.m.1.r2", True))
        with self.assertRaises(edits.Refusal):
            edits.walk(recs("infrastructure"), "t.B.m.1")
        with self.assertRaises(edits.Refusal):
            edits.walk(recs("setup"), "t.B.m.1")  # the replacement it is owed is missing
        # a dropped instance leaves both arms for that model
        task = INSTANCES[0]
        for reasons in (["setup", "setup"], ["start-error", "start-error", "start-error"]):
            root = self.tmp()
            receipts, transcripts = build_pilot(root, excl={(edits.MODELS[0], task, "S"): reasons})
            out = edits.count(receipts, transcripts, edits.ARMS)
            ids = {a["id"] for a in out["attempts"]}
            self.assertFalse(any(i.startswith(task + ".B." + edits.MODELS[0]) for i in ids))
            self.assertEqual(len(ids), 72 - 2)
        # three dropped instances stop the pilot; two do not
        def drop(n):
            return {(edits.MODELS[0], INSTANCES[k], "S"): ["setup", "setup"] for k in range(n)}
        run_verdict(self.tmp(), excl=drop(2))
        with self.assertRaises(edits.Refusal):
            run_verdict(self.tmp(), excl=drop(3))
        with self.assertRaises(edits.Refusal):
            run_verdict(self.tmp(), excl={(edits.MODELS[0], task, "B"): ["infrastructure"]})
        with self.assertRaises(edits.Refusal):
            run_verdict(self.tmp(), excl={(edits.MODELS[0], task, "B"): ["setup"]})  # the replacement it is owed is missing
        with self.assertRaises(edits.Refusal):
            run_verdict(self.tmp(), extra=[(edits.MODELS[0], f"{task}.B.{edits.MODELS[0]}.1.r1", None)])  # started, no receipt

    def test_the_verdict_refuses_another_instance_list(self):
        def swap(model, run):
            run["tasks"][0], run["tasks"][1] = run["tasks"][1], run["tasks"][0]

        def omit(model, run):
            run["tasks"].pop()

        def duplicate(model, run):
            run["tasks"][-1] = run["tasks"][0]

        def foreign(model, run):
            run["tasks"][-1] = "django__django-1"

        def another_model(model, run):
            run["models"] = ["claude-haiku"]

        def third_arm(model, run):
            run["configs"] = ["B", "S", "A"]

        def second_repeat(model, run):
            run["repeats"] = 2
        for edit in (swap, omit, duplicate, foreign, another_model, third_arm, second_repeat):
            with self.assertRaises(edits.Refusal, msg=edit.__name__):
                run_verdict(self.tmp(), run_edit=edit)
        model, task = edits.MODELS[0], INSTANCES[0]
        for extra in ((model, f"{task}.A.{model}.1", receipt(task, "A", model)),
                      (model, f"{task}.B.{model}.2", dict(receipt(task, "B", model), repeat=2)),
                      (model, f"django__django-1.B.{model}.1", receipt("django__django-1", "B", model)),
                      (model, f"{task}.B.other-model.1", receipt(task, "B", "other-model")),
                      (model, f"{task}.B.{model}.1.r1", receipt(task, "S", model))):  # a receipt that is not its file's attempt
            with self.assertRaises(edits.Refusal, msg=extra[1]):
                run_verdict(self.tmp(), extra=[extra])
        lines = run_verdict(self.tmp(), excl={(model, INSTANCES[3], "S"): ["setup", ""]})  # retries are not counted toward the 72
        self.assertTrue(any(row.startswith(model + ": 18 paired instances") for row in lines))
        self.assertEqual(lines[-1], "verdict: SHIP")

    def test_counts_must_match_the_final_receipts_and_digests(self):
        root = self.tmp()
        receipts, transcripts = build_pilot(root)
        good = edits.count(receipts, transcripts, edits.ARMS)
        self.assertEqual(len(good["attempts"]), 72)
        path = Path(root) / "counts.json"

        def check(data, **kw):
            path.write_text(json.dumps(data))
            return edits.verdict(receipts, path, **kw)
        self.assertEqual(check(good, transcripts=transcripts)[-1], "verdict: SHIP")
        extra = dict(good, attempts=good["attempts"] + [dict(good["attempts"][0], id="ghost.B.m.1")])
        missing = dict(good, attempts=good["attempts"][1:])
        twice = dict(good, attempts=good["attempts"] + [good["attempts"][0]])
        for bad in (extra, missing, twice):
            with self.assertRaises(edits.Refusal):
                check(bad)
        wrong = json.loads(json.dumps(good))
        wrong["attempts"][5]["transcript_sha256"] = hashlib.sha256(b"another transcript").hexdigest()
        check(wrong)  # no transcripts to recompute from: the digest is only well formed
        with self.assertRaises(edits.Refusal):
            check(wrong, transcripts=transcripts)
        wrong["attempts"][5]["transcript_sha256"] = "nope"
        with self.assertRaises(edits.Refusal):
            check(wrong)
        # one model short of PASS makes the verdict NO SHIP

        def sonnet_wordy(model, name, i):
            a = default_attempt(model, name, i)
            return dict(a, out=400) if model == edits.MODELS[1] and name == "S" else a
        self.assertEqual(run_verdict(self.tmp(), attempt=sonnet_wordy)[-1], "verdict: NO SHIP")


if __name__ == "__main__":
    unittest.main()
