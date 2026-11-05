"""Mechanism counts for arm R (docs/bench/observability/HOOK-PREREGISTRATION.md), frozen before the first call.

For each stream-json transcript: host Reads, overlapping reads (an `mrw read` of a file a host Read showed
since the file's last landed write), hook-licensed writes (an applied `mrw write` to a file the cell never
`mrw read` but did host-Read), refused unread writes (a write refused for an unread line after a host Read),
and unknown commands (an mrw call whose files this cannot name). Usage: bench_mechanism.py TRANSCRIPT...
"""
import json
import os
import re
import shlex
import sys

READ_VALUE_FLAGS = {"-C", "--max-lines", "--max-chars", "--max-cols", "--max-matches", "--exclude", "--if-changed",
                    "--if-served", "--echo-pad"}
WRITE_VALUE_FLAGS = {"--then", "--then-sh", "--echo-pad"}
UNNAMEABLE = {"--grep", "--ast-grep", "--files-from", "--json-path", "--sub", "--rewrite"}
SEPARATORS = {";", "&&", "||", "|", "&", "(", ")"}
HEREDOC = re.compile(r"<<-?\s*(['\"]?)(\w+)\1")


def norm(path, cwd):
    """A path relative to the cell's tree, as host Read (absolute) and mrw (relative) spell it."""
    strip = lambda p: p[len("/private"):] if p.startswith("/private/") else p  # noqa: E731
    full = os.path.normpath(strip(os.path.join(cwd, path)))
    rel = os.path.relpath(full, os.path.normpath(strip(cwd)))
    return full if rel.startswith("..") else rel


def split_heredocs(command):
    """The command without heredoc bodies, and the bodies' text."""
    lines, kept, bodies, ending = command.split("\n"), [], [], None
    for line in lines:
        if ending is not None:
            if line.strip() == ending:
                ending = None
            else:
                bodies.append(line)
            continue
        kept.append(line)
        match = HEREDOC.search(line)
        if match:
            ending = match.group(2)
    return "\n".join(kept), "\n".join(bodies)


def mrw_calls(command):
    """[(subcommand, args)] for each mrw invocation, in order, or None when the command will not tokenize."""
    shell, bodies = split_heredocs(command)
    try:
        lexer = shlex.shlex(shell.replace("\n", " ; "), posix=True, punctuation_chars=True)
        lexer.whitespace_split = True
        tokens = list(lexer)
    except ValueError:
        return None, bodies
    calls, segment = [], []
    for token in tokens + [";"]:
        if token in SEPARATORS or set(token) <= set(";&|()<>"):
            words = [w for w in segment if not re.match(r"^\w+=", w)]
            if words and (words[0] == "mrw" or words[0].endswith("/mrw")):
                calls.append(words[1:])
            segment = []
        else:
            segment.append(token)
    return calls, bodies


def files_of(args, bodies, cwd):
    """('read'|'write'|None, [files]) for one mrw call, or (kind, None) when its files cannot be named."""
    if not args:
        return None, []
    if args[0] in ("-C", "--root"):
        kind = args[2] if len(args) > 2 else None
        return (kind, None) if kind in ("read", "write") else (None, [])
    kind, rest = args[0], args[1:]
    if kind not in ("read", "write"):
        return None, []
    # A shell variable or command substitution names files this cannot see; a `$` address does not.
    if any(re.search(r"\$[A-Za-z_{(]|`", a) or a.split("=")[0] in UNNAMEABLE or a.startswith("--format") for a in rest):
        return kind, None
    if any(a.split("=")[0] in UNNAMEABLE or a.startswith("--format") for a in rest):
        return kind, None
    values = READ_VALUE_FLAGS if kind == "read" else WRITE_VALUE_FLAGS
    positional, skip = [], False
    for a in rest:
        if skip:
            skip = False
        elif a == "--create":
            continue
        elif a in values:
            skip = True
        elif a == "--" or (a.startswith("-") and a != "-"):
            continue
        else:
            positional.append(a)
    if kind == "read":
        if not positional or "-" in positional:
            return kind, None
        return kind, sorted({norm(spec.split(":")[0], cwd) for spec in positional})
    if "--create" in rest:
        return kind, [norm(positional[0], cwd)] if positional else None
    if positional != ["-"]:
        return kind, None
    named = re.findall(r"^@@ (\S+)", bodies, re.M)
    return kind, sorted({norm(p, cwd) for p in named}) or None


def result_text(content):
    if isinstance(content, str):
        return content
    return "".join(part.get("text", "") for part in content or [] if isinstance(part, dict))


def classify(raw):
    """The counts for one transcript."""
    cwd, uses, results = "", [], {}
    for line in raw.splitlines():
        try:
            row = json.loads(line)
        except ValueError:
            continue
        if not isinstance(row, dict):
            continue
        if row.get("type") == "system" and row.get("subtype") == "init":
            cwd = row.get("cwd", "")
        for part in (row.get("message") or {}).get("content", []) if isinstance((row.get("message") or {}).get("content"), list) else []:
            if part.get("type") == "tool_use":
                uses.append(part)
            elif part.get("type") == "tool_result":
                results[part.get("tool_use_id")] = result_text(part.get("content"))
    counts = {"host_reads": 0, "overlapping_reads": 0, "hook_licensed_writes": 0, "refused_unread_writes": 0, "unknown": 0}
    host_since_write, host_ever, mrw_read_ever = set(), set(), set()
    for use in uses:
        text = results.get(use.get("id"), "")
        if use.get("name") == "Read":
            path = norm(use.get("input", {}).get("file_path", ""), cwd)
            counts["host_reads"] += 1
            host_since_write.add(path)
            host_ever.add(path)
            continue
        if use.get("name") != "Bash":
            continue
        calls, bodies = mrw_calls(use.get("input", {}).get("command", ""))
        if calls is None:
            counts["unknown"] += "mrw" in use.get("input", {}).get("command", "")
            continue
        for args in calls:
            kind, files = files_of(args, bodies, cwd)
            if kind is None:
                continue
            if files is None:
                counts["unknown"] += 1
                continue
            if kind == "read":
                counts["overlapping_reads"] += len(set(files) & host_since_write)
                mrw_read_ever.update(files)
                continue
            applied = "— applied" in text and "NOTHING WRITTEN" not in text
            for path in files:
                if applied:
                    counts["hook_licensed_writes"] += path in host_ever and path not in mrw_read_ever
                    host_since_write.discard(path)
                elif "has not been read" in text and path in host_ever:
                    counts["refused_unread_writes"] += 1
    return counts


def main(paths):
    cells = {p: classify(open(p, encoding="utf-8", errors="replace").read()) for p in paths}
    total = {key: sum(cell[key] for cell in cells.values()) for key in next(iter(cells.values()), {})}
    total["cells_with_host_read"] = sum(1 for cell in cells.values() if cell["host_reads"])
    print(json.dumps({"cells": cells, "total": total}, indent=1, sort_keys=True))


if __name__ == "__main__":
    main(sys.argv[1:])
