"""Self-test for scan.py: one synthetic session tree per host adapter, with known answers.

Run: python3 -I docs/bench/context-study/test_scan.py   (exit 0 green, 1 red)
Each tree has a re-read inside an epoch, a compaction, a re-read across it, a write that ends
redundancy, and a subagent that re-reads lines its parent was already served.
"""
import json
import shutil
import sqlite3
import subprocess
import sys
import tempfile
from pathlib import Path

sys.dont_write_bytecode = True                   # no __pycache__ in the tree
sys.path.insert(0, str(Path(__file__).resolve().parent))
import scan  # noqa: E402

A = ["alpha line %02d holds some text" % i for i in range(1, 11)]
B = ["beta line %d" % i for i in range(1, 6)]
c = scan.cs.chars


def iso(t):
    return "2026-10-01T00:00:%06.3fZ" % t


# ------------------------------------------------------------ Claude Code

def cl_use(t, i, name, inp, out=100):
    return {"type": "assistant", "timestamp": iso(t), "message": {
        "id": "msg_%d" % i, "role": "assistant", "usage": {"output_tokens": out},
        "content": [{"type": "tool_use", "id": "tu%d" % i, "name": name, "input": inp}]}}


def cl_res(t, i, text, extra=None):
    o = {"type": "user", "timestamp": iso(t), "message": {"role": "user", "content": [
        {"type": "tool_result", "tool_use_id": "tu%d" % i, "content": text}]}}
    if extra:
        o["toolUseResult"] = extra
    return o


def numbered(lines, first=1):
    return "\n".join("%6d→%s" % (n, x) for n, x in enumerate(lines, first))


def cl_read(t, i, path, lines):
    return [cl_use(t, i, "Read", {"file_path": path}),
            cl_res(t + 0.5, i, numbered(lines), {"file": {"filePath": path}})]


AGENT_INPUT = {"description": "explore", "prompt": "go explore b.py"}
CREATE = "mrw write --no-check --create h.txt <<'EOF'\nhello\nEOF"


def claude_tree(root):
    main = [{"type": "assistant", "timestamp": iso(1), "message": {
        "id": "msg_0", "role": "assistant", "usage": {"output_tokens": 1000},
        "content": [{"type": "text", "text": "Hello world"}]}}]
    main += cl_read(2, 1, "/p/a.py", A)                     # served
    main += cl_read(3, 2, "/p/a.py", A)                     # within-epoch redundant
    main += [{"type": "system", "subtype": "compact_boundary", "timestamp": iso(4)}]
    main += cl_read(5.5, 10, "/p/b.py", B[:2])              # b.py's read at 6 then mixes old and new lines
    main += cl_read(5, 3, "/p/a.py", A)                     # cross-epoch redundant
    main += cl_read(6, 4, "/p/b.py", B)
    main += [cl_use(7, 5, "Bash", {"command": "sed -n '1,5p' b.py"}), cl_res(7.5, 5, "\n".join(B) + "\n")]
    main += [cl_use(8, 6, "Edit", {"file_path": "/p/a.py", "old_string": "OLD!", "new_string": "NEW"})]
    main += [cl_use(8.5, 9, "Bash", {"command": CREATE})]   # a create body counts once: 6 chars
    main += [cl_use(9, 7, "Agent", AGENT_INPUT)]
    main += cl_read(12, 8, "/p/a.py", A)                    # after the write: not redundant
    sub1 = [{"type": "user", "timestamp": iso(10), "isSidechain": True,
             "message": {"role": "user", "content": "go explore b.py"}}]
    sub1 += cl_read(10.2, 1, "/p/b.py", B)                   # parent had it: savable
    sub1 += cl_read(10.4, 2, "/p/a.py", A)                   # parent wrote a.py at 8: not savable
    sub2 = [{"type": "user", "timestamp": iso(11), "message": {"role": "user", "content": "x"}}]
    sub2 += cl_read(11.2, 1, "b.py", B)                      # savable, and a sibling read it at 10.2
    sub3 = [{"type": "user", "timestamp": iso(9.8), "message": {"role": "user", "content": "y"}}]
    sub3 += cl_read(12, 1, "/p/b.py", B)                     # starts first, reads last: sibling overlap
    d = root / "proj1"
    (d / "S1" / "subagents").mkdir(parents=True)
    dump(d / "S1.jsonl", main)
    dump(d / "S1" / "subagents" / "agent-a1.jsonl", sub1)
    (d / "S1" / "subagents" / "agent-a1.meta.json").write_text(json.dumps({"toolUseId": "tu7"}), encoding="utf-8")
    dump(d / "S1" / "subagents" / "agent-a2.jsonl", sub2)
    dump(d / "S1" / "subagents" / "agent-a3.jsonl", sub3)


def dump(path, records, pad=True):
    with open(path, "w", encoding="utf-8") as fh:
        for o in records:
            fh.write(json.dumps(o, separators=(",", ":")) + "\n")
        if pad:
            fh.write(" " * 11000 + "\n")              # over the 10 KB floor


# ------------------------------------------------------------------ Codex

def cx(t, typ, payload, n=[0]):
    n[0] += 1
    return {"timestamp": iso(t), "ordinal": n[0], "type": typ, "payload": payload}


def cx_sed(t, cid, cmd, lines):
    return [cx(t, "response_item", {"type": "function_call", "name": "exec_command", "call_id": cid,
                                    "arguments": json.dumps({"cmd": cmd})}),
            cx(t + 0.5, "response_item", {"type": "function_call_output", "call_id": cid,
                                          "output": "Exit code: 0\nWall time: 0.1 seconds\nOutput:\n" +
                                          "\n".join(lines) + "\n"})]


PATCH = "*** Begin Patch\n*** Update File: a.py\n@@\n-old line\n+new line\n context\n*** End Patch"


def codex_tree(root):
    d = root / "2026" / "10" / "01"
    d.mkdir(parents=True)
    main = [cx(1, "session_meta", {"id": "M", "timestamp": iso(1), "source": "cli"}),
            cx(1.1, "response_item", {"type": "message", "role": "assistant",
                                      "content": [{"type": "output_text", "text": "Hi there"}]}),
            cx(1.2, "response_item", {"type": "reasoning", "summary": [{"type": "summary_text", "text": "think"}]})]
    main += cx_sed(2, "c1", "sed -n '1,10p' a.py", A)
    js = "const r = await tools.exec_command({\"cmd\":\"nl -ba a.py | sed -n '1,10p'\"}); text(r.output);"
    main += [cx(3, "response_item", {"type": "custom_tool_call", "name": "exec", "call_id": "c2", "input": js}),
             cx(3.5, "response_item", {"type": "custom_tool_call_output", "call_id": "c2", "output": [
                 {"type": "input_text", "text": "Script completed\nWall time 0.1 seconds\nOutput:\n"},
                 {"type": "input_text", "text": "\n".join("%6d\t%s" % (i, x) for i, x in enumerate(A, 1))}]})]
    main += [cx(5, "compacted", {"message": "", "replacement_history": []})]
    main += cx_sed(6, "c3", "cd /p && sed -n '1,10p' a.py", A)
    main += [cx(7, "response_item", {"type": "custom_tool_call", "name": "apply_patch", "call_id": "c4",
                                     "input": PATCH})]
    main += cx_sed(8, "c5", "sed -n '1,10p' a.py", A)
    main += [cx(9, "event_msg", {"type": "token_count", "info": {"total_token_usage": {
        "output_tokens": 500, "reasoning_output_tokens": 200}}})]
    sub = [cx(3.7, "session_meta", {"id": "S", "timestamp": iso(3.7), "parent_thread_id": "M",
                                    "source": {"subagent": {"thread_spawn": {"parent_thread_id": "M"}}}})]
    sub += cx_sed(4, "s1", "sed -n '1,10p' a.py", A)            # parent had it at 2: savable
    sub += cx_sed(4.2, "s2", "sed -n '1,5p' a.py && sed -n '6,10p' a.py", A)   # a chain: within-epoch
    # b.py ends before its range: 6-8 of a.py's output would fit b.py's range, so nothing is served
    sub += cx_sed(4.3, "s4", "nl -ba b.py | sed -n '1,8p' && nl -ba a.py | sed -n '6,8p'", NUMBERED_BA[:8])
    sub += cx_sed(4.35, "s5", "nl -ba b.py | sed -n '1,5p' && nl -ba a.py | sed -n '6,10p'", NUMBERED_BA)  # complete
    # b.py ends early but a.py restarts at 1, its own start: unambiguous, served
    sub += cx_sed(4.36, "s6", "nl -ba b.py | sed -n '1,8p' && nl -ba a.py | sed -n '1,3p'", NUMBERED_BA3)
    # the first file printed nothing, so a.py's line 1 could start either segment: nothing is served
    sub += cx_sed(4.37, "s7", "nl -ba empty.py | sed -n '1,3p' && nl -ba a.py | sed -n '1,3p'", NUMBERED_A3)
    # the same range on two files, both printed: one run per segment, served
    sub += cx_sed(4.38, "s8", "nl -ba b.py | sed -n '1,3p' && nl -ba a.py | sed -n '1,3p'",
                  NUMBERED_BA3[:3] + NUMBERED_A3)
    two = ("const a = await tools.exec_command({cmd:\"nl -ba b.py | sed -n '1,5p'\"}); text(a.output);"
           " const b = await tools.exec_command({cmd:\"sed -n '1,5p' b.py\"}); text(b.output);")
    sub += [cx(4.4, "response_item", {"type": "custom_tool_call", "name": "exec", "call_id": "s3", "input": two}),
            cx(4.5, "response_item", {"type": "custom_tool_call_output", "call_id": "s3", "output": [
                {"type": "input_text", "text": "Script completed\nWall time 0.1 seconds\nOutput:\n"},
                {"type": "input_text", "text": "\n".join("%6d\t%s" % (i, x) for i, x in enumerate(B, 1))},
                {"type": "input_text", "text": json.dumps({"chunk_id": "x", "output": "\n".join(B)})}]})]
    sub += [cx(4.6, "event_msg", {"type": "token_count", "info": {     # a fork carries 900 in
        "total_token_usage": {"output_tokens": 1000, "reasoning_output_tokens": 500},
        "last_token_usage": {"output_tokens": 100, "reasoning_output_tokens": 50}}}),
            cx(4.8, "event_msg", {"type": "token_count", "info": {
                "total_token_usage": {"output_tokens": 1300, "reasoning_output_tokens": 600},
                "last_token_usage": {"output_tokens": 300, "reasoning_output_tokens": 100}}})]
    dump(d / "rollout-main.jsonl", main, pad=False)
    plain = d / "rollout-sub.jsonl"
    dump(plain, sub, pad=False)
    if shutil.which("zstd"):                                     # exercise the .zst path when it can run
        subprocess.run(["zstd", "-q", "--rm", str(plain), "-o", str(plain) + ".zst"], check=True)
        return True
    return False


# ------------------------------------------------------------- Cursor IDE

def bubble(t, typ, **kw):
    return dict({"type": typ, "createdAt": iso(t)}, **kw)


def read_tool(path, lines, offset=None):
    p = {"targetFile": path}
    if offset:
        p["offset"] = offset
    return {"name": "read_file_v2", "params": json.dumps(p), "result": json.dumps({"contents": "\n".join(lines)})}


NUMBERED_BA = ["%6d\t%s" % (i, x) for i, x in enumerate(B, 1)] + ["%6d\t%s" % (i, A[i - 1]) for i in range(6, 11)]
NUMBERED_BA3 = ["%6d\t%s" % (i, x) for i, x in enumerate(B, 1)] + ["%6d\t%s" % (i, A[i - 1]) for i in range(1, 4)]
NUMBERED_A3 = ["%6d\t%s" % (i, A[i - 1]) for i in range(1, 4)]


def cursor_db(path):
    db = sqlite3.connect(str(path))
    db.execute("create table cursorDiskKV (key text unique on conflict replace, value blob)")
    db.execute("create table composerHeaders (composerId text primary key, workspaceId text, createdAt integer,"
               " lastUpdatedAt integer, isArchived integer, isSubagent integer, recency integer,"
               " checkpointAt integer, value text, subagentTypeName text)")
    t0 = scan.ts_of(iso(0)) * 1000

    def composer(cid, created, bubbles, parent=None, stale=()):
        heads = [{"bubbleId": "%s-%d" % (cid, i), "type": b["type"]} for i, b in enumerate(bubbles)]
        db.execute("insert into cursorDiskKV values (?, ?)", ("composerData:" + cid, json.dumps(
            {"composerId": cid, "fullConversationHeadersOnly": heads})))
        for i, b in enumerate(list(bubbles) + list(stale)):  # a stale bubble has no header
            b["bubbleId"] = "%s-%d" % (cid, i)
            db.execute("insert into cursorDiskKV values (?, ?)", ("bubbleId:%s:%s" % (cid, b["bubbleId"]),
                                                                  json.dumps(b)))
        info = {"subagentInfo": {"parentComposerId": parent}} if parent else {}
        db.execute("insert into composerHeaders (composerId, createdAt, isSubagent, value) values (?, ?, ?, ?)",
                   (cid, int(t0 + created * 1000), int(bool(parent)), json.dumps(info)))
    composer("P", 1, [
        bubble(1, 1, text="please"),
        bubble(2, 2, text="answer", thinking={"text": "hmm"}, toolFormerData=read_tool("a.py", A)),
        bubble(3, 2, toolFormerData=read_tool("a.py", A[2:4], offset=3)),     # within-epoch redundant
        bubble(5, 2, conversationSummary={"summary": "s"}),
        bubble(6, 2, toolFormerData=read_tool("a.py", A)),                    # cross-epoch redundant
        bubble(7, 2, toolFormerData={"name": "search_replace", "rawArgs": json.dumps(
            {"file_path": "a.py", "old_string": "xx", "new_string": "yyy"})}),
        # after the write in header order, though its timestamp says 2.5: not redundant
        bubble(2.5, 2, toolFormerData=read_tool("a.py", A)),
    ], stale=[bubble(4, 2, toolFormerData=read_tool("b.py", B))])           # outside the conversation
    composer("C", 4, [bubble(4.5, 2, toolFormerData=read_tool("a.py", A[2:4], offset=3))], parent="P")
    db.commit()
    db.close()


# ------------------------------------------------------------------ check

def main():
    with tempfile.TemporaryDirectory() as tmp:
        tmp = Path(tmp)
        claude_tree(tmp / "claude")
        zst = codex_tree(tmp / "codex")
        cursor_db(tmp / "state.vscdb")
        res = scan.main(["--claude-root", str(tmp / "claude"), "--codex-root", str(tmp / "codex"),
                         "--cursor-db", str(tmp / "state.vscdb"), "--out", str(tmp / "r.json")])
    h = res["hosts"]
    cl, cx_, cu = h["claude"], h["codex"], h["cursor"]
    m1 = cl["main"]["M1"]["visible_chars"]
    want = {
        # Claude: M2 strict is copy-study's rule; the sed read only reaches the extended pool
        "claude strict served": (cl["main"]["M2"]["strict"]["served_chars"], 4 * c(A) + c(B) + c(B[:2])),
        "claude strict within": (cl["main"]["M2"]["strict"]["within_epoch_chars"], c(A) + c(B[:2])),
        "claude strict cross": (cl["main"]["M2"]["strict"]["cross_epoch_chars"], c(A)),
        "claude ext within": (cl["main"]["M2"]["ext"]["within_epoch_chars"], c(A) + c(B) + c(B[:2])),
        # a whole-file guard answers a call only when all it served was a within-epoch re-serve:
        # the read at 3 and the sed at 7.5, not the read at 6 that also served new lines
        "claude whole-file unchanged": (cl["main"]["M2"]["ext"]["whole_file_unchanged_within_chars"],
                                        c(A) + c(B)),
        "claude reconciles": (cl["copy_study_scanner_today"]["redundant_chars"],
                              cl["all"]["M2"]["strict"]["redundant_chars"]),
        "claude compactions": (cl["main"]["M2"]["mid_session_compactions"], 1),
        "claude reread after": (cl["main"]["M2"]["sessions_rereading_after_compaction"], 1),
        "claude prose": (m1["prose"], len("Hello world")),
        "claude write new/old": ((m1["write_new"], m1["write_old"]), (3 + 6, 4)),
        "claude subagent prompt": (m1["subagent_prompt"], len(json.dumps(AGENT_INPUT))),
        "claude reported": (cl["main"]["M1"]["reported_output_tokens"], 1000 + 10 * 100),
        "claude M3 parent": (cl["subagent"]["M3"]["parent_savable_chars"], 3 * c(B)),
        "claude M3 sibling": (cl["subagent"]["M3"]["sibling_overlap_chars"], 2 * c(B)),
        "claude M3 served": (cl["subagent"]["M3"]["served_chars"], 3 * c(B) + c(A)),
        # Codex
        "codex served": (cx_["main"]["M2"]["ext"]["served_chars"], 4 * c(A)),
        "codex within": (cx_["main"]["M2"]["ext"]["within_epoch_chars"], c(A)),
        "codex cross": (cx_["main"]["M2"]["ext"]["cross_epoch_chars"], c(A)),
        "codex whole-file unchanged": (cx_["main"]["M2"]["ext"]["whole_file_unchanged_within_chars"], c(A)),
        "codex write new/old": ((cx_["main"]["M1"]["visible_chars"]["write_new"],
                                 cx_["main"]["M1"]["visible_chars"]["write_old"]), (9, 17)),
        "codex prose/thinking": ((cx_["main"]["M1"]["visible_chars"]["prose"],
                                  cx_["main"]["M1"]["visible_chars"]["thinking_visible"]), (8, 5)),
        "codex reasoning": (cx_["main"]["M1"]["reasoning_tokens_reported"], 200),
        "codex M3 parent": (cx_["subagent"]["M3"]["parent_savable_chars"], c(A)),
        "codex sub chain+cells within": (cx_["subagent"]["M2"]["ext"]["within_epoch_chars"],
                                         c(A) + c(A[5:10]) + 3 * c(B) + 2 * c(A[:3]) + c(B[:3])),
        "codex sub served": (cx_["subagent"]["M2"]["ext"]["served_chars"],
                             2 * c(A) + c(A[5:10]) + 4 * c(B) + 2 * c(A[:3]) + c(B[:3])),
        "codex fork usage baseline": ((cx_["subagent"]["M1"]["reported_output_tokens"],
                                       cx_["subagent"]["M1"]["reasoning_tokens_reported"]), (400, 150)),
        "codex main usage": (cx_["main"]["M1"]["reported_output_tokens"], 500),
        "codex zst read": (cx_["corpus"].get("zst", 0), 1 if zst else 0),
        # Cursor IDE
        "cursor served": (cu["main"]["M2"]["ext"]["served_chars"], 3 * c(A) + c(A[2:4])),
        "cursor within": (cu["main"]["M2"]["ext"]["within_epoch_chars"], c(A[2:4])),
        "cursor cross": (cu["main"]["M2"]["ext"]["cross_epoch_chars"], c(A)),
        "cursor whole-file unchanged": (cu["main"]["M2"]["ext"]["whole_file_unchanged_within_chars"], c(A[2:4])),
        "cursor write new/old": ((cu["main"]["M1"]["visible_chars"]["write_new"],
                                  cu["main"]["M1"]["visible_chars"]["write_old"]), (3, 2)),
        "cursor prose/thinking": ((cu["main"]["M1"]["visible_chars"]["prose"],
                                   cu["main"]["M1"]["visible_chars"]["thinking_visible"]), (6, 3)),
        "cursor M3 parent": (cu["subagent"]["M3"]["parent_savable_chars"], c(A[2:4])),
    }
    bad = ["%s: got %r, want %r" % (name, got, exp) for name, (got, exp) in want.items() if got != exp]
    for b in bad:
        print("FAIL", b)
    print("PASS (%d checks%s)" % (len(want), "" if zst else "; no zstd CLI, .zst path not exercised")
          if not bad else "%d of %d failed" % (len(bad), len(want)))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
