"""Run the oracle pool beside the remaining Go tests on macOS or Linux.

The oracle parent waits for its parallel children before later tests can run.
A separate process overlaps that pool without sharing environment or signal state.
"""
import contextlib
import os
from pathlib import Path
import shutil
import signal
import sys
import subprocess
import tempfile
import time

ORACLE = "TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit"
PATTERN = f"^{ORACLE}$"


def run_commands(commands, timeout=240, required=None):
    """Run bounded process groups; output and exit codes survive either failure."""
    def interrupted(_signum, _frame):
        raise KeyboardInterrupt

    jobs = []
    with contextlib.ExitStack() as stack:
        previous = signal.signal(signal.SIGTERM, interrupted)
        stack.callback(signal.signal, signal.SIGTERM, previous)
        try:
            for name, args in commands:
                log = stack.enter_context(tempfile.TemporaryFile(mode="w+t", encoding="utf-8"))
                proc = subprocess.Popen(args, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
                jobs.append((name, proc, log))
                print(f"starting {name}: {' '.join(args)}", flush=True)
            deadline = time.monotonic() + timeout
            failed = False
            for name, proc, log in jobs:
                code = proc.wait(timeout=max(0, deadline - time.monotonic()))
                print(f"== {name}: exit {code}", flush=True)
                log.seek(0)
                if required is not None and required not in log.read().splitlines():
                    print(f"test-fast: missing expected test {required}")
                    failed = True
                log.seek(0)
                shutil.copyfileobj(log, sys.stdout)
                sys.stdout.flush()
                failed |= code != 0
            return int(failed)
        except (OSError, subprocess.TimeoutExpired, KeyboardInterrupt) as err:
            signal.signal(signal.SIGTERM, signal.SIG_IGN)
            print(f"test-fast: cannot finish: {err or 'interrupted'}", flush=True)
            # Each group owns its descendants, including Go compilers and oracle checks.
            for _, proc, _ in jobs:
                with contextlib.suppress(ProcessLookupError):
                    os.killpg(proc.pid, signal.SIGTERM)
            for _, proc, _ in jobs:
                with contextlib.suppress(subprocess.TimeoutExpired):
                    proc.wait(timeout=3)
                with contextlib.suppress(ProcessLookupError):
                    os.killpg(proc.pid, signal.SIGKILL)
                proc.wait()
            for name, _, log in jobs:
                print(f"== {name}: stopped", flush=True)
                log.seek(0)
                shutil.copyfileobj(log, sys.stdout)
            return 1


def main():
    """Keep the default gate intact and offer a measured local alternative."""
    if os.name != "posix":
        print("test-fast: use go test ./... on Windows")
        return 1
    os.chdir(Path(__file__).resolve().parent.parent)
    # Two go commands, each with two package workers and two runtime workers.
    # The oracle package still limits expensive cases to its existing four slots.
    os.environ["GOMAXPROCS"] = "2"
    if run_commands([
        ("list", ["go", "test", "-p=2", "-list", PATTERN, "./internal/bench"]),
    ], timeout=45, required=ORACLE):
        return 1
    return run_commands([
        ("oracles", ["go", "test", "-p=2", "-parallel=4", "-timeout=4m", "-run", PATTERN, "./internal/bench"]),
        ("remaining", ["go", "test", "-p=2", "-parallel=4", "-timeout=4m", "-skip", PATTERN, "./..."]),
    ])


if __name__ == "__main__":
    raise SystemExit(main())
