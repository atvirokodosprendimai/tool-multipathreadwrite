"""Count one pilot run from its hook log, by README.md's rules.

    summarise.py ARM HOOKS.jsonl

Prints: writes by form, landings, refusals by kind in order, and what voids the run.
"""

import json
import re
import sys

WRITERS = re.compile(r"(^|[;&|]\s*)(sed\s+-i|perl\s+-i|python3?\b|tee\b|cp\b|mv\b|git\s+(checkout|restore|apply)\b)|>\s*\S*pkg/")
MRW_WRITE = re.compile(r"\bmrw\b(\s+(--root|-C)\s+\S+)?\s+write\b")
MRW_READ = re.compile(r"\bmrw\b(\s+(--root|-C)\s+\S+)?\s+read\b")


def form(cmd):
    """Return the form of an mrw write command."""
    if "--sub" in cmd or "--rewrite" in cmd:
        return "computed"
    if "search_replace" in cmd:
        return "block"
    if "apply_patch" in cmd:
        return "patch"
    return "native"


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
            void.append(f"host-{tool}")
        if tool != "Bash":
            continue
        resp = d.get("tool_response") or {}
        text = (resp.get("stdout") or "") + "\n" + (resp.get("stderr") or "")
        if MRW_READ.search(cmd) and "recorded as licensed, not shown" in text:
            switched_read = True
        if MRW_WRITE.search(cmd) and "--dry-run" not in cmd:
            forms.append(form(cmd))
            if "applied" in text and "NOTHING WRITTEN" not in text:
                landed += 1
                kinds.append("landed")
            elif "no recorded showing" in text:
                kinds.append("number")
            elif "needs a served line after" in text:
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
