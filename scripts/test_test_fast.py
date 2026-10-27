"""Checks for the bounded, process-isolated local test runner."""
import contextlib
import importlib.util
import io
import os
import signal
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

_spec = importlib.util.spec_from_file_location("test_fast", Path(__file__).with_name("test-fast.py"))


@unittest.skipUnless(os.name == "posix", "the optional runner uses POSIX process groups")
class RunnerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.runner = importlib.util.module_from_spec(_spec)
        _spec.loader.exec_module(cls.runner)

    def run_commands(self, commands, timeout=10):
        with contextlib.redirect_stdout(io.StringIO()) as output:
            code = self.runner.run_commands(commands, timeout=timeout)
        return code, output.getvalue()

    def test_groups_overlap_in_separate_processes(self):
        with tempfile.TemporaryDirectory() as d:
            script = (
                "import pathlib,sys,time; "
                "p=pathlib.Path(sys.argv[1]); p.write_text('ready'); "
                "other=pathlib.Path(sys.argv[2]); deadline=time.monotonic()+3; "
                "\nwhile not other.exists() and time.monotonic()<deadline: time.sleep(.01)"
                "\nassert other.exists(), 'the other group never started'"
            )
            a, b = str(Path(d) / "a"), str(Path(d) / "b")
            code, _ = self.run_commands([
                ("a", [sys.executable, "-c", script, a, b]),
                ("b", [sys.executable, "-c", script, b, a]),
            ])
            self.assertEqual(code, 0)

    def test_either_failure_fails_even_when_output_says_pass(self):
        for failing in range(2):
            with self.subTest(failing=failing):
                commands = [
                    (str(i), [sys.executable, "-c", f"print('PASS'); raise SystemExit({7 if i == failing else 0})"])
                    for i in range(2)
                ]
                code, output = self.run_commands(commands)
                self.assertEqual(code, 1)
                self.assertEqual(output.splitlines().count("PASS"), 2)

    def test_timeout_kills_the_group_including_its_child(self):
        with tempfile.TemporaryDirectory() as d:
            pidfile = Path(d) / "child"
            script = (
                "import pathlib,subprocess,sys,time; "
                "p=subprocess.Popen([sys.executable,'-c','import time; time.sleep(60)']); "
                "pathlib.Path(sys.argv[1]).write_text(str(p.pid)); time.sleep(60)"
            )
            code, output = self.run_commands(
                [("hang", [sys.executable, "-c", script, str(pidfile)])], timeout=1
            )
            self.assertEqual(code, 1)
            self.assertIn("timed out", output)
            pid = int(pidfile.read_text())
            status = subprocess.run(
                ["ps", "-o", "stat=", "-p", str(pid)], capture_output=True, text=True, check=False
            )
            self.assertTrue(status.returncode != 0 or status.stdout.strip().startswith("Z"), status.stdout)

    def test_a_command_that_cannot_start_fails(self):
        code, output = self.run_commands([("missing", ["/no/such/mrw-test-command"])])
        self.assertEqual(code, 1)
        self.assertIn("cannot finish", output)

    def test_sigterm_stops_the_running_group(self):
        script = """
import importlib.util, subprocess, sys
spec = importlib.util.spec_from_file_location("runner", sys.argv[1])
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)
real_popen = subprocess.Popen
def started(*args, **kwargs):
    p = real_popen(*args, **kwargs)
    print("childpid=" + str(p.pid), flush=True)
    return p
subprocess.Popen = started
raise SystemExit(runner.run_commands([("sleep", [sys.executable, "-c", "import time; time.sleep(60)"])]))
"""
        p = subprocess.Popen(
            [sys.executable, "-c", script, str(Path(__file__).with_name("test-fast.py"))],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
        )
        child = None
        try:
            for _ in range(2):
                line = p.stdout.readline()
                if line.startswith("childpid="):
                    child = int(line.removeprefix("childpid="))
            self.assertIsNotNone(child)
            p.terminate()
            output, _ = p.communicate(timeout=8)
            self.assertEqual(p.returncode, 1, output)
            status = subprocess.run(
                ["ps", "-o", "stat=", "-p", str(child)], capture_output=True, text=True, check=False
            )
            self.assertTrue(status.returncode != 0 or status.stdout.strip().startswith("Z"), status.stdout)
        finally:
            if p.poll() is None:
                p.kill()
                p.wait()
            if child is not None:
                with contextlib.suppress(ProcessLookupError):
                    os.killpg(child, signal.SIGKILL)


if __name__ == "__main__":
    unittest.main()
