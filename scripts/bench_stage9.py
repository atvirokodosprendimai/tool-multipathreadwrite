"""ADR-206's counting script (Decision 7), frozen before the first counted attempt.

  bench_stage9.py count  --receipts DIR --transcripts DIR [--classes FILE] [--pilot pilot.md]
  bench_stage9.py report --receipts DIR --counts FILE
  bench_stage9.py spend  --receipts DIR

DIR holds <model>/<arm>/ run directories, arm a, m or n: each has the run's run.json, its <attempt>.receipt.json, and
for arm n the <attempt>.kept log; the transcript <attempt>.transcript.jsonl is found by name anywhere below
--transcripts. count refuses (exit 2, the reason on standard error) run directories whose run.json differ in anything
but the models, configs, tasks, mrw_version, mrw_sha256, mrw_instructions_sha256, keep_state_file and max_spend_usd,
whose arm M and arm N mrw hashes are equal, and, with --pilot, whose hash is not the arm's `m sha256` or `n sha256`.
It prints, as JSON, what became of each planned attempt. report prints the text report from that JSON. spend prints
one line, `stage9 spend usd <number> missing <count>`. The script reads; it writes only to standard output.
Imports mrw_calls and result_text from bench_mechanism.py, which is not edited.
"""
import argparse
import hashlib
import json
import math
import random
import re
import sys
from pathlib import Path

from bench_mechanism import mrw_calls, result_text

ARMS = ("a", "m", "n")
ARM_CONFIGS = {"a": ["A"], "m": ["B"], "n": ["B"]}
# run.json keys that may differ between the six directories; every other key must be equal.
MAY_DIFFER = {"models", "configs", "tasks", "mrw_version", "mrw_sha256", "mrw_instructions_sha256", "keep_state_file", "max_spend_usd"}
# (first arm, second arm, class): the comparisons the backlog entry reads, the statistic being first over second.
COMPARISONS = (("m", "n", "follow-on"), ("m", "n", "control"), ("m", "a", "single-site"), ("m", "a", "batch"))
EXPOSURE_CLASSES = ("follow-on", "control")
CLASS_ORDER = ("single-site", "batch", "follow-on", "control")
PAIRED_MIN = 10
RESAMPLES, SEED, MAX_DROPPED_FRACTION = 4000, 206, 0.05
EXPOSED = re.compile(r"line \d+ of [^\n]+? has no recorded showing in this version")
RETRY = re.compile(r"\.r(\d+)$")
DEFAULT_CLASSES = Path(__file__).resolve().parent.parent / "bench" / "stage-9" / "classes.json"


class Refusal(Exception):
    """The input is not what the registration allows; nothing is counted."""


def read_json(path):
    try:
        return json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError) as err:
        raise Refusal(f"{path}: {err}")


# ---- the run directories ---------------------------------------------------------------------------------------

def load_runs(root, pilot=None):
    """{(model, arm): (directory, run.json)}, after every refusal Decision 7 names."""
    runs = {}
    for model_dir in sorted(p for p in Path(root).iterdir() if p.is_dir()):
        for arm in ARMS:
            d = model_dir / arm
            if d.is_dir():
                runs[(model_dir.name, arm)] = (d, read_json(d / "run.json"))
    if not runs:
        raise Refusal(f"no <model>/<arm> run directory under {root}")
    first_key = sorted(runs)[0]
    base = {k: v for k, v in runs[first_key][1].items() if k not in MAY_DIFFER}
    for (model, arm), (d, run) in sorted(runs.items()):
        if run.get("models") != [model]:
            raise Refusal(f"{d}/run.json: models is {run.get('models')!r}, not [{model!r}]")
        if run.get("configs") != ARM_CONFIGS[arm]:
            raise Refusal(f"{d}/run.json: configs is {run.get('configs')!r}, arm {arm} runs {ARM_CONFIGS[arm]!r}")
        other = {k: v for k, v in run.items() if k not in MAY_DIFFER}
        differing = sorted(k for k in set(base) | set(other) if base.get(k) != other.get(k))
        if differing:
            raise Refusal(f"{d}/run.json differs from {runs[first_key][0]}/run.json in {', '.join(differing)}")
        if arm != "a" and not run.get("mrw_sha256"):
            raise Refusal(f"{d}/run.json: arm {arm} carries no mrw_sha256")
    hashes = {arm: {run["mrw_sha256"] for (_, a), (_, run) in runs.items() if a == arm} for arm in ("m", "n")}
    if hashes["m"] & hashes["n"]:
        raise Refusal("arm M and arm N have the same mrw hash: the two builds are one")
    if pilot is not None:
        text = Path(pilot).read_text(encoding="utf-8")
        for arm in ("m", "n"):
            found = re.search(rf"^{arm} sha256: ([0-9a-f]{{64}})\s*$", text, re.M)
            if not found:
                raise Refusal(f"{pilot} names no `{arm} sha256:` line")
            wrong = sorted(h for h in hashes[arm] if h != found.group(1))
            if wrong:
                raise Refusal(f"arm {arm}'s run.json hash {wrong[0]} is not the {arm} sha256 {found.group(1)} in {pilot}")
    return runs


def run_summary(run):
    return {k: run.get(k) for k in ("mrw_sha256", "mrw_instructions_sha256", "max_spend_usd", "tasks")}


def load_classes(path):
    doc = read_json(path)
    where = {}
    for cls, tasks in doc.items():
        for task in tasks:
            where[task] = cls
    return where


def final_receipts(directory):
    """({task: (path, receipt) that counts}, set of tasks started): a replacement supersedes the attempt it replaces."""
    found, started = {}, set()
    for path in sorted(directory.rglob("*.receipt.json")):
        rec = read_json(path)
        stem = path.name[:-len(".receipt.json")]
        task = rec.get("task")
        retry = RETRY.search(stem)
        rank = int(retry.group(1)) if retry else 0
        if task in found and found[task][0] == rank:
            raise Refusal(f"two receipts for {task} at the same try under {directory}")
        if task not in found or found[task][0] < rank:
            found[task] = (rank, path, rec)
        started.add(task)
    marks = [p.name[:-len(".started")] for p in directory.rglob("*.started")]
    return {t: (p, r) for t, (_, p, r) in found.items()}, started, marks


def transcripts_by_name(root):
    found = {}
    for path in sorted(Path(root).rglob("*.transcript.jsonl")):
        found.setdefault(path.name, []).append(path)
    return found


# ---- one attempt -----------------------------------------------------------------------------------------------

def outcome_of(rec):
    """correct, outside, other or missing: Decision 7's partition of a planned attempt."""
    if rec is None or rec.get("exclusion"):
        return "missing"
    oracle = rec.get("oracle") or {}
    if not rec.get("is_error") and not rec.get("over_budget") and oracle.get("solved"):
        return "correct"
    return "outside" if oracle.get("outside") else "other"


def is_write(args):
    """True for an mrw call that, after the global flags -C DIR, --root DIR and --root=DIR, is `write`; a dry run is one."""
    i = 0
    while i < len(args):
        if args[i] in ("-C", "--root"):
            i += 2
        elif args[i].startswith("--root="):
            i += 1
        else:
            break
    return i < len(args) and args[i] == "write"


def transcript_exposure(raw):
    """(write calls, write results read, refusal matches): ADR-203's refusal as read from the output of a write command."""
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
    calls_made = read = hunks = 0
    for use in uses:
        command = (use.get("input") or {}).get("command", "")
        if not isinstance(command, str):
            continue
        calls, _ = mrw_calls(command)
        if calls is None:
            ran_write = "mrw" in command and "write" in command
        else:
            ran_write = any(is_write(c) for c in calls)
        if not ran_write:
            continue
        calls_made += 1
        text = results.get(use.get("id"), "")
        if text.strip():
            read += 1
            hunks += len(EXPOSED.findall(text))
    return calls_made, read, hunks


def kept_hunks(path):
    """The lines of arm N's kept log: one exposed hunk that landed each; 0 when the attempt left none."""
    if not path.exists():
        return 0
    n = 0
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if not line.strip():
            continue
        try:
            ok = isinstance(json.loads(line), dict)
        except ValueError:
            ok = False
        if not ok:
            raise Refusal(f"{path}: line {number} of the kept log is not a JSON object")
        n += 1
    return n


def attempt_entry(task, cls, arm, picked, started, index, transcripts):
    path, rec = picked if picked else (None, None)
    outcome = outcome_of(rec)
    entry = {"class": cls, "attempt": path.name[:-len(".receipt.json")] if path else None, "started": started, "outcome": outcome,
             "claimed": bool(rec) and rec.get("status") == "done", "exposed": None, "hunks": None, "write_calls": None,
             "write_results_read": None, "spend": None, "num_turns": None, "tool_calls": None, "tokens": None, "wall_ms": None,
             "transcript_sha256": None, "mrw_after_differs": bool(rec) and bool(rec.get("mrw_sha256_after"))}
    if rec is None:
        return entry
    usage = rec.get("usage") or {}
    entry.update(spend=rec.get("total_cost_usd"), num_turns=rec.get("num_turns"), tool_calls=rec.get("tool_calls"), wall_ms=rec.get("wall_ms"),
                 tokens={"input": usage.get("input_tokens"), "output": usage.get("output_tokens"),
                         "cache_read": usage.get("cache_read_input_tokens"), "cache_write": usage.get("cache_creation_input_tokens")})
    if outcome == "missing":
        return entry
    found = index.get(entry["attempt"] + ".transcript.jsonl", [])
    if len(found) != 1:
        raise Refusal(f"{entry['attempt']}: {len(found)} transcripts named so under {transcripts}")
    entry["transcript_sha256"] = hashlib.sha256(found[0].read_bytes()).hexdigest()
    if arm == "n":
        entry["hunks"] = kept_hunks(path.with_name(entry["attempt"] + ".kept"))
        entry["exposed"] = entry["hunks"] > 0
    elif arm == "m":
        entry["write_calls"], entry["write_results_read"], entry["hunks"] = transcript_exposure(found[0].read_text(encoding="utf-8", errors="replace"))
        entry["exposed"] = entry["hunks"] > 0
    return entry


def count(receipts, transcripts, classes, pilot=None):
    runs = load_runs(receipts, pilot)
    where = load_classes(classes)
    models = {}
    for (model, arm), (directory, run) in sorted(runs.items()):
        picked, started, marks = final_receipts(directory)
        where_t = Path(transcripts) / model / arm
        index = transcripts_by_name(where_t)
        tasks = {}
        for task in run.get("tasks") or []:
            if task not in where:
                raise Refusal(f"{directory}/run.json: task {task} is in no class of {classes}")
            was_started = task in started or any(m.startswith(task + ".") for m in marks)
            tasks[task] = attempt_entry(task, where[task], arm, picked.get(task), was_started, index, where_t)
        models.setdefault(model, {})[arm] = {"run": run_summary(run), "tasks": tasks}
    return {"stage": "stage9", "models": models}


# ---- spend -----------------------------------------------------------------------------------------------------

def spend(receipts):
    runs = load_runs(receipts)
    costs, missing = [], 0
    for _, (directory, _) in sorted(runs.items()):
        for path in sorted(directory.rglob("*.receipt.json")):
            cost = read_json(path).get("total_cost_usd")
            if cost is None:
                missing += 1
            else:
                costs.append(cost)
    return f"stage9 spend usd {math.fsum(costs):.4f} missing {missing}"


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


def compare(first, second, tasks_x, tasks_y, cls):
    """The text of one comparison: INSUFFICIENT with its reason, or the measures."""
    head = f"{first} vs {second} ({cls}):"
    if tasks_x is None or tasks_y is None:
        return f"{head} INSUFFICIENT: arm {first if tasks_x is None else second} has no run"
    px = {t for t, e in tasks_x.items() if e["class"] == cls}
    py = {t for t, e in tasks_y.items() if e["class"] == cls}
    paired = sorted(t for t in px & py if tasks_x[t]["outcome"] != "missing" and tasks_y[t]["outcome"] != "missing")
    planned = max(len(px), len(py))
    if len(paired) < PAIRED_MIN:
        return f"{head} INSUFFICIENT: {len(paired)} of {planned} tasks paired, under {PAIRED_MIN}"
    unpriced = sum(e["spend"] is None for t in paired for e in (tasks_x[t], tasks_y[t]))
    if unpriced:
        return f"{head} INSUFFICIENT: {unpriced} paired cost(s) missing"
    correct = {}
    for arm, tasks in ((first, tasks_x), (second, tasks_y)):
        correct[arm] = sum(tasks[t]["outcome"] == "correct" for t in paired)
        if not correct[arm]:
            return f"{head} INSUFFICIENT: no correct attempt in arm {arm}"
    rows = [(tasks_x[t]["spend"], int(tasks_x[t]["outcome"] == "correct"), tasks_y[t]["spend"], int(tasks_y[t]["outcome"] == "correct")) for t in paired]
    low, high, dropped = bootstrap(rows)
    if dropped > MAX_DROPPED_FRACTION * RESAMPLES:
        return f"{head} INSUFFICIENT: {dropped} of {RESAMPLES} resamples dropped, over {MAX_DROPPED_FRACTION:.0%}"
    fx = sum(r[0] for r in rows) / correct[first]
    fy = sum(r[2] for r in rows) / correct[second]
    return (f"{head} paired {len(paired)}; correct {first} {correct[first]} {second} {correct[second]}; spend per correct {first} {fx:.4f} {second} {fy:.4f}; "
            f"ratio {first}/{second} {fx / fy:.4f} [95% {low:.4f}, {high:.4f}]; resamples dropped {dropped} of {RESAMPLES}")


def report(receipts, counts):
    runs = load_runs(receipts)
    for model, arms in counts["models"].items():
        for arm, doc in arms.items():
            have = runs.get((model, arm))
            if have is None or run_summary(have[1]) != doc["run"]:
                raise Refusal(f"the counts file does not match the run.json of {model}/{arm} under {receipts}")
    out = []
    for model in sorted(counts["models"]):
        arms = {arm: doc["tasks"] for arm, doc in counts["models"][model].items()}
        out.append(f"model {model}")
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
                           f"correct {by['correct']} outside {by['outside']} other {by['other']}; claimed {sum(e['claimed'] for e in es)}")
            for first, second, c in COMPARISONS:
                if c == cls:
                    out.append("    " + compare(first, second, arms.get(first), arms.get(second), cls))
            if cls in EXPOSURE_CLASSES:
                out.extend(exposure_lines(arms, cls))
    return "\n".join(out) + "\n"


def exposure_lines(arms, cls):
    out = []
    for arm in ("m", "n"):
        if arm not in arms:
            out.append(f"    exposed {arm}: no run")
            continue
        es = [e for e in arms[arm].values() if e["class"] == cls and e["outcome"] != "missing"]
        line = f"    exposed {arm}: {sum(bool(e['exposed']) for e in es)} of {len(es)} graded attempts; hunks {sum(e['hunks'] or 0 for e in es)}"
        if arm == "m":
            line += f"; write results read {sum(e['write_results_read'] or 0 for e in es)} of {sum(e['write_calls'] or 0 for e in es)} write calls"
        out.append(line)
    if "n" in arms:
        listed = sorted(e["attempt"] for e in arms["n"].values() if e["class"] == cls and e["exposed"] and e["outcome"] == "outside")
        out.append("    n exposed and outside: " + (", ".join(listed) if listed else "none"))
    return out


# ---- the command line ------------------------------------------------------------------------------------------

def main(argv):
    parser = argparse.ArgumentParser(prog="bench_stage9.py", description=__doc__.split("\n\n")[0])
    sub = parser.add_subparsers(dest="verb", required=True)
    c = sub.add_parser("count")
    c.add_argument("--receipts", required=True)
    c.add_argument("--transcripts", required=True)
    c.add_argument("--classes", default=str(DEFAULT_CLASSES))
    c.add_argument("--pilot")
    r = sub.add_parser("report")
    r.add_argument("--receipts", required=True)
    r.add_argument("--counts", required=True)
    s = sub.add_parser("spend")
    s.add_argument("--receipts", required=True)
    args = parser.parse_args(argv)
    try:
        if args.verb == "count":
            sys.stdout.write(json.dumps(count(args.receipts, args.transcripts, args.classes, args.pilot), indent=1, sort_keys=True) + "\n")
        elif args.verb == "report":
            sys.stdout.write(report(args.receipts, read_json(args.counts)))
        else:
            sys.stdout.write(spend(args.receipts) + "\n")
    except Refusal as err:
        sys.stderr.write(f"bench_stage9: refused: {err}\n")
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
