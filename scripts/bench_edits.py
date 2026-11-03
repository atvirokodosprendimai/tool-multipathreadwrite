"""ADR-193's counting script and decision rule (Decisions 4 and 5), frozen before the first counted call.

  bench_edits.py count --receipts DIR --transcripts DIR [--configs B,S]
  bench_edits.py verdict --receipts DIR --counts FILE [--transcripts DIR]
  bench_edits.py null --transcripts DIR

count walks the final receipts under --receipts (every *.receipt.json below it, recursively) and prints, per
counted attempt, what its transcript holds, as JSON; the transcript <attempt>.transcript.jsonl is found by name
anywhere below --transcripts. verdict refuses anything but the registered matrix, then prints the measures and a
last line `verdict: SHIP` or `verdict: NO SHIP`; with --transcripts it also recomputes each digest. null redraws
Decision 5's resampling over the B attempts (and their receipts) under --transcripts. Imports mrw_calls and
result_text from bench_mechanism.py, which is not edited (ADR-192 froze it).
"""
import argparse
import hashlib
import json
import random
import re
import sys
from pathlib import Path

from bench_mechanism import mrw_calls, result_text

INSTANCES = [
    "pytest-dev__pytest-5631", "django__django-16642", "sphinx-doc__sphinx-8265", "scikit-learn__scikit-learn-25747",
    "matplotlib__matplotlib-23314", "sympy__sympy-18199", "sympy__sympy-24661", "django__django-11532",
    "django__django-14011", "scikit-learn__scikit-learn-12585", "matplotlib__matplotlib-25775", "django__django-14434",
    "django__django-16116", "django__django-11555", "astropy__astropy-14539", "django__django-11451",
    "django__django-14404", "django__django-13807",
]
MODELS = ["claude-opus-5-5", "claude-sonnet-5-5"]
ARMS = ["B", "S"]
MAX_DROPPED = 2
RESAMPLES, SEED = 20000, 193
BOOTSTRAP, BOOTSTRAP_SEED = 4000, 193

HOOK_BLOCK = "PreToolUse:Bash hook error"
REFUSAL = re.compile(r"plan has \d+ error|search_replace: |apply_patch: |flag provided but not defined"
                     r"|cannot hold a line break|is not relative to the root|plan is empty")
STEM = re.compile(r"^(?P<task>.+)\.(?P<config>[A-Z])\.(?P<model>[^.]+)\.(?P<repeat>\d+)(?P<retry>\.r\d+)?$")


class Refusal(Exception):
    """The input is not what the registration allows; nothing is judged."""


# ---- Decision 4: counting write calls in one transcript -------------------------------------------------------

def is_write(args):
    """True for an mrw call that, after the global flags -C DIR, --root DIR and --root=DIR, is `write` without --dry-run."""
    i = 0
    while i < len(args):
        if args[i] in ("-C", "--root"):
            i += 2
        elif args[i].startswith("--root="):
            i += 1
        else:
            break
    return i < len(args) and args[i] == "write" and "--dry-run" not in args[i + 1:]


def is_search_replace(args):
    return "--format=search_replace" in args or any(a == "--format" and b == "search_replace" for a, b in zip(args, args[1:]))


def outcome(text):
    """blocked, refused, landed or undecided, in Decision 4's order."""
    if HOOK_BLOCK in text:
        return "blocked"
    if "nothing was written" in text.lower() or REFUSAL.search(text):
        return "refused"
    if "— applied" in text or "PARTIALLY APPLIED" in text:
        return "landed"
    if re.search(r"^mrw: ", text, re.M):
        return "refused"
    return "undecided"


def classify(raw):
    """The counts for one transcript."""
    uses, results, has_result = [], {}, False
    for line in raw.splitlines():
        try:
            row = json.loads(line)
        except ValueError:
            continue
        if not isinstance(row, dict):
            continue
        has_result = has_result or row.get("type") == "result"
        content = (row.get("message") or {}).get("content")
        for part in content if isinstance(content, list) else []:
            if not isinstance(part, dict):
                continue
            if part.get("type") == "tool_use" and part.get("name") == "Bash":
                uses.append(part)
            elif part.get("type") == "tool_result":
                results[part.get("tool_use_id")] = result_text(part.get("content"))
    counts = {"landed": 0, "refused": 0, "undecided": 0, "untokenized": 0, "blocked": 0, "attributed": 0, "attributed_landed": 0}
    for use in uses:
        command = (use.get("input") or {}).get("command", "")
        if not isinstance(command, str):
            continue
        calls, _ = mrw_calls(command)
        writes = None if calls is None else [a for a in calls if is_write(a)]
        if calls is None and not ("mrw" in command and "write" in command):
            continue
        if writes is not None and not writes:
            continue
        what = outcome(results.get(use.get("id"), ""))
        if what == "blocked":
            counts["blocked"] += 1
        elif writes is None:
            counts["untokenized"] += 1
            counts["undecided"] += 1
        else:
            counts[what] += 1
            if len(writes) == 1 and is_search_replace(writes[0]) and what in ("landed", "refused"):
                counts["attributed"] += 1
                counts["attributed_landed"] += what == "landed"
    counts["decided"] = counts["landed"] + counts["refused"]
    counts["has_result"] = has_result
    return counts


# ---- Decision 5: the final-receipt walk -----------------------------------------------------------------------

def replaced(exclusion, tried):
    """bench's replaced(): a start error is replaced up to twice, a setup failure once."""
    return tried < 2 if exclusion == "start-error" else tried < 1 if exclusion == "setup" else False


def load_receipts(root):
    recs = {}
    for path in sorted(Path(root).rglob("*.receipt.json")):
        stem = path.name[:-len(".receipt.json")]
        if stem in recs:
            raise Refusal(f"two receipts named {stem}")
        try:
            rec = json.loads(path.read_text(encoding="utf-8"))
        except ValueError as err:
            raise Refusal(f"{path}: not JSON: {err}")
        m = STEM.match(stem)
        if not m or not isinstance(rec, dict) or (rec.get("task"), rec.get("config"), rec.get("model"), str(rec.get("repeat"))) != (
                m["task"], m["config"], m["model"], m["repeat"]):
            raise Refusal(f"{path}: the receipt does not read back as the attempt {stem}")
        recs[stem] = rec
    return recs


def started_without_receipt(root, recs, configs):
    for path in sorted(Path(root).rglob("*.started")):
        stem = path.name[:-len(".started")]
        m = STEM.match(stem)
        if m and m["config"] in configs and stem not in recs:
            raise Refusal(f"{stem} started and has no receipt")


def walk(recs, base):
    """(stem, receipt, dropped): the receipt that counts for base, and whether the walk ended excluded."""
    stem, tried = base, 0
    while True:
        rec = recs.get(stem)
        if rec is None:
            raise Refusal(f"{stem}: no receipt" + (" (the replacement an excluded attempt is owed)" if tried else ""))
        reason = rec.get("exclusion", "")
        if not reason:
            return stem, rec, False
        if reason not in ("start-error", "setup"):
            raise Refusal(f"{stem}: excluded as {reason!r}, which no rule of the pilot admits")
        if not replaced(reason, tried):
            return stem, rec, True
        tried += 1
        stem = f"{base}.r{tried}"


def finals(recs, units, configs):
    """{model: ([(stem, receipt) counted], [dropped (task, repeat)])} for units = {model: [(task, repeat)]}."""
    out = {}
    for model, wanted in units.items():
        counted, dropped = [], []
        for task, repeat in wanted:
            arms = [walk(recs, f"{task}.{arm}.{model}.{repeat}") for arm in configs]
            if any(drop for _, _, drop in arms):
                dropped.append((task, repeat))
            else:
                counted += [(stem, rec) for stem, rec, _ in arms]
        out[model] = (counted, dropped)
    return out


def units_seen(recs, configs):
    units = {}
    for stem, rec in recs.items():
        m = STEM.match(stem)
        if m["config"] in configs and not m["retry"]:
            units.setdefault(rec["model"], set()).add((rec["task"], rec["repeat"]))
    return {model: sorted(u) for model, u in units.items()}


def check_matrix(root, recs):
    """Refuse any instance list, order, model, arm or repeat but Decision 5's."""
    for stem, rec in recs.items():
        if rec["model"] not in MODELS:
            raise Refusal(f"{stem}: another model than {MODELS}")
        if rec["task"] not in INSTANCES:
            raise Refusal(f"{stem}: an instance the registration does not list")
        if rec["config"] not in ARMS:
            raise Refusal(f"{stem}: a third arm, {rec['config']}")
        if rec["repeat"] != 1:
            raise Refusal(f"{stem}: a repeat other than 1")
    for model in MODELS:
        path = Path(root) / model / "run.json"
        try:
            run = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError) as err:
            raise Refusal(f"{path}: {err}")
        if run.get("tasks") != INSTANCES:
            raise Refusal(f"{path}: the instances are not Decision 5's 18, in its order")
        if run.get("models") != [model] or sorted(run.get("configs") or []) != ARMS or run.get("repeats") != 1:
            raise Refusal(f"{path}: not one model, arms B and S, one repeat")


# ---- count -----------------------------------------------------------------------------------------------------

def transcripts_by_name(root):
    found = {}
    for path in sorted(Path(root).rglob("*.transcript.jsonl")):
        found.setdefault(path.name, []).append(path)
    return found


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def count(receipts, transcripts, configs):
    recs = load_receipts(receipts)
    started_without_receipt(receipts, recs, configs)
    seen = units_seen(recs, configs)
    if not seen:
        raise Refusal(f"no receipt of {configs} under {receipts}")
    index = transcripts_by_name(transcripts)
    attempts = []
    for model, (counted, _) in finals(recs, seen, configs).items():
        for stem, rec in counted:
            paths = index.get(stem + ".transcript.jsonl", [])
            if len(paths) != 1:
                raise Refusal(f"{stem}: {len(paths)} transcripts named so under {transcripts}")
            entry = classify(paths[0].read_text(encoding="utf-8", errors="replace"))
            entry.update(id=stem, task=rec["task"], model=model, config=rec["config"], transcript_sha256=sha256(paths[0]))
            attempts.append(entry)
    attempts.sort(key=lambda e: e["id"])
    return {"configs": configs, "attempts": attempts}


# ---- Decision 5: the measures and the rule ----------------------------------------------------------------------

def record(entry, rec):
    """One attempt's numbers: its counts, and its receipt's solved, output tokens, cost (nano-dollars) and turns."""
    usage = rec.get("usage") or {}
    cost = rec.get("total_cost_usd")
    present = entry["has_result"]
    return dict(entry, solved=bool((rec.get("oracle") or {}).get("solved")), turns=rec.get("num_turns") or 0,
                out=usage.get("output_tokens") if present and usage.get("output_tokens") is not None else None,
                cost=round(cost * 10 ** 9) if present and cost is not None else None)


def totals(recs, fill_out, fill_cost):
    t = {"n": len(recs)}
    for key in ("landed", "refused", "undecided", "untokenized", "blocked", "attributed_landed", "turns"):
        t[key] = sum(r[key] for r in recs)
    t["solved"] = sum(bool(r["solved"]) for r in recs)
    t["uptake"] = sum(r["attributed"] >= 1 for r in recs)
    t["with_refusal"] = sum(r["refused"] >= 1 for r in recs)
    for key, fill in (("out", fill_out), ("cost", fill_cost)):
        vals = [r[key] if r[key] is not None else fill for r in recs]
        t[key] = None if None in vals else sum(vals)
    return t


def evaluate(b, s):
    """The per-arm totals for one model, a missing output or cost imputed against S (Decision 5)."""
    pool = b + s
    fills = {}
    for key in ("out", "cost"):
        vals = [r[key] for r in pool if r[key] is not None]
        fills[key] = (min(vals), max(vals)) if vals else (None, None)
    return {"B": totals(b, fills["out"][0], fills["cost"][0]), "S": totals(s, fills["out"][1], fills["cost"][1])}


def decide(m, first_branch=True, gates=True):
    """Decision 5's rule over evaluate()'s totals: UNTESTED, PASS or FAIL, with each clause."""
    tb, ts = m["B"], m["S"]
    dec = {a: m[a]["landed"] + m[a]["refused"] for a in ARMS}
    cover = {a: dec[a] > 0 and 10 * dec[a] >= 9 * (dec[a] + m[a]["undecided"]) for a in ARMS}
    uptake = ts["n"] > 0 and 2 * ts["uptake"] >= ts["n"]
    nb, ns = tb["n"], ts["n"]
    clauses = {
        "a": ts["solved"] >= tb["solved"] - 2,
        "b1": first_branch and tb["refused"] >= 3 and dec["B"] > 0 and dec["S"] > 0 and 3 * ts["refused"] * dec["B"] <= 2 * tb["refused"] * dec["S"],
        "b2": None not in (tb["out"], ts["out"]) and nb > 0 and ns > 0 and 100 * ts["out"] * nb <= 85 * tb["out"] * ns,
        "g": tb["landed"] >= 1 and ts["landed"] >= 1 and 5 * ts["landed"] * nb <= 6 * tb["landed"] * ns,
        "c": None not in (tb["cost"], ts["cost"]) and tb["solved"] > 0 and ts["solved"] > 0
        and 10 * ts["cost"] * tb["solved"] <= 11 * tb["cost"] * ts["solved"],
    }
    clauses["b"] = clauses["b1"] or clauses["b2"]
    if gates and not (uptake and cover["B"] and cover["S"]):
        result = "UNTESTED"
    else:
        result = "PASS" if all(clauses[k] for k in ("a", "b", "g", "c")) else "FAIL"
    return {"result": result, "uptake": uptake, "coverage": cover, "clauses": clauses, "decided": dec}


def bootstrap(pairs):
    """The percentile 95% interval of S's spend over B's, minus one, over instances both arms solved; None when empty."""
    if not pairs:
        return None
    rng, stats = random.Random(BOOTSTRAP_SEED), []
    for _ in range(BOOTSTRAP):
        pick = [pairs[rng.randrange(len(pairs))] for _ in pairs]
        base = sum(p[0] for p in pick)
        if base:
            stats.append(sum(p[1] for p in pick) / base - 1)
    stats.sort()
    return (stats[len(stats) // 40], stats[len(stats) - 1 - len(stats) // 40]) if stats else None


def pct(num, den):
    return "n/a" if not den else f"{100 * num / den:.1f}%"


def per(num, den, places=2):
    return "n/a" if not den or num is None else f"{num / den:.{places}f}"


def model_lines(model, b, s, dropped):
    m = evaluate(b, s)
    d = decide(m)
    tb, ts, c = m["B"], m["S"], d["clauses"]
    mark = lambda ok: "ok" if ok else "fails"  # noqa: E731
    out = [f"{model}: {len(b)} paired instances ({len(dropped)} dropped: {', '.join(t for t, _ in dropped) or 'none'})"]
    for arm, t in (("B", tb), ("S", ts)):
        out.append(f"  {arm}: solved {t['solved']}/{t['n']}, landed {t['landed']}, refused {t['refused']}, undecided {t['undecided']},"
                   f" (i) refused rate {pct(t['refused'], d['decided'][arm])}, (ii) output per attempt {per(t['out'], t['n'], 1)},"
                   f" (iii) spend per success {per(None if t['cost'] is None else t['cost'] / 1e9, t['solved'], 4)},"
                   f" (g) landed per attempt {per(t['landed'], t['n'])}, (O) coverage {pct(d['decided'][arm], d['decided'][arm] + t['undecided'])}")
    out += [
        f"  (U) S attempts with an attributed SEARCH/REPLACE write call: {ts['uptake']} of {ts['n']}, need {-(-ts['n'] // 2)}: {mark(d['uptake'])}",
        f"  (O) coverage at least 90% and a decided call on each arm: {mark(all(d['coverage'].values()))}",
        f"  (a) S solved {ts['solved']} >= B solved {tb['solved']} - 2: {mark(c['a'])}",
        f"  (b) B refused {tb['refused']} >= 3 and S (i) <= 2/3 of B's: {mark(c['b1'])}; or S (ii) <= 0.85 of B's: {mark(c['b2'])}; (b): {mark(c['b'])}",
        f"  (g) S landed per attempt <= 1.2 times B's, both landed: {mark(c['g'])}",
        f"  (c) S spend per success <= 1.10 times B's, both solved: {mark(c['c'])}",
    ]
    both = {r["task"] for r in b if r["solved"]} & {r["task"] for r in s if r["solved"]}
    cost = {a: {r["task"]: r["cost"] for r in recs} for a, recs in (("B", b), ("S", s))}
    # The subset uses the recorded costs; an imputed one belongs to totals().
    pairs = [(cost["B"][t], cost["S"][t]) for t in sorted(both) if cost["B"][t] is not None and cost["S"][t] is not None]
    ci = bootstrap(pairs)
    out.append(f"  descriptive: both-solved subset {len(both)}, S spend over B's minus one "
               + ("none" if ci is None else f"[{100 * ci[0]:.1f}%, {100 * ci[1]:.1f}%] (4,000 resamples, seed 193)"))
    for arm, t in (("B", tb), ("S", ts)):
        out.append(f"  descriptive {arm}: output per landed write call {per(t['out'], t['landed'], 0)}, turns per attempt {per(t['turns'], t['n'], 1)},"
                   f" attempts with a refusal {t['with_refusal']}, blocked {t['blocked']}, untokenized {t['untokenized']},"
                   f" landed attributed SEARCH/REPLACE calls {t['attributed_landed']}")
    out.append(f"  {model}: {d['result']}")
    return d["result"], out


def load_counts(path, wanted, transcripts):
    try:
        data = json.loads(Path(path).read_text(encoding="utf-8"))
        attempts = data["attempts"]
    except (OSError, ValueError, KeyError, TypeError) as err:
        raise Refusal(f"{path}: not a count file: {err}")
    ids = [a.get("id") for a in attempts]
    if len(set(ids)) != len(ids):
        raise Refusal("the counts name an attempt twice")
    if set(ids) != wanted:
        extra, missing = sorted(set(ids) - wanted), sorted(wanted - set(ids))
        raise Refusal(f"the counts are not the final receipts: not final {extra[:3]}, missing {missing[:3]}")
    index = transcripts_by_name(transcripts) if transcripts else None
    for a in attempts:
        if not re.fullmatch(r"[0-9a-f]{64}", str(a.get("transcript_sha256"))):
            raise Refusal(f"{a['id']}: no transcript digest")
        if index is not None:
            paths = index.get(a["id"] + ".transcript.jsonl", [])
            if len(paths) != 1 or sha256(paths[0]) != a["transcript_sha256"]:
                raise Refusal(f"{a['id']}: the transcript is not the one the counts digest")
    return {a["id"]: a for a in attempts}


def verdict(receipts, counts_path, transcripts=None):
    recs = load_receipts(receipts)
    started_without_receipt(receipts, recs, ARMS)
    check_matrix(receipts, recs)
    done = finals(recs, {m: [(t, 1) for t in INSTANCES] for m in MODELS}, ARMS)
    for model, (_, dropped) in done.items():
        if len(dropped) > MAX_DROPPED:
            raise Refusal(f"{model}: {len(dropped)} dropped instances, more than {MAX_DROPPED}: the pilot stops")
    counts = load_counts(counts_path, {stem for counted, _ in done.values() for stem, _ in counted}, transcripts)
    lines, results = ["ADR-193 pilot: 18 instances, B and S, one repeat, claude-opus-5-5 and claude-sonnet-5-5"], []
    for model, (counted, dropped) in done.items():
        by_arm = {a: [record(counts[stem], rec) for stem, rec in counted if rec["config"] == a] for a in ARMS}
        result, rows = model_lines(model, by_arm["B"], by_arm["S"], dropped)
        lines += rows
        results.append(result)
    lines.append("verdict: " + ("SHIP" if results == ["PASS"] * len(MODELS) else "NO SHIP"))
    return lines


# ---- null: how often the rule says PASS with no effect at all ------------------------------------------------------

def as_recorded(r, rec):
    """r with a missing output or cost left as the receipt holds it (zero tokens, no cost counted)."""
    cost = rec.get("total_cost_usd")
    return dict(r, out=(rec.get("usage") or {}).get("output_tokens") or 0, cost=round((cost or 0) * 10 ** 9))


def null(transcripts):
    """Decision 5's resampling: per iteration, per model, 36 of its B attempts drawn without replacement, the first
    18 taken as B and the last 18 as S; uptake and coverage taken to hold. The models are drawn in the order of
    their names, one rng seeded 193, 20,000 iterations. Each draw is judged twice: with a missing output or cost
    imputed against S (the frozen rule), and with it as the receipt holds it, which is how the Context's figures
    were drawn (the attempts with no result record zero tokens and no cost)."""
    pools = {}
    for path in sorted(Path(transcripts).rglob("*.receipt.json")):
        rec = json.loads(path.read_text(encoding="utf-8"))
        stem = path.name[:-len(".receipt.json")]
        sibling = path.with_name(stem + ".transcript.jsonl")
        if rec.get("config") != "B" or STEM.match(stem)["retry"] or not sibling.exists():
            continue
        r = record(classify(sibling.read_text(encoding="utf-8", errors="replace")), rec)
        pools.setdefault(rec["model"], []).append((r, as_recorded(r, rec)))
    if not pools or any(len(p) < 36 for p in pools.values()):
        raise Refusal(f"fewer than 36 B attempts with a transcript for a model under {transcripts}")
    rng, models = random.Random(SEED), sorted(pools)
    readings = ("imputed", "as recorded")
    tally = {(m, k, first): 0 for m in models for k in range(2) for first in (True, False)}
    both = {(k, first): 0 for k in range(2) for first in (True, False)}
    for _ in range(RESAMPLES):
        passed = {key: [] for key in both}
        for m in models:
            pick = rng.sample(range(len(pools[m])), 36)
            for k in range(2):
                ev = evaluate([pools[m][i][k] for i in pick[:18]], [pools[m][i][k] for i in pick[18:]])
                for first in (True, False):
                    ok = decide(ev, first_branch=first, gates=False)["result"] == "PASS"
                    tally[m, k, first] += ok
                    passed[k, first].append(ok)
        for key, oks in passed.items():
            both[key] += all(oks)
    lines = [f"null: {RESAMPLES} resamples, seed {SEED}, per model 36 of its B attempts without replacement, 18 as B and 18 as S"]
    for k, reading in enumerate(readings):
        lines.append(f"{reading}{' (the frozen rule)' if k == 0 else ''}:")
        for m in models:
            lines.append(f"  {m}: {len(pools[m])} B attempts, PASS {100 * tally[m, k, True] / RESAMPLES:.1f}%"
                         f" (with (i) unavailable {100 * tally[m, k, False] / RESAMPLES:.1f}%)")
        lines.append(f"  false SHIP (every model PASS): {100 * both[k, True] / RESAMPLES:.1f}%; with (i) unavailable: {100 * both[k, False] / RESAMPLES:.1f}%")
    return lines


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sub = ap.add_subparsers(dest="mode", required=True)
    c = sub.add_parser("count")
    c.add_argument("--receipts", required=True)
    c.add_argument("--transcripts", required=True)
    c.add_argument("--configs", default=",".join(ARMS))
    v = sub.add_parser("verdict")
    v.add_argument("--receipts", required=True)
    v.add_argument("--counts", required=True)
    v.add_argument("--transcripts")
    n = sub.add_parser("null")
    n.add_argument("--transcripts", required=True)
    args = ap.parse_args(argv)
    try:
        if args.mode == "count":
            print(json.dumps(count(args.receipts, args.transcripts, args.configs.split(",")), indent=1, sort_keys=True))
        elif args.mode == "verdict":
            print("\n".join(verdict(args.receipts, args.counts, args.transcripts)))
        else:
            print("\n".join(null(args.transcripts)))
    except Refusal as err:
        print(f"bench_edits: refused: {err}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
