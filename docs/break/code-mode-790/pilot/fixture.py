"""The ADR-218 pilot fixture: 12 files of 60 lines, and the tree each task must leave.

    fixture.py make DIR [TASK]   write the base tree, or the tree task 1-5 must leave
    fixture.py sha DIR           sha256 over every pkg/*.go, path then bytes, in name order

Nothing here is compiled: the files only look like Go, so an edit has closers to trip on.
"""

import hashlib
import pathlib
import sys

FILES = range(1, 13)


def module(n, limit=None, clamp=None, total=None, call="scale"):
    """Return the 60 lines of module n; the keyword arguments are what a task changes."""
    nn = f"{n:02d}"
    limit = n * 10 if limit is None else limit
    clamp = clamp or [
        f"    y := x + {n}",
        f"    if y > Limit{nn} {{",
        f"        y = Limit{nn}",
        "    }",
        "    return y",
    ]
    total = total or [
        "    total := 0",
        "    for _, x := range xs {",
        f"        total += {call}(x)",
        "    }",
        "    return total",
    ]
    out = [
        "package pilot",
        "",
        f"// Module {nn} of the ADR-218 pilot fixture.",
        "",
        f"const Limit{nn} = {limit}",
        "",
        f"func Clamp{nn}(x int) int {{",
        *clamp,
        "}",
        "",
        f"func Sum{nn}(xs []int) int {{",
        *total,
        "}",
        "",
        f"func Mean{nn}(xs []int) int {{",
        "    if len(xs) == 0 {",
        "        return 0",
        "    }",
        f"    return {call}(Sum{nn}(xs)) / len(xs)",
        "}",
        "",
        f"func Max{nn}(xs []int) int {{",
        "    best := 0",
        "    for _, x := range xs {",
        "        if x > best {",
        "            best = x",
        "        }",
        "    }",
        "    return best",
        "}",
        "",
        f"func Min{nn}(xs []int) int {{",
        f"    best := Limit{nn}",
        "    for _, x := range xs {",
        "        if x < best {",
        "            best = x",
        "        }",
        "    }",
        "    return best",
        "}",
    ]
    # Twelve fillers bring the base file to 60 lines; a task that shortens a body shortens the file.
    out.extend(f"// filler {nn}.{pad}" for pad in range(1, 13))
    return out


def tree(task):
    """Return {relative path: lines} for the base tree (task 0) or for what a task leaves."""
    double = ["    y := x * 2", "    return y"]
    plain = ["    total := len(xs)", "    return total"]
    change = {
        1: {3: {"limit": 45}},
        2: {n: {"call": "rescale"} for n in (5, 6, 7, 8)},
        3: {2: {"clamp": double}},
        4: {9: {"clamp": double, "total": plain}, 10: {"clamp": double}},
        5: {11: {"limit": 99, "clamp": double}},
    }.get(task, {})
    return {f"pkg/m{n:02d}.go": module(n, **change.get(n, {})) for n in FILES}


def make(out, task):
    """Write the tree under out."""
    for rel, lines in tree(task).items():
        path = pathlib.Path(out, rel)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")


def sha(out):
    """Return the tree's digest: each path, a NUL, the file's bytes."""
    h = hashlib.sha256()
    for path in sorted(pathlib.Path(out, "pkg").glob("*.go")):
        h.update(f"pkg/{path.name}".encode() + b"\0" + path.read_bytes())
    return h.hexdigest()


if __name__ == "__main__":
    if len(sys.argv) >= 3 and sys.argv[1] == "make":
        make(sys.argv[2], int(sys.argv[3]) if len(sys.argv) > 3 else 0)
    elif len(sys.argv) == 3 and sys.argv[1] == "sha":
        print(sha(sys.argv[2]))
    else:
        sys.exit(__doc__)
