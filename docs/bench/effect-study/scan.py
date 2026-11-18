"""effect-study scanner: four measurements pre-registered in docs/adr/BACKLOG.md
("Pre-registered for four ideas from the 2026-11-18 research reading").

Stdlib only. Run: python3 -I scan.py [--root DIR] [--since ISO] [--until ISO] [--repo DIR] [--skip TEXT]...
It reads every Claude Code transcript (*.jsonl) under DIR (default ~/.claude/projects),
subagent transcripts included, and prints one JSON document of aggregates: no path,
file content, project name or session id. --since/--until count an entry by its own
timestamp. --repo names a git checkout whose tracked text files E2(a) reads.

E1  priced cost by token class; file sites acted on per request
E2  (a) line-tag collisions with a neighbouring line; (b) hunks, anchors and anchor misses
E3  host Edit/Write share of file-changing calls before and after a compaction
E4  "changed since mrw last saw it" refusals, by what the session itself did to the path

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
MRW_WRITE = re.compile(r"\bmrw\b[^|;&\n]*\bwrite\b")
MRW_READ = re.compile(r"\bmrw\b[^|;&\n]*\bread\b")
HUNK = re.compile(r"^@@ (\S+) (\S+) (replace|insert-after|insert-before|delete|create|unlink|rename)\b(.*)$", re.M)
ANCHOR = re.compile(r'anchor="((?:[^"\\]|\\.)*)"|anchor=([^\s"]\S*)')
ANCHOR_MISS = re.compile(r'anchor ".*" not in line \d+')
CHANGED = re.compile(r"(\S+) changed since mrw last saw it")
SUMMARY = re.compile(r"^(\d+) hunk\(s\), \d+ file\(s\)", re.M)
WINDOW = 30
REWRITERS = re.compile(
    r"gofmt -w|goimports|\bgit\s+(checkout|stash|pull|merge|rebase|reset|restore|apply|cherry-pick|revert|switch)\b"
    r"|sed -i|perl -[a-z]*i|prettier|\bruff\b|\bblack\b|go generate|go mod tidy|adr-verify|\bcodex\b"
    r"|\bmv\b|\bcp\b|\btee\b|python3?\b|npm run")


def when(o):
    """Return the entry's timestamp as an aware datetime, or None."""
    t = o.get("timestamp")
    if not isinstance(t, str):
        return None
    try:
        return datetime.datetime.fromisoformat(t.replace("Z", "+00:00"))
    except ValueError:
        return None


def text_of(content):
    """Return a tool result's text, whichever of the two shapes it has."""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(b.get("text", "") for b in content if isinstance(b, dict) and isinstance(b.get("text"), str))
    return ""


def events(path, since, until):
    """Yield the transcript's entries in order as tuples.

    ("usage", msg_id, usage) · ("use", msg_id, tool_id, name, input) · ("res", tool_id, text) · ("compact",)
    """
    last = None
    with open(path, "r", encoding="utf-8", errors="replace") as fh:
        for line in fh:
            try:
                o = json.loads(line)
            except ValueError:
                continue
            if not isinstance(o, dict):
                continue
            last = when(o) or last
            if last is not None and ((since and last < since) or (until and last >= until)):
                continue
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
        found = ANCHOR.search(m.group(4))
        if found:
            anchored += 1
            chars += len(found.group(1) if found.group(1) is not None else found.group(2))
        elif m.group(3) == "replace" and re.fullmatch(r"\d+|\$", m.group(2)):
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
    """Fold one transcript into t."""
    usage, uses, seen_use = {}, {}, set()
    order = []          # ("native"|"mrw", ...) file-changing calls and ("compact",) in order
    trail = []          # every tool call in order: (kind, payload), for E4's look-back
    request_sites = collections.defaultdict(lambda: [0, 0])   # msg id -> [mrw sites, native sites]
    for ev in events(path, since, until):
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
            if name in NATIVE_SITE:
                request_sites[mid][1] += 1
            if name in NATIVE_CHANGE:
                order.append(("native",))
                trail.append(("native", inp.get("file_path")))
            elif name in ("Agent", "Task"):
                trail.append(("agent", None))
            elif cmd:
                if MRW_WRITE.search(cmd):
                    order.append(("mrw",))
                    t.c["mrw_write_calls"] += 1
                trail.append(("bash", cmd))
            plan = cmd if cmd and MRW_WRITE.search(cmd) else inp.get("content") if name == "Write" else ""
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
            if not cmd or "mrw" not in cmd:
                continue
            t.c["anchor_misses"] += len(ANCHOR_MISS.findall(text))
            sites = 0
            if MRW_READ.search(cmd):
                sites += sum(1 for ln in text.split("\n") if ln.startswith("@@ "))
            if MRW_WRITE.search(cmd):
                sites += sum(int(m.group(1)) for m in SUMMARY.finditer(text))
            if sites:
                t.mrw_call_sites.append(sites)
                request_sites[mid][0] += sites
            trail.append(("mrwres", text))
            for m in CHANGED.finditer(text):
                t.c["changed_since_refusals"] += 1
                t.c["changed_since_" + cause(trail, m.group(1))] += 1
    for u in usage.values():
        cc = u.get("cache_creation") if isinstance(u.get("cache_creation"), dict) else {}
        h1 = cc.get("ephemeral_1h_input_tokens") or 0
        t.tokens["input"] += u.get("input_tokens") or 0
        t.tokens["output"] += u.get("output_tokens") or 0
        t.tokens["cache_read"] += u.get("cache_read_input_tokens") or 0
        t.tokens["cache_write_1h"] += h1
        t.tokens["cache_write_5m"] += max((u.get("cache_creation_input_tokens") or 0) - h1, 0)
    t.c["requests"] += len(usage)
    for mrw_sites, native_sites in request_sites.values():
        if mrw_sites:
            t.mrw_request_sites.append(mrw_sites)
        elif native_sites:
            t.native_request_sites.append(native_sites)
    around_compactions(order, t)


def cause(trail, rel):
    """Say what the session itself did to rel since mrw last showed or wrote it."""
    marks = ("==> " + rel + " ", "   " + rel + " ", "created " + rel + " ")
    start = 0
    for i in range(len(trail) - 2, -1, -1):          # the last entry is the refusal's own result
        if trail[i][0] == "mrwres" and any(k in trail[i][1] for k in marks):
            start = i + 1
            break
    found = "nothing_found"
    for kind, payload in trail[start:-1]:
        if kind == "native" and same_path(payload, rel):
            return "own_edit"
        if kind == "agent":
            found = "own_subagent"
        elif kind == "bash" and found == "nothing_found" and not MRW_WRITE.search(payload) \
                and not MRW_READ.search(payload) and REWRITERS.search(payload):
            found = "own_shell"
    return found


def around_compactions(order, t):
    """Pair the file-changing calls before and after each compaction that follows an mrw write."""
    cuts = [i for i, e in enumerate(order) if e[0] == "compact"]
    for k, i in enumerate(cuts):
        lo = cuts[k - 1] + 1 if k else 0
        hi = cuts[k + 1] if k + 1 < len(cuts) else len(order)
        if not any(e[0] == "mrw" for e in order[:i]):
            t.c["compactions_before_any_mrw_write"] += 1
            continue
        before, after = order[lo:i][-WINDOW:], order[i + 1:hi][:WINDOW]
        if len(before) < 5 or len(after) < 5:
            t.c["compactions_too_few_calls"] += 1
            continue
        t.c["compactions_paired"] += 1
        b = sum(e[0] == "native" for e in before)
        a = sum(e[0] == "native" for e in after)
        t.before.update(native=b, calls=len(before))
        t.after.update(native=a, calls=len(after))
        t.pair_diffs.append(a / len(after) - b / len(before))


def tag_collisions(repo):
    """Count neighbouring line pairs in repo's tracked text files that share a tag."""
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
        lines = data.split(b"\n")
        tags = [hashlib.sha1(ln).hexdigest() for ln in lines]
        c["files"] += 1
        for i in range(len(lines) - 1):
            c["pairs"] += 1
            if lines[i] == lines[i + 1]:
                c["identical_pairs"] += 1
            else:
                c["different_pairs_same_2hex"] += tags[i][:2] == tags[i + 1][:2]
                c["different_pairs_same_3hex"] += tags[i][:3] == tags[i + 1][:3]
    return dict(c)


def pct(a, b):
    """Return a as a percentage of b, rounded, or None when b is zero."""
    return round(100.0 * a / b, 3) if b else None


def spread(xs):
    """Summarise a list of counts."""
    if not xs:
        return {"n": 0}
    xs = sorted(xs)
    return {"n": len(xs), "mean": round(statistics.fmean(xs), 2), "median": statistics.median(xs),
            "p90": xs[int(0.9 * (len(xs) - 1))], "share_of_one": pct(sum(1 for x in xs if x == 1), len(xs))}


def report(t, files, unreadable, repo):
    """Build the JSON document."""
    priced = {k: t.tokens[k] * w for k, w in WEIGHTS.items()}
    total = sum(priced.values())
    c = t.c
    out = {
        "corpus": {"transcripts": files, "unreadable": unreadable, "requests": c["requests"],
                   "mrw_write_calls": c["mrw_write_calls"], "compactions": c["compactions"]},
        "E1": {"weights": WEIGHTS, "tokens": dict(t.tokens),
               "priced_share_pct": {k: pct(v, total) for k, v in priced.items()},
               "cache_share_pct": pct(priced["cache_read"] + priced["cache_write_5m"] + priced["cache_write_1h"], total),
               "sites_per_mrw_call": spread(t.mrw_call_sites),
               "sites_per_request_with_mrw": spread(t.mrw_request_sites),
               "sites_per_request_native_only": spread(t.native_request_sites)},
        "E2": {"hunks_typed": c["hunks_typed"], "anchored_hunks": c["anchored_hunks"],
               "bare_single_line_replaces": c["bare_single_line_replaces"],
               "bare_single_line_share_pct": pct(c["bare_single_line_replaces"], c["hunks_typed"]),
               "anchor_chars": c["anchor_chars"],
               "anchor_share_of_output_pct": pct(c["anchor_chars"] / 4.0, t.tokens["output"]),
               "anchor_misses": c["anchor_misses"],
               "anchor_miss_per_anchored_hunk_pct": pct(c["anchor_misses"], c["anchored_hunks"]),
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
               "nothing_found_per_mrw_write_pct": pct(c["changed_since_nothing_found"], c["mrw_write_calls"])},
    }
    tc = out["E2"]["tag_collisions"]
    if tc:
        different = tc.get("pairs", 0) - tc.get("identical_pairs", 0)
        tc["identical_pct"] = pct(tc.get("identical_pairs", 0), tc.get("pairs", 0))
        tc["different_same_2hex_pct"] = pct(tc.get("different_pairs_same_2hex", 0), different)
        tc["different_same_3hex_pct"] = pct(tc.get("different_pairs_same_3hex", 0), different)
    return out


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
        files += 1
        try:
            scan_session(path, since, until, t)
        except OSError:
            unreadable += 1
    return report(t, files, unreadable, repo)


def main():
    """Parse the flags, scan, print."""
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--root", default=os.path.expanduser("~/.claude/projects"))
    ap.add_argument("--since")
    ap.add_argument("--until")
    ap.add_argument("--repo")
    ap.add_argument("--skip", action="append", default=[], help="leave out transcripts whose path holds this text")
    a = ap.parse_args()
    json.dump(run(a.root, parse_time(a.since), parse_time(a.until), a.repo, a.skip), sys.stdout, indent=1, sort_keys=True)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
