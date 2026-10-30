#!/usr/bin/env python3
"""Experimental, local-only MRW measurements; not a stable mrw receipt format.

native reads existing JSONL without running commands. costs and warnings use
synthetic temporary fixtures through mrw. JSON goes to stdout, failures included.
"""
import argparse
from collections import defaultdict
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time


UNITS = {"R1": "lines", "R2": "files", "W2": "sites", "L1": "ledger_entries"}


def percentile(values, percent):
    """Nearest rank, including failed samples rather than selecting fast successes."""
    ordered = sorted(values)
    return ordered[max(0, math.ceil(percent * len(ordered) / 100) - 1)]


def summarize(samples):
    """Describe samples; only clean positive single-dimension cells get ratios."""
    if not samples:
        raise ValueError("no samples")
    cells = defaultdict(list)
    for row in samples:
        if not isinstance(row, dict):
            raise ValueError("sample must be an object")
        for field in ("wall_ns", "cpu_ns", "stdout_bytes", "stderr_bytes"):
            value = row.get(field, 0 if field == "stderr_bytes" else None)
            if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or value < 0:
                raise ValueError(f"invalid {field}")
        if type(row.get("exit")) is not int:
            raise ValueError("invalid exit")
        for field in ("workload", "size", "binary"):
            if not isinstance(row.get(field), str) or not row[field]:
                raise ValueError(f"invalid {field}")
        cells[row["workload"], row["binary"], row["size"]].append(row)
    reports = []
    for workload, binary in sorted({(w, b) for w, b, _ in cells}):
        keys = [key for key in cells if key[:2] == (workload, binary)]
        keys.sort(key=lambda key: (not key[2].isdigit(), int(key[2]) if key[2].isdigit() else key[2]))
        baseline = previous = None
        for key in keys:
            rows = cells[key]
            wall = [s["wall_ns"] for s in rows]
            report = {"workload": workload, "binary": binary, "size": key[2], "n": len(rows),
                      "failed": sum(s["exit"] != 0 or s.get("valid") is False for s in rows),
                      "stdout_bytes": [min(s["stdout_bytes"] for s in rows), max(s["stdout_bytes"] for s in rows)],
                      "stderr_bytes": ([min(s["stderr_bytes"] for s in rows), max(s["stderr_bytes"] for s in rows)]
                                       if all("stderr_bytes" in s for s in rows) else None),
                      "cpu_p50_ns": percentile([s["cpu_ns"] for s in rows], 50), "max_ns": max(wall)}
            report.update({f"p{p}_ns": percentile(wall, p) for p in (25, 50, 75, 90, 95)})
            numeric = int(key[2]) if key[2].isdigit() else None
            unit = UNITS.get(workload, "edited_files" if workload.startswith("cost:") else None)
            report.update(units=unit, quantity=numeric if unit else None,
                          p50_vs_smallest=None, growth_vs_previous=None)
            if baseline is None:
                baseline = report
            if unit and numeric and not report["failed"]:
                if baseline["quantity"] and not baseline["failed"] and baseline["p50_ns"]:
                    report["p50_vs_smallest"] = report["p50_ns"] / baseline["p50_ns"]
                if previous and previous["quantity"] and not previous["failed"] and previous["p50_ns"]:
                    report["growth_vs_previous"] = (report["p50_ns"] / previous["p50_ns"]) / (numeric / previous["quantity"])
            reports.append(report)
            previous = report
    return reports


def warning_evidence(closer, replaced, body, written, start):
    """Validate the actual receipt's match, without reimplementing the heuristic."""
    last = next((line.strip() for line in reversed(body) if line.strip()), "")
    old_last = next((line.strip() for line in reversed(replaced) if line.strip()), "")
    result = {"rule": "ADR-119", "status": "reported" if closer else "not_reported",
              "body_last": last, "old_ended_in_body_last": old_last == last,
              "matched_line": None, "following_nonblank_distance": None}
    if not closer:
        return result
    match = re.fullmatch(r"line (\d+) repeats the body's last line: (.+)", closer)
    if not match:
        raise ValueError("unrecognized closer receipt")
    line = int(match[1])
    body_end = start - 1 + len(body)
    if not body_end < line <= len(written) or written[line - 1].strip() != match[2] or match[2] != last:
        raise ValueError("closer receipt does not match the written file")
    result.update(matched_line=line, following_nonblank_distance=sum(bool(s.strip()) for s in written[body_end:line]))
    return result


def metadata(binary):
    resolved = Path(shutil.which(binary) or binary).resolve(strict=True)
    version = subprocess.run([str(resolved), "version"], check=True, capture_output=True, timeout=5).stdout.decode().strip()
    revision = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, timeout=5, cwd=Path(__file__).resolve().parents[1])
    dirty = subprocess.run(["git", "status", "--porcelain"], capture_output=True, timeout=5, cwd=Path(__file__).resolve().parents[1])
    return resolved, {"binary": {"name": resolved.name, "sha256": hashlib.sha256(resolved.read_bytes()).hexdigest(), "version": version},
                      "script_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                      "source_revision": revision.stdout.decode().strip(), "source_dirty": bool(dirty.stdout),
                      "python": platform.python_version(), "os": platform.platform(), "arch": platform.machine(),
                      "cpus": os.cpu_count(), "load_before": os.getloadavg(),
                      "quantile": "nearest-rank: ceil(p*n/100)-1", "performance_verdict": "not_attempted"}


class Calls:
    """Own bounded subprocesses and an isolated ledger for one disposable fixture."""
    def __init__(self, binary, root, deadline, timeout):
        self.binary, self.root, self.deadline, self.timeout = binary, root, deadline, timeout
        self.environment = dict(os.environ, XDG_STATE_HOME=str(root / "state"), MRW_STEP_DEPTH="0", MRW_WRITE_LOCK_TIMEOUT="0")

    def call(self, args, data=None):
        import resource  # Collection is POSIX-only; offline native analysis is portable.
        remaining = min(self.timeout, self.deadline - time.monotonic())
        if remaining <= 0:
            raise ValueError("collection budget exhausted")
        before = resource.getrusage(resource.RUSAGE_CHILDREN)
        start = time.monotonic_ns()
        with subprocess.Popen([str(self.binary), "--root", str(self.root), *args], stdin=subprocess.PIPE,
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=self.environment,
                              start_new_session=True) as child:
            timed_out = interrupted = False
            try:
                stdout, stderr = child.communicate(data.encode() if data is not None else None, timeout=remaining)
            except (subprocess.TimeoutExpired, KeyboardInterrupt) as error:
                timed_out = isinstance(error, subprocess.TimeoutExpired)
                interrupted = isinstance(error, KeyboardInterrupt)
                # SIGINT lets mrw cancel its own check's separate process group.
                try:
                    os.killpg(child.pid, signal.SIGINT)
                except ProcessLookupError:
                    pass
                try:
                    stdout, stderr = child.communicate(timeout=3)
                except subprocess.TimeoutExpired:
                    os.killpg(child.pid, signal.SIGKILL)
                    stdout, stderr = child.communicate()
        after = resource.getrusage(resource.RUSAGE_CHILDREN)
        record = {"args": args, "wall_ns": time.monotonic_ns() - start,
                  "cpu_ns": max(0, round((after.ru_utime + after.ru_stime - before.ru_utime - before.ru_stime) * 1e9)),
                  "stdout_bytes": len(stdout), "stderr_bytes": len(stderr), "exit": 130 if interrupted else (124 if timed_out else child.returncode),
                  "timed_out": timed_out}
        return record, stdout

    def require(self, args, data=None):
        record, stdout = self.call(args, data)
        if record["exit"]:
            raise ValueError(f"fixture command {args[0]} exited {record['exit']}")
        return stdout

    def create(self, files):
        plan = "".join(f"@@ {name} 0 create body={len(text.splitlines())} raw=true\n{text}" for name, text in files.items())
        self.require(["write", "--no-check", "-"], plan)


def checker_coverage(stdout):
    receipt = json.loads(stdout)
    check = receipt.get("check", receipt)
    if not check.get("ran") or check.get("exit_code") != 0:
        raise ValueError("checker did not run and pass")
    tail = check.get("tail", [])
    matches = [re.fullmatch(r"checked (\d+)", line) for line in tail]
    counts = [int(m[1]) for m in matches if m]
    if len(counts) != 1:
        raise ValueError("checker did not report coverage")
    return counts[0]


CHECKER = "awk 'NF != 1 || ($0 != \"old\" && $0 != \"new\") { exit 1 } END { print \"checked \" ARGC-1 }' \"$@\"\n"
MODES = ("read", "write_no_check", "write_scoped", "write_full", "check_scoped", "check_full", "queue_full")


def collect_costs(binary, sizes, runs, warmups, budget, timeout, max_load):
    binary, meta = metadata(binary)
    meta.update(sizes=sizes, runs=runs, warmups=warmups, budget_seconds=budget, timeout_seconds=timeout,
                max_load=max_load, corpus_files=100, cpu_measure="RUSAGE_CHILDREN delta; includes waited descendants")
    samples, discarded = [], []
    result = {"kind": "costs", "metadata": meta, "samples": samples, "warmups": discarded, "status": "complete"}
    deadline = time.monotonic() + budget
    with tempfile.TemporaryDirectory(prefix="mrw-costs-") as temp:
        root = Path(temp)
        calls = Calls(binary, root, deadline, timeout)
        names = [f"f{i:03d}.src" for i in range(100)]
        config = {"check": "sh checker.sh f*.src", "scoped_check": "sh checker.sh {files}", "timeout_seconds": 2}
        try:
            calls.create({**dict.fromkeys(names, "old\n"), "checker.sh": CHECKER,
                          ".quality-harness.json": json.dumps(config) + "\n"})
            for size in sizes:
                load = os.getloadavg()[0]
                if load >= max_load:
                    raise ValueError("load gate refused collection")
                for mode in MODES:
                    for run in range(-warmups, runs):
                        scoped = mode not in ("write_full", "check_full")
                        current_config = config if scoped else {"check": config["check"], "timeout_seconds": 2}
                        reset = "".join(f"@@ {name} 1 replace\nold\n" for name in names)
                        config_body = json.dumps(current_config)
                        reset += f"@@ .quality-harness.json 1 replace\n{config_body}\n"
                        calls.require(["write", "--no-check", "-"], reset)
                        edited = names[:size]
                        calls.require(["read", "--quiet", *edited])
                        plan = "".join(f"@@ {name} 1 replace\nnew\n" for name in edited)
                        if mode.startswith("check_"):
                            calls.require(["write", "--no-check", "-"], plan)
                        commands = []
                        row = {"workload": "cost:" + mode, "size": str(size), "binary": "installed", "run": run,
                               "commands": commands, "load_before": os.getloadavg()[0]}
                        (discarded if run < 0 else samples).append(row)

                        def timed(args, data=None, coverage=None):
                            record, output = calls.call(args, data)
                            commands.append(record)
                            if record["exit"]:
                                raise ValueError(f"timed command exited {record['exit']}")
                            if coverage is not None:
                                record["checked_files"] = checker_coverage(output)
                                if record["checked_files"] != coverage:
                                    raise ValueError("checker coverage differs from requested scope")

                        try:
                            if mode == "read":
                                timed(["read", *edited])
                            elif mode == "queue_full":
                                for name in edited:
                                    timed(["write", "--json", "--no-check", "-"], f"@@ {name} 1 replace\nnew\n")
                                timed(["check", "--json", "--full"], coverage=100)
                            elif mode.startswith("check_"):
                                timed(["check", "--json", *(edited if scoped else ["--full"])], coverage=size if scoped else 100)
                            else:
                                args = ["write", "--json"] + (["--no-check"] if mode == "write_no_check" else []) + ["-"]
                                timed(args, plan, coverage=None if mode == "write_no_check" else (size if scoped else 100))
                            if mode not in ("write_full", "check_full", "queue_full"):
                                verification, output = calls.call(["check", "--json", "--full"])
                                row["final_verification"] = verification
                                verification["checked_files"] = checker_coverage(output) if not verification["exit"] else 0
                                if verification["exit"] or verification["checked_files"] != 100:
                                    raise ValueError("final full verification failed")
                            expected = "old\n" if mode == "read" else "new\n"
                            if any((root / name).read_text() != (expected if name in edited else "old\n") for name in names):
                                raise ValueError("written contents differ from the registered edit")
                        except (ValueError, OSError, subprocess.SubprocessError, KeyboardInterrupt):
                            row["valid"] = False
                            raise
                        finally:
                            final = row.get("final_verification", {})
                            for key in ("wall_ns", "cpu_ns", "stdout_bytes", "stderr_bytes"):
                                row[key] = sum(c[key] for c in commands)
                                row["workflow_" + key] = row[key] + final.get(key, 0)
                            row["exit"] = next((c["exit"] for c in commands if c["exit"]), 0)
                            row["load_after"] = os.getloadavg()[0]
        except (ValueError, OSError, subprocess.SubprocessError, KeyboardInterrupt) as error:
            result.update(status="incomplete", error="interrupted" if isinstance(error, KeyboardInterrupt) else str(error))
    meta["load_after"] = os.getloadavg()
    result["groups"] = summarize(samples) if samples else []
    return result


def warning_cases():
    sibling = ['RSpec.describe X do', '  it "a" do', '    expect(A).to(receive(:call).and_call_original)', '',
               '    expect { run }.to(', '      change { y }', '    )', '  end', 'end']
    return [
        ("sibling_spec.rb", sibling, 3, 3, ['    expect(A).to(', '      receive(:call).', '        and_call_original', '    )'], False),
        ("duplicate.md", ['intro', '```', 'body', '```', 'end'], 3, 3, ['changed', '```'], True),
        ("statement.src", ['old()', 'work()'], 1, 1, ['work()'], False),
        ("nested.go", ['func f() {', '  old()', '}'], 2, 2, ['  if ready {', '    work()', '  }'], False),
        ("blanks.md", ['intro', '```', 'body', '', '', '```'], 3, 3, ['changed', '```'], True),
        ("no_repeat.src", ['old', 'tail'], 1, 1, ['new', ')'], False),
        ("own_closer.src", ['old', ')', ')'], 1, 2, ['new', ')'], False),
        ("outside_window.src", ['old', 'a', 'b', 'c', 'd', ')'], 1, 1, ['new', ')'], False),
    ]


def collect_warnings(binary, timeout):
    binary, meta = metadata(binary)
    rows = []
    result = {"kind": "warnings", "metadata": meta, "cases": rows, "status": "complete"}
    with tempfile.TemporaryDirectory(prefix="mrw-warnings-") as temp:
        root = Path(temp)
        calls = Calls(binary, root, time.monotonic() + 60, timeout)
        try:
            for name, old, start, end, body, expected in warning_cases():
                calls.create({name: "\n".join(old) + "\n"})
                calls.require(["read", name])
                plan = f"@@ {name} {start}-{end} replace anchor={json.dumps(old[start - 1])}\n" + "\n".join(body) + "\n"
                record, output = calls.call(["write", "--json", "--no-check", "-"], plan)
                row = {"fixture": name, "span": [start, end], "body": body, "expected_hint": expected, "write": record}
                rows.append(row)
                if record["exit"]:
                    raise ValueError("warning fixture write failed")
                written = (root / name).read_text().splitlines()
                if written != old[:start - 1] + body + old[end:]:
                    raise ValueError("warning fixture content differs")
                receipt = json.loads(output)
                hunk, = receipt["hunks"]
                closer = hunk.get("closer", "")
                row.update(actual_hint=bool(closer), closer=closer,
                           evidence=warning_evidence(closer, old[start - 1:end], body, written, start))
        except (ValueError, OSError, KeyError, subprocess.SubprocessError, KeyboardInterrupt) as error:
            result.update(status="incomplete", error="interrupted" if isinstance(error, KeyboardInterrupt) else str(error))
    result["mismatches"] = sum(r.get("actual_hint") != r["expected_hint"] for r in rows)
    result["quality"] = "pass" if result["status"] == "complete" and result["mismatches"] == 0 else "fail"
    result["confusion"] = {name: sum(r.get("actual_hint") is got and r["expected_hint"] is want for r in rows)
                           for name, got, want in [("true_positive", True, True), ("false_positive", True, False),
                                                   ("true_negative", False, False), ("false_negative", False, True)]}
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    native = sub.add_parser("native", help="summarize existing nativebench JSONL; no execution")
    native.add_argument("samples", type=Path)
    for command in ("costs", "warnings"):
        child = sub.add_parser(command)
        child.add_argument("--mrw", default="mrw")
        child.add_argument("--timeout", type=float, default=5)
        if command == "costs":
            child.add_argument("--runs", type=int, default=3)
            child.add_argument("--warmups", type=int, default=1)
            child.add_argument("--sizes", type=int, nargs="+", default=[1, 10, 100])
            child.add_argument("--budget", type=float, default=120)
            child.add_argument("--max-load", type=float, default=4)
    args = parser.parse_args(argv)
    try:
        if args.command == "native":
            with args.samples.open(encoding="utf-8") as stream:
                samples = [json.loads(line) for line in stream if line.strip()]
            result = {"kind": "native", "input_sha256": hashlib.sha256(args.samples.read_bytes()).hexdigest(),
                      "quantile": "nearest-rank: ceil(p*n/100)-1", "groups": summarize(samples)}
        else:
            if os.name != "posix" or not hasattr(os, "getloadavg"):
                raise ValueError("live collection currently requires POSIX")
            if not math.isfinite(args.timeout) or args.timeout <= 0:
                raise ValueError("timeout must be positive and finite")
            if args.command == "costs":
                if args.runs < 1 or args.warmups < 0 or not 0 < args.budget <= 120 or not 0 < args.max_load <= 4:
                    raise ValueError("runs >= 1, warmups >= 0, budget <= 120 and max-load <= 4 required")
                if not args.sizes or any(n not in (1, 10, 100) for n in args.sizes) or len(set(args.sizes)) != len(args.sizes):
                    raise ValueError("sizes must be distinct members of 1, 10, 100")
                result = collect_costs(args.mrw, sorted(args.sizes), args.runs, args.warmups, args.budget, args.timeout, args.max_load)
            else:
                result = collect_warnings(args.mrw, args.timeout)
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print(f"mrw-observe: {error}", file=sys.stderr)
        return 2
    print(json.dumps(result, indent=2, allow_nan=False))
    return 1 if result.get("status") == "incomplete" or result.get("quality") == "fail" else 0


if __name__ == "__main__":
    raise SystemExit(main())
