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
            # Any fenced json block, object or not: the last one is the answer.
            # Matching only {...} skipped a final non-object fence and let an
            # earlier block win (Codex review of #206; ADR-070 T3).
            blocks = re.findall(r"```json\s*(.*?)\s*```", s, re.S)
            if blocks:
                found = blocks[-1]
    return found


SEPARATORS = {";", "&", "&&", "|", "||", "|&", ";;", "(", ")"}
ASSIGNMENT = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")
# Words that run the NEXT word as the command: `command cat f` runs cat, and
# `env mrw read f` runs mrw (Codex review of #206; ADR-070 T3).
WRAPPERS = {"command", "builtin", "exec", "env", "nohup", "time", "sudo", "xargs"}
HEREDOC = re.compile(r"(?<!<)<<(?!<)(-?)\s*(['\"]?)([A-Za-z_][A-Za-z0-9_]*)\2")


def unquoted_spans(line):
    """Positions in line that are outside single and double quotes."""
    out, q, esc = set(), None, False
    for i, ch in enumerate(line):
        if esc:
            esc = False
            continue
        if ch == "\\" and q != "'":
            esc = True
            continue
        if q is None and ch in "'\"":
            q = ch
            continue
        if q is not None and ch == q:
            q = None
            continue
        if q is None:
            out.add(i)
    return out


def strip_heredocs(cmd):
    """Drop every heredoc body: its lines are data, not commands. Reading 03
    parsed each body line as a command (ADR-070 T3)."""
    lines, out, i = cmd.split("\n"), [], 0
    while i < len(lines):
        line = lines[i]
        out.append(line)
        i += 1
        free = unquoted_spans(line)
        for m in HEREDOC.finditer(line):
            if m.start() not in free:
                continue
            dash, word = m.group(1), m.group(3)
            while i < len(lines):
                body = lines[i]
                i += 1
                if (body.lstrip("\t") if dash else body) == word:
                    break
    return "\n".join(out)


def segments(cmd):
    """The token list of every pipeline or list segment, quotes removed, with
    the words of every `$(…)` inside double quotes as segments of their own.

    Quote-aware, so a plan body or a pattern inside printf '...' is never read
    as a command, and "$MRW" read … counts as an invocation of the binary while
    MRW=/path/mrw (an assignment) and ls -la $MRW (an argument) do not. Blind
    reading 02 is void because a regex over the raw command did both wrong: it
    counted `MRW=…/mrw` and missed `"$MRW"`. Leading VAR=value words are
    skipped; `$(` opens a segment.
    """
    cmd = strip_heredocs(cmd)
    # One pass, quote-aware: drop a `#` comment that starts a word outside
    # quotes (through end of line), and turn an unquoted newline into `;`.
    # shlex's own comment handling is off, because once newlines are `;` a
    # comment would swallow the rest of the command (found on reading 02's
    # Haiku transcripts, whose commands carry `# Task 1: …` lines). A newline
    # after a backslash continues the command (reading 03), and a `$(…)` inside
    # double quotes is a command of its own (Codex review of #206).
    marked, q, comment, prev, esc, inner = [], None, False, "\n", False, []
    i = 0
    while i < len(cmd):
        ch = cmd[i]
        i += 1
        if comment:
            if ch == "\n":
                comment = False
                marked.append(";")
            prev = ch
            continue
        if esc:
            esc = False
            if ch == "\n" and q is None:
                marked.pop()  # the backslash
                marked.append(" ")
            else:
                marked.append(ch)
            prev = ch
            continue
        if ch == "\\" and q != "'":
            esc = True
            marked.append(ch)
            prev = ch
            continue
        if q == '"' and ch == "$" and i < len(cmd) and cmd[i] == "(":
            depth, j = 1, i + 1
            while j < len(cmd) and depth:
                depth += {"(": 1, ")": -1}.get(cmd[j], 0)
                j += 1
            inner.append(cmd[i + 1 : j - 1])
            marked.append("_")
            i = j
            prev = ")"
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
    out, seg = [], []
    for tok in tokens + [";"]:
        if tok in SEPARATORS or (tok and set(tok) <= set(";&|()")):
            while seg and (ASSIGNMENT.match(seg[0]) or seg[0] in WRAPPERS):
                seg = seg[1:]
                while seg and seg[0].startswith("-"):
                    seg = seg[1:]  # a wrapper's own flags: env -i, xargs -n1
            if seg:
                out.append(seg)
            seg = []
        else:
            seg.append(tok)
    for s in inner:
        out.extend(segments(s))
    return out


def command_words(cmd):
    """The command word of every segment (see segments)."""
    return [s[0] for s in segments(cmd)]


def is_mrw(word):
    return word in ("$MRW", "${MRW}", "mrw") or word.endswith("/mrw")


def check(f):
    """A task check that cannot crash the scorer: an answer of the wrong type
    is a miss, not a traceback (Codex review of #206; ADR-070 T3)."""
    try:
        return bool(f())
    except Exception:
        return False


def score(d, transcript):
    key = json.loads((d / "answer-key.json").read_text())
    out = {"trial": d.name}
    try:
        calls = tool_calls(transcript)
    except OSError as e:
        out.update(verdict="VOID", reason=f"transcript unreadable: {e}")
        return out
    if not calls:
        out.update(verdict="VOID", reason="transcript holds no tool calls")
        return out
    violations, mrw_calls = [], 0
    for name, inp in calls:
        if name in BANNED_TOOLS:
            violations.append(f"tool {name}")
        if name != "Bash":
            continue
        for seg in segments(inp.get("command", "")):
            w = seg[0]
            if w.rsplit("/", 1)[-1] in BANNED_CMDS:
                violations.append(f"command {w.rsplit('/', 1)[-1]}")
            if is_mrw(w):
                mrw_calls += 1
            # Any --help is banned (the criterion), as an argument WORD of any
            # command. A raw substring test also voided text that only
            # mentions it, `echo "see --help"` (Codex review of #206; ADR-070 T3).
            if any(tok == "--help" or tok.startswith("--help=") for tok in seg[1:]):
                violations.append("--help")
    out["mrw_calls"] = mrw_calls
    if violations:
        out.update(verdict="VOID", reason="banned: " + ", ".join(sorted(set(violations))))
        return out
    block = final_answer(transcript)
    if block is None:
        out.update(verdict="MISS", correct=0, reason="no json block anywhere in the output", per_task={})
        return out
    try:
        a = json.loads(block)
    except ValueError as e:
        out.update(verdict="MISS", correct=0, reason=f"json: {e}", per_task={})
        return out
    if not isinstance(a, dict):
        out.update(verdict="MISS", correct=0, reason="the final json block is not an object", per_task={})
        return out
    tree = d / "tree"
    norm = lambda p: str(p).removeprefix("./")
    ok = {}
    ok["t1"] = check(lambda: sorted((norm(f), int(n), s) for f, n, s in a.get("t1", [])) == sorted(tuple(x) for x in key["t1"]))
    ok["t2"] = check(lambda: a.get("t2") == key["t2"])
    ok["t3"] = check(lambda: a.get("t3") == key["t3"])
    ok["t4"] = check(lambda: sorted((norm(f), int(n)) for f, n in a.get("t4", [])) == sorted(tuple(x) for x in key["t4"]))
    ok["t5"] = check(lambda: list(a.get("t5", [])) == key["t5"])
    ok["t6"] = check(lambda: norm(str(a.get("t6", ""))).strip() == key["t6"])
    ok["t7"] = check(lambda: str(a.get("t7", "")).strip() == key["t7"])
    ok["t8"] = check(lambda: a.get("t8", {}).get("ambiguous_exit") == key["t8"]["ambiguous_exit"]
                     and (tree / "docs/meta.yaml").read_text() == key["t8"]["final"])
    ok["t9"] = check(lambda: a.get("t9", {}).get("exit") == key["t9"]["exit"]
                     and (tree / "docs/notes.txt").read_text() == key["t9"]["notes"])
    correct = sum(ok.values())
    # At least one mrw call: 8 correct answers with no call at all did not use
    # the tool the bench measures (pre-registered in BACKLOG before reading 05).
    meets = correct >= 8 and 1 <= mrw_calls <= 20
    out.update(verdict="MEETS" if meets else "MISS", correct=correct, per_task=ok)
    return out


def main():
    print(json.dumps(score(Path(sys.argv[1]), Path(sys.argv[2]))))


if __name__ == "__main__":
    main()
