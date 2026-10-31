"""Controls for the serial grouped pilot; uses a stub, never a real model."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import grouped_pilot as grouped


class ScheduleAndScore(unittest.TestCase):
    def test_schedule_is_repeatable_and_alternates_pair_order(self):
        names = ["a", "b", "c", "d", "e"]
        first = grouped.paired_plan(names)
        self.assertEqual(first, grouped.paired_plan(list(reversed(names))))
        self.assertEqual(sorted(name for name, _ in first), names)
        self.assertEqual([arms for _, arms in first], ["AB", "BA", "AB", "BA", "AB"])

    def test_success_text_or_correct_tree_cannot_override_a_failed_run(self):
        record = {"process_exit":0,"result_error":False,"timed_out":False,"num_turns":3,"stream_errors":0,
                  "actual_model":"pinned","tools":grouped.TOOLS["A"],"mcp_servers":[],"plugins":[],
                  "evidence":{"status":"verified"}}
        self.assertTrue(grouped.verdict(record, {"solved":True}, 0, "A", "pinned")["solved"])
        for change in [{"process_exit":1}, {"num_turns":11}, {"actual_model":"other"}, {"plugins":["unexpected"]}]:
            with self.subTest(change=change):
                self.assertFalse(grouped.verdict(dict(record, **change), {"solved":True}, 0, "A", "pinned")["solved"])
        self.assertFalse(grouped.verdict(record, {"solved":False}, 0, "A", "pinned")["solved"])
        self.assertFalse(grouped.verdict(record, {"solved":True}, 1, "A", "pinned")["solved"])

    def test_malformed_metrics_remain_unknown_instead_of_dropping_a_cell(self):
        metrics = grouped.tool_metrics(b'[]\n{')
        self.assertEqual(metrics["parse_errors"], 2)
        self.assertIsNone(metrics["tool_calls"])


class PartialRuns(unittest.TestCase):
    def test_started_cell_failure_is_retained_and_not_called_unrun(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "tasks/job").mkdir(parents=True)
            (root / "evidence").mkdir()
            def fail_after_start(task, arm, config):
                (root / "evidence" / (task.name + "." + arm + ".started")).touch()
                raise ValueError("adapter record missing")
            with mock.patch.object(grouped, "execute_cell", side_effect=fail_after_start):
                summary = grouped.run({"root":d,"corpus":str(root / "tasks")})
            self.assertEqual(summary["attempts"], [{"task":"job","arm":"A","status":"failed",
                             "stage":"execute_cell","adapter_started":True,"error":"adapter record missing"}])
            self.assertEqual(summary["not_run"], [{"task":"job","arm":"B"}])
            self.assertEqual(json.loads((root / "run-summary.json").read_text()), json.loads(json.dumps(summary)))

    def test_interrupt_outside_child_wait_saves_partial_summary(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "tasks/job").mkdir(parents=True)
            with mock.patch.object(grouped, "execute_cell", side_effect=KeyboardInterrupt):
                summary = grouped.run({"root":d,"corpus":str(root / "tasks")})
            self.assertEqual(summary["stop_reason"], "interrupted")
            self.assertEqual(summary["attempts"][0]["status"], "interrupted")
            self.assertEqual(summary["not_run"], [{"task":"job","arm":"B"}])
            self.assertEqual(json.loads((root / "run-summary.json").read_text()), json.loads(json.dumps(summary)))
            with mock.patch.object(grouped, "run", return_value=summary), mock.patch.object(
                    sys, "argv", ["grouped-pilot", str(root / "run-summary.json")]):
                self.assertEqual(grouped.main(), 130)


@unittest.skipUnless(os.name == "posix", "pilot execution requires POSIX")
class StubExecution(unittest.TestCase):
    def test_cell_retains_grading_and_metrics_before_removing_the_tree(self):
        with tempfile.TemporaryDirectory(prefix="grouped-test-") as d:
            root = Path(d)
            task = Path(__file__).resolve().parents[1] / "bench/grouped-pilot/tasks/grouped-routing"
            stub = root / "stub"
            stub.write_text("#!" + sys.executable + "\n"
                            "import json,subprocess,sys\n"
                            "subprocess.run(['git','apply'," + repr(str(task/"reference.patch")) + "],check=True)\n"
                            "tools=sys.argv[sys.argv.index('--tools')+1].split(',')\n"
                            "print(json.dumps({'type':'system','subtype':'init','model':'stub-model','tools':tools,'mcp_servers':[],'plugins':[]}))\n"
                            "print(json.dumps({'type':'result','is_error':False,'num_turns':1,'total_cost_usd':.01,'usage':{'input_tokens':1,'output_tokens':1}}))\n")
            stub.chmod(0o755)
            for folder in ("evidence","cells","transcripts"):
                (root/folder).mkdir()
            adapter = {"claude":str(stub),"mrw":shutil.which("mrw"),"out":str(root/"evidence"),"max_launches":12}
            (root/"adapter.json").write_text(json.dumps(adapter))
            config = {"root":str(root),"adapter":str(root/"adapter.json"),"bench":"/unused",
                      "model":"stub-model","profile":str(root/"profile")}
            row = grouped.execute_cell(task, "A", config)
            self.assertTrue(row["verdict"]["solved"], row)
            self.assertEqual(row["matched_sites"], 10)
            self.assertEqual(row["changed_files"], 5)
            self.assertEqual(row["record"]["num_turns"], 1)
            self.assertTrue((root/"cells/grouped-routing.A.json").exists())
            self.assertTrue((root/"transcripts/grouped-routing.A.jsonl").exists())
            self.assertEqual(row["record"]["instruction_sha256"].get("appended_file"), None)

    def test_core_cell_appends_the_registered_core_hides_its_arm_name_and_pins_mrw(self):
        with tempfile.TemporaryDirectory(prefix="grouped-test-") as d:
            root = Path(d)
            task = Path(__file__).resolve().parents[1] / "bench/grouped-pilot/tasks/grouped-routing"
            stub = root / "stub"
            stub.write_text("#!" + sys.executable + "\n"
                            "import json,os,shutil,subprocess,sys\n"
                            "open(" + repr(str(root / "seen.json")) + ",'w').write(json.dumps("
                            "{'arm':os.environ.get('BENCH_ARM'),'mrw':shutil.which('mrw')}))\n"
                            "subprocess.run(['git','apply'," + repr(str(task/"reference.patch")) + "],check=True)\n"
                            "tools=sys.argv[sys.argv.index('--tools')+1].split(',')\n"
                            "print(json.dumps({'type':'system','subtype':'init','model':'stub-model','tools':tools,'mcp_servers':[],'plugins':[]}))\n"
                            "print(json.dumps({'type':'result','is_error':False,'num_turns':1,'total_cost_usd':.01,'usage':{'input_tokens':1,'output_tokens':1}}))\n")
            stub.chmod(0o755)
            for folder in ("evidence","cells","transcripts"):
                (root/folder).mkdir()
            adapter = {"claude":str(stub),"mrw":shutil.which("mrw"),"out":str(root/"evidence"),"max_launches":12}
            (root/"adapter.json").write_text(json.dumps(adapter))
            core = subprocess.run([shutil.which("mrw"), "instructions", "--core"], check=True, capture_output=True).stdout
            # The core this mrw prints, not run 2's constant: CI builds mrw from the tree (ADR-184).
            config = {"root":str(root),"adapter":str(root/"adapter.json"),"bench":"/bin/true",
                      "model":"stub-model","profile":str(root/"profile"),
                      "registered_guides":{"C":hashlib.sha256(core).hexdigest()}}
            row = grouped.execute_cell(task, "C", config)
            self.assertEqual(row["record"]["instruction_sha256"]["appended_file"], hashlib.sha256(core).hexdigest())
            self.assertEqual(row["record"]["tools"], grouped.TOOLS["B"])
            self.assertTrue(row["verdict"]["isolation_ok"], row["verdict"])
            self.assertTrue((root/"evidence/grouped-routing.C.json").exists())
            seen = json.loads((root/"seen.json").read_text())
            self.assertIsNone(seen["arm"])
            self.assertTrue(seen["mrw"])


@unittest.skipUnless(os.name == "posix", "pilot execution requires POSIX")
class RegisteredGuideFromConfiguration(unittest.TestCase):
    def test_a_configured_registered_hash_replaces_the_constant_and_refuses_before_launch(self):
        with tempfile.TemporaryDirectory(prefix="grouped-test-") as d:
            root = Path(d)
            task = Path(__file__).resolve().parents[1] / "bench/grouped-pilot/tasks/grouped-routing"
            stub = root / "stub"
            stub.write_text("#!" + sys.executable + "\nopen(" + repr(str(root / "launched")) + ",'w').write('x')\n")
            stub.chmod(0o755)
            for folder in ("evidence", "cells", "transcripts"):
                (root / folder).mkdir()
            adapter = {"claude": str(stub), "mrw": shutil.which("mrw"), "out": str(root / "evidence"), "max_launches": 12}
            (root / "adapter.json").write_text(json.dumps(adapter))
            config = {"root": str(root), "adapter": str(root / "adapter.json"), "bench": "/bin/true",
                      "model": "stub-model", "profile": str(root / "profile"), "registered_guides": {"C": "0" * 64}}
            with self.assertRaisesRegex(ValueError, "not the registered one"):
                grouped.execute_cell(task, "C", config)
            self.assertFalse((root / "launched").exists())
            self.assertEqual(grouped.registered_guides(config)["F"], grouped.REGISTERED_GUIDES["F"])
            self.assertEqual(grouped.registered_guides({})["C"], grouped.REGISTERED_GUIDES["C"])


class GuideAblation(unittest.TestCase):
    def test_full_and_core_arms_take_arm_b_tools_and_their_own_guide(self):
        self.assertEqual(grouped.TOOLS["F"], grouped.TOOLS["B"])
        self.assertEqual(grouped.TOOLS["C"], grouped.TOOLS["B"])
        self.assertEqual(grouped.APPENDED["F"], ["instructions"])
        self.assertEqual(grouped.APPENDED["C"], ["instructions", "--core"])

    def test_registered_guide_hashes_match_the_installed_mrw(self):
        mrw = shutil.which("mrw")
        if not mrw or subprocess.run([mrw, "version"], capture_output=True, text=True).stdout.strip() != "v1.79.0":
            self.skipTest("registered hashes are for mrw v1.79.0")
        for arm, digest in grouped.REGISTERED_GUIDES.items():
            out = subprocess.run([mrw, *grouped.APPENDED[arm]], check=True, capture_output=True).stdout
            self.assertEqual(hashlib.sha256(out).hexdigest(), digest, arm)

    def test_round_two_keeps_job_order_and_reverses_every_pair(self):
        names = ["a", "b", "c", "d", "e"]
        first, second = grouped.paired_plan(names, "FC"), grouped.paired_plan(names, "FC", reverse=True)
        self.assertEqual([arms for _, arms in first], ["FC", "CF", "FC", "CF", "FC"])
        self.assertEqual([n for n, _ in first], [n for n, _ in second])
        self.assertEqual([arms for _, arms in second], [arms[::-1] for _, arms in first])
        self.assertEqual(grouped.paired_plan(names), grouped.paired_plan(names, "AB"))

    def round_one(self, d, cost=.2, stop="completed", cell_ok=True, canary_ok=True):
        prior = Path(d) / "round1"
        (prior / "evidence").mkdir(parents=True)
        identities = ["canary.F", "canary.C", "job.F", "job.C"]
        for identity in identities:
            row = {"cost_usd": cost / len(identities), "usage": {"input_tokens": 1, "output_tokens": 1}}
            (prior / "evidence" / (identity + ".json")).write_text(json.dumps(row))
        summary = {"stop_reason": stop, "plan": [["job", "FC"]], "not_run": [],
                   "cells": [{"task": "job", "arm": arm, "verdict": {"isolation_ok": cell_ok}} for arm in "FC"],
                   "canaries": [{"task": "canary", "arm": arm, "verdict": {"isolation_ok": canary_ok},
                                 "problems": []} for arm in "FC"]}
        (prior / "run-summary.json").write_text(json.dumps(summary))
        (prior / "adapter.json").write_text(json.dumps({"out": str(prior / "evidence")}))
        adapter = Path(d) / "adapter2.json"
        adapter.write_text(json.dumps({"max_usd": round(3 - cost, 6)}))
        return {"round": 2, "arms": "FC", "round1_root": str(prior), "adapter": str(adapter)}

    def test_round_two_gate_refuses_an_incomplete_or_unknown_round_one(self):
        def drop_row(key, arm):
            def change(prior):
                path = prior / "run-summary.json"
                summary = json.loads(path.read_text())
                summary[key] = [row for row in summary[key] if row["arm"] != arm]
                path.write_text(json.dumps(summary))
            return change
        def evidence(edit):
            def change(prior):
                edit(prior / "evidence")
            return change
        cases = {"missing canary": drop_row("canaries", "C"), "missing cell": drop_row("cells", "F"),
                 "missing record": evidence(lambda e: (e / "job.C.json").unlink()),
                 "unknown usage in the last record": evidence(lambda e: (e / "job.C.json").write_text(
                     json.dumps({"cost_usd": .05, "usage": None}))),
                 "unfinished launch": evidence(lambda e: (e / "extra.F.started").touch())}
        for name, change in cases.items():
            with self.subTest(name), tempfile.TemporaryDirectory() as d:
                config = self.round_one(d)
                change(Path(config["round1_root"]))
                with self.assertRaises(ValueError):
                    grouped.round_two_gate(config)

    def test_round_two_gate_passes_a_clean_cheap_round_one(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertAlmostEqual(grouped.round_two_gate(self.round_one(d)), .2)

    def test_round_two_gate_refuses_each_failing_condition(self):
        cases = {"stop": dict(stop="insufficient time"), "cell": dict(cell_ok=False),
                 "canary": dict(canary_ok=False), "cost": dict(cost=1.5)}
        for name, change in cases.items():
            with self.subTest(name), tempfile.TemporaryDirectory() as d:
                with self.assertRaises(ValueError):
                    grouped.round_two_gate(self.round_one(d, **change))
        with tempfile.TemporaryDirectory() as d:
            config = self.round_one(d)
            Path(config["adapter"]).write_text(json.dumps({"max_usd": 3}))
            with self.assertRaisesRegex(ValueError, "ceiling"):
                grouped.round_two_gate(config)
        with tempfile.TemporaryDirectory() as d:
            config = self.round_one(d)
            (Path(config["round1_root"]) / "run-summary.json").unlink()
            with self.assertRaises((ValueError, OSError)):
                grouped.round_two_gate(config)

    def test_failing_gate_launches_nothing_not_even_a_canary(self):
        with tempfile.TemporaryDirectory() as d:
            config = self.round_one(d, cost=1.6)
            (Path(d) / "tasks/job").mkdir(parents=True)
            config.update(root=str(Path(d) / "round2"), corpus=str(Path(d) / "tasks"))
            Path(config["root"]).mkdir()
            with mock.patch.object(grouped, "execute_cell") as launch:
                summary = grouped.run(config)
            launch.assert_not_called()
            self.assertIn("round 1", summary["stop_reason"])
            self.assertEqual(summary["not_run"], [{"task": "job", "arm": "C"}, {"task": "job", "arm": "F"}])

    def test_a_failed_canary_stops_before_any_counted_cell(self):
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "tasks/job").mkdir(parents=True)
            config = {"root": d, "corpus": str(Path(d) / "tasks"), "arms": "FC", "round": 1}
            bad = {"verdict": {"execution_ok": True, "isolation_ok": False, "exact_tree_ok": True}}
            with mock.patch.object(grouped, "execute_cell", return_value=bad) as launch, \
                 mock.patch.object(grouped, "canary_problems", return_value=["isolation failed"]):
                summary = grouped.run(config)
            self.assertEqual([call.args[1] for call in launch.call_args_list], ["F"])
            self.assertEqual(summary["stop_reason"], "canary failed")
            self.assertEqual(summary["not_run"], [{"task": "job", "arm": "F"}, {"task": "job", "arm": "C"}])

    def test_canary_reports_a_leak_a_wrong_tree_and_a_failed_isolation(self):
        ok = {"verdict": {"execution_ok": True, "isolation_ok": True, "exact_tree_ok": True}}
        memo = "This sentence is distinctive enough to be a fingerprint of the user's file.\n"
        self.assertEqual(grouped.canary_problems(ok, b'{"type":"result"}', memo), [])
        self.assertTrue(grouped.canary_problems(ok, memo.encode(), memo))
        self.assertTrue(grouped.canary_problems(ok, b'{"x":"memoryd data"}', memo))
        for key in ("execution_ok", "isolation_ok", "exact_tree_ok"):
            row = {"verdict": dict(ok["verdict"], **{key: False})}
            self.assertTrue(grouped.canary_problems(row, b"", memo), key)


if __name__ == "__main__":
    unittest.main()
