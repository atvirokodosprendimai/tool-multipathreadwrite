#!/usr/bin/env python3
"""Experimental bench CLI adapter: forward Claude output and retain final-check evidence."""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import time
import tempfile

import mrw_observe



def exact_tree_grade(task, observed):
    """Apply the registered reference in a disposable tree and compare every measured file."""
    with tempfile.TemporaryDirectory(prefix="reference-grade-") as d:
        root = Path(d)
        shutil.copytree(task / "base", root, dirs_exist_ok=True)
        subprocess.run(["git", "init", "-q", str(root)], check=True, capture_output=True, timeout=5)
        subprocess.run(["git", "-C", str(root), "apply", str((task / "reference.patch").resolve())],
                       check=True, capture_output=True, timeout=5)
        expected = {}
        for path in root.rglob("*"):
            name = path.relative_to(root)
            if ".git" not in name.parts and path.is_file():
                expected[str(name)] = {"sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                                       "mode": "100755" if path.stat().st_mode & 0o100 else "100644"}
    differences = sorted(name for name in set(expected) | set(observed) if expected.get(name) != observed.get(name))
    return {"solved": not differences, "differences": differences, "expected_files": expected,
            "expected_tree_sha256": mrw_observe.canonical_sha(expected)}


def save(root, identity, value):
    with (root / (identity + ".json")).open("x", encoding="utf-8") as stream:
        json.dump(value, stream, indent=2, allow_nan=False)


def prior_usage(root):
    """Launches and dollars already in this ledger; refuses when any of them is unknown."""
    if any(not path.with_suffix(".json").exists() for path in root.glob("*.started")):
        raise ValueError("unfinished prior launch has unknown cost")
    rows = [json.loads(path.read_text()) for path in root.glob("*.json")]
    costs = [row.get("cost_usd") for row in rows]
    if any(type(cost) not in (int, float) or not math.isfinite(cost) or cost < 0 for cost in costs):
        raise ValueError("unknown prior cost: no further paid launch")
    for row in rows:
        usage = row.get("usage")
        if not isinstance(usage, dict) or any(type(usage.get(key)) is not int or usage[key] < 0
                                              for key in ("input_tokens", "output_tokens")):
            raise ValueError("unknown prior usage: no further paid launch")
    return len(rows), sum(costs)


def cost_guard(root, max_launches=8, max_usd=3):
    if type(max_launches) is not int or not 1 <= max_launches <= 20:
        raise ValueError("max_launches must be an integer between 1 and 20")
    # A configured ceiling may only lower the $3 stop, never raise it.
    if type(max_usd) not in (int, float) or not 0 < max_usd <= 3:
        raise ValueError("max_usd must be a number above 0 and at most 3")
    launches, spent = prior_usage(root)
    if launches >= max_launches or spent >= max_usd:
        raise ValueError("pilot budget exhausted")


def arm_for(argv, named):
    """The cell's arm: inferred A/B when unnamed; a named F or C must carry an appended guide."""
    appended = "--append-system-prompt-file" in argv
    if named is None:
        return "B" if appended else "A"
    if named not in ("F", "C"):
        raise ValueError("arm name must be F or C")
    if not appended:
        raise ValueError("arm " + named + " needs an appended guide")
    return named


def transcript_meta(raw):
    rows, errors = [], 0
    for line in raw.splitlines():
        if not line.strip():
            continue
        try:
            row = json.loads(line)
            if not isinstance(row, dict):
                raise ValueError("stream row is not an object")
            rows.append(row)
        except (ValueError, UnicodeDecodeError):
            errors += 1
    init = next((row for row in rows if row.get("type") == "system" and row.get("subtype") == "init"), {})
    result = next((row for row in reversed(rows) if row.get("type") == "result"), {})
    return {"actual_model": init.get("model"), "tools": init.get("tools"), "mcp_servers": init.get("mcp_servers"),
            "plugins": [p.get("name") for p in init.get("plugins", [])],
            "cost_usd": result.get("total_cost_usd"), "usage": result.get("usage"),
            "num_turns": result.get("num_turns"), "result_error": result.get("is_error"), "stream_errors": errors}



def isolated_settings(settings, plugins):
    """Keep the experiment's hooks while excluding host instructions and synced context."""
    result = dict(settings)
    disabled = dict(result.get("enabledPlugins", {}))
    disabled.update(dict.fromkeys(plugins, False))
    result.update(claudeMdExcludes=["**"], autoMemoryEnabled=False, syncClaudeAiSkills=False,
                  disableBundledSkills=True, enabledPlugins=disabled)
    return result


def wrap(argv, config):
    real, root, out = config["claude"], Path.cwd(), Path(config["out"])
    if argv == ["--version"]:
        return subprocess.run([real, *argv], check=False, timeout=5).returncode
    cost_guard(out, config.get("max_launches", 8), config.get("max_usd", 3))
    prompt = argv[argv.index("-p") + 1]
    arm = arm_for(argv, os.environ.get("BENCH_ARM"))
    task = json.loads((root / "TASK.json").read_text())["id"] if (root / "TASK.json").exists() else "canary"
    identity = task + "." + arm
    if not re.fullmatch(r"[a-z0-9-]+\.[ABFC]", identity):
        raise ValueError("unsafe task identity")
    if (out / (identity + ".json")).exists():
        raise ValueError("started cell is never rerun")
    instruction_hashes = {"prompt": hashlib.sha256(prompt.encode()).hexdigest()}
    if "--append-system-prompt-file" in argv:
        path = Path(argv[argv.index("--append-system-prompt-file") + 1])
        instruction_hashes["appended_file"] = hashlib.sha256(path.read_bytes()).hexdigest()
    settings = json.loads(Path(argv[argv.index("--settings") + 1]).read_text()) if "--settings" in argv else {}
    settings = isolated_settings(settings, config.get("disabled_plugins", []))
    argv = list(argv)
    if "--settings" in argv:
        argv[argv.index("--settings") + 1] = json.dumps(settings)
    else:
        argv.extend(["--settings", json.dumps(settings)])
    argv.extend(["--setting-sources", ""])
    instruction_hashes["settings"] = mrw_observe.canonical_sha(settings)
    baseline = mrw_observe.git_output(root, "rev-parse", "HEAD").decode().strip()
    with (out / (identity + ".started")).open("x"):
        pass
    start = time.monotonic()
    interrupted = timed_out = False

    def interrupt(signum, frame):
        raise KeyboardInterrupt

    handlers = {sig: signal.signal(sig, interrupt) for sig in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP)}
    try:
        # Own and stop this CLI group before fingerprinting, including on clean return.
        # The arm name is the driver's label for the adapter; Claude never sees it.
        child_env = {key: value for key, value in os.environ.items() if key != "BENCH_ARM"}
        with subprocess.Popen([real, *argv, "--max-budget-usd", "0.50"], stdin=subprocess.DEVNULL,
                              stdout=subprocess.PIPE, start_new_session=True, env=child_env) as child:
            try:
                raw, _ = child.communicate(timeout=75)
            except (subprocess.TimeoutExpired, KeyboardInterrupt) as error:
                timed_out = isinstance(error, subprocess.TimeoutExpired)
                interrupted = isinstance(error, KeyboardInterrupt)
            finally:
                try:
                    os.killpg(child.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                raw, _ = child.communicate()
            code = 130 if interrupted else (124 if timed_out else child.returncode)
    finally:
        for sig, handler in handlers.items():
            signal.signal(sig, handler)
    sys.stdout.buffer.write(raw)
    sys.stdout.buffer.flush()
    record = transcript_meta(raw)
    record.update(process_exit=code, timed_out=timed_out, interrupted=interrupted,
                  claude_wall_ms=round((time.monotonic() - start) * 1000),
                  instruction_sha256=instruction_hashes, baseline_revision=baseline,
                  wrapper_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest())
    if task != "canary":
        before = time.monotonic()
        try:
            record["evidence"] = mrw_observe.verify_trial(config["mrw"], root, baseline,
                                                         ["accept.py", "TASK.json"], ["TASK.json"], 5)
        except (ValueError, OSError, subprocess.SubprocessError) as error:
            record["evidence_error"] = str(error)
        record["observer_wall_ms"] = round((time.monotonic() - before) * 1000)
    save(out, identity, record)
    return code


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("config", type=Path)
    parser.add_argument("claude_args", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    try:
        return wrap(args.claude_args, json.loads(args.config.read_text()))
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print(f"claude-evidence: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
