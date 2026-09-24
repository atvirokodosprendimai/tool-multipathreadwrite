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
import shlex
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


SEPARATORS = {";", "&", "&&", "|", "||", "|&", ";;", "(", ")"}
ASSIGNMENT = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")


def command_words(cmd):
    """The command word of every pipeline or list segment, quotes removed.

    Quote-aware, so a plan body or a pattern inside printf '...' is never read
    as a command, and "$MRW" read … counts as an invocation of the binary while
    MRW=/path/mrw (an assignment) and ls -la $MRW (an argument) do not. Blind
    reading 02 is void because a regex over the raw command did both wrong: it
    counted `MRW=…/mrw` and missed `"$MRW"`. Leading VAR=value words are
    skipped; `$(` opens a segment.
    """
    # One pass, quote-aware: drop a `#` comment that starts a word outside
    # quotes (through end of line), and turn an unquoted newline into `;`.
    # shlex's own comment handling is off, because once newlines are `;` a
    # comment would swallow the rest of the command (found on reading 02's
    # Haiku transcripts, whose commands carry `# Task 1: …` lines).
    marked, q, comment, prev = [], None, False, "\n"
    for ch in cmd:
        if comment:
            if ch == "\n":
                comment = False
                marked.append(";")
            prev = ch
            continue
        if q is None and ch == "#" and prev in " \t\n;&|(":
            comment = True
            prev = ch
            continue
        if q is None and ch in "'\"":
            q = ch
        elif q is not None and ch == q:
            q = None
        marked.append(";" if (ch == "\n" and q is None) else ch)
        prev = ch
    try:
        lex = shlex.shlex("".join(marked), posix=True, punctuation_chars=True)
        lex.commenters = ""
        lex.whitespace_split = True
        tokens = list(lex)
    except ValueError:
        stripped = re.sub(r"'[^']*'|\"(?:\\.|[^\"\\])*\"", "''", cmd)
        tokens = re.sub(r"(\|\||&&|[|;&\n()])", r" \1 ", stripped).split()
    words, seg = [], []
    for tok in tokens + [";"]:
        if tok in SEPARATORS or (tok and set(tok) <= set(";&|()")):
            while seg and ASSIGNMENT.match(seg[0]):
                seg = seg[1:]
            if seg:
                words.append(seg[0])
            seg = []
        else:
            seg.append(tok)
    return words


def is_mrw(word):
    return word in ("$MRW", "${MRW}", "mrw") or word.endswith("/mrw")


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
        for w in command_words(cmd):
            if w.rsplit("/", 1)[-1] in BANNED_CMDS:
                violations.append(f"command {w.rsplit('/', 1)[-1]}")
            if is_mrw(w):
                mrw_calls += 1
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
