// ADR-089. Drives the built plugin's tools the way opencode calls them —
// execute(args, context) — against the mrw binary this checkout builds, so a
// tool that cannot spawn, licenses what the host cut, drops an argument or
// misroutes its root fails here. Build first: `go build -o bin/mrw ./cmd/mrw`
// at the repository root, then `npm run build` here.
import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
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

// tmp makes a directory the test removes when it ends.
function tmp(t, prefix) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
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
  const write = await tl.mrw_write.execute({ plan: "@@ a.go 2 replace\nfunc A() { _ = 1 }\n", ack: acks(read.output) }, ctx(dir));
  assert.equal(write.metadata.isError, true, write.output);
  assert.doesNotMatch(write.output, /refused/, write.output);
  assert.match(write.output, /COULD NOT RUN/, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "a.go"), "utf8"), "package a\nfunc A() { _ = 1 }\n");
});
