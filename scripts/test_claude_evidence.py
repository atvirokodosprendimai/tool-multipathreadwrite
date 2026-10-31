"""Local controls for the experimental Claude evidence wrapper; never call a model."""
import json
import tempfile
import unittest
from pathlib import Path
import claude_evidence as pilot


class EvidenceGuard(unittest.TestCase):
    def test_unknown_cost_stops_the_next_paid_launch(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            pilot.save(root, "one", {"cost_usd": None})
            with self.assertRaisesRegex(ValueError, "unknown"):
                pilot.cost_guard(root)

    def test_known_cost_is_summed_and_budget_exhaustion_stops(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            for i in range(6):
                pilot.save(root, str(i), {"cost_usd": .5, "usage": {"input_tokens": 1, "output_tokens": 1}})
            with self.assertRaisesRegex(ValueError, "exhausted"):
                pilot.cost_guard(root)

    def test_known_cost_does_not_hide_unknown_usage(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            pilot.save(root, "one", {"cost_usd": .1, "usage": None})
            with self.assertRaisesRegex(ValueError, "unknown"):
                pilot.cost_guard(root)


    def test_report_extracts_model_usage_without_inventing_absent_cost(self):
        raw = b'{"type":"system","subtype":"init","model":"pinned","tools":["Bash"],"mcp_servers":[]}\n'
        row = pilot.transcript_meta(raw)
        self.assertEqual(row["actual_model"], "pinned")
        self.assertIsNone(row["cost_usd"])
        raw += b'{"type":"result","total_cost_usd":0.12,"usage":{"output_tokens":9}}\n'
        row = pilot.transcript_meta(raw)
        self.assertEqual(row["cost_usd"], .12)
        self.assertEqual(row["usage"]["output_tokens"], 9)

    def test_records_are_never_overwritten(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            pilot.save(root, "one", {"cost_usd": .1})
            with self.assertRaises(FileExistsError):
                pilot.save(root, "one", {"cost_usd": .2})
            self.assertEqual(json.loads((root / "one.json").read_text())["cost_usd"], .1)

    def test_unfinished_launch_with_unknown_cost_stops_future_calls(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "unfinished.A.started").touch()
            with self.assertRaisesRegex(ValueError, "unknown"):
                pilot.cost_guard(root)




class WrapperExecution(unittest.TestCase):
    def test_cli_output_budget_group_cleanup_and_observation_are_connected(self):
        import io
        import types
        import unittest.mock as mock
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "TASK.json").write_text('{"id":"pilot-fixture"}')
            out = root / "evidence"
            out.mkdir()
            raw = b'{"type":"system","subtype":"init","model":"pinned"}\n{"type":"result","total_cost_usd":0.1}\n'
            output = types.SimpleNamespace(buffer=io.BytesIO())
            with mock.patch.object(pilot.Path, "cwd", return_value=root), \
                 mock.patch.object(pilot.mrw_observe, "git_output", return_value=b"baseline\n"), \
                 mock.patch.object(pilot.mrw_observe, "verify_trial", return_value={"status":"verified"}) as verify, \
                 mock.patch.object(pilot.subprocess, "Popen") as spawn, \
                 mock.patch.object(pilot.os, "killpg") as kill, \
                 mock.patch.object(pilot.sys, "stdout", output):
                child = spawn.return_value.__enter__.return_value
                child.pid, child.returncode = 12345, 0
                child.communicate.return_value = (raw, None)
                self.assertEqual(pilot.wrap(["-p","repair"], {"claude":"/claude","mrw":"/mrw","out":str(out)}), 0)
            self.assertEqual(output.buffer.getvalue(), raw)
            self.assertEqual(spawn.call_args.args[0][-2:], ["--max-budget-usd","0.50"])
            kill.assert_called_once_with(12345, pilot.signal.SIGKILL)
            self.assertEqual(verify.call_args.args[3], ["accept.py","TASK.json"])
            self.assertEqual(json.loads((out/"pilot-fixture.A.json").read_text())["evidence"]["status"], "verified")


class IsolationSettings(unittest.TestCase):
    def test_isolation_keeps_the_declared_write_guard(self):
        guard = {"PreToolUse": [{"matcher":"Bash","hooks":[{"type":"command","command":"guard"}]}]}
        result = pilot.isolated_settings({"hooks": guard}, ["custom@synced"])
        self.assertEqual(result["hooks"], guard)
        self.assertEqual(result["claudeMdExcludes"], ["**"])
        self.assertFalse(result["enabledPlugins"]["custom@synced"])
        self.assertFalse(result["autoMemoryEnabled"])

    def test_partial_stream_keeps_known_metadata_and_unknown_cost(self):
        row = pilot.transcript_meta(b'{"type":"system","subtype":"init","model":"pinned"}\n{')
        self.assertEqual(row["actual_model"], "pinned")
        self.assertIsNone(row["cost_usd"])
        self.assertEqual(row["stream_errors"], 1)


class ExactTreeGrade(unittest.TestCase):
    def test_registered_patch_passes_and_any_extra_or_wrong_file_fails(self):
        import hashlib
        task = Path(__file__).resolve().parents[1] / "bench/evidence-pilot/tasks/pilot-singleline"
        expected = {str(p.relative_to(task/"base")): {"sha256":hashlib.sha256(p.read_bytes()).hexdigest(),"mode":"100644"}
                    for p in (task/"base").rglob("*") if p.is_file()}
        expected["settings.py"]["sha256"] = hashlib.sha256(b"TIMEOUT_SECONDS = 45\nRETRY_SECONDS = 30\n").hexdigest()
        self.assertTrue(pilot.exact_tree_grade(task, expected)["solved"])
        altered = dict(expected, extra={"sha256":"unexpected","mode":"100644"})
        self.assertFalse(pilot.exact_tree_grade(task, altered)["solved"])
        altered = dict(expected, **{"settings.py":{"sha256":"wrong","mode":"100644"}})
        self.assertFalse(pilot.exact_tree_grade(task, altered)["solved"])


    def test_larger_pilot_can_extend_the_launch_count_without_weakening_cost_guard(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            for i in range(8):
                pilot.save(root, str(i), {"cost_usd": .01, "usage": {"input_tokens": 1, "output_tokens": 1}})
            with self.assertRaisesRegex(ValueError, "exhausted"):
                pilot.cost_guard(root)
            pilot.cost_guard(root, max_launches=12)
            for i in range(8, 12):
                pilot.save(root, str(i), {"cost_usd": .01, "usage": {"input_tokens": 1, "output_tokens": 1}})
            with self.assertRaisesRegex(ValueError, "exhausted"):
                pilot.cost_guard(root, max_launches=12)


    def test_a_configured_ceiling_stops_below_three_dollars_and_cannot_raise_it(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            pilot.save(root, "one", {"cost_usd": 1.0, "usage": {"input_tokens": 1, "output_tokens": 1}})
            pilot.cost_guard(root, max_usd=1.01)
            with self.assertRaisesRegex(ValueError, "exhausted"):
                pilot.cost_guard(root, max_usd=1.0)
            for bad in (0, -1, 3.01, "2"):
                with self.subTest(bad=bad), self.assertRaisesRegex(ValueError, "max_usd"):
                    pilot.cost_guard(root, max_usd=bad)


class ArmNames(unittest.TestCase):
    def test_unnamed_calls_keep_the_a_b_inference(self):
        self.assertEqual(pilot.arm_for(["-p", "x"], None), "A")
        self.assertEqual(pilot.arm_for(["-p", "x", "--append-system-prompt-file", "f"], None), "B")

    def test_named_full_and_core_arms_need_an_appended_file(self):
        appended = ["-p", "x", "--append-system-prompt-file", "f"]
        self.assertEqual(pilot.arm_for(appended, "F"), "F")
        self.assertEqual(pilot.arm_for(appended, "C"), "C")
        for name, argv in (("F", ["-p", "x"]), ("C", ["-p", "x"]), ("B", appended), ("X", appended), ("", appended)):
            with self.subTest(name=name), self.assertRaises(ValueError):
                pilot.arm_for(argv, name)
if __name__ == "__main__":
    unittest.main()
