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
  cwd, stopping at the first .git; a Bash command's paths resolve from cwd,
  because that is where the command ran, and every other tool's from the root;
- plan headers are tokenised as internal/plan tokenises them, and a plan mrw
  refuses delivers nothing, because it wrote nothing;
- globs match by segment in a table sized (pattern segments x path segments),
  so nothing backtracks and the cost is that product;
- dedup is an atomic O_EXCL claim per rule under a 0700 cache directory that
  is never inside the project; a claim that cannot be filed delivers anyway;
- exit 0 is unconditional, closed stdout included: a hook must never take
  the turn down.

An early delivery is never a silence. Every path the hook takes from a call is
a guess that a file was read — `echo docs/adr/x.md` names the record without
reading it — and a guess that is wrong puts a rule in context a call sooner
than the harness would have. That costs context; a path the hook fails to see
costs the rule. The code below errs on the first side throughout.

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
import time

HANDLED = {"Bash", "Write", "mcp__mrw__mrw_read", "mcp__mrw__mrw_write"}
# An mrw spec is PATH[:RANGE[,RANGE...]]; the range starts with a digit, "-",
# "$" or "/" after the FIRST colon followed by one of those.
_SPEC_SPLIT = re.compile(r":(?=[0-9$/-])")
# mrw prints one header per file it looked at: "==> path  12L  340B  sha ...",
# "==> path  REFUSED ..." or "==> path  UNREADABLE ...". Two spaces end the
# path, so a path with a space in it is read whole.
_SERVED = re.compile(r"^==> (.+?)  \S", re.M)
_BOM = "\ufeff"
_STATE_MAX_AGE = 7 * 24 * 3600
_MAX_TOKENS = 500
_OPS = {"replace", "insert-after", "insert-before", "delete", "create"}
_INT = re.compile(r"[+-]?[0-9]+")  # what strconv.Atoi accepts


def main():
    out = None
    try:
        out = run(json.loads(sys.stdin.read()))
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
    # One base per call. A Bash command's operands, and the headers mrw printed
    # for it, are relative to where the command ran (mrw's own --root defaults
    # to "."); a Write's path, an MCP spec and an MCP result are relative to the
    # project the server serves. A path retried against the other base names a
    # file the call never touched.
    paths = resolve(root, cwd if tool == "Bash" else root, cands)
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
    directory above cwd that holds .claude/rules. cwd itself follows a `cd`
    and is not the project. The walk stops at the first .git it meets: a
    repository without rules of its own does not inherit an enclosing one's."""
    env = os.environ.get("CLAUDE_PROJECT_DIR")
    if env:
        return os.path.realpath(env)
    d = cwd
    while True:
        if os.path.isdir(os.path.join(d, ".claude", "rules")):
            return d
        if os.path.exists(os.path.join(d, ".git")):
            return None
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
    """Paths named by a plan's headers, read as internal/plan reads the plan.

    The tokeniser is splitHeader ported line for line: double quotes only, a
    backslash escapes a quote or a backslash, a /pattern/ address is one token
    with its spaces. The document rules are Parse's: one BOM stripped before
    the header test on every line, a counted body= honoured, a valid header
    inside a counted body refused unless raw=true, text outside any hunk
    refused. The header rules are parseHeader's: at least path, address and
    op; a known op; options key=value, each key once, from the five mrw knows;
    raw=true only with body=. A plan mrw refuses wrote nothing and delivers
    nothing.

    One thing is not mirrored: a pattern address is checked for shape, never
    compiled, because Go's regexp and Python's are not one language. A plan
    mrw refuses for a bad pattern delivers for its paths — early, not silently.
    """
    out = []
    cur, want, fixed, raw = False, -1, False, False
    for line in plan.splitlines():
        hdr = line[1:] if line.startswith(_BOM) else line
        is_hdr = hdr.startswith("@@ ")
        if cur and want > 0:
            if is_hdr and not raw and parse_header(hdr) is not None:
                return []  # an overcounted body= would swallow that hunk
            want -= 1
            continue
        if cur and fixed and want == 0 and not is_hdr:
            if line.strip():
                return []  # body= is satisfied; this line is in no hunk
            continue
        if not is_hdr:
            t = line.strip()
            if not cur and t and not t.startswith("#"):
                return []  # text before the first header
            continue
        h = parse_header(hdr)
        if h is None:
            return []
        path, explicit, raw = h
        out.append(path)
        cur, want, fixed = True, explicit, explicit >= 0
    if cur and fixed and want > 0:
        return []  # body= asked for more lines than the plan holds
    return out


def parse_header(hdr):
    """(path, declared body count or -1, raw) for a header parseHeader
    accepts, else None."""
    fields = split_header(hdr[3:])
    if fields is None or len(fields) < 3 or fields[2] not in _OPS or not valid_addr(fields[1]):
        return None
    explicit, raw, seen = -1, False, set()
    for opt in fields[3:]:
        k, eq, v = opt.partition("=")
        if not eq or k in seen:
            return None
        seen.add(k)
        if k == "sha":
            if len(v) < 8 or v.lower().strip("0123456789abcdef"):
                return None
        elif k == "lines":
            if nonneg(v) is None:
                return None
        elif k == "anchor":
            if v == "":
                return None
        elif k == "raw":
            if v != "true":
                return None
            raw = True
        elif k == "body":
            explicit = nonneg(v)
            if explicit is None:
                return None
        else:
            return None
    if raw and explicit < 0:
        return None
    return fields[0], explicit, raw


def nonneg(v):
    """An integer strconv.Atoi would return, if it is not negative."""
    if not _INT.fullmatch(v) or int(v) < 0:
        return None
    return int(v)


def valid_addr(s):
    """ParseAddr's shapes: "-", "$", N, N-, N-M (either end may be $), or a
    /pattern/ or /from/,/to/ that is closed and non-empty."""
    if s in ("-", "$"):
        return True
    if s.startswith("/"):
        return pattern_shape(s, 0)
    lo, dash, hi = s.partition("-")
    return linenum(lo) and (not dash or hi == "" or linenum(hi))


def linenum(t):
    return t == "$" or nonneg(t) is not None


def pattern_shape(s, depth):
    end = -1
    for i in range(1, len(s)):
        if s[i] == "/" and s[i - 1] != "\\":
            end = i
            break
    if end <= 1:  # never closed, or the empty pattern //
        return False
    rest = s[end + 1:]
    if rest == "":
        return True
    if rest.startswith(",/") and depth == 0:
        return pattern_shape(rest[1:], 1)
    return False


def split_header(s):
    """internal/plan's splitHeader, ported: the fields of a header after
    `@@ `, or None for an unterminated quote."""
    out, cur = [], []
    in_tok = in_q = in_pat = False
    i, n = 0, len(s)
    while i < n:
        r = s[i]
        if in_pat and r == "\\" and i + 1 < n and s[i + 1] == "/":
            cur += [r, s[i + 1]]  # \/ is a literal slash inside a pattern
            i += 1
        elif in_pat and r == "/":
            cur.append(r)
            if i + 2 < n and s[i + 1] == "," and s[i + 2] == "/":
                cur += [",", "/"]  # /a/,/b/ is one address
                i += 2
            else:
                in_pat = False
        elif r == "/" and not in_tok and not in_q:
            in_pat = in_tok = True
            cur.append(r)
        elif r == "\\" and i + 1 < n and s[i + 1] in '"\\':
            cur.append(s[i + 1])
            in_tok = True
            i += 1
        elif r == '"':
            in_q, in_tok = not in_q, True
        elif r in " \t" and not in_q and not in_pat:
            if in_tok:
                out.append("".join(cur))
                cur, in_tok = [], False
        else:
            cur.append(r)
            in_tok = True
        i += 1
    if in_q:
        return None
    if in_tok:
        out.append("".join(cur))
    return out


def resolve(root, base, cands):
    """Root-relative paths that exist, each relative candidate taken from the
    one base its tool implies. Symlinks are resolved before the containment
    check, so a link inside the project pointing outside is refused.
    PostToolUse runs after the write, so a just-created file counts."""
    out = set()
    for c in cands:
        if not c:
            continue
        full = os.path.realpath(c if os.path.isabs(c) else os.path.join(base, c))
        if full.startswith(root + os.sep) and os.path.isfile(full):
            out.add(os.path.relpath(full, root).replace(os.sep, "/"))
    return out


class Rule:
    def __init__(self, rel, globs, body):
        self.rel, self.body = rel, body
        self.globs = [g for pat in globs for g in expand_braces(pat)]

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


def expand_braces(pat):
    """Flat `{a,b}` alternation, expanded before matching so an alternative
    may hold a glob or a slash. One level only: the grammar ADR-022 promises
    has no nested braces, and a `{` inside a group is literal."""
    i = pat.find("{")
    j = pat.find("}", i + 1) if i >= 0 else -1
    if j < 0:
        return [pat]
    head, alts, tail = pat[:i], pat[i + 1:j].split(","), pat[j + 1:]
    return [head + a + t for a in alts for t in expand_braces(tail)]


_SEG = {}


def seg_re(seg):
    """One path segment as a regexp: * and ? never cross a slash. A `**`
    inside a segment is two stars, as Git specifies; braces were expanded
    before the pattern was split, so here they are literal."""
    r = _SEG.get(seg)
    if r is None:
        out = []
        for c in seg:
            out.append("[^/]*" if c == "*" else "[^/]" if c == "?" else re.escape(c))
        r = _SEG[seg] = re.compile("^" + "".join(out) + "$")
    return r


def glob_match(pattern, path):
    """Match by segment. A `**` segment stands for zero or more directories;
    `*` and `?` stay inside one; a slash-less pattern names a root file. One
    row per pattern segment over the path positions, so the cost is the
    product of the two counts and nothing is rescanned. (Git is borrowed for
    the `**` boundary rule and for nothing else: a trailing `dir/` names no
    file here — write `dir/**`.)"""
    pseg = [s for s in pattern.split("/") if s]
    ps = path.split("/")
    n = len(ps)
    ok = [True] + [False] * n  # ok[j]: the pattern so far matches ps[:j]
    for seg in pseg:
        nxt = [False] * (n + 1)
        if seg == "**":
            reach = False
            for j in range(n + 1):
                reach = reach or ok[j]
                nxt[j] = reach
        else:
            r = seg_re(seg)
            for j in range(n):
                if ok[j] and r.match(ps[j]):
                    nxt[j + 1] = True
        ok = nxt
    return ok[n]


def state_dir(root):
    """The claim directory: $XDG_CACHE_HOME, else ~/.cache, then
    claude-rules-on-read, created 0700. A base that is not absolute would
    land under whatever cwd the hook was given, and one inside the project
    would land in the tree; ADR-004 promises the tree is left alone, so both
    are refused and the caller delivers without a claim."""
    base = os.environ.get("XDG_CACHE_HOME") or os.path.join(os.path.expanduser("~"), ".cache")
    if not os.path.isabs(base):
        raise OSError("cache base is not absolute: %r" % base)
    d = os.path.realpath(os.path.join(base, "claude-rules-on-read"))
    if d == root or d.startswith(root + os.sep):
        raise OSError("cache base lies inside the project: %r" % d)
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
    """True when this hook is the first in (session, agent, project) to reach
    this rule: an O_EXCL create is the atomic claim two concurrent hooks race
    for, and one loses. Once per session holds only while a claim can be
    filed. A state directory that cannot be used — no absolute base, a base
    in the tree, a file where the directory should be — delivers on EVERY
    call rather than on none: a repeat costs context, a silence costs the
    rule."""
    try:
        d = state_dir(root)
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
