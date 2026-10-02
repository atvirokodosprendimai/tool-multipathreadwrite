#!/usr/bin/env python3
"""Replay real history as replaces and count how often each ADR-119 hint fires.

ADR-119 T2. Every modification hunk of `git diff -U0 k^ k`, over the non-merge history of each
repository named, is taken as the `replace` a caller would have sent: the parent's lines as the
original, the hunk's old range as the address, its added lines as the body. A committed hunk is
taken to be a correct edit, so a hint that fires on one is a false positive.

The bar is the one BACKLOG registered before this ran ("Pre-registered for ADR-119"): under 5% of
replaces in every language bucket with at least 50 of them, and the field fixtures caught. The
REGISTERED rows are the verdict; every other row is a variant measured beside it so a decision to
change a heuristic has numbers, and such a change is measured again before it ships.

Without --mrw the hints are computed here, by the registered definitions. With --mrw BIN each
replace also goes through `BIN write --dry-run --json` in a scratch root and the receipt's keys
are compared with the computed ones, so the run proves the binary agrees with what was measured.

Usage: stress.py [--cap N] [--max-commits N] [--json FILE] [--mrw BIN] DIR...
Each DIR's children that hold a `.git` directory are the corpus; a worktree (`.git` file) is
skipped and counted, since it repeats its main checkout's history.
"""

import argparse
import json
import os
import random
import re
import subprocess
import sys
import tempfile
import time

PROSE = (".md", ".markdown", ".txt", ".rst", ".adoc")  # internal/apply IsProse
SKIP_PARTS = {"vendor", "node_modules", "dist", "build", ".next", "third_party", "testdata", "target"}
SKIP_NAMES = {"go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "Cargo.lock",
              "composer.lock", "poetry.lock", "Gemfile.lock", "uv.lock", "bun.lockb"}
SKIP_SUFFIX = (".min.js", ".min.css", ".pb.go", "_gen.go", ".gen.go", ".snap", ".svg", ".map",
               ".lock", ".golden", ".jsonl")
HUNK = re.compile(r"^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@")
CLOSER_TOKEN = re.compile(r"^(`{3,}|~{3,}|[}\])]+[;,)]*|</[\w.-]+>|@end\w*|end|fi|done|esac)$")


def git(repo, *args):
    p = subprocess.run(["git", "-C", repo, *args], capture_output=True, timeout=60)
    return p.stdout if p.returncode == 0 else None


def split(text):
    lines = text.split("\n")
    if lines and lines[-1] == "":
        lines.pop()
    return lines


def skipped(path):
    parts = path.split("/")
    name = parts[-1]
    return (any(p in SKIP_PARTS for p in parts[:-1]) or name in SKIP_NAMES
            or name.endswith(SKIP_SUFFIX) or ".generated." in name)


def bucket(path):
    name = path.rsplit("/", 1)[-1].lower()
    if name.endswith(".blade.php"):
        return "blade"
    ext = os.path.splitext(name)[1]
    return {".yaml": ".yml", ".jsx": ".js", ".mjs": ".js", ".cjs": ".js", ".tsx": ".ts",
            ".htm": ".html", ".markdown": ".md"}.get(ext, ext or "(none)")


def nonblank(lines):
    return [s for s in lines if s.strip()]


def ws(s):
    return s[: len(s) - len(s.lstrip(" \t"))]


def closer(orig, end, body, k=1, trim=False, token=False):
    """The registered closer is k=1, exact: the line right after the range exists, is not blank,
    and equals the body's last non-blank line. k>1 looks at the next k non-blank lines instead;
    token=True also requires that line to look like a closer."""
    nb = nonblank(body)
    if not nb:
        return False
    last = nb[-1].strip() if trim else nb[-1]
    if token and not CLOSER_TOKEN.match(nb[-1].strip()):
        return False
    after = orig[end:]
    if k == 1:
        if not after or not after[0].strip():
            return False
        cand = [after[0]]
    else:
        cand = nonblank(after)[:k]
    return any((c.strip() if trim else c) == last for c in cand)


def indent(path, replaced, body, first_only=False):
    """The registered indent: on a non-prose file, the body's first non-blank line's leading
    whitespace differs from the first replaced line's, or its last from the last's. Blank replaced
    lines carry no indentation to compare, so the first and last NON-BLANK replaced lines stand."""
    if path.lower().endswith(PROSE):
        return False
    b, r = nonblank(body), nonblank(replaced)
    if not b or not r:
        return False
    if ws(b[0]) != ws(r[0]):
        return True
    return not first_only and ws(b[-1]) != ws(r[-1])


def hints(path, orig, start, end, body):
    replaced = orig[start - 1:end]
    return {
        "closer": closer(orig, end, body),                      # REGISTERED
        "closer_trim": closer(orig, end, body, trim=True),
        "closer_k4_trim": closer(orig, end, body, k=4, trim=True),
        "closer_k4_token": closer(orig, end, body, k=4, trim=True, token=True),  # SHIPPED (amended bar)
        "closer_k4_token_ne": closer(orig, end, body, k=4, trim=True, token=True) and not ends_alike(replaced, body),  # SHIPPED (run 3)
        "closer_k8_token": closer(orig, end, body, k=8, trim=True, token=True),
        "indent": indent(path, replaced, body),                 # REGISTERED
        "indent_first": indent(path, replaced, body, first_only=True),
    }


def ends_alike(replaced, body):
    """True when the replaced range already ended in the line the body ends in, trimmed: a replace THROUGH its
    own closer, which leaves no duplicate (BACKLOG, "Amended after review")."""
    r, b = nonblank(replaced), nonblank(body)
    return bool(r and b) and r[-1].strip() == b[-1].strip()


def through_closer(orig, end, body):
    """The same edit replayed through the first closer-shaped line among the next 4 non-blank original lines: the
    range ends there and the body carries the same unchanged lines, as a caller taught to replace through the
    closer writes it. None when no closer follows within 4."""
    seen = 0
    for j in range(end, len(orig)):
        t = orig[j].strip()
        if not t:
            continue
        if CLOSER_TOKEN.match(t):
            return j + 1, body + orig[end:j + 1]
        seen += 1
        if seen == 4:
            return None
    return None

REGISTERED = ("closer", "indent")
# SHIPPED maps what the binary reports to the variant that ships (BACKLOG, "Amended after the first
# measurement"): --mrw compares the binary's key against it, so the run proves they agree.
SHIPPED = {"closer": "closer_k4_token_ne"}
VARIANTS = ("closer", "closer_trim", "closer_k4_trim", "closer_k4_token", "closer_k4_token_ne", "closer_k8_token",
            "indent", "indent_first")


def fixtures():
    """The five field failures, built to the offsets BACKLOG records ("From the field reports"):
    Blade replaced 7 wrote 7-9 orphan at 10; HTML replaced 5 wrote 5-7 orphan at 8; YAML block
    scalar replaced 8 wrote 8-10 orphan at 11; markdown fence replaced 333 wrote 333-336 orphan at
    340; and the Ansible task-level when: re-indented by two spaces. The orphan's new line number
    fixes which original line it is; the text around it is a reconstruction."""
    pad = lambda n: ["line %d of prose." % i for i in range(1, n + 1)]
    blade = (["<div>", "  <h1>{{ $title }}</h1>", "", "  <ul>", "  @foreach ($xs as $x)", "  @endforeach",
              "    @if ($items)", "    @endif", "  </ul>", "</div>"],
             7, 7, ["    @if ($items->isNotEmpty())", "        <li>{{ $items->first() }}</li>", "    @endif"],
             "closer")
    html = (["<main>", "  <h1>Title</h1>", "  <p>Intro</p>", "", '  <div class="card">', "  </div>", "</main>"],
            5, 5, ['  <div class="card card-wide">', "    <p>Body</p>", "  </div>"], "closer")
    md = (pad(332) + ["```bash", "make build", "make test", "make install", "```"] + pad(3),
          333, 333, ["```bash", "go build ./...", "go test ./...", "```"], "closer")
    yml = (["name: ci", "on: push", "jobs:", "  build:", "    runs-on: ubuntu-latest", "    steps:",
            "      - uses: actions/checkout@v4", "      - run: |", "          make build", "          make test",
            "      - name: next", "        run: echo done"],
           8, 8, ["      - run: |", "          go build ./...", "          go vet ./..."], "indent")
    ans = (["- name: install nginx", "  ansible.builtin.apt:", "    name: nginx", '  when: ansible_os_family == "Debian"'],
           4, 4, ['    when: ansible_os_family == "Debian"'], "indent")
    return {"blade.blade.php": blade, "page.html": html, "README.md": md, "ci.yml": yml, "tasks.yml": ans}


def corpus(dirs):
    repos, worktrees = [], []
    for d in dirs:
        for name in sorted(os.listdir(d)):
            p = os.path.join(d, name)
            g = os.path.join(p, ".git")
            if os.path.isdir(g):
                repos.append(p)
            elif os.path.isfile(g):
                worktrees.append(p)
    return repos, worktrees


def replay(repo, cap, max_commits, rng, mrw):
    out = git(repo, "log", "--no-merges", "--format=%H")
    commits = out.decode().split() if out else []
    rng.shuffle(commits)
    rows, mismatched, disagreed, base = [], 0, [], 0  # base: -U0 replaces, the unit --cap counts
    for k in commits[:max_commits]:
        if base >= cap:
            break
        diff = git(repo, "diff", "-U0", "--no-color", "--no-ext-diff", "--no-renames",
                   "--diff-filter=M", k + "^", k)
        if diff is None:
            continue
        try:
            text = diff.decode("utf-8")
        except UnicodeDecodeError:
            continue
        path, orig, hunk = None, None, None
        for line in text.split("\n") + ["@@END"]:
            if hunk and not (line.startswith("+") or line.startswith("-") or line.startswith("\\")):
                start, n, body, old = hunk
                hunk = None
                if orig is not None and base < cap:
                    end = start + n - 1
                    if orig[start - 1:end] != old:
                        mismatched += 1
                    else:
                        h = hints(path, orig, start, end, body)
                        base += 1
                        rows.append((bucket(path), h))
                        replays = [(start, end, body, h)]
                        thru = through_closer(orig, end, body)
                        if thru:
                            # Counted in its own bucket, so the bar applies to each replay separately.
                            h2 = hints(path, orig, start, thru[0], thru[1])
                            rows.append((bucket(path) + " thru", h2))
                            replays.append((start, thru[0], thru[1], h2))
                        if mrw:
                            for s, e, bd, hh in replays:
                                got = through_binary(mrw, path, orig, s, e, bd)
                                want = {k: hh[v] for k, v in SHIPPED.items()}
                                if got != want:
                                    disagreed.append((repo, k, path, s, e, got, want))
            if line.startswith("diff --git "):
                path, orig = None, None
            elif line.startswith("+++ b/"):
                path = line[6:]
                orig = None
                if not skipped(path):
                    blob = git(repo, "show", k + "^:" + path)
                    if blob is not None and len(blob) <= 1 << 20 and b"\0" not in blob[:8192]:
                        try:
                            orig = split(blob.decode("utf-8"))
                        except UnicodeDecodeError:
                            orig = None
            elif line.startswith("@@ ") and path:
                m = HUNK.match(line)
                if m:
                    a, b, d = int(m.group(1)), int(m.group(2) or 1), int(m.group(4) or 1)
                    if b > 0 and d > 0:
                        hunk = (a, b, [], [])
            elif hunk and line.startswith("+"):
                hunk[2].append(line[1:])
            elif hunk and line.startswith("-"):
                hunk[3].append(line[1:])
    return rows, mismatched, disagreed


def through_binary(mrw, path, orig, start, end, body):
    """The registered keys as the binary reports them, from a dry run in a scratch root. The
    original is written there and read through mrw first, so the replace is licensed."""
    with tempfile.TemporaryDirectory() as root:
        name = "f" + os.path.splitext(path)[1] if not path.endswith(".blade.php") else "f.blade.php"
        with open(os.path.join(root, name), "w", newline="") as f:
            f.write("\n".join(orig) + "\n")
        env = dict(os.environ, XDG_STATE_HOME=os.path.join(root, ".state"))
        subprocess.run([mrw, "--root", root, "read", name], capture_output=True, env=env, timeout=60)
        addr = str(start) if start == end else "%d-%d" % (start, end)
        # anchor= is required only on a multi-line replace (ADR-035). Its text is taken raw — mrw does not
        # unescape it, so a JSON-quoted anchor of a non-ASCII line never matched — and the longest piece of
        # the line holding no quote or backslash stands in, since a quoted anchor cannot carry those.
        guard = ""
        if start != end:
            piece = max(re.split(r'["\\]', orig[start - 1].strip()), key=len).strip()
            guard = ' anchor="%s"' % piece
        plan = "@@ %s %s replace%s body=%d\n%s\n" % (name, addr, guard, len(body), "\n".join(body))
        p = subprocess.run([mrw, "--root", root, "write", "--dry-run", "--json", "--no-check", "-"],
                           input=plan.encode(), capture_output=True, env=env, timeout=60)
        try:
            hunks = json.loads(p.stdout).get("hunks") or [{}]
        except ValueError:
            return {"error": p.stderr.decode()[-200:]}
        return {"closer": bool(hunks[0].get("closer"))}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--cap", type=int, default=400, help="replaces per repository")
    ap.add_argument("--max-commits", type=int, default=3000, help="commits sampled per repository")
    ap.add_argument("--seed", type=int, default=119, help="the commit shuffle; a fresh sample takes a new one")
    ap.add_argument("--json", help="write the counts here too")
    ap.add_argument("--mrw", help="also drive this binary and compare its keys")
    ap.add_argument("dirs", nargs="+")
    a = ap.parse_args()

    fx = {}
    for name, (orig, s, e, body, kind) in fixtures().items():
        fx[name] = {"kind": kind, **hints(name, orig, s, e, body)}

    rng = random.Random(a.seed)
    repos, worktrees = corpus(a.dirs)
    per, total, mism, disagree, t0 = {}, 0, 0, [], time.time()
    for repo in repos:
        rows, m, d = replay(repo, a.cap, a.max_commits, rng, a.mrw)
        mism += m
        disagree += d
        total += len(rows)
        for b, h in rows:
            c = per.setdefault(b, {"n": 0, **{v: 0 for v in VARIANTS}})
            c["n"] += 1
            for v in VARIANTS:
                c[v] += h[v]
        print("%-60s %4d replaces" % (repo, len(rows)), file=sys.stderr)

    print("corpus: %d repositories (%d worktrees skipped), %d replaces, %d hunks whose old lines did not "
          "match the parent (skipped), %.0fs" % (len(repos), len(worktrees), total, mism, time.time() - t0))
    print()
    print("fixtures (kind = which heuristic the bar asks to catch it):")
    for name, r in fx.items():
        print("  %-16s %-7s %s" % (name, r["kind"], " ".join("%s=%d" % (v, r[v]) for v in VARIANTS)))
    print()
    print("false-positive rate per bucket, % of replaces (* = bucket under 50, outside the bar; "
          "REGISTERED: closer, indent):")
    print("  %-8s %6s " % ("bucket", "n") + " ".join("%15s" % v for v in VARIANTS))
    for b, c in sorted(per.items(), key=lambda kv: -kv[1]["n"]):
        mark = "*" if c["n"] < 50 else " "
        print("  %-8s %5d%s " % (b, c["n"], mark) + " ".join("%14.2f%%" % (100.0 * c[v] / c["n"]) for v in VARIANTS))
    print()
    for v in VARIANTS:
        over = [b for b, c in per.items() if c["n"] >= 50 and 100.0 * c[v] / c["n"] >= 5.0]
        print("  %-16s %s" % (v, "under 5% in every bucket of 50+" if not over else "OVER in " + ", ".join(sorted(over))))
    if a.mrw:
        print("\nbinary agreement: %d of %d replaces disagreed" % (len(disagree), total))
        for row in disagree[:20]:
            print("  ", row)
    if a.json:
        with open(a.json, "w") as f:
            json.dump({"repos": len(repos), "worktrees": len(worktrees), "replaces": total,
                       "mismatched": mism, "fixtures": fx, "buckets": per,
                       "disagreed": len(disagree)}, f, indent=1)


if __name__ == "__main__":
    main()
