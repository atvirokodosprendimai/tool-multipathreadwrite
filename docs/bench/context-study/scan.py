"""context-study scanner: should mrw keep a session working-memory map?

Three measurements over recorded agent sessions, per host (Claude Code, Codex, Cursor IDE):
  M1  where output tokens go (prose, write bodies, commands, other tool input, reasoning)
  M2  read redundancy split at context compactions (within-epoch vs cross-epoch)
  M3  subagent reads of lines the parent had already been served, and sibling overlap

Stdlib only (zstd: the zstd CLI or the zstandard module, optional). Runs unmodified on
macOS, Linux and Windows. Publishes aggregates only: no paths, contents, project names or ids.

Run: python3 -I scan.py [--hosts claude,codex,cursor] [--limit N] [--out results.json] [--examples]
README.md beside this file gives the method and its limits.
"""
import argparse
import bisect
import collections
import datetime
import hashlib
import importlib.util
import io
import json
import os
import random
import re
import shutil
import sqlite3
import subprocess
import sys
import time
from pathlib import Path

sys.dont_write_bytecode = True                   # no __pycache__ in the tree
HERE = Path(__file__).resolve().parent
_spec = importlib.util.spec_from_file_location("copy_study_scan", HERE.parent / "copy-study" / "scan.py")
cs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cs)                      # copy-study's parsers and redundancy rule, reused unchanged

CATS = ("prose", "thinking_visible", "write_new", "write_old", "commands", "subagent_prompt", "other_tool")
SUBAGENT_TOOLS = {"Agent", "Task", "task_v2", "task", "spawn_agent"}
COMMAND_TOOLS = {"Bash", "Grep", "Glob", "Read", "LS", "exec", "exec_command", "write_stdin", "shell",
                 "run_terminal_command_v2", "run_terminal_cmd", "grep", "ripgrep_raw_search", "glob_file_search",
                 "read_file", "read_file_v2", "list_dir", "list_dir_v2", "codebase_search", "read_lints",
                 "semantic_search_full", "BashOutput", "Monitor"}
TRUNC = re.compile(r"tokens truncated|chars truncated|\[\.\.\. ?\d+ (?:more )?lines")
NUMBERED = re.compile(r"^\s*(\d+)\t(.*)$")        # nl -ba / cat -n output
CODEX_HEAD = re.compile(r'^\{"timestamp":"([^"]*)","ordinal":\d+,"type":"(\w+)"(?:,"payload":\{"type":"(\w+)")?')
JS_CMD = re.compile(r'exec_command\(\{\s*"?cmd"?\s*:\s*"((?:[^"\\]|\\.)*)"')
PATCH_FILE = re.compile(r"^\*\*\* (?:Update|Add|Delete) File: (.+?)\s*$|^\*\*\* Move to: (.+?)\s*$", re.M)


def ts_of(s):
    """Seconds since the epoch from an ISO string or epoch milliseconds; None when absent."""
    if isinstance(s, (int, float)) and not isinstance(s, bool):
        return s / 1000.0 if s > 1e11 else float(s)
    if isinstance(s, str) and s:
        try:
            return datetime.datetime.fromisoformat(s.replace("Z", "+00:00")).timestamp()
        except ValueError:
            return None
    return None


def jload(x):
    if isinstance(x, (dict, list)):
        return x
    if isinstance(x, (str, bytes)) and x:
        try:
            return json.loads(x)
        except ValueError:
            return None
    return None


# ------------------------------------------------------------- the model

class Sess:
    """One transcript, normalised: events in order and typed output by category.

    events: ("serve", ts, path, n, text, strict), ("write", ts, path), ("compact", ts, tool calls so far).
    strict marks a serve copy-study's parsers also see (numbered Read output, mrw output).
    """

    def __init__(self, host, sid, kind):
        self.host, self.sid, self.kind = host, sid, kind
        self.parent = None            # sid of the parent session, when known
        self.parent_tool = None       # the parent's tool-call id that spawned this one
        self.start = None             # seconds; when the session started
        self.events = []
        self.out = collections.Counter()
        self.reported_output = 0      # output tokens the host recorded
        self.reported_reasoning = None  # reasoning tokens, where the host records them separately
        self.usage_records = 0
        self.tool_ids = set()
        self.tools = 0
        self.flags = collections.Counter()

    def serve(self, ts, path, n, text, strict):
        if path:
            self.events.append(("serve", ts, path, n, text, strict))

    def write(self, ts, path):
        if path:
            self.events.append(("write", ts, path))

    def compact(self, ts):
        self.events.append(("compact", ts, self.tools))


class Recorder:
    """Duck-types copy-study's Session for its parse functions: every see() becomes a serve event."""

    def __init__(self, sess, ts, strict):
        self.sess, self.ts, self.strict = sess, ts, strict

    def see(self, path, n, text):
        self.sess.serve(self.ts, path, n, text, self.strict)


class Tracker(cs.Session):
    """copy-study's redundancy rule (same file+line+text, no own write to the file between),
    with each serve stamped by its context epoch; a compaction starts a new epoch."""

    def __init__(self):
        super().__init__()
        self.epoch = 0
        self.ep_of = {}
        self.c = collections.Counter()
        self.last = None

    def see(self, path, n, text):
        key = (self.fid(path), n)
        size = len(text) + 1
        self.c["served"] += size
        if self.epoch:
            self.c["served_after_compaction"] += size
        self.last = None
        if self.served.get(key) == text:
            self.c["redundant"] += size
            self.last = "within" if self.ep_of.get(key) == self.epoch else "cross"
            self.c[self.last] += size
        self.served[key] = text
        self.ep_of[key] = self.epoch
        self.served_by_file[key[0]].add(n)

    def compact(self):
        self.epoch += 1


# ------------------------------------------------------- shared parsing

def is_write_heredoc(cmd, body):
    return cs.is_plan(body) or "mrw write" in cmd or "--create" in cmd or bool(cs.CAT_HEREDOC.search(cmd)) \
        or re.search(r"\btee\b", cmd) is not None


def patch_parts(patch):
    """apply_patch text: (new chars, old chars, paths written). '+' lines are new; '-' and ' ' are retyped old text."""
    new = old = 0
    for line in patch.split("\n"):
        if line.startswith("***") or line.startswith("@@"):
            continue
        if line.startswith("+"):
            new += len(line)
        elif line.startswith("-") or line.startswith(" "):
            old += len(line)
    paths = [a or b for a, b in PATCH_FILE.findall(patch)]
    return new, old, paths


def split_command(cmd, files_written):
    """A shell command's typed chars that are file content: (write_new, write_old, paths written)."""
    new = old = 0
    paths = []
    units, _ = cs.bash_writes(cmd, files_written)
    counted = []                                   # bodies already in a write unit: count each once
    for _, p, _, _, body in units:
        new += cs.chars(body)
        paths.append(p)
        counted.append(body)
    patched = False
    for _, _, body in cs.HEREDOC.findall(cmd):
        if "*** Begin Patch" in body:
            n, o, ps = patch_parts(body)
            new, old, paths = new + n, old + o, paths + ps
            patched = True
        elif cs.is_plan(body):
            continue                              # already in units
        elif cs.text_lines(body) in counted:
            counted.remove(cs.text_lines(body))   # an mrw --create body: already in units
        elif is_write_heredoc(cmd, body):
            new += len(body)
    for m in cs.CAT_HEREDOC.finditer(cmd):
        paths.append(m.group(1).strip("'\""))
    if not patched and "apply_patch" in cmd and "*** Begin Patch" in cmd:
        n, o, ps = patch_parts(cmd)
        new, old, paths = new + n, old + o, paths + ps
    return new, old, paths


READ_FORMS = [
    (re.compile(r"^sed -n ['\"]?(\d+)(?:,(\d+))?p['\"]? (\S+)$"), "sed"),
    (re.compile(r"^(?:nl -ba|cat -n) (\S+)(?: \| sed -n ['\"]?(\d+)(?:,(\d+))?p['\"]?)?$"), "numbered"),
    (re.compile(r"^cat (\S+)$"), "whole"),
    (re.compile(r"^head -(?:n )?(\d+) (\S+)$"), "head"),
]
CHAIN = ("&&", ";", "||", "\x24(", "\x60", "\n", ">")


def strip_cmd(cmd):
    cmd = cmd.strip()
    cmd = re.sub(r"^cd \S+ && ", "", cmd)
    cmd = re.sub(r"\s+2>(?:&1|/dev/null)$", "", cmd)
    return cmd.strip()


def infer_read(cmd):
    """A single plain read command: (path, kind, first line) or None. Chains are never inferred."""
    c = strip_cmd(cmd)
    if any(t in c for t in CHAIN) or c.count("|") > 1:
        return None
    for rx, kind in READ_FORMS:
        m = rx.match(c)
        if not m:
            continue
        g = m.groups()
        if kind == "sed":
            return (g[2].strip("'\""), "sed", int(g[0]))
        if kind == "numbered":
            return (g[0].strip("'\""), "numbered", None)
        if kind == "whole":
            return None if g[0].startswith("-") else (g[0].strip("'\""), "sed", 1)
        return (g[1].strip("'\""), "sed", 1)
    return None


READ_LIKE = re.compile(r"^(?:sed -n|nl -ba|cat|head|tail)\b")


SEG_SPLIT = re.compile(r"\s*(?:&&|;|\n)\s*")
SED_RANGE = re.compile(r"^sed -n ['\"]?(\d+)(?:,(\d+))?p['\"]? (\S+)$")
NL_RANGE = re.compile(r"^(?:nl -ba|cat -n) (\S+) \| sed -n ['\"]?(\d+)(?:,(\d+))?p['\"]?$")


def infer_chain(cmd):
    """Ranged reads chained by &&, ; or newlines ("sed -n 'A,Bp' F", "nl -ba F | sed -n 'A,Bp'"):
    [(path, numbered, A, B)], or None when any segment is anything else."""
    c = cmd.strip()
    if any(t in c for t in ("||", "\x24(", "\x60", ">")):
        return None
    segs = []
    for part in SEG_SPLIT.split(c):
        if not part or re.match(r"^cd \S+$", part):
            continue
        m = SED_RANGE.match(part)
        if m:
            segs.append((m.group(3).strip("'\""), False, int(m.group(1)), int(m.group(2) or m.group(1))))
            continue
        m = NL_RANGE.match(part)
        if m:
            segs.append((m.group(1).strip("'\""), True, int(m.group(2)), int(m.group(3) or m.group(2))))
            continue
        return None
    return segs if len(segs) > 1 and len({num for _, num, _, _ in segs}) == 1 else None


def serve_chain(sess, ts, segs, lines):
    """Attribute a chain's output to its segments, or serve nothing when that is ambiguous.

    Numbered output: a segment's lines run A, A+1, ... and may stop early (the file ended); the next
    segment starts at its own A. A line that both continues the current segment and starts a later
    one could belong to either file, so the whole chain is refused. Plain output carries no numbers:
    it is served only when it has exactly the lines the ranges ask for. Truncated output: nothing."""
    served = 0
    if any(TRUNC.search(line) for line in lines):
        return 0
    if segs[0][1]:
        out, i, prev = [], -1, None
        for line in lines:
            m = NUMBERED.match(line)
            if not m:
                continue
            n = int(m.group(1))
            cont = i >= 0 and n == prev + 1 and n <= segs[i][3]
            starts = [j for j in range(i + 1, len(segs)) if segs[j][2] == n]
            if cont and starts:
                return 0                           # this file or the next one: cannot tell
            if not cont:
                if not starts:
                    return 0                       # matches no segment: the command did not run as parsed
                i = starts[0]
            out.append((segs[i][0], n, m.group(2)))
            prev = n
        for path, n, text in out:
            sess.serve(ts, path, n, text, False)
            served += 1
        return served
    if len(lines) != sum(b - a + 1 for _, _, a, b in segs):
        return 0
    it = iter(lines)
    for path, _, a, b in segs:
        for n in range(a, b + 1):
            sess.serve(ts, path, n, next(it), False)
            served += 1
    return served


def serve_inferred(sess, ts, cmd, text):
    """Serve a plain read's output with the line numbers the command implies (never strict)."""
    r = infer_read(cmd)
    segs = None if r else infer_chain(cmd)
    read_like = bool(READ_LIKE.match(strip_cmd(cmd)))
    sess.flags["read_like_commands"] += read_like   # coverage: how many read-like commands got line numbers
    if not r and not segs:
        return 0
    lines = text.split("\n")
    if lines and lines[-1] == "":
        lines.pop()
    n_served = 0
    if segs:
        n_served = serve_chain(sess, ts, segs, lines)
    else:
        path, kind, first = r
        for i, line in enumerate(lines):
            if TRUNC.search(line):
                break
            if kind == "numbered":
                m = NUMBERED.match(line)
                if not m:
                    continue
                sess.serve(ts, path, int(m.group(1)), m.group(2), False)
            else:
                sess.serve(ts, path, first + i, line, False)
            n_served += 1
    sess.flags["read_like_commands_inferred"] += read_like and n_served > 0
    return n_served


def strip_output_header(text):
    """Codex prefixes command output with Exit code / Wall time / Output: lines."""
    i = text.find("Output:\n")
    if 0 <= i < 400:
        return text[i + len("Output:\n"):]
    return text


def account_tool(sess, name, total, new, old):
    """Split one tool call's typed chars into categories."""
    sess.out["write_new"] += new
    sess.out["write_old"] += old
    rest = max(0, total - new - old)
    if name in SUBAGENT_TOOLS:
        sess.out["subagent_prompt"] += rest
    elif name in COMMAND_TOOLS or name.endswith("mrw_read") or name.endswith("mrw_write"):
        sess.out["commands"] += rest
    else:
        sess.out["other_tool"] += rest


# ------------------------------------------------------------ Claude Code

def claude_trees(root, min_bytes):
    """[(main path or None, [subagent paths])], one per session; transcripts at or under min_bytes skipped."""
    trees = collections.OrderedDict()
    root = Path(root)
    if not root.is_dir():
        return []
    for proj in sorted(p for p in root.iterdir() if p.is_dir()):
        for p in sorted(proj.rglob("*.jsonl")):
            try:
                if p.stat().st_size <= min_bytes:
                    continue
            except OSError:
                continue
            if "subagents" in p.parts:
                key = (proj.name, p.parts[p.parts.index("subagents") - 1])
                trees.setdefault(key, [None, []])[1].append(p)
            else:
                trees.setdefault((proj.name, p.stem), [None, []])[0] = p
    return list(trees.values())


def claude_session(path, kind):
    s = Sess("claude", str(path), kind)
    tools = {}
    files_written = {}
    out_tokens = {}
    seen_text = set()
    meta = path.with_name(path.stem + ".meta.json")
    if kind == "subagent" and meta.exists():
        try:
            s.parent_tool = json.loads(meta.read_text(encoding="utf-8")).get("toolUseId")
        except (OSError, ValueError):
            pass
    with open(path, "r", encoding="utf-8", errors="replace") as fh:
        for line in fh:
            if s.start is not None and '"tool_use"' not in line and '"tool_result"' not in line \
                    and '"assistant"' not in line and "compact_boundary" not in line:
                continue
            try:
                o = json.loads(line)
            except ValueError:
                continue
            if not isinstance(o, dict):
                continue
            ts = ts_of(o.get("timestamp"))
            if s.start is None and ts is not None:
                s.start = ts
            if o.get("type") == "system" and o.get("subtype") == "compact_boundary":
                s.compact(ts)
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
                t = b.get("type")
                if o.get("type") == "assistant" and t in ("text", "thinking"):
                    body = (b.get("text") if t == "text" else b.get("thinking")) or ""
                    k = (msg.get("id"), t, hashlib.sha1(body.encode("utf-8", "replace")).hexdigest())
                    if k not in seen_text:            # a block repeated in the transcript counts once
                        seen_text.add(k)
                        s.out["prose" if t == "text" else "thinking_visible"] += len(body)
                elif t == "tool_use":
                    name, inp = b.get("name") or "", b.get("input") or {}
                    if not isinstance(inp, dict) or b.get("id") in tools:
                        continue
                    tools[b.get("id")] = (name, inp)
                    s.tool_ids.add(b.get("id"))
                    s.tools += 1
                    claude_tool_use(s, ts, name, inp, files_written)
                elif t == "tool_result":
                    name, inp = tools.get(b.get("tool_use_id"), ("", {}))
                    text = cs.result_text(b.get("content"))
                    if not text:
                        continue
                    if name == "Read":
                        tur = o.get("toolUseResult")
                        fp = tur["file"].get("filePath") if isinstance(tur, dict) and \
                            isinstance(tur.get("file"), dict) else None
                        cs.parse_read_result(Recorder(s, ts, True), text, fp or inp.get("file_path"))
                    elif (name == "Bash" and "mrw" in (inp.get("command") or "")) or name.endswith("mrw_read"):
                        cs.parse_mrw_output(Recorder(s, ts, True), text)
                    elif name == "Bash":
                        serve_inferred(s, ts, inp.get("command") or "", text)
    s.reported_output = sum(out_tokens.values())
    s.usage_records = len(out_tokens)
    return s


def claude_tool_use(s, ts, name, inp, files_written):
    total = len(json.dumps(inp, ensure_ascii=False))
    units, _ = cs.write_units(name, inp, files_written)
    for u in units:
        s.write(ts, u[1])
    new = old = 0
    if name == "Bash":
        new, old, paths = split_command(inp.get("command") or "", files_written)
        for p in paths:
            s.write(ts, p)
    elif name in ("Edit", "MultiEdit"):
        for e in (inp.get("edits") if name == "MultiEdit" else [inp]) or []:
            if isinstance(e, dict):
                new += len(e.get("new_string") or "")
                old += len(e.get("old_string") or "")
    elif name == "Write":
        new = len(inp.get("content") or "")
    elif name.endswith("mrw_write"):
        new = sum(cs.chars(u[4]) for u in units)
    account_tool(s, name, total, new, old)


# ------------------------------------------------------------------ Codex

def zstd_lines(path):
    """Decompressed lines of a .zst file: the zstd CLI if present, else the zstandard module."""
    exe = shutil.which("zstd")
    if exe:
        proc = subprocess.Popen([exe, "-dc", str(path)], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
        try:
            for raw in proc.stdout:
                yield raw.decode("utf-8", "replace")
        finally:
            proc.stdout.close()
            proc.kill()
            proc.wait()
        return
    try:
        import zstandard  # noqa: PLC0415 - optional
    except ImportError:
        raise OSError("no zstd decoder: install the zstd CLI or the zstandard module") from None
    with open(path, "rb") as fh:
        yield from io.TextIOWrapper(zstandard.ZstdDecompressor().stream_reader(fh), encoding="utf-8",
                                    errors="replace")


def codex_lines(path):
    if str(path).endswith(".zst"):
        return zstd_lines(path)
    return open(path, "r", encoding="utf-8", errors="replace")


def codex_files(roots):
    out = []
    for r in roots:
        r = Path(r)
        if r.is_dir():
            out += sorted(p for p in r.rglob("rollout-*") if p.name.endswith((".jsonl", ".jsonl.zst")))
    return out


def codex_meta(path):
    """session_meta, the first record of a rollout: (id, parent id or None, label, start seconds)."""
    it = codex_lines(path)
    try:
        for line in it:
            o = jload(line)
            if not isinstance(o, dict) or o.get("type") != "session_meta":
                return None
            pl = o.get("payload") or {}
            src = pl.get("source")
            parent, label = None, "main"
            if isinstance(src, dict) and isinstance(src.get("subagent"), dict):
                sp = src["subagent"].get("thread_spawn") or {}
                parent = pl.get("parent_thread_id") or sp.get("parent_thread_id") or pl.get("forked_from_id")
                label = (pl.get("thread_source") or "subagent") + ("_fork" if pl.get("forked_from_id") else "")
            return pl.get("id"), parent, label, ts_of(pl.get("timestamp") or o.get("timestamp"))
    finally:
        if hasattr(it, "close"):
            it.close()
    return None


def codex_cmds(name, args):
    """Shell commands inside one Codex tool call."""
    if name == "exec":                             # a JS cell calling tools.exec_command({cmd: ...})
        cmds = []
        for m in JS_CMD.finditer(args):
            try:
                cmds.append(json.loads('"' + m.group(1) + '"'))
            except ValueError:
                cmds.append(m.group(1))
        return cmds
    a = jload(args)
    if isinstance(a, dict):
        c = a.get("cmd") or a.get("command")
        if isinstance(c, list):
            c = c[-1] if c and c[0] in ("bash", "zsh", "sh") else " ".join(map(str, c))
        if isinstance(c, str):
            return [c]
    return []


def codex_output_text(out):
    if isinstance(out, str):
        return strip_output_header(out)
    if isinstance(out, list):
        parts = [x.get("text", "") for x in out if isinstance(x, dict)]
        if parts and "Output:" in parts[0] and len(parts[0]) < 400:
            return "".join(parts[1:])
        return strip_output_header("".join(parts))
    if isinstance(out, dict):
        return strip_output_header(out.get("output") or out.get("content") or "")
    return ""


def per_call_texts(out):
    """A JS exec cell's output items after its header, one per text() call, or None."""
    if not isinstance(out, list):
        return None
    parts = [x.get("text", "") for x in out if isinstance(x, dict)]
    if not parts or "Output:" not in parts[0] or len(parts[0]) >= 400:
        return None
    texts = []
    for p in parts[1:]:
        j = jload(p) if p.startswith("{") else None
        texts.append(j["output"] if isinstance(j, dict) and isinstance(j.get("output"), str)
                     else strip_output_header(p))
    return texts


def codex_session(path, meta):
    sid, parent, label, start = meta
    s = Sess("codex", sid, "main" if parent is None else "subagent")
    s.parent, s.start = parent, start
    s.flags["label_" + label] = 1
    calls = {}
    files_written = {}
    last_usage = None
    for line in codex_lines(path):
        m = CODEX_HEAD.match(line)
        if m:
            ts_s, typ, ptyp = m.groups()
        else:
            o = jload(line)
            if not isinstance(o, dict):
                continue
            ts_s, typ = o.get("timestamp"), o.get("type")
            ptyp = (o.get("payload") or {}).get("type") if isinstance(o.get("payload"), dict) else None
        ts = ts_of(ts_s)
        if typ == "compacted" or ptyp == "compaction":
            s.compact(ts)
            continue
        if typ == "event_msg" and ptyp == "token_count":
            o = jload(line) or {}
            info = (o.get("payload") or {}).get("info") or {}
            tu, lu = info.get("total_token_usage"), info.get("last_token_usage") or {}
            if isinstance(tu, dict):
                if last_usage is None:              # a fork or resume starts with the parent's running total
                    base = {f: (tu.get(f) or 0) - (lu.get(f) or 0) if lu else 0
                            for f in ("output_tokens", "reasoning_output_tokens")}
                    s.flags["usage_carried_in"] += base["output_tokens"] > 0
                last_usage = tu
                s.usage_records += 1
            continue
        if typ != "response_item" or ptyp not in ("message", "reasoning", "function_call", "custom_tool_call",
                                                 "function_call_output", "custom_tool_call_output"):
            continue
        o = jload(line)
        if not isinstance(o, dict):
            continue
        pl = o.get("payload") or {}
        if ptyp == "message":
            if pl.get("role") == "assistant":
                for c in pl.get("content") or []:
                    if isinstance(c, dict) and c.get("type") == "output_text":
                        s.out["prose"] += len(c.get("text") or "")
        elif ptyp == "reasoning":
            for c in (pl.get("summary") or []) + (pl.get("content") or []):
                if isinstance(c, dict):
                    s.out["thinking_visible"] += len(c.get("text") or "")
        elif ptyp in ("function_call", "custom_tool_call"):
            name = pl.get("name") or ptyp
            args = pl.get("arguments") if ptyp == "function_call" else pl.get("input")
            args = args if isinstance(args, str) else json.dumps(args or {})
            s.tools += 1
            new = old = 0
            cmds = []
            if name == "apply_patch":
                a = jload(args)
                patch = a["input"] if isinstance(a, dict) and isinstance(a.get("input"), str) else args
                new, old, paths = patch_parts(patch)
                for p in paths:
                    s.write(ts, p)
            else:
                cmds = codex_cmds(name, args)
                for c in cmds:
                    n, o2, paths = split_command(c, files_written)
                    new, old = new + n, old + o2
                    for p in paths:
                        s.write(ts, p)
            account_tool(s, name, len(args), new, old)
            calls[pl.get("call_id")] = cmds
        else:
            cmds = calls.get(pl.get("call_id"), [])
            if not cmds:
                continue
            items = per_call_texts(pl.get("output")) if len(cmds) > 1 else None
            if items is not None and len(items) == len(cmds):     # one output item per exec_command call
                for c, t in zip(cmds, items):
                    if "mrw" in c:
                        cs.parse_mrw_output(Recorder(s, ts, True), t)
                    else:
                        serve_inferred(s, ts, c, t)
                continue
            text = codex_output_text(pl.get("output"))
            if not text:
                continue
            if any("mrw" in c for c in cmds):
                cs.parse_mrw_output(Recorder(s, ts, True), text)
            elif len(cmds) == 1:
                serve_inferred(s, ts, cmds[0], text)
            else:
                s.flags["multi_command_cells_not_inferred"] += 1
    if last_usage:
        s.reported_output = (last_usage.get("output_tokens") or 0) - base["output_tokens"]
        s.reported_reasoning = (last_usage.get("reasoning_output_tokens") or 0) - base["reasoning_output_tokens"]
    return s


# ------------------------------------------------------------- Cursor IDE

def cursor_db_default():
    home = Path.home()
    if sys.platform == "darwin":
        base = home / "Library" / "Application Support"
    elif os.name == "nt":
        base = Path(os.environ.get("APPDATA") or (home / "AppData" / "Roaming"))
    else:
        base = home / ".config"
    return base / "Cursor" / "User" / "globalStorage" / "state.vscdb"


def cursor_connect(path):
    """Read-only and immutable: no lock is taken, nothing is written; Cursor may be running."""
    return sqlite3.connect(Path(path).resolve().as_uri() + "?mode=ro&immutable=1", uri=True)


def cursor_composers(db):
    """{composerId: (parent or None, created seconds)} from composerHeaders, plus every composerData key."""
    out = {}
    try:
        for cid, created, value in db.execute("select composerId, createdAt, value from composerHeaders"):
            info = (jload(value) or {}).get("subagentInfo") or {}
            out[cid] = (info.get("parentComposerId"), ts_of(created))
    except sqlite3.Error:
        pass
    for (key,) in db.execute("select key from cursorDiskKV where key >= 'composerData:' and key < 'composerData;'"):
        out.setdefault(key.split(":", 1)[1], (None, None))
    return out


def cursor_session(db, cid, info):
    parent, created = info
    s = Sess("cursor", cid, "main" if parent is None else "subagent")
    s.parent, s.start = parent, created
    row = db.execute("select value from cursorDiskKV where key = ?", ("composerData:" + cid,)).fetchone()
    cd = jload(row[0]) if row and row[0] else None
    if not isinstance(cd, dict):
        return None
    if parent is None and isinstance(cd.get("subagentInfo"), dict) and cd["subagentInfo"].get("parentComposerId"):
        s.parent, s.kind = cd["subagentInfo"]["parentComposerId"], "subagent"
    if s.start is None:
        s.start = ts_of(cd.get("createdAt"))
    order = {h.get("bubbleId"): i for i, h in enumerate(cd.get("fullConversationHeadersOnly") or [])
             if isinstance(h, dict)}
    bubbles = []
    for _, val in db.execute("select key, value from cursorDiskKV where key >= ? and key < ?",
                             ("bubbleId:%s:" % cid, "bubbleId:%s;" % cid)):
        b = jload(val)
        if not isinstance(b, dict):
            continue
        if order and b.get("bubbleId") not in order:   # not in the conversation: a reverted or stale branch
            s.flags["bubbles_outside_conversation"] += 1
            continue
        bubbles.append((order.get(b.get("bubbleId"), 10 ** 9), ts_of(b.get("createdAt")) or 0, b))
    if not bubbles:
        return None
    bubbles.sort(key=lambda x: (x[0], x[1]))
    files_written = {}
    calls_seen = set()
    last_ts = s.start or 0
    for _, bts, b in bubbles:
        ts = bts or last_ts
        last_ts = ts
        if b.get("conversationSummary"):          # the summary that replaced the context at this point
            s.compact(ts)
        if b.get("cachedConversationSummary"):    # a summary prepared ahead; not counted as a compaction
            s.flags["cached_summaries"] += 1
        if b.get("type") == 2:
            tc = b.get("tokenCount") or {}
            if tc.get("outputTokens"):
                s.reported_output += tc["outputTokens"]
                s.usage_records += 1
            s.out["prose"] += len(b.get("text") or "")
            th = b.get("thinking")
            if isinstance(th, dict):
                s.out["thinking_visible"] += len(th.get("text") or "")
        tf = b.get("toolFormerData")
        if isinstance(tf, dict) and tf.get("name"):
            if tf.get("toolCallId") and tf["toolCallId"] in calls_seen:   # a tool call repeated counts once
                s.flags["repeated_tool_calls"] += 1
                continue
            calls_seen.add(tf.get("toolCallId"))
            cursor_tool(s, ts, tf, files_written)
    return s


def cursor_tool(s, ts, tf, files_written):
    name = tf["name"]
    raw = tf.get("rawArgs")
    params = jload(tf.get("params"))
    params = params if isinstance(params, dict) else {}
    args = jload(raw) if raw else None
    args = args if isinstance(args, dict) else {}
    result = jload(tf.get("result"))
    result = result if isinstance(result, dict) else {}
    s.tools += 1
    if tf.get("toolCallId"):
        s.tool_ids.add(tf["toolCallId"])
    new = old = 0
    typed = raw if isinstance(raw, str) and raw else ""
    path = params.get("relativeWorkspacePath") or params.get("targetFile") or args.get("file_path") \
        or args.get("target_file") or args.get("path")
    if name in ("read_file", "read_file_v2"):
        c = result.get("contents")
        if isinstance(c, str) and c and not c.startswith("Error: "):
            first = params.get("offset") or args.get("offset") or 1    # 1-based: checked against whole-file reads
            lines = c.split("\n")
            if lines and lines[-1] == "":
                lines.pop()
            for i, line in enumerate(lines):
                s.serve(ts, path, int(first) + i, line, False)
    elif name == "edit_file_v2":
        body = params.get("streamingContent") or ""
        typed = typed or body
        new = len(body)
        s.write(ts, path)
    elif name in ("search_replace", "edit_file", "write", "delete_file"):
        new = len(args.get("new_string") or args.get("contents") or args.get("code_edit") or "")
        old = len(args.get("old_string") or "")
        s.write(ts, path)
    elif name == "apply_patch":
        new, old, paths = patch_parts(raw if isinstance(raw, str) else "")
        for p in paths or [path]:
            s.write(ts, p)
    elif name in ("run_terminal_command_v2", "run_terminal_cmd"):
        cmd = params.get("command") or args.get("command") or ""
        typed = typed or cmd
        new, old, paths = split_command(cmd, files_written)
        for p in paths:
            s.write(ts, p)
        outp = result.get("output")
        if isinstance(outp, str) and outp:
            if "mrw" in cmd:
                cs.parse_mrw_output(Recorder(s, ts, True), outp)
            else:
                serve_inferred(s, ts, cmd, outp)
    elif name in ("task_v2", "task"):
        typed = typed or (params.get("prompt") or "") + (params.get("description") or "")
    if not typed:
        typed = json.dumps(params, ensure_ascii=False) if params else ""
        s.flags["tool_chars_from_params"] += 1
    account_tool(s, name, len(typed), new, old)


# ------------------------------------------------------------- measuring

class Host:
    """Aggregates for one host."""

    def __init__(self, name, rng):
        self.name = name
        self.rng = rng
        self.c = collections.defaultdict(collections.Counter)   # kind -> counters
        self.per_session = []
        self.per_sub = []
        self.ex = {"M1": cs.Reservoir(5), "M2": cs.Reservoir(5), "M3": cs.Reservoir(5)}

    def session(self, s):
        k = self.c[s.kind]
        k["sessions"] += 1
        for cat in CATS:
            k["out_" + cat] += s.out[cat]
        k["reported_output"] += s.reported_output
        k["sessions_with_usage"] += bool(s.usage_records)
        if s.reported_reasoning is not None:
            k["reported_reasoning"] += s.reported_reasoning
            k["sessions_with_reasoning_count"] += 1
        elif s.reported_output and s.reported_output < cs.tokens(sum(s.out.values())):
            k["sessions_negative_residual"] += 1
        for f, v in s.flags.items():
            k["flag_" + f] += v
        strict, ext = Tracker(), Tracker()
        compactions = mid = 0
        reread = False
        for ev in s.events:
            if ev[0] == "serve":
                _, ts, path, n, text, is_strict = ev
                ext.see(path, n, text)
                if ext.last:
                    reread = reread or ext.last == "cross"
                    self.ex["M2"].offer(self.rng, {"host": self.name, "kind": s.kind, "class": ext.last,
                                                   "line_chars": len(text) + 1, "epoch": ext.epoch,
                                                   "strict_source": is_strict})
                if is_strict:
                    strict.see(path, n, text)
            elif ev[0] == "write":
                ext.invalidate(ev[2])
                strict.invalidate(ev[2])
            else:
                compactions += 1
                if ev[2] > 0:                     # after the session's first tool call: context was lost
                    mid += 1
                    ext.compact()
                    strict.compact()
        k["compactions"] += compactions
        k["mid_session_compactions"] += mid
        k["sessions_with_compaction"] += bool(mid)
        k["sessions_rereading_after_compaction"] += reread
        for tag, t in (("strict", strict), ("ext", ext)):
            for key, v in t.c.items():
                k[f"{tag}_{key}"] += v
        k["tools"] += s.tools
        self.per_session.append({"within": ext.c["within"], "cross": ext.c["cross"], "served": ext.c["served"]})
        self.ex["M1"].offer(self.rng, {"host": self.name, "kind": s.kind, "reported_output": s.reported_output,
                                       "reasoning_reported": s.reported_reasoning,
                                       "visible_tokens_est": {c: cs.tokens(s.out[c]) for c in CATS if s.out[c]}})

    def tree(self, sessions):
        """M3 over one tree: each subagent against its parent, and against its earlier siblings."""
        table = cs.Session()                       # one path identity across the tree
        hist = {}
        for s in sessions.values():
            serves = collections.defaultdict(list)    # (fid, n) -> [(ts, text)] in time order
            writes = collections.defaultdict(list)    # fid -> [ts]
            for ev in s.events:
                if ev[0] == "serve" and ev[1] is not None:
                    serves[(table.fid(ev[2]), ev[3])].append((ev[1], ev[4]))
                elif ev[0] == "write" and ev[1] is not None:
                    writes[table.fid(ev[2])].append(ev[1])
            for v in serves.values():
                v.sort(key=lambda x: x[0])
            hist[s.sid] = (serves, writes)
        by_parent = collections.defaultdict(list)
        for s in sessions.values():
            if s.kind == "subagent":
                by_parent[s.parent].append(s)
        k = self.c["subagent"]
        for parent_id, subs in by_parent.items():
            subs.sort(key=lambda x: x.start or 0)
            p = sessions.get(parent_id)
            firsts = collections.defaultdict(dict)    # ((fid, n), text) -> {sibling index: its earliest serve ts}
            for j, sub in enumerate(subs):
                for ev in sub.events:
                    if ev[0] == "serve" and ev[1] is not None:
                        kk = ((table.fid(ev[2]), ev[3]), ev[4])
                        t = firsts[kk].get(j)
                        if t is None or ev[1] < t:
                            firsts[kk][j] = ev[1]
            for j, sub in enumerate(subs):
                k["m3_subagents"] += 1
                k["m3_parent_missing"] += p is None
                forked = any(f.endswith("_fork") for f in sub.flags)   # Codex fork: parent history in context
                k["m3_forked"] += forked
                k["m3_nested"] += p is not None and p.kind == "subagent"
                served = parent_hit = sib_hit = first = 0
                own = set()
                for ev in sub.events:
                    if ev[0] != "serve":
                        continue
                    kk = ((table.fid(ev[2]), ev[3]), ev[4])
                    size = len(ev[4]) + 1
                    served += size
                    if kk in own:
                        continue                       # a re-read inside the subagent is M2's
                    own.add(kk)
                    first += size
                    if p is not None and sub.start is not None and parent_had(hist[p.sid], kk[0], kk[1], sub.start):
                        parent_hit += size
                    # chronological, whatever order the siblings started in
                    if ev[1] is not None and any(t < ev[1] for i, t in firsts[kk].items() if i != j):
                        sib_hit += size
                k["m3_served"] += served
                k["m3_first_serves"] += first
                k["m3_parent_savable"] += parent_hit
                k["m3_parent_savable_forked"] += parent_hit if forked else 0
                k["m3_served_forked"] += served if forked else 0
                k["m3_sibling_overlap"] += sib_hit
                k["m3_subagents_with_parent_overlap"] += bool(parent_hit)
                k["m3_subagents_with_sibling_overlap"] += bool(sib_hit)
                self.per_sub.append({"served": served, "parent": parent_hit, "sib": sib_hit})
                if served:
                    self.ex["M3"].offer(self.rng, {"host": self.name, "served_chars": served,
                                                   "first_serve_chars": first, "parent_savable_chars": parent_hit,
                                                   "sibling_overlap_chars": sib_hit, "siblings": len(subs),
                                                   "parent_is_subagent": bool(p is not None and p.kind == "subagent")})

    def summary(self):
        out = {}
        kinds = sorted(self.c)
        for kind in kinds + ["all"]:
            k = collections.Counter()
            for kk in (kinds if kind == "all" else [kind]):
                k.update(self.c[kk])
            out[kind] = view(k)
        out["distribution"] = {
            "top_decile_share_of_within_epoch": top_decile([s["within"] for s in self.per_session if s["served"]]),
            "top_decile_share_of_cross_epoch": top_decile([s["cross"] for s in self.per_session if s["served"]]),
            "top_decile_share_of_parent_savable": top_decile([s["parent"] for s in self.per_sub if s["served"]]),
            "top_decile_share_of_sibling_overlap": top_decile([s["sib"] for s in self.per_sub if s["served"]]),
            "sessions_with_serves": sum(1 for s in self.per_session if s["served"]),
            "subagents_with_serves": sum(1 for s in self.per_sub if s["served"]),
        }
        return out


def parent_had(h, key, text, t):
    """Had the parent been served text at key before t, with no parent write to that file since?"""
    serves, writes = h
    lst = serves.get(key)
    if not lst:
        return False
    i = bisect.bisect_left([x[0] for x in lst], t) - 1
    if i < 0 or lst[i][1] != text:
        return False
    return not any(lst[i][0] < w < t for w in writes.get(key[0], ()))


def share(a, b):
    return round(a / b, 4) if b else None


def top_decile(xs):
    xs = sorted(xs, reverse=True)
    t = sum(xs)
    return round(sum(xs[:max(1, len(xs) // 10)]) / t, 4) if t else None


def view(k):
    vis = sum(k["out_" + c] for c in CATS)
    rep = k["reported_output"]
    vis_tok = cs.tokens(vis)
    m1 = {"sessions": k["sessions"], "sessions_with_usage": k["sessions_with_usage"],
          "reported_output_tokens": rep, "visible_chars": {c: k["out_" + c] for c in CATS},
          "visible_tokens_est": {c: cs.tokens(k["out_" + c]) for c in CATS}, "visible_tokens_est_total": vis_tok}
    if k["sessions_with_reasoning_count"]:
        reasoning = k["reported_reasoning"]
        m1["reasoning_tokens_reported"] = reasoning
        m1["reasoning_source"] = "recorded by the host (reasoning_output_tokens, inside output_tokens)"
    else:
        reasoning = max(0, rep - vis_tok)
        m1["reasoning_tokens_residual"] = rep - vis_tok
        m1["reasoning_source"] = "residual: reported output minus visible chars/4, an estimate"
        m1["sessions_negative_residual"] = k["sessions_negative_residual"]
    m1["reported_output_coverage"] = share(k["sessions_with_usage"], k["sessions"])
    if rep < vis_tok:                              # the host's count is below what is visibly typed: unusable
        m1["reported_output_unusable"] = True
    elif rep:
        m1["share_of_reported_output"] = {
            "write_bodies_new": share(cs.tokens(k["out_write_new"]), rep),
            "write_old_text": share(cs.tokens(k["out_write_old"]), rep),
            "no_retype_ceiling": share(cs.tokens(k["out_write_new"] + k["out_write_old"]), rep),
            "commands": share(cs.tokens(k["out_commands"]), rep),
            "prose": share(cs.tokens(k["out_prose"]), rep),
            "subagent_prompts": share(cs.tokens(k["out_subagent_prompt"]), rep),
            "other_tool_input": share(cs.tokens(k["out_other_tool"]), rep),
            "thinking_visible": share(cs.tokens(k["out_thinking_visible"]), rep),
            "reasoning": share(reasoning, rep),
        }
    m1["share_of_visible"] = {c: share(k["out_" + c], vis) for c in CATS}
    m2 = {}
    for tag in ("strict", "ext"):
        sv = k[f"{tag}_served"]
        m2[tag] = {"served_chars": sv, "served_tokens_est": cs.tokens(sv),
                   "redundant_chars": k[f"{tag}_redundant"], "redundant_share": share(k[f"{tag}_redundant"], sv),
                   "within_epoch_chars": k[f"{tag}_within"], "within_epoch_share": share(k[f"{tag}_within"], sv),
                   "within_epoch_tokens_est": cs.tokens(k[f"{tag}_within"]),
                   "cross_epoch_chars": k[f"{tag}_cross"], "cross_epoch_share": share(k[f"{tag}_cross"], sv),
                   "served_after_a_compaction_chars": k[f"{tag}_served_after_compaction"],
                   "cross_epoch_share_of_post_compaction_serves": share(k[f"{tag}_cross"],
                                                                         k[f"{tag}_served_after_compaction"])}
    m2.update(compaction_records=k["compactions"], mid_session_compactions=k["mid_session_compactions"],
              sessions_with_mid_session_compaction=k["sessions_with_compaction"],
              sessions_rereading_after_compaction=k["sessions_rereading_after_compaction"])
    res = {"M1": m1, "M2": m2, "tool_calls": k["tools"]}
    flags = {f[5:]: v for f, v in k.items() if f.startswith("flag_")}
    if flags:
        res["flags"] = flags
    if k["m3_subagents"]:
        res["M3"] = {
            "subagents": k["m3_subagents"], "forked_subagents": k["m3_forked"], "nested_subagents": k["m3_nested"],
            "parent_missing": k["m3_parent_missing"], "served_chars": k["m3_served"],
            "first_serve_chars": k["m3_first_serves"], "parent_savable_chars": k["m3_parent_savable"],
            "parent_savable_chars_in_forked": k["m3_parent_savable_forked"],
            "served_chars_in_forked": k["m3_served_forked"],
            "parent_savable_tokens_est": cs.tokens(k["m3_parent_savable"]),
            "parent_savable_share_of_served": share(k["m3_parent_savable"], k["m3_served"]),
            "sibling_overlap_chars": k["m3_sibling_overlap"],
            "sibling_overlap_share_of_served": share(k["m3_sibling_overlap"], k["m3_served"]),
            "subagents_with_parent_overlap": k["m3_subagents_with_parent_overlap"],
            "subagents_with_sibling_overlap": k["m3_subagents_with_sibling_overlap"],
        }
    return res


# ---------------------------------------------------------------- drivers

def walk(children, root):
    out, stack = [], [root]
    while stack:
        x = stack.pop()
        out.append(x)
        stack += children.get(x, [])
    return out


def run_claude(host, root, min_bytes, limit, corpus, reconcile):
    trees = claude_trees(root, min_bytes)
    if limit:
        trees = trees[:limit]
    cscan = cs.Scan() if reconcile else None
    rec = collections.Counter()
    for main_path, sub_paths in trees:
        sessions = {}
        main_id = None
        for p, kind in ([(main_path, "main")] if main_path else []) + [(q, "subagent") for q in sub_paths]:
            try:
                s = claude_session(p, kind)
                corpus["bytes"] += p.stat().st_size
                if cscan is not None:              # copy-study's own scanner on today's corpus, for reconciling
                    r = cscan.file(str(p), kind)
                    rec["served_chars"] += r["served_chars"]
                    rec["redundant_chars"] += r["redundant_chars"]
            except OSError:
                corpus["unreadable"] += 1
                continue
            corpus["transcripts_" + kind] += 1
            sessions[s.sid] = s
            if kind == "main":
                main_id = s.sid
        owner = {t: s.sid for s in sessions.values() for t in s.tool_ids}
        for s in sessions.values():
            if s.kind == "subagent":
                s.parent = owner.get(s.parent_tool, main_id)
                corpus["subagents_linked_by_tool_id"] += s.parent_tool in owner
        for s in sessions.values():
            host.session(s)
        host.tree(sessions)
        corpus["trees"] += 1
    return dict(rec, redundant_share=share(rec["redundant_chars"], rec["served_chars"])) if reconcile else None


def run_codex(host, roots, limit, corpus):
    metas = {}
    for p in codex_files(roots):
        try:
            m = codex_meta(p)
        except OSError:
            corpus["unreadable"] += 1
            continue
        if not m or not m[0]:
            corpus["no_session_meta"] += 1
            continue
        metas[m[0]] = (p, m)
    children = collections.defaultdict(list)
    for sid, (_, m) in metas.items():
        children[m[1] if m[1] in metas else None].append(sid)
    corpus["subagents_parent_absent"] = sum(1 for _, m in metas.values() if m[1] and m[1] not in metas)
    tops = sorted(children[None])
    if limit:
        tops = tops[:limit]
    for r in tops:
        sessions = {}
        for sid in walk(children, r):
            p, m = metas[sid]
            try:
                s = codex_session(p, m)
                corpus["bytes"] += p.stat().st_size
            except OSError:
                corpus["unreadable"] += 1
                continue
            corpus["transcripts_" + s.kind] += 1
            corpus["zst" if str(p).endswith(".zst") else "jsonl"] += 1
            sessions[sid] = s
        for s in sessions.values():
            host.session(s)
        host.tree(sessions)
        corpus["trees"] += 1


def run_cursor(host, dbpath, limit, sample, corpus):
    dbpath = Path(dbpath)
    if not dbpath.exists():
        corpus["absent"] = 1
        return
    db = cursor_connect(dbpath)
    try:
        comps = cursor_composers(db)
        children = collections.defaultdict(list)
        for cid, (parent, _) in comps.items():
            children[parent if parent in comps else None].append(cid)
        tops = sorted(children[None])
        if sample < 1:                             # deterministic: by a hash of the root composer id
            tops = [r for r in tops if int(hashlib.sha1(r.encode()).hexdigest()[:8], 16) / 0xFFFFFFFF < sample]
        if limit:
            tops = tops[:limit]
        corpus["composers_listed"] = len(comps)
        corpus["sample_fraction"] = sample
        for r in tops:
            sessions = {}
            for cid in walk(children, r):
                s = cursor_session(db, cid, comps[cid])
                if s is None:
                    corpus["composers_without_bubbles"] += 1
                    continue
                corpus["transcripts_" + s.kind] += 1
                sessions[cid] = s
            for s in sessions.values():
                host.session(s)
            host.tree(sessions)
            corpus["trees"] += 1
    finally:
        db.close()


def main(argv=None):
    home = Path.home()
    ap = argparse.ArgumentParser(description="context-study scanner")
    ap.add_argument("--hosts", default="claude,codex,cursor")
    ap.add_argument("--claude-root", default=str(home / ".claude" / "projects"))
    ap.add_argument("--codex-root", action="append", default=None,
                    help="repeatable; default ~/.codex/sessions and ~/.codex/archived_sessions")
    ap.add_argument("--cursor-db", default=str(cursor_db_default()))
    ap.add_argument("--cursor-sample", type=float, default=1.0, help="fraction of Cursor root composers, by hash")
    ap.add_argument("--limit", type=int, default=0, help="first N session trees per host")
    ap.add_argument("--min-bytes", type=int, default=10 * 1024, help="Claude transcripts at or under this are skipped")
    ap.add_argument("--no-reconcile", action="store_true", help="skip re-running copy-study's scanner on Claude")
    ap.add_argument("--out", default="")
    ap.add_argument("--examples", action="store_true", help="print 5 sampled cases per measurement to stderr")
    a = ap.parse_args(argv)
    codex_roots = a.codex_root or [str(home / ".codex" / "sessions"), str(home / ".codex" / "archived_sessions")]
    rng = random.Random(168)
    res = {"assumptions": {"chars_per_token": 4, "claude_min_bytes": a.min_bytes, "limit_trees_per_host": a.limit},
           "hosts": {}}
    for h in [x.strip() for x in a.hosts.split(",") if x.strip()]:
        t0 = time.time()
        host = Host(h, rng)
        corpus = collections.Counter()
        extra = {}
        if h == "claude":
            rec = run_claude(host, a.claude_root, a.min_bytes, a.limit, corpus, not a.no_reconcile)
            if rec is not None:
                extra["copy_study_scanner_today"] = rec
        elif h == "codex":
            run_codex(host, codex_roots, a.limit, corpus)
            extra["zstd_decoder"] = "zstd CLI" if shutil.which("zstd") else "zstandard module (or none)"
        elif h == "cursor":
            run_cursor(host, a.cursor_db, a.limit, a.cursor_sample, corpus)
        else:
            print("unknown host: " + h, file=sys.stderr)
            continue
        res["hosts"][h] = dict(host.summary(), corpus=dict(corpus, runtime_seconds=round(time.time() - t0, 1)),
                               **extra)
        if a.examples:
            for m, r in host.ex.items():
                for e in r.items:
                    print(m, json.dumps(e, sort_keys=True), file=sys.stderr)
        print("%s done in %.1fs" % (h, time.time() - t0), file=sys.stderr)
    text = json.dumps(res, indent=2, sort_keys=True)
    if a.out:
        with open(a.out, "w", encoding="utf-8") as fh:
            fh.write(text + "\n")
    else:
        print(text)
    return res


if __name__ == "__main__":
    main()
