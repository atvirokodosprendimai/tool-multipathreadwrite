#!/usr/bin/env python3
"""licence194_replay: price ADR-194's narrow licence on the Stage 8 pilot transcripts (ADR-194 T1).

ADR-194 records, after a write to a partly licensed file, the caller's licence carried into the new
numbering plus the lines written, where mrw v1.83.0 records the whole file. This replays each attempt's
`mrw` calls in transcript order and counts the landed hunks the narrow rule would refuse, by the
criterion pre-registered in docs/adr/BACKLOG.md ("From ADR-194") before this script existed:

  W  hunks of writes that printed `applied`, path ops included;
  U  hunks of W the replay cannot place, or that fall on an UNCERTAIN file;
  R  hunks of W not in U that the narrow rule refuses.

Verdict, first rule that applies: INCONCLUSIVE if W < 50 or U > W/5; STOP if R > W/20; GO if
R + U <= W/20; otherwise INCONCLUSIVE.

How the replay decides what the criterion leaves to it, each written here before the first run:
- It runs mrw v1.83.0's rule beside the narrow one. A hunk the replay finds unlicensed under BOTH
  rules landed all the same in the real run, so the replay lost a licence it could not see: it is U
  ("the replay lost a licence"), never R.
- A SEARCH/REPLACE hunk's quoted lines are licensed by the quote (ADR-190). For a multi-line one, the
  line after is the quoted kept line or the ledger's. One that v1.83.0's rule did not license either
  landed by its quote and still would; one only v1.83.0's whole-file licence covered is U, since the
  replay cannot tell whether the quote kept it.
- Several `mrw write` calls in one command are matched to their receipts in order; when the counts
  differ, every hunk of that result is U.
- A `--terse` receipt names no hunk address, so its hunks are U and its files UNCERTAIN.

Usage: licence194_replay.py [--corpus DIR] [--out DIR]     run, write replay.json, manifest.txt, REPLAY.md
       licence194_replay.py --verify docs/bench/licence194/replay.json
                                                         re-hash the corpus, re-run, exit 1 on any difference
"""
import argparse
import glob
import hashlib
import json
import os
import re
import sys

WHOLE = "whole"
UNCERTAIN_FLAGS = ("--quiet", "--max-cols", "--fold", "--no-comments", "--if-changed", "--if-served")

HEADER = re.compile(r"^==> (.+?)  (\d+)L  ")
SPAN = re.compile(r"^@@ (\d+)-(\d+)\s*$")
NUMBERED = re.compile(r"^\s*(\d+)\| ")
ROW = re.compile(r"^(ok|FAIL|skip|dry)\s+(\S+)\s+(\S+)\s+([a-z-]+)\s+-(\d+) \+(\d+)")
TERSE = re.compile(r"^(ok|dry)\s+(\S+)\s+(\d+) hunk\(s\)")
WROTE = re.compile(r"^(?:wrote|created) (.+?)  (\d+)L -> (\d+)L")
SUMMARY = re.compile(r"^(\d+) hunk\(s\), \d+ file\(s\), \d+ failed.* — (.+)$")
INVOCATION = re.compile(r"(?:^|[\s;&|(])(?:\S*/)?mrw((?:\s+--?[\w-]+(?:[= ]\S+)?)*?)\s+(read|write)\b([^\n;&|]*)(\|(?!\|))?")


def verdict(w, r, u):
    """The pre-registered verdict (docs/adr/BACKLOG.md, From ADR-194)."""
    if w < 50 or u > w / 5:
        return "INCONCLUSIVE"
    if r > w / 20:
        return "STOP"
    if r + u <= w / 20:
        return "GO"
    return "INCONCLUSIVE"


class File:
    """One file's licence under the narrow rule and under v1.83.0's, and whether it is uncertain."""

    def __init__(self):
        self.narrow, self.today, self.uncertain, self.lines = set(), set(), False, 0

    @staticmethod
    def covers(lic, needed):
        return lic == WHOLE or needed <= lic


def invocations(cmd):
    """The mrw read and write calls of one Bash command, in order, each as (sub, global flags, rest, piped)."""
    out = []
    for line in cmd.split("\n"):
        for m in INVOCATION.finditer(line):
            out.append((m.group(2), m.group(1) or "", m.group(3) or "", bool(m.group(4))))
    return out


def write_mode(glob_flags, rest):
    text = glob_flags + " " + rest
    if re.search(r"--root[= ](?!\.(\s|$))", glob_flags):
        return "elsewhere"
    if re.search(r"--format[= ]search_replace", text):
        return "sr"
    if "--grep" in text or "--ast-grep" in text:
        return "sub"
    if "--force" in text:
        return "force"
    return "plan"


def needed_lines(op, a, b, lines):
    """The lines a hunk needs licensed, and the line after a multi-line replace (ADR-052)."""
    if op in ("unlink", "rename"):
        return None, None  # needs the whole file
    if op == "create":
        return set(), None
    if op in ("insert-after", "insert-before"):
        return ({a} if a > 0 else set()), None
    after = b + 1 if op == "replace" and b > a and b < lines else None
    return set(range(a, b + 1)), after


def carry(lic, hunks, new_lines):
    """lic (a set of old line numbers) carried into the numbering after hunks, plus the lines written."""
    if lic == WHOLE:
        return WHOLE
    removed = set()
    for h in hunks:
        if h["op"] in ("replace", "delete"):
            removed |= set(range(h["a"], h["b"] + 1))

    def shift(line):
        s = 0
        for h in hunks:
            delta = h["added"] - h["removed"]
            if h["op"] in ("replace", "delete") and h["b"] < line:
                s += delta
            elif h["op"] == "insert-after" and h["a"] < line:
                s += h["added"]
            elif h["op"] == "insert-before" and h["a"] <= line:
                s += h["added"]
        return s

    out = {line + shift(line) for line in lic if line not in removed}
    for h in hunks:
        if not h["added"]:
            continue
        if h["op"] == "replace":
            start = h["a"] + shift(h["a"])
        elif h["op"] == "insert-after":
            start = h["a"] + shift(h["a"]) + 1
        elif h["op"] == "insert-before":
            start = h["a"] + shift(h["a"]) - h["added"]
        else:
            continue
        out |= set(range(start, start + h["added"]))
    return {line for line in out if 1 <= line <= new_lines}


class Replay:
    """One attempt's replay; outcomes holds one (kind, path, line, op, reason) per hunk of W."""

    def __init__(self):
        self.files, self.outcomes = {}, []

    def file(self, path):
        return self.files.setdefault(path, File())

    def event(self, cmd, result):
        calls = invocations(cmd)
        reads_uncertain = any(sub == "read" and (piped or any(f in rest for f in UNCERTAIN_FLAGS))
                              for sub, _, rest, piped in calls) or "MRW_MAX_COLS" in cmd
        writes = [(g, rest) for sub, g, rest, _ in calls if sub == "write"]
        items = self.parse(result)
        paired = sum(1 for kind, _ in items if kind == "receipt") == len(writes)
        pending = iter(writes)
        # In output order, so a read printed after a write licenses nothing that write needed (Codex, review of #620).
        for kind, payload in items:
            if kind == "read":
                self.read(*payload, reads_uncertain)
                continue
            rows, status, terse, wrote = payload
            applied = status.startswith("applied") or "PARTIALLY" in status
            if not paired:
                if applied:
                    for row in rows:
                        self.unplaced(row["path"], row.get("a"), row["op"], "write calls and receipts do not pair")
                    for path, _ in terse:
                        self.unplaced(path, None, "terse", "write calls and receipts do not pair")
                continue
            g, rest = next(pending)
            if applied:
                self.write(write_mode(g, rest), rows, terse, wrote, partial="PARTIALLY" in status,
                           created_by_flag="--create" in rest)

    @staticmethod
    def parse(result):
        """The result's read blocks and write receipts, in the order they were printed."""
        items = []
        rows, terse, wrote = [], [], {}
        cur = None
        for line in result.split("\n"):
            m = HEADER.match(line)
            if m:
                cur = [m.group(1), int(m.group(2)), [], set()]
                items.append(("read", cur))
                continue
            if cur is not None:
                s, n = SPAN.match(line), NUMBERED.match(line)
                if s:
                    cur[2].append((int(s.group(1)), int(s.group(2))))
                    continue
                if n and cur[2]:
                    cur[3].add(int(n.group(1)))
                    continue
            r = ROW.match(line)
            if r:
                cur = None
                addr = r.group(3)
                row = {"status": r.group(1), "path": r.group(2), "op": r.group(4),
                       "removed": int(r.group(5)), "added": int(r.group(6))}
                if re.fullmatch(r"\d+", addr):
                    row["a"] = row["b"] = int(addr)
                elif re.fullmatch(r"\d+-\d+", addr):
                    row["a"], row["b"] = map(int, addr.split("-"))
                elif addr == "-":
                    row["a"] = row["b"] = 0
                else:
                    row["unplaced"] = f"address {addr!r} is not a line number"
                rows.append(row)
                continue
            t = TERSE.match(line)
            if t:
                cur = None
                terse.append((t.group(2), int(t.group(3))))
                continue
            w = WROTE.match(line)
            if w:
                cur = None
                wrote[w.group(1)] = int(w.group(3))
                continue
            sm = SUMMARY.match(line)
            if sm:
                cur = None
                items.append(("receipt", ([x for x in rows if x["status"] in ("ok", "dry")], sm.group(2), terse, wrote)))
                rows, terse, wrote = [], [], {}
        return items

    def read(self, path, nl, spans, printed, uncertain):
        f = self.file(path)
        f.lines = nl
        if any(not set(range(a, b + 1)) <= printed for a, b in spans):
            uncertain = True  # the output ends inside a span it headed: cut by a pipe or the host
        if uncertain:
            f.uncertain = True
            return
        if printed >= set(range(1, nl + 1)):
            f.narrow = f.today = WHOLE
            f.uncertain = False
            return
        for attr in ("narrow", "today"):
            if getattr(f, attr) != WHOLE:
                setattr(f, attr, getattr(f, attr) | printed)

    def unplaced(self, path, line, op, reason):
        self.file(path).uncertain = True
        self.outcomes.append(("U", path, line, op, reason))

    def write(self, mode, rows, terse, wrote, partial, created_by_flag):
        for path, n in terse:
            for _ in range(n):
                self.unplaced(path, None, "terse", "a --terse receipt names no hunk address")
        by_file = {}
        for row in rows:
            by_file.setdefault(row["path"], []).append(row)
        for path, hunks in by_file.items():
            f = self.file(path)
            created = any(h["op"] == "create" and "unplaced" not in h for h in hunks) or created_by_flag
            if created and not partial and mode != "elsewhere":
                f.uncertain = False  # a create resets the file: every line in it is the caller's
            before_uncertain, lost = f.uncertain, False
            for h in hunks:
                if mode == "elsewhere" or "unplaced" in h or partial or h["op"] == "rename":
                    why = h.get("unplaced") or ("a partial commit" if partial else
                                                "a rename" if h["op"] == "rename" else "a write under another --root")
                    self.unplaced(path, h.get("a"), h["op"], why)
                    continue
                if before_uncertain:
                    self.outcomes.append(("U", path, h["a"], h["op"], "the file is UNCERTAIN"))
                    continue
                outcome = self.judge(mode, f, h, created)
                lost = lost or outcome[0] == "U"
                self.outcomes.append(outcome)
            if lost:
                f.uncertain = True  # a U the judge returned leaves the file UNCERTAIN, as the criterion says
            lines = wrote.get(path, f.lines)
            if f.uncertain:
                f.lines = lines
                continue
            if created:
                f.narrow = WHOLE
            elif mode == "force":
                f.narrow = carry(set(), hunks, lines)  # Decision 3: a forced write carries nothing
            else:
                f.narrow = carry(f.narrow, hunks, lines)
            f.today = WHOLE
            f.lines = lines

    @staticmethod
    def judge(mode, f, h, created):
        path, line, op = h.get("path"), h["a"], h["op"]
        if created or op == "create" or mode == "sub":
            return ("ok", path, line, op, "")
        need, after = needed_lines(op, h["a"], h["b"], f.lines)
        if mode in ("sr", "force"):
            need = set()
        if mode == "sr" and after is not None:
            # The line after is the quoted kept line, or the ledger's (ADR-190 Decision 2). Unlicensed
            # under v1.83.0's rule too, yet landed: the quote kept it, and still would. Licensed only
            # by v1.83.0's whole-file licence: the replay cannot tell which, so U.
            if not File.covers(f.narrow, {after}) and File.covers(f.today, {after}):
                return ("U", path, line, op, "a multi-line SEARCH/REPLACE hunk whose line after only the whole-file licence covered")
            after = None
        if need is None:  # a path op needs the whole file
            narrow_ok, today_ok = f.narrow == WHOLE, f.today == WHOLE
        else:
            full = need | ({after} if after else set())
            narrow_ok, today_ok = File.covers(f.narrow, full), File.covers(f.today, full)
        if narrow_ok:
            return ("ok", path, line, op, "")
        if not today_ok:
            return ("U", path, line, op, "the replay lost a licence: refused under v1.83.0's rule too, yet it landed")
        return ("R", path, line, op, "outside the caller's carried licence and the lines written")


def transcript_events(path):
    """(command, result text) for every Bash call whose command names mrw, in transcript order."""
    calls, order, results = {}, [], {}
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            try:
                d = json.loads(line)
            except ValueError:
                continue
            content = (d.get("message") or {}).get("content")
            if not isinstance(content, list):
                continue
            for c in content:
                if not isinstance(c, dict):
                    continue
                if c.get("type") == "tool_use" and c.get("name") == "Bash":
                    cmd = (c.get("input") or {}).get("command", "")
                    if re.search(r"\bmrw\b", cmd):
                        calls[c["id"]] = cmd
                        order.append(c["id"])
                elif c.get("type") == "tool_result" and c.get("tool_use_id") in calls:
                    body = c.get("content")
                    if isinstance(body, list):
                        body = "".join(x.get("text", "") for x in body if isinstance(x, dict))
                    results[c["tool_use_id"]] = body or ""
    return [(calls[i], results.get(i, "")) for i in order]


def corpus_dir(root):
    with open(os.path.join(root, "docs/bench/stage-8/RESULTS.md"), encoding="utf-8") as fh:
        for line in fh:
            if line.startswith("transcripts:"):
                return line.split(":", 1)[1].strip()
    raise SystemExit("docs/bench/stage-8/RESULTS.md names no transcripts: line")


def manifest(directory):
    rows = []
    for p in sorted(glob.glob(os.path.join(directory, "*", "*.transcript.jsonl"))):
        with open(p, "rb") as fh:
            rows.append(f"{hashlib.sha256(fh.read()).hexdigest()}  {os.path.relpath(p, directory)}")
    return rows


def run(directory):
    total = {"W": 0, "R": 0, "U": 0}
    cells, hunks = {}, []
    for p in sorted(glob.glob(os.path.join(directory, "*", "*.transcript.jsonl"))):
        rel = os.path.relpath(p, directory)
        model, stem = os.path.dirname(rel), os.path.basename(rel)
        cell = f"{model} {stem.split('.')[1]}"
        rp = Replay()
        for cmd, result in transcript_events(p):
            rp.event(cmd, result)
        c = cells.setdefault(cell, {"W": 0, "R": 0, "U": 0})
        for kind, path, line, op, reason in rp.outcomes:
            for d in (c, total):
                d["W"] += 1
                if kind in ("R", "U"):
                    d[kind] += 1
            if kind in ("R", "U"):
                hunks.append({"attempt": stem.removesuffix(".transcript.jsonl"), "kind": kind, "file": path,
                              "line": line, "op": op, "reason": reason})
    return total, cells, hunks


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--corpus")
    ap.add_argument("--out", default="docs/bench/licence194")
    ap.add_argument("--verify")
    a = ap.parse_args(argv)
    root = os.getcwd()
    if a.verify:
        with open(a.verify, encoding="utf-8") as fh:
            got = json.load(fh)
        directory = got["corpus"]["dir"]
        with open(os.path.join(os.path.dirname(a.verify), "manifest.txt"), encoding="utf-8") as fh:
            want_manifest = fh.read().splitlines()
        if manifest(directory) != want_manifest:
            print("verify: the corpus differs from manifest.txt (a file changed, was added or is missing)")
            return 1
        total, _, _ = run(directory)
        now = dict(total, verdict=verdict(total["W"], total["R"], total["U"]))
        old = {k: got[k] for k in ("W", "R", "U", "verdict")}
        if now != old:
            print(f"verify: the replay gives {now}, the committed result says {old}")
            return 1
        print(f"verify: corpus and result agree, {now}")
        return 0
    directory = a.corpus or corpus_dir(root)
    rows = manifest(directory)
    total, cells, hunks = run(directory)
    v = verdict(total["W"], total["R"], total["U"])
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "manifest.txt"), "w", encoding="utf-8") as fh:
        fh.write("\n".join(rows) + "\n")
    msha = hashlib.sha256(("\n".join(rows) + "\n").encode()).hexdigest()
    out = dict(total, verdict=v, corpus={"dir": directory, "files": len(rows), "manifest_sha256": msha},
               cells=cells, hunks=hunks)
    with open(os.path.join(a.out, "replay.json"), "w", encoding="utf-8") as fh:
        json.dump(out, fh, indent=1, sort_keys=True)
        fh.write("\n")
    line = f"verdict: {v} W={total['W']} R={total['R']} U={total['U']}"
    with open(os.path.join(a.out, "REPLAY.md"), "w", encoding="utf-8") as fh:
        fh.write(line + "\n\n")
        fh.write("ADR-194 T1: the narrow licence replayed on the Stage 8 pilot transcripts, against the criterion\n"
                 "pre-registered in docs/adr/BACKLOG.md (From ADR-194). Script: scripts/licence194_replay.py;\n"
                 f"corpus: {len(rows)} transcripts, manifest sha256 {msha}.\n\n")
        fh.write("| cell | W | R | U |\n|------|---|---|---|\n")
        for cell in sorted(cells):
            c = cells[cell]
            fh.write(f"| {cell} | {c['W']} | {c['R']} | {c['U']} |\n")
        fh.write("\n| attempt | kind | file | line | op | reason |\n|---|---|---|---|---|---|\n")
        for h in hunks:
            fh.write(f"| {h['attempt']} | {h['kind']} | {h['file']} | {h['line']} | {h['op']} | {h['reason']} |\n")
    print(line)
    return 0


if __name__ == "__main__":
    sys.exit(main())
