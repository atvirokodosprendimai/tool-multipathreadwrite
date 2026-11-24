"""The pilot's logging hook: every payload, strings cut at 4,000 characters, one JSON line per event."""

import json
import os
import shutil
import sys


def cut(v, n=4000):
    """Return v with every string in it cut to n characters."""
    if isinstance(v, str):
        return v if len(v) <= n else v[:n] + f"...[{len(v)} chars]"
    if isinstance(v, dict):
        return {k: cut(x, n) for k, x in v.items()}
    if isinstance(v, list):
        return [cut(x, n) for x in v[:50]]
    return v


d = cut(json.load(sys.stdin))
d["_mrw"] = shutil.which("mrw")
d["_shown"] = os.environ.get("MRW_SHOWN")
with open(os.environ["CM_HOOK_LOG"], "a", encoding="utf-8") as f:
    f.write(json.dumps(d) + "\n")
