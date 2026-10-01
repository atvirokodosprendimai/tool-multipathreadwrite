#!/usr/bin/env python3
"""Combination stress of `mrw write` through a built binary (B2, BACKLOG "From ADR-108").

Flags were tested one at a time; this drives their product. Each run gets a fresh checkout, a seeded
choice of target, plan format, plan shape, check configuration and flags, and is held to invariants
that no single-flag test states together.

Run: python3 docs/break/write-flags/stress.py            # the mrw on PATH
     MRW=$PWD/bin/mrw SEED=7 N=400 python3 docs/break/write-flags/stress.py
Unix only. Exit 0 = no violation; 1 = violations, written to a JSON file in the system temp directory
whose path is printed. Writes nothing into the checkout.

Two targets, because mrw treats them differently: a.txt is prose (no default check, no balance rows)
and x.go is code (the check runs by default, and a hunk that moves the delimiter balance prints a
balance row; under --strict-balance a single-line replace of the wrap-tail shape — the replaced line's
delimiters do not balance and the body does not match them, ADR-055 — is refused).

Invariants, checked on every run:
  W1 the exit code is 0, 1, 2 or 3
  W2 under --json stdout is exactly one JSON object whose every key path docs/receipts.txt lists for
     `write` (ADR-111); the one exemption is --check beside --dry-run, a usage refusal made before the
     plan, which prints nothing on stdout as a flag the parser rejects does
  W3 exit 1 leaves the tree byte-identical (ADR-001: nothing written)
  W4 --dry-run leaves the tree byte-identical
  W5 a landed write (exit 0 or 3, no --dry-run, a plan that applies) changed the target to exactly the
     planned bytes and nothing else
  W6 under --json with exit 0, 1 or 3 the receipt carries exactly the plan's one hunk, with a status of
     ok, failed or skipped — ok when the write landed
  W7 under --json, exit 3 means the check ran and did not pass (a non-zero exit, or a timeout or an
     interruption), or a step failed, timed out or was interrupted
  W8 --strict-balance refuses a wrap-tail single-line code replace (`func f() {` replaced by
     `func g()`): exit 1, tree unchanged; without it the hunk lands and the receipt counts a balance
     advisory
  W9 a landed code write with a check configured and neither --check nor --no-check runs the check;
     a landed prose write under the same flags does not
"""
import json, os, random, shutil, subprocess, sys, tempfile

MRW = os.environ.get("MRW") or shutil.which("mrw")
HERE = os.path.dirname(os.path.abspath(__file__))
RECEIPTS = os.path.join(HERE, "..", "..", "receipts.txt")
WORK = tempfile.mkdtemp(prefix="stress-write-")
TMP = os.path.join(WORK, "tmp")
os.makedirs(TMP)
SEED = int(os.environ.get("SEED", "1380"))
N = int(os.environ.get("N", "400"))
violations, cov = [], {}

# target: (the file's lines, the line a good plan replaces (0-based), its new text)
TARGETS = {"a.txt": (["a"], 0, "b"), "x.go": (["package x", "func f() {", "}"], 0, "package y")}
CHECKS = {"none": None, "pass": {"check": "true"}, "fail": {"check": "exit 1"}}
STEP_FAILED = ("fail", "timed_out", "interrupted")


def plan(fmt, target, line, old, new):
    if fmt == "native":
        return "@@ %s %d replace\n%s\n" % (target, line + 1, new)
    if fmt == "apply_patch":
        return "*** Begin Patch\n*** Update File: %s\n@@\n-%s\n+%s\n*** End Patch\n" % (target, old, new)
    return "%s\n<<<<<<< SEARCH\n%s\n=======\n%s\n>>>>>>> REPLACE\n" % (target, old, new)


def listed():
    keys = set()
    with open(RECEIPTS) as fh:
        for line in fh:
            f = line.split()
            if len(f) == 2 and f[0] == "write":
                keys.add(f[1])
    return keys


WRITE_KEYS = listed()


def paths(v, prefix=""):
    """Key paths of a JSON value, spelled as docs/receipts.txt spells them."""
    out = set()
    if isinstance(v, dict):
        for k, x in v.items():
            p = prefix + k
            out.add(p)
            out |= paths(x, p + ".")
    elif isinstance(v, list):
        for x in v:
            out |= paths(x, prefix.rstrip(".") + "[].")
    return out


def snapshot(d):
    snap = {}
    for root, _dirs, files in os.walk(d):
        for name in files:
            p = os.path.join(root, name)
            with open(p, "rb") as fh:
                snap[os.path.relpath(p, d)] = fh.read()
    return snap


def one(i, rng):
    target = rng.choice(list(TARGETS))
    lines, line, new = TARGETS[target]
    old = lines[line]
    fmt = rng.choice(["native", "apply_patch", "search_replace"])
    shape = rng.choice(["good", "good", "bad", "unread", "unbalanced"])
    if shape == "unbalanced" and (fmt != "native" or target != "x.go"):
        shape = "good"
    if shape == "unbalanced":
        line, old, new = 1, lines[1], "func g()"
    if shape == "bad":
        old, new = "nope", "x"
    check = rng.choice(list(CHECKS))
    flags = []
    for f, p in (("--json", 0.6), ("--dry-run", 0.2), ("--strict-balance", 0.3)):
        if rng.random() < p:
            flags.append(f)
    c = rng.choice(["default", "default", "--check", "--no-check"])
    if c != "default":
        flags.append(c)
    step = rng.choice([None, None, "true", "exit 1"])
    if step:
        flags += ["--then-sh", step]
    if rng.random() < 0.2:
        flags += ["--echo-pad", str(rng.randint(0, 3))]
    if fmt != "native":
        flags.append("--format=" + fmt)

    d = tempfile.mkdtemp(dir=WORK, prefix="t%d-" % i)
    state = os.path.join(WORK, "state%d" % i)
    for t, (ls, _l, _n) in TARGETS.items():
        with open(os.path.join(d, t), "w") as fh:
            fh.write("\n".join(ls) + "\n")
    if CHECKS[check]:
        with open(os.path.join(d, ".quality-harness.json"), "w") as fh:
            json.dump(CHECKS[check], fh)
    text = plan(fmt, target, line, old, new)
    if shape == "bad" and fmt == "native":
        text = "@@ %s 9 replace\nx\n" % target
    pf = os.path.join(WORK, "p%d" % i)
    with open(pf, "w") as fh:
        fh.write(text)
    env = {k: v for k, v in os.environ.items() if k != "MRW_STEP_DEPTH"}
    env.update(XDG_STATE_HOME=state, TMPDIR=TMP)
    if shape != "unread":
        subprocess.run([MRW, "-C", d, "read", target], env=env, capture_output=True, timeout=20)
    before = snapshot(d)
    case = {"run": i, "flags": flags, "target": target, "fmt": fmt, "shape": shape, "check": check}
    try:
        p = subprocess.run([MRW, "-C", d, "write"] + flags + [pf], env=env, capture_output=True, text=True, timeout=30)
    except subprocess.TimeoutExpired:
        violations.append(dict(case, why="timeout"))
        return
    code, out = p.returncode, p.stdout
    after = snapshot(d)
    case["exit"] = code
    key = "%s/%s/%s/%s/exit%d" % (target, fmt, shape, check, code)
    cov[key] = cov.get(key, 0) + 1
    dry = "--dry-run" in flags
    conflict = "--check" in flags and dry

    def bad(why):
        violations.append(dict(case, why=why, stdout=out[:400], stderr=p.stderr[:400]))

    if code not in (0, 1, 2, 3):
        bad("W1 exit code outside 0-3")
    doc = None
    if "--json" in flags and not (conflict and code == 2 and out.strip() == ""):
        try:
            parsed = json.loads(out)
        except ValueError:
            bad("W2 --json stdout is not one JSON object")
        else:
            # Any decoded value that is not an object fails, null included:
            # None would otherwise read as "no receipt to check".
            if isinstance(parsed, dict):
                doc = parsed
            else:
                bad("W2 --json stdout is %s, not one object" % type(parsed).__name__)
        if doc is not None:
            extra = sorted(paths(doc) - WRITE_KEYS)
            if extra:
                bad("W2 keys not in docs/receipts.txt: %s" % extra)
    if code == 1 and after != before:
        bad("W3 exit 1 changed the tree")
    if dry and after != before:
        bad("W4 --dry-run changed the tree")
    landed = code in (0, 3) and not dry and shape in ("good", "unbalanced")
    if landed:
        changed = {k for k in set(before) | set(after) if before.get(k) != after.get(k)}
        want = list(lines)
        want[line] = new
        if after.get(target) != ("\n".join(want) + "\n").encode() or changed - {target}:
            bad("W5 a landed write did not have exactly the planned effect: %s" % sorted(changed))
    if doc is not None and code in (0, 1, 3):
        hunks = doc.get("hunks")
        if not isinstance(hunks, list) or len(hunks) != 1:
            bad("W6 the receipt does not carry exactly the plan's one hunk")
        else:
            st = hunks[0].get("status") if isinstance(hunks[0], dict) else None
            if st not in ("ok", "failed", "skipped"):
                bad("W6 a hunk has no valid status: %r" % st)
            elif code in (0, 3) and st != "ok":
                bad("W6 a write that exited %d has a hunk with status %s" % (code, st))
    if doc is not None and code == 3:
        chk = doc.get("check")
        checked_bad = isinstance(chk, dict) and chk.get("ran") is True and (
            (isinstance(chk.get("exit_code"), int) and chk["exit_code"] != 0) or bool(chk.get("skipped")))
        steps = (doc.get("then") or {}).get("steps") or []
        step_bad = any(isinstance(s, dict) and s.get("status") in STEP_FAILED for s in steps)
        if not (checked_bad or step_bad):
            bad("W7 exit 3 without a check that ran and failed or a step that failed")
    if shape == "unbalanced" and not conflict:
        if "--strict-balance" in flags:
            if code != 1 or after != before:
                bad("W8 --strict-balance did not refuse a wrap-tail single-line code replace")
        else:
            # Without the flag the wrap-tail replace is valid: validation never
            # refuses it, whatever the check, a step or --dry-run then does.
            if code == 1:
                bad("W8 a wrap-tail replace was refused without --strict-balance")
            if doc is not None:
                hunks = doc.get("hunks") or []
                st = hunks[0].get("status") if hunks and isinstance(hunks[0], dict) else None
                if st != "ok" or not (doc.get("advisories", 0) >= 1):
                    bad("W8 a wrap-tail replace without --strict-balance did not pass with a balance advisory")
    if landed and check != "none" and "--check" not in flags and "--no-check" not in flags and doc is not None:
        ran = isinstance(doc.get("check"), dict) and doc["check"].get("ran") is True
        if target == "x.go" and not ran:
            bad("W9 a landed code write with a check configured did not run it")
        if target == "a.txt" and ran:
            bad("W9 a landed prose write ran the check by default")


def main():
    if not MRW:
        print("no mrw on PATH and no MRW set", file=sys.stderr)
        return 2
    rng = random.Random(SEED)
    try:
        for i in range(N):
            one(i, rng)
    finally:
        # The work directory goes with the run, an interrupted one included.
        shutil.rmtree(WORK, ignore_errors=True)
    print("seed %d, %d runs, %d violation(s), %d distinct target/format/shape/check/exit cells" % (SEED, N, len(violations), len(cov)))
    for k in sorted(cov):
        print("  %-48s %d" % (k, cov[k]))
    if violations:
        fd, path = tempfile.mkstemp(prefix="stress-write-violations-", suffix=".json")
        with os.fdopen(fd, "w") as fh:
            json.dump(violations, fh, indent=2)
        print("violations written to", path)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
