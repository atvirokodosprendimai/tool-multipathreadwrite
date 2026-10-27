"""Within each session: how much tool output repeats something served earlier
in the same session (whole results, and mrw file blocks by path+sha+text).
Read-only; prints one JSON summary.

    python3 -I measure_repeat.py --since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z

The window is on each transcript line's own timestamp, as in measure_output.py.
A block counts as content only when it holds a numbered line (`NNN| `): an
answer by reference (`-- same as served`, ADR-179) or a one-line unchanged
answer (ADR-169) is counted apart, never as a repeat."""
import argparse, collections, datetime, glob, hashlib, json, os, re

ap = argparse.ArgumentParser()
ap.add_argument("--since", required=True, help="ISO 8601 UTC start, inclusive")
ap.add_argument("--until", required=True, help="ISO 8601 UTC end, exclusive")
ap.add_argument("--glob", default="~/.claude/projects/*/*.jsonl", help="main-session transcripts only by default")
args = ap.parse_args()


def ts(v):
    return datetime.datetime.fromisoformat(v.replace("Z", "+00:00"))


since, until = ts(args.since), ts(args.until)
files = [f for f in glob.glob(os.path.expanduser(args.glob)) if os.path.getmtime(f) >= since.timestamp()]
HDR = re.compile(r"^==> (\S+)\s+\d+L\s+\d+B\s+sha ([0-9a-f]{8})")
NUMBERED = re.compile(r"^\s*\d+\| ")
s = collections.Counter()
sessions = 0


def blocks(text):
    """Yield (path, sha, body lines) for each mrw file block in one result."""
    blk, buf = None, []
    for ln in text.splitlines() + ["==> END 0L 0B sha 00000000"]:
        mh = HDR.match(ln)
        if mh or ln.startswith("==> END"):
            if blk:
                yield blk[0], blk[1], buf
            blk = (mh.group(1), mh.group(2)) if mh else None
            buf = []
        elif blk:
            buf.append(ln)


for f in files:
    seen_res, seen_blk, uses, any_ = set(), set(), {}, False
    for line in open(f, encoding="utf-8", errors="replace"):
        if '"tool_' not in line:
            continue
        try:
            m = json.loads(line)
        except Exception:
            continue
        c = (m.get("message") or {}).get("content")
        if not isinstance(c, list):
            continue
        try:
            inside = m.get("timestamp") is not None and since <= ts(m["timestamp"]) < until
        except ValueError:
            inside = False
        for b in c:
            if b.get("type") == "tool_use":
                uses[b.get("id")] = b.get("name")
            if b.get("type") != "tool_result" or b.get("tool_use_id") not in uses or not inside:
                continue
            cc = b.get("content")
            t = "".join(x.get("text", "") for x in cc if isinstance(x, dict)) if isinstance(cc, list) else str(cc or "")
            any_ = True
            s["all"] += len(t)
            if len(t) >= 500:
                h = hashlib.sha1(t.encode()).hexdigest()
                if h in seen_res:
                    s["repeat_whole_chars"] += len(t)
                    s["repeat_whole_calls"] += 1
                seen_res.add(h)
            for path, sha, body in blocks(t):
                ref = [ln for ln in body if ln.startswith("-- same as served")]
                if ref:
                    s["reference_chars"] += sum(len(ln) + 1 for ln in ref)
                    s["reference_count"] += len(ref)
                if not any(NUMBERED.match(ln) for ln in body):
                    continue  # no content served: a reference, an unchanged answer, a refusal
                text = "\n".join(ln for ln in body if not ln.startswith("-- same as served"))
                key = (path, sha, hashlib.sha1(text.encode()).hexdigest())
                s["mrw_block_chars"] += len(text)
                s["mrw_block_count"] += 1
                if key in seen_blk:
                    s["repeat_block_chars"] += len(text)
                    s["repeat_block_count"] += 1
                seen_blk.add(key)
    sessions += any_
print(json.dumps({"since": args.since, "until": args.until, "sessions": sessions, **s}, indent=1))
