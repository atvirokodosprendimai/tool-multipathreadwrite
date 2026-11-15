"""anchor-miss scanner (ADR-197): how often is a refused `anchor=` followed by a read?

Stdlib only. Run: python3 -I scan.py [--root DIR] [--since YYYY-MM-DD]
It reads the top-level Claude Code transcripts under DIR (default ~/.claude/projects),
one level of project folders, subagent transcripts excluded, and prints aggregates
only: no paths, file contents, project names or session ids.

Limits: a count is a tool result, not an incident; "a read" is the next tool call
being Read, or a shell call holding `mrw … read`, cat, sed -n or grep; and a read
after a miss may be one the guard requires anyway (ADR-002, ADR-052), so the share
is an upper bound on what a better refusal can save.
"""
import argparse
import datetime
import glob
import json
import os
import re

MISS = re.compile(r'anchor ".*" not in line \d+')
READ = re.compile(r"mrw[^|]*\bread\b|\bcat\b|sed -n|grep")


def text_of(content):
    """Return a tool result's text, whichever of the two shapes it has."""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(b.get("text", "") for b in content if isinstance(b, dict))
    return ""


def events(path):
    """Yield ("res", text) and ("use", name, input) in transcript order."""
    with open(path, errors="replace") as fh:
        for line in fh:
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            content = (entry.get("message") or {}).get("content")
            if not isinstance(content, list):
                continue
            for block in content:
                if not isinstance(block, dict):
                    continue
                if block.get("type") == "tool_result":
                    yield ("res", text_of(block.get("content")))
                elif block.get("type") == "tool_use":
                    yield ("use", block.get("name", ""), json.dumps(block.get("input") or {})[:400])


def main():
    """Count anchor misses and the kind of tool call that follows each."""
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=os.path.expanduser("~/.claude/projects"))
    ap.add_argument("--since", help="only transcripts modified on or after this date")
    args = ap.parse_args()
    floor = 0.0
    if args.since:
        floor = datetime.datetime.strptime(args.since, "%Y-%m-%d").timestamp()
    out = {"transcripts": 0, "with_a_miss": 0, "misses": 0, "next_read": 0, "next_other": 0, "next_none": 0}
    for path in sorted(glob.glob(os.path.join(args.root, "*", "*.jsonl"))):
        if os.path.getmtime(path) < floor:
            continue
        out["transcripts"] += 1
        evs = list(events(path))
        hit = False
        for i, ev in enumerate(evs):
            if ev[0] != "res" or not MISS.search(ev[1]):
                continue
            hit = True
            out["misses"] += 1
            nxt = next((e for e in evs[i + 1:] if e[0] == "use"), None)
            if nxt is None:
                out["next_none"] += 1
            elif nxt[1] == "Read" or (nxt[1] == "Bash" and READ.search(nxt[2])):
                out["next_read"] += 1
            else:
                out["next_other"] += 1
        out["with_a_miss"] += hit
    print(json.dumps(out, indent=1))


if __name__ == "__main__":
    main()
