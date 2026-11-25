"""Count one run of the second pilot from its hook log, by README.md's rules.

    summarise.py ARM HOOKS.jsonl

Prints: writes by form, landings, refusals by kind in order, and what voids the run.
A log with no flags of its own (the first pilot's) is counted from the text it kept.
"""

import json
import re
import sys

WRITERS = re.compile(r"(^|[;&|]\s*)(sed\s+-i|perl\s+-i|python3?\b|tee\b|cp\b|mv\b|git\s+(checkout|restore|apply)\b)|>\s*\S*pkg/")
MRW_WRITE = re.compile(r"\bmrw\b(?:\s+(?:--root|-C)\s+\S+)?\s+write\b([^;&|\n]*)")
MRW_READ = re.compile(r"\bmrw\b(\s+(--root|-C)\s+\S+)?\s+read\b")
NOT_A_WRITE = re.compile(r"(^|\s)(--help|-h|--dry-run)(\s|=|$)")
LANDED = re.compile(r'(?m)^\d+ hunk\(s\), .*— applied\s*$|"mutation": "complete"')


def flags(text):
    """Return the logger's flags, taken from kept text: only for a log that has none."""
    return {
        "note": "recorded as licensed, not shown" in text,
        "landed": bool(LANDED.search(text)) and "NOTHING WRITTEN" not in text,
        "number": "no recorded showing" in text,
        "line_after": "needs a served line after" in text,
    }


def form(cmd):
    """Return the form of an mrw write command."""
    if "--sub" in cmd or "--rewrite" in cmd:
        return "computed"
    if "search_replace" in cmd:
        return "block"
    if "apply_patch" in cmd:
        return "patch"
    return "native"


def is_write(cmd):
    """Report whether a command holds an mrw write that is neither help nor a dry run."""
    return any(not NOT_A_WRITE.search(rest) for rest in MRW_WRITE.findall(cmd))


def main(arm, path):
    """Print the run's counts on one line."""
    forms, kinds, void = [], [], []
    landed, switched_read = 0, False
    try:
        rows = [json.loads(line) for line in open(path, encoding="utf-8")]
    except OSError:
        print("void=no-hook-log")
        return
    for d in rows:
        tool, cmd = d.get("tool_name"), (d.get("tool_input") or {}).get("command") or ""
        if d.get("hook_event_name") != "PostToolUse":
            continue
        if tool in ("Edit", "Write", "NotebookEdit"):
            if d.get("_under_pkg", "/pkg/" in str((d.get("tool_input") or {}).get("file_path"))):
                void.append(f"host-{tool}-under-pkg")
            continue
        if tool != "Bash":
            continue
        resp = d.get("tool_response") or {}
        seen = d.get("_flags") or flags((resp.get("stdout") or "") + "\n" + (resp.get("stderr") or ""))
        if MRW_READ.search(cmd) and seen["note"]:
            switched_read = True
        if is_write(cmd):
            forms.append(form(cmd))
            if seen["landed"]:
                landed += 1
                kinds.append("landed")
            elif seen["number"]:
                kinds.append("number")
            elif seen["line_after"]:
                kinds.append("line-after")
            else:
                kinds.append("other")
        elif WRITERS.search(cmd) and "mrw" not in cmd.split()[0:1]:
            void.append("other-writer")
    if arm == "switched" and not switched_read:
        void.append("no-switched-read")
    refused = [k for k in kinds if k != "landed"]
    streak = any(kinds[i] != "landed" and kinds[i] == kinds[i + 1] == kinds[i + 2] for i in range(len(kinds) - 2))
    hand = sum(1 for f, k in zip(forms, kinds) if k == "landed" and f in ("native", "block"))
    print(
        f"writes={len(forms)} forms={','.join(forms) or '-'} landed={landed} refused={len(refused)} "
        f"kinds={','.join(kinds) or '-'} three-in-a-row={'yes' if streak else 'no'} hand-landings={hand} "
        f"void={','.join(void) or 'no'}"
    )


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    main(sys.argv[1], sys.argv[2])
