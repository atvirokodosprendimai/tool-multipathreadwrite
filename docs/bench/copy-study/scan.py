"""copy-study scanner: how many output characters would ADR-167's `from=PATH:ADDR`
copy have saved in recorded Claude Code sessions?

Stdlib only. Run: python3 -I scan.py [--root DIR] [--limit N] [--out FILE] [--examples]
Publishes aggregates only: no paths, file contents, project names or session ids.
README.md beside this file gives the method and its limits.
"""
import argparse
import collections
import json
import os
import random
import re
import sys
import time

KS = (1, 3, 5, 10)
MIN_NONWS = 40          # a run must carry this many non-whitespace characters
START_NONWS = 8         # a run is seeded only at a line this substantial
MAX_CANDIDATES = 200    # pool positions tried per seed line (most recent first)
HEADER_COST = 45        # chars of one `from=path:N-M` hunk header (assumption)
OPS = r"replace|insert-after|insert-before|delete|create|unlink|rename"
PLAN_HEADER = re.compile(r"^@@ (\S+) (\S+) (" + OPS + r")\b(.*)$")
HEREDOC = re.compile(r"<<-?\s*(['\"]?)([A-Za-z_]\w*)\1[^\n]*\n(.*?)\n[ \t]*\2\)?[ \t]*$", re.S | re.M)
CAT_HEREDOC = re.compile(r"cat\s*>\s*(\S+)\s*<<-?\s*(['\"]?)([A-Za-z_]\w*)\2[^\n]*\n(.*?)\n[ \t]*\3\)?[ \t]*$",
                         re.S | re.M)
MRW_HEAD = re.compile(r"^==> (\S+)")
MRW_LINE = re.compile(r"^\s*(\d+)\| ?(.*)$")
CATN_LINE = re.compile(r"^\s*(\d+)[\t→](.*)$")
RANGE = re.compile(r"^(\d+)(?:-(\d+)|,\+(\d+))?$")
BODY_N = re.compile(r"\bbody=(\d+)\b")
USAGE = re.compile(r'"id":\s*"(msg_[^"]+)".*?"output_tokens":\s*(\d+)')


def nonws(s):
    return sum(1 for c in s if not c.isspace())


def chars(lines):
    return sum(len(s) + 1 for s in lines)


def tokens(c):
    return round(c / 4)


class Session:
    """Per-transcript state: the SEEN pool and path identities."""

    def __init__(self):
        self.paths = []
        self.cache = {}
        self.pos = {}                                   # (fid, n) -> exact text
        self.posn = {}                                  # (fid, n) -> stripped text
        self.idx = collections.defaultdict(list)        # exact text -> [(fid, n)]
        self.idxn = collections.defaultdict(list)       # stripped text -> [(fid, n)]
        self.by_file = collections.defaultdict(dict)    # fid -> {n: text}
        self.served = {}                                # (fid, n) -> text served since the file's last write
        self.served_by_file = collections.defaultdict(set)
        self.served_chars = 0
        self.redundant_chars = 0

    def fid(self, path):
        """Same file iff equal, or one path is a '/'-suffix of the other."""
        if path in self.cache:
            return self.cache[path]
        p = path.rstrip("/")
        for i, q in enumerate(self.paths):
            if p == q or q.endswith("/" + p) or p.endswith("/" + q):
                self.cache[path] = i
                return i
        self.paths.append(p)
        self.cache[path] = len(self.paths) - 1
        return self.cache[path]

    def see(self, path, n, text):
        key = (self.fid(path), n)
        size = len(text) + 1
        self.served_chars += size
        if self.served.get(key) == text:                # same line, same number, no write to the file since
            self.redundant_chars += size
        self.served[key] = text
        self.served_by_file[key[0]].add(n)
        old = self.pos.get(key)
        if old == text:
            return
        if old is not None:                             # the line changed: its old text seeds no copy
            self.idx[old].remove(key)
            self.idxn[old.strip()].remove(key)
        self.pos[key] = text
        self.posn[key] = text.strip()
        self.idx[text].append(key)
        self.idxn[text.strip()].append(key)
        self.by_file[key[0]][n] = text

    def invalidate(self, path):
        """A write to a file: nothing served from it before counts as redundant after."""
        f = self.fid(path)
        for n in self.served_by_file.pop(f, ()):
            self.served.pop((f, n), None)


# ------------------------------------------------------------------ SEEN

def parse_mrw_output(sess, text):
    path = None
    for line in text.split("\n"):
        m = MRW_HEAD.match(line)
        if m:
            path = m.group(1)
            continue
        if path is None or line.startswith("-- ck ") or line.startswith("@@ "):
            continue
        m = MRW_LINE.match(line)
        if m:
            sess.see(path, int(m.group(1)), m.group(2))


def parse_read_result(sess, text, path):
    lines = []
    for line in text.split("\n"):
        m = CATN_LINE.match(line)
        if m:
            lines.append((int(m.group(1)), m.group(2)))
    if any(t.startswith("==> ") for _, t in lines):     # a Read of saved mrw output
        parse_mrw_output(sess, "\n".join(t for _, t in lines))
    elif path:
        for n, t in lines:
            sess.see(path, n, t)


def result_text(content):
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(b.get("text", "") for b in content
                         if isinstance(b, dict) and b.get("type") == "text")
    return ""


# ----------------------------------------------------------------- WRITES

def text_lines(s):
    """The lines of a file's text: a terminal newline ends the last line, it does not add an empty one."""
    lines = s.split("\n")
    return lines[:-1] if s.endswith("\n") else lines


def is_plan(text):
    return any(PLAN_HEADER.match(line) for line in text.split("\n"))


def parse_plan(text):
    """Hunks [(path, addr, op, body_lines)] from mrw plan text."""
    hunks = []
    lines = text.split("\n")
    i = 0
    while i < len(lines):
        m = PLAN_HEADER.match(lines[i])
        if not m:
            i += 1
            continue
        path, addr, op, rest = m.groups()
        bn = BODY_N.search(rest)
        j = i + 1
        if bn:
            body = lines[j:j + int(bn.group(1))]
            j += int(bn.group(1))
        else:
            k = j
            while k < len(lines) and not PLAN_HEADER.match(lines[k]):
                k += 1
            body = lines[j:k]
            j = k
            while body and body[-1] == "":
                body.pop()
        if op in ("delete", "unlink", "rename"):
            body = []
        hunks.append((path, addr, op, body))
        i = j
    return hunks


def addr_range(addr):
    m = RANGE.match(addr)
    if not m:
        return None
    a = int(m.group(1))
    if m.group(2):
        return (a, int(m.group(2)))
    if m.group(3):
        return (a, a + int(m.group(3)))
    return (a, a)


def bash_writes(cmd, files_written):
    """Write units from one Bash command, and a count of zero-body writes."""
    units = []
    zero = 0
    docs = [b for _, _, b in HEREDOC.findall(cmd)]
    for body in docs:
        if is_plan(body):
            units += [("mrw_plan", p, op, a, b) for p, a, op, b in parse_plan(body)]
    for m in CAT_HEREDOC.finditer(cmd):
        files_written[os.path.basename(m.group(1).strip("'\""))] = m.group(4)
    if "mrw" in cmd and "--create" in cmd:
        for m in re.finditer(r"--create\s+(\S+)([^\n]*)", cmd):
            path, tail = m.group(1).strip("'\""), m.group(2)
            body = None
            src = re.search(r"<\s*(\S+)", tail)
            if "<<" in tail:
                body = next((b for b in docs if not is_plan(b)), None)
            elif src:
                body = files_written.get(os.path.basename(src.group(1).strip("'\"")))
            if body is not None:
                units.append(("mrw_create", path, "create", "0", text_lines(body)))
    if "mrw write" in cmd and not docs:
        if "--sub" in cmd or "--rewrite" in cmd:
            zero += 1
        elif "@@ " in cmd:
            units += [("mrw_plan", p, op, a, b) for p, a, op, b in parse_plan(cmd.replace("\\n", "\n"))]
    return units, zero


def write_units(name, inp, files_written):
    """A tool_use as write units (family, path, op, addr, body, kept-or-None)."""
    zero = 0
    if name == "Bash":
        raw, zero = bash_writes(inp.get("command") or "", files_written)
    elif name.endswith("mrw_write"):
        raw = [("mrw_plan", p, op, a, b) for p, a, op, b in parse_plan(inp.get("plan") or "")]
        if not raw and (inp.get("sub") or inp.get("rewrite")):
            zero = 1
    elif name in ("Edit", "MultiEdit"):
        units = []
        for e in (inp.get("edits") if name == "MultiEdit" else [inp]) or []:
            if not isinstance(e, dict):
                continue
            old_s = e.get("old_string") or ""
            avail = collections.Counter(text_lines(old_s))
            new = text_lines(e.get("new_string") or "")
            kept = []
            for line in new:
                kept.append(avail[line] > 0)
                if avail[line] > 0:
                    avail[line] -= 1
            units.append((name, inp.get("file_path") or "", "edit", old_s, new, kept))
        return units, 0
    elif name == "Write":
        content = inp.get("content") or ""
        if is_plan(content):
            raw = [("mrw_plan", p, op, a, b) for p, a, op, b in parse_plan(content)]
        else:
            raw = [("Write", inp.get("file_path") or "", "write", "", text_lines(content))]
    else:
        return [], 0
    return [(f, p, op, a, b, None) for f, p, op, a, b in raw], zero


# --------------------------------------------------------- classification

def find_runs(sess, body, kept, region, normalised):
    """Greedy maximal runs of body lines that occur consecutively in the pool.

    region: (fid, n) positions excluded as sources (the lines being edited).
    Returns [(start, end_exclusive, src_fid, src_first_line)].
    """
    pos = sess.posn if normalised else sess.pos
    idx = sess.idxn if normalised else sess.idx
    lines = [s.strip() for s in body] if normalised else body
    runs = []
    floor = 0
    i = 0
    nb = len(lines)
    while i < nb:
        if kept[i] or nonws(lines[i]) < START_NONWS:
            i += 1
            continue
        best = None
        for (f, n) in idx.get(lines[i], ())[-MAX_CANDIDATES:]:
            if (f, n) in region or pos.get((f, n)) != lines[i]:
                continue
            a = i
            while a - 1 >= floor and not kept[a - 1]:
                src = (f, n - (i - a) - 1)
                if src in region or pos.get(src) != lines[a - 1]:
                    break
                a -= 1
            b = i + 1
            while b < nb and not kept[b]:
                src = (f, n + (b - i))
                if src in region or pos.get(src) != lines[b]:
                    break
                b += 1
            if best is None or b - a > best[1] - best[0]:
                best = (a, b, f, n - (i - a))
        if best:                              # a copy does not start or end on a blank line
            a, b, f, n0 = best
            while a < b and not lines[a].strip():
                a, n0 = a + 1, n0 + 1
            while b > a and not lines[b - 1].strip():
                b -= 1
            best = (a, b, f, n0)
        if best and sum(nonws(s) for s in lines[best[0]:best[1]]) >= MIN_NONWS:
            runs.append(best)
            floor = i = best[1]
        else:
            i += 1
    return runs


def classify(sess, unit):
    """KEPT flags and runs for one write unit."""
    family, path, op, addr, body, kept = unit
    f = sess.fid(path) if path else None
    region = set()
    if f is not None and op == "replace":
        r = addr_range(addr)
        if r:
            region = {(f, n) for n in range(r[0], r[1] + 1)}
    elif f is not None and (family == "Write" or op == "create"):   # rewriting a file already read
        region = {(f, n) for n in sess.by_file.get(f, {})}
    if kept is None:
        texts = {sess.pos[p] for p in region if p in sess.pos}
        kept = [s in texts for s in body]
    res = {"kept": kept, "fid": f, "pattern_replace": op == "replace" and not addr_range(addr)}
    for norm in (False, True):
        res["norm" if norm else "exact"] = find_runs(sess, body, kept, region, norm)
    return res


def removals(sess, units):
    """What a write takes away: (fid, (a, b)) line ranges or (fid, old_string)."""
    rem = []
    for _, path, op, addr, _, _ in units:
        if not path:
            continue
        if op in ("delete", "replace") and addr_range(addr):
            rem.append((sess.fid(path), addr_range(addr)))
        elif op == "edit" and addr:
            rem.append((sess.fid(path), addr))
    return rem


def is_moved(run, rem):
    src, n0, n1, text = run
    for f, what in rem:
        if f == src and (what[0] <= n0 and n1 <= what[1] if isinstance(what, tuple) else text in what):
            return True
    return False


def run_lines(body, a, b):
    """A run's length for k: its non-blank lines, so blank padding cannot lengthen it."""
    return sum(1 for s in body[a:b] if s.strip())


def bucket(n):
    for lo, hi in ((3, 4), (5, 9), (10, 19), (20, 49)):
        if lo <= n <= hi:
            return f"{lo}-{hi}"
    return "50+"


class Reservoir:
    def __init__(self, k):
        self.k, self.n, self.items = k, 0, []

    def offer(self, rng, item):
        self.n += 1
        if len(self.items) < self.k:
            self.items.append(item)
        elif rng.randrange(self.n) < self.k:
            self.items[rng.randrange(self.k)] = item


# ------------------------------------------------------------------- scan

class Scan:
    def __init__(self, seed=167):
        self.rng = random.Random(seed)
        self.stats = collections.defaultdict(collections.Counter)
        self.hist = collections.Counter()
        self.moves = collections.Counter()
        self.examples = Reservoir(5)

    def account(self, keys, fam, body, res, s):
        nb = len(body)
        total = chars(body)
        kept_c = chars([x for x, k in zip(body, res["kept"]) if k])
        s["body_chars"] += total
        s["kept_chars"] += kept_c
        for key in keys:
            c = self.stats[key]
            c["units"] += 1
            c["body_lines"] += nb
            c["body_chars"] += total
            c["kept_chars"] += kept_c
            c["pattern_replace_hunks"] += res["pattern_replace"]
        for tag in ("exact", "norm"):
            for k in KS:
                sel = [r for r in res[tag] if run_lines(body, r[0], r[1]) >= k]
                cc = sum(chars(body[a:b]) for a, b, _, _ in sel)
                # ADR-167: from= is per hunk, so only a body that is ONE run saves as-is
                whole = len(sel) == 1 and sel[0][0] == 0 and sel[0][1] == nb
                whole_save = max(0, total - HEADER_COST) if whole else 0
                # split: each run its own from= hunk; a middle run also needs a header to resume typing
                split = sum(max(0, chars(body[a:b]) - HEADER_COST * (1 if a == 0 or b == nb else 2))
                            for a, b, _, _ in sel)
                for key in keys:
                    c = self.stats[key]
                    c[f"copy_chars_{tag}_k{k}"] += cc
                    c[f"copy_runs_{tag}_k{k}"] += len(sel)
                    c[f"whole_units_{tag}_k{k}"] += whole
                    c[f"whole_save_{tag}_k{k}"] += whole_save
                    c[f"split_save_{tag}_k{k}"] += split
                if k == 3:
                    s[f"copy_{tag}_k3"] += cc
                    s[f"whole_save_{tag}_k3"] += whole_save
                    s[f"split_save_{tag}_k3"] += split

    def file(self, path, kind):
        sess = Session()
        tools = {}
        files_written = {}
        out_tokens = {}
        s = collections.Counter()
        pending = []
        with open(path, "r", encoding="utf-8", errors="replace") as fh:
            for line in fh:
                if '"tool_use"' not in line and '"tool_result"' not in line:
                    if '"output_tokens"' in line and '"assistant"' in line:
                        m = USAGE.search(line)
                        if m:
                            out_tokens[m.group(1)] = int(m.group(2))
                    continue
                try:
                    o = json.loads(line)
                except ValueError:
                    continue
                msg = o.get("message")
                if not isinstance(msg, dict):
                    continue
                if o.get("type") == "assistant" and isinstance(msg.get("usage"), dict) and msg.get("id"):
                    out_tokens[msg["id"]] = msg["usage"].get("output_tokens") or 0
                content = msg.get("content")
                if not isinstance(content, list):
                    continue
                for b in content:
                    if not isinstance(b, dict):
                        continue
                    if b.get("type") == "tool_use":
                        name, inp = b.get("name") or "", b.get("input") or {}
                        if not isinstance(inp, dict):
                            continue
                        if b.get("id") in tools:      # a tool_use repeated in the transcript counts once
                            continue
                        tools[b.get("id")] = (name, inp)
                        units, zero = write_units(name, inp, files_written)
                        s["zero_body_writes"] += zero
                        if units:
                            pending = self.write(sess, units, kind, s, pending)
                    elif b.get("type") == "tool_result":
                        name, inp = tools.get(b.get("tool_use_id"), ("", {}))
                        text = result_text(b.get("content"))
                        if not text:
                            continue
                        if name == "Read":
                            tur = o.get("toolUseResult")
                            fp = tur["file"].get("filePath") if isinstance(tur, dict) and \
                                isinstance(tur.get("file"), dict) else None
                            parse_read_result(sess, text, fp or inp.get("file_path"))
                        elif (name == "Bash" and "mrw" in (inp.get("command") or "")) or name.endswith("mrw_read"):
                            parse_mrw_output(sess, text)
        s["output_tokens"] = sum(out_tokens.values())
        s["served_chars"] = sess.served_chars
        s["redundant_chars"] = sess.redundant_chars
        s["is_subagent"] = int(kind == "subagent")
        return s

    def write(self, sess, units, kind, s, pending):
        rem = removals(sess, units)
        for u in units:
            if u[1]:
                sess.invalidate(u[1])
        for r in pending:                     # runs of the previous write whose source this one removes
            if is_moved(r[:4], rem):
                self.moves["moved_runs"] += 1
                self.moves["moved_chars"] += r[4]
        new_pending = []
        for u in units:
            fam, body = u[0], u[4]
            res = classify(sess, u)
            s["write_units"] += 1
            self.account([("all",), ("family", fam), ("kind", kind)], fam, body, res, s)
            for a, b, src, n0 in res["exact"]:
                n = run_lines(body, a, b)
                if n < 3:
                    continue
                c = chars(body[a:b])
                self.hist[bucket(n)] += 1
                self.moves["runs"] += 1
                self.moves["chars"] += c
                run = (src, n0, n0 + n - 1, "\n".join(body[a:b]), c)
                if is_moved(run[:4], rem):
                    self.moves["moved_runs"] += 1
                    self.moves["moved_chars"] += c
                else:
                    new_pending.append(run)
                self.examples.offer(self.rng, {
                    "lines": n, "chars": c, "family": fam, "same_file": src == res["fid"],
                    "whole_body": a == 0 and b == len(body), "_peek": body[a][:60]})
        return new_pending


def iter_transcripts(root, min_bytes):
    out = []
    for proj in sorted(os.listdir(root)):
        pdir = os.path.join(root, proj)
        if not os.path.isdir(pdir):
            continue
        for dirpath, _, files in os.walk(pdir):
            for fn in files:
                p = os.path.join(dirpath, fn)
                if fn.endswith(".jsonl") and os.path.getsize(p) > min_bytes:
                    kind = "subagent" if os.sep + "subagents" + os.sep in p else "main"
                    out.append((proj, kind, p))
    return out


def view(c):
    bc = c["body_chars"] or 1
    v = {"write_units": c["units"], "body_lines": c["body_lines"], "body_chars": c["body_chars"],
         "body_tokens_est": tokens(c["body_chars"]), "kept_chars": c["kept_chars"],
         "kept_share": round(c["kept_chars"] / bc, 4), "pattern_replace_hunks": c["pattern_replace_hunks"]}
    for tag in ("exact", "norm"):
        for k in KS:
            v[f"{tag}_k{k}"] = {
                "copyable_runs": c[f"copy_runs_{tag}_k{k}"],
                "copyable_chars": c[f"copy_chars_{tag}_k{k}"],
                "copyable_share": round(c[f"copy_chars_{tag}_k{k}"] / bc, 4),
                "whole_body_units": c[f"whole_units_{tag}_k{k}"],
                "whole_body_saving_chars": c[f"whole_save_{tag}_k{k}"],
                "whole_body_saving_tokens_est": tokens(c[f"whole_save_{tag}_k{k}"]),
                "split_hunk_saving_chars": c[f"split_save_{tag}_k{k}"],
                "split_hunk_saving_tokens_est": tokens(c[f"split_save_{tag}_k{k}"]),
            }
    return v


def summarise(scan, per_session, per_project, corpus, secs):
    ws = [s for s in per_session if s["write_units"]]
    dec = max(1, len(ws) // 10)

    def top_share(key):
        xs = sorted((s[key] for s in ws), reverse=True)
        t = sum(xs)
        return round(sum(xs[:dec]) / t, 4) if t else None
    out_tok = sum(s["output_tokens"] for s in per_session)
    allc = scan.stats[("all",)]
    rows = []
    for i, c in enumerate(sorted(per_project.values(), key=lambda c: -c["body_chars"])[:10]):
        bc = c["body_chars"] or 1
        rows.append({"label": f"P{i + 1}", "transcripts": c["transcripts"], "body_chars": c["body_chars"],
                     "copyable_exact_k3_share": round(c["copy_exact_k3"] / bc, 4),
                     "whole_body_saving_exact_k3_tokens_est": tokens(c["whole_save_exact_k3"])})

    def redundancy(rows):
        served = sum(s["served_chars"] for s in rows)
        red = sum(s["redundant_chars"] for s in rows)
        return {"transcripts": len(rows), "served_chars": served, "served_tokens_est": tokens(served),
                "redundant_chars": red, "redundant_tokens_est": tokens(red),
                "redundant_share": round(red / served, 4) if served else None}
    rr = [s for s in per_session if s["served_chars"]]
    reds = sorted((s["redundant_chars"] for s in rr), reverse=True)
    read_redundancy = dict(
        redundancy(rr),
        top_decile_share_of_redundant=round(sum(reds[:max(1, len(rr) // 10)]) / sum(reds), 4) if sum(reds) else None,
        main=redundancy([s for s in rr if not s["is_subagent"]]),
        subagent=redundancy([s for s in rr if s["is_subagent"]]))
    return {
        "corpus": dict(corpus, transcripts_with_writes=len(ws), runtime_seconds=round(secs, 1),
                       zero_body_writes=sum(s["zero_body_writes"] for s in per_session),
                       output_tokens_reported=out_tok),
        "assumptions": {"k_headline": 3, "min_nonws_chars_per_run": MIN_NONWS,
                        "seed_line_min_nonws": START_NONWS, "max_candidates_per_seed": MAX_CANDIDATES,
                        "header_cost_chars": HEADER_COST, "chars_per_token": 4},
        "all": view(allc),
        "headline_share_of_reported_output_tokens": {
            tag: round(tokens(allc[f"whole_save_{tag}_k3"]) / out_tok, 5) if out_tok else None
            for tag in ("exact", "norm")},
        "split_hunk_share_of_reported_output_tokens": {
            tag: round(tokens(allc[f"split_save_{tag}_k3"]) / out_tok, 5) if out_tok else None
            for tag in ("exact", "norm")},
        "by_family": {k[1]: view(v) for k, v in sorted(scan.stats.items()) if k[0] == "family"},
        "by_kind": {k[1]: view(v) for k, v in sorted(scan.stats.items()) if k[0] == "kind"},
        "distribution_exact_k3": {
            "transcripts_with_any_copyable_run": sum(1 for s in ws if s["copy_exact_k3"]),
            "transcripts_with_whole_body_saving": sum(1 for s in ws if s["whole_save_exact_k3"]),
            "top_decile_share_of_whole_body_saving": top_share("whole_save_exact_k3"),
            "top_decile_share_of_split_hunk_saving": top_share("split_save_exact_k3"),
            "top_decile_share_of_write_body_chars": top_share("body_chars"),
        },
        "run_length_histogram_exact_k3": dict(sorted(scan.hist.items())),
        "moves_exact_k3": dict(scan.moves),
        "by_project_top10": rows,
        "read_redundancy": read_redundancy,
    }


def main(argv=None):
    ap = argparse.ArgumentParser(description="copy-study scanner (ADR-167)")
    ap.add_argument("--root", default=os.path.expanduser("~/.claude/projects"))
    ap.add_argument("--limit", type=int, default=0, help="scan only the first N transcripts (sorted by path)")
    ap.add_argument("--min-bytes", type=int, default=10 * 1024)
    ap.add_argument("--out", default="")
    ap.add_argument("--examples", action="store_true", help="print 5 sampled runs to stderr; never published")
    a = ap.parse_args(argv)
    t0 = time.time()
    scan = Scan()
    files = sorted(iter_transcripts(a.root, a.min_bytes), key=lambda x: x[2])
    if a.limit:
        files = files[:a.limit]
    per_session = []
    per_project = collections.defaultdict(collections.Counter)
    corpus = collections.Counter()
    for proj, kind, p in files:
        try:
            s = scan.file(p, kind)
        except OSError:
            corpus["unreadable"] += 1
            continue
        corpus[f"transcripts_{kind}"] += 1
        corpus["bytes"] += os.path.getsize(p)
        s["transcripts"] = 1
        per_session.append(s)
        per_project[proj].update(s)
    corpus["project_dirs"] = len(per_project)
    res = summarise(scan, per_session, per_project, dict(corpus), time.time() - t0)
    text = json.dumps(res, indent=2, sort_keys=True)
    if a.out:
        with open(a.out, "w", encoding="utf-8") as fh:
            fh.write(text + "\n")
    else:
        print(text)
    if a.examples:
        for e in scan.examples.items:
            print(json.dumps(e), file=sys.stderr)
    return res


if __name__ == "__main__":
    main()
