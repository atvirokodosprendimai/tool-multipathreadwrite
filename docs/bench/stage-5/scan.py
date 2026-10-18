"""ADR-162 Decision 8: scan every transcript in DIR for the dataset's and selections' names and the
instance's FAIL_TO_PASS test names that its own problem statement does not already give. Exit 0 clean,
1 on any hit (each printed), 2 on a usage or read error.

Amended after block 1 on Zy's ruling of 2026-10-18 ("Continue, record it"):
- a FAIL_TO_PASS name the instance's base commit already contains is skipped, since the agent's own
  tree shows it (block 1: 23 sympy hits, every one in the base commit);
- a name the base lacks, whose first appearance in the transcript is text the agent itself wrote
  (a tool input or its own message), is printed as RECALL and is not a hit: the model reproduced an
  upstream test from memory (block 1: 12, all Opus, in A, B and P alike). A first appearance in tool
  output or anywhere else stays a HIT."""
import json
import pathlib
import subprocess
import sys

sys.excepthook = lambda kind, value, tb: (print(f"scan.py: {kind.__name__}: {value}", file=sys.stderr), sys.exit(2))

if len(sys.argv) != 4:
    print("usage: scan.py DATASET DIR REPOS", file=sys.stderr)
    sys.exit(2)
dataset, out, repos = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]), pathlib.Path(sys.argv[3])
rows = {r["instance_id"]: r for r in json.loads(dataset.read_text())}
fixed = [dataset.name, str(dataset.parent), "swe-probe", "huggingface", "selection-adr-162", "selection-adr-160"]


def key(name):
    """The test's own identifier: `test_x (mod.Class)` and `path::Class::test_x[param]` both give test_x.
    The longer forms appear only in a test runner's output, so they say nothing about who wrote the name."""
    return name.split(" (")[0].split("::")[-1].split("[")[0]


def in_base(row, name):
    repo = repos / row["repo"].replace("/", "__")
    p = subprocess.run(["git", "-C", str(repo), "grep", "-q", "-F", name, row["base_commit"]], capture_output=True)
    if p.returncode > 1:
        raise RuntimeError(f"git grep in {repo}: {p.stderr.decode(errors='replace').strip()}")
    return p.returncode == 0


def strings(v):
    if isinstance(v, str):
        yield v
    elif isinstance(v, dict):
        for x in v.values():
            yield from strings(x)
    elif isinstance(v, list):
        for x in v:
            yield from strings(x)


def written_first(events, name):
    """True when the name's first appearance is in what the agent wrote, False otherwise."""
    for o in events:
        if o.get("type") == "assistant":
            for c in o.get("message", {}).get("content", []):
                if c.get("type") in ("tool_use", "text") and any(name in s for s in strings(c.get("input", c.get("text")))):
                    return True
        if any(name in s for s in strings(o)):
            return False
    return False


hits = recalls = 0
files = sorted(out.glob("*.transcript.jsonl"))
for t in files:
    instance = t.name.split(".")[0]
    row = rows.get(instance)
    if row is None:
        print(f"{t.name}: instance {instance} is not in the dataset", file=sys.stderr)
        sys.exit(2)
    f2p = row["FAIL_TO_PASS"]
    names = [n for n in (json.loads(f2p) if isinstance(f2p, str) else f2p) if n not in row["problem_statement"]]
    text = t.read_text(errors="replace")
    events = [json.loads(line) for line in text.splitlines() if line.startswith("{")]
    for needle in fixed:
        if needle and needle in text:
            hits += 1
            print(f"HIT {t.name}: {needle}")
    for name in names:
        if key(name) not in text or in_base(row, key(name)):
            continue
        if written_first(events, key(name)):
            recalls += 1
            print(f"RECALL {t.name}: {name}")
        else:
            hits += 1
            print(f"HIT {t.name}: {name}")
print(f"scanned {len(files)} transcript(s), {hits} hit(s), {recalls} recall(s)")
sys.exit(1 if hits else 0)
