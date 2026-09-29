/**
 * mrw-opencode-plugin (ADR-089) — mrw as opencode tools.
 *
 * mrw_read and mrw_write go through `mrw mcp`, the surface built for a host that
 * can cut a result: a page stays under opencode's output limit, carries
 * checkpoints, and licenses nothing until the caller acknowledges the runs it
 * received whole. A CLI read would license every line it served, and opencode
 * truncates a tool's output at 2,000 lines or 50 KiB, so the hidden tail would
 * be writable unseen (ADR-002 inverted). The other tools run the CLI.
 */

import { type Plugin, type ToolContext, tool } from "@opencode-ai/plugin";
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const z = tool.schema;

// CEILING bounds an mrw_read or mrw_write result in encoded characters. It sits
// under opencode's 50 KiB truncation; a page cut anyway (the 2,000-line limit)
// loses the close markers of its tail, and those runs cannot be acknowledged.
const CEILING = 40_000;

// ---------------------------------------------------------------------------
// Running mrw
// ---------------------------------------------------------------------------

// root is the checkout mrw works in: the worktree, or the session directory
// when opencode has no git worktree and reports "/".
function root(ctx: ToolContext): string {
  return ctx.worktree && ctx.worktree !== "/" ? ctx.worktree : ctx.directory;
}

// resolveBinary prefers the checkout's own build — bin/mrw, bin/mrw.exe on
// Windows — because a PATH mrw may be older than the checkout, and falls back to
// mrw on PATH.
function resolveBinary(dir: string): string {
  const local = path.join(dir, "bin", process.platform === "win32" ? "mrw.exe" : "mrw");
  return fs.existsSync(local) ? local : "mrw";
}

type Run = { stdout: string; stderr: string; exitCode: number };

// run spawns mrw in dir with args, feeds input on stdin and collects the
// output. A child that exits without reading its input is reported by its exit
// code, not raised; an aborted call kills the child.
function run(dir: string, args: string[], abort: AbortSignal, input = ""): Promise<Run> {
  return new Promise((resolve, reject) => {
    const proc = spawn(resolveBinary(dir), args, { cwd: dir, stdio: ["pipe", "pipe", "pipe"], signal: abort });
    const stdout: Buffer[] = [];
    const stderr: Buffer[] = [];
    proc.stdout.on("data", (chunk: Buffer) => stdout.push(chunk));
    proc.stderr.on("data", (chunk: Buffer) => stderr.push(chunk));
    proc.stdin.on("error", () => {}); // EPIPE: the child stopped reading; its exit code says why
    proc.on("error", reject);
    proc.on("close", (code: number | null) => {
      resolve({
        stdout: Buffer.concat(stdout).toString("utf8"),
        stderr: Buffer.concat(stderr).toString("utf8"),
        exitCode: code ?? -1,
      });
    });
    proc.stdin.end(input);
  });
}

// cli runs one mrw subcommand and reports it: the exit code first, because it
// is the whole verdict, then stderr, then the output.
async function cli(title: string, ctx: ToolContext, args: string[]) {
  const r = await run(root(ctx), args, ctx.abort);
  const stderr = r.stderr ? `stderr: ${r.stderr.trimEnd()}\n` : "";
  return { title, output: `exit: ${r.exitCode}\n${stderr}${r.stdout || "(no output)"}`, metadata: { exitCode: r.exitCode } };
}

// mcp calls one tool of `mrw mcp` — a single JSON-RPC tools/call on stdin —
// and reports its content blocks. An error result says so on its first line.
async function mcp(title: string, ctx: ToolContext, dir: string, name: string, args: Record<string, unknown>) {
  const request = { jsonrpc: "2.0", id: 1, method: "tools/call", params: { name, arguments: args } };
  // The binary is the session checkout's; --root names the checkout it serves.
  const r = await run(root(ctx), ["--root", dir, "mcp", "--max-result-chars", String(CEILING)], ctx.abort, JSON.stringify(request) + "\n");
  let response: { result?: { content?: { text?: string }[]; isError?: boolean }; error?: { message?: string } } | undefined;
  for (const line of r.stdout.split("\n")) {
    try {
      const msg = JSON.parse(line);
      if (msg.id === 1) response = msg;
    } catch {
      // not a JSON-RPC line
    }
  }
  if (!response?.result) {
    const why = response?.error?.message ?? (r.stderr.trim() || `mrw mcp exited ${r.exitCode} without an answer`);
    return { title, output: `error: ${why}`, metadata: { isError: true } };
  }
  const text = (response.result.content ?? []).map((c) => c.text ?? "").join("\n\n");
  const isError = response.result.isError === true;
  return { title, output: (isError ? "error: the call was refused\n" : "") + text, metadata: { isError } };
}

// set copies the fields the caller gave; an empty string or a zero is given.
function set(from: Record<string, unknown>, map: Record<string, string>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, to] of Object.entries(map)) if (from[k] !== undefined) out[to] = from[k];
  return out;
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

const toolRead = tool({
  description:
    "Use for every file read in this checkout, in place of other read tools: one call serves " +
    "every site. Read line ranges from files, or find them with grep. Addresses: path:12-45, path:12, " +
    "path:A,+N, path:$, path:/regexp/, path:/from/,/to/. A served read licenses NOTHING until " +
    "acknowledged: the answer brackets each run of lines with '-- ck <id> open lines A-B (N lines " +
    "follow)' and '-- ck <id> close'; send an id in ack (here or on mrw_write) only if you hold " +
    "BOTH its markers and counted its N lines. A large read is a PAGE: send next_read back as specs.",
  args: {
    specs: z
      .array(z.string())
      .optional()
      .describe(
        "Range specs, e.g. internal/x/y.go:40-60 or path:/regexp/. With grep or astGrep set, these are " +
          "the directories or files to search instead, carry no range, and may be omitted to search " +
          "the whole checkout.",
      ),
    grep: z
      .string()
      .optional()
      .describe(
        "A regexp. Walks specs (or the whole checkout) and serves every matching range, to find files " +
          "you cannot name. Too many matches return an INDEX, one spec per file and no content: send " +
          "the ones you want back as specs.",
      ),
    astGrep: z
      .string()
      .optional()
      .describe("A structural pattern for the ast-grep CLI on PATH; walks like grep. Not together with grep."),
    exclude: z
      .array(z.string())
      .optional()
      .describe(
        "Globs to skip, matched against the root-relative path and the basename ('*_test.go', " +
          "'vendor'). Only with grep or astGrep: without them it is refused.",
      ),
    after: z.string().optional().describe("The next_index of an INDEX page, to fetch the next page."),
    ack: z
      .array(z.string())
      .optional()
      .describe(
        "ck ids from an earlier read whose open and close markers you both hold and whose N lines " +
          "you counted. Acknowledging licenses those lines for mrw_write.",
      ),
    root: z.string().optional().describe("Another checkout to read instead of this one."),
  },
  async execute(args, ctx) {
    const dir = args.root ?? root(ctx);
    const call = set(args, { specs: "specs", grep: "grep", astGrep: "ast_grep", exclude: "exclude", after: "after", ack: "ack" });
    return mcp("mrw read", ctx, dir, "mrw_read", call);
  },
});

const toolWrite = tool({
  description:
    "Use for every edit, create, delete or rename in this checkout, in place of other edit tools: " +
    "put every site in ONE plan. Every hunk gets a verdict; a plan that fails validation " +
    "writes nothing. Addresses resolve against the ORIGINAL file. Ops: replace, insert-after, " +
    "insert-before, delete, create, unlink, rename. A new file is '@@ path 0 create'. A " +
    "multi-line replace needs anchor= AND a served line after its range: read past the end " +
    "first. mrw will not edit a line it has not served AND you have acknowledged: pass the ck " +
    "ids from mrw_read in ack. It runs no check; call mrw_check with the files you wrote.",
  args: {
    plan: z.string().describe(
      "The plan document. Each hunk: '@@ <path> <address> <op> [guards]' + body lines.\n" +
        "Example:\n" +
        '@@ internal/apply/apply.go 42-58 replace anchor="func Apply" lines=17\n' +
        "        ... new lines ...\n" +
        "@@ cmd/mrw/main.go 12 insert-after\n" +
        '        "sort"',
    ),
    ack: z
      .array(z.string())
      .optional()
      .describe("The ck ids from the mrw_read this plan was written against. A hunk on lines you have not acknowledged is refused."),
    dryRun: z.boolean().optional().describe("Validate and report without writing: the same receipt, with dry_run true."),
    format: z
      .enum(["plan", "apply_patch", "search_replace"])
      .optional()
      .describe(
        "plan (default) is the native @@ document; apply_patch compiles a Codex *** Begin Patch " +
          "document; search_replace compiles an Aider SEARCH/REPLACE document.",
      ),
    echoPad: z
      .number()
      .optional()
      .describe("Print N lines after each applied body so a surviving closer is visible. Not a checker. Default 0."),
    strictBalance: z
      .boolean()
      .optional()
      .describe(
        "Refuse a single-line replace on a code path whose {} () [] do not balance against the line " +
          "it replaces (the wrap-tail shape). Off by default.",
      ),
  },
  async execute(args, ctx) {
    const call = set(args, { plan: "plan", ack: "ack", dryRun: "dry_run", format: "format", echoPad: "echo_pad", strictBalance: "strict_balance" });
    return mcp("mrw write", ctx, root(ctx), "mrw_write", call);
  },
});

const toolCheck = tool({
  description:
    "Run the project's declared check, scoped to the working set or to the paths you name. " +
    "After mrw_write, name the files it wrote: with no paths the check covers the working set " +
    "(mrw_iter), which a write does not change. NOT read-only: the declared command may " +
    "generate code or write fixtures.",
  args: {
    paths: z.array(z.string()).optional().describe("Files or directories to check; omit for the working set."),
  },
  async execute(args, ctx) {
    return cli("mrw check", ctx, ["check", "--", ...(args.paths ?? [])]);
  },
});

const toolStats = tool({
  description:
    "Print what became of the plans this checkout has been given — applied, refused, " +
    "failed_check, check_not_run — plus the recent-window pattern and strict-balance pricing.",
  args: {
    json: z.boolean().optional(),
  },
  async execute(args, ctx) {
    return cli("mrw stats", ctx, args.json ? ["stats", "--json"] : ["stats"]);
  },
});

const toolSeen = tool({
  description:
    "Print the read-before-modify ledger: which lines are licensed for a write. Use when a " +
    "write is refused for a line you believe you read. prune removes the state of checkouts " +
    "that are gone; dryRun with prune shows what it would remove.",
  args: {
    prune: z.boolean().optional(),
    dryRun: z.boolean().optional(),
  },
  async execute(args, ctx) {
    const mrwArgs = ["seen"];
    if (args.prune) mrwArgs.push("--prune");
    if (args.dryRun) mrwArgs.push("--dry-run");
    return cli("mrw seen", ctx, mrwArgs);
  },
});

const toolIter = tool({
  description:
    "Show or edit the working set, the specs mrw is carrying. No args prints it; otherwise " +
    "args is a verb and its specs: ['add', 'a.go:10-20'], ['rm', 'a.go'], ['clear'], ['note', 'text'].",
  args: {
    args: z.array(z.string()).optional(),
  },
  async execute(args, ctx) {
    return cli("mrw iter", ctx, ["iter", ...(args.args ?? [])]);
  },
});

const toolVersion = tool({
  description: "Print mrw's version string.",
  args: {},
  async execute(_args, ctx) {
    return cli("mrw version", ctx, ["version"]);
  },
});

const toolInstructions = tool({
  description:
    "Print the contract from the binary: use mrw always and plan the activity, the two rules " +
    "that produce most refusals, the traps that make a red run look green, the plan ops, and " +
    "the read side.",
  args: {},
  async execute(_args, ctx) {
    return cli("mrw instructions", ctx, ["instructions"]);
  },
});

// ---------------------------------------------------------------------------
// Plugin entry point
// ---------------------------------------------------------------------------

export const mrwPlugin: Plugin = async () => ({
  tool: {
    mrw_read: toolRead,
    mrw_write: toolWrite,
    mrw_check: toolCheck,
    mrw_stats: toolStats,
    mrw_seen: toolSeen,
    mrw_iter: toolIter,
    mrw_version: toolVersion,
    mrw_instructions: toolInstructions,
  },
});
