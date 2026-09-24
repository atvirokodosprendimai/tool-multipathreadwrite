#!/usr/bin/env python3
"""blind-score.py DIR TRANSCRIPT — score one blind-agent trial against its key.

DIR is what scripts/blind-agent.sh built. TRANSCRIPT is the agent's JSONL
transcript. Compliance is read from the transcript's own tool calls, never from
the agent's report: a run that used a banned tool is VOID (BACKLOG, "From
ADR-009"). Prints one JSON object and exits 0; a transcript it cannot read is
reported as VOID with the reason, never as a score.
"""
import json
import re
import sys
from pathlib import Path

BANNED_CMDS = {"grep", "rg", "find", "ls", "cat", "sed", "awk", "head", "tail"}
BANNED_TOOLS = {"Read", "Grep", "Glob", "Edit", "Write", "NotebookEdit"}


def tool_calls(transcript):
    calls = []
    for line in transcript.read_text(errors="replace").splitlines():
        try:
            o = json.loads(line)
        except ValueError:
            continue
        m = o.get("message") or {}
        if m.get("role") != "assistant" or not isinstance(m.get("content"), list):
            continue
        for b in m["content"]:
            if isinstance(b, dict) and b.get("type") == "tool_use":
                calls.append((b.get("name"), b.get("input") or {}))
    return calls


def final_answer(transcript):
    """The LAST fenced ```json block the agent emitted, wherever it put it.

    An agent's output arrives on two channels: its assistant text, and the
    SubagentHandback tool call that carries its final report. Blind reading 01
    read only the last assistant text and missed reports sent through the
    hand-back and blocks followed by more text (void, docs/blind/blind-01-void.md).
    The rule since blind reading 02: every assistant text and every hand-back
    message, in transcript order, and the last json block among them wins.
    """
    found = None
    for line in transcript.read_text(errors="replace").splitlines():
        try:
            o = json.loads(line)
        except ValueError:
            continue
        m = o.get("message") or {}
        if m.get("role") != "assistant" or not isinstance(m.get("content"), list):
            continue
        for b in m["content"]:
            if not isinstance(b, dict):
                continue
            s = ""
            if b.get("type") == "text":
                s = b.get("text", "")
            elif b.get("type") == "tool_use" and b.get("name") == "SubagentHandback":
                s = (b.get("input") or {}).get("message", "")
            blocks = re.findall(r"```json\s*(\{.*?\})\s*```", s, re.S)
            if blocks:
                found = blocks[-1]
    return found


def segments(cmd):
    # First word of every pipeline or list segment; quoted text is removed first
    # so a word inside a plan body or a pattern is not read as a command.
    stripped = re.sub(r"'[^']*'|\"(?:\\.|[^\"\\])*\"", "''", cmd)
    for seg in re.split(r"\|\||&&|[|;&\n()]", stripped):
        words = seg.strip().split()
        while words and re.match(r"^[A-Za-z_][A-Za-z0-9_]*=", words[0]):
            words = words[1:]
        if words:
            yield words[0].rsplit("/", 1)[-1]


def main():
    d, transcript = Path(sys.argv[1]), Path(sys.argv[2])
    key = json.loads((d / "answer-key.json").read_text())
    out = {"trial": d.name}
    try:
        calls = tool_calls(transcript)
    except OSError as e:
        out.update(verdict="VOID", reason=f"transcript unreadable: {e}")
        print(json.dumps(out)); return
    if not calls:
        out.update(verdict="VOID", reason="transcript holds no tool calls")
        print(json.dumps(out)); return
    violations, mrw_calls = [], 0
    for name, inp in calls:
        if name in BANNED_TOOLS:
            violations.append(f"tool {name}")
        if name != "Bash":
            continue
        cmd = inp.get("command", "")
        if "--help" in cmd:
            violations.append("--help")
        for w in segments(cmd):
            if w in BANNED_CMDS:
                violations.append(f"command {w}")
        mrw_calls += len(re.findall(r"(?:^|[\s|;&(])(?:\S*/)?(?:mrw|\$MRW|\$\{MRW\})(?=\s)", cmd))
    out["mrw_calls"] = mrw_calls
    if violations:
        out.update(verdict="VOID", reason="banned: " + ", ".join(sorted(set(violations))))
        print(json.dumps(out)); return
    block = final_answer(transcript)
    if block is None:
        out.update(verdict="MISS", correct=0, reason="no json block anywhere in the output", per_task={})
        print(json.dumps(out)); return
    try:
        a = json.loads(block)
    except ValueError as e:
        out.update(verdict="MISS", correct=0, reason=f"json: {e}", per_task={})
        print(json.dumps(out)); return
    tree = d / "tree"
    norm = lambda p: str(p).removeprefix("./")
    ok = {}
    ok["t1"] = sorted((norm(f), int(n), s) for f, n, s in a.get("t1", [])) == sorted(tuple(x) for x in key["t1"])
    ok["t2"] = a.get("t2") == key["t2"]
    ok["t3"] = a.get("t3") == key["t3"]
    ok["t4"] = sorted((norm(f), int(n)) for f, n in a.get("t4", [])) == sorted(tuple(x) for x in key["t4"])
    ok["t5"] = list(a.get("t5", [])) == key["t5"]
    ok["t6"] = norm(str(a.get("t6", ""))).strip() == key["t6"]
    ok["t7"] = str(a.get("t7", "")).strip() == key["t7"]
    ok["t8"] = (a.get("t8", {}).get("ambiguous_exit") == key["t8"]["ambiguous_exit"]
                and (tree / "docs/meta.yaml").read_text() == key["t8"]["final"])
    ok["t9"] = (a.get("t9", {}).get("exit") == key["t9"]["exit"]
                and (tree / "docs/notes.txt").read_text() == key["t9"]["notes"])
    correct = sum(ok.values())
    meets = correct >= 8 and mrw_calls <= 20
    out.update(verdict="MEETS" if meets else "MISS", correct=correct, per_task=ok)
    print(json.dumps(out))


if __name__ == "__main__":
    main()
