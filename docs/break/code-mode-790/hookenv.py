"""Log one line per hook event: which tool, its id, and what the hook process's environment holds."""
import json
import os
import sys

d = json.load(sys.stdin)
row = {
    "event": d.get("hook_event_name"),
    "tool": d.get("tool_name"),
    "id": d.get("tool_use_id"),
    "MRW_SHOWN": os.environ.get("MRW_SHOWN"),
    "CODE_MODE": os.environ.get("CLAUDE_CODE_EXPERIMENTAL_CODE_MODE"),
    "command": (d.get("tool_input") or {}).get("command"),
}
with open(os.environ["CM_HOOK_LOG"], "a", encoding="utf-8") as f:
    f.write(json.dumps(row) + "\n")
