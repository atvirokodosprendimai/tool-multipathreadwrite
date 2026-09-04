#!/usr/bin/env python3
"""Deliver path-scoped .claude/rules/*.md on reads that are not the Read tool.

Claude Code injects a rule whose `paths:` globs match a file ONLY when its own
Read tool reads that file (issue #86, measured 2026-09-04 in two repositories).
This repository sends sessions through mrw, so it would lose every path-scoped
rule silently. This PostToolUse hook is ADR-022: it takes the paths a Bash,
Write, mrw_read or mrw_write call NAMED and the paths its result SERVED (mrw
prints `==> path` per served file, which is where a grep's files are), matches
them against every rule's globs, and returns the matching bodies as
`additionalContext` once per rule per session per agent — the harness's own
behaviour, extended to the reads it does not follow.

Decisions the record makes and this file keeps:
- the project root is $CLAUDE_PROJECT_DIR, else the nearest .claude/rules above
  cwd; relative paths resolve from cwd, because cwd follows a `cd`;
- plan headers are read as mrw reads them (BOM stripped, quoted first field,
  every body=N honoured, raw=true without body= delivers nothing);
- globs match by segment, so `**` crosses directories only at a boundary and
  nothing can backtrack;
- dedup is an atomic O_EXCL claim per rule under a 0700 cache directory;
- exit 0 is unconditional, closed stdout included: a hook must never take
  the turn down.

The matcher in .claude/settings.json names the MCP tools as mcp__mrw__*, which
assumes the server is registered as `mrw`. Another registration name gets the
Bash and Write half only.
"""
import hashlib
import json
import os
import re
import shlex
import sys
import tempfile
import time

HANDLED = {"Bash", "Write", "mcp__mrw__mrw_read", "mcp__mrw__mrw_write"}
# An mrw spec is PATH[:RANGE[,RANGE...]]; the range starts with a digit, "-",
# "$" or "/" after the FIRST colon followed by one of those.
_SPEC_SPLIT = re.compile(r":(?=[0-9$/-])")
# mrw read prints one header per served file: "==> path  12L  340B  sha ...".
_SERVED = re.compile(r"^==> (\S+)\s", re.M)
_BOM = "\ufeff"
_STATE_MAX_AGE = 7 * 24 * 3600
_MAX_INPUT = 8 << 20
_MAX_TOKENS = 500


def main():
    out = None
    try:
        out = run(json.loads(sys.stdin.read(_MAX_INPUT)))
    except Exception:  # noqa: BLE001 - see the module comment
        out = None
    if out:
        try:
            sys.stdout.write(json.dumps({"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": out}}))
            sys.stdout.flush()
        except Exception:  # noqa: BLE001 - a closed stdout is not our turn to end
            pass
    # No interpreter-shutdown flush, which is what turned a closed pipe into
    # exit 120 in the first cut.
    os._exit(0)


def run(data):
    if data.get("hook_event_name") != "PostToolUse":
        return None
    tool = data.get("tool_name", "")
    if tool not in HANDLED:
        return None
    cwd = os.path.realpath(data.get("cwd") or os.getcwd())
    root = project_root(cwd)
    if not root:
        return None
    rules = load_rules(root)
    if not rules:
        return None
    cands = candidates(tool, data.get("tool_input") or {}) + served_paths(data.get("tool_response"))
    paths = resolve(root, cwd, cands)
    if not paths:
        return None
    session = data.get("session_id") or "nosession"
    agent = data.get("agent_id") or ""
    parts = []
    for rule in rules:
        matched = sorted(p for p in paths if rule.matches(p))
        if matched and claim(session, agent, root, rule.rel):
            parts.append("<!-- %s, delivered by .claude/hooks/rules-on-read.py because %s read %s -->\n%s"
                         % (rule.rel, tool, ", ".join(matched), rule.body.rstrip()))
    return "\n\n".join(parts) if parts else None


def project_root(cwd):
    """$CLAUDE_PROJECT_DIR, which Claude Code sets for hooks, else the nearest
    directory above cwd that holds .claude/rules. cwd itself follows a `cd` and
    is not the project."""
    env = os.environ.get("CLAUDE_PROJECT_DIR")
    if env:
        return os.path.realpath(env)
    d = cwd
    while True:
        if os.path.isdir(os.path.join(d, ".claude", "rules")):
            return d
        parent = os.path.dirname(d)
        if parent == d:
            return None
        d = parent


def candidates(tool, inp):
    """The paths a tool call NAMED, as the caller wrote them."""
    if tool == "Write":
        return [inp.get("file_path") or ""]
    if tool == "mcp__mrw__mrw_read":
        return [_SPEC_SPLIT.split(s, 1)[0] for s in (inp.get("specs") or []) if isinstance(s, str)]
    if tool == "mcp__mrw__mrw_write":
        return plan_paths(inp.get("plan") or "")
    if tool == "Bash":
        return bash_paths(inp.get("command") or "")
    return []


def bash_paths(cmd):
    """Every token of a shell command; existence decides later which are
    paths. A leading `cd DIR &&` (or `;`) moves the base later tokens resolve
    against, because the command's own reads happened there."""
    try:
        toks = shlex.split(cmd)
    except ValueError:
        toks = cmd.split()
    toks = toks[:_MAX_TOKENS]
    base = ""
    if len(toks) >= 3 and toks[0] == "cd" and toks[2] in ("&&", ";"):
        base, toks = toks[1], toks[3:]
    out = []
    for t in toks:
        t = _SPEC_SPLIT.split(t, 1)[0]
        if not t or t.startswith("-"):
            continue
        out.append(t if (os.path.isabs(t) or not base) else os.path.join(base, t))
    return out


def served_paths(resp):
    """Paths a tool RESULT names as served — every `==> path` header in any
    string anywhere in the response, whatever shape the tool gave it."""
    out, stack = [], [resp]
    while stack:
        node = stack.pop()
        if isinstance(node, str):
            out += _SERVED.findall(node)
        elif isinstance(node, dict):
            stack.extend(node.values())
        elif isinstance(node, list):
            stack.extend(node)
    return out


def plan_paths(plan):
    """Paths named by plan headers, read as mrw reads them: a BOM stripped, the
    first field quoted or bare, every declared body=N skipped whether or not
    raw=true, and raw=true without body= — a plan mrw refuses outright —
    delivering nothing."""
    out, skip = [], 0
    for line in plan.splitlines():
        if skip:
            skip -= 1
            continue
        line = line.lstrip(_BOM)
        if not line.startswith("@@ "):
            continue
        try:
            toks = shlex.split(line[3:])
        except ValueError:
            continue
        if len(toks) < 3:
            continue
        guards = {}
        for g in toks[3:]:
            k, _, v = g.partition("=")
            guards[k] = v
        if guards.get("raw") == "true" and "body" not in guards:
            return []
        if "body" in guards:
            try:
                skip = int(guards["body"])
            except ValueError:
                skip = 0
        out.append(toks[0])
    return out


def resolve(root, cwd, cands):
    """Root-relative paths that exist. A relative candidate is tried from cwd
    first (a Bash read happened there) and then from the root (an MCP or
    served path is root-relative). Symlinks are resolved before the
    containment check, so a link inside the project pointing outside is
    refused. PostToolUse runs after the write, so a just-created file counts."""
    out = set()
    for c in cands:
        if not c:
            continue
        tries = [c] if os.path.isabs(c) else [os.path.join(cwd, c), os.path.join(root, c)]
        for t in tries:
            full = os.path.realpath(t)
            if full.startswith(root + os.sep) and os.path.isfile(full):
                out.add(os.path.relpath(full, root).replace(os.sep, "/"))
                break
    return out


class Rule:
    def __init__(self, rel, globs, body):
        self.rel, self.body, self.globs = rel, body, globs

    def matches(self, path):
        return any(glob_match(g, path) for g in self.globs)


def load_rules(root):
    base = os.path.join(root, ".claude", "rules")
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
                rules.append(Rule(os.path.relpath(full, root).replace(os.sep, "/"), globs, body))
    return rules


def frontmatter_paths(text):
    """The `paths:` list of a rule's frontmatter and the body after it. Reads
    the shapes the docs show — a block list of quoted or bare items, or an
    inline list — brace-aware, with a trailing `# comment` dropped. A rule
    without `paths:` is unconditional and already loaded: no globs."""
    lines = text.replace("\r\n", "\n").splitlines()
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
                globs += split_list(rest.strip()[1:].rstrip("]"))
                in_paths = False
            else:
                in_paths = True
            continue
        if in_paths and s.startswith("- "):
            globs += split_list(s[2:])
        elif s and not s.startswith("#"):
            in_paths = False
    return [g for g in globs if g], "\n".join(lines[end + 1:])


def split_list(s):
    """Comma-separated items, quotes honoured, commas inside {} kept, and an
    unquoted trailing `# comment` dropped."""
    items, cur, depth, quote = [], "", 0, None
    i = 0
    while i < len(s):
        ch = s[i]
        if quote:
            if ch == quote:
                quote = None
            else:
                cur += ch
        elif ch in "\"'":
            quote = ch
        elif ch == "{":
            depth += 1
            cur += ch
        elif ch == "}":
            depth = max(0, depth - 1)
            cur += ch
        elif ch == "," and depth == 0:
            items.append(cur.strip())
            cur = ""
        elif ch == "#" and (i == 0 or s[i - 1].isspace()):
            break
        else:
            cur += ch
        i += 1
    items.append(cur.strip())
    return [it for it in items if it]


_SEG = {}


def seg_re(seg):
    """One path segment as a regexp: * and ? never cross a slash, {a,b}
    alternates. A `**` inside a segment is two stars, as Git specifies."""
    r = _SEG.get(seg)
    if r is None:
        i, out = 0, []
        while i < len(seg):
            c = seg[i]
            if c == "*":
                out.append("[^/]*")
            elif c == "?":
                out.append("[^/]")
            elif c == "{":
                j = seg.find("}", i)
                if j < 0:
                    out.append(re.escape(c))
                else:
                    out.append("(?:" + "|".join(re.escape(a) for a in seg[i + 1:j].split(",")) + ")")
                    i = j
            else:
                out.append(re.escape(c))
            i += 1
        r = _SEG[seg] = re.compile("^" + "".join(out) + "$")
    return r


def glob_match(pattern, path):
    """gitignore-style match by segment: `**` at a boundary matches zero or
    more directories; a slash-less pattern is root-only. Memoised over
    (pattern segment, path segment), so no input can make it backtrack."""
    pseg = [s for s in pattern.split("/") if s]
    ps = path.split("/")
    memo = {}

    def m(i, j):
        k = (i, j)
        if k in memo:
            return memo[k]
        if i == len(pseg):
            r = j == len(ps)
        elif pseg[i] == "**":
            r = any(m(i + 1, x) for x in range(j, len(ps) + 1))
        elif j < len(ps) and seg_re(pseg[i]).match(ps[j]):
            r = m(i + 1, j + 1)
        else:
            r = False
        memo[k] = r
        return r

    return m(0, 0)


def state_dir():
    base = os.environ.get("XDG_CACHE_HOME")
    if not base:
        home = os.path.expanduser("~")
        base = os.path.join(home, ".cache") if home and home != "~" else tempfile.gettempdir()
    d = os.path.join(base, "claude-rules-on-read")
    os.makedirs(d, mode=0o700, exist_ok=True)
    try:
        os.chmod(d, 0o700)
    except OSError:
        pass
    return d


def sweep(d):
    cutoff = time.time() - _STATE_MAX_AGE
    try:
        for name in os.listdir(d):
            p = os.path.join(d, name)
            if os.path.isfile(p) and os.path.getmtime(p) < cutoff:
                os.remove(p)
    except OSError:
        pass


def claim(session, agent, root, rel):
    """True exactly once per (session, agent, project, rule): an O_EXCL create
    is the atomic claim two concurrent hooks race for, and one loses. A state
    directory that cannot be used delivers rather than suppresses — a repeat
    beats a silence."""
    try:
        d = state_dir()
        sweep(d)
        key = hashlib.sha1("\0".join((session, agent, root, rel)).encode()).hexdigest()
        flags = os.O_CREAT | os.O_EXCL | os.O_WRONLY | getattr(os, "O_NOFOLLOW", 0)
        os.close(os.open(os.path.join(d, key), flags, 0o600))
        return True
    except FileExistsError:
        return False
    except OSError:
        return True


if __name__ == "__main__":
    main()
