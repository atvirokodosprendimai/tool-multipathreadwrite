#!/usr/bin/env python3
"""Run scripts/static.sh after a commit and hand the verdict to the model (ADR-088).

PostToolUse on Bash. It runs when HEAD moved AND the command named `commit`:
HEAD moving is what says a commit landed — a refused commit moves nothing — and
the command's words keep a checkout, a pull or a rebase from paying for an
analysis nobody asked for. Every call records HEAD, so a HEAD moved under
another command is not analysed later by the next commit that moves nothing.
The first call in a checkout only records HEAD: opening a session is not a
commit.

The verdict goes back as additionalContext: "clean", or the script's output
with an instruction to fix the findings in the next commit or say why not.

Exit 0 always, closed stdout included: a hook must never take the turn down.
The script is bounded at 280 s, inside the 300 s the registration allows.
"""

import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile

LIMIT = 6000
TIMEOUT = 280
COMMIT = re.compile(r"\bgit\b[^|;&]*\bcommit\b")


def head(root):
    try:
        p = subprocess.run(["git", "-C", root, "rev-parse", "HEAD"],
                           capture_output=True, text=True, timeout=10)
    except (OSError, subprocess.SubprocessError):
        return ""
    return p.stdout.strip() if p.returncode == 0 else ""


def analyse(root, script):
    try:
        p = subprocess.run([script], cwd=root, capture_output=True, text=True, timeout=TIMEOUT)
    except subprocess.TimeoutExpired:
        return 1, f"scripts/static.sh did not finish in {TIMEOUT} s; run it yourself"
    except OSError as e:
        return 1, f"scripts/static.sh could not start: {e}"
    return p.returncode, (p.stdout + p.stderr).strip()


def main():
    try:
        call = json.load(sys.stdin)
    except (ValueError, OSError):
        return
    command = str((call.get("tool_input") or {}).get("command", ""))
    root = os.environ.get("CLAUDE_PROJECT_DIR") or call.get("cwd") or os.getcwd()
    script = os.path.join(root, "scripts", "static.sh")
    now = head(root)
    if not now or not os.path.isfile(script):
        return
    cache = os.path.join(tempfile.gettempdir(), "claude-static-after-commit")
    os.makedirs(cache, mode=0o700, exist_ok=True)
    mark = os.path.join(cache, hashlib.sha256(os.path.realpath(root).encode()).hexdigest()[:16])
    try:
        with open(mark) as f:
            last = f.read().strip()
    except OSError:
        last = None
    if last == now:
        return
    with open(mark, "w") as f:
        f.write(now + "\n")
    if last is None or not COMMIT.search(command):
        return
    code, out = analyse(root, script)
    if code == 0:
        msg = f"Static analysis after commit {now[:7]} (scripts/static.sh, ADR-088): clean."
    else:
        if len(out) > LIMIT:
            out = "…" + out[-LIMIT:]
        msg = (f"Static analysis after commit {now[:7]} (scripts/static.sh, ADR-088) FAILED, "
               f"exit {code}. Fix these in the next commit, or say why not "
               f"(.claude/rules/static-analysis.md):\n{out}")
    print(json.dumps({"hookSpecificOutput": {"hookEventName": "PostToolUse",
                                             "additionalContext": msg}}))


if __name__ == "__main__":
    try:
        main()
    except Exception:  # a hook must never take the turn down
        pass
    try:
        sys.stdout.flush()
    except OSError:
        pass
    sys.exit(0)
