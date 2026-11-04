#!/usr/bin/env python3
"""fence-prose: every task fence's grep over a tracked file still says what it said.

A done task's Acceptance fence often greps prose: README.md, AGENTS.md, CONTRIBUTING.md, a usage
string. adr-lint never runs a fence, and ADR-053 freed README from contract.sh, so rewording the
prose turns a done task's fence red with nothing noticing — measured 2026-09-29: 8 clauses in 7 done
tasks were red on main (ADR-007/008/011/014/023/033 against README, and ADR-088 T2 would have gone red
against CONTRIBUTING in the change that added this check). Running whole fences takes hours, because
many run contract.sh; this runs only their grep clauses, each on its own, which takes about a second.

A clause is checked when it is a plain `grep` whose file operands are existing files in the tree.
A clause over /tmp, over a variable bash would expand, or fed by a pipe is a check of a run's own
output and is skipped. A clause preceded by `!` must not match. Clauses are split only at UNQUOTED
`&&`, `||`, `;` and newlines, so a pattern holding one is one clause, not two broken halves.

EXEMPT names the clauses a later record deliberately freed, each with that record: they are still
counted and reported as exempt, never silently dropped. A task whose row in its tasks/README.md gives
a status other than `done` (`pending`, `partial`, `blocked`) is not finished, and its fence is red by
design until the work lands (.claude/rules/lifecycle.md §2), so its clauses are counted and reported
as not yet done, not run. A task with no such row is checked.

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

# (task file prefix, clause substring) -> the record that freed it. ADR-053 retired contract.sh's
# README heading and tutorial-phrase greps (§39's `### Use it from an MCP host` and its `mcpServers`
# block, §64's README greps for `,+N`): a tidy of a heading must not look like a product break. These
# are the same assertions in the task fences that shipped them (Codex, third review of #283).
EXEMPT = {
    ("docs/adr/ADR-010-", "### Use it from an MCP host"): "ADR-053",
    ("docs/adr/ADR-010-", '"command": "mrw"'): "ADR-053",
    ("docs/adr/ADR-026-", "',+N' README.md"): "ADR-053",
    # ADR-108 T10 bumped the ledger to #mrw-seen v3: a v2 span may have been issued by an MCP checkpoint
    # that spanned a sparse read's gaps. ADR-038 T1 pinned v2 as its own go/no-go.
    ("docs/adr/ADR-038-", "'#mrw-seen v2' internal/seen/seen.go"): "ADR-108",
    # ADR-113 moved the CLI write's sequence into internal/writer/flow.go, whose Land applies (applyCounted since ADR-127);
    # cmd/mrw/main.go and internal/mcp/tools.go reach it through writer.Prepare and Land. ADR-075 T1 pinned both sites.
    ("docs/adr/ADR-075-", "'writer\\.Apply(' cmd/mrw/main.go"): "ADR-113",
    ("docs/adr/ADR-075-", "'writer\\.Apply(' internal/mcp/tools.go"): "ADR-113",
}


def split_unquoted(text):
    """Split a fence at the shell operators bash splits at: unquoted, unescaped `&&`, `||`, `;` and
    newlines. A pattern such as 'a;b' is one clause (Codex, third review of #283)."""
    parts, cur, quote, i = [], [], "", 0
    while i < len(text):
        ch = text[i]
        if quote != "'" and ch == "\\" and i + 1 < len(text):
            cur.append(text[i:i + 2])
            i += 2
            continue
        if ch in "'\"" and quote in ("", ch):
            quote = "" if quote == ch else ch
        elif not quote and (text.startswith("&&", i) or text.startswith("||", i)):
            parts.append("".join(cur))
            cur, i = [], i + 2
            continue
        elif not quote and ch in ";\n":
            parts.append("".join(cur))
            cur, i = [], i + 1
            continue
        cur.append(ch)
        i += 1
    parts.append("".join(cur))
    return parts


def clauses(fence):
    """Yield (negated, text, argv) for each plain grep clause of one fence.

    text is the clause as the fence wrote it and is what runs, through bash, so quoting means what
    bash makes of it (a backslash-escaped backtick inside double quotes is one backtick); argv is
    only shlex's reading of it, used to find the file operands."""
    for seg in split_unquoted(fence.replace("\\\n", " ")):
        seg = seg.strip().lstrip("{(").strip()
        negated = seg.startswith("!")
        seg = seg.lstrip("!").strip()
        if not seg.startswith("grep "):
            continue
        try:
            argv = shlex.split(seg)
        except ValueError:
            continue
        if "|" in argv or expands(seg) or any(a.startswith("/tmp") for a in argv):
            continue
        yield negated, seg, argv


def expands(seg):
    """True when seg has a `$` bash would expand: outside single quotes. A `$` inside single quotes
    is a regex end anchor, and skipping those skipped three real checks (Codex, review of #283)."""
    quote, escaped = "", False
    for ch in seg:
        if escaped:
            escaped = False
        elif ch == "\\" and quote != "'":
            escaped = True
        elif ch in "'\"" and quote in ("", ch):
            quote = "" if quote == ch else ch
        elif ch == "$" and quote != "'":
            return True
    return False


def exempt(task, text):
    """The record that freed this clause, or ""."""
    for (prefix, needle), record in EXEMPT.items():
        if task.startswith(prefix) and needle in text:
            return record
    return ""


STATUSES = ("pending", "partial", "blocked", "done")


def status_of(task):
    """The status the task's row in its tasks/README.md gives, or "" when there is no such row."""
    tid = os.path.basename(task).split("-", 1)[0].removesuffix(".md")
    try:
        with open(os.path.join(os.path.dirname(task), "README.md"), encoding="utf-8") as fh:
            rows = fh.read().splitlines()
    except OSError:
        return ""
    for row in rows:
        cells = [c.strip() for c in row.strip().strip("|").split("|")]
        if row.lstrip().startswith("|") and cells and cells[0] == tid:
            return next((c for c in cells[1:] if c in STATUSES), "")
    return ""


def check(root):
    """Return (checked, red, exempted, unfinished): red, exempted and unfinished list (task, clause[, record])."""
    checked, red, exempted, unfinished = 0, [], [], []
    for task in sorted(glob.glob(os.path.join(root, "docs/adr/ADR-*/tasks/T*.md"))):
        with open(task, encoding="utf-8") as fh:
            m = FENCE.search(fh.read())
        if not m:
            continue
        rel = os.path.relpath(task, root)
        finished = status_of(task) in ("", "done")
        for negated, text, argv in clauses(m.group(1)):
            files = [a for a in argv[1:] if not a.startswith("-") and os.path.isfile(os.path.join(root, a))]
            if not files:
                continue
            if not finished:
                unfinished.append((rel, text))
                continue
            record = exempt(rel, text)
            if record:
                exempted.append((rel, text, record))
                continue
            checked += 1
            rc = subprocess.run(["bash", "-c", text], cwd=root, stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL).returncode
            if (rc == 0) == negated:
                red.append((rel, ("! " if negated else "") + text))
    return checked, red, exempted, unfinished


def self_test():
    """A tree whose fence holds, misses, negates, quotes a backtick, anchors with a quoted `$`, and
    quotes a `;` and a `&&`: exactly the missing clause and the two quoted-operator misses are red,
    and every clause is checked. A pending task's red clause is counted as not yet done and not run;
    a task its README lists as done, and one it does not list, are checked."""
    with tempfile.TemporaryDirectory() as root:
        os.makedirs(os.path.join(root, "docs/adr/ADR-001-x/tasks"))
        with open(os.path.join(root, "README.md"), "w", encoding="utf-8") as fh:
            fh.write("zero means zero\na `tick`\n")
        with open(os.path.join(root, "docs/adr/ADR-001-x/tasks/T1-x.md"), "w", encoding="utf-8") as fh:
            fh.write("## Acceptance\n\n```bash\n"
                     "grep -q 'zero means zero' README.md \\\n"
                     "  && grep -q 'a phrase nobody wrote' README.md \\\n"
                     "  && ! grep -q 'removed' README.md \\\n"
                     "  && grep -q \"a \\`tick\\`\" README.md \\\n"
                     "  && grep -q 'zero$' README.md \\\n"
                     "  && grep -q 'absent;needle' README.md \\\n"
                     "  && grep -q 'absent&&needle' README.md\n```\n")
        for tid, clause in (("T2", "grep -q 'pending miss' README.md"), ("T3", "grep -q 'unlisted miss' README.md")):
            with open(os.path.join(root, f"docs/adr/ADR-001-x/tasks/{tid}-x.md"), "w", encoding="utf-8") as fh:
                fh.write(f"## Acceptance\n\n```bash\n{clause}\n```\n")
        with open(os.path.join(root, "docs/adr/ADR-001-x/tasks/README.md"), "w", encoding="utf-8") as fh:
            fh.write("| Task | Goal | Status |\n|---|---|---|\n| T1 | x | done |\n| T2 | x | pending |\n")
        checked, red, _, unfinished = check(root)
        missed = sorted(c for _, c in red)
        want = ["grep -q 'a phrase nobody wrote' README.md", "grep -q 'absent&&needle' README.md",
                "grep -q 'absent;needle' README.md", "grep -q 'unlisted miss' README.md"]
        if checked != 8 or missed != want or len(unfinished) != 1:
            print(f"fence-prose self-test FAILED: checked {checked}, red {missed}, not yet done {len(unfinished)}")
            return 1
    print("fence-prose self-test: every red clause is reported, quoted operators included; a pending task's is not run")
    return 0


def main():
    if sys.argv[1:] == ["--self-test"]:
        return self_test()
    root = sys.argv[1] if len(sys.argv) > 1 else "."
    checked, red, exempted, unfinished = check(root)
    for task, clause in red:
        print(f"RED  {task}\n     {clause}")
    print(f"fence-prose: {checked} prose clause(s) checked, {len(red)} red, "
          f"{len(exempted)} exempt ({', '.join(sorted({r for _, _, r in exempted})) or 'none'}), "
          f"{len(unfinished)} in task(s) not yet done, not run")
    return 1 if red else 0


if __name__ == "__main__":
    sys.exit(main())
