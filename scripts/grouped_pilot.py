#!/usr/bin/env python3
"""Serial synthetic grouped-edit pilot; native tools and MRW receive identical task bytes."""
import argparse
from collections import Counter
import hashlib
import json
import os
from pathlib import Path
import random
import re
import shlex
import shutil
import signal
import subprocess
import sys
import tempfile
import time

import claude_evidence as pilot
import mrw_observe

TOOLS = {"A": ["Bash", "Read", "Edit", "Write", "Glob", "Grep"], "B": ["Bash", "Glob"]}
# F and C are the core-first guide ablation (docs/bench/observability/CORE-PREREGISTRATION.md):
# arm B in every respect except the guide each appends.
TOOLS["F"] = TOOLS["C"] = TOOLS["B"]
APPENDED = {"B": ["instructions"], "F": ["instructions"], "C": ["instructions", "--core"]}
REGISTERED_GUIDES = {"F": "a76dbd46ae65184aae51853b372c174f845a99fac6e2ad45c3a632a9fb7830f7",
                     "C": "3b90f436e1519f5afc95fce16bdfd97d955611e3d361240fd4dc3b41ae2227f3"}
CANARY = Path(__file__).resolve().parents[1] / "bench/tasks/canary"
# Text that only reaches a transcript when the user's own hooks run in the attempt.
HOOK_MARKERS = ("memoryd data", "branch-state:", "qh:standing", "[codebase-memory]")


def registered_guides(config):
    """The registered guide hashes, a run's configuration naming any it changes (Amendment 2)."""
    return dict(REGISTERED_GUIDES, **config.get("registered_guides", {}))


def paired_plan(names, arms="AB", reverse=False):
    ordered = sorted(names)
    random.Random(5030).shuffle(ordered)
    first, second = (arms[::-1], arms) if reverse else (arms, arms[::-1])
    return [(name, first if i % 2 == 0 else second) for i, name in enumerate(ordered)]


def strings_in(value):
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for item in value.values():
            yield from strings_in(item)
    elif isinstance(value, list):
        for item in value:
            yield from strings_in(item)


def canary_problems(row, transcript, claude_md):
    """What makes a canary fail: a bad verdict, the user's CLAUDE.md quoted, or a user hook's output."""
    problems = [key for key in ("execution_ok", "isolation_ok", "exact_tree_ok") if not row["verdict"].get(key)]
    text = transcript.decode("utf-8", "replace")
    decoded = []
    for line in text.splitlines():
        try:
            decoded.extend(strings_in(json.loads(line)))
        except ValueError:
            pass  # a line that is not JSON is still searched as raw text above
    text += "\n" + "\n".join(decoded)
    quoted = [line.strip() for line in claude_md.splitlines() if len(line.strip()) >= 40 and line.strip() in text]
    if quoted:
        problems.append("the user's CLAUDE.md is quoted: " + quoted[0][:80])
    problems.extend("a user hook's output is present: " + marker for marker in HOOK_MARKERS if marker in text)
    return problems


def round_two_gate(config):
    """Round 2 runs only after a completed, isolated round 1 that left room under the shared $3."""
    prior = Path(config["round1_root"])
    summary = json.loads((prior / "run-summary.json").read_text())
    if summary.get("stop_reason") != "completed":
        raise ValueError("round 1 did not complete: " + str(summary.get("stop_reason")))
    rows = summary.get("canaries", []) + summary.get("cells", [])
    plan = summary.get("plan") or []
    # The whole registered population, canaries included: a missing row escapes every check below.
    want = {("canary", "F"), ("canary", "C")} | {(name, arm) for name, arms in plan for arm in arms}
    if not plan or summary.get("not_run") or {(row.get("task"), row.get("arm")) for row in rows} != want:
        raise ValueError("round 1 is missing a canary or a registered cell")
    if any(not row["verdict"].get("isolation_ok") for row in rows):
        raise ValueError("round 1 has a canary or cell without passing isolation")
    evidence = Path(json.loads((prior / "adapter.json").read_text())["out"])
    missing = sorted(task + "." + arm for task, arm in want if not (evidence / (task + "." + arm + ".json")).exists())
    if missing:
        raise ValueError("round 1 adapter records are missing: " + ", ".join(missing))
    usage = pilot.prior_usage(evidence)[1]
    if usage >= 1.5:
        raise ValueError(f"round 1 used ${usage:.4f}, not below $1.50")
    ceiling = json.loads(Path(config["adapter"]).read_text()).get("max_usd", 3)
    if ceiling > 3 - usage + 1e-9:
        raise ValueError(f"round 2 ceiling ${ceiling} exceeds $3 minus round 1's ${usage:.4f}")
    return usage


def verdict(record, grade, code, arm, model):
    turns = record.get("num_turns")
    execution = (code == 0 and record.get("process_exit") == 0 and record.get("result_error") is False
                 and record.get("timed_out") is False and record.get("stream_errors") == 0
                 and type(turns) is int and 1 <= turns <= 10)
    isolation = (record.get("actual_model") == model and sorted(record.get("tools") or []) == sorted(TOOLS[arm])
                 and record.get("mcp_servers") == [] and record.get("plugins") == [])
    checked = record.get("evidence", {}).get("status") == "verified"
    return {"execution_ok": execution, "isolation_ok": isolation, "exact_tree_ok": grade.get("solved") is True,
            "verification_ok": checked, "solved": execution and isolation and checked and grade.get("solved") is True}


def tool_metrics(raw):
    calls, errors = {}, 0
    for line in raw.splitlines():
        try:
            row = json.loads(line)
            if not isinstance(row, dict):
                raise ValueError("stream row is not an object")
            if row.get("type") == "assistant":
                for part in row.get("message", {}).get("content", []):
                    if part.get("type") == "tool_use":
                        calls[part["id"]] = part
        except (ValueError, KeyError, UnicodeDecodeError):
            errors += 1
    commands = [part.get("input", {}).get("command", "") for part in calls.values() if part["name"] == "Bash"]
    return {"tool_calls": len(calls) if not errors else None,
            "by_kind": dict(Counter(part["name"] for part in calls.values())),
            "bash_mrw_write_mentions": sum(bool(re.search(r"\bmrw\s+write\b", command)) for command in commands),
            "parse_errors": errors}


def execute_cell(task, arm, config):
    out = Path(config["root"])
    spec = json.loads((task / "task.json").read_text())
    identity = spec["id"] + "." + arm
    record_path = out / "evidence" / (identity + ".json")
    if record_path.exists() or (out / "evidence" / (identity + ".started")).exists():
        raise ValueError("cell already started; use a fresh run directory")
    with tempfile.TemporaryDirectory(prefix="grouped-trial-") as d:
        scratch = Path(d)
        tree, tools = scratch / "tree", scratch / "tools"
        shutil.copytree(task / "base", tree)
        tools.mkdir()
        for name, target in {"git": shutil.which("git"), "python3": sys.executable, "python": sys.executable}.items():
            (tools / name).symlink_to(target)
        adapter = json.loads(Path(config["adapter"]).read_text())
        if arm in APPENDED:
            (tools / "mrw").symlink_to(adapter["mrw"])
        env = {"HOME": os.environ["HOME"], "USER": os.environ.get("USER", ""), "LANG": "C.UTF-8",
               "PATH": str(tools) + ":/usr/bin:/bin:/usr/sbin:/sbin", "TMPDIR": str(scratch),
               "CLAUDE_CONFIG_DIR": config["profile"], "CLAUDE_CODE_SHELL": "/bin/bash",
               "CLAUDE_CODE_TMPDIR": str(scratch / "cctmp"), "XDG_CONFIG_HOME": str(scratch / "config"),
               "XDG_STATE_HOME": str(scratch / "state"), "GIT_CONFIG_GLOBAL": os.devnull, "PYTHONNOUSERSITE": "1"}
        if arm in REGISTERED_GUIDES:
            env["BENCH_ARM"] = arm
        for name in ("cctmp", "config", "state"):
            (scratch / name).mkdir()
        git = lambda *args: subprocess.run(["git", "-C", str(tree), *args], env=env, check=True, capture_output=True, timeout=10)
        git("init", "-q")
        git("add", "-A")
        git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false",
            "-c", "core.hooksPath=/dev/null", "commit", "-qm", "frozen synthetic baseline")
        initial = mrw_observe.tree_evidence(tree, [])
        settings = {}
        if arm in APPENDED:
            command = shlex.quote(config["bench"]) + " hook --log " + shlex.quote(str(scratch / "hook.log")) + " --tree " + shlex.quote(str(tree))
            settings["hooks"] = {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": command}]}]}
        (scratch / "settings.json").write_text(json.dumps(settings))
        (scratch / "mcp.json").write_text('{"mcpServers":{}}')
        args = [sys.executable, str(Path(pilot.__file__).resolve()), config["adapter"], "-p", spec["instruction"],
                "--output-format", "stream-json", "--verbose", "--model", config["model"],
                "--tools", ",".join(TOOLS[arm]), "--strict-mcp-config", "--mcp-config", str(scratch / "mcp.json"),
                "--disable-slash-commands", "--no-session-persistence", "--no-chrome",
                "--permission-mode", "bypassPermissions", "--exclude-dynamic-system-prompt-sections",
                "--settings", str(scratch / "settings.json"), "--max-turns", "10"]
        if arm in APPENDED:
            instructions = subprocess.run([adapter["mrw"], *APPENDED[arm]], check=True, capture_output=True, timeout=5).stdout
            if arm in REGISTERED_GUIDES and hashlib.sha256(instructions).hexdigest() != registered_guides(config)[arm]:
                raise ValueError(f"arm {arm}'s guide is not the registered one")
            (scratch / "instructions.txt").write_bytes(instructions)
            args.extend(["--append-system-prompt-file", str(scratch / "instructions.txt")])
        stdout_path, stderr_path = out / "transcripts" / (identity + ".jsonl"), out / "transcripts" / (identity + ".stderr")
        start = time.monotonic()
        with stdout_path.open("xb") as stdout, stderr_path.open("xb") as stderr:
            with subprocess.Popen(args, cwd=tree, env=env, stdout=stdout, stderr=stderr, start_new_session=True) as child:
                pid = child.pid
                try:
                    code = child.wait(timeout=90)
                except (subprocess.TimeoutExpired, KeyboardInterrupt) as error:
                    try:
                        os.killpg(pid, signal.SIGINT)
                    except ProcessLookupError:
                        pass
                    try:
                        child.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        try:
                            os.killpg(pid, signal.SIGKILL)
                        except ProcessLookupError:
                            pass
                        child.wait()
                    code = 130 if isinstance(error, KeyboardInterrupt) else 124
        wall = round((time.monotonic() - start) * 1000)
        if not record_path.exists():
            raise ValueError(f"{identity}: no adapter record; raw output retained")
        record = json.loads(record_path.read_text())
        observed = record.get("evidence", {}).get("before", {}).get("files")
        if observed is None:
            observed = mrw_observe.tree_evidence(tree, initial["files"])["files"]
        grade = pilot.exact_tree_grade(task, observed)
        matched = sum((tree / edit["file"]).is_file() and edit["new"] in (tree / edit["file"]).read_text()
                      for edit in spec.get("replacements", []))
        row = {"task": spec["id"], "arm": arm, "site_budget": spec.get("edit_sites"), "file_budget": spec.get("edited_files"),
               "matched_sites": matched, "changed_files": sum(initial["files"].get(name) != observed.get(name)
                                                              for name in set(initial["files"]) | set(observed)),
               "initial_tree": initial, "record": record, "grade": grade,
               "verdict": verdict(record, grade, code, arm, config["model"]), "outer_wall_ms": wall, "outer_exit": code,
               "metrics": tool_metrics(stdout_path.read_bytes()), "supervisor_pid": pid,
               "transcript_sha256": hashlib.sha256(stdout_path.read_bytes()).hexdigest(),
               "stderr_sha256": hashlib.sha256(stderr_path.read_bytes()).hexdigest()}
        pilot.save(out / "cells", identity, row)
        return row


def run(config):
    out, corpus = Path(config["root"]), Path(config["corpus"])
    if (out / "run-summary.json").exists():
        raise ValueError("run already completed")
    arms_label = config.get("arms", "AB")
    plan = paired_plan((p.name for p in corpus.iterdir() if p.is_dir()), arms_label, reverse=config.get("round") == 2)
    cells, attempts, canaries, reason = [], [], [], "completed"
    deadline = time.monotonic() + 540
    try:
        if config.get("round") == 2:
            try:
                round_two_gate(config)
            except (ValueError, OSError) as error:
                reason = "round 1 gate: " + str(error)
        if reason == "completed" and arms_label == "FC":
            claude_md_path = Path.home() / ".claude" / "CLAUDE.md"
            claude_md = claude_md_path.read_text() if claude_md_path.exists() else ""
            for arm in arms_label:
                attempt = {"task":"canary","arm":arm,"status":"started","stage":"canary"}
                attempts.append(attempt)
                try:
                    row = execute_cell(CANARY, arm, config)
                except (ValueError, OSError, subprocess.SubprocessError) as error:
                    reason = str(error)
                    attempt.update(status="failed", error=reason)
                    break
                path = out / "transcripts" / ("canary." + arm + ".jsonl")
                transcript = path.read_bytes() if path.exists() else b""
                row["problems"] = canary_problems(row, transcript, claude_md) + ([] if path.exists() else ["no transcript"])
                canaries.append(row)
                attempt.update(status="completed", stage="recorded")
                print("canary", arm, row["problems"] or "pass", flush=True)
                if row["problems"]:
                    reason = "canary failed"
                    break
        for name, arms in plan if reason == "completed" else []:
            if deadline - time.monotonic() < 180:
                reason = "insufficient time for a complete pair"
                break
            for arm in arms:
                attempt = {"task":name,"arm":arm,"status":"started","stage":"execute_cell"}
                attempts.append(attempt)
                try:
                    row = execute_cell(corpus / name, arm, config)
                except (ValueError, OSError, subprocess.SubprocessError) as error:
                    reason = str(error)
                    attempt.update(status="failed", error=reason)
                    break
                cells.append(row)
                attempt.update(status="completed", stage="recorded")
                print(name, arm, row["verdict"], "turns", row["record"].get("num_turns"), flush=True)
                if row["outer_exit"] == 130:
                    reason = "interrupted"
                    break
                if not row["verdict"]["isolation_ok"]:
                    reason = "isolation failed"
                    break
            if reason != "completed":
                break
    except KeyboardInterrupt:
        reason = "interrupted"
        if attempts and attempts[-1]["status"] == "started":
            attempts[-1].update(status="interrupted", error="KeyboardInterrupt")
    finally:
        for attempt in attempts:
            marker = out / "evidence" / (attempt["task"] + "." + attempt["arm"] + ".started")
            attempt["adapter_started"] = marker.exists()
        attempted = {(attempt["task"], attempt["arm"]) for attempt in attempts}
        summary = {"plan": plan, "round": config.get("round"), "canaries": canaries, "cells": cells, "attempts":attempts, "stop_reason": reason,
                   "performance_verdict":"not_attempted",
                   "not_run":[{"task":name,"arm":arm} for name, arms in plan for arm in arms
                              if (name, arm) not in attempted]}
        pilot.save(out, "run-summary", summary)
    return summary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("config", type=Path)
    args = parser.parse_args()
    try:
        summary = run(json.loads(args.config.read_text()))
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print(f"grouped-pilot: {error}", file=sys.stderr)
        return 2
    if summary["stop_reason"] == "interrupted":
        return 130
    return 0 if summary["stop_reason"] == "completed" and all(cell["verdict"]["solved"] for cell in summary["cells"]) else 1


if __name__ == "__main__":
    raise SystemExit(main())
