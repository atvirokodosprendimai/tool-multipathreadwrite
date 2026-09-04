#!/usr/bin/env python3
"""Deliver path-scoped .claude/rules/*.md on reads that are not the Read tool.

Claude Code injects a rule whose `paths:` globs match a file ONLY when its own
Read tool reads that file. An mrw read, a Bash read, and a Write deliver none
of them (issue #86, measured 2026-09-04 in two repositories). A repository
that tells sessions to read through mrw therefore loses every path-scoped rule
silently. This PostToolUse hook closes that gap: it takes the paths a tool call
touched, matches them against every rule's globs, and hands the matching rule
bodies back as `additionalContext` -- once per rule per session, which is what
the harness itself does.

It exits 0 on everything. A hook that breaks must not take the turn down with
it, so every failure here is a silent no-op; the contract row asserts the
mechanism, not the hook's opinion of itself.
"""
import hashlib
import json
import os
import re
import shlex
import sys
import tempfile

# Tools whose reads the harness does not follow. Read is the native trigger and
# Edit is refused by the harness unless the file was already Read, so both
# already carry their rules.
HANDLED = {"Bash", "Write", "mcp__mrw__mrw_read", "mcp__mrw__mrw_write"}

# An mrw spec is PATH[:RANGE[,RANGE...]]; the range starts with a digit, "-",
# "$" or "/" after the FIRST colon that is followed by one of those.
_SPEC_SPLIT = re.compile(r":(?=[0-9$/-])")
_HEADER = re.compile(r"^@@ (\S+) \S+ \S+(.*)$")


def main():
    try:
        out = run(json.load(sys.stdin))
    except Exception:  # noqa: BLE001 - see the module comment
        out = None
    if out:
        json.dump({"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": out}}, sys.stdout)
    return 0


def run(data):
    if data.get("hook_event_name") != "PostToolUse":
        return None
    tool = data.get("tool_name", "")
    if tool not in HANDLED:
        return None
    cwd = data.get("cwd") or os.getcwd()
    rules = load_rules(cwd)
    if not rules:
        return None
    paths = resolve(cwd, candidates(tool, data.get("tool_input") or {}))
    if not paths:
        return None
    hits = []
    for rule in rules:
        matched = sorted(p for p in paths if rule.matches(p))
        if matched:
            hits.append((rule, matched))
    hits = undelivered(data.get("session_id", "nosession"), cwd, hits)
    if not hits:
        return None
    parts = []
    for rule, matched in hits:
        parts.append("<!-- %s, delivered by .claude/hooks/rules-on-read.py because %s read %s -->\n%s"
                     % (rule.rel, tool, ", ".join(matched), rule.body.rstrip()))
    return "\n\n".join(parts)


def candidates(tool, inp):
    """The paths a tool call may have touched, as the caller wrote them."""
    if tool == "Write":
        return [inp.get("file_path", "")]
    if tool == "mcp__mrw__mrw_read":
        specs = inp.get("specs") or []
        return [_SPEC_SPLIT.split(s, 1)[0] for s in specs if isinstance(s, str)]
    if tool == "mcp__mrw__mrw_write":
        return plan_paths(inp.get("plan") or "")
    if tool == "Bash":
        cmd = inp.get("command") or ""
        try:
            toks = shlex.split(cmd)
        except ValueError:
            toks = cmd.split()
        out = []
        for t in toks:
            t = _SPEC_SPLIT.split(t, 1)[0]
            if "/" in t or "." in t:
                out.append(t)
        return out
    return []


def plan_paths(plan):
    """Paths named by plan headers. A body under raw=true may itself begin
    with @@ and must not be read as a header, so those bodies are skipped by
    their declared length."""
    out, skip = [], 0
    for line in plan.splitlines():
        if skip:
            skip -= 1
            continue
        m = _HEADER.match(line)
        if not m:
            continue
        out.append(m.group(1))
        guards = m.group(2)
        if "raw=true" in guards:
            n = re.search(r"\bbody=(\d+)", guards)
            skip = int(n.group(1)) if n else 0
    return out


def resolve(cwd, cands):
    """Root-relative paths that exist. PostToolUse runs after the write, so a
    file a plan just created is on disk by now and is matched like any other."""
    root = os.path.realpath(cwd)
    out = set()
    for c in cands:
        if not c:
            continue
        full = os.path.realpath(c if os.path.isabs(c) else os.path.join(root, c))
        if not full.startswith(root + os.sep):
            continue
        if os.path.isfile(full):
            out.add(os.path.relpath(full, root).replace(os.sep, "/"))
    return out


class Rule:
    def __init__(self, rel, globs, body):
        self.rel, self.body = rel, body
        self.res = [glob_re(g) for g in globs]

    def matches(self, path):
        return any(r.match(path) for r in self.res)


def load_rules(cwd):
    base = os.path.join(cwd, ".claude", "rules")
    rules = []
    for dirpath, _, files in os.walk(base):
        for name in sorted(files):
            if not name.endswith(".md"):
                continue
            full = os.path.join(dirpath, name)
            with open(full, encoding="utf-8", errors="replace") as f:
                text = f.read()
            globs, body = frontmatter_paths(text)
            if globs:
                rules.append(Rule(os.path.relpath(full, cwd).replace(os.sep, "/"), globs, body))
    return rules


def frontmatter_paths(text):
    """The `paths:` list of a rule's YAML frontmatter, and the body after it.
    A rule without `paths:` is unconditional and already loaded, so it yields
    no globs. Only the list shapes the docs show are read: `- "glob"`, `- glob`,
    and the inline `paths: ["a", "b"]`."""
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return [], text
    end = next((i for i in range(1, len(lines)) if lines[i].strip() == "---"), None)
    if end is None:
        return [], text
    globs, in_paths = [], False
    for line in lines[1:end]:
        s = line.strip()
        if s.startswith("paths:"):
            rest = s[len("paths:"):].strip()
            if rest.startswith("["):
                globs += [g.strip().strip("\"'") for g in rest.strip("[]").split(",") if g.strip()]
                in_paths = False
            else:
                in_paths = True
            continue
        if in_paths and s.startswith("- "):
            globs.append(s[2:].strip().strip("\"'"))
        elif s and not s.startswith("#"):
            in_paths = False
    return [g for g in globs if g], "\n".join(lines[end + 1:])


def glob_re(glob):
    """gitignore-style glob to a regexp: ** crosses directories, * and ? do
    not, {a,b} alternates. A pattern with no slash matches at the root only,
    as the Claude Code docs describe for `*.md`."""
    i, out = 0, []
    while i < len(glob):
        c = glob[i]
        if glob.startswith("**/", i):
            out.append("(?:.*/)?")
            i += 3
        elif glob.startswith("**", i):
            out.append(".*")
            i += 2
        elif c == "*":
            out.append("[^/]*")
            i += 1
        elif c == "?":
            out.append("[^/]")
            i += 1
        elif c == "{":
            j = glob.find("}", i)
            if j < 0:
                out.append(re.escape(c))
                i += 1
            else:
                out.append("(?:" + "|".join(re.escape(a) for a in glob[i + 1:j].split(",")) + ")")
                i = j + 1
        else:
            out.append(re.escape(c))
            i += 1
    return re.compile("^" + "".join(out) + "$")


def undelivered(session_id, cwd, hits):
    """Filter out rules already delivered in this session, then record the
    rest. State lives outside the tree and is keyed by session and project."""
    key = hashlib.sha1((session_id + "\0" + os.path.realpath(cwd)).encode()).hexdigest()
    state_dir = os.path.join(tempfile.gettempdir(), "claude-rules-on-read")
    os.makedirs(state_dir, exist_ok=True)
    path = os.path.join(state_dir, key)
    seen = set()
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            seen = {l.strip() for l in f if l.strip()}
    fresh = [(r, m) for r, m in hits if r.rel not in seen]
    if fresh:
        with open(path, "a", encoding="utf-8") as f:
            for r, _ in fresh:
                f.write(r.rel + "\n")
    return fresh


if __name__ == "__main__":
    sys.exit(main())
