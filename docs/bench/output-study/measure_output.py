"""Measure where tool output goes across Claude Code transcripts, and what each
candidate mrw feature would have cut. Read-only; prints one JSON summary.

    python3 -I measure_output.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z

A tool result is counted when its transcript line's own timestamp falls in
[since, until): a fixed window, so a later run over the same window counts the
same results whatever the files' mtimes (ADR-177/178/179 pre-registration)."""
import argparse, collections, datetime, glob, json, os, re

ap = argparse.ArgumentParser()
ap.add_argument("--since", required=True, help="ISO 8601 UTC start, inclusive")
ap.add_argument("--until", required=True, help="ISO 8601 UTC end, exclusive")
ap.add_argument("--glob", default="~/.claude/projects/*/*.jsonl", help="main-session transcripts only by default")
args = ap.parse_args()


def ts(s):
    return datetime.datetime.fromisoformat(s.replace("Z", "+00:00"))


since, until = ts(args.since), ts(args.until)
files = glob.glob(os.path.expanduser(args.glob))

MRW = r"\bmrw(?: (?:--root|-C) (?:'[^']*'|\S+))?"
MRW_READ = re.compile(MRW + r" read\b")
MRW_WRITE = re.compile(MRW + r" write\b")
MRW_RUN = re.compile(MRW + r" run\b")
PIPE_TRIM = re.compile(r"\|\s*(tail|head)\b")
NOTICE = re.compile(r"^mrw: (the answer went to the null device|still licensed in this version)")
# a line mrw prints about the answer, not a line of the file: stderr notices and
# `-- ` notes, without the per-range reports a read is meant to repeat
NOTE = re.compile(r"^(mrw: |-- (?!ck |\d+ matching line))")
NUMBERED = re.compile(r"^\s*\d+\| ?")
NUM = re.compile(r"\d+")
# a file block's header: the path runs up to the metadata suffix, so it may hold spaces
HDR = re.compile(r"^==> (.+?)\s+(\d+)L\s+\d+B\s+sha [0-9a-f]{8}\s*$")

tot = collections.Counter()
calls = collections.Counter()
s = collections.Counter()
sessions_with = collections.Counter()


def cat(name, inp):
    if name != "Bash":
        return name
    c = inp.get("command", "")
    r, w, run = MRW_READ.search(c), MRW_WRITE.search(c), MRW_RUN.search(c)
    if sum(bool(x) for x in (r, w, run)) > 1:
        return "mrw mixed"  # attributed to no single mrw bucket
    if run:
        return "mrw run"
    if r:
        return "mrw read"
    if w:
        return "mrw write"
    if re.search(r"(^|[;&|]\s*)(cat|sed -n|head|tail|grep|rg)\b", c):
        return "bash read-ish (cat/grep/rg/sed)"
    if re.search(r"\b(go test|qh-check|contract\.sh|static\.sh|npm test|pytest|cargo test)\b", c):
        return "bash test/gate"
    if re.search(r"\bgit\b", c):
        return "bash git"
    if re.search(r"\bgh\b", c):
        return "bash gh"
    return "bash other"


def is_whole_file_mrw_read(cmd):
    # every spec has no ':' range and no --grep / -
    if not MRW_READ.search(cmd) or "--grep" in cmd or "--ast-grep" in cmd:
        return False
    after = MRW_READ.split(cmd, 1)[-1].split("&&")[0].split("|")[0].split(";")[0]
    specs = [a.strip("'\"") for a in after.split() if not a.startswith("-")]
    return bool(specs) and all(":" not in sp for sp in specs)


def served_blocks(text):
    """Yield (path, header line count, characters served, numbered lines served)
    for each mrw file block in one result. The characters are the header, the
    range lines and the numbered lines: not the notes mrw prints beside them."""
    blk, chars, numbered = None, 0, 0
    for ln in text.splitlines() + ["==> END 0L 0B sha 00000000"]:
        mh = HDR.match(ln)
        if mh or ln.startswith("==> END"):
            if blk:
                yield blk[0], blk[1], chars, numbered
            blk = (mh.group(1), int(mh.group(2))) if mh else None
            chars, numbered = (len(ln) + 1, 0) if mh else (0, 0)
        elif blk and NUMBERED.match(ln):
            chars += len(ln) + 1
            numbered += 1
        elif blk and ln.startswith("@@ "):
            chars += len(ln) + 1


def json_docs(text):
    out = []
    for line in text.splitlines():
        line = line.strip()
        if len(line) >= 1000 and line[:1] in "{[":
            try:
                json.loads(line)
                out.append(len(line))
            except Exception:
                pass
    if not out:
        t = text.strip()
        if len(t) >= 1000 and t[:1] in "{[":
            try:
                json.loads(t)
                out.append(len(t))
            except Exception:
                pass
    return out


def fold_savings(lines):
    # mrw's `NNN| ` prefix is stripped first: a numbered blank line is a blank
    # line, and two numbered lines are compared by their text, as --fold will.
    exact = near = 0
    prev = prevn = None
    for raw in lines:
        line = NUMBERED.sub("", raw, count=1)
        if not line.strip():
            prev = prevn = None
            continue
        n = NUM.sub("#", line)
        if line == prev:
            exact += len(raw) + 1
        if n == prevn:
            near += len(raw) + 1
        prev, prevn = line, n
    return exact, near


for f in files:
    uses = {}
    seen_flags = set()
    mrw_calls = 0
    results = 0
    try:
        fh = open(f, encoding="utf-8", errors="replace")
    except OSError:
        continue
    for line in fh:
        if '"tool_' not in line:
            continue
        try:
            m = json.loads(line)
        except Exception:
            continue
        c = (m.get("message") or {}).get("content")
        if not isinstance(c, list):
            continue
        stamp = m.get("timestamp")
        try:
            inside = stamp is not None and since <= ts(stamp) < until
        except ValueError:
            inside = False
        for b in c:
            t = b.get("type")
            if t == "tool_use":
                uses[b.get("id")] = (b.get("name"), b.get("input") or {})
            elif t == "tool_result" and inside:
                u = uses.get(b.get("tool_use_id"))
                if not u:
                    continue
                name, inp = u
                cc = b.get("content")
                text = "".join(x.get("text", "") for x in cc if isinstance(x, dict)) if isinstance(cc, list) else str(cc or "")
                results += 1
                k = cat(name, inp)
                n = len(text)
                tot[k] += n
                calls[k] += 1
                tot["ALL"] += n
                calls["ALL"] += 1
                lines = text.splitlines()
                cmd = inp.get("command", "") if name == "Bash" else ""
                if k.startswith("mrw"):
                    mrw_calls += 1
                # 1. long lines (>400) in reads: chars saved by cutting at 200
                if k in ("mrw read", "bash read-ish (cat/grep/rg/sed)", "Read"):
                    sav = sum(len(ln) - 200 for ln in lines if len(ln) > 400)
                    if sav:
                        s[f"longlines_saved_{k}"] += sav
                        s[f"longlines_calls_{k}"] += 1
                        seen_flags.add("longlines")
                    if k == "mrw read" and "--max-cols" in cmd:
                        s["mrw_read_with_maxcols"] += 1
                if k == "mrw read" and "--grep" in cmd:
                    s["mrw_read_grep_chars"] += n
                    s["mrw_read_grep_calls"] += 1
                # 2. JSON docs >=1KB in any output
                docs = json_docs(text)
                if docs:
                    s["json_docs_chars"] += sum(docs)
                    s["json_docs_count"] += len(docs)
                    s["json_docs_calls"] += 1
                    seen_flags.add("json")
                    if k == "mrw read":
                        s["json_docs_chars_via_mrw_read"] += sum(docs)
                if k == "mrw read" and "--json-path" in cmd:
                    s["mrw_json_path_calls"] += 1
                    s["mrw_json_path_chars"] += n
                # 3. fold
                ex, nr = fold_savings(lines)
                s["fold_exact_saved"] += ex
                s["fold_near_saved"] += nr
                if nr > 500:
                    s["fold_near_calls_over_500"] += 1
                    seen_flags.add("fold")
                # 4. whole-file reads (outline candidate)
                if k == "Read" and not inp.get("offset") and not inp.get("limit") and len(lines) > 150:
                    s["whole_file_Read_chars"] += n
                    s["whole_file_Read_calls"] += 1
                    seen_flags.add("outline")
                if k == "mrw read" and is_whole_file_mrw_read(cmd):
                    # one count per file block that was served completely (every
                    # line its header names) and holds more than 150 lines: a
                    # capped read of a long file, or a batch of short ones, is not
                    for _path, nl, chars, numbered in served_blocks(text):
                        if nl > 150 and numbered == nl:
                            s["whole_file_mrw_read_chars"] += chars
                            s["whole_file_mrw_read_calls"] += 1
                            seen_flags.add("outline")
                # 5. mrw run: big non-mrw bash outputs, and exit-code-losing pipes
                if name == "Bash" and not k.startswith("mrw"):
                    s["bash_nonmrw_calls"] += 1
                    if n > 4000:
                        s["bash_big_chars"] += n
                        s["bash_big_calls"] += 1
                        s["bash_big_saved_if_summary_300"] += n - 300
                        seen_flags.add("run")
                        if k != "bash read-ish (cat/grep/rg/sed)":
                            s["bash_big_nonread_chars"] += n
                            s["bash_big_nonread_calls"] += 1
                    if PIPE_TRIM.search(cmd):
                        s["bash_piped_head_tail_calls"] += 1
                if k == "mrw run":
                    s["mrw_run_calls"] += 1
                    s["mrw_run_chars"] += n
                if "mrw-runs/" in cmd and k != "mrw run":
                    s["mrw_runs_log_read_calls"] += 1
                    s["mrw_runs_log_read_chars"] += n
                # 6. the write receipt
                if k == "mrw write" and "--terse" in cmd:
                    s["mrw_write_terse_calls"] += 1
                    s["mrw_write_terse_chars"] += n
                # 7. stderr notices: the two counted at baseline, and any notice line
                # repeated identically inside one tool result
                rep = sum(len(ln) for ln in lines if NOTICE.match(ln))
                if rep:
                    s["mrw_notice_chars"] += rep
                    s["mrw_notice_calls"] += 1
                notes = collections.Counter(ln for ln in lines if NOTE.match(ln))
                dup = sum(len(ln) * (c - 1) for ln, c in notes.items() if c > 1)
                if dup:
                    s["notice_repeat_chars"] += dup
                    s["notice_repeat_calls"] += 1
                # 8. answers by reference (ADR-179 d)
                ref = sum(len(ln) + 1 for ln in lines if ln.startswith("-- same as served"))
                if ref:
                    s["same_as_served_chars"] += ref
                    s["same_as_served_lines"] += sum(1 for ln in lines if ln.startswith("-- same as served"))
    fh.close()
    if mrw_calls:
        sessions_with["any mrw call"] += 1
    if results:
        sessions_with["in window"] += 1
    for fl in seen_flags:
        sessions_with[fl] += 1

print(json.dumps({"since": args.since, "until": args.until, "files": len(files),
                  "sessions_with": dict(sessions_with), "totals": tot.most_common(), "calls": calls.most_common(),
                  "candidates": dict(s)}, indent=1))
