"""effect-study scanner: four measurements pre-registered in docs/adr/BACKLOG.md
("Pre-registered for four ideas from the 2026-11-18 research reading").

Stdlib only. Run: python3 -I scan.py [--root DIR] [--since ISO] [--until ISO] [--repo DIR] [--skip TEXT]...
It reads every Claude Code transcript (*.jsonl) under DIR (default ~/.claude/projects),
subagent transcripts included, and prints one JSON document of aggregates: no path,
file content, project name or session id. --since/--until count an entry by its own
timestamp, and leave out an entry that has none. --repo names a git checkout whose
tracked text files E2(a) reads.

E1  priced cost by token class; file sites acted on per shell call and per request
E2  (a) lines whose tag equals a neighbour's; (b) hunks, anchors and anchor misses
E3  host Edit/Write share of file-changing calls before and after a compaction
E4  "changed since mrw last saw it" refusals, by what the session itself did since
E5  landed write calls an anchor requirement on unshown numbered lines would refuse (ADR-200)
E6  shell reverts (git checkout, restore, reset --hard, stash) of files a landed mrw write changed

Limits are in README.md. Every count is of transcript entries, not of incidents.
"""
import argparse
import collections
import datetime
import glob
import hashlib
import json
import os
import re
import statistics
import subprocess
import sys

# Price weights relative to one input token. An assumption, printed with the result.
WEIGHTS = {"input": 1.0, "output": 5.0, "cache_write_5m": 1.25, "cache_write_1h": 2.0, "cache_read": 0.1}
NATIVE_CHANGE = {"Edit", "Write", "MultiEdit", "NotebookEdit"}
NATIVE_SITE = NATIVE_CHANGE | {"Read"}
# mrw at a command position (not inside a quoted string or another command's arguments),
# its global flags, then the subcommand: the first word that is not a flag. It is matched
# on the command with heredoc bodies dropped and quoted strings folded to one word.
MRW_CALL = re.compile(
    r"(?:^|[;&|(\n]|\$\()\s*(?:[A-Za-z_][A-Za-z0-9_]*=\S*\s+)*(?:[^\s=;&|()]*/)?mrw"
    r"(?:\s+(?:--root|-C)(?:=|\s+)\S+)*\s+(read|write)\b")
HEREDOC = re.compile(r"<<-?\s*(['\"]?)([A-Za-z_]\w*)\1([^\n]*)\n.*?\n[ \t]*\2[ \t]*(?=\n|$)", re.S)
QUOTED = re.compile(r'"(?:[^"\\]|\\.)*"|\'[^\']*\'')
PATH = r"""(?:"[^"\n]*"|'[^'\n]*'|\S+)"""
PATTERN = r"/(?:[^/\\\n]|\\.)*/"
ADDR = r"(" + PATTERN + r"(?:,(?:" + PATTERN + r"|\+?\d+|\$))?|\S+)"
HUNK = re.compile(
    r"^@@ " + PATH + r" " + ADDR + r" (replace|insert-after|insert-before|delete|create|unlink|rename)\b(.*)$", re.M)
HUNK_PATH = re.compile(r"^@@ (" + PATH + r") ", re.M)
ANCHOR = re.compile(
    r"""anchor=(?:"((?:[^"\\]|\\.)*)"|'([^']*)'|(.*?))(?=\s+(?:body|lines|sha|occurrence|from|raw)=|\s*$)""")
ANCHOR_MISS = re.compile(r'^.*anchor ".*" not in line \d+.*$', re.M)
CHANGED = re.compile(r"(?:: |^)([^:\n]+?) changed since mrw last saw it", re.M)
SUMMARY = re.compile(r"^(\d+) hunk\(s\), \d+ file\(s\)", re.M)
WINDOW = 30
REWRITERS = re.compile(
    r"gofmt -w|goimports|\bgit\s+(checkout|stash|pull|merge|rebase|reset|restore|apply|cherry-pick|revert|switch)\b"
    r"|sed -i|perl -[a-z]*i|prettier|\bruff\b|\bblack\b|go generate|go mod tidy|adr-verify|\bcodex\b"
    r"|\bmv\b|\bcp\b|\btee\b|python3?\b|npm run")


def folded(cmd):
    """Return cmd with heredoc bodies dropped and each quoted string folded to the word Q.

    A double-quoted string that holds a command substitution, or names the mrw binary, keeps
    its text, so `out="$(mrw read f)"` and `"$t/mrw" read f` are still seen.
    """
    def fold(m):
        s = m.group(0)
        return s[1:-1] if s[0] == '"' and ("$(" in s or s.endswith('mrw"')) else "Q"
    return QUOTED.sub(fold, HEREDOC.sub(r"<<H\3", cmd))


def subcommands(cmd):
    """Return the set of mrw subcommands ("read", "write") a shell command runs."""
    return {m.group(1) for m in MRW_CALL.finditer(folded(cmd))}


def after_mrw(cmd):
    """Return what a shell command runs after its last mrw call, or all of it when it has none."""
    text, end = folded(cmd), 0
    for m in MRW_CALL.finditer(text):
        end = m.end()
    return text[end:]


def when(o):
    """Return the entry's timestamp as an aware datetime, or None; a zone-less one is read as UTC."""
    t = o.get("timestamp")
    if not isinstance(t, str):
        return None
    try:
        d = datetime.datetime.fromisoformat(t.replace("Z", "+00:00"))
    except ValueError:
        return None
    return d if d.tzinfo else d.replace(tzinfo=datetime.timezone.utc)


def text_of(content):
    """Return a tool result's text, whichever of the two shapes it has."""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(b.get("text", "") for b in content if isinstance(b, dict) and isinstance(b.get("text"), str))
    return ""


def events(path, since, until):
    """Yield the transcript's entries in order as tuples.

    ("seen",) for every entry in the window, then for what it holds:
    ("usage", msg_id, usage) · ("use", msg_id, tool_id, name, input) · ("res", tool_id, text) · ("compact",)
    """
    with open(path, "r", encoding="utf-8", errors="replace") as fh:
        for line in fh:
            try:
                o = json.loads(line)
            except ValueError:
                continue
            if not isinstance(o, dict):
                continue
            if since or until:
                ts = when(o)
                if ts is None:
                    yield ("undated",)
                    continue
                if (since and ts < since) or (until and ts >= until):
                    continue
            yield ("seen",)
            if o.get("type") == "system" and o.get("subtype") == "compact_boundary":
                yield ("compact",)
                continue
            msg = o.get("message")
            if not isinstance(msg, dict):
                continue
            if o.get("type") == "assistant" and isinstance(msg.get("usage"), dict) and msg.get("id"):
                yield ("usage", msg["id"], msg["usage"])
            content = msg.get("content")
            if not isinstance(content, list):
                continue
            for b in content:
                if not isinstance(b, dict):
                    continue
                if b.get("type") == "tool_use" and isinstance(b.get("input"), dict):
                    yield ("use", msg.get("id"), b.get("id"), b.get("name") or "", b["input"])
                elif b.get("type") == "tool_result":
                    yield ("res", b.get("tool_use_id"), text_of(b.get("content")))


def hunks_in(text):
    """Return (hunks, single-line replaces with no anchor, anchored hunks, anchor characters) typed in text."""
    n = bare = anchored = chars = 0
    for m in HUNK.finditer(text):
        n += 1
        found = ANCHOR.search(m.group(3))
        if found:
            anchored += 1
            chars += len(next(g for g in found.groups() if g is not None))
            continue
        addr = m.group(1)
        span = re.fullmatch(r"(\d+)-(\d+)", addr)
        if m.group(2) == "replace" and (re.fullmatch(r"\d+|\$", addr) or (span and span.group(1) == span.group(2))):
            bare += 1
    return n, bare, anchored, chars


def same_path(native, rel):
    """Report whether a host tool's file_path names the path a refusal printed."""
    return isinstance(native, str) and (native == rel or native.endswith("/" + rel))


HEREDOC_BODY = re.compile(r"<<(-?)\s*(['\"]?)([A-Za-z_][\w.-]*)\2([^\n]*)\n(.*?)\n[ \t]*\3[ \t]*(?=\n|$)", re.S)
MASK = re.compile(r"""(?<!body=)(?:"(?:[^"\\]|\\.)*"|'[^']*')""")
READ_HEAD = re.compile(r"^==> (.+?)  \d+L\b")
NUMBERED = re.compile(r"^\s*(\d+)\| ")
RECEIPT_ROW = re.compile(r"^(?:ok\s+|wrote |created |replaced )(\S+)", re.M)
LINE_OPS = ("replace", "delete", "insert-after", "insert-before")
EVERY = float("inf")


def shell_of(cmd):
    """Return cmd as E5 reads it for commands: every heredoc body dropped, each quoted string folded as in folded()."""
    def fold(m):
        s = m.group(0)
        return s[1:-1] if s[0] == '"' and ("$(" in s or s.endswith('mrw"')) else "Q"
    return QUOTED.sub(fold, HEREDOC_BODY.sub(r"<<H\4", cmd))


def quoted_words(cmd):
    """Return the text of each quoted string in a shell command outside its heredoc bodies."""
    return [m.group(0)[1:-1] for m in QUOTED.finditer(HEREDOC_BODY.sub(r"<<H\4", cmd))]


def plans_in(cmd):
    """Return the hunks typed in a shell command: those of each heredoc body, then those of its quoted strings.

    Each heredoc is a plan of its own. Quoted strings are read one a line; the last hunk among them
    has no known body unless it declares a count, since a later quoted string (`echo "done"`) is not
    told from a body line.
    """
    hunks = []
    for m in HEREDOC_BODY.finditer(cmd):
        body = m.group(5)
        hunks += typed_hunks(re.sub(r"(?m)^\t+", "", body) if m.group(1) else body)
    return hunks + typed_hunks("\n".join(w.replace("\\n", "\n") for w in quoted_words(cmd)), open_end=True)


def numbers(addr):
    """Return (first, last) for an address typed in line numbers, or None; `N-$` is read as its first line."""
    m = re.fullmatch(r"(\d+)(?:-(\d+|\$)|,\+(\d+))?", addr)
    if not m:
        return None
    a = int(m.group(1))
    if m.group(3):
        return a, a + int(m.group(3))
    return a, int(m.group(2)) if m.group(2) and m.group(2) != "$" else a


def typed_hunks(text, open_end=False):
    """Return (path, address, op, header options, body lines or None) for each hunk header in a plan text.

    The options are returned with every quoted value folded to Q, so a key is looked for only where
    it is an option. A header that declares `body=N` takes its next N lines, whatever they look like.
    With open_end, the last hunk's body is unknown unless it declared a count.
    """
    out, lines, i = [], text.split("\n"), 0
    while i < len(lines):
        m = HUNK.match(lines[i])
        if not m:
            i += 1
            continue
        path = HUNK_PATH.match(lines[i]).group(1).strip("\"'")
        opts = MASK.sub("Q", m.group(3))
        declared = re.search(r'(?:^|\s)body=(?:"(\d+)"|(\d+))(?=\s|$)', opts)
        j = i + 1
        if declared:
            body = int(declared.group(1) or declared.group(2))
            j += body
        else:
            while j < len(lines) and not HUNK.match(lines[j]):
                j += 1
            unknown = re.search(r"(?:^|\s)(?:body|from)=", opts) or (open_end and j >= len(lines))
            body = None if unknown else j - i - 1
        out.append((path[2:] if path.startswith("./") else path, m.group(1), m.group(2), opts, body))
        i = j
    return out


def read_args(cmd, piped_only):
    """Return each mrw read's own arguments in a shell command, up to a pipe or a redirect.

    With piped_only, only the reads whose output does not reach the caller: piped, redirected or captured.
    """
    text, out = shell_of(cmd), []
    for m in MRW_CALL.finditer(text):
        if m.group(1) != "read":
            continue
        tail = re.split(r";|&&|\|\||\n", text[m.end():], maxsplit=1)[0]
        own = re.split(r"\|(?!\|)|>", re.sub(r"2>\s*\S+", "", tail), maxsplit=1)
        if len(own) > 1 or "$(" in text[max(m.start() - 1, 0):m.start() + 3] or not piped_only:
            out.append(own[0])
    return out


class Guard:
    """E5: which landed write calls an anchor requirement on unshown numbered lines would refuse.

    Width A forgets every shown line of a file at each write to it. Width B keeps a line shown
    before the write that the write left where it was. A file is tracked from the session's own
    first mrw write to it; the host's Edit or Write of it ends the tracking, since mrw then refuses
    the file until it is read again.
    """

    def __init__(self, t):
        self.t = t
        self.written, self.created, self.uncertain = set(), set(), set()
        self.shown_a = collections.defaultdict(set)
        self.shown_b = collections.defaultdict(set)
        self.on_write = None    # called with each path a landed write changed (E6)

    def native(self, file_path):
        """The host's own tool was called to change a file: nothing mrw showed of it stands."""
        for rel in [r for r in set(self.written) | set(self.shown_b) if same_path(file_path, r)]:
            self.written.discard(rel)
            self.created.discard(rel)
            self.uncertain.discard(rel)
            self.shown_a.pop(rel, None)
            self.shown_b.pop(rel, None)

    def doubt(self, cmd, args):
        """Make uncertain each written file one of these reads names; a quoted spec is looked for among the quoted strings."""
        quoted = " ".join(quoted_words(cmd))
        for own in args:
            names = own + " " + quoted if "Q" in own else own
            self.uncertain |= {rel for rel in self.written if rel in names}

    def served(self, text, cmd):
        """An mrw read answer: remember the line numbers it printed under each file."""
        self.doubt(cmd, read_args(cmd, True))
        cur = None
        for ln in text.split("\n"):
            head = READ_HEAD.match(ln)
            if head:
                cur = head.group(1)
                continue
            num = NUMBERED.match(ln)
            if num and cur is not None:
                self.shown_b[cur].add(int(num.group(1)))
                self.shown_a[cur].add(int(num.group(1)))

    def judge(self, hunks):
        """Count one landed write call's typed hunks against the state before it."""
        c = self.t.c
        c["e5_landed_plan_calls"] += 1
        hit = collections.Counter()
        for path, addr, op, rest, _ in hunks:
            span = numbers(addr)
            if (op not in LINE_OPS or span is None or span[0] < 1 or re.search(r"(?:^|\s)anchor=", rest)
                    or path not in self.written):
                continue
            hit["ceiling"] += 1
            lines = range(span[0], span[1] + 1)
            doubt = "_uncertain" if path in self.uncertain else ""
            if any(n not in self.shown_a[path] for n in lines):
                hit["a" + doubt] += 1
                if not doubt:
                    c["e5_refused_hunks_" + op] += 1
                    c["e5_refused_hunks_on_created"] += path in self.created
            if any(n not in self.shown_b[path] for n in lines):
                hit["b" + doubt] += 1
        c["e5_ceiling_calls"] += bool(hit["ceiling"])
        c["e5_refused_calls_a"] += bool(hit["a"])
        c["e5_uncertain_calls_a"] += bool(hit["a_uncertain"]) and not hit["a"]
        c["e5_refused_calls_b"] += bool(hit["b"])
        c["e5_uncertain_calls_b"] += bool(hit["b_uncertain"]) and not hit["b"]

    def landed(self, hunks, text):
        """A write that applied: forget what it moved, and track every file it changed."""
        by_path = collections.defaultdict(list)
        for h in hunks:
            by_path[h[0]].append(h)
        for path, hs in by_path.items():
            if any(h[2] in ("unlink", "rename") for h in hs):
                self.native("/" + path)
                self.shown_b.pop(path, None)
                continue
            moved, gone = EVERY, set()
            for _, addr, op, _, body in hs:
                span = numbers(addr)
                if op == "create":
                    self.created.add(path)
                if op not in LINE_OPS or span is None or body is None:
                    moved = 0
                    continue
                if "$" in addr:                          # `N-$`: everything from N on is replaced or gone
                    moved = min(moved, span[0])
                    continue
                a, b = span
                if op == "insert-after":
                    moved = min(moved, a + 1)
                elif op == "insert-before" or op == "delete" or body != b - a + 1:
                    moved = min(moved, a)
                if op in ("replace", "delete"):
                    gone |= set(range(a, b + 1))
            self.shown_b[path] = {n for n in self.shown_b[path] if n < moved and n not in gone}
            self.mark(path)
        for m in RECEIPT_ROW.finditer(text):
            rel = m.group(1)
            if rel not in by_path:                       # a plan file, --create or a plan maker wrote it
                self.shown_b.pop(rel, None)
                if text[m.start():m.start() + 8] == "created ":
                    self.created.add(rel)
                self.mark(rel)

    def mark(self, rel):
        """Record that mrw wrote rel: nothing of it has been shown since."""
        self.written.add(rel)
        self.uncertain.discard(rel)
        self.shown_a[rel] = set()
        if self.on_write:
            self.on_write(rel)

    def answer(self, cmd, text):
        """Fold one shell call that ran mrw, and its answer, into the counts and the state.

        Reads that all come before the first write are applied before the plan is judged, and reads
        that all come after the last write after it. Reads on both sides cannot be told apart in one
        answer: none is applied, and the written files the command names become uncertain.
        """
        calls = [m.group(1) for m in MRW_CALL.finditer(shell_of(cmd))]
        writes = [i for i, s in enumerate(calls) if s == "write"]
        reads = [i for i, s in enumerate(calls) if s == "read"]
        hunks = plans_in(cmd) if writes else []
        before = bool(reads) and bool(writes) and reads[-1] < writes[0]
        mixed = bool(reads) and bool(writes) and not before and reads[0] < writes[-1]
        if before:
            self.served(text, cmd)
        elif mixed:
            self.doubt(cmd, read_args(cmd, False))
        self.t.c["e5_chained_write_commands"] += len(writes) > 1
        if writes and "— applied" in text:
            if hunks:
                self.judge(hunks)
            else:
                self.t.c["e5_landed_calls_with_no_typed_plan"] += 1
            self.landed(hunks, text)
        if mixed:
            self.doubt(cmd, read_args(cmd, False))        # a write clears the doubt; these reads put it back
        elif reads and not before:
            self.served(text, cmd)


GIT_REVERT = re.compile(
    r"(?:^|[;&|(\n]|\$\()\s*git(?:\s+-C\s+\S+)*\s+(checkout|restore|reset|stash)(?=\s|$)([^;&|\n]*)")
NEAR = 5


class Reverts:
    """E6: shell reverts of files a landed mrw write of the session changed.

    A path revert is `git checkout` or `git restore` naming a written file; it is near when it
    comes within NEAR tool calls of the write that last changed that file. A tree revert is
    `git checkout .`, `git restore .`, `git reset --hard` or `git stash` after the session's
    first landed write, counted apart since nothing ties it to a file mrw wrote.
    """

    def __init__(self, t):
        self.t = t
        self.calls = 0
        self.ordinal = {}       # tool call id -> its count among the session's tool calls
        self.landing = 0        # the count of the call whose answer is being read
        self.written = {}       # path -> the count of the tool call whose write last changed it
        self.any = False

    def wrote(self, rel):
        """A landed mrw write changed rel."""
        self.written[rel] = self.landing
        self.any = True

    def call(self, tid):
        """One more tool call."""
        self.calls += 1
        self.ordinal[tid] = self.calls

    def answered(self, tid):
        """The answer about to be read belongs to this call: a write it reports landed then."""
        self.landing = self.ordinal.get(tid, self.calls)

    def shell(self, cmd):
        """Count one shell command's reverts, at most one path revert and one tree revert."""
        hit, near, tree = False, False, False
        for m in GIT_REVERT.finditer(shell_of(cmd)):
            verb, args = m.group(1), m.group(2).split()
            if verb == "stash":
                tree |= not args or args[0] not in ("pop", "apply", "list", "show", "drop")
                continue
            if verb == "reset":
                tree |= "--hard" in args
                continue
            if "-b" in args or "-B" in args or (verb == "restore" and ("--staged" in args or "-S" in args)):
                continue
            names = args[args.index("--") + 1:] if "--" in args else [a for a in args if not a.startswith("-")]
            tree |= "." in names
            for name in names:                       # `HEAD` or a branch among them names no written file
                for rel in [r for r in self.written if name == r or name.endswith("/" + r)]:
                    hit = True
                    near |= self.calls - self.written.pop(rel) <= NEAR
        c = self.t.c
        c["e6_path_reverts"] += hit
        c["e6_near_path_reverts"] += near
        c["e6_tree_reverts"] += tree and self.any


class Totals:
    """Counts across every transcript."""

    def __init__(self):
        self.c = collections.Counter()
        self.tokens = collections.Counter()
        self.mrw_call_sites = []
        self.mrw_request_sites = []
        self.native_request_sites = []
        self.before = collections.Counter()
        self.after = collections.Counter()
        self.pair_diffs = []


def scan_session(path, since, until, t):
    """Fold one transcript into t; report whether any entry of it was in the window."""
    usage, uses, seen_use, misses = {}, {}, set(), set()
    order = []          # ("native"|"mrw"|"compact",) in order: file-changing calls and compactions
    trail = []          # every tool call and mrw answer in order, for E4's look-back
    request_sites = collections.defaultdict(lambda: [0, 0, False])   # msg id -> [mrw sites, host sites, mrw ran]
    any_entry = False
    guard = Guard(t)
    reverts = Reverts(t)
    guard.on_write = reverts.wrote
    for ev in events(path, since, until):
        if ev[0] == "undated":
            # Not reported: a live session keeps appending undated entries, so the count is not the window's.
            continue
        any_entry = True
        if ev[0] == "seen":
            continue
        if ev[0] == "usage":
            usage[ev[1]] = ev[2]
        elif ev[0] == "compact":
            order.append(("compact",))
            t.c["compactions"] += 1
        elif ev[0] == "use":
            _, mid, tid, name, inp = ev
            if tid in seen_use:
                continue
            seen_use.add(tid)
            uses[tid] = (mid, name, inp)
            reverts.call(tid)
            cmd = inp.get("command") if name == "Bash" and isinstance(inp.get("command"), str) else ""
            subs = subcommands(cmd) if cmd else set()
            if name in NATIVE_SITE:
                request_sites[mid][1] += 1
            if name in NATIVE_CHANGE:
                order.append(("native",))
                trail.append(("native", inp.get("file_path")))
                guard.native(inp.get("file_path"))
            elif name in ("Agent", "Task"):
                trail.append(("agent", None))
            elif cmd:
                if "write" in subs:
                    order.append(("mrw",))
                    t.c["mrw_write_calls"] += 1
                trail.append(("bash", cmd, after_mrw(cmd)))
                reverts.shell(cmd)
            plan = cmd if "write" in subs else inp.get("content") if name == "Write" else ""
            if isinstance(plan, str) and plan:
                n, bare, anchored, chars = hunks_in(plan)
                t.c["hunks_typed"] += n
                t.c["bare_single_line_replaces"] += bare
                t.c["anchored_hunks"] += anchored
                t.c["anchor_chars"] += chars
        elif ev[0] == "res":
            _, tid, text = ev
            mid, name, inp = uses.get(tid, (None, "", {}))
            cmd = inp.get("command") if name == "Bash" and isinstance(inp.get("command"), str) else ""
            subs = subcommands(cmd) if cmd else set()
            if not subs:
                continue
            text = unfolded(text)
            reverts.answered(tid)
            guard.answer(cmd, text)
            for miss in ANCHOR_MISS.findall(text):
                t.c["anchor_miss_lines"] += 1
                misses.add(miss)
            sites = 0
            if "read" in subs:
                sites += sum(1 for ln in text.split("\n") if ln.startswith("@@ "))
            if "write" in subs:
                sites += sum(int(m.group(1)) for m in SUMMARY.finditer(text))
            t.mrw_call_sites.append(sites)           # a call that served or changed nothing is a call of zero
            request_sites[mid][0] += sites
            request_sites[mid][2] = True
            trail.append(("mrwres", text, cmd))
            for m in CHANGED.finditer(text):
                t.c["changed_since_refusals"] += 1
                t.c["changed_since_" + cause(trail, m.group(1).strip())] += 1
    t.c["anchor_misses_distinct"] += len(misses)
    for u in usage.values():
        cc = u.get("cache_creation") if isinstance(u.get("cache_creation"), dict) else {}
        h1 = cc.get("ephemeral_1h_input_tokens") or 0
        t.tokens["input"] += u.get("input_tokens") or 0
        t.tokens["output"] += u.get("output_tokens") or 0
        t.tokens["cache_read"] += u.get("cache_read_input_tokens") or 0
        t.tokens["cache_write_1h"] += h1
        t.tokens["cache_write_5m"] += max((u.get("cache_creation_input_tokens") or 0) - h1, 0)
    t.c["requests"] += len(usage)
    for mrw_sites, native_sites, ran in request_sites.values():
        if ran:
            t.mrw_request_sites.append(mrw_sites + native_sites)
        elif native_sites:
            t.native_request_sites.append(native_sites)
    around_compactions(order, t)
    return any_entry


def as_plain(doc):
    """Return the plain lines a `--json` document stands for, or None when it is not an mrw answer."""
    if not isinstance(doc, dict):
        return None
    if isinstance(doc.get("content"), str):
        return doc["content"]
    if not isinstance(doc.get("hunks"), list):
        return None
    rows = []
    for f in doc.get("files") or []:
        if isinstance(f, dict) and f.get("written"):
            rows.append("ok   %s  " % f.get("path"))
    for h in doc["hunks"]:
        if isinstance(h, dict) and h.get("reason"):
            rows.append("FAIL %s %s %s: %s" % (h.get("path"), h.get("addr"), h.get("op"), h["reason"]))
    rows.append("%d hunk(s), %d file(s), %d failed — %s" % (
        len(doc["hunks"]), len(doc.get("files") or []), doc.get("failed") or 0,
        "applied" if doc.get("applied") else "nothing written"))
    return "\n".join(rows) + "\n"


def unfolded(text):
    """Return an mrw answer with each `--json` document in it rewritten as the lines the plain answer prints.

    A read's document becomes its content. A write's becomes a row per written file, a FAIL row per
    refused hunk and the summary line, so the patterns for plain answers read both alike. What
    stands before, between and after the documents is kept.
    """
    out, at = [], 0
    for m in re.finditer(r"^\{", text, re.M):
        if m.start() < at:
            continue                                 # a brace inside a document already taken
        try:
            doc, used = json.JSONDecoder().raw_decode(text[m.start():])
        except ValueError:
            continue
        plain = as_plain(doc)
        if plain is not None:
            out.append(text[at:m.start()] + plain)
            at = m.start() + used
    return "".join(out) + text[at:]


def shows(text, rel, cmd=""):
    """Report whether an mrw answer served lines of rel or wrote it.

    A read: rel's header with a line count, then a range line, or the licence line a quiet read
    prints. A write: an answer whose summary says applied, which a dry run's does not, with rel's
    receipt row, or, for a quiet write that prints no row, with rel in a hunk header of the command.
    """
    q = re.escape(rel)
    if re.search(r"^==> " + q + r"\s+\d+L\b[^\n]*\n@@ ", text, re.M):
        return True
    if re.search(r"^==> " + q + r"\s+\d+L\b", text, re.M) and "-- licensed " in text:
        return True
    if "— applied" not in text:
        return False
    if re.search(r"^(?:ok\s+|created |replaced )" + q + r"\s", text, re.M):
        return True
    return any(m.group(1).strip("\"'") == rel for m in HUNK_PATH.finditer(cmd))


def cause(trail, rel):
    """Say what the session itself did since mrw last served or wrote rel.

    own_edit is the host's Edit/Write of that path. own_shell and own_subagent are candidates
    only: a shell command that can rewrite files, or an Agent call, with nothing tying it to rel.
    """
    start, first = 0, None
    for i in range(len(trail) - 2, -1, -1):          # the last entry is the refusal's own answer
        if trail[i][0] == "mrwres" and shows(trail[i][1], rel, trail[i][2]):
            start = i + 1
            # The shell call that produced the answer may have gone on to rewrite files
            # (`mrw read f && gofmt -w .`): what it ran after its last mrw call counts.
            if i and trail[i - 1][0] == "bash":
                first = trail[i - 1][2]
            break
    found = "own_shell" if first and REWRITERS.search(first) else "nothing_found"
    for entry in trail[start:-1]:
        kind, payload = entry[0], entry[1]
        if kind == "native" and same_path(payload, rel):
            return "own_edit"
        if kind == "agent" and found == "nothing_found":
            found = "own_subagent"
        elif kind == "bash" and found != "own_shell" and REWRITERS.search(folded(payload)):
            found = "own_shell"
    return found


def around_compactions(order, t):
    """Pair the 30 file-changing calls before and after each compaction that follows an mrw write."""
    calls = [e[0] for e in order if e[0] != "compact"]
    at, seen = [], 0                                 # at[k]: how many calls precede compaction k
    for e in order:
        if e[0] == "compact":
            at.append(seen)
        else:
            seen += 1
    for i in at:
        if "mrw" not in calls[:i]:
            t.c["compactions_before_any_mrw_write"] += 1
            continue
        before, after = calls[max(i - WINDOW, 0):i], calls[i:i + WINDOW]
        if len(before) < 5 or len(after) < 5:
            t.c["compactions_too_few_calls"] += 1
            continue
        t.c["compactions_paired"] += 1
        b, a = before.count("native"), after.count("native")
        t.before.update(native=b, calls=len(before))
        t.after.update(native=a, calls=len(after))
        t.pair_diffs.append(a / len(after) - b / len(before))


def tag_lines(data, c):
    """Count in c the lines of one file whose neighbour is identical, or different with the same tag."""
    lines = data.split(b"\n")
    if lines and lines[-1] == b"":
        lines.pop()                                  # a final newline ends the last line, it starts none
    tags = [hashlib.sha1(ln).hexdigest() for ln in lines]
    for i, ln in enumerate(lines):
        near = [j for j in (i - 1, i + 1) if 0 <= j < len(lines)]
        c["lines"] += 1
        c["lines_with_identical_neighbour"] += any(lines[j] == ln for j in near)
        c["lines_same_2hex_as_different_neighbour"] += any(lines[j] != ln and tags[j][:2] == tags[i][:2] for j in near)
        c["lines_same_3hex_as_different_neighbour"] += any(lines[j] != ln and tags[j][:3] == tags[i][:3] for j in near)


def tag_collisions(repo):
    """Count, over repo's tracked text files, the lines a 2- or 3-hex tag cannot tell from a neighbour."""
    c = collections.Counter()
    try:
        names = subprocess.run(["git", "-C", repo, "ls-files", "-z"], capture_output=True, check=True).stdout
    except (OSError, subprocess.CalledProcessError):
        return None
    for name in names.split(b"\0"):
        if not name:
            continue
        try:
            with open(os.path.join(os.fsencode(repo), name), "rb") as fh:
                data = fh.read()
        except OSError:
            continue
        if b"\0" in data[:8192]:
            continue
        c["files"] += 1
        tag_lines(data, c)
    out = dict(c)
    for k in ("lines_with_identical_neighbour", "lines_same_2hex_as_different_neighbour",
              "lines_same_3hex_as_different_neighbour"):
        out[k + "_pct"] = pct(c[k], c["lines"])
    return out


def pct(a, b):
    """Return a as a percentage of b, rounded, or None when b is zero."""
    return round(100.0 * a / b, 3) if b else None


def spread(xs):
    """Summarise a list of counts."""
    if not xs:
        return {"n": 0}
    xs = sorted(xs)
    return {"n": len(xs), "mean": round(statistics.fmean(xs), 2), "median": float(statistics.median(xs)),
            "p90": xs[int(0.9 * (len(xs) - 1))], "share_of_one": pct(sum(1 for x in xs if x == 1), len(xs))}


def guard_verdict(w, r_a, u_a, r_b, u_b):
    """The verdict pre-registered in docs/adr/BACKLOG.md (Pre-registered for ADR-200), first rule that applies."""
    if w < 200:
        return "INSUFFICIENT"
    if 20 * r_a <= w and 10 * (r_a + u_a) <= w:
        return "BUILD A"
    if 20 * r_b <= w and 10 * (r_b + u_b) <= w:
        return "BUILD B"
    return "STOP"


def guard_report(c):
    """E5's part of the JSON document."""
    w = c["e5_landed_plan_calls"]
    r_a, u_a, r_b, u_b = (c["e5_refused_calls_a"], c["e5_uncertain_calls_a"],
                          c["e5_refused_calls_b"], c["e5_uncertain_calls_b"])
    return {"landed_plan_calls": w,
            "refused_calls_a": r_a, "uncertain_calls_a": u_a, "refused_a_pct": pct(r_a, w),
            "refused_calls_b": r_b, "uncertain_calls_b": u_b, "refused_b_pct": pct(r_b, w),
            "ceiling_calls": c["e5_ceiling_calls"], "ceiling_pct": pct(c["e5_ceiling_calls"], w),
            "refused_hunks_by_op": {op: c["e5_refused_hunks_" + op] for op in LINE_OPS},
            "refused_hunks_on_created": c["e5_refused_hunks_on_created"],
            "chained_write_commands": c["e5_chained_write_commands"],
            "landed_calls_with_no_typed_plan": c["e5_landed_calls_with_no_typed_plan"],
            "verdict": guard_verdict(w, r_a, u_a, r_b, u_b)}


def revert_verdict(landed, near):
    """The verdict pre-registered in docs/adr/BACKLOG.md (Pre-registered for an undo, E6), first rule that applies."""
    if landed < 1000:
        return "INSUFFICIENT"
    if near >= 25 and 200 * near >= landed:
        return "RECORD"
    return "STOP"


def revert_report(c):
    """E6's part of the JSON document."""
    landed = c["e5_landed_plan_calls"] + c["e5_landed_calls_with_no_typed_plan"]
    return {"landed_write_calls": landed, "path_reverts": c["e6_path_reverts"],
            "near_path_reverts": c["e6_near_path_reverts"], "near_pct": pct(c["e6_near_path_reverts"], landed),
            "tree_reverts": c["e6_tree_reverts"], "near_window_calls": NEAR,
            "verdict": revert_verdict(landed, c["e6_near_path_reverts"])}


def report(t, files, unreadable, repo):
    """Build the JSON document."""
    priced = {k: t.tokens[k] * w for k, w in WEIGHTS.items()}
    total = sum(priced.values())
    c = t.c
    return {
        "corpus": {"transcripts": files, "unreadable": unreadable, "requests": c["requests"],
                   "mrw_write_calls": c["mrw_write_calls"], "compactions": c["compactions"]},
        "E1": {"weights": WEIGHTS, "tokens": dict(t.tokens),
               "priced_share_pct": {k: pct(v, total) for k, v in priced.items()},
               "cache_share_pct": pct(priced["cache_read"] + priced["cache_write_5m"] + priced["cache_write_1h"], total),
               "sites_per_shell_call_running_mrw": spread(t.mrw_call_sites),
               "sites_per_request_with_mrw": spread(t.mrw_request_sites),
               "sites_per_request_native_only": spread(t.native_request_sites)},
        "E2": {"hunks_typed": c["hunks_typed"], "anchored_hunks": c["anchored_hunks"],
               "bare_single_line_replaces": c["bare_single_line_replaces"],
               "bare_single_line_share_pct": pct(c["bare_single_line_replaces"], c["hunks_typed"]),
               "anchor_chars": c["anchor_chars"],
               "anchor_share_of_output_pct": pct(c["anchor_chars"] / 4.0, t.tokens["output"]),
               "anchor_miss_lines": c["anchor_miss_lines"],
               "anchor_misses_distinct": c["anchor_misses_distinct"],
               "anchor_miss_lines_per_anchored_hunk_pct": pct(c["anchor_miss_lines"], c["anchored_hunks"]),
               "anchor_misses_distinct_per_anchored_hunk_pct": pct(c["anchor_misses_distinct"], c["anchored_hunks"]),
               "tag_collisions": tag_collisions(repo) if repo else None},
        "E3": {"window": WINDOW, "compactions_paired": c["compactions_paired"],
               "compactions_before_any_mrw_write": c["compactions_before_any_mrw_write"],
               "compactions_too_few_calls": c["compactions_too_few_calls"],
               "before": dict(t.before), "after": dict(t.after),
               "native_share_before_pct": pct(t.before["native"], t.before["calls"]),
               "native_share_after_pct": pct(t.after["native"], t.after["calls"]),
               "mean_paired_difference_pp": round(100 * statistics.fmean(t.pair_diffs), 3) if t.pair_diffs else None},
        "E4": {"changed_since_refusals": c["changed_since_refusals"],
               "own_edit": c["changed_since_own_edit"], "own_shell": c["changed_since_own_shell"],
               "own_subagent": c["changed_since_own_subagent"], "nothing_found": c["changed_since_nothing_found"],
               "nothing_found_per_mrw_write_pct": pct(c["changed_since_nothing_found"], c["mrw_write_calls"]),
               "not_own_edit_per_mrw_write_pct": pct(c["changed_since_refusals"] - c["changed_since_own_edit"],
                                                     c["mrw_write_calls"])},
        "E5": guard_report(c),
        "E6": revert_report(c),
    }


def parse_time(s):
    """Parse an ISO time given on the command line; a bare date is midnight UTC."""
    if not s:
        return None
    d = datetime.datetime.fromisoformat(s.replace("Z", "+00:00"))
    return d if d.tzinfo else d.replace(tzinfo=datetime.timezone.utc)


def run(root, since=None, until=None, repo=None, skip=()):
    """Scan every transcript under root, but those whose path holds a skip string, and return the report."""
    t, files, unreadable = Totals(), 0, 0
    for path in sorted(glob.glob(os.path.join(root, "**", "*.jsonl"), recursive=True)):
        if any(s in os.path.relpath(path, root) for s in skip):
            continue
        try:
            files += scan_session(path, since, until, t)
        except OSError:
            unreadable += 1
    return report(t, files, unreadable, repo)


class Quiet(argparse.ArgumentParser):
    """A parser whose usage error names no argument: a value could be a path or a session."""

    def error(self, message):
        """Exit with a fixed line."""
        sys.exit("scan.py: bad arguments; see --help")


def main():
    """Parse the flags, scan, print."""
    ap = Quiet(prog="scan.py", description=__doc__.split("\n")[0])
    ap.add_argument("--root", default=os.path.expanduser("~/.claude/projects"))
    ap.add_argument("--since", help="an ISO time; a bare date is midnight UTC")
    ap.add_argument("--until", help="an ISO time, exclusive")
    ap.add_argument("--repo")
    ap.add_argument("--skip", action="append", default=[], help="leave out transcripts whose path holds this text")
    a = ap.parse_args()
    try:                                             # no message here may echo a value or print a traceback:
        since, until = parse_time(a.since), parse_time(a.until)      # either could name a path or a session
    except ValueError:
        sys.exit("scan.py: --since and --until take an ISO time")
    try:
        out = run(a.root, since, until, a.repo, a.skip)
        json.dump(out, sys.stdout, indent=1, sort_keys=True)
        sys.stdout.write("\n")
        sys.stdout.flush()
    except Exception as err:
        try:
            sys.stdout.close()
        except OSError:
            pass
        sys.exit("scan.py: stopped on %s" % type(err).__name__)


if __name__ == "__main__":
    main()
