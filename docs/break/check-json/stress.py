#!/usr/bin/env python3
"""Stress of `mrw check --json` refusals and exit codes (ADR-100, ADR-101) through a built binary.

Run: python3 docs/break/check-json/stress.py            # the mrw on PATH
     MRW=$PWD/bin/mrw SEED=7 N=1500 python3 docs/break/check-json/stress.py
Unix only (a step-log case chmods a temp directory). Exit 0 = no violation; 1 = violations, written
to a JSON file in the system temp directory whose path is printed. Writes nothing into the checkout.

Invariants, checked on every run:
  I1 exit code is 0, 2 or 3
  I2 when stdout is non-empty under --json it is exactly one JSON object
  I3 an object carrying "error" has no "exit_code"; its only keys are error and, maybe, then
  I4 "ran": false in a check result or a step => exit_code == -1; exit_code 0 => ran true
  I5 a --json run that exits 2 and was not refused before flag parsing prints exactly one object
  I6 a refusal refused before flag parsing prints nothing on stdout
  I7 MRW_STEP_DEPTH parsing as an int >= 8 => exit 2 naming MRW_STEP_DEPTH (unless refused pre-parse)
  I8 without --json stdout never holds a JSON error document
  I9 a run that answered with an error document ran no check (the marker the check touches is absent)
"""
import concurrent.futures as cf, json, os, random, shutil, subprocess, sys, tempfile

MRW = os.environ.get("MRW") or shutil.which("mrw")
WORK = tempfile.mkdtemp(prefix="stress134-")
SEED = int(os.environ.get("SEED", "1340"))
N = int(os.environ.get("N", "600"))
violations, counts, COV = [], {}, {}


def tree(kind):
    d = tempfile.mkdtemp(dir=WORK, prefix=kind + "-")
    open(os.path.join(d, "a.txt"), "w").write("a\n")
    open(os.path.join(d, "x.go"), "w").write("package x\n")
    h = {
        "nocheck": None,
        "pass": {"check": "touch ran.marker", "steps": {"a": "true", "bad": "exit 1"}},
        "fail": {"check": "touch ran.marker; exit 1", "steps": {"a": "true"}},
        "malformed": "{not json",
        "timeout": {"check": "sleep 5", "timeout_seconds": 1},
        "signal": {"check": "kill -9 $$"},
        "big": {"check": "touch ran.marker; exit 255"},
    }[kind if kind != "unreadable" else "pass"]
    if h is not None:
        open(os.path.join(d, ".quality-harness.json"), "w").write(h if isinstance(h, str) else json.dumps(h))
    return d


def base_env(state, extra=None):
    env = {k: v for k, v in os.environ.items() if k not in ("MRW_STEP_DEPTH",)}
    env["XDG_STATE_HOME"] = state
    env.update(extra or {})
    return env


def run(argv, cwd, env, t=20):
    try:
        p = subprocess.run([MRW] + argv, cwd=cwd, env=env, capture_output=True, text=True, timeout=t)
        return p.returncode, p.stdout, p.stderr
    except subprocess.TimeoutExpired:
        return "TIMEOUT", "", ""


def docs(s):
    """Every top-level JSON value in s, or None if s is not a clean sequence of them."""
    dec, i, out = json.JSONDecoder(), 0, []
    while True:
        while i < len(s) and s[i].isspace():
            i += 1
        if i == len(s):
            return out
        try:
            v, i = dec.raw_decode(s, i)
        except ValueError:
            return None
        out.append(v)


def preparse(err):
    return ("argument parser strips" in err and "as its own argument" in err) or \
        "flag provided but not defined" in err or "is not a flag" in err or "Incorrect Usage" in err


def depth_int(v):
    """strconv.Atoi on a 64-bit build: no spaces, optional sign, int64 range; otherwise 0 (ADR-095 D3)."""
    if v is None or v != v.strip() or v == "":
        return 0
    try:
        n = int(v, 10)
    except ValueError:
        return 0
    if n < 0 or n > 2**63 - 1:
        return 0
    return n


def results_in(o):
    """(where, dict) for every check result and step in a document."""
    out = []
    if isinstance(o, dict):
        if "ran" in o and "exit_code" in o:
            out.append(("check", o))
        if isinstance(o.get("check"), dict):
            out.append(("write.check", o["check"]))
        for k in ("then",):
            th = o.get(k)
            if isinstance(th, dict):
                for s in th.get("steps") or []:
                    out.append(("step", s))
    return out


def judge(suite, argv, cwd, env, code, out, err, marker=None):
    counts[suite] = counts.get(suite, 0) + 1
    bad = []
    json_flag = "--json" in (argv[:argv.index("--")] if "--" in argv else argv)
    if code == "TIMEOUT":
        bad.append("I1 hung")
    elif code not in (0, 2, 3):
        bad.append(f"I1 exit {code}")
    pre = preparse(err)
    ds = docs(out)
    if json_flag and out.strip():
        if ds is None or len(ds) != 1 or not isinstance(ds[0], dict):
            bad.append("I2 stdout is not exactly one JSON object")
    if ds:
        for o in ds:
            if isinstance(o, dict) and "error" in o:
                if "exit_code" in o:
                    bad.append("I3 error document carries exit_code")
                # a write refusal is its receipt plus error (ADR-072); a check refusal is error (+then)
                if argv[:1] == ["check"] and set(o) - {"error", "then"}:
                    bad.append(f"I3 error document keys {sorted(o)}")
            for where, r in results_in(o if isinstance(o, dict) else {}):
                if r.get("ran") is False and r.get("exit_code") != -1:
                    bad.append(f"I4 {where} ran false with exit_code {r.get('exit_code')}")
                if r.get("exit_code") == 0 and r.get("ran") is not True:
                    bad.append(f"I4 {where} exit_code 0 without ran true")
    if json_flag and code == 2 and not pre:
        if ds is None or len(ds) != 1:
            bad.append(f"I5 --json exit 2 printed {0 if not ds else len(ds)} documents")
    if pre and out.strip():
        bad.append("I6 pre-parse refusal wrote stdout")
    if depth_int(env.get("MRW_STEP_DEPTH")) >= 8 and not pre and argv and argv[0] in ("check",):
        if code != 2 or "MRW_STEP_DEPTH" not in err:
            bad.append(f"I7 depth {env.get('MRW_STEP_DEPTH')!r}: exit {code}, no depth refusal")
    if not json_flag and '"error"' in out:
        bad.append("I8 plain run printed an error document")
    if marker and ds and len(ds) == 1 and isinstance(ds[0], dict) and "error" in ds[0] and os.path.exists(marker):
        bad.append("I9 an error document but the check ran")
    tag = []
    if pre: tag.append("preparse-refusal")
    if ds and len(ds) == 1 and isinstance(ds[0], dict):
        o = ds[0]
        if "error" in o: tag.append("error-doc" + ("+then" if "then" in o else ""))
        if o.get("ran") is False: tag.append("ran-false(-1)")
        if o.get("ran") is True: tag.append(f"ran-true(exit {o.get('exit_code')})")
        if isinstance(o.get("check"), dict): tag.append(f"write.check ran={o['check'].get('ran')} exit={o['check'].get('exit_code')}")
        for s in ((o.get("then") or {}).get("steps") or []): tag.append(f"step {s.get('status')} {s.get('exit_code')}")
    if "MRW_STEP_DEPTH" in err: tag.append("depth-refusal")
    if not json_flag and code == 2 and not out.strip(): tag.append("plain-refusal-empty-stdout")
    tag.append(f"exit {code}")
    for x in tag:
        COV[x] = COV.get(x, 0) + 1
    for b in bad:
        violations.append({"suite": suite, "violation": b, "argv": argv, "cwd": cwd,
                           "depth": env.get("MRW_STEP_DEPTH"), "tmpdir": env.get("TMPDIR"),
                           "exit": code, "stdout": out[:300], "stderr": err[:300]})


def c(suite, t, argv, env, cwd=None):
    m = os.path.join(t, "ran.marker")
    if os.path.exists(m):
        os.remove(m)
    code, out, err = run(["-C", t] + argv, cwd or t, env)
    judge(suite, argv, t, env, code, out, err, m)
    return code, out, err


def targeted():
    st = tempfile.mkdtemp(dir=WORK, prefix="state-")
    env = base_env(st)
    trees = {k: tree(k) for k in ("nocheck", "pass", "fail", "malformed", "timeout", "signal", "big", "unreadable")}
    # the unreadable working set: the iteration file is a directory
    u = trees["unreadable"]
    run(["-C", u, "iter", "add", "a.txt"], u, env)
    p = subprocess.run(["find", st, "-name", "iteration", "-type", "f"], capture_output=True, text=True).stdout.split()
    for f in p:
        if open(os.path.join(os.path.dirname(f), "root")).read().strip() in (u, os.path.realpath(u)):
            os.remove(f); os.mkdir(f)
    paths = ["x.go ", " x.go", "x.go\t", "../outside", "/etc", "nosuchdir", "chek.go", "a.txt", "help", "h",
             "sp ace", "ünï", "-- -x"]
    for k, t in trees.items():
        for j in (["--json"], []):
            c("t.full-alone", t, ["check"] + j + ["--full"], env)
            c("t.working-set", t, ["check"] + j, env)
            for pth in paths:
                args = pth.split(" ", 1) if pth.startswith("-- ") else [pth]
                c("t.path", t, ["check"] + j + args, env)
                c("t.full-path", t, ["check"] + j + ["--full"] + args, env)
            c("t.full-many", t, ["check"] + j + ["--full", "a.txt", "x.go", "../outside"], env)
            c("t.then-bad", t, ["check"] + j + ["--full", "--then", "nope"], env)
            c("t.then-placeholder", t, ["check"] + j + ["--full", "--then-sh", "echo {files}"], env)
            c("t.then-ok", t, ["check"] + j + ["--full", "--then", "a", "--then-sh", "true"], env)
            c("t.preparse-pad", t, ["check"] + j + ["--then-sh=true "], env)
            c("t.preparse-flag", t, ["check"] + j + ["--bogus"], env)
            c("t.preparse-subcmd", t, ["check"] + j + ["--read"], env)
            for dv in ("7", "8", "9", "08", "+8", " 8", "-1", "abc", "99999999999999999999", ""):
                e2 = dict(env, MRW_STEP_DEPTH=dv)
                c("t.depth", t, ["check"] + j + ["--full", "x.go"], e2)
                c("t.depth", t, ["check"] + j + ["x.go "], e2)
                c("t.depth", t, ["check"] + j + ["--full"], e2)
            gone = os.path.join(WORK, "gone-" + k)
            c("t.tmp-gone", t, ["check"] + j + ["--full"], dict(env, TMPDIR=gone))
            c("t.tmp-gone-steps", t, ["check"] + j + ["--full", "--then-sh", "true"], dict(env, TMPDIR=gone))
            f = os.path.join(WORK, "tmpfile-" + k)
            open(f, "w").close()
            c("t.tmp-is-file", t, ["check"] + j + ["--full"], dict(env, TMPDIR=f))
    # write --check --json in each tree: read, then a plan resolved from cwd
    for k, t in trees.items():
        run(["-C", t, "read", "a.txt"], t, env)
        open(os.path.join(t, "p.mrw"), "w").write("@@ a.txt 1 replace\nw\n")
        c("t.write-check", t, ["write", "--check", "--json", "p.mrw"], env)
        run(["-C", t, "read", "a.txt"], t, env)
        c("t.write-check-then", t, ["write", "--check", "--json", "--then-sh", "true", "p.mrw"], env)
    # ADR-101's step path end to end: the check passes, then makes TMPDIR unwritable,
    # so the step cannot create its log. Its entry must say ran false, exit_code -1.
    t = tempfile.mkdtemp(dir=WORK, prefix="steplog-")
    td = tempfile.mkdtemp(dir=WORK, prefix="td-")
    open(os.path.join(t, ".quality-harness.json"), "w").write(json.dumps({"check": 'chmod 555 "$TMPDIR"', "steps": {"a": "true"}}))
    code, out, err = c("t.step-log", t, ["check", "--json", "--full", "--then", "a"], dict(env, TMPDIR=td))
    os.chmod(td, 0o755)
    ds = docs(out) or [{}]
    steps = ((ds[0] or {}).get("then") or {}).get("steps") or []
    if not steps or steps[0].get("exit_code") != -1 or steps[0].get("ran") is not False:
        violations.append({"suite": "t.step-log", "violation": "ADR-101 step path not -1", "exit": code,
                           "stdout": out[:400], "stderr": err[:300], "argv": ["check", "--json", "--full", "--then", "a"]})
    counts["t.step-log.detail"] = f"exit {code}, step {steps[0] if steps else None}"


def concurrent():
    st = tempfile.mkdtemp(dir=WORK, prefix="state-par-")
    env = base_env(st)
    t1, t2 = tree("nocheck"), tree("pass")
    jobs = [(t1, ["check", "--json", "nosuchdir"]), (t1, ["check", "--json", "--full"]),
            (t2, ["check", "--json", "--full", "x.go"]), (t2, ["check", "--json", "../outside"])] * 12
    with cf.ThreadPoolExecutor(max_workers=8) as ex:
        futs = [ex.submit(run, ["-C", t] + a, t, env) for t, a in jobs]
        for (t, a), f in zip(jobs, futs):
            code, out, err = f.result()
            judge("c.parallel", a, t, env, code, out, err)


def fuzz():
    rnd = random.Random(SEED)
    st = tempfile.mkdtemp(dir=WORK, prefix="state-fz-")
    kinds = ["nocheck", "pass", "fail", "malformed", "big"]
    trees = {k: tree(k) for k in kinds}
    pool_paths = ["x.go", "x.go ", " a.txt", "a.txt", "../o", "/tmp", "missing.go", "help", "h", "sub/"]
    pool_flags = [["--json"], ["--full"], ["--then", "a"], ["--then", "bad"], ["--then", "nope"],
                  ["--then-sh", "true"], ["--then-sh", "exit 3"], ["--then-sh=x "], ["--bogus"], ["--"]]
    depths = [None, "7", "8", "9", "08", " 8", "-1", "abc", "+8"]
    for i in range(N):
        k = rnd.choice(kinds)
        t = trees[k]
        argv = ["check"]
        for _ in range(rnd.randint(0, 4)):
            argv += rnd.choice(pool_flags) if rnd.random() < 0.6 else [rnd.choice(pool_paths)]
        extra = {}
        dv = rnd.choice(depths)
        if dv is not None:
            extra["MRW_STEP_DEPTH"] = dv
        if rnd.random() < 0.15:
            extra["TMPDIR"] = os.path.join(WORK, "fz-gone")
        c("f.random", t, argv, base_env(st, extra))


def main():
    if not MRW:
        print("mrw not on PATH"); sys.exit(2)
    v = subprocess.run([MRW, "version"], capture_output=True, text=True).stdout.strip()
    print(f"mrw {v} · work {WORK} · seed {SEED} · fuzz N={N}")
    targeted(); concurrent(); fuzz()
    total = sum(x for x in counts.values() if isinstance(x, int))
    for k in sorted(counts):
        print(f"  {k:24} {counts[k]}")
    for k in sorted(COV, key=lambda k: -COV[k]):
        print(f"  cov {COV[k]:6}  {k}")
    print(f"runs {total} · violations {len(violations)}")
    seen = set()
    for vl in violations:
        key = (vl["suite"], vl["violation"].split(" ")[0] + vl["violation"][:40])
        if key in seen:
            continue
        seen.add(key)
        print(json.dumps(vl)[:900])
    if violations:
        path = os.path.join(tempfile.gettempdir(), f"mrw-stress-violations-{SEED}.json")
        json.dump(violations, open(path, "w"), indent=1)
        print(f"violations written to {path}")
    shutil.rmtree(WORK, ignore_errors=True)
    sys.exit(1 if violations else 0)


if __name__ == "__main__":
    main()
