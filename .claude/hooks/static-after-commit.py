#!/usr/bin/env python3
"""Run scripts/static.sh after a commit and hand the verdict to the model (ADR-088).

PostToolUse on Bash. It analyses HEAD when three things hold:
- the command ran `git commit`, read the way a shell splits it: a quoted `&`
  does not end the command, `echo "git commit"` is not one, and a heredoc
  body is not parsed as commands;
- HEAD's newest reflog entry is a commit (`commit:`, `commit (amend):`, ...)
  made in the last 15 minutes, which is what says a commit landed — a refused
  one leaves no entry;
- no session has analysed that commit yet: one claim file per commit, created
  exclusively, so two sessions never both analyse it and a command in another
  session never consumes it.

The script runs in its own process group, and the whole group is killed at
280 s — inside the 300 s the registration allows — or when this hook is
signalled, so nothing it started outlives the hook.

Exit 0 always, closed stdout included, which is why it ends in os._exit: a
hook must never take the turn down.
"""

import hashlib
import json
import os
import re
import signal
import subprocess
import sys
import tempfile
import time

LIMIT = 6000
TIMEOUT = 280
RECENT = 900
GIT_OPTIONS_WITH_VALUE = {"-c", "-C", "--git-dir", "--work-tree", "--namespace", "--exec-path", "--config-env"}

child = None


def simple_commands(text):
    """Yield the words of each simple command in text, split as a shell splits them.

    Quotes and backslashes are removed from words, so a quoted `;` or `&` is part
    of a word and never ends a command. An unquoted `;`, `&`, `|`, `(`, `)` or
    newline ends one. A backslash-newline joins lines. A `#` starting a word is a
    comment. A heredoc's body — the lines after `<<DELIM` up to DELIM — is text,
    not commands.
    """
    words, cur, has_word, heredocs = [], [], False, []
    i, n = 0, len(text)

    def end_word():
        nonlocal cur, has_word
        if has_word:
            words.append("".join(cur))
        cur, has_word = [], False

    while i < n:
        c = text[i]
        if c == "\\":
            if i + 1 < n and text[i + 1] != "\n":
                cur.append(text[i + 1])
                has_word = True
            i += 2
            continue
        if c == "'":
            j = text.find("'", i + 1)
            j = n if j < 0 else j
            cur.append(text[i + 1:j])
            has_word, i = True, j + 1
            continue
        if c == '"':
            j = i + 1
            while j < n and text[j] != '"':
                if text[j] == "\\" and j + 1 < n and text[j + 1] in '"\\$`\n':
                    j += 1
                cur.append(text[j])
                j += 1
            has_word, i = True, j + 1
            continue
        if c == "#" and not has_word:
            while i < n and text[i] != "\n":
                i += 1
            continue
        if text.startswith("<<<", i):
            end_word()  # a here-string: the next word is its text, no body follows
            i += 3
            continue
        if text.startswith("<<", i):
            end_word()
            i += 2
            strip = i < n and text[i] == "-"
            i += 1 if strip else 0
            while i < n and text[i] in " \t":
                i += 1
            # The delimiter is one shell word: quotes group and are removed.
            delim = []
            while i < n and text[i] not in " \t\n;&|()<>":
                if text[i] == "'":
                    j = text.find("'", i + 1)
                    j = n if j < 0 else j
                    delim.append(text[i + 1:j])
                    i = j + 1
                    continue
                if text[i] == '"':
                    i += 1
                    while i < n and text[i] != '"':
                        if text[i] == "\\" and i + 1 < n and text[i + 1] in '"\\$`':
                            i += 1
                        delim.append(text[i])
                        i += 1
                    i += 1
                    continue
                if text[i] == "\\" and i + 1 < n:
                    i += 1
                delim.append(text[i])
                i += 1
            heredocs.append(("".join(delim), strip))
            continue
        if c == "\n":
            end_word()
            yield words
            words = []
            i += 1
            for delim, strip in heredocs:
                while i < n:
                    j = text.find("\n", i)
                    j = n if j < 0 else j
                    line = text[i:j]
                    i = j + 1
                    if (line.lstrip("\t") if strip else line) == delim:
                        break
            heredocs = []
            continue
        if c in ";&|()":
            end_word()
            yield words
            words = []
            i += 1
            continue
        if c in " \t\r":
            end_word()
            i += 1
            continue
        cur.append(c)
        has_word = True
        i += 1
    end_word()
    yield words


def is_git_commit(words):
    i = 0
    while i < len(words) and re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*=.*", words[i]):
        i += 1  # VAR=value before the command
    if i >= len(words) or os.path.basename(words[i]) != "git":
        return False
    i += 1
    while i < len(words) and words[i].startswith("-"):
        i += 2 if words[i] in GIT_OPTIONS_WITH_VALUE else 1
    return i < len(words) and words[i] == "commit"


def runs_git_commit(command):
    return any(is_git_commit(words) for words in simple_commands(command))


def git(root, *args):
    try:
        p = subprocess.run(["git", "-C", root, *args], capture_output=True, text=True, timeout=10)
    except (OSError, subprocess.SubprocessError):
        return ""
    return p.stdout.strip() if p.returncode == 0 else ""


def recent_commit(root):
    """HEAD's sha when its newest reflog entry is a commit made in the last RECENT seconds."""
    parts = git(root, "log", "-g", "-1", "--date=unix", "--format=%H%x00%gd%x00%gs", "HEAD").split("\0")
    if len(parts) != 3 or not parts[2].startswith("commit"):
        return ""
    m = re.search(r"\{(\d+)\}", parts[1])
    if not m or time.time() - int(m.group(1)) > RECENT:
        return ""
    return parts[0]


def claim(root, sha):
    """True for exactly one caller per commit, across sessions."""
    d = os.path.join(tempfile.gettempdir(), "claude-static-after-commit-claims",
                     hashlib.sha256(os.path.realpath(root).encode()).hexdigest()[:16])
    os.makedirs(d, mode=0o700, exist_ok=True)
    try:
        os.close(os.open(os.path.join(d, sha), os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600))
    except FileExistsError:
        return False
    return True


def kill_group():
    """Kill everything the script started. The group outlives its leader: a
    backgrounded grandchild keeps it, and its stdout, after the shell exits."""
    if child is not None:
        try:
            os.killpg(child.pid, signal.SIGKILL)
        except OSError:
            # The group is already gone (the script and every child exited):
            # nothing is left to kill, and the verdict does not depend on it.
            pass


def on_signal(_signum, _frame):
    kill_group()
    os._exit(0)


def analyse(root, script):
    global child
    try:
        child = subprocess.Popen([script], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                 text=True, start_new_session=True)
    except OSError as e:
        return 1, f"scripts/static.sh could not start: {e}"
    try:
        out, _ = child.communicate(timeout=TIMEOUT)
    except subprocess.TimeoutExpired:
        kill_group()
        try:
            child.communicate(timeout=10)
        except subprocess.TimeoutExpired:
            # The group was killed and still has not closed its pipe: the
            # verdict is already "stopped", so waiting longer changes nothing.
            pass
        return 1, f"scripts/static.sh did not finish in {TIMEOUT} s and was stopped; run it yourself"
    kill_group()  # nothing the script started outlives it
    return child.returncode, out.strip()


def main():
    for s in (signal.SIGTERM, signal.SIGINT, signal.SIGHUP):
        signal.signal(s, on_signal)
    try:
        call = json.load(sys.stdin)
    except (ValueError, OSError):
        return
    command = str((call.get("tool_input") or {}).get("command", ""))
    if not runs_git_commit(command):
        return
    root = os.environ.get("CLAUDE_PROJECT_DIR") or call.get("cwd") or os.getcwd()
    script = os.path.join(root, "scripts", "static.sh")
    if not os.path.isfile(script):
        return
    sha = recent_commit(root)
    if not sha or not claim(root, sha):
        return
    code, out = analyse(root, script)
    if code == 0:
        msg = f"Static analysis after commit {sha[:7]} (scripts/static.sh, ADR-088): clean."
    else:
        if len(out) > LIMIT:
            out = "…" + out[-LIMIT:]
        msg = (f"Static analysis after commit {sha[:7]} (scripts/static.sh, ADR-088) FAILED, "
               f"exit {code}. Fix these in the next commit, or say why not "
               f"(.claude/rules/static-analysis.md):\n{out}")
    sys.stdout.write(json.dumps({"hookSpecificOutput": {"hookEventName": "PostToolUse",
                                                        "additionalContext": msg}}) + "\n")
    sys.stdout.flush()


if __name__ == "__main__":
    try:
        main()
    except BaseException:  # a hook must never take the turn down
        pass
    kill_group()
    os._exit(0)
