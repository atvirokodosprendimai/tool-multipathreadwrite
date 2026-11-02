"""Controls for the serial grouped pilot; uses a stub, never a real model."""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import shutil
import subprocess
import sys
import tempfile
import unittest
import unittest.mock

mock = unittest.mock

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

    def test_a_configured_order_overrides_the_round_and_is_what_run_plans(self):
        self.assertTrue(grouped.reversed_order({"round": 2}))
        self.assertFalse(grouped.reversed_order({"round": 1}))
        self.assertTrue(grouped.reversed_order({"round": 1, "reverse": True}))
        self.assertFalse(grouped.reversed_order({"round": 2, "reverse": False}))
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "tasks/job").mkdir(parents=True)
            config = {"root": d, "corpus": str(Path(d) / "tasks"), "arms": "FC", "round": 1, "reverse": True}
            with mock.patch.object(grouped, "execute_cell", side_effect=ValueError("stop")):
                summary = grouped.run(config)
            self.assertEqual(summary["plan"], [("job", "CF")])

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

    def test_a_native_and_mrw_run_opens_with_a_canary_for_each_arm(self):
        # NATIVE-PREREGISTRATION.md: arms AC get the canary pair FC got.
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "tasks/job").mkdir(parents=True)
            config = {"root": d, "corpus": str(Path(d) / "tasks"), "arms": "AC", "round": 1}
            (Path(d) / "transcripts").mkdir()
            for arm in "AC":
                (Path(d) / "transcripts" / ("canary." + arm + ".jsonl")).write_text("")
            ok = {"verdict": {"execution_ok": True, "isolation_ok": True, "exact_tree_ok": True}}
            with mock.patch.object(grouped, "execute_cell", side_effect=ValueError("stop after canaries")) as launch, \
                 mock.patch.object(grouped, "canary_problems", return_value=[]):
                launch.side_effect = [dict(ok, task="canary", arm="A"), dict(ok, task="canary", arm="C"), ValueError("stop")]
                summary = grouped.run(config)
            self.assertEqual([(c.args[0].name, c.args[1]) for c in launch.call_args_list][:2], [("canary", "A"), ("canary", "C")])
            self.assertEqual([r["arm"] for r in summary["canaries"]], ["A", "C"])

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


class HostReadArm(unittest.TestCase):
    """Arm R: MRW with the host's Read and ADR-191's read hook; edits still go through mrw."""

    def test_r_reads_with_the_host_but_has_no_edit_or_write(self):
        self.assertEqual(grouped.TOOLS["R"], ["Bash", "Read", "Glob", "Grep"])
        self.assertNotIn("Edit", grouped.TOOLS["R"])
        self.assertNotIn("Write", grouped.TOOLS["R"])
        self.assertEqual(grouped.APPENDED["R"], grouped.APPENDED["C"])

    def test_r_carries_the_bench_hook_and_the_runs_own_mrw_read_hook(self):
        config, scratch = {"bench": "/opt/bench"}, Path("/scratch")
        mrw = "/runs/own dir/mrw"
        settings = grouped.cell_settings("R", config, mrw, scratch, scratch / "tree")
        self.assertEqual(settings["hooks"]["PreToolUse"],
                         grouped.cell_settings("C", config, mrw, scratch, scratch / "tree")["hooks"]["PreToolUse"])
        # The read hook's stderr (a missed lock deadline, a newer ledger) goes to a file the row keeps.
        self.assertEqual(settings["hooks"]["PostToolUse"],
                         [{"matcher": "Read", "hooks": [{"type": "command", "command": "'/runs/own dir/mrw' hook read 2>> /scratch/hook.stderr"}]}])
        # The pair: no other arm gets the read hook, and the native arm gets no hook at all.
        for arm in "BFC":
            self.assertEqual(list(grouped.cell_settings(arm, config, mrw, scratch, scratch / "tree")["hooks"]), ["PreToolUse"], arm)
        self.assertEqual(grouped.cell_settings("A", config, mrw, scratch, scratch / "tree"), {})

    def test_a_cells_read_hook_stderr_is_kept_and_empty_without_one(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertEqual(grouped.hook_stderr(Path(d)), "")
            (Path(d) / "hook.stderr").write_text("mrw: hook read: the ledger lock was not taken in 1s\n")
            self.assertEqual(grouped.hook_stderr(Path(d)), "mrw: hook read: the ledger lock was not taken in 1s\n")

    def test_r_appends_c_guide_and_refuses_one_not_registered(self):
        self.assertEqual(grouped.REGISTERED_GUIDES["R"], grouped.REGISTERED_GUIDES["C"])
        mrw = shutil.which("mrw")
        if not mrw:
            self.skipTest("no mrw on PATH")
        core = subprocess.run([mrw, "instructions", "--core"], check=True, capture_output=True).stdout
        good = {"registered_guides": {"R": hashlib.sha256(core).hexdigest()}}
        self.assertEqual(grouped.appended_guide("R", mrw, good), core)
        with self.assertRaisesRegex(ValueError, "arm R's guide is not the registered one"):
            grouped.appended_guide("R", mrw, {"registered_guides": {"R": "0" * 64}})

    def test_a_plan_with_arms_acr_lists_r_cells(self):
        plan = grouped.paired_plan(["a", "b", "c"], "ACR")
        self.assertEqual([arms for _, arms in plan], ["ACR", "RCA", "ACR"])
        self.assertEqual([arms for _, arms in grouped.paired_plan(["a", "b", "c"], "ACR", reverse=True)], ["RCA", "ACR", "RCA"])
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "tasks/job").mkdir(parents=True)
            config = {"root": d, "corpus": str(Path(d) / "tasks"), "arms": "ACR", "round": 1}
            with mock.patch.object(grouped, "execute_cell", side_effect=ValueError("stop")) as launch:
                summary = grouped.run(config)
            self.assertEqual([(c.args[0].name, c.args[1]) for c in launch.call_args_list], [("canary", "A")])
            self.assertEqual(summary["plan"], [("job", "ACR")])
            self.assertEqual(summary["not_run"], [{"task": "job", "arm": arm} for arm in "ACR"])


class Compaction(unittest.TestCase):
    """compact_pct: a 100K window that compacts at that percent, and a hook that counts each compaction."""
    scratch = PurePosixPath("/scratch")

    def settings(self, arm, **config):
        return grouped.cell_settings(arm, dict({"bench": "/opt/bench"}, **config), "/runs/mrw", self.scratch, self.scratch / "tree")

    def test_without_compact_pct_no_cell_compacts(self):
        self.assertEqual(grouped.compaction_launch({}), ([], {}))
        self.assertEqual(self.settings("A"), {})
        for arm in "BFCR":
            self.assertNotIn("SessionStart", self.settings(arm)["hooks"], arm)

    def test_compact_pct_sets_the_window_the_threshold_and_a_counting_hook(self):
        self.assertEqual(grouped.compaction_launch({"compact_pct": 50}),
                         (["--autocompact", "100000"], {"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "50"}))
        count = "printf 'compacted\\n' >> /scratch/compacted.marker"
        self.assertEqual(self.settings("A", compact_pct=50),
                         {"hooks": {"SessionStart": [{"matcher": "compact", "hooks": [{"type": "command", "command": count}]}]}})
        # An MRW arm also forgets the ledger, with the run's own mrw (docs/agents.md section 8).
        for arm in "BFCR":
            hooks = self.settings(arm, compact_pct=50)["hooks"]
            self.assertEqual(hooks["SessionStart"], [{"matcher": "compact", "hooks": [{"type": "command", "command":
                             count + "; /runs/mrw --root /scratch/tree seen --forget"}]}], arm)
            self.assertEqual(hooks["PreToolUse"], self.settings(arm)["hooks"]["PreToolUse"], arm)
        self.assertEqual(self.settings("R", compact_pct=50)["hooks"]["PostToolUse"], self.settings("R")["hooks"]["PostToolUse"])

    def test_a_compact_pct_that_is_not_a_percent_is_refused(self):
        for bad in ("50", 50.0, True, 0, 101):
            with self.subTest(bad=bad), self.assertRaisesRegex(ValueError, "compact_pct"):
                grouped.compaction_launch({"compact_pct": bad})
            with self.subTest(bad=bad), self.assertRaisesRegex(ValueError, "compact_pct"):
                self.settings("C", compact_pct=bad)

    def test_compactions_count_the_marker_lines_and_zero_without_one(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertEqual(grouped.compactions(Path(d)), 0)
            (Path(d) / "compacted.marker").write_text("compacted\ncompacted\n")
            self.assertEqual(grouped.compactions(Path(d)), 2)


def compacting_stub(root, task):
    """A stand-in host for a compacting cell: it appends to the marker once, as the SessionStart hook would."""
    return ("#!" + sys.executable + "\n"
            "import json,os,subprocess,sys\n"
            "open(" + repr(str(root / "seen.json")) + ",'w').write(json.dumps("
            "{'pct':os.environ.get('CLAUDE_AUTOCOMPACT_PCT_OVERRIDE'),'argv':sys.argv}))\n"
            "open(os.path.join('..','compacted.marker'),'a').write('compacted\\n')\n"
            "subprocess.run(['git','apply'," + repr(str(task / "reference.patch")) + "],check=True)\n"
            "tools=sys.argv[sys.argv.index('--tools')+1].split(',')\n"
            "print(json.dumps({'type':'system','subtype':'init','model':'stub-model','tools':tools,'mcp_servers':[],'plugins':[]}))\n"
            "print(json.dumps({'type':'result','is_error':False,'num_turns':1,'total_cost_usd':.01,'usage':{'input_tokens':1,'output_tokens':1}}))\n")


class CompactingStub(unittest.TestCase):
    # Runs on every platform, so a stub that does not compile is caught where the cell cannot run.
    def test_the_compacting_stub_compiles_and_writes_one_marker_line_per_run(self):
        root = Path(tempfile.gettempdir())
        source = compacting_stub(root, root / "task")
        compile(source, "stub", "exec")
        marker = [line for line in source.splitlines() if "compacted.marker" in line]
        self.assertEqual(marker, ["open(os.path.join('..','compacted.marker'),'a').write('compacted\\n')"])
        # The pair: the defect this replaces, a real newline inside the literal, does not compile.
        with self.assertRaises(SyntaxError):
            compile(source.replace("compacted\\n", "compacted\n"), "stub", "exec")


@unittest.skipUnless(os.name == "posix", "pilot execution requires POSIX")
class CompactingCell(unittest.TestCase):
    def test_a_compacting_cell_launches_compacting_and_records_its_compactions(self):
        with tempfile.TemporaryDirectory(prefix="grouped-test-") as d:
            root = Path(d)
            task = Path(__file__).resolve().parents[1] / "bench/grouped-pilot/tasks/grouped-routing"
            stub = root / "stub"
            stub.write_text(compacting_stub(root, task))
            stub.chmod(0o755)
            for folder in ("evidence", "cells", "transcripts"):
                (root / folder).mkdir()
            adapter = {"claude": str(stub), "mrw": shutil.which("mrw"), "out": str(root / "evidence"), "max_launches": 12}
            (root / "adapter.json").write_text(json.dumps(adapter))
            config = {"root": str(root), "adapter": str(root / "adapter.json"), "bench": "/unused",
                      "model": "stub-model", "profile": str(root / "profile"), "compact_pct": 40}
            row = grouped.execute_cell(task, "A", config)
            self.assertEqual(row["compactions"], 1)
            seen = json.loads((root / "seen.json").read_text())
            self.assertEqual(seen["pct"], "40")
            self.assertEqual(seen["argv"][seen["argv"].index("--autocompact") + 1], "100000")
            self.assertEqual(json.loads((root / "cells/grouped-routing.A.json").read_text())["compactions"], 1)


if __name__ == "__main__":
    unittest.main()
