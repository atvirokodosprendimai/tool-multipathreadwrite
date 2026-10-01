#!/usr/bin/env python3
"""Combination stress of `mrw write` through a built binary (B2, BACKLOG "From ADR-108").

Flags were tested one at a time; this drives their product. Each run gets a fresh checkout, a seeded
choice of plan format, plan shape, check configuration and flags, and is held to invariants that no
single-flag test states together.

Run: python3 docs/break/write-flags/stress.py            # the mrw on PATH
     MRW=$PWD/bin/mrw SEED=7 N=400 python3 docs/break/write-flags/stress.py
Unix only. Exit 0 = no violation; 1 = violations, written to a JSON file in the system temp directory
whose path is printed. Writes nothing into the checkout.

Invariants, checked on every run:
  W1 the exit code is 0, 1, 2 or 3
  W2 under --json stdout is exactly one JSON object, and every key path it carries is listed for
     `write` in docs/receipts.txt (ADR-111) — except a usage refusal (exit 2) made before any
     receipt, such as --check beside --dry-run, which prints nothing on stdout, as a flag the
     parser rejects does
  W3 exit 1 leaves the tree byte-identical (ADR-001: nothing written)
  W4 --dry-run leaves the tree byte-identical
  W5 a landed write (exit 0 or 3, no --dry-run, a plan that can apply) changed a.txt to exactly the
     planned bytes and nothing else
  W6 under --json with exit 0, 1 or 3 every hunk the plan holds has a verdict
  W7 exit 3 means a check ran and did not pass, or a step did not pass
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

PLANS = {
    "native": "@@ a.txt 1 replace\nb\n",
    "apply_patch": "*** Begin Patch\n*** Update File: a.txt\n@@\n-a\n+b\n*** End Patch\n",
    "search_replace": "a.txt\n<<<<<<< SEARCH\na\n=======\nb\n>>>>>>> REPLACE\n",
}
BAD = {
    "native": "@@ a.txt 9 replace\nx\n",
    "apply_patch": "*** Begin Patch\n*** Update File: a.txt\n@@\n-nope\n+x\n*** End Patch\n",
    "search_replace": "a.txt\n<<<<<<< SEARCH\nnope\n=======\nx\n>>>>>>> REPLACE\n",
}
CHECKS = {"none": None, "pass": {"check": "true"}, "fail": {"check": "exit 1"}}


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
    for root, dirs, files in os.walk(d):
        for name in files:
            p = os.path.join(root, name)
            with open(p, "rb") as fh:
                snap[os.path.relpath(p, d)] = fh.read()
    return snap


def one(i, rng):
    fmt = rng.choice(list(PLANS))
    shape = rng.choice(["good", "good", "bad", "unread"])
    check = rng.choice(list(CHECKS))
    flags = []
    for f, p in (("--json", 0.5), ("--dry-run", 0.2), ("--strict-balance", 0.2)):
        if rng.random() < p:
            flags.append(f)
    c = rng.choice(["default", "--check", "--no-check"])
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
    with open(os.path.join(d, "a.txt"), "w") as fh:
        fh.write("a\n")
    if CHECKS[check]:
        with open(os.path.join(d, ".quality-harness.json"), "w") as fh:
            json.dump(CHECKS[check], fh)
    plan = PLANS[fmt] if shape != "bad" else BAD[fmt]
    pf = os.path.join(WORK, "p%d" % i)
    with open(pf, "w") as fh:
        fh.write(plan)
    env = {k: v for k, v in os.environ.items() if k != "MRW_STEP_DEPTH"}
    env.update(XDG_STATE_HOME=state, TMPDIR=TMP)
    if shape != "unread":
        subprocess.run([MRW, "-C", d, "read", "a.txt"], env=env, capture_output=True, timeout=20)
    before = snapshot(d)
    try:
        p = subprocess.run([MRW, "-C", d, "write"] + flags + [pf], env=env, capture_output=True, text=True, timeout=30)
    except subprocess.TimeoutExpired:
        violations.append({"run": i, "flags": flags, "fmt": fmt, "shape": shape, "check": check, "why": "timeout"})
        return
    code, out = p.returncode, p.stdout
    after = snapshot(d)
    case = {"run": i, "flags": flags, "fmt": fmt, "shape": shape, "check": check, "exit": code}
    key = "%s/%s/%s/exit%d" % (fmt, shape, check, code)
    cov[key] = cov.get(key, 0) + 1

    def bad(why):
        violations.append(dict(case, why=why, stdout=out[:400], stderr=p.stderr[:400]))

    if code not in (0, 1, 2, 3):
        bad("W1 exit code outside 0-3")
    doc = None
    if "--json" in flags and not (code == 2 and out.strip() == ""):
        try:
            doc = json.loads(out)
            if not isinstance(doc, dict):
                bad("W2 --json stdout is not one object")
            else:
                extra = sorted(paths(doc) - WRITE_KEYS)
                if extra:
                    bad("W2 keys not in docs/receipts.txt: %s" % extra)
        except ValueError:
            bad("W2 --json stdout is not one JSON object")
    if code == 1 and after != before:
        bad("W3 exit 1 changed the tree")
    if "--dry-run" in flags and after != before:
        bad("W4 --dry-run changed the tree")
    if code in (0, 3) and "--dry-run" not in flags and shape == "good":
        changed = {k for k in set(before) | set(after) if before.get(k) != after.get(k)}
        if after.get("a.txt") != b"b\n" or changed - {"a.txt"}:
            bad("W5 a landed write did not have exactly the planned effect: %s" % sorted(changed))
    if doc and isinstance(doc, dict) and code in (0, 1, 3):
        if len(doc.get("hunks") or []) < 1:
            bad("W6 a hunk has no verdict")
    if code == 3:
        chk = (doc or {}).get("check") if doc else None
        steps = ((doc or {}).get("then") or {}).get("steps") if doc else None
        failed_step = any(s.get("status") not in ("passed", "pass", "ok") for s in steps or [])
        if doc and not ((chk and not chk.get("exit_code") == 0) or failed_step):
            bad("W7 exit 3 with a passing check and no failing step")


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
    print("seed %d, %d runs, %d violation(s), %d distinct format/shape/check/exit cells" % (SEED, N, len(violations), len(cov)))
    for k in sorted(cov):
        print("  %-40s %d" % (k, cov[k]))
    if violations:
        fd, path = tempfile.mkstemp(prefix="stress-write-violations-", suffix=".json")
        with os.fdopen(fd, "w") as fh:
            json.dump(violations, fh, indent=2)
        print("violations written to", path)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
