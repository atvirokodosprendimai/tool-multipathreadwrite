#!/usr/bin/env python3
"""scripts/chaos.py — a randomised chaos pass over the mrw binary.

Run by hand, like scripts/break-campaign.sh; never in CI (it spends minutes).

    python3 scripts/chaos.py MRW WORKDIR [--seed N] [--scale K] [--race-strict]
    CORPUS=list.txt python3 scripts/chaos.py …   # trees built from real files

Each suite drives the BUILT binary and checks it against an independent model,
not against mrw's own output:
  read     served lines equal the model's lines (lines as the write engine
           numbers them: CRLF, CR-only, mixed), the right exit, the tree unchanged
  write    random plans, 35% deliberately corrupted (bad range, anchor, sha,
           unread file, path escape, overlap, rename onto/into the impossible…):
           a valid plan must leave exactly the model's tree; a corrupted one
           must exit non-zero and leave the tree byte-identical
  grep     --grep/--exclude hits equal a model of ADR-007's walk
  spec     hostile read specs: no panic, no hang, no read outside the root
  mutate   byte-mutated plans: a refused plan changes nothing
  race     8 concurrent writers on one file: counts lost updates (a known,
           accepted risk — BACKLOG "concurrent writes"); --race-strict fails on it
  symlink  no read, grep, write, unlink, rename or create reaches outside the root
  foreign  mutated apply_patch/search_replace/--files-from documents
  mcp      garbage JSON-RPC: the server keeps answering and never applies a
           write without an acknowledged read

Every failure is kept under WORKDIR/fail/<suite>-<n>/ with the command, stdin,
output and the tree as it was. mrw's state goes to WORKDIR/state, never the
user's. It found the CR-only read/write split (ADR-065) and the half-applied
rename (ADR-066) on 2026-09-24; a failure it reports is a LEAD — confirm it by
hand before calling it a defect, since the model can be wrong too.
"""
import json, os, random, re, shutil, subprocess, sys, threading, time, hashlib

MRW = os.path.abspath(sys.argv[1])
WORK = os.path.abspath(sys.argv[2])
# WORKDIR is created, filled and pruned (`fresh()` removes runs/ under it), so
# it must not sit inside this checkout: the harness never touches the tree.
# Compared by file IDENTITY, not by spelling: on a case-insensitive filesystem
# /users/x and /Users/x are one directory, and a path on another drive has no
# common prefix to compare. Walk WORKDIR's existing ancestors and ask each.
REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_p = os.path.realpath(WORK)
while True:
    if os.path.exists(_p) and os.path.samefile(_p, REPO):
        sys.exit(f"chaos.py: WORKDIR {WORK} is inside the checkout {REPO}; use a scratch directory")
    _up = os.path.dirname(_p)
    if _up == _p:
        break
    _p = _up
SEED = int(sys.argv[sys.argv.index("--seed") + 1]) if "--seed" in sys.argv else int(time.time())
SCALE = float(sys.argv[sys.argv.index("--scale") + 1]) if "--scale" in sys.argv else 1.0
rng = random.Random(SEED)
ENV = dict(os.environ, XDG_STATE_HOME=os.path.join(WORK, "state"))
FAILS, STATS = [], {}
os.makedirs(os.path.join(WORK, "fail"), exist_ok=True)


def rd(path, mode="rb"):
    with open(path, mode) as f:
        return f.read()


def wr(path, data, mode):
    with open(path, mode) as f:
        f.write(data)


def run(args, stdin=None, cwd=None, timeout=20):
    t0 = time.time()
    try:
        p = subprocess.run([MRW] + args, input=stdin, capture_output=True, cwd=cwd, env=ENV, timeout=timeout)
        return p.returncode, p.stdout.decode("utf-8", "replace"), p.stderr.decode("utf-8", "replace"), time.time() - t0
    except subprocess.TimeoutExpired:
        return "TIMEOUT", "", "", time.time() - t0


def snapshot(root):
    out = {}
    for dp, dns, fns in os.walk(root, followlinks=False):
        for n in fns + [d for d in dns if os.path.islink(os.path.join(dp, d))]:
            p = os.path.join(dp, n)
            rel = os.path.relpath(p, root)
            out[rel] = ("L:" + os.readlink(p)).encode() if os.path.islink(p) else rd(p)
        for d in dns:
            p = os.path.join(dp, d)
            if not os.listdir(p):
                out[os.path.relpath(p, root) + "/"] = b"<emptydir>"
    return out


def fail(suite, why, root=None, cmd=None, stdin=None, res=None, extra=None):
    n = len(FAILS)
    d = os.path.join(WORK, "fail", f"{suite}-{n}")
    os.makedirs(d, exist_ok=True)
    rec = {"suite": suite, "why": why, "seed": SEED, "cmd": cmd, "extra": extra}
    if res:
        rec.update(rc=res[0], stdout=res[1][-4000:], stderr=res[2][-4000:])
    if stdin is not None:
        wr(os.path.join(d, "stdin"), stdin if isinstance(stdin, bytes) else stdin.encode(), "wb")
    if root and os.path.isdir(root):
        shutil.copytree(root, os.path.join(d, "tree"), symlinks=True, dirs_exist_ok=True)
    wr(os.path.join(d, "case.json"), json.dumps(rec, indent=1, default=str), "w")
    FAILS.append((suite, why, d))


def generic(suite, res, root=None, cmd=None, stdin=None):
    """Invariants every invocation must hold."""
    STATS[suite] = STATS.get(suite, 0) + 1
    rc, out, err, dt = res
    if rc == "TIMEOUT":
        fail(suite, "hang: no exit within timeout", root, cmd, stdin, res); return False
    if rc not in (0, 1, 2, 3):
        fail(suite, f"exit code {rc} outside the contract", root, cmd, stdin, res); return False
    if "panic:" in err or "goroutine " in err or "panic:" in out:
        fail(suite, "panic", root, cmd, stdin, res); return False
    return True


# ---------- model of mrw's line conventions (internal/apply readLines/join) ----------
def eol_of(s):
    if "\n" not in s:
        return "\r" if "\r" in s else "\n"
    if "\r\n" in s and s.count("\r\n") == s.count("\n"):
        return "\r\n"
    return "\n"


def split(b):
    s = b.decode("utf-8", "surrogateescape")
    if not s:
        return [], "\n", False
    e = eol_of(s)
    final = s.endswith(e)
    if final:
        s = s[: -len(e)]
    return s.split(e), e, final


def rsplit_(b):
    """read's view since ADR-065 (v1.23.0): the write engine's lines."""
    return split(b)[0]


def join(lines, e, final):
    body = e.join(lines)
    if final and lines:
        body += e
    return body.encode("utf-8", "surrogateescape")


WORDS = ["alpha", "beta", "gamma", "TOKEN", "func Handle()", "  indented", "\ttabbed", "trailing  ",
         "ünïcødé", "漢字テキスト", "emoji 🙂🚀", "a|b|c", "x:1:2", "/re/gex/", "@@ not a header", "@@",
         "{", "}", "\"quoted\"", "back\\slash", "#comment", "- dash", "=eq=", "", "", "$", "end"]


def rand_line():
    if rng.random() < 0.02:
        return "L" * rng.randint(2000, 20000)
    return " ".join(rng.choice(WORDS) for _ in range(rng.randint(0, 4)))


def rand_content():
    k = rng.random()
    if k < 0.04:
        return b""
    n = rng.randint(1, 40)
    lines = [rand_line() for _ in range(n)]
    e = rng.choice(["\n"] * 8 + ["\r\n", "\r"])
    if e != "\n":
        lines = [l for l in lines if "\r" not in l]
    final = rng.random() < 0.8
    b = join(lines, e, final)
    if rng.random() < 0.03:
        b = b"\xef\xbb\xbf" + b
    if rng.random() < 0.03:
        b = b.replace(b"a", b"\xff", 1)  # invalid UTF-8
    return b


NAMES = ["a.txt", "b.go", "c.md", "notes.txt", "x y.txt", "ünï.txt", "-dash.txt", "k=v.txt", "d.yaml",
         "UP.TXT", "deep/one/two.txt", "sub/a.txt", "sub/b.go", "vendor/v.go", "build/g.go", ".hidden"]


# Iterate the file, not splitlines(): splitlines() also breaks on \v, \f and
# U+2028, which are legal in a path (Codex on #211).
if os.environ.get("CORPUS"):
    with open(os.environ["CORPUS"]) as _f:
        CORPUS = [l.strip() for l in _f]
else:
    CORPUS = None
CRLF_POOL = [p for p in CORPUS if b"\r\n" in rd(p)] if CORPUS else []


def make_tree(root, k=None):
    os.makedirs(root)
    if CORPUS:
        picks = rng.sample(CORPUS, k or rng.randint(2, 7))
        if CRLF_POOL and rng.random() < 0.5:
            picks[0] = rng.choice(CRLF_POOL)
        names = []
        for i, src in enumerate(picks):
            rel = f"c{i}/" + os.path.basename(src)
            os.makedirs(os.path.join(root, f"c{i}"), exist_ok=True)
            shutil.copyfile(src, os.path.join(root, rel))
            names.append(rel)
        return names
    names = rng.sample(NAMES, k or rng.randint(2, 7))
    for n in names:
        p = os.path.join(root, n)
        os.makedirs(os.path.dirname(p), exist_ok=True)
        wr(p, rand_content(), "wb")
    return names


def fresh(suite, i):
    r = os.path.join(WORK, "runs", f"{suite}-{i}")
    shutil.rmtree(r, ignore_errors=True)
    return r


# ---------- suite: reads served exactly ----------
def parse_read(out):
    served, cur = {}, None
    for line in out.split("\n"):
        m = re.match(r"^==> (.*?)  \d+L  \d+B  sha [0-9a-f]+$", line)
        if m:
            cur = m.group(1); served.setdefault(cur, {}); continue
        m = re.match(r"^\s*(\d+)\|(.*)$", line)
        if m and cur is not None:
            c = m.group(2)
            served[cur][int(m.group(1))] = c[1:] if c.startswith(" ") else c
    return served


def suite_read(n):
    for i in range(n):
        root = fresh("read", i)
        names = make_tree(root)
        f = rng.choice(names)
        b = rd(os.path.join(root, f))
        lines, e = rsplit_(b), eol_of(b.decode("utf-8", "surrogateescape"))
        L = len(lines)
        kind = rng.choice(["N", "N-M", "N-", "-M", "A,+N", "$", "list", "whole"])
        want, bad = set(), False
        a = rng.randint(1, max(1, L + 2)); z = rng.randint(1, max(1, L + 2))
        if kind == "N": spec = f"{f}:{a}"; want = {a}; bad = a > L
        elif kind == "N-M":
            a, z = min(a, z), max(a, z); spec = f"{f}:{a}-{z}"; want = set(range(a, min(z, L) + 1)); bad = a > L
        elif kind == "N-": spec = f"{f}:{a}-"; want = set(range(a, L + 1)); bad = a > L
        elif kind == "-M": spec = f"{f}:-{z}"; want = set(range(1, min(z, L) + 1)); bad = L == 0
        elif kind == "A,+N":
            k = rng.randint(1, 5); spec = f"{f}:{a},+{k}"; want = set(range(a, min(a + k, L) + 1)); bad = a > L
        elif kind == "$": spec = f"{f}:$"; want = {L} if L else set(); bad = L == 0
        elif kind == "list":
            r1, r2 = sorted([rng.randint(1, L + 1), rng.randint(1, L + 1)]); r3 = rng.randint(1, L + 1)
            spec = f"{f}:{r1}-{r2},{r3}"; want = set(range(r1, min(r2, L) + 1)) | ({r3} if r3 <= L else set())
            bad = r1 > L or r3 > L
        else: spec = f; want = set(range(1, L + 1))
        want = {x for x in want if 1 <= x <= L}
        before = snapshot(root)
        cmd = ["--root", root, "read", "--", spec]
        res = run(cmd)
        if not generic("read", res, root, cmd): continue
        rc, out, err, _ = res
        if snapshot(root) != before:
            fail("read", "a read changed the tree", root, cmd, None, res); continue
        if b and b"\xff" in b or b.startswith(b"\xef\xbb\xbf"):
            continue  # encoding choices: invariants only
        got = parse_read(out).get(f, {})
        if rc == 0 and bad:
            fail("read", f"exit 0 for an unservable {kind} spec", root, cmd, None, res, {"L": L}); continue
        if rc not in (0, 1) and L > 0:
            fail("read", f"exit {rc} for a well-formed spec", root, cmd, None, res, {"L": L}); continue
        if set(got) != want:
            fail("read", f"served lines {sorted(got)[:12]} != model {sorted(want)[:12]} ({kind})", root, cmd, None, res, {"L": L, "eol": repr(e)}); continue
        for ln, c in got.items():
            if c != lines[ln - 1] and len(lines[ln - 1]) < 2000:
                fail("read", f"line {ln} content differs: {c[:60]!r} vs {lines[ln-1][:60]!r}", root, cmd, None, res, {"eol": repr(e)}); break
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: write plans against a model ----------
def q(path):
    return f'"{path}"' if (" " in path or '"' in path) else path


def anchor_for(line):
    m = re.search(r"[A-Za-z]{3,}", line)
    return m.group(0) if m else None


def gen_plan(root, names):
    """Return (plan_text, model_snapshot) for a VALID plan, or None."""
    snap = snapshot(root)
    model = dict(snap)
    hunks = []
    files = [n for n in names if split(snap[n])[0] and b"\xff" not in snap[n]]
    rng.shuffle(files)
    pathops = 0
    for f in files[: rng.randint(1, 3)]:
        lines, e, final = split(snap[f])
        if e != "\n" and rng.random() < 0.5:
            pass
        if rng.random() < 0.12 and pathops == 0:
            pathops += 1
            if rng.random() < 0.5:
                hunks.append(f"@@ {q(f)} - unlink"); del model[f]
            else:
                dest = "moved/" + os.path.basename(f).replace(" ", "_")
                if dest in model: continue
                hunks.append(f"@@ {q(f)} - rename\n{dest}"); model[dest] = model.pop(f)
            continue
        L = len(lines)
        claimed = set()
        events = {}  # line -> (before_lines, action, after_lines) ; action None|'del'|('rep',body)
        for _ in range(rng.randint(1, 4)):
            op = rng.choice(["replace1", "replaceN", "insert-after", "insert-before", "delete"])
            a = rng.randint(1, L)
            z = min(L, a + rng.randint(0, 3)) if op in ("replaceN", "delete") else a
            if op == "replaceN" and z == a: z = min(L, a + 1)
            span = set(range(a, z + 1))
            if op == "insert-after": span |= {a + 1}   # the gap after a is also the gap before a+1
            if op == "insert-before": span |= {a - 1}
            if span & claimed: continue
            if op == "replaceN" and (z == a or not anchor_for(lines[a - 1])): continue
            claimed |= span
            body = [rand_line()[:200].replace("\r", "") for _ in range(rng.randint(1, 3))]
            need_decl = any(x == "" or x.startswith("@@") for x in body) or rng.random() < 0.3
            opts = ""
            if op == "replaceN":
                opts += f' anchor="{anchor_for(lines[a-1])}"'
            if need_decl:
                opts += f" body={len(body)}"
                if any(x.startswith("@@") for x in body): opts += " raw=true"
            addr = f"{a}" if z == a else f"{a}-{z}"
            name = {"replace1": "replace", "replaceN": "replace"}.get(op, op)
            if op == "delete":
                hunks.append(f"@@ {q(f)} {addr} delete"); events[a] = ("del", z)
            else:
                hunks.append(f"@@ {q(f)} {addr} {name}{opts}\n" + "\n".join(body))
                events[a] = (op, z, body)
        if not events: continue
        out, i = [], 1
        while i <= L:
            ev = events.get(i)
            if not ev: out.append(lines[i - 1]); i += 1; continue
            if ev[0] == "del": i = ev[1] + 1
            elif ev[0] in ("replace1", "replaceN"): out += ev[2]; i = ev[1] + 1
            elif ev[0] == "insert-after": out += [lines[i - 1]] + ev[2]; i += 1
            elif ev[0] == "insert-before": out += ev[2] + [lines[i - 1]]; i += 1
        model[f] = join(out, e, final if out else False) if out else b""
    if rng.random() < 0.2:
        newf = f"new/{rng.randint(0,999)}/n.txt"
        body = [rand_line()[:80].replace("\r", "") or "x" for _ in range(rng.randint(1, 3))]
        hunks.append(f"@@ {newf} 0 create body={len(body)}" + (" raw=true" if any(x.startswith('@@') for x in body) else "") + "\n" + "\n".join(body))
        model[newf] = ("\n".join(body) + "\n").encode()
    if not hunks: return None
    return "\n".join(hunks) + "\n", model


CORRUPT = ["range", "anchor", "sha", "lines", "unread", "escape", "abs", "overlap", "create-existing",
           "unlink-missing", "atat", "dupkey", "badop", "rename-onto", "rename-longdir", "rename-longleaf"]


def corrupt(plan, root, names):
    kind = rng.choice(CORRUPT)
    f = rng.choice(names)
    lines = split(rd(os.path.join(root, f)))[0]
    L = len(lines)
    bad = {
        "range": f"@@ {q(f)} {L + rng.randint(1, 50)} replace\nX",
        "anchor": f'@@ {q(f)} 1-{max(2, min(L, 2))} replace anchor="ZZZ_NOT_THERE"\nX',
        "sha": f"@@ {q(f)} 1 replace sha=deadbeef\nX",
        "lines": f"@@ {q(f)} 1 replace lines=7\nX",
        "unread": "@@ unread.txt 1 replace\nX",
        "escape": "@@ ../outside.txt 1 replace\nX",
        "abs": f"@@ {WORK}/outside.txt 1 replace\nX",
        "overlap": f"@@ {q(f)} 1 replace\nX\n@@ {q(f)} 1 delete",
        "create-existing": f"@@ {q(f)} 0 create body=1\nX",
        "unlink-missing": "@@ nope/missing.txt - unlink",
        "atat": f"@@ {q(f)} 1 replace\n@@ looks like a header",
        "dupkey": f'@@ {q(f)} 1 replace anchor="a" anchor="a"\nX',
        "badop": f"@@ {q(f)} 1 frobnicate\nX",
        "rename-onto": f"@@ {q(f)} - rename\n{rng.choice(names)}",
        "rename-longdir": f"@@ {q(f)} - rename\nnd/{'x' * 300}/f.txt",
        "rename-longleaf": f"@@ {q(f)} - rename\n{'y' * 300}",
    }[kind]
    if kind == "rename-onto" and (bad.endswith("\n" + f) or f"@@ {q(bad.split(chr(10))[1])} - unlink" in plan or f"- rename\n{bad.split(chr(10))[1]}" in plan):
        kind, bad = "unlink-missing", "@@ nope/missing.txt - unlink"
    # Only the plan's ends are safe places for the bad hunk: a body line may
    # itself begin with "@@", so splitting the text on "\n@@ " can land INSIDE
    # a counted (or raw=true) body, where mrw rightly writes the bad header as
    # content — three false alarms on 2026-09-24 came from exactly that.
    if rng.random() < 0.5:
        return bad + "\n" + plan, kind
    return plan + bad + "\n", kind


def suite_write(n):
    for i in range(n):
        root = fresh("write", i)
        names = make_tree(root)
        wr(os.path.join(WORK, "outside.txt"), "outside\n", "w")
        wr(os.path.join(root, "unread.txt"), "never served\nline2\n", "w")
        rc, out, err, _ = run(["--root", root, "read", "--"] + names)
        if rc not in (0, 1): fail("write", f"setup read exit {rc}", root, None, None, (rc, out, err, 0)); continue
        g = gen_plan(root, names)
        if not g: continue
        plan, model = g
        model = dict(model); model["unread.txt"] = snapshot(root)["unread.txt"]
        corrupted = rng.random() < 0.35
        kind = None
        if corrupted:
            plan, kind = corrupt(plan, root, names)
        before = snapshot(root)
        cmd = ["--root", root, "write", "-"]
        res = run(cmd, plan.encode("utf-8", "surrogateescape"))
        if not generic("write", res, root, cmd, plan): continue
        rc, out, err, _ = res
        after = snapshot(root)
        leftovers = [k for k in after if k not in before and k not in model and not k.endswith("/")]
        if corrupted:
            if rc == 0:
                fail("write", f"corrupted plan ({kind}) applied at exit 0", root, cmd, plan, res); continue
            if after != before:
                fail("write", f"failed plan ({kind}, exit {rc}) changed the tree: {sorted(set(after) ^ set(before)) or 'content'}", root, cmd, plan, res); continue
            if " ok " in out and kind not in ("atat", "dupkey", "badop"):
                if re.search(r"^ok ", out, re.M) and rc == 1:
                    fail("write", f"a hunk reported ok in a failed plan ({kind})", root, cmd, plan, res)
            if rd(os.path.join(WORK, "outside.txt"), "r") != "outside\n":
                fail("write", "file outside the root changed", root, cmd, plan, res)
        else:
            if rc != 0:
                fail("write", f"valid plan refused (exit {rc})", root, cmd, plan, res); continue
            if leftovers:
                fail("write", f"stray files left: {leftovers}", root, cmd, plan, res); continue
            diff = [k for k in set(after) | set(model) if after.get(k) != model.get(k) and not k.endswith("/")]
            if diff:
                k = diff[0]
                fail("write", f"tree != model at {k}", root, cmd, plan, res,
                     {"got": repr(after.get(k, b'<absent>')[:300]), "want": repr(model.get(k, b'<absent>')[:300])}); continue
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: --grep / --exclude walk against a model ----------
def gomatch(pat, s):
    rx = "".join("[^/]*" if c == "*" else "[^/]" if c == "?" else re.escape(c) for c in pat)
    return re.fullmatch(rx, s) is not None


def excluded(rel, ex):
    return any(gomatch(g, rel) or gomatch(g, os.path.basename(rel)) for g in ex)


def model_grep(root, pat, ex, paths):
    hits = set()
    starts = paths or ["."]
    for s in starts:
        full = os.path.join(root, s)
        if os.path.isfile(full):
            files = [s]
        else:
            files = []
            for dp, dns, fns in os.walk(full):
                keep = []
                for d in sorted(dns):
                    rel = os.path.normpath(os.path.relpath(os.path.join(dp, d), root))
                    if d == ".git" or excluded(rel, ex): continue
                    keep.append(d)
                dns[:] = keep
                for fn in fns:
                    rel = os.path.normpath(os.path.relpath(os.path.join(dp, fn), root))
                    if not excluded(rel, ex): files.append(rel)
        for rel in files:
            for i, l in enumerate(rsplit_(rd(os.path.join(root, rel))), 1):
                if pat in l: hits.add((os.path.normpath(rel), i))
    return hits


def suite_grep(n):
    for i in range(n):
        root = fresh("grep", i)
        names = make_tree(root, rng.randint(4, 9))
        os.makedirs(os.path.join(root, ".git"), exist_ok=True)
        wr(os.path.join(root, ".git", "HEAD"), "TOKEN in git\n", "w")
        ex = rng.sample(["vendor", "*.md", "sub", "build", "*.go", "deep/one", "one", "a.txt", ".hidden"], rng.randint(0, 3))
        cand = [n for n in names] + ["sub", "deep", "vendor"]
        paths = [p for p in rng.sample(cand, rng.randint(0, 2)) if os.path.exists(os.path.join(root, p))]
        pat = rng.choice(["TOKEN", "alpha", "Handle", "漢字", "@@"] + (["return", "function", "import", "TODO", "class", "const"] if CORPUS else []))
        cmd = ["--root", root, "read", "--grep", pat] + sum((["--exclude", g] for g in ex), []) + ["--"] + paths
        before = snapshot(root)
        res = run(cmd)
        if not generic("grep", res, root, cmd): continue
        rc, out, err, _ = res
        if snapshot(root) != before: fail("grep", "a grep changed the tree", root, cmd, None, res); continue
        if any(b"\xff" in v or v.startswith(b"\xef\xbb\xbf") for v in before.values()): continue
        want = model_grep(root, pat, ex, paths)
        got = {(os.path.normpath(f), ln) for f, m in parse_read(out).items() for ln, c in m.items() if pat in c}
        if got != want:
            fail("grep", f"hits differ: extra {sorted(got - want)[:5]} missing {sorted(want - got)[:5]}", root, cmd, None, res,
                 {"ex": ex, "paths": paths}); continue
        if (rc == 0) != bool(want):
            fail("grep", f"exit {rc} with {len(want)} expected hits", root, cmd, None, res); continue
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: hostile read specs ----------
SPECS = ["{f}:0", "{f}:-0", "{f}:5-3", "{f}:1,+-2", "{f}:$-1", "{f}:", ":", "{f}:1,,2", "{f}:99999999999999999999",
         "{f}:-99999999999999999999", "{f}:/(/", "{f}:/a/,/b/,/c/", "{f}:/", "{f}://", "{f}:1-$", "{f}:$-$",
         "{f}:1,+99999999999999999999", "../../etc/passwd", "/etc/passwd", "{f}:/(a+)+$/", "{f}:\x00", "",
         "{f}:1-2-3", "{f}:+1", "{f}:1,+", "{f}:,", "nonexist.txt:1", "{f}:/[/", "{f}:/\\/", "{f}:  1",
         "{f}:1 ", "{f}:٣", "{f}:1e3", "{f}:0x10", "{f}:/^/,+2", "{f}:$,+1", "{f}:-", "{f}:--1"]


def suite_specs(n):
    for i in range(n):
        root = fresh("spec", i)
        names = make_tree(root, 3)
        f = rng.choice(names)
        s = rng.choice(SPECS).replace("{f}", f)
        if rng.random() < 0.3:
            s = s + rng.choice([":", ",", "-", "/", "$", ",+1", "\t"])
        extra = rng.choice([[], ["-C", str(rng.choice([0, 1, 99999999999]))], ["--max-lines", str(rng.choice([0, 1, -1]))],
                            ["--stat"], ["--no-numbers"], ["-C", "-1"]])
        before = snapshot(root)
        if "\x00" in s:
            continue  # argv cannot carry NUL
        cmd = ["--root", root, "read"] + extra + [s]
        res = run(cmd, timeout=10)
        if not generic("spec", res, root, cmd): continue
        if snapshot(root) != before: fail("spec", "read changed tree", root, cmd, None, res)
        if "root:x:0:0" in res[1]: fail("spec", "served a file outside the root", root, cmd, None, res)
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: mutated plans ----------
def suite_mutate(n):
    for i in range(n):
        root = fresh("mut", i)
        names = make_tree(root)
        run(["--root", root, "read", "--"] + names)
        g = gen_plan(root, names)
        if not g: continue
        b = bytearray(g[0].encode("utf-8", "surrogateescape"))
        for _ in range(rng.randint(1, 6)):
            k = rng.random(); p = rng.randrange(len(b) + 1)
            if k < 0.3 and b: b[min(p, len(b) - 1)] = rng.randrange(256)
            elif k < 0.6: b[p:p] = rng.choice([b"@@ ", b"\n", b"\r", b'"', b"'", b"/", b" ", b"=", b"\x00", b"-", b"$", b"\xff"])
            elif k < 0.8 and b: del b[p:p + rng.randint(1, 8)]
            else: b[p:p] = b[max(0, p - 20):p]
        before = snapshot(root)
        cmd = ["--root", root, "write", "-"]
        res = run(cmd, bytes(b))
        if not generic("mutate", res, root, cmd, bytes(b)): continue
        after = snapshot(root)
        if res[0] != 0 and after != before:
            fail("mutate", f"exit {res[0]} but the tree changed", root, cmd, bytes(b), res); continue
        if res[0] == 0:
            stray = [k for k in after if k not in before and (".tmp" in k or ".mrw" in k or k.startswith("."))]
            if stray: fail("mutate", f"stray files {stray}", root, cmd, bytes(b), res)
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: concurrent writers on one file ----------
def suite_race(n):
    for i in range(n):
        root = fresh("race", i)
        os.makedirs(root)
        L = 16
        orig = [f"line {k}" for k in range(1, L + 1)]
        wr(os.path.join(root, "f.txt"), "\n".join(orig) + "\n", "w")
        run(["--root", root, "read", "f.txt"])
        W = 8
        results = [None] * W
        def w(j):
            results[j] = run(["--root", root, "write", "-"], f"@@ f.txt {j+1} replace\nwriter {j}\n".encode())
        ts = [threading.Thread(target=w, args=(j,)) for j in range(W)]
        [t.start() for t in ts]; [t.join() for t in ts]
        final = rd(os.path.join(root, "f.txt"), "r").split("\n")[:-1]
        for j, r in enumerate(results):
            generic("race", r, root)
        if len(final) != L:
            fail("race", f"line count {len(final)} != {L}", root, None, None, None, {"final": final}); continue
        lost = [j for j, r in enumerate(results) if r[0] == 0 and final[j] != f"writer {j}"]
        torn = [k for k, l in enumerate(final) if l != orig[k] and l != f"writer {k}"]
        STATS["race_lost"] = STATS.get("race_lost", 0) + len(lost)
        if lost and "--race-strict" in sys.argv:
            fail("race", f"{len(lost)} writer(s) exited 0 but their edit is gone: {lost}", root, None, None, None,
                 {"rcs": [r[0] for r in results], "final": final, "outs": [r[1][-200:] + r[2][-200:] for r in results]}); continue
        if torn: fail("race", f"torn lines {torn}", root)
        if lost: continue
        stray = [k for k in os.listdir(root) if k != "f.txt"]
        if stray: fail("race", f"stray files {stray}", root)
        STATS["race_applied"] = STATS.get("race_applied", 0) + sum(1 for r in results if r[0] == 0)
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: symlink escapes ----------
def suite_symlink():
    root = fresh("sym", 0); os.makedirs(root)
    out_dir = os.path.join(WORK, "outside-dir"); os.makedirs(out_dir, exist_ok=True)
    secret = os.path.join(out_dir, "secret.txt"); wr(secret, "SECRET-OUTSIDE\n", "w")
    os.symlink(secret, os.path.join(root, "link.txt"))
    os.symlink(out_dir, os.path.join(root, "linkdir"))
    wr(os.path.join(root, "in.txt"), "inside\n", "w")
    os.symlink("in.txt", os.path.join(root, "inlink.txt"))
    cases = [(["read", "link.txt"], "read via file link"), (["read", "linkdir/secret.txt"], "read via dir link"),
             (["read", "--grep", "SECRET"], "grep walk"), (["read", "--grep", "SECRET", "linkdir"], "grep named link dir")]
    for args, why in cases:
        res = run(["--root", root] + args)
        generic("symlink", res, root, args)
        if "SECRET-OUTSIDE" in res[1]:
            fail("symlink", f"{why}: served content from outside the root", root, args, None, res)
    for plan, why in [("@@ link.txt 1 replace\nPWNED\n", "write via file link"),
                      ("@@ linkdir/secret.txt 1 replace\nPWNED\n", "write via dir link"),
                      ("@@ link.txt - unlink\n", "unlink link"), ("@@ in.txt - rename\nlinkdir/moved.txt\n", "rename into link dir"),
                      ("@@ linkdir/new.txt 0 create body=1\nPWNED\n", "create via dir link")]:
        res = run(["--root", root, "write", "-"], plan.encode())
        generic("symlink", res, root, plan)
        if rd(secret, "r") != "SECRET-OUTSIDE\n" or os.path.exists(os.path.join(out_dir, "moved.txt")) or os.path.exists(os.path.join(out_dir, "new.txt")):
            fail("symlink", f"{why}: outside the root changed (exit {res[0]})", root, None, plan, res)
            wr(secret, "SECRET-OUTSIDE\n", "w")
            for x in ("moved.txt", "new.txt"):
                if os.path.exists(os.path.join(out_dir, x)): os.remove(os.path.join(out_dir, x))
            if not os.path.exists(os.path.join(root, "in.txt")): wr(os.path.join(root, "in.txt"), "inside\n", "w")


# ---------- suite: MCP garbage ----------
def suite_mcp(n):
    for i in range(n):
        root = fresh("mcp", i); names = make_tree(root, 3)
        p = subprocess.Popen([MRW, "--root", root, "mcp"], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE, env=ENV)
        msgs, expect = [], 0
        def send(obj_or_raw, has_id):
            nonlocal expect
            raw = obj_or_raw if isinstance(obj_or_raw, bytes) else json.dumps(obj_or_raw).encode()
            msgs.append(raw)
            if has_id: expect += 1
        send({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "chaos", "version": "0"}}}, True)
        send({"jsonrpc": "2.0", "method": "notifications/initialized"}, False)
        garbage = [b"not json", b"{", b"[]", b"{}", b'{"jsonrpc":"2.0"}', b"\xff\xfe", b"null", b"123", b'"str"',
                   b"[" * 5000 + b"]" * 5000, b'{"jsonrpc":"2.0","id":1,"method":"nope"}',
                   json.dumps({"jsonrpc": "2.0", "id": "s", "method": "tools/call", "params": {"name": "mrw_read", "arguments": {"specs": 5}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 2.5, "method": "tools/call", "params": {"name": "mrw_read", "arguments": {"specs": [1, None, {}]}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 10**30, "method": "tools/call", "params": {"name": "mrw_write", "arguments": {"plan": None}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": {"name": "mrw_write", "arguments": {"plan": f"@@ {names[0]} 1 replace\nX\n", "ack": "zzz"}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": {"name": "mrw_read", "arguments": {"specs": ["../../etc/passwd", "/etc/passwd"]}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 6, "method": "tools/call", "params": {"name": "mrw_read", "arguments": {"grep": "(", "exclude": [1]}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 7, "method": "tools/call", "params": {"name": "evil", "arguments": {}}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 8, "method": "tools/call", "params": None}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 9, "method": "tools/list", "params": {"cursor": "x" * 100000}}).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": None, "method": "tools/list"}).encode(),
                   json.dumps([{"jsonrpc": "2.0", "id": 11, "method": "tools/list"}]).encode(),
                   json.dumps({"jsonrpc": "2.0", "id": 12, "method": "tools/call", "params": {"name": "mrw_read", "arguments": {"specs": [names[0] + ":1-" + "9" * 30]}}}).encode()]
        for g in rng.sample(garbage, len(garbage)):
            msgs.append(g)
        msgs.append(json.dumps({"jsonrpc": "2.0", "id": 99, "method": "tools/list"}).encode())
        data = b"\n".join(msgs) + b"\n"
        try:
            out, err = p.communicate(data, timeout=20)
        except subprocess.TimeoutExpired:
            p.kill(); out, err = p.communicate()
            fail("mcp", "server hung", root, None, data, (None, out.decode("utf-8", "replace"), err.decode("utf-8", "replace"), 0)); continue
        STATS["mcp"] = STATS.get("mcp", 0) + 1
        o, e = out.decode("utf-8", "replace"), err.decode("utf-8", "replace")
        if "panic:" in e or "goroutine " in e:
            fail("mcp", "panic", root, None, data, (p.returncode, o, e, 0)); continue
        resp = []
        for line in o.splitlines():
            try: resp.append(json.loads(line))
            except ValueError: fail("mcp", f"non-JSON line on stdout: {line[:120]!r}", root, None, data, (p.returncode, o, e, 0)); break
        ids = [r.get("id") for r in resp if isinstance(r, dict)]
        if 99 not in ids:
            fail("mcp", "no answer to the final tools/list: the server stopped serving", root, None, data, (p.returncode, o, e, 0)); continue
        if "root:x:0:0" in o: fail("mcp", "served /etc/passwd", root, None, data, (p.returncode, o, e, 0))
        wr = [r for r in resp if isinstance(r, dict) and r.get("id") == 4]
        if rd(os.path.join(root, names[0])).startswith(b"X\n"):
            fail("mcp", "mrw_write applied with a bogus ack / no read", root, None, data, (p.returncode, o, e, 0))
        shutil.rmtree(root, ignore_errors=True)


# ---------- suite: foreign grammars and --files-from ----------
def suite_foreign(n):
    for i in range(n):
        root = fresh("foreign", i)
        names = [x for x in make_tree(root) if " " not in x]
        if not names: continue
        run(["--root", root, "read", "--"] + names)
        f = rng.choice(names)
        lines = split(rd(os.path.join(root, f)))[0] or [""]
        ln = rng.choice(lines)
        fmt = rng.choice(["apply_patch", "search_replace", "files-from"])
        if fmt == "apply_patch":
            doc = (f"*** Begin Patch\n*** Update File: {f}\n@@\n-{ln}\n+CHANGED\n*** End Patch\n")
        elif fmt == "search_replace":
            doc = f"{f}\n<<<<<<< SEARCH\n{ln}\n=======\nCHANGED\n>>>>>>> REPLACE\n"
        else:
            doc = "\n".join(rng.choice([f"{f}:1", f"{f}:/x/", "# c", "", "  ", f"{f}:$", "../x:1", "/etc/passwd", f"{f}:9999", "\t" + f]) for _ in range(rng.randint(1, 6))) + "\n"
        b = bytearray(doc.encode("utf-8", "surrogateescape"))
        for _ in range(rng.randint(0, 4)):
            p_ = rng.randrange(len(b) + 1)
            if rng.random() < 0.5 and b: del b[p_:p_ + rng.randint(1, 6)]
            else: b[p_:p_] = rng.choice([b"*** ", b"@@", b"\n", b"-", b"+", b"<<<<<<< SEARCH\n", b"=======\n", b">>>>>>> REPLACE\n", b"\xff", b"*** Delete File: " + f.encode() + b"\n", b"*** Move to: moved/q.txt\n"])
        before = snapshot(root)
        cmd = ["--root", root, "read", "--files-from", "-"] if fmt == "files-from" else ["--root", root, "write", f"--format={fmt}", "-"]
        res = run(cmd, bytes(b), timeout=10)
        if not generic("foreign", res, root, cmd, bytes(b)): continue
        after = snapshot(root)
        if res[0] != 0 and after != before:
            fail("foreign", f"{fmt}: exit {res[0]} but the tree changed", root, cmd, bytes(b), res); continue
        if fmt == "files-from" and after != before:
            fail("foreign", "a --files-from read changed the tree", root, cmd, bytes(b), res); continue
        if "root:x:0:0" in res[1]:
            fail("foreign", "served /etc/passwd", root, cmd, bytes(b), res); continue
        shutil.rmtree(root, ignore_errors=True)


def main():
    t0 = time.time()
    k = SCALE
    for name, fn, arg in [("read", suite_read, int(400 * k)), ("write", suite_write, int(500 * k)), ("grep", suite_grep, int(250 * k)),
                          ("spec", suite_specs, int(300 * k)), ("mutate", suite_mutate, int(300 * k)), ("race", suite_race, int(25 * k)),
                          ("symlink", lambda _: suite_symlink(), 0), ("foreign", suite_foreign, int(400 * k)), ("mcp", suite_mcp, int(20 * k))]:
        t = time.time(); fn(arg)
        print(f"{name:8s} done in {time.time()-t:5.1f}s  fails so far {len(FAILS)}", flush=True)
    by = {}
    for s, why, d in FAILS:
        key = (s, re.sub(r"\d+", "N", why)[:90])
        by.setdefault(key, []).append(d)
    print(json.dumps({"seed": SEED, "invocations": STATS, "seconds": round(time.time() - t0, 1), "failures": len(FAILS)}))
    for (s, why), ds in sorted(by.items()):
        print(f"FAIL [{s}] x{len(ds)}: {why}\n      e.g. {ds[0]}")
    sys.exit(1 if FAILS else 0)


main()
