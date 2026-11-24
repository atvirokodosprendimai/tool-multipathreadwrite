"""Dump the whole hook payload, long strings cut, one JSON line per event (the probe hook of #790)."""
import json
import os
import sys


def cut(v, n=1500):
    if isinstance(v, str):
        return v if len(v) <= n else v[:n] + f"...[{len(v)} chars]"
    if isinstance(v, dict):
        return {k: cut(x, n) for k, x in v.items()}
    if isinstance(v, list):
        return [cut(x, n) for x in v[:20]]
    return v


d = json.load(sys.stdin)
with open(os.environ["CM_HOOK_LOG"] + ".full", "a", encoding="utf-8") as f:
    f.write(json.dumps(cut(d)) + "\n")
