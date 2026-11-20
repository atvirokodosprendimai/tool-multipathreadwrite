"""ADR-206's counting script (Decision 7), amended by ADR-211, frozen before the first counted attempt.

  bench_stage9.py count  --receipts DIR --transcripts DIR --pilot pilot.md [--classes FILE] [--plans FILE]
  bench_stage9.py report --receipts DIR --counts FILE
  bench_stage9.py spend  --receipts DIR
  bench_stage9.py gate run|canary MODEL ARM --target PATH --pilot pilot.md --gates gates.txt [--plans FILE] [--after "MODEL ARM"]
  bench_stage9.py review --results RESULTS.md --counts counts.json --receipts DIR
  bench_stage9.py project --receipts SMOKE_DIR [--plans FILE]
  bench_stage9.py registration --backlog docs/adr/BACKLOG.md

DIR holds <model>/<arm>/ run directories, arm a, m or n: each has the run's run.json, its <attempt>.receipt.json (directly in
it, or in its side/), and for arm n the <attempt>.kept scratch file; the transcript <attempt>.transcript.jsonl is found by
name under --transcripts/<model>/<arm>. count reads the registered plans, bench/stage-9/plans.json for the pilot and
smoke-plans.json for the smoke (--plans), and refuses (exit 2, the reason on standard error) run directories that are not
exactly those plans, a run.json whose figure or task list is not its plan's, run directories whose run.json differ in anything
but the models, configs, tasks, mrw_version, mrw_sha256, mrw_instructions_sha256, keep_state_file, keep_scratch_file and
max_spend_usd, instruction hashes that are not absent in A, present and equal in M and N and pilot.md's `instructions sha256`,
whose arm M and arm N mrw hashes are equal, and whose hash is not the arm's `m sha256` or `n sha256` in --pilot, which it
requires. A graded receipt must carry the SHA-256 of its transcript, which must be the transcript's, and an init_model that is
pilot.md's `init model <model>`. A receipt below the run directory's own or its side/ is refused: the runner's spend stop does not
sum it. It prints, as JSON, what became of each planned attempt and how each run's spend reconciles to its receipts. report
prints the text report from that JSON, and refuses when a run.json is not the one the counts were made from. spend prints
one line, `stage9 spend usd <number> missing <count>`, the exact decimal sum of the receipts' costs. The script reads; it writes
only to standard output. A refusal in M's transcript counts only from a Bash call that is one simple `mrw write` command
(optionally after `cd DIR &&`, optionally with its plan as a trailing heredoc, `2>&1` the only redirection) whose result is
present, whole and ends in the receipt's summary line; any other call that mentions `mrw write` is unattributable and its output
is never counted. Imports result_text from bench_mechanism.py, which is not edited.
"""
import argparse
import hashlib
import json
import math
import random
import re
import sys
from datetime import datetime, timezone
from decimal import Decimal, localcontext
from pathlib import Path

from bench_mechanism import result_text

ARMS = ("a", "m", "n")
ARM_CONFIGS = {"a": ["A"], "m": ["B"], "n": ["B"]}
# run.json keys that may differ between the run directories; every other key must be equal. The instruction hash is checked on
# its own (check_instructions), not by being equal.
MAY_DIFFER = {"models", "configs", "tasks", "mrw_version", "mrw_sha256", "mrw_instructions_sha256", "keep_state_file", "keep_scratch_file", "max_spend_usd"}
# (first arm, second arm, class): the comparisons the backlog entry reads, the statistic being first over second. M against A on
# every class, M against N on the two classes the control reads (ADR-211 Decision 2).
COMPARISONS = (("m", "a", "single-site"), ("m", "a", "batch"), ("m", "a", "follow-on"), ("m", "a", "control"), ("m", "n", "follow-on"), ("m", "n", "control"))
EXPOSURE_CLASSES = ("follow-on", "control")
CLASS_ORDER = ("single-site", "batch", "follow-on", "control")
PAIRED_MIN = 10
RESAMPLES, SEED, MAX_DROPPED_FRACTION = 4000, 206, 0.05
# ADR-203's refusal, as a FAIL row of the text receipt starts it.
EXPOSED = re.compile(r"line \d+ of [^\n]+? has no recorded showing in this version")
FAIL_ROW = re.compile(r"^FAIL .*$")
# The text receipt's last line: `N hunk(s), M file(s), F failed, A advisories[, H hints] — <what became of the plan>`.
SUMMARY = re.compile(r"^\d+ hunk\(s\), \d+ file\(s\), \d+ failed, \d+ advisor(?:y|ies)(?:, \d+ hints?)? — .+$")
# What a host leaves when it cuts a tool result.
TRUNCATION = re.compile(r"\[output truncated\]|\[truncated\]|\[\d+ (?:lines?|chars?|characters|bytes) (?:truncated|omitted)\]|<persisted-output>|\.\.\. \[[^\]]*truncated[^\]]*\]|Output too large", re.I)
STAGE9_LINE = re.compile(r"^mrw: stage9 log: ", re.M)
OPS = ("replace", "delete", "insert-after", "insert-before")
EVENT_KEYS = {"path", "op", "start", "end", "first_unshown", "sha_before"}
SHA64 = re.compile(r"[0-9a-f]{64}")
# A command that mentions `mrw write` anywhere: bare, quoted, via $MRW, after any flags, or inside a string. A flag takes the value
# after it unless that is `write` itself.
FLAGS = r'''(?:-{1,2}[A-Za-z][\w-]*(?:=[^\s"']+|[\s"'\\]+(?!write\b)[^\s"'-][^\s"']*)?[\s"'\\]*)*'''
MENTION = re.compile(r'''(?:\bmrw\b|\$\{?MRW\}?)[\s"'\\]*''' + FLAGS + r'''write\b''')
# One trailing heredoc that feeds the plan: `<<DELIM`, `<<'DELIM'` or `<<-DELIM` ending the first line (2>&1 may follow it).
HEREDOC = re.compile(r"(?<!<)<<-?[ \t]*(?P<q>['\"]?)(?P<delim>[^\s<'\"]\S*?)(?P=q)(?P<tail>[ \t]*2>&1)?[ \t]*\n(?P<body>(?:.*\n)?)[ \t]*(?P=delim)[ \t]*\n?\Z", re.S)
CD = re.compile(r'''\s*cd\s+[^\s;&|<>$`()'"]+\s*&&\s*''')
# What is left is one invocation: bare, `command`, `env VAR=x`, a path ending in /mrw, or "$MRW", then any flags, then write.
SIMPLE = re.compile(r'''(?:command\s+|env\s+(?:\w+=\S*\s+)*)?(?:"?\$\{?MRW\}?"?|(?:[\w./~+@-]*/)?mrw)\s+(?:-{1,2}[A-Za-z][\w-]*(?:=\S+|\s+(?!write\b)[^\s-]\S*)?\s+)*write\b.*''', re.S)
CHAINING = re.compile(r"[;&|<>`]|\$\(")
# A producer stage in front of `| mrw write -`: printf, echo or cat FILE, and nothing else.
PRODUCER = re.compile(r"(?:printf|echo)(?:\s.*)?|cat\s+\S.*", re.S)
DEFAULT_CLASSES = Path(__file__).resolve().parent.parent / "bench" / "stage-9" / "classes.json"
DEFAULT_PLANS = Path(__file__).resolve().parent.parent / "bench" / "stage-9" / "plans.json"


class Refusal(Exception):
    """The input is not what the registration allows; nothing is counted."""


def read_json(path):
    try:
        return json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError) as err:
        raise Refusal(f"{path}: {err}")


def micros(x):
    """internal/bench's micros(): USD in millionths, rounded half away from zero as Go's math.Round does."""
    return int(math.floor(abs(x) * 1e6 + 0.5)) * (1 if x >= 0 else -1)


# ---- the plans and the run directories --------------------------------------------------------------------------

def load_plans(path):
    """The registered runs: a list of {model, arm, directory, configs, max_spend_usd, tasks}, a probe also marked."""
    doc = read_json(path)
    if not isinstance(doc, list) or not doc:
        raise Refusal(f"{path}: a plans file is a non-empty list of runs")
    seen = set()
    for plan in doc:
        if not isinstance(plan, dict) or any(k not in plan for k in ("model", "arm", "directory", "configs", "max_spend_usd", "tasks")):
            raise Refusal(f"{path}: a run lacks model, arm, directory, configs, max_spend_usd or tasks: {plan!r}")
        if plan["arm"] not in ARMS or plan["configs"] != ARM_CONFIGS[plan["arm"]]:
            raise Refusal(f"{path}: {plan['directory']} runs arm {plan['arm']!r} with configs {plan['configs']!r}")
        if plan["directory"] in seen:
            raise Refusal(f"{path}: {plan['directory']} is planned twice")
        seen.add(plan["directory"])
    return doc


def run_directories(root):
    """{<model>/<name>: path} for every directory two levels under root."""
    found = {}
    for model_dir in sorted(p for p in Path(root).iterdir() if p.is_dir() and not p.name.startswith(".")):
        for d in sorted(p for p in model_dir.iterdir() if p.is_dir() and not p.name.startswith(".")):
            found[f"{model_dir.name}/{d.name}"] = d
    return found


def pilot_line(text, pilot, key, pattern, shape):
    """The one line `key: ...` of pilot.md, matched whole by pattern: its group 1."""
    named = re.findall(rf"^{re.escape(key)}:.*$", text, re.M)
    if len(named) != 1:
        raise Refusal(f"{pilot} names `{key}:` {len(named)} times, not once")
    found = re.fullmatch(rf"{re.escape(key)}: ({pattern})\s*", named[0])
    if not found:
        raise Refusal(f"{pilot}: `{named[0]}` is not `{key}: ` and {shape}")
    return found.group(1)


def load_runs(root, pilot=None, plans=None):
    """{directory name: (directory, run.json, plan)}, after every refusal Decision 7 names. Without plans (spend, report) every
    <model>/<arm> directory of an arm a, m or n is a run."""
    found = run_directories(root)
    if plans is None:
        plans = [{"model": name.split("/")[0], "arm": name.split("/")[1], "directory": name, "configs": ARM_CONFIGS[name.split("/")[1]],
                  "max_spend_usd": None, "tasks": None} for name in found if name.split("/")[1] in ARMS]
    else:
        planned = {p["directory"] for p in plans}
        if extra := sorted(set(found) - planned):
            raise Refusal(f"{root} holds {', '.join(extra)}, which no registered plan names")
        if missing := sorted(planned - set(found)):
            raise Refusal(f"the registered plans name {', '.join(missing)}, which {root} does not hold")
    runs = {p["directory"]: (found[p["directory"]], read_json(found[p["directory"]] / "run.json"), p) for p in plans}
    if not runs:
        raise Refusal(f"no <model>/<arm> run directory under {root}")
    first = sorted(runs)[0]
    base = {k: v for k, v in runs[first][1].items() if k not in MAY_DIFFER}
    for name, (d, run, plan) in sorted(runs.items()):
        model, arm = plan["model"], plan["arm"]
        if run.get("models") != [model]:
            raise Refusal(f"{d}/run.json: models is {run.get('models')!r}, not [{model!r}]")
        if run.get("configs") != ARM_CONFIGS[arm]:
            raise Refusal(f"{d}/run.json: configs is {run.get('configs')!r}, arm {arm} runs {ARM_CONFIGS[arm]!r}")
        if plan["max_spend_usd"] is not None and run.get("max_spend_usd") != plan["max_spend_usd"]:
            raise Refusal(f"{d}/run.json: max_spend_usd is {run.get('max_spend_usd')!r}, the registered plan's is {plan['max_spend_usd']!r}")
        if plan["tasks"] is not None and run.get("tasks") != plan["tasks"]:
            raise Refusal(f"{d}/run.json: tasks are not the registered plan's list of {len(plan['tasks'])}")
        other = {k: v for k, v in run.items() if k not in MAY_DIFFER}
        differing = sorted(k for k in set(base) | set(other) if base.get(k) != other.get(k))
        if differing:
            raise Refusal(f"{d}/run.json differs from {runs[first][0]}/run.json in {', '.join(differing)}")
        if arm != "a" and not run.get("mrw_sha256"):
            raise Refusal(f"{d}/run.json: arm {arm} carries no mrw_sha256")
    hashes = {arm: {run["mrw_sha256"] for (_, run, p) in runs.values() if p["arm"] == arm} for arm in ("m", "n")}
    if hashes["m"] & hashes["n"]:
        raise Refusal("arm M and arm N have the same mrw hash: the two builds are one")
    if pilot is not None:
        text = Path(pilot).read_text(encoding="utf-8")
        for arm in ("m", "n"):
            want = pilot_line(text, pilot, f"{arm} sha256", "[0-9a-f]{64}", "64 hex digits")
            if wrong := sorted(h for h in hashes[arm] if h != want):
                raise Refusal(f"arm {arm}'s run.json hash {wrong[0]} is not the {arm} sha256 {want} in {pilot}")
        instructions = pilot_line(text, pilot, "instructions sha256", "[0-9a-f]{64}", "64 hex digits")
        for name, (d, run, plan) in sorted(runs.items()):
            have = run.get("mrw_instructions_sha256")
            if plan["arm"] == "a" and have is not None:
                raise Refusal(f"{d}/run.json: arm a carries mrw_instructions_sha256 {have}: it is given no instructions")
            if plan["arm"] != "a" and have != instructions:
                raise Refusal(f"{d}/run.json: mrw_instructions_sha256 is {have!r}, the instructions sha256 {instructions} of {pilot}")
    return runs


def run_digest(directory):
    """The sha256 of a run directory's run.json: what binds a counts file to the plan it was made from."""
    return hashlib.sha256((directory / "run.json").read_bytes()).hexdigest()


def run_summary(run):
    return {k: run.get(k) for k in ("mrw_sha256", "mrw_instructions_sha256", "max_spend_usd", "tasks")}


def load_classes(path):
    doc = read_json(path)
    where = {}
    for cls, tasks in doc.items():
        for task in tasks:
            where[task] = cls
    return where


def replaced(exclusion, tried):
    """internal/bench's replaced(): a start error is re-queued up to twice, a setup failure once, nothing else."""
    return tried < 2 if exclusion == "start-error" else tried < 1 if exclusion == "setup" else False


def safe(name):
    """internal/bench's safe(): how a model name appears in an attempt id."""
    return "".join("_" if c in "/ :" else c for c in name)


def receipts_in(directory):
    """({stem: (path, receipt)}, the stems of the .started marks). Only the receipts the runner's spend stop sums: those
    directly in the run directory and directly in its side/. A receipt anywhere deeper is refused, not summed."""
    allowed = {directory, directory / "side"}
    found = {}
    for path in sorted(directory.rglob("*.receipt.json")):
        if path.parent not in allowed:
            raise Refusal(f"{path}: a receipt below {directory} and its side/, which the runner's spend stop does not sum")
        stem = path.name[:-len(".receipt.json")]
        if stem in found:
            raise Refusal(f"two receipts named {stem} under {directory}")
        found[stem] = (path, read_json(path))
    return found, [p.name[:-len(".started")] for p in directory.rglob("*.started")]


def walk(base, task, receipts):
    """((path, receipt) that counts or None, the stems of its chain, the stems of the task's receipts the chain does not reach).

    The producer's rule (internal/bench runner.go): the base id's receipt counts unless its exclusion warrants another try,
    then <id>.r1, then <id>.r2, each the exact id the runner names; every other receipt of the task is stray and counts for
    nothing.
    """
    stem, tried, picked, chain = base, 0, None, []
    while stem in receipts:
        picked = receipts[stem]
        chain.append(stem)
        if not replaced(picked[1].get("exclusion", ""), tried):
            break
        tried += 1
        stem = f"{base}.r{tried}"
    return picked, chain, sorted(s for s, (_, rec) in receipts.items() if s not in chain and (rec.get("task") == task or s.startswith(task + ".")))


def transcripts_by_name(root):
    found = {}
    for path in sorted(Path(root).rglob("*.transcript.jsonl")):
        found.setdefault(path.name, []).append(path)
    return found


# ---- one attempt -----------------------------------------------------------------------------------------------

def outcome_of(rec):
    """correct, outside, other or missing: Decision 7's partition of a planned attempt. An attempt whose bin/mrw changed while
    it ran ran bytes no receipt describes: it is missing."""
    if rec is None or rec.get("exclusion") or rec.get("mrw_sha256_after"):
        return "missing"
    oracle = rec.get("oracle") or {}
    if not rec.get("is_error") and not rec.get("over_budget") and oracle.get("solved"):
        return "correct"
    return "outside" if oracle.get("outside") else "other"


def classify_command(command):
    """None when the command does not mention `mrw write`; "single" when it is one simple write command, whose output is the
    write's; "unattributable" for any other command that mentions it (a chain, a pipe, a substitution, a redirect, text in an
    argument), whose output is never read."""
    if not MENTION.search(command):
        return None
    head = command
    heredoc = HEREDOC.search(command)
    if heredoc:
        if any(line.strip() == heredoc["delim"] for line in heredoc["body"].splitlines()):
            return "unattributable"  # the shell ends the heredoc at that line and runs what follows
        head = command[:heredoc.start()] + (heredoc["tail"] or "")
    head = head.strip()
    cd = CD.match(head)
    rest = head[cd.end():] if cd else head
    # A pipeline's output is its last stage's. A single-quoted argument holds no operator, so it is blanked before the stages are read.
    flat = re.sub(r"'[^']*'", "''", rest)
    *producers, last = flat.split("|")
    simple = SIMPLE.fullmatch(last.strip()) and not CHAINING.search(re.sub(r"(?<!\S)2>&1(?!\S)", "", last))
    if "\n" in flat or not simple or not all(PRODUCER.fullmatch(p.strip()) and not CHAINING.search(p) for p in producers):
        return "unattributable"
    return "single"


def json_receipt(text):
    """The --json receipt a write printed, or None: the first object in the text that has a hunks list."""
    decoder = json.JSONDecoder()
    for m in re.finditer(r"^\{", text, re.M):
        try:
            doc, _ = decoder.raw_decode(text, m.start())
        except ValueError:
            continue
        if isinstance(doc, dict) and isinstance(doc.get("hunks"), list):
            return doc
    return None


def whole_result(text):
    """Whether a write's result is one the parser can read to its end: no host truncation marker anywhere in it, and a receipt
    that ends in its summary line (the lines standard error adds after it, `mrw: ...`, apart) or a complete --json receipt."""
    if not text.strip() or TRUNCATION.search(text):
        return False
    if json_receipt(text) is not None:
        return True
    lines = [ln for ln in text.splitlines() if ln.strip()]
    while lines and lines[-1].startswith("mrw: "):
        lines.pop()
    return bool(lines) and bool(SUMMARY.match(lines[-1].strip()))


def refusals_of(command, text):
    """(ADR-203 refusals in a read write result, their kind): a FAIL row of the text receipt, or a failed hunk of the --json
    receipt, whose reason is ADR-203's. The same words anywhere else, a quoted imitation included, are not one. The kind is
    `dry_run` when the invocation carries --dry-run or the receipt says dry_run, else `lone` when it is the plan's only failed
    hunk, else `beside` another failed hunk."""
    dry = re.search(r"(?<![\w-])--dry-run(?![\w-])", command) is not None
    doc = json_receipt(text)
    if doc is not None:
        failed = [h for h in doc["hunks"] if isinstance(h, dict) and h.get("status") == "failed"]
        n = sum(1 for h in failed if EXPOSED.search(str(h.get("reason", ""))))
        dry = dry or doc.get("dry_run") is True
    else:
        rows = [ln for ln in text.splitlines() if FAIL_ROW.match(ln)]
        failed, n = rows, sum(1 for ln in rows if EXPOSED.search(ln))
    return n, "dry_run" if dry else "lone" if len(failed) == 1 else "beside"


def write_calls(raw):
    """Every Bash call of one transcript that mentions `mrw write`: {kind, command, text, read}. A call is read when it is one
    simple write command whose result is present, whole and unclipped; any other is unattributable and nothing in its result is
    counted."""
    uses, results = [], {}
    for line in raw.splitlines():
        try:
            row = json.loads(line)
        except ValueError:
            continue
        content = (row.get("message") or {}).get("content") if isinstance(row, dict) else None
        for part in content if isinstance(content, list) else []:
            if not isinstance(part, dict):
                continue
            if part.get("type") == "tool_use" and part.get("name") == "Bash":
                uses.append(part)
            elif part.get("type") == "tool_result":
                results[part.get("tool_use_id")] = result_text(part.get("content"))
    calls = []
    for use in uses:
        command = (use.get("input") or {}).get("command", "")
        kind = classify_command(command) if isinstance(command, str) else None
        if kind is None:
            continue
        text = results.get(use.get("id"), "")
        calls.append({"kind": kind, "command": command, "text": text, "read": kind == "single" and whole_result(text)})
    return calls


def m_exposure(calls):
    """The exposure of an attempt in arm M: exposed, not seen or unknown, and what it rests on."""
    refusals = {"dry_run": 0, "lone": 0, "beside": 0}
    for c in calls:
        if c["read"]:
            n, kind = refusals_of(c["command"], c["text"])
            refusals[kind] += n
    hunks = sum(refusals.values())
    unattributable = sum(not c["read"] for c in calls)
    state = "exposed" if hunks else "unknown" if unattributable else "not seen"
    return {"exposure": state, "hunks": hunks, "refusals": refusals, "flagged": bool(hunks and unattributable), "write_calls": len(calls),
            "write_results_read": sum(c["read"] for c in calls), "write_unattributable": unattributable}


def is_event(doc):
    """One line of N's kept log: an object with exactly the six keys, a non-empty path, an op of the four, integer
    1 <= start <= first_unshown <= end, and a sha_before of 64 lower-case hex digits."""
    if not isinstance(doc, dict) or set(doc) != EVENT_KEYS:
        return False
    ints = [doc[k] for k in ("start", "end", "first_unshown")]
    if any(isinstance(v, bool) or not isinstance(v, int) for v in ints):
        return False
    return (isinstance(doc["path"], str) and doc["path"] != "" and doc["op"] in OPS and 1 <= doc["start"] <= doc["first_unshown"] <= doc["end"]
            and isinstance(doc["sha_before"], str) and SHA64.fullmatch(doc["sha_before"]) is not None)


def kept_events(path):
    """(events, lines that are not events) of arm N's kept log; (0, 0) when the attempt left none."""
    if not path.exists():
        return 0, 0
    events = bad = 0
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        if not line.strip():
            continue
        try:
            doc = json.loads(line)
        except ValueError:
            doc = None
        if is_event(doc):
            events += 1
        else:
            bad += 1
    return events, bad


def scratch_note(rec):
    """True when the receipt says the scratch file asked for was not kept (a kept_note beginning `scratch:`)."""
    return any(n.startswith("scratch:") for n in (rec.get("kept_note") or "").split("; "))


def n_exposure(rec, kept, calls):
    """The exposure of an attempt in arm N, from the runner's kept log and what the transcript's write calls say."""
    events, bad = kept_events(kept)
    stage9 = any(STAGE9_LINE.search(c["text"]) for c in calls)
    unattributable = sum(not c["read"] for c in calls)
    doubt = bool(bad or scratch_note(rec) or stage9 or unattributable)
    state = "exposed" if events else "unknown" if doubt else "not seen"
    return {"exposure": state, "hunks": events, "refusals": None, "flagged": bool(events and doubt), "write_calls": len(calls),
            "write_results_read": sum(c["read"] for c in calls), "write_unattributable": unattributable}


def attempt_entry(cls, arm, picked, chain, stray, started, index, transcripts, init_model, run_mrw=""):
    path, rec = picked if picked else (None, None)
    outcome = outcome_of(rec)
    entry = {"class": cls, "attempt": path.name[:-len(".receipt.json")] if path else None, "started": started, "outcome": outcome,
             "claimed": bool(rec) and rec.get("status") == "done", "exposure": None, "exposed": None, "exposure_unknown": False,
             "flagged": False, "hunks": None, "refusals": None, "write_calls": None, "write_results_read": None, "write_unattributable": None,
             "stray": stray, "spend": None, "num_turns": None, "tool_calls": None, "tokens": None, "wall_ms": None,
             "transcript_sha256": None, "mrw_after_differs": bool(rec) and bool(rec.get("mrw_sha256_after")),
             "slipped": bool(rec) and bool(rec.get("slipped_writes"))}
    if rec is None:
        return entry
    usage = rec.get("usage") or {}
    entry.update(spend=rec.get("total_cost_usd"), num_turns=rec.get("num_turns"), tool_calls=rec.get("tool_calls"), wall_ms=rec.get("wall_ms"),
                 tokens={"input": usage.get("input_tokens"), "output": usage.get("output_tokens"),
                         "cache_read": usage.get("cache_read_input_tokens"), "cache_write": usage.get("cache_creation_input_tokens")})
    if outcome == "missing":
        return entry
    if rec.get("init_model") != init_model:
        raise Refusal(f"{entry['attempt']}: init_model is {rec.get('init_model')!r}, the observed {init_model!r} of pilot.md")
    if (rec.get("mrw_sha256") or "") != run_mrw:
        raise Refusal(f"{entry['attempt']}: its receipt's mrw_sha256 is {rec.get('mrw_sha256')!r}, its run's is {run_mrw!r}: the receipt is another run's")
    if not rec.get("transcript_sha256"):
        raise Refusal(f"{entry['attempt']}: a graded receipt with no transcript_sha256")
    found = index.get(entry["attempt"] + ".transcript.jsonl", [])
    if len(found) != 1:
        raise Refusal(f"{entry['attempt']}: {len(found)} transcripts named so under {transcripts}")
    entry["transcript_sha256"] = hashlib.sha256(found[0].read_bytes()).hexdigest()
    if entry["transcript_sha256"] != rec["transcript_sha256"]:
        raise Refusal(f"{entry['attempt']}: its transcript hashes to {entry['transcript_sha256']}, its receipt says {rec['transcript_sha256']}")
    if arm in ("m", "n"):
        calls = write_calls(found[0].read_text(encoding="utf-8", errors="replace"))
        seen = m_exposure(calls) if arm == "m" else n_exposure(rec, path.with_name(entry["attempt"] + ".kept"), calls)
        entry.update(seen)
        entry["exposed"] = {"exposed": True, "not seen": False}.get(seen["exposure"])
        entry["exposure_unknown"] = seen["exposure"] == "unknown"
    return entry


def money(x):
    """An exact decimal USD amount as the receipts' own text would give it."""
    return format(x.normalize(), "f") if x else "0"


def known(cost):
    return Decimal(repr(cost)) if cost is not None else None


def reconcile(directory, found, picked_stems, chain_stems, outcomes, figure):
    """Each receipt the runner's spend stop sums falls in exactly one bucket: terminal graded, terminal excluded (the picked
    receipt is an exclusion, or its bin/mrw changed), superseded (a receipt a replacement followed) or stray (reached by no chain).
    The four sum to the stop's own sum over the directory's receipts and its side/'s, or this refuses."""
    names = ("terminal_graded", "terminal_excluded", "superseded", "stray")
    sums = {n: [Decimal(0), 0, 0] for n in names}  # known USD, missing costs, receipts
    for stem, (_, rec) in found.items():
        if stem in picked_stems:
            name = "terminal_excluded" if outcomes[stem] == "missing" else "terminal_graded"
        elif stem in chain_stems:
            name = "superseded"
        else:
            name = "stray"
        cost = known(rec.get("total_cost_usd"))
        sums[name][0] += cost if cost is not None else 0
        sums[name][1] += cost is None
        sums[name][2] += 1
    stop = [Decimal(0), 0, 0]  # the stop's own listing: *.receipt.json directly in the directory and in side/
    for where in (directory, directory / "side"):
        for path in sorted(where.glob("*.receipt.json")):
            cost = known(read_json(path).get("total_cost_usd"))
            stop[0] += cost if cost is not None else 0
            stop[1] += cost is None
            stop[2] += 1
    total = [sum(sums[n][i] for n in names) for i in range(3)]
    if total != stop:
        raise Refusal(f"{directory}: the receipts' buckets sum to {money(total[0])} USD, {total[1]} missing, {total[2]} receipts; the runner's stop sums {money(stop[0])}, {stop[1]}, {stop[2]}")
    out = {n: {"usd": money(sums[n][0]), "missing": sums[n][1], "receipts": sums[n][2]} for n in names}
    out["total"] = {"usd": money(stop[0]), "missing": stop[1], "receipts": stop[2]}
    # The stop compares in rounded millionths: the sum of the recorded costs as floats, as the runner adds them.
    recorded = 0.0
    for where in (directory, directory / "side"):
        for path in sorted(where.glob("*.receipt.json")):
            cost = read_json(path).get("total_cost_usd")
            recorded += cost if cost is not None else 0.0
    out["figure_usd"] = figure
    out["known_micros"] = micros(recorded)
    out["figure_micros"] = micros(figure) if figure is not None else None
    out["figure_reached"] = figure is not None and micros(recorded) >= micros(figure)
    return out


def count(receipts, transcripts, classes, pilot, plans=None):
    if pilot is None:
        raise Refusal("count needs --pilot: the arms' mrw hashes are checked against pilot.md")
    where = load_classes(classes)
    plan_list = load_plans(plans or DEFAULT_PLANS)
    for plan in plan_list:
        for task in plan["tasks"]:
            if task not in where:
                raise Refusal(f"{plans or DEFAULT_PLANS}: task {task} is in no class of {classes}")
    runs = load_runs(receipts, pilot, plan_list)
    pilot_text = Path(pilot).read_text(encoding="utf-8")
    models = {}
    for name, (directory, run, plan) in sorted(runs.items()):
        if plan.get("probe"):
            continue  # registered and checked above; its attempts are not the stage's
        model, arm = plan["model"], plan["arm"]
        if run.get("repeats") != 1:
            raise Refusal(f"{directory}/run.json: repeats is {run.get('repeats')!r}; the stage runs one")
        init_model = pilot_line(pilot_text, pilot, f"init model {model}", r"\S+(?: \S+)*", "the model its init line names")
        found, marks = receipts_in(directory)
        where_t = Path(transcripts) / model / arm
        index = transcripts_by_name(where_t)
        tasks, picked_stems, chain_stems, outcomes = {}, set(), set(), {}
        for task in run.get("tasks") or []:
            if task not in where:
                raise Refusal(f"{directory}/run.json: task {task} is in no class of {classes}")
            was_started = any(s.startswith(task + ".") for s in [*found, *marks])
            picked, chain, stray = walk(f"{task}.{ARM_CONFIGS[arm][0]}.{safe(model)}.1", task, found)
            tasks[task] = attempt_entry(where[task], arm, picked, chain, stray, was_started, index, where_t, init_model, run.get("mrw_sha256") or "")
            chain_stems.update(chain)
            if picked:
                stem = picked[0].name[:-len(".receipt.json")]
                picked_stems.add(stem)
                outcomes[stem] = tasks[task]["outcome"]
        models.setdefault(model, {})[arm] = {"run": run_summary(run), "run_sha256": run_digest(directory), "tasks": tasks,
                                              "spend": reconcile(directory, found, picked_stems, chain_stems, outcomes, run.get("max_spend_usd"))}
    return {"stage": "stage9", "models": models}


# ---- spend -----------------------------------------------------------------------------------------------------

def spend(receipts):
    """The exact decimal sum of the receipts' costs, read as the JSON's own number text, and how many carry none."""
    runs = load_runs(receipts)
    missing = 0
    with localcontext() as ctx:
        ctx.prec = 200
        total = Decimal(0)
        for _, (directory, _, _) in sorted(runs.items()):
            receipts_in(directory)  # a receipt the stop does not sum is refused here too
            for path in sorted(directory.rglob("*.receipt.json")):
                try:
                    rec = json.loads(path.read_text(encoding="utf-8"), parse_float=Decimal)
                except (OSError, ValueError) as err:
                    raise Refusal(f"{path}: {err}")
                cost = rec.get("total_cost_usd") if isinstance(rec, dict) else None
                if cost is None:
                    missing += 1
                elif isinstance(cost, bool) or not isinstance(cost, (int, Decimal)):
                    raise Refusal(f"{path}: total_cost_usd is {cost!r}, not a number")
                else:
                    total += cost
        return f"stage9 spend usd {format(total.normalize(), 'f')} missing {missing}"


# ---- report ----------------------------------------------------------------------------------------------------

def bootstrap(rows):
    """(low, high, dropped) of first-over-second spend per correct, resampling rows = [(spend x, correct x, spend y, correct y)].

    A resample in which either arm's figure is undefined (no correct) or zero is dropped and counted; the interval is the
    2.5th and 97.5th percentile of the rest, None when every resample was dropped. The seed is fixed.
    """
    rng = random.Random(SEED)
    n, ratios, dropped = len(rows), [], 0
    for _ in range(RESAMPLES):
        pick = [rows[rng.randrange(n)] for _ in range(n)]
        sx, cx = sum(r[0] for r in pick), sum(r[1] for r in pick)
        sy, cy = sum(r[2] for r in pick), sum(r[3] for r in pick)
        if cx == 0 or cy == 0 or sx == 0 or sy == 0:
            dropped += 1
            continue
        ratios.append((sx / cx) / (sy / cy))
    if not ratios:
        return None, None, dropped
    ratios.sort()
    return ratios[int(0.025 * len(ratios))], ratios[min(len(ratios) - 1, math.ceil(0.975 * len(ratios)) - 1)], dropped



def arm_spend(tasks, paired, cls):
    """(paired known USD, paired missing costs, unpaired known USD, unpaired missing costs) of an arm's terminal graded attempts in
    a class: the attempts the comparison pairs, and the others it leaves out. An attempt counted missing is in neither."""
    out = [Decimal(0), 0, Decimal(0), 0]
    for t, e in tasks.items():
        if e["class"] != cls or e["outcome"] == "missing":
            continue
        i = 0 if t in paired else 2
        if e["spend"] is None:
            out[i + 1] += 1
        else:
            out[i] += known(e["spend"])
    return out


def compare(first, second, tasks_x, tasks_y, cls):
    """The lines of one comparison: INSUFFICIENT with its reason, or the measures, then the spend of each arm in the class,
    paired and unpaired, whatever the comparison came to."""
    head = f"{first} vs {second} ({cls}):"
    if tasks_x is None or tasks_y is None:
        return [f"{head} INSUFFICIENT: arm {first if tasks_x is None else second} has no run"]
    px = {t for t, e in tasks_x.items() if e["class"] == cls}
    py = {t for t, e in tasks_y.items() if e["class"] == cls}
    paired = sorted(t for t in px & py if tasks_x[t]["outcome"] != "missing" and tasks_y[t]["outcome"] != "missing")
    planned = max(len(px), len(py))
    lines = []
    for arm, tasks in ((first, tasks_x), (second, tasks_y)):
        pk, pm, uk, um = arm_spend(tasks, set(paired), cls)
        lines.append(f"  spend {arm}: paired {money(pk)} USD ({pm} missing), unpaired {money(uk)} USD ({um} missing), terminal graded in {cls} {money(pk + uk)} USD ({pm + um} missing)")
    if len(paired) < PAIRED_MIN:
        return [f"{head} INSUFFICIENT: {len(paired)} of {planned} tasks paired, under {PAIRED_MIN}", *lines]
    unpriced = sum(e["spend"] is None for t in paired for e in (tasks_x[t], tasks_y[t]))
    if unpriced:
        return [f"{head} INSUFFICIENT: {unpriced} paired cost(s) missing", *lines]
    correct = {}
    for arm, tasks in ((first, tasks_x), (second, tasks_y)):
        correct[arm] = sum(tasks[t]["outcome"] == "correct" for t in paired)
        if not correct[arm]:
            return [f"{head} INSUFFICIENT: no correct attempt in arm {arm}", *lines]
    rows = [(tasks_x[t]["spend"], int(tasks_x[t]["outcome"] == "correct"), tasks_y[t]["spend"], int(tasks_y[t]["outcome"] == "correct")) for t in paired]
    low, high, dropped = bootstrap(rows)
    if dropped > MAX_DROPPED_FRACTION * RESAMPLES:
        return [f"{head} INSUFFICIENT: {dropped} of {RESAMPLES} resamples dropped, over {MAX_DROPPED_FRACTION:.0%}", *lines]
    fx = sum(r[0] for r in rows) / correct[first]
    fy = sum(r[2] for r in rows) / correct[second]
    excludes = "yes" if low > 1 or high < 1 else "no"  # judged on the unrounded ends
    return [(f"{head} paired {len(paired)}; correct {first} {correct[first]} {second} {correct[second]}; "
             f"spend per correct {first} {fx:.6g} {second} {fy:.6g}; ratio {first}/{second} {fx / fy:.6g} "
             f"[95% {low:.6g}, {high:.6g}]; excludes 1: {excludes}; resamples dropped {dropped} of {RESAMPLES}"), *lines]


def report(receipts, counts):
    runs = load_runs(receipts)
    named = {(model, arm) for model, arms in counts["models"].items() for arm in arms}
    present = {(plan["model"], plan["arm"]) for (_, _, plan) in runs.values()}
    if named != present:
        raise Refusal("the counts file and the receipts hold different runs: "
                      + ", ".join(f"{m}/{a} only in the {'receipts' if (m, a) in present else 'counts'}" for m, a in sorted(named ^ present)))
    by_arm = {(plan["model"], plan["arm"]): d for (d, _, plan) in runs.values()}
    for model, arms in counts["models"].items():
        for arm, doc in arms.items():
            if run_digest(by_arm[(model, arm)]) != doc.get("run_sha256"):
                raise Refusal(f"the counts file was not made from the run.json of {model}/{arm} under {receipts}")
    out = []
    for model in sorted(counts["models"]):
        arms = {arm: doc["tasks"] for arm, doc in counts["models"][model].items()}
        out.append(f"model {model}")
        for arm in ARMS:
            stray = sorted(s for e in arms.get(arm, {}).values() for s in e["stray"])
            if stray:
                out.append(f"  stray receipts ignored ({arm}): {', '.join(stray)}")
            changed = sorted(e["attempt"] for e in arms.get(arm, {}).values() if e["mrw_after_differs"])
            if changed:
                out.append(f"  bin/mrw changed during the attempt, so missing, its spend kept in the reconciliation ({arm}): {', '.join(changed)}")
        for arm in ARMS:
            if arm in counts["models"][model]:
                out.extend(spend_lines(arm, counts["models"][model][arm]["spend"]))
        for cls in CLASS_ORDER:
            out.append(f"  class {cls}")
            for arm in ARMS:
                if arm not in arms:
                    continue
                es = [e for e in arms[arm].values() if e["class"] == cls]
                if not es:
                    continue
                by = {o: sum(e["outcome"] == o for e in es) for o in ("correct", "outside", "other", "missing")}
                out.append(f"    {arm}: planned {len(es)} started {sum(e['started'] for e in es)} graded {len(es) - by['missing']} missing {by['missing']}; "
                           f"correct {by['correct']} outside {by['outside']} other {by['other']}; claimed {sum(e['claimed'] for e in es)}; "
                           f"attempts with slipped writes {sum(e['slipped'] for e in es)}")
            for first, second, c in COMPARISONS:
                if c == cls:
                    out.extend("    " + ln for ln in compare(first, second, arms.get(first), arms.get(second), cls))
            if cls in EXPOSURE_CLASSES:
                out.extend(exposure_lines(arms, cls))
        out.append("  One repeat leaves each task one observation per arm: the comparisons are of paired tasks, not of repeated trials.")
    return "\n".join(out) + "\n"


def spend_lines(arm, doc):
    """Where an arm's recorded spend went: each bucket's known USD and its receipts with no cost, and the stop's own total."""
    lines = [f"  spend {arm}: " + "; ".join(f"{name.replace('_', ' ')} {doc[name]['usd']} USD ({doc[name]['missing']} missing, {doc[name]['receipts']} receipts)"
                                           for name in ("terminal_graded", "terminal_excluded", "superseded", "stray", "total"))]
    if doc["figure_usd"] is not None:
        lines.append(f"    the stop's figure {doc['figure_usd']} USD: recorded {doc['known_micros']} millionths against {doc['figure_micros']}, reached: "
                     + ("yes" if doc["figure_reached"] else "no"))
    return lines


def exposure_lines(arms, cls):
    out = []
    for arm in ("m", "n"):
        if arm not in arms:
            out.append(f"    exposure {arm}: no run")
            continue
        es = [e for e in arms[arm].values() if e["class"] == cls and e["outcome"] != "missing"]
        n = {s: sum(e["exposure"] == s for e in es) for s in ("exposed", "not seen", "unknown")}
        line = (f"    exposure {arm}: exposed {n['exposed']}, not seen {n['not seen']}, unknown {n['unknown']} (in no rate's denominator); "
                f"flagged {sum(bool(e['flagged']) for e in es)} (hunks a lower bound); hunks {sum(e['hunks'] or 0 for e in es if e['exposure'] == 'exposed')}")
        if arm == "m":
            r = {k: sum((e["refusals"] or {}).get(k, 0) for e in es) for k in ("dry_run", "lone", "beside")}
            line += (f"; refusals on a dry run {r['dry_run']}, the plan's only failed hunk {r['lone']}, beside another failed hunk {r['beside']}"
                     f"; write results read {sum(e['write_results_read'] or 0 for e in es)} of {sum(e['write_calls'] or 0 for e in es)} write calls "
                     f"({sum(e['write_unattributable'] or 0 for e in es)} unattributable)")
        out.append(line)
    out.append("    unattributable in m: a write call that was not one simple `mrw write` (a chain, a pipe, a redirect, a substitution, text in an "
               "argument), or whose result was missing, empty, cut or without its summary line: nothing in its result is counted")
    out.append("    not seen in n is the absence of a signal, not a measured zero: no signal the runner owns can say capture worked when nothing was logged")
    if "n" in arms:
        mine = [e for e in arms["n"].values() if e["class"] == cls and e["outcome"] == "outside"]
        listed = sorted(e["attempt"] for e in mine if e["exposure"] == "exposed")
        unknown = sorted(e["attempt"] for e in mine if e["exposure"] == "unknown")
        out.append("    n exposed and outside: " + (", ".join(listed) if listed else "none"))
        out.append("    n outside with exposure unknown: " + (", ".join(unknown) if unknown else "none"))
    return out


# ---- the paid steps' gates, the review, the smoke's projection and the registration --------------------------------
# (ADR-211 Decisions 1, 2, 5, 9 and 10). Each reads and prints; none writes.

# What each canary may spend, ADR-211 Decision 9: the Opus canaries $0.75, the Sonnet ones $0.25.
CANARY_FIGURES = {"claude-opus-5-5": 0.75, "claude-sonnet-5-5": 0.25}
# The configurations a canary runs, once each per model: every one must be in its costs and its init models.
CANARY_CONFIGS = ("A", "B")
# Decision 9's order of the steps, one list for the smoke (the canaries, then the runs) and one for the pilot (the runs): a
# step is cleared only after the one before it in its list, and the first of a list after nothing.
_MODELS = ("claude-opus-5-5", "claude-sonnet-5-5")
_RUNS = [("run", m, a) for m in _MODELS for a in ("n", "m", "a")]
SEQUENCES = {"smoke": [("canary", m, a) for m in _MODELS for a in ("n", "m")] + _RUNS, "pilot": _RUNS}
# What a row of RESULTS.md may adjudicate about an exposed attempt, the Stage 9 entry of BACKLOG.md's three.
ADJUDICATIONS = ("an edit to a line the task did not ask for", "no such edit", "unassessable")
CLEARED = re.compile(r"^gate (\S+) (\S+) cleared (\S+)$", re.M)
WITHDRAWN = ("excludes 1 for M against N", "exposed attempt ended `outside`", "fewer than 3 of the 12 follow-on", "at least 3 of the 12 follow-on")
# What the entry's one gate is made of, besides `on the same side of 1`: M against A, on single-site or on batch, on both
# models, the interval excluding 1, and the models disagreeing registering nothing.
REQUIRED = ("M against A", "on single-site or on batch", "on both models", "excludes 1 and lies", "the models are reported as disagreeing and nothing is registered")


def when(text, what):
    """A recorded time, RFC 3339, as an aware datetime."""
    try:
        t = datetime.fromisoformat(str(text).strip().replace("Z", "+00:00"))
    except ValueError:
        raise Refusal(f"{what}: {text!r} is not a time")
    return t if t.tzinfo else t.replace(tzinfo=timezone.utc)


def clearances(gates):
    """{(model, arm): time text} of the lines `gate <model> <arm> cleared <time>` in gates.txt, in the order they were appended."""
    path = Path(gates)
    if not path.exists():
        return {}
    return {(m, a): t for m, a, t in CLEARED.findall(path.read_text(encoding="utf-8"))}


def exit_status(base):
    """The `bench` exit status recorded beside a step, in <base>.exit."""
    path = Path(str(base) + ".exit")
    if not path.exists():
        raise Refusal(f"{path}: no recorded exit status, so the step is not cleared")
    return path.read_text(encoding="utf-8").strip()


def run_start(directory, started):
    """A run step starts at its earliest .started; every receipt, a replacement included, must have one."""
    if not started:
        raise Refusal(f"{directory}: no .started, so the step never began")
    times = [when(p.read_text(encoding="utf-8"), str(p)) for p in sorted(directory.rglob("*.started"))]
    return min(times)


def check_run(directory, plan, pilot_text, pilot, now_micros=None):
    """Everything Decision 9 asks of one run directory before the next step may start; returns its earliest start."""
    model, arm = plan["model"], plan["arm"]
    run = read_json(directory / "run.json")
    for key, want in (("models", [model]), ("configs", plan["configs"]), ("max_spend_usd", plan["max_spend_usd"]), ("tasks", plan["tasks"])):
        if run.get(key) != want:
            raise Refusal(f"{directory}/run.json: {key} is {run.get(key)!r}, the registered plan's is {want!r}")
    found, marks = receipts_in(directory)
    for stem, (path, _) in found.items():
        if stem not in marks:
            raise Refusal(f"{path.name}: a receipt with no .started: the attempt never began under this runner")
    for stem in marks:
        if stem not in found:
            raise Refusal(f"{stem}: started and never given a receipt: the step stopped inside an attempt")
    init_model = pilot_line(pilot_text, pilot, f"init model {model}", r"\S+(?: \S+)*", "the model its init line names")
    chained = set()
    for task in plan["tasks"]:
        base = f"{task}.{ARM_CONFIGS[arm][0]}.{safe(model)}.1"
        picked, chain, _ = walk(base, task, found)
        chained.update(chain)
        if picked is None:
            raise Refusal(f"{directory}: no receipt for planned task {task}")
        rec = picked[1]
        tried = len(chain) - 1
        if replaced(rec.get("exclusion", ""), tried):
            raise Refusal(f"{picked[0].name}: excluded ({rec.get('exclusion')}) and its replacement {base}.r{tried + 1} is owed and missing")
        for stem in chain:  # each receipt of the chain is of this run and this attempt, not another's under the name (the runner's readsBack)
            got = found[stem][1]
            for key, want in (("attempt", stem), ("task", task), ("model", model), ("config", ARM_CONFIGS[arm][0]), ("mrw_sha256", run.get("mrw_sha256") or "")):
                if (got.get(key) or "") != want:
                    raise Refusal(f"{found[stem][0].name}: its {key} is {got.get(key)!r}, the attempt's is {want!r}: it is not the receipt of this attempt")
        if outcome_of(rec) != "missing" and rec.get("init_model") != init_model:
            raise Refusal(f"{picked[0].name}: init_model is {rec.get('init_model')!r}, the observed {init_model!r}")
    if stray := sorted(set(found) - chained):
        raise Refusal(f"{directory}: receipts reached by no planned attempt: {', '.join(stray)}")
    for stem, (path, rec) in found.items():
        for ext, key in ((".diff", "kept_diff_sha256"), (".kept", "kept_state_file_sha256"), (".kept", "kept_scratch_file_sha256")):
            want, kept = rec.get(key), path.with_name(stem + ext)
            if want:
                if not kept.exists() or hashlib.sha256(kept.read_bytes()).hexdigest() != want:
                    raise Refusal(f"{path.name}: its {ext} file is not the one its receipt hashes")
        has = {ext: path.with_name(stem + ext).exists() for ext in (".diff", ".kept")}
        if has[".diff"] and not rec.get("kept_diff_sha256"):
            raise Refusal(f"{path.name}: a .diff its receipt does not name")
        if has[".kept"] and not (rec.get("kept_state_file_sha256") or rec.get("kept_scratch_file_sha256")):
            raise Refusal(f"{path.name}: a .kept its receipt does not name")
    known_sum, missing = 0.0, 0
    for _, (_, rec) in found.items():
        cost = rec.get("total_cost_usd")
        if cost is None:
            missing += 1
        else:
            known_sum += cost
    figure = plan["max_spend_usd"]
    if micros(known_sum) >= micros(figure):
        raise Refusal(f"{directory}: ${known_sum:.6f} is recorded and the figure is ${figure}: the run reached it, in millionths as the runner compares")
    if missing >= 3:
        raise Refusal(f"{directory}: {missing} receipts carry no cost, the runner's own stop")
    return run_start(directory, marks)


def check_canary(path, model, arm, pilot_text, pilot):
    """The canary of one build: it passed, ran under its registered figure with no null cost, and its configurations
    named the model pilot.md observed; returns when it started."""
    doc = read_json(path)
    if doc.get("pass") is not True:
        raise Refusal(f"{path}: the canary did not pass: {doc.get('problems')!r}")
    if doc.get("model") != model:
        raise Refusal(f"{path}: the canary is of model {doc.get('model')!r}, not {model!r}")
    if doc.get("max_spend_usd") != CANARY_FIGURES[model]:
        raise Refusal(f"{path}: max_spend_usd is {doc.get('max_spend_usd')!r}, the registered {CANARY_FIGURES[model]!r}")
    costs = doc.get("total_cost_usd") or {}
    if not costs or any(c is None or isinstance(c, bool) or not isinstance(c, (int, float)) for c in costs.values()):
        raise Refusal(f"{path}: a canary with a null cost, or none, does not say what it spent")
    if missing := [c for c in CANARY_CONFIGS if c not in costs]:
        raise Refusal(f"{path}: no cost for configuration {', '.join(missing)}: a canary runs each configuration once, and the spend is not known without them")
    if micros(sum(costs.values())) >= micros(CANARY_FIGURES[model]):
        raise Refusal(f"{path}: the canary's costs reached its figure")
    init = doc.get("init_models")
    if not isinstance(init, dict) or not init:
        raise Refusal(f"{path}: no init_models: the model its attempts ran is not known")
    if missing := [c for c in CANARY_CONFIGS if c not in init]:
        raise Refusal(f"{path}: no init model for configuration {', '.join(missing)}: the model its attempt ran is not known")
    want = pilot_line(pilot_text, pilot, f"init model {model}", r"\S+(?: \S+)*", "the model its init line names")
    for config, got in sorted(init.items()):
        if got != want:
            raise Refusal(f"{path}: configuration {config} reported {got!r}, the observed {want!r} of pilot.md")
    return when(doc.get("started", ""), f"{path} started")


def gate(kind, model, arm, target, plans, pilot, gates, sequence, after=None, now=None):
    """One step's clearance line, or a refusal. A run's target is its directory; a canary's is its canary-<model>-<arm>.json.
    The step must have exited 0, be the registered one, be the step of `sequence` ("smoke" or "pilot") that follows `after`
    (none for the first), start after its predecessor's recorded clearance and be cleared after it started."""
    pilot_text = Path(pilot).read_text(encoding="utf-8")
    status = exit_status(target if kind == "run" else str(target)[:-len(".json")])
    if status != "0":
        raise Refusal(f"{target}: bench exited {status!r}, not 0: the step stops the stage")
    if kind == "run":
        by_dir = {p["directory"]: p for p in load_plans(plans)}
        plan = by_dir.get(f"{model}/{arm}")
        if plan is None:
            raise Refusal(f"{plans}: no registered run {model}/{arm}")
        start = check_run(Path(target), plan, pilot_text, pilot)
    else:
        start = check_canary(Path(target), model, arm, pilot_text, pilot)
        other = Path(str(target).replace(f"-{arm}.json", f"-{'n' if arm == 'm' else 'm'}.json"))
        if other.exists():
            mine, theirs = read_json(target).get("init_models"), read_json(other).get("init_models")
            if mine != theirs:
                raise Refusal(f"{target} and {other} are canaries of {model} that report different models: {mine!r} against {theirs!r}")
    recorded = clearances(gates)
    label = f"canary-{arm}" if kind == "canary" else arm  # a canary and a run of one arm are two steps of one model
    # The step's place in the registered order: every step after the first needs its predecessor's recorded clearance, and
    # the first needs none; a caller cannot name another predecessor, or none, for a later step.
    order = SEQUENCES[sequence]
    if (kind, model, arm) not in order:
        raise Refusal(f"{target}: {kind} {model} {arm} is no step of the {sequence} order")
    at = order.index((kind, model, arm))
    expected = None if at == 0 else (lambda k, m, a: f"{m} canary-{a}" if k == "canary" else f"{m} {a}")(*order[at - 1])
    if after != expected:
        raise Refusal(f"{target}: its predecessor in the {sequence} order is {expected or 'none'}, not {after or 'none'}")
    if after is not None:
        if after not in {f"{m} {a}" for m, a in recorded}:
            raise Refusal(f"{gates}: its predecessor {after} has no recorded clearance")
        before = when(recorded[tuple(after.split(" "))], f"{gates} clearance of {after}")
        if start < before:
            raise Refusal(f"{target}: the step started at {start.isoformat()}, before {after} cleared at {before.isoformat()}")
    # A clearance already recorded is the one the replay judges by; a new one is the time it is made. Neither may precede
    # the step's own start: a step is cleared after it began.
    time = recorded.get((model, label)) or now or datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    if when(time, f"{gates} clearance of {model} {label}") < start:
        raise Refusal(f"{target}: cleared at {time}, before the step started at {start.isoformat()}")
    return f"gate {model} {label} cleared {time}"


# ---- review ------------------------------------------------------------------------------------------------------

def review_reason(rec):
    """The oracle's and the budget's reason a receipt gives for an attempt: each of the lists the oracle names, then
    over_budget and timed_out; `none` when it gives none."""
    oracle = rec.get("oracle") or {}
    parts = [f"{key}: " + "; ".join(oracle[key]) for key in ("missing", "outside", "structure") if oracle.get(key)]
    parts += [flag for flag in ("over_budget", "timed_out") if rec.get(flag)]
    return " | ".join(parts) or "none"


ROW = re.compile(r"^### (\S+) (\S+) (\S+)$", re.M)


def parse_rows(text):
    """{(model, arm, attempt): block}: RESULTS.md's review rows, each from its heading to the next heading."""
    heads = list(ROW.finditer(text))
    rows = {}
    for i, h in enumerate(heads):
        key = (h.group(1), h.group(2), h.group(3))
        if key in rows:
            raise Refusal(f"RESULTS.md: two rows for {' '.join(key)}")
        rows[key] = text[h.end():heads[i + 1].start() if i + 1 < len(heads) else len(text)]
    return rows


def review(results, counts, receipts):
    """RESULTS.md against counts.json, the receipts and the kept diffs: one row per exposed attempt of M and N, keyed
    `<model> <arm> <attempt>` (M and N share their stems), whatever its outcome; each quotes the attempt's own kept
    .diff byte for byte, or says `unassessable: <reason>` where none was kept, gives the reason its receipt gives, the limits
    that apply and one adjudication; no row for an attempt that is not exposed."""
    text = Path(results).read_text(encoding="utf-8")
    if not text.strip():
        raise Refusal("RESULTS.md is empty: a row is owed for every exposed attempt, and a pilot with none still says so")
    rows = parse_rows(text)
    runs = {(p["model"], p["arm"]): d for (d, _, p) in load_runs(receipts).values()}
    for (model, arm), d in sorted(runs.items()):  # the counts must be of the runs the receipts hold, or an empty one clears them all
        if arm in ("m", "n") and not (counts.get("models", {}).get(model, {}).get(arm) or {}).get("tasks"):
            raise Refusal(f"counts.json holds no {model} {arm}, which {d} does")
    want = set()
    for model, arms in counts["models"].items():
        for arm in ("m", "n"):
            for entry in (arms.get(arm) or {}).get("tasks", {}).values():
                if entry["exposure"] == "exposed":
                    want.add((model, arm, entry["attempt"]))
    for key in sorted(want - set(rows)):
        raise Refusal(f"RESULTS.md has no row for the exposed attempt {' '.join(key)}")
    for key in sorted(set(rows) - want):
        raise Refusal(f"RESULTS.md has a row for {' '.join(key)}, which is not an exposed attempt of M or N")
    for (model, arm, attempt), block in sorted(rows.items()):
        found, _ = receipts_in(runs[(model, arm)])
        rec = found[attempt][1]
        fields = dict(re.findall(r"^(adjudication|reason|limits): (.*)$", block, re.M))
        if fields.get("adjudication") not in ADJUDICATIONS:
            raise Refusal(f"RESULTS.md row {model} {arm} {attempt}: adjudication {fields.get('adjudication')!r} is none of {ADJUDICATIONS}")
        if fields.get("reason") != review_reason(rec):
            raise Refusal(f"RESULTS.md row {model} {arm} {attempt}: reason {fields.get('reason')!r}, its receipt gives {review_reason(rec)!r}")
        if not (fields.get("limits") or "").strip():
            raise Refusal(f"RESULTS.md row {model} {arm} {attempt}: no limits")
        diff = found[attempt][0].with_name(attempt + ".diff")
        if diff.exists():
            quoted = re.search(r"^diff:\n```\n(.*)\n```[ \t]*$", block, re.S | re.M)
            kept = diff.read_text(encoding="utf-8")
            if quoted is None or quoted.group(1) != (kept[:-1] if kept.endswith("\n") else kept):  # one framing newline, nothing else
                raise Refusal(f"RESULTS.md row {model} {arm} {attempt}: it does not quote its attempt's own kept diff byte for byte")
        elif not re.search(r"^diff: unassessable: \S.*$", block, re.M):
            raise Refusal(f"RESULTS.md row {model} {arm} {attempt}: no kept diff, and the row does not say `diff: unassessable: <reason>`")
    return f"review {len(rows)} row(s) for {len(want)} exposed attempt(s)"


# ---- the smoke's projection and the registration -------------------------------------------------------------------

def project(smoke, plans):
    """Each arm's smoke cost per graded attempt times the attempts the registered plan gives it, against its figure: the
    lines, and whether any arm reaches its figure."""
    lines, over = [], False
    if not Path(smoke).is_dir():
        raise Refusal(f"{smoke}: no smoke root")
    if not [p for p in load_plans(plans) if not p.get("probe")]:
        raise Refusal(f"{plans}: no registered run to project")
    for plan in (p for p in load_plans(plans) if not p.get("probe")):
        d = Path(smoke) / plan["directory"]
        if not d.is_dir():
            raise Refusal(f"{smoke}: no smoke run {plan['directory']}")
        found, _ = receipts_in(d)
        costs = [rec.get("total_cost_usd") for _, rec in found.values() if outcome_of(rec) != "missing"]
        if not costs or any(c is None for c in costs):
            raise Refusal(f"{d}: no graded attempt, or one with no cost: the arm cannot be projected")
        projected = sum(costs) / len(costs) * len(plan["tasks"])
        reached = micros(projected) >= micros(plan["max_spend_usd"])
        over = over or reached
        lines.append(f"project {plan['model']} {plan['arm']} {projected:.4f} of {plan['max_spend_usd']}" + (" REACHES THE FIGURE" if reached else ""))
    return lines, over


ENTRY = re.compile(r"^## Pre-registered: cost per correctly completed edit task.*?(?=^## |\Z)", re.S | re.M)


def registration(backlog):
    """Reads the Stage 9 entry of the BACKLOG, from its heading to the next `## `, and nothing else: it holds the M-against-A
    gate on the same side of 1 and none of the conditions Decision 2 withdraws."""
    m = ENTRY.search(Path(backlog).read_text(encoding="utf-8"))
    if m is None:
        raise Refusal(f"{backlog}: no Stage 9 entry (`## Pre-registered: cost per correctly completed edit task`)")
    entry = " ".join(m.group(0).split())  # a phrase may wrap across lines of the file
    if "on the same side of 1" not in entry:
        raise Refusal("the Stage 9 entry holds no gate for M against A `on the same side of 1` on both models")
    for text in REQUIRED:
        if text not in entry:
            raise Refusal(f"the Stage 9 entry lacks {text!r}, which the one gate is made of")
    for text in WITHDRAWN:
        if text in entry:
            raise Refusal(f"the Stage 9 entry still holds a withdrawn condition: {text!r}")
    return "registration: one gate per direction, none withdrawn"


# ---- the command line ------------------------------------------------------------------------------------------

def main(argv):
    parser = argparse.ArgumentParser(prog="bench_stage9.py", description=__doc__.split("\n\n")[0])
    sub = parser.add_subparsers(dest="verb", required=True)
    c = sub.add_parser("count")
    c.add_argument("--receipts", required=True)
    c.add_argument("--transcripts", required=True)
    c.add_argument("--classes", default=str(DEFAULT_CLASSES))
    c.add_argument("--plans", default=str(DEFAULT_PLANS))
    c.add_argument("--pilot")
    r = sub.add_parser("report")
    r.add_argument("--receipts", required=True)
    r.add_argument("--counts", required=True)
    s = sub.add_parser("spend")
    s.add_argument("--receipts", required=True)
    g = sub.add_parser("gate")
    g.add_argument("kind", choices=("run", "canary"))
    g.add_argument("model")
    g.add_argument("arm", choices=("a", "m", "n"))
    g.add_argument("--target", required=True)
    g.add_argument("--plans", default=str(DEFAULT_PLANS))
    g.add_argument("--pilot", required=True)
    g.add_argument("--gates", required=True)
    g.add_argument("--after")
    g.add_argument("--sequence", required=True, choices=sorted(SEQUENCES))
    rv = sub.add_parser("review")
    rv.add_argument("--results", required=True)
    rv.add_argument("--counts", required=True)
    rv.add_argument("--receipts", required=True)
    pj = sub.add_parser("project")
    pj.add_argument("--receipts", required=True)
    pj.add_argument("--plans", default=str(DEFAULT_PLANS))
    rg = sub.add_parser("registration")
    rg.add_argument("--backlog", required=True)
    args = parser.parse_args(argv)
    try:
        if args.verb == "count":
            sys.stdout.write(json.dumps(count(args.receipts, args.transcripts, args.classes, args.pilot, args.plans), indent=1, sort_keys=True) + "\n")
        elif args.verb == "report":
            sys.stdout.write(report(args.receipts, read_json(args.counts)))
        elif args.verb == "gate":
            sys.stdout.write(gate(args.kind, args.model, args.arm, args.target, args.plans, args.pilot, args.gates, args.sequence, args.after) + "\n")
        elif args.verb == "review":
            sys.stdout.write(review(args.results, read_json(args.counts), args.receipts) + "\n")
        elif args.verb == "project":
            lines, over = project(args.receipts, args.plans)
            sys.stdout.write("\n".join(lines) + "\n")
            return 1 if over else 0
        elif args.verb == "registration":
            sys.stdout.write(registration(args.backlog) + "\n")
        else:
            sys.stdout.write(spend(args.receipts) + "\n")
    except Refusal as err:
        sys.stderr.write(f"bench_stage9: refused: {err}\n")
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
