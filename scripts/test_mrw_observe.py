"""Behavioral checks for experimental MRW cost and warning reports."""
import importlib.util
import json
import os
import subprocess
import time
import shutil
import unittest.mock as mock
import sys
from pathlib import Path
import unittest

SPEC = importlib.util.spec_from_file_location("mrw_observe", Path(__file__).with_name("mrw_observe.py"))
observe = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(observe)


def sample(size, wall, code=0, binary="A", workload="R2"):
    return {"workload": workload, "size": str(size), "binary": binary,
            "wall_ns": wall, "cpu_ns": wall // 2, "stdout_bytes": 3,
            "stderr_bytes": 4, "exit": code}


class NativeReport(unittest.TestCase):
    def test_percentiles_and_ratios_use_sorted_values_and_sizes(self):
        rows = [sample(10, 400), sample(1, 100), sample(10, 200), sample(1, 200)]
        groups = observe.summarize(rows)
        self.assertEqual([g["size"] for g in groups], ["1", "10"])
        self.assertEqual(groups[0]["p50_ns"], 100)
        self.assertEqual(groups[0]["p95_ns"], 200)
        self.assertEqual(groups[1]["p50_vs_smallest"], 2)
        self.assertEqual(groups[1]["growth_vs_previous"], 0.2)
        self.assertEqual(groups[1]["n"], 2)

    def test_failures_remain_in_distribution_but_not_scaling_claims(self):
        groups = observe.summarize([sample(1, 10), sample(10, 100, 3)])
        self.assertEqual(groups[1]["failed"], 1)
        self.assertEqual(groups[1]["p50_ns"], 100)
        self.assertIsNone(groups[1]["growth_vs_previous"])
        self.assertIsNone(groups[1]["p50_vs_smallest"])

    def test_zero_units_and_mixed_dimensions_are_not_growth_ratios(self):
        rows = [sample(0, 10, workload="L1"), sample(1000, 20, workload="L1"),
                sample("1h/1f", 10, workload="W1"), sample("50h/10f", 50, workload="W1")]
        self.assertTrue(all(g["growth_vs_previous"] is None for g in observe.summarize(rows)))

    def test_unrecorded_legacy_stderr_is_unknown_not_zero(self):
        row = sample(1, 100)
        del row["stderr_bytes"]
        for rows in [[row], [row, sample(1, 200)]]:
            self.assertIsNone(observe.summarize(rows)[0]["stderr_bytes"])

    def test_binary_baselines_do_not_mix(self):
        rows = [sample(1, 100), sample(10, 200), sample(1, 1000, binary="B"), sample(10, 4000, binary="B")]
        ratios = {(g["binary"], g["size"]): g["p50_vs_smallest"] for g in observe.summarize(rows)}
        self.assertEqual(ratios[("A", "10")], 2)
        self.assertEqual(ratios[("B", "10")], 4)

    def test_bad_raw_data_is_rejected_instead_of_becoming_zero(self):
        for bad in [-1, float("nan"), True, "100"]:
            row = sample(1, 100)
            row["wall_ns"] = bad
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                observe.summarize([row])
        with self.assertRaises(ValueError):
            observe.summarize([])


    def test_missing_fields_and_nonobjects_are_rejected(self):
        for row in [None, [], {}, {"workload": "R1"}, dict(sample(1, 100), cpu_ns=None)]:
            with self.subTest(row=row), self.assertRaises(ValueError):
                observe.summarize([row])


class CheckerCoverage(unittest.TestCase):
    def test_actual_pass_and_coverage_are_required(self):
        good = {"ran": True, "exit_code": 0, "tail": ["checked 100"]}
        self.assertEqual(observe.checker_coverage(json.dumps({"check": good})), 100)
        for bad in [dict(good, ran=False), dict(good, exit_code=1), dict(good, tail=[]),
                    dict(good, tail=["checked 1", "checked 100"])]:
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                observe.checker_coverage(json.dumps(bad))


@unittest.skipUnless(os.name == "posix", "live collection requires POSIX")
class ChildBoundaries(unittest.TestCase):
    def test_timeout_and_interruption_keep_output_and_stop_the_child(self):
        for error, code in [(subprocess.TimeoutExpired("mrw", 1), 124), (KeyboardInterrupt(), 130)]:
            with self.subTest(code=code), mock.patch.object(observe.subprocess, "Popen") as spawn, mock.patch.object(observe.os, "killpg") as kill:
                child = spawn.return_value.__enter__.return_value
                child.pid, child.returncode = 12345, -2
                child.communicate.side_effect = [error, (b"partial", b"diagnostic")]
                calls = observe.Calls("mrw", Path("."), time.monotonic() + 5, 1)
                record, output = calls.call(["check"])
                self.assertEqual(record["exit"], code)
                self.assertEqual(output, b"partial")
                self.assertEqual(record["stderr_bytes"], 10)
                kill.assert_called_once_with(12345, observe.signal.SIGINT)

    def test_expired_budget_starts_no_process(self):
        calls = observe.Calls("mrw", Path("."), time.monotonic() - 1, 1)
        with mock.patch.object(observe.subprocess, "Popen") as spawn, self.assertRaises(ValueError):
            calls.call(["check"])
        spawn.assert_not_called()


class WarningEvidence(unittest.TestCase):
    def test_evidence_comes_from_the_receipts_actual_match(self):
        result = observe.warning_evidence("line 6 repeats the body's last line: )", ["old"],
                                          ["new", ")"], ["new", ")", "", "sibling", "", ")"], 1)
        self.assertEqual(result["rule"], "ADR-119")
        self.assertEqual(result["matched_line"], 6)
        self.assertEqual(result["following_nonblank_distance"], 2)
        self.assertEqual(result["body_last"], ")")
        self.assertFalse(result["old_ended_in_body_last"])

    def test_absent_warning_is_not_an_invented_explanation(self):
        result = observe.warning_evidence("", ["old"], ["new"], ["new", "tail"], 1)
        self.assertEqual(result["status"], "not_reported")
        self.assertIsNone(result["matched_line"])

    def test_inconsistent_receipt_evidence_is_refused(self):
        for receipt in ["line 99 repeats the body's last line: )", "line 2 repeats the body's last line: )", "unknown hint"]:
            with self.subTest(receipt=receipt), self.assertRaises(ValueError):
                observe.warning_evidence(receipt, ["old"], ["new", ")"], ["new", ")", "tail"], 1)



@unittest.skipUnless(os.name == "posix", "verification requires POSIX")
class TrialVerification(unittest.TestCase):
    def setUp(self):
        import tempfile
        self.temp = self.enterContext(tempfile.TemporaryDirectory(prefix="mrw-evidence-test-"))
        self.root = Path(self.temp) / "repo"
        subprocess.run(["git", "init", "-q", str(self.root)], check=True, capture_output=True, timeout=5)
        self.enterContext(mock.patch.dict(os.environ, {"XDG_STATE_HOME": str(Path(self.temp) / "state")}))
        self.git("config", "user.name", "Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        # Fixture source bytes are authored through mrw too.
        files = {"task.src": "old\n", "instructions.md": "Use the checked baseline.\n",
                 "checker.sh": "test \"$(cat task.src)\" = new\n",
                 ".quality-harness.json": json.dumps({"check": "sh checker.sh", "timeout_seconds": 2}) + "\n"}
        self.write(files, create=True)
        self.git("add", ".")
        self.git("commit", "-qm", "synthetic baseline")
        self.baseline = self.git("rev-parse", "HEAD").decode().strip()
        self.write({"task.src": "new\n"})

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.root, check=True, capture_output=True, timeout=5).stdout

    def write(self, files, create=False):
        binary = shutil.which("mrw")
        if not binary:
            self.fail("mrw must be installed for the evidence integration controls")
        if not create:
            subprocess.run([binary, "--root", str(self.root), "read", "--quiet", *files],
                           check=True, capture_output=True, timeout=5)
        plan = "".join(f"@@ {name} {'0 create' if create else '1-$ replace'} body={len(text.splitlines())}\n{text}"
                       for name, text in files.items())
        subprocess.run([binary, "--root", str(self.root), "write", "--no-check", "-"],
                       input=plan.encode(), check=True, capture_output=True, timeout=5)

    def verify(self):
        return observe.verify_trial("mrw", self.root, self.baseline, ["checker.sh"], ["instructions.md"], 5)

    def test_clean_final_check_binds_the_actual_tree_and_instructions(self):
        result = self.verify()
        self.assertEqual(result["status"], "verified")
        self.assertTrue(result["check"]["receipt"]["ran"])
        self.assertEqual(result["check"]["receipt"]["exit_code"], 0)
        self.assertEqual(result["before"]["tree_sha256"], result["after"]["tree_sha256"])
        self.assertEqual(result["before"]["files"]["task.src"]["sha256"], observe.hashlib.sha256(b"new\n").hexdigest())
        self.assertIn("sha256", result["instructions"]["instructions.md"])
        self.assertEqual(result["performance_verdict"], "not_attempted")

    def test_modified_committed_hidden_or_deleted_gate_never_runs(self):
        for kind in ("working", "committed", "hidden", "deleted", "config"):
            with self.subTest(kind=kind):
                self.git("reset", "--hard", self.baseline)
                self.write({"task.src": "new\n", "checker.sh": "exit 0\n"})
                if kind == "committed":
                    self.git("add", ".")
                    self.git("commit", "-qm", "altered gate")
                if kind == "hidden":
                    self.git("update-index", "--assume-unchanged", "checker.sh")
                if kind == "deleted":
                    subprocess.run(["mrw", "--root", str(self.root), "write", "--no-check", "-"],
                                   input=b"@@ checker.sh - unlink\n", capture_output=True, check=True, timeout=5)
                if kind == "config":
                    self.write({"checker.sh": "test \"$(cat task.src)\" = new\n",
                                ".quality-harness.json": '{"check":"true"}\n'})
                result = self.verify()
                self.assertEqual(result["status"], "rejected")
                self.assertIsNone(result["check"])
                self.assertTrue(result["gate_changes"])
                self.git("update-index", "--no-assume-unchanged", "checker.sh")

    def test_failed_check_and_check_that_changes_source_are_rejected(self):
        self.write({"task.src": "bad\n"})
        result = self.verify()
        self.assertEqual(result["status"], "rejected")
        self.assertNotEqual(result["check"]["exit"], 0)
        # A baseline gate may itself write; even an exit-zero check cannot attest a different tree.
        self.write({"checker.sh": "printf 'drift\\n' | mrw write --no-check --create extra.src\n"})
        self.git("add", "checker.sh")
        self.git("commit", "-qm", "mutating gate control")
        self.baseline = self.git("rev-parse", "HEAD").decode().strip()
        result = self.verify()
        self.assertEqual(result["check"]["exit"], 0)
        self.assertEqual(result["status"], "rejected")
        self.assertIn("tree changed during verification", result["problems"])

    def test_untracked_files_affect_identity_and_symlinks_are_refused(self):
        first = self.verify()
        self.write({"extra.src": "extra\n"}, create=True)
        second = self.verify()
        self.assertNotEqual(first["before"]["tree_sha256"], second["before"]["tree_sha256"])
        os.symlink("task.src", self.root / "link.src")
        with self.assertRaisesRegex(ValueError, "symlink"):
            self.verify()

    def test_cli_records_success_and_gate_rejection_with_distinct_exits(self):
        argv = [sys.executable, str(Path(observe.__file__)), "verify", "--root", str(self.root),
                "--baseline", self.baseline, "--gate", "checker.sh", "--instruction", "instructions.md"]
        for changed, code, status in [(False, 0, "verified"), (True, 1, "rejected")]:
            if changed:
                self.write({"checker.sh": "exit 0\n"})
            run = subprocess.run(argv, capture_output=True, timeout=10)
            self.assertEqual(run.returncode, code, run.stderr.decode())
            self.assertEqual(json.loads(run.stdout)["status"], status)

    def test_interrupted_or_malformed_check_receipt_remains_rejected_evidence(self):
        for code, output in [(130, b"{"), (0, b"{"), (0, b"[]"),
                             (0, b'{"ran":false,"exit_code":0}'),
                             (0, b'{"ran":true,"exit_code":1}'),
                             (0, b'{"ran":true,"exit_code":false}')]:
            with self.subTest(code=code, output=output), mock.patch.object(observe.Calls, "call", return_value=(
                    {"exit": code, "timed_out": False}, output)):
                result = self.verify()
                self.assertEqual(result["status"], "rejected")
                self.assertEqual(result["check"]["exit"], code)
                self.assertTrue(result["problems"])

    def test_instruction_drift_is_detected_even_when_the_check_passes(self):
        # Use an explicit read in the gate: verification gets its own fresh ledger.
        self.write({"checker.sh": "mrw read --quiet instructions.md\nprintf '@@ instructions.md 1 replace\\nchanged\\n' | mrw write --no-check -\n"})
        self.git("add", "checker.sh")
        self.git("commit", "-qm", "instruction drift control")
        self.baseline = self.git("rev-parse", "HEAD").decode().strip()
        result = self.verify()
        self.assertEqual(result["check"]["exit"], 0)
        self.assertEqual(result["status"], "rejected")
        self.assertIn("instructions changed during verification", result["problems"])



    def test_gate_owner_execute_flag_matches_the_git_baseline(self):
        (self.root / "checker.sh").chmod(0o755)
        self.git("add", "checker.sh")
        self.git("commit", "-qm", "executable gate baseline")
        self.baseline = self.git("rev-parse", "HEAD").decode().strip()
        self.assertEqual(self.verify()["status"], "verified")
        # Filesystem metadata control: retain g+x while removing u+x.
        (self.root / "checker.sh").chmod(0o654)
        result = self.verify()
        self.assertEqual(result["status"], "rejected")
        self.assertIn("checker.sh", result["gate_changes"])
        self.assertIsNone(result["check"])

if __name__ == "__main__":
    unittest.main()
