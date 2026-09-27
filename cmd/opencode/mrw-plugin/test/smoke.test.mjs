// ADR-089. Drives the built plugin's tools the way opencode calls them —
// execute(args, context) — against the mrw binary this checkout builds, so a
// tool that cannot spawn, drops its plan or misorders a flag fails here.
// Build first: `go build -o bin/mrw ./cmd/mrw` at the repository root, then
// `npm run build` here.
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

// A worktree with one file, and the built binary at bin/mrw unless noBinary.
function worktree({ noBinary = false } = {}) {
  assert.ok(fs.existsSync(built), `build the binary first: go build -o bin/${exe} ./cmd/mrw (${built})`);
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-plugin-"));
  fs.writeFileSync(path.join(dir, "f.txt"), "one\ntwo\n");
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

function ctx(dir) {
  return { directory: dir, worktree: dir, abort: new AbortController().signal, metadata() {} };
}

test("a read serves the lines and a write to them lands", async (t) => {
  process.env.XDG_STATE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-state-"));
  const dir = worktree();
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["f.txt"] }, ctx(dir));
  assert.equal(read.metadata.exitCode, 0, read.output);
  assert.match(read.output, /2\| two/);
  const write = await tl.mrw_write.execute({ plan: "@@ f.txt 2 replace\nTWO\n", noCheck: true }, ctx(dir));
  assert.equal(write.metadata.exitCode, 0, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "one\nTWO\n");
  assert.ok(!fs.existsSync(path.join(dir, ".mrw-plan-input")), "the plan was written into the checkout");
});

test("a write to lines never read is refused, and the file is unchanged", async () => {
  process.env.XDG_STATE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-state-"));
  const dir = worktree();
  const tl = await tools(dir);
  const write = await tl.mrw_write.execute({ plan: "@@ f.txt 2 replace\nTWO\n", noCheck: true }, ctx(dir));
  assert.equal(write.metadata.exitCode, 1, write.output);
  assert.equal(fs.readFileSync(path.join(dir, "f.txt"), "utf8"), "one\ntwo\n");
});

test("root reaches mrw before the subcommand", async () => {
  process.env.XDG_STATE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-state-"));
  const dir = worktree();
  const other = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-other-"));
  fs.writeFileSync(path.join(other, "g.txt"), "elsewhere\n");
  const tl = await tools(dir);
  const read = await tl.mrw_read.execute({ specs: ["g.txt"], root: other }, ctx(dir));
  assert.equal(read.metadata.exitCode, 0, read.output);
  assert.match(read.output, /1\| elsewhere/);
});

test("iter takes its verb and specs", async () => {
  process.env.XDG_STATE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-state-"));
  const dir = worktree();
  const tl = await tools(dir);
  const add = await tl.mrw_iter.execute({ args: ["add", "f.txt"] }, ctx(dir));
  assert.equal(add.metadata.exitCode, 0, add.output);
  const list = await tl.mrw_iter.execute({}, ctx(dir));
  assert.match(list.output, /f\.txt/);
});

test("without bin/mrw in the worktree, mrw is found on PATH", async () => {
  process.env.XDG_STATE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-state-"));
  const dir = worktree({ noBinary: true });
  const saved = process.env.PATH;
  process.env.PATH = path.dirname(built) + path.delimiter + bare;
  try {
    const tl = await tools(dir);
    const version = await tl.mrw_version.execute({}, ctx(dir));
    assert.equal(version.metadata.exitCode, 0, version.output);
    assert.match(version.output, /\S/);
  } finally {
    process.env.PATH = saved;
  }
});

test("every tool reports the exit code of a refusal", async () => {
  process.env.XDG_STATE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "mrw-state-"));
  const dir = worktree();
  const tl = await tools(dir);
  const seen = await tl.mrw_seen.execute({ dryRun: true }, ctx(dir));
  assert.equal(seen.metadata.exitCode, 2, seen.output);
  assert.match(seen.output, /^exit: 2/);
});
