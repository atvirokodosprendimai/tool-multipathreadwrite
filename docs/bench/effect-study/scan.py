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
            cmd = inp.get("command") if name == "Bash" and isinstance(inp.get("command"), str) else ""
            subs = subcommands(cmd) if cmd else set()
            if name in NATIVE_SITE:
                request_sites[mid][1] += 1
            if name in NATIVE_CHANGE:
                order.append(("native",))
                trail.append(("native", inp.get("file_path")))
            elif name in ("Agent", "Task"):
                trail.append(("agent", None))
            elif cmd:
                if "write" in subs:
                    order.append(("mrw",))
                    t.c["mrw_write_calls"] += 1
                trail.append(("bash", cmd, after_mrw(cmd)))
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
