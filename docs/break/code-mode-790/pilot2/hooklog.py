"""The second pilot's logging hook: flags taken on the whole payload, then strings cut at 4,000 characters."""

import json
import os
import re
import shutil
import sys

LANDED = re.compile(r'(?m)^\d+ hunk\(s\), .*— applied\s*$|"mutation": "complete"')


def flags(text):
    """Return what the counting rules look for in a Bash call's whole output."""
    return {
        "note": "recorded as licensed, not shown" in text,
        "landed": bool(LANDED.search(text)) and "NOTHING WRITTEN" not in text,
        "nothing_written": "NOTHING WRITTEN" in text,
        "number": "no recorded showing" in text,
        "line_after": "needs a served line after" in text,
    }


def under_pkg(d):
    """Report whether a host file tool's target is inside the working directory's pkg/."""
    target = (d.get("tool_input") or {}).get("file_path") or (d.get("tool_input") or {}).get("notebook_path")
    if not target:
        return False
    cwd = os.path.realpath(d.get("cwd") or os.getcwd())
    path = os.path.realpath(os.path.join(cwd, target))
    return path.startswith(os.path.join(cwd, "pkg") + os.sep)


def cut(v, n=4000):
    """Return v with every string in it cut to n characters."""
    if isinstance(v, str):
        return v if len(v) <= n else v[:n] + f"...[{len(v)} chars]"
    if isinstance(v, dict):
        return {k: cut(x, n) for k, x in v.items()}
    if isinstance(v, list):
        return [cut(x, n) for x in v[:50]]
    return v


def main():
    """Append one event to the run's log."""
    d = json.load(sys.stdin)
    extra = {"_mrw": shutil.which("mrw"), "_shown": os.environ.get("MRW_SHOWN")}
    if d.get("hook_event_name") == "PostToolUse":
        if d.get("tool_name") == "Bash":
            resp = d.get("tool_response")
            resp = resp if isinstance(resp, dict) else {"stdout": str(resp or "")}
            extra["_flags"] = flags((resp.get("stdout") or "") + "\n" + (resp.get("stderr") or ""))
        elif d.get("tool_name") in ("Edit", "Write", "NotebookEdit"):
            extra["_under_pkg"] = under_pkg(d)
    d = cut(d)
    d.update(extra)
    with open(os.environ["CM_HOOK_LOG"], "a", encoding="utf-8") as f:
        f.write(json.dumps(d) + "\n")


if __name__ == "__main__":
    main()
