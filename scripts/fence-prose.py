#!/usr/bin/env python3
"""fence-prose: every task fence's grep over a tracked file still says what it said.

A done task's Acceptance fence often greps prose: README.md, AGENTS.md, CONTRIBUTING.md, a usage
string. adr-lint never runs a fence, and ADR-053 freed README from contract.sh, so rewording the
prose turns a done task's fence red with nothing noticing — measured 2026-09-29: 8 clauses in 7 done
tasks were red on main (ADR-007/008/011/014/023/033 against README, and ADR-088 T2 would have gone red
against CONTRIBUTING in the change that added this check). Running whole fences takes hours, because
many run contract.sh; this runs only their grep clauses, each on its own, which takes about a second.

A clause is checked when it is a plain `grep` whose file operands are existing files in the tree.
A clause over /tmp, over a variable, or fed by a pipe is a check of a run's own output and is skipped.
A clause preceded by `!` must not match.

Usage: fence-prose.py [ROOT]        exit 0 when every checked clause holds, 1 when one does not
       fence-prose.py --self-test   proves a red clause is reported (a check that cannot fail is none)
"""
import glob
import os
import re
import shlex
import subprocess
import sys
import tempfile

FENCE = re.compile(r"^## Acceptance\s*$.*?^```(?:bash|sh|shell)\s*$(.*?)^```", re.S | re.M)
SPLIT = re.compile(r"&&|\|\||;|\n")


def clauses(fence):
    """Yield (negated, text, argv) for each plain grep clause of one fence.

    text is the clause as the fence wrote it and is what runs, through bash, so quoting means what
    bash makes of it (a backslash-escaped backtick inside double quotes is one backtick); argv is
    only shlex's reading of it, used to find the file operands."""
    for seg in SPLIT.split(fence.replace("\\\n", " ")):
        seg = seg.strip().lstrip("{(").strip()
        negated = seg.startswith("!")
        seg = seg.lstrip("!").strip()
        if not seg.startswith("grep "):
            continue
        try:
            argv = shlex.split(seg)
        except ValueError:
            continue
        if "|" in argv or any("$" in a or a.startswith("/tmp") for a in argv):
            continue
        yield negated, seg, argv


def check(root):
    """Return (checked, red) where red lists (task, clause) that no longer hold."""
    checked, red = 0, []
    for task in sorted(glob.glob(os.path.join(root, "docs/adr/ADR-*/tasks/T*.md"))):
        with open(task, encoding="utf-8") as fh:
            m = FENCE.search(fh.read())
        if not m:
            continue
        for negated, text, argv in clauses(m.group(1)):
            files = [a for a in argv[1:] if not a.startswith("-") and os.path.isfile(os.path.join(root, a))]
            if not files:
                continue
            checked += 1
            rc = subprocess.run(["bash", "-c", text], cwd=root, stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL).returncode
            if (rc == 0) == negated:
                red.append((os.path.relpath(task, root), ("! " if negated else "") + text))
    return checked, red


def self_test():
    """A tree with one clause that holds, one that does not and one negated: exactly one is red."""
    with tempfile.TemporaryDirectory() as root:
        os.makedirs(os.path.join(root, "docs/adr/ADR-001-x/tasks"))
        with open(os.path.join(root, "README.md"), "w", encoding="utf-8") as fh:
            fh.write("zero means zero\na `tick`\n")
        with open(os.path.join(root, "docs/adr/ADR-001-x/tasks/T1-x.md"), "w", encoding="utf-8") as fh:
            fh.write("## Acceptance\n\n```bash\n"
                     "grep -q 'zero means zero' README.md \\\n"
                     "  && grep -q 'a phrase nobody wrote' README.md \\\n"
                     "  && ! grep -q 'removed' README.md \\\n"
                     "  && grep -q \"a \\`tick\\`\" README.md\n```\n")
        checked, red = check(root)
        if checked != 4 or len(red) != 1 or "a phrase nobody wrote" not in red[0][1]:
            print(f"fence-prose self-test FAILED: checked {checked}, red {red}")
            return 1
    print("fence-prose self-test: a red clause is reported")
    return 0


def main():
    if sys.argv[1:] == ["--self-test"]:
        return self_test()
    root = sys.argv[1] if len(sys.argv) > 1 else "."
    checked, red = check(root)
    for task, clause in red:
        print(f"RED  {task}\n     {clause}")
    print(f"fence-prose: {checked} prose clause(s) checked, {len(red)} red")
    return 1 if red else 0


if __name__ == "__main__":
    sys.exit(main())
