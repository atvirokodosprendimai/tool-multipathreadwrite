// ADR-089. Drives the built plugin's tools the way opencode calls them —
// execute(args, context) — against the mrw binary this checkout builds, so a
// tool that cannot spawn, licenses what the host cut, drops an argument or
// misroutes its root fails here. Build first: `go build -o bin/mrw ./cmd/mrw`
// at the repository root, then `npm run build` here.
import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { mrwPlugin } from "../dist/index.js";

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const exe = process.platform === "win32" ? "mrw.exe" : "mrw";
const built = path.join(repo, "bin", exe);

// No mrw on PATH unless a test puts one there, so a tool that answers proves it
// ran the worktree's bin/mrw rather than whatever mrw this machine has installed.
const bare = process.platform === "win32" ? process.env.PATH : "/usr/bin:/bin";
process.env.PATH = bare;

// ADR-170: the binary serves its tools only when enabled, so the fixture says off
// in its own environment (whatever this shell exports): only the plugin's own
// MRW_MCP=on for its child can turn them on.
process.env.MRW_MCP = "off";

// tmp makes a directory the test removes when it ends.
function tmp(t, prefix) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  // A process that has just exited on Windows can hold the directory briefly;
  // ten retries back off linearly, about 5.5 s in all, before it throws.
  t.after(() => fs.rmSync(dir, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 }));
  return dir;
}

// worktree makes a checkout holding content as f.txt, with the built binary at
// bin/mrw unless noBinary, and gives the test its own mrw state.
function worktree(t, { content = "one\ntwo\n", noBinary = false } = {}) {
  assert.ok(fs.existsSync(built), `build the binary first: go build -o bin/${exe} ./cmd/mrw (${built})`);
  process.env.XDG_STATE_HOME = tmp(t, "mrw-state-");
  const dir = tmp(t, "mrw-plugin-");
  fs.writeFileSync(path.join(dir, "f.txt"), content);
  if (!noBinary) {
    fs.mkdirSync(path.join(dir, "bin"));
    fs.copyFileSync(built, path.join(dir, "bin", exe));
    fs.chmodSync(path.join(dir, "bin", exe), 0o755);
  }
  return dir;
}

async function tools(dir) {
  const hooks = await mrwPlugin({ directory: dir, worktree: dir });
  return hooks.tool;
}

function ctx(dir, worktree = dir) {
  return { directory: dir, worktree, abort: new AbortController().signal, metadata() {} };
}

// acks returns the checkpoint ids whose open AND close markers both appear in
// text — the rule a caller must follow before acknowledging a run.
function acks(text) {
  const open = [...text.matchAll(/^-- ck ([0-9a-f]+) open lines /gm)].map((m) => m[1]);
  const closed = new Set([...text.matchAll(/^-- ck ([0-9a-f]+) close$/gm)].map((m) => m[1]));
  return open.filter((id) => closed.has(id));
}

test("a read carries checkpoints, and a write that acknowledges them lands", async (t) => {
  const dir = worktree(t);
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["f.txt"] }, ctx(dir));
  assert.equal(read.metadata.isError, false, read.output);
  assert.match(read.output, /2\| two/);
  const ids = acks(read.output);
  assert.ok(ids.length > 0, `no checkpoint came back: ${read.output}`);
  const write = await tl.mrw_write.execute({ plan: "@@ f.txt 2 replace\nTWO\n", ack: ids }, ctx(dir));
  assert.equal(write.metadata.isError, false, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "one\nTWO\n");
});

test("grep can count matching lines without serving or licensing them", async (t) => {
  const content = "one hit hit\ntwo\nthree hit\n";
  const dir = worktree(t, { content });
  const tl = await tools(dir);
  assert.ok(tl.mrw_read.args.count, "count must be visible in the host tool schema");
  const result = await tl.mrw_read.execute({ specs: ["f.txt"], grep: "hit", count: true }, ctx(dir));
  assert.equal(result.metadata.isError, false, result.output);
  assert.match(result.output, /2 matching line\(s\) in 1 file\(s\)/);
  assert.doesNotMatch(result.output, /one hit hit|three hit|-- ck /);
  const write = await tl.mrw_write.execute({ plan: "@@ f.txt 1 replace\nchanged\n" }, ctx(dir));
  assert.equal(write.metadata.isError, true, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), content);
  const invalid = await tl.mrw_read.execute({ specs: ["f.txt"], grep: "hit", count: true, maxCols: 0 }, ctx(dir));
  assert.equal(invalid.metadata.isError, true, invalid.output);
  assert.match(invalid.output, /drop max_cols/);
});

test("grep returns a window around a match on a line larger than the plugin limit", async (t) => {
  const content = "ab".repeat(18000) + " NEEDLE " + "yz".repeat(18000) + "\n";
  const dir = worktree(t, { content });
  const tl = await tools(dir);
  assert.ok(tl.mrw_read.args.maxCols, "maxCols must be visible in the host tool schema");
  const result = await tl.mrw_read.execute({ specs: ["f.txt"], grep: "NEEDLE", maxCols: 100 }, ctx(dir));
  assert.equal(result.metadata.isError, false, result.output);
  assert.match(result.output, /ab NEEDLE yz/);
  assert.match(result.output, /\[cols \d+-\d+ of 72008\]/);
  assert.doesNotMatch(result.output, /-- ck /);
  const write = await tl.mrw_write.execute({ plan: "@@ f.txt 1 replace\nchanged\n" }, ctx(dir));
  assert.equal(write.metadata.isError, true, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), content);
});

test("a conditional reread serves and licenses only the changed file", async (t) => {
  const dir = worktree(t);
  const tl = await tools(dir);
  const specs = ["f.txt", "g.txt"];
  fs.writeFileSync(path.join(dir, "g.txt"), "original second file\n");
  const snapshot = await tl.mrw_read.execute({ specs, stat: true }, ctx(dir));
  assert.equal(snapshot.metadata.isError, false, snapshot.output);
  const headers = [...snapshot.output.matchAll(/^==> (\S+) .* sha ([0-9a-f]{8})\b/gm)];
  const ifChanged = Object.fromEntries(headers.map(([, file, sha]) => [file, sha]));
  assert.deepEqual(Object.keys(ifChanged).sort(), specs);
  const same = await tl.mrw_read.execute({ specs, ifChanged }, ctx(dir));
  assert.equal(same.metadata.isError, false, same.output);
  assert.match(same.output, /f\.txt +unchanged since sha/);
  assert.match(same.output, /g\.txt +unchanged since sha/);
  assert.doesNotMatch(same.output, /-- ck /);
  fs.writeFileSync(path.join(dir, "g.txt"), "updated\n");
  const result = await tl.mrw_read.execute({ specs, ifChanged }, ctx(dir));
  assert.equal(result.metadata.isError, false, result.output);
  assert.match(result.output, /f\.txt +unchanged since sha/);
  assert.doesNotMatch(result.output, /1\| one/);
  assert.match(result.output, /1\| updated/);
  assert.equal(acks(result.output).length, 1, result.output);
  const changed = await tl.mrw_write.execute({ plan: "@@ g.txt 1 replace\naccepted\n", ack: acks(result.output) }, ctx(dir));
  assert.equal(changed.metadata.isError, false, changed.output);
  assert.equal(fs.readFileSync(path.join(dir, "g.txt"), "utf8"), "accepted\n");
  const unchanged = await tl.mrw_write.execute({ plan: "@@ f.txt 1 replace\nforbidden\n" }, ctx(dir));
  assert.equal(unchanged.metadata.isError, true, unchanged.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "one\ntwo\n");
  assert.ok(tl.mrw_read.args.ifChanged, "ifChanged must be visible in the host tool schema");
});

test("a write without the acknowledgement, or to a file never read, is refused", async (t) => {
  const dir = worktree(t);
  const tl = await tools(dir);
  await tl.mrw_read.execute({ specs: ["f.txt"] }, ctx(dir));
  const unacked = await tl.mrw_write.execute({ plan: "@@ f.txt 2 replace\nTWO\n" }, ctx(dir));
  assert.equal(unacked.metadata.isError, true, unacked.output);
  fs.writeFileSync(path.join(dir, "g.txt"), "never\nread\n");
  const unread = await tl.mrw_write.execute({ plan: "@@ g.txt 2 replace\nX\n" }, ctx(dir));
  assert.equal(unread.metadata.isError, true, unread.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "one\ntwo\n");
  assert.equal(fs.readFileSync(path.join(dir, "g.txt"), "utf8"), "never\nread\n");
});

// ADR-161 through the plugin: grep with sub builds the plan, so a rename lands in
// one call with no read or ack, and it lands exactly what the hand-written plan
// lands. paths scopes the walk: a match outside it is kept; without paths it is
// changed too.
test("grep with sub renames in one call, scoped by paths, as a hand-written plan does", async (t) => {
  const files = {
    "a/def.py": "def _old_name(x):\n    return x\n",
    "a/use.py": "from a.def import _old_name\n\nprint(_old_name(1))\n",
    "outside.py": "# keeps _old_name\n",
  };
  const make = () => {
    const dir = worktree(t);
    for (const [name, text] of Object.entries(files)) {
      fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
      fs.writeFileSync(path.join(dir, name), text);
    }
    return dir;
  };
  const snapshot = (dir) => Object.fromEntries(Object.keys(files).map((n) => [n, fs.readFileSync(path.join(dir, n), "utf8")]));
  const args = { grep: "\\b_old_name\\b", sub: "_new_name" };

  // Scoped: the two files under a/ change; outside.py keeps its match.
  const scoped = make();
  let tl = await tools(scoped);
  const w = await tl.mrw_write.execute({ ...args, paths: ["a"] }, ctx(scoped));
  assert.equal(w.metadata.isError, false, w.output);
  assert.match(w.output, /3 hunk\(s\), 2 file\(s\), 0 failed/);
  const got = snapshot(scoped);
  assert.equal(got["outside.py"], files["outside.py"]);

  // The same change by a hand-written plan, after a read and its acks.
  const manual = make();
  tl = await tools(manual);
  const read = await tl.mrw_read.execute({ specs: ["a/def.py", "a/use.py"] }, ctx(manual));
  const plan =
    "@@ a/def.py 1 replace\ndef _new_name(x):\n" +
    "@@ a/use.py 1 replace\nfrom a.def import _new_name\n" +
    "@@ a/use.py 3 replace\nprint(_new_name(1))\n";
  const mw = await tl.mrw_write.execute({ plan, ack: acks(read.output) }, ctx(manual));
  assert.equal(mw.metadata.isError, false, mw.output);
  assert.deepEqual(got, snapshot(manual));

  // Unscoped: the walk covers the whole checkout, so outside.py changes too.
  const whole = make();
  tl = await tools(whole);
  const ww = await tl.mrw_write.execute(args, ctx(whole));
  assert.equal(ww.metadata.isError, false, ww.output);
  assert.equal(fs.readFileSync(path.join(whole, "outside.py"), "utf8"), "# keeps _new_name\n");
});

// The server's refusals reach the caller unchanged, and nothing is written.
test("a generator given with a plan, or half a generator, is refused and writes nothing", async (t) => {
  const dir = worktree(t, { content: "old\n" });
  const tl = await tools(dir);
  // Each case is refused for its own reason, as the server words it, so an
  // argument the plugin dropped would show as another reason.
  for (const [args, reason] of [
    [{ plan: "@@ f.txt 1 replace\nX\n", grep: "old", sub: "new" }, "makes its own plan: drop plan and format"],
    [{ grep: "old", sub: "new", format: "plan" }, "makes its own plan: drop plan and format"],
    [{ sub: "new" }, "grep and sub go together"],
    [{ grep: "old" }, "grep and sub go together"],
    [{ plan: "@@ f.txt 1 replace\nX\n", paths: ["f.txt"] }, "paths shape the walk of grep with sub"],
    [{}, "mrw_write needs a plan"],
  ]) {
    const r = await tl.mrw_write.execute(args, ctx(dir));
    assert.equal(r.metadata.isError, true, `${JSON.stringify(args)}: ${r.output}`);
    assert.ok(r.output.includes(reason), `${JSON.stringify(args)}: want ${JSON.stringify(reason)} in ${r.output}`);
    assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "old\n", JSON.stringify(args));
  }
  const dry = await tl.mrw_write.execute({ grep: "old", sub: "new", dryRun: true }, ctx(dir));
  assert.equal(dry.metadata.isError, false, dry.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "old\n");
  // An empty sub is a replacement too: it deletes each match, as the CLI's --sub '' does.
  const del = await tl.mrw_write.execute({ grep: "ol", sub: "" }, ctx(dir));
  assert.equal(del.metadata.isError, false, del.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "d\n");
});

test("a page the host cuts licenses only the runs that arrived whole", async (t) => {
  const lines = Array.from({ length: 1000 }, (_, i) => `line ${i + 1}`).join("\n") + "\n";
  const dir = worktree(t, { content: lines });
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["f.txt"] }, ctx(dir));
  assert.equal(read.metadata.isError, false, read.output);
  // opencode truncates a long result; cut this one inside its third run.
  const third = [...read.output.matchAll(/^-- ck [0-9a-f]+ open lines /gm)][2];
  assert.ok(third, `the fixture did not produce three runs: ${read.output.slice(0, 300)}`);
  const cut = read.output.slice(0, third.index + 200);
  const ids = acks(cut);
  assert.equal(ids.length, 2, `the cut kept ${ids.length} whole runs`);
  const tail = await tl.mrw_write.execute({ plan: "@@ f.txt 900 replace\nX\n", ack: ids }, ctx(dir));
  assert.equal(tail.metadata.isError, true, `a line the host cut was licensed: ${tail.output}`);
  const head = await tl.mrw_write.execute({ plan: "@@ f.txt 10 replace\nTEN\n", ack: ids }, ctx(dir));
  assert.equal(head.metadata.isError, false, head.output);
  assert.match(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), /^line 9\nTEN\nline 11\n/m);
});

test("root reads another checkout", async (t) => {
  const dir = worktree(t);
  const other = tmp(t, "mrw-other-");
  fs.writeFileSync(path.join(other, "g.txt"), "elsewhere\n");
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["g.txt"], root: other }, ctx(dir));
  assert.equal(read.metadata.isError, false, read.output);
  assert.match(read.output, /1\| elsewhere/);
});

test("a session with no git worktree works in its directory", async (t) => {
  const dir = worktree(t);
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["f.txt"] }, ctx(dir, "/"));
  assert.equal(read.metadata.isError, false, read.output);
  assert.match(read.output, /1\| one/);
});


test("iter takes its verb and specs", async (t) => {
  const dir = worktree(t);
  const tl = await tools(dir);
  const add = await tl.mrw_iter.execute({ args: ["add", "f.txt"] }, ctx(dir));
  assert.equal(add.metadata.exitCode, 0, add.output);
  const list = await tl.mrw_iter.execute({}, ctx(dir));
  assert.match(list.output, /f\.txt/);
});

test("without bin/mrw in the worktree, mrw is found on PATH", async (t) => {
  const dir = worktree(t, { noBinary: true });
  process.env.PATH = path.dirname(built) + path.delimiter + bare;
  t.after(() => {
    process.env.PATH = bare;
  });
  const tl = await tools(dir);
  const version = await tl.mrw_version.execute({}, ctx(dir));
  assert.equal(version.metadata.exitCode, 0, version.output);
  assert.match(version.output, /\S/);
});

test("a CLI tool reports the exit code of a refusal", async (t) => {
  const dir = worktree(t);
  const tl = await tools(dir);
  const seen = await tl.mrw_seen.execute({ dryRun: true }, ctx(dir));
  assert.equal(seen.metadata.exitCode, 2, seen.output);
  assert.match(seen.output, /^exit: 2/);
});

// within is how p settled — { value } or { error } — or { late: true } once ms
// pass first, with no timer left behind.
async function within(p, ms) {
  let timer;
  const late = new Promise((r) => { timer = setTimeout(() => r({ late: true }), ms); });
  try {
    return await Promise.race([p.then((value) => ({ value }), (error) => ({ error })), late]);
  } finally {
    clearTimeout(timer);
  }
}

// The check's child for the cancellation test: it connects to the test's own
// socket and holds the connection until it is told "stop", the test closes it,
// or 45 s pass. The operating system closes that connection when the process
// ends, however it ends, and only then, so the connection is the test's view of
// the child: no pid is read, polled or signalled.
const fixtureChild = (port) =>
  `const s=require("net").connect(${port},"127.0.0.1");` +
  's.on("data",(d)=>{if(String(d).includes("stop"))process.exit(0)});' +
  's.on("close",()=>process.exit(0));s.on("error",()=>process.exit(0));' +
  "setTimeout(()=>process.exit(0),45000)";

// ADR-089 Decision 3: an aborted call kills its child. The user cancels a check
// that is running; the call must end as aborted and the check's own child must
// end with it. A child that is still alive, stalled or suspended, keeps its
// connection and fails the test.
test("cancelling a running check aborts the call and ends the check's child", async (t) => {
  const saved = { TMPDIR: process.env.TMPDIR, TMP: process.env.TMP, TEMP: process.env.TEMP, XDG_STATE_HOME: process.env.XDG_STATE_HOME };
  const ac = new AbortController();
  const server = net.createServer();
  const held = new Set(); // every connection the fixture made, however late
  let call;
  // Registered before the directories, so it runs before they are removed. It
  // releases a child the assertions left running only after they ran: stop
  // listening, abort, tell each connection to stop and wait for it (bounded),
  // destroy what is left, wait for the call's promise (bounded: it settles on
  // the abort, which can be before mrw has exited, so this is not a wait for
  // mrw), and put the environment back whatever happened.
  t.after(async () => {
    try {
      server.close();
      ac.abort();
      for (const { conn, closed } of held) {
        if (!conn.destroyed) {
          conn.end("stop\n");
          await within(closed, 10000);
        }
        conn.destroy();
      }
      if (call) await within(call, 10000);
    } finally {
      for (const [k, v] of Object.entries(saved)) if (v === undefined) delete process.env[k]; else process.env[k] = v;
    }
  });
  const connected = new Promise((resolve) => server.on("connection", (conn) => {
    conn.on("error", () => {}); // a killed child resets the connection on Windows
    const child = { conn, closed: new Promise((r) => conn.once("close", r)) };
    held.add(child);
    resolve(child);
  }));
  await new Promise((r) => server.listen(0, "127.0.0.1", r));
  const dir = worktree(t);
  const logs = tmp(t, "mrw-logs-");
  process.env.TMPDIR = process.env.TMP = process.env.TEMP = logs; // the check's log lands in the test's own directory
  let node = process.execPath;
  if (process.platform === "win32") node = node.replaceAll("\\", "/"); // Git sh reads C:/… whole
  const check = `exec '${node.replaceAll("'", "'\\''")}' -e '${fixtureChild(server.address().port)}'`;
  fs.writeFileSync(path.join(dir, ".quality-harness.json"), JSON.stringify({ check }));
  const tl = await tools(dir);
  call = tl.mrw_check.execute({ paths: ["f.txt"] }, { ...ctx(dir), abort: ac.signal });
  const ready = await within(Promise.race([connected, call.then((value) => ({ early: { value } }), (error) => ({ early: { error } }))]), 15000);
  if (ready.late) assert.fail("the check's child never connected");
  if (ready.value.early) {
    const e = ready.value.early;
    assert.fail(`the call ended before the abort: ${e.error ? String(e.error) : e.value.output}`);
  }
  const child = ready.value;
  ac.abort();
  const outcome = await within(call, 10000);
  if (outcome.error?.name !== "AbortError") {
    assert.fail(`the call was not aborted: ${outcome.late ? "not settled in 10 s" : outcome.error ? String(outcome.error) : outcome.value.output}`);
  }
  const gone = await within(child.closed, 10000);
  if (gone.late) assert.fail("the check's child still holds its connection 10 s after the aborted call");
});

test("a child that exits without reading its input is reported, not raised", { skip: process.platform === "win32" }, async (t) => {
  const dir = worktree(t, { noBinary: true });
  fs.mkdirSync(path.join(dir, "bin"));
  fs.writeFileSync(path.join(dir, "bin", "mrw"), "#!/bin/sh\nexit 3\n", { mode: 0o755 });
  const tl = await tools(dir);
  const write = await tl.mrw_write.execute({ plan: "@@ f.txt 1 replace\n" + "x".repeat(2 << 20) + "\n" }, ctx(dir));
  assert.equal(write.metadata.isError, true, write.output);
  assert.match(write.output, /exited 3/);
});

test("a write whose check cannot run landed, and the answer does not call it refused", { skip: process.platform === "win32" }, async (t) => {
  const dir = worktree(t);
  fs.writeFileSync(path.join(dir, "a.go"), "package a\nfunc A() {}\n");
  fs.writeFileSync(path.join(dir, ".quality-harness.json"), '{"check":"exit 0"}\n');
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["a.go"] }, ctx(dir));
  const saved = process.env.TMPDIR;
  process.env.TMPDIR = path.join(dir, "missing");
  t.after(() => { if (saved === undefined) delete process.env.TMPDIR; else process.env.TMPDIR = saved; });
  const write = await tl.mrw_write.execute({ plan: "@@ a.go 2 replace\nfunc A() { _ = 1 }\n", ack: acks(read.output), check: true }, ctx(dir));
  assert.equal(write.metadata.isError, true, write.output);
  assert.doesNotMatch(write.output, /refused/, write.output);
  assert.match(write.output, /COULD NOT RUN/, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "a.go"), "utf8"), "package a\nfunc A() { _ = 1 }\n");
});
