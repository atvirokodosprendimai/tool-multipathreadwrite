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
  cwd, stopping at the first .git; a Bash command's OPERANDS resolve from where
  the command ran — cwd, moved by a leading `cd` — and the headers mrw PRINTED
  for it from the root mrw was given; every other tool's paths from the root;
- plan headers are tokenised as internal/plan tokenises them, and whether mrw
  accepts the plan is not mirrored: the first field of every header-shaped line
  the tokeniser can split is
  a candidate;
- globs match by segment in a table sized (pattern segments x path segments),
  each segment by a two-pointer walk, so nothing backtracks;
- dedup is an atomic O_EXCL claim per rule under a 0700 cache directory that
  is never inside the project; a claim that cannot be filed delivers anyway,
  and a claim whose envelope never reached the harness is withdrawn;
- exit 0 is unconditional, closed stdout included: a hook must never take
  the turn down.

An early delivery is never a silence. Every path the hook takes from a call is
a guess that a file was read — `echo docs/adr/x.md` names the record without
reading it — and a guess that is wrong puts a rule in context a call sooner
than the harness would have. That costs context; a path the hook fails to see
costs the rule. The code below errs on the first side throughout, and the
third-round mirror of mrw's plan acceptance was removed for exactly that
reason: everywhere it was stricter than mrw, a successful write delivered
nothing.

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
# mrw prints one header per file it SERVED: "==> path  12L  340B  sha abcd1234".
# The path is read back from that suffix, greedily, so any run of spaces inside
# it survives; a REFUSED or UNREADABLE header served nothing and names nothing.
_SERVED = re.compile(r"^==> (.+)  \d+L  \d+B  sha [0-9a-f]+$", re.M)
# mrw's own global root flag, which comes BEFORE the subcommand (after it, -C
# is the integer context flag). The paths mrw prints are relative to it.
_MRW_ROOT = re.compile(r"^(?:--root|-C)(?:=(.*))?$")
# `env`'s own --chdir moves where the command runs, exactly as a leading `cd`
# does, so it moves both bases.
_ENV_CHDIR = re.compile(r"^(?:--chdir|-C)(?:=(.*))?$")
_ENV_NAMES = {"env"}
# A leading NAME=VALUE is an assignment, not the command and not a path.
_ASSIGN = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")
# Shell control operators, which need no whitespace around them.
_CTRL = re.compile(r"([;&|]+)")
_MRW_NAMES = {"mrw", "mrw.exe"}
_BOM = "\ufeff"
_STATE_MAX_AGE = 7 * 24 * 3600


def main():
    out, claimed = None, []
    try:
        out, claimed = run(json.loads(sys.stdin.read()))
    except Exception:  # noqa: BLE001 - see the module comment
        out = None
    if out:
        try:
            sys.stdout.write(json.dumps({"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": out}}))
            sys.stdout.flush()
        except Exception:  # noqa: BLE001 - a closed stdout is not our turn to end
            # The envelope never reached the harness, so the claims this call
            # filed would suppress the next real read for seven days. Withdraw
            # them: a repeat, never a silence.
            for p in claimed:
                try:
                    os.remove(p)
                except OSError:
                    pass
    # No interpreter-shutdown flush, which is what turned a closed pipe into
    # exit 120 in the first cut.
    os._exit(0)


def run(data):
    """The additionalContext to deliver (or None) and the claim files this
    call created, so main() can withdraw them if the delivery fails."""
    if data.get("hook_event_name") != "PostToolUse":
        return None, []
    tool = data.get("tool_name", "")
    if tool not in HANDLED:
        return None, []
    cwd = os.path.realpath(data.get("cwd") or os.getcwd())
    root = project_root(cwd)
    if not root:
        return None, []
    rules = load_rules(root)
    if not rules:
        return None, []
    inp = data.get("tool_input") or {}
    # Every base a call plausibly used, because choosing one is how a path
    # resolves against the wrong directory and its rule goes missing, and a
    # base that was never used costs an early delivery at worst — the side
    # Decision 2 chose. For Bash: the directory the command ran in (cwd, and
    # every directory a leading `cd` or an `env --chdir` names), and for the
    # headers mrw PRINTED, those directories crossed with every root any mrw in
    # the command was given. For every other tool the project root, which is
    # what the server serves.
    if tool == "Bash":
        dirs, mrw_roots, cands = bash_paths(inp.get("command") or "")
        bases = list(dict.fromkeys(os.path.join(cwd, d) for d in dirs))
        served_bases = bases + [os.path.join(b, r) for b in bases for r in mrw_roots]
    else:
        cands, bases = candidates(tool, inp), [root]
        served_bases = [root]
    paths = resolve(root, bases, cands) | resolve(root, served_bases, served_paths(data.get("tool_response")))
    if not paths:
        return None, []
    session = data.get("session_id") or "nosession"
    agent = data.get("agent_id") or ""
    parts, claimed = [], []
    for rule in rules:
        matched = sorted(p for p in paths if rule.matches(p))
        if not matched:
            continue
        deliver, path = claim(session, agent, root, rule.rel)
        if deliver:
            if path:
                claimed.append(path)
            parts.append("<!-- %s, delivered by .claude/hooks/rules-on-read.py because %s read %s -->\n%s"
                         % (rule.rel, tool, ", ".join(matched), rule.body.rstrip()))
    return ("\n\n".join(parts) if parts else None), claimed


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
    """The paths a non-Bash tool call NAMED, as the caller wrote them."""
    if tool == "Write":
        return [inp.get("file_path") or ""]
    if tool == "mcp__mrw__mrw_read":
        return [_SPEC_SPLIT.split(s, 1)[0] for s in (inp.get("specs") or []) if isinstance(s, str)]
    if tool == "mcp__mrw__mrw_write":
        return plan_paths(inp.get("plan") or "")
    return []


def bash_paths(cmd):
    """Every directory the command may have run in relative to cwd, every root
    an mrw call gave its own `--root`/`-C` flag, and every other token of the
    command. Existence decides later which tokens are paths; there is no token
    cap, because a cap is a silent way to lose the last operand.

    Both lists are plural on purpose. `env -C a -C b` runs in `b`, a leading
    `cd x &&` before it runs in `x/b`, and a command may call mrw twice with
    two roots — so a single answer is a guess, and the wrong guess loses the
    rule the real read served. Every candidate is returned and the caller tries
    them all: a directory the command never used simply matches no file.

    mrw is found by NAME anywhere in the command rather than by position. The
    positional reading this replaced had to know every wrapper — `env`,
    `command`, `nice` — and every wrapper's own flags, and a review found
    eight valid shapes it still missed (`command --`, `time -p`, `nice -n 5`,
    `/usr/bin/env`, …), each of them a silently missing rule. Finding the name
    needs no vocabulary at all. A stray token that merely SPELLS mrw (`echo mrw
    --root ..`) can name a root the command never used, which is why every root
    is returned rather than one: a false one delivers a rule early at worst.

    ⚠ This is a heuristic reading of a shell command, not a shell parser. It
    knows quoting, control operators, assignments, `cd` and the two flags above.
    A shape outside that — a subshell that changes directory, a variable holding
    the path, a here-doc — yields no candidate from the COMMAND, and the served
    `==>` header only makes up for it when the header reaches the RESULT and its
    path resolves against a directory this reading did recognise. Neither is
    guaranteed: `(cd docs && mrw read adr/x.md)` prints a header this hook
    cannot place, and a redirected or captured read prints none at all. Those
    reads deliver nothing, the session falls back to reading the rule itself,
    and ADR-022's Out of Scope says so."""
    try:
        toks = shlex.split(cmd)
    except ValueError:
        toks = cmd.split()
    # A control operator needs no whitespace around it: `read x.md:1;mrw -C ..`
    # leaves `;mrw` as one token, and a name-based scan then misses the second
    # call entirely. The scan reads the split parts; the composite token is kept
    # only as a CANDIDATE PATH, because a quoted operand may legitimately hold
    # one (`cat 'a;b.txt'`) — putting it back in the token stream would sit a
    # bare word after a program name and end its flag scan.
    split, composite = [], []
    for t in toks:
        parts = [p for p in _CTRL.split(t) if p]
        split.extend(parts)
        if len(parts) > 1:
            composite.append(t)
    toks = split
    dirs = [""]
    if len(toks) >= 3 and toks[0] == "cd" and toks[2] in ("&&", ";"):
        dirs, toks = ["", toks[1]], toks[3:]
    taken = set()

    def flag_value(j, pat):
        """(value, index consumed) for `--flag=V` or `--flag V` at j."""
        m = pat.match(toks[j])
        if not m:
            return None, j
        taken.add(j)
        if m.group(1) is not None:
            return m.group(1), j
        if j + 1 < len(toks):
            taken.add(j + 1)
            return toks[j + 1], j + 1
        return None, j

    def scan(names, pat):
        """The values of `pat` given to each program called one of `names`,
        one list per invocation, reading only that program's own flags — a bare
        word ends them, because that is the subcommand or the wrapped command.
        Every occurrence is read, not just the first: `mrw -C a read x ; mrw -C
        b read y` is two calls, and the second read's header is relative to the
        second."""
        found = []
        for n, t in enumerate(toks):
            if os.path.basename(t) not in names:
                continue
            here, j = [], n + 1
            while j < len(toks):
                v, j = flag_value(j, pat)
                if v:
                    here.append(v)
                elif v is None and not toks[j].startswith("-") and not _ASSIGN.match(toks[j]):
                    break
                j += 1
            found.append(here)
        return found

    # env --chdir runs the command elsewhere, so it moves the base for the
    # operands too; mrw's --root moves only the paths mrw prints. Both are read
    # only for the program that owns the flag: -C means something else to git,
    # make and tar, and a program this hook does not recognise gets neither.
    #
    # One `env` takes the LAST of its own --chdir flags, relative to where it
    # started; a `cd` or a nested `env` before it composes. Both readings are
    # offered — the running composition, and each value alone — which is two
    # candidates per invocation. ⚠ The first cut of this offered every value
    # against every directory so far, which DOUBLES the list per flag: a review
    # measured 262,144 candidates from eighteen flags, enough to spend the 10 s
    # budget and deliver nothing. Linear, not exponential.
    cur = dirs[-1]
    for here in scan(_ENV_NAMES, _ENV_CHDIR):
        if not here:
            continue
        cur = os.path.join(cur, here[-1])
        dirs.append(cur)
        dirs.append(here[-1])
    mrw_roots = [v for here in scan(_MRW_NAMES, _MRW_ROOT) for v in here]
    out = list(composite)
    for n, t in enumerate(toks):
        if n in taken or not t or t.startswith("-"):
            continue
        # A token is offered BOTH as written and with an mrw range stripped:
        # `docs/adr/x.md:1-3` is a spec whose file is the part before the colon,
        # and `docs/adr/note:1` is a filename a shell command may read whole.
        # Only one of the two will exist, and resolve() keeps that one.
        out.append(t)
        stripped = _SPEC_SPLIT.split(t, 1)[0]
        if stripped != t and stripped:
            out.append(stripped)
    return dirs, mrw_roots, out


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
    """The first field of every header-shaped line of a plan, tokenised as
    internal/plan tokenises a header (split_header decides WHICH string is the
    path). Whether mrw would ACCEPT the plan is not mirrored — not a counted
    body, not an op, not a guard, not an address: a body line that looks like
    a header delivers early for a file the plan did not touch, and a plan mrw
    refuses delivers early for the files it names. The mirror this replaced
    could only add silence: everywhere it was stricter than mrw, a successful
    write delivered nothing. A BOM is stripped once, as mrw strips it."""
    out = []
    for line in plan.splitlines():
        hdr = line[1:] if line.startswith(_BOM) else line
        if hdr.startswith("@@ "):
            fields = split_header(hdr[3:])
            if fields:
                out.append(fields[0])
    return out


def split_header(s):
    """internal/plan's splitHeader, ported: the fields of a header after
    `@@ `, or None for an unterminated quote. Double quotes only; a backslash
    escapes a quote or a backslash; a /pattern/ address is one token with its
    spaces, and /a/,/b/ is one address."""
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


def resolve(root, bases, cands):
    """Root-relative paths that exist, each relative candidate tried against
    every base its tool implies. Symlinks are resolved before the containment
    check, so a link inside the project pointing outside is refused.
    PostToolUse runs after the write, so a just-created file counts."""
    out = set()
    for c in cands:
        if not c:
            continue
        for base in bases:
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
                # The comment goes before the brackets do: stripping `]` first
                # left `docs/adr/**]` for `paths: ["docs/adr/**"] # a comment`,
                # a glob that matches nothing and says nothing.
                inner = split_list(rest[1:])
                if inner and inner[-1].endswith("]"):
                    inner[-1] = inner[-1][:-1]
                globs += [g for g in inner if g]
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
    may hold a glob or a slash. One level only, as ADR-022 promises: a pattern
    holding a group inside a group, or a group never closed, is literal — and
    that is decided for the WHOLE pattern before any group is expanded, since
    expanding a flat group first would leave a half-expanded pattern behind."""
    depth = 0
    for c in pat:
        if c == "{":
            depth += 1
            if depth > 1:
                return [pat]
        elif c == "}":
            depth = max(0, depth - 1)
    if depth:
        return [pat]
    return _expand_flat(pat)


def _expand_flat(pat):
    i = pat.find("{")
    j = pat.find("}", i + 1) if i >= 0 else -1
    if j < 0:
        return [pat]
    head, alts, tail = pat[:i], pat[i + 1:j].split(","), pat[j + 1:]
    return [head + a + t for a in alts for t in _expand_flat(tail)]


def seg_match(pat, s):
    """`*` and `?` inside one path segment, by the two-pointer walk: a failed
    literal falls back to the last `*` and lets it swallow one more character.
    The cost is bounded by len(pat) x len(s); a regex of `[^/]*` runs, which
    this replaced, backtracked past a 2 s alarm on sixteen stars. Braces were
    expanded before the pattern was split, so `{` and `}` are literal here,
    and a `**` inside a segment is two stars, as Git specifies."""
    p = i = 0
    star = mark = -1
    while i < len(s):
        if p < len(pat) and pat[p] == "*":
            star, mark = p, i
            p += 1
        elif p < len(pat) and (pat[p] == "?" or pat[p] == s[i]):
            p += 1
            i += 1
        elif star >= 0:
            mark += 1
            p, i = star + 1, mark
        else:
            return False
    while p < len(pat) and pat[p] == "*":
        p += 1
    return p == len(pat)


def glob_match(pattern, path):
    """Match by segment. A `**` segment stands for zero or more directories;
    `*` and `?` stay inside one; a slash-less pattern names a file at the root,
    except a bare `**`, which is the whole tree; a pattern ending in `/` names a
    directory and so no file. One row per pattern segment over the path
    positions, so the cost is the product of the two counts and nothing is
    rescanned. (Git is borrowed for the `**` boundary rule and for nothing
    else.)"""
    if pattern.endswith("/"):
        return False
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
            for j in range(n):
                if ok[j] and seg_match(seg, ps[j]):
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
    """(deliver, claim file). Deliver is True when this hook is the first in
    (session, agent, project) to reach this rule: an O_EXCL create is the
    atomic claim two concurrent hooks race for, and one loses. Once per
    session holds only while a claim can be filed. A state directory that
    cannot be used — no absolute base, a base in the tree, a file where the
    directory should be — delivers on EVERY call rather than on none, with no
    file to withdraw: a repeat costs context, a silence costs the rule.

    ⚠ The two `try` blocks are one finding, not style. FileExistsError is an
    OSError, and `os.makedirs` raises it when the state DIRECTORY's path is
    already a regular file — so a single block that read that exception as
    "another hook holds the claim" suppressed every rule for the whole session,
    which is the silence Decision 6 forbids. Only the O_EXCL create may be read
    that way."""
    try:
        d = state_dir(root)
        sweep(d)
    except OSError:
        return True, None
    key = hashlib.sha1("\0".join((session, agent, root, rel)).encode()).hexdigest()
    flags = os.O_CREAT | os.O_EXCL | os.O_WRONLY | getattr(os, "O_NOFOLLOW", 0)
    path = os.path.join(d, key)
    try:
        os.close(os.open(path, flags, 0o600))
        return True, path
    except FileExistsError:
        return False, None
    except OSError:
        return True, None


if __name__ == "__main__":
    main()
