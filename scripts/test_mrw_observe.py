"""Behavioral checks for experimental MRW cost and warning reports."""
import importlib.util
import json
import os
import subprocess
import time
from unittest import mock
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


if __name__ == "__main__":
    unittest.main()
