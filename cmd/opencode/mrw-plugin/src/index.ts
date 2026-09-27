/**
 * mrw-opencode-plugin (ADR-089) — exposes the mrw CLI as opencode tools. Each
 * tool spawns the mrw binary, so a tool refuses exactly what the CLI refuses,
 * and the read-before-write ledger is the one every other mrw caller uses.
 */

import { type Plugin, tool } from "@opencode-ai/plugin";
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const z = tool.schema;

// ---------------------------------------------------------------------------
// mrw subprocess runner
// ---------------------------------------------------------------------------

// resolveBinary prefers the checkout's own build — bin/mrw, bin/mrw.exe on
// Windows — because a PATH mrw may be older than the checkout, and falls back to
// mrw on PATH.
function resolveBinary(worktree: string): string {
  const local = path.join(worktree, "bin", process.platform === "win32" ? "mrw.exe" : "mrw");
  return fs.existsSync(local) ? local : "mrw";
}

type Run = { stdout: string; stderr: string; exitCode: number };

// runMrw runs mrw in the worktree with args, feeding input on stdin when given.
// An aborted call kills the child.
function runMrw(worktree: string, args: string[], abort: AbortSignal, input?: string): Promise<Run> {
  return new Promise((resolve, reject) => {
    const proc = spawn(resolveBinary(worktree), args, { cwd: worktree, stdio: ["pipe", "pipe", "pipe"], signal: abort });
    const stdout: Buffer[] = [];
    const stderr: Buffer[] = [];
    proc.stdout.on("data", (chunk: Buffer) => stdout.push(chunk));
    proc.stderr.on("data", (chunk: Buffer) => stderr.push(chunk));
    proc.on("error", reject);
    proc.on("close", (code: number | null) => {
      resolve({
        stdout: Buffer.concat(stdout).toString("utf8"),
        stderr: Buffer.concat(stderr).toString("utf8"),
        exitCode: code ?? -1,
      });
    });
    proc.stdin.end(input ?? "");
  });
}

// report is every tool's answer: the exit code first, because it is the whole
// verdict (a plan that failed validation wrote nothing), then stderr, then the
// output.
function report(title: string, r: Run) {
  const stderr = r.stderr ? `stderr: ${r.stderr.trimEnd()}\n` : "";
  return { title, output: `exit: ${r.exitCode}\n${stderr}${r.stdout || "(no output)"}`, metadata: { exitCode: r.exitCode } };
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

const toolRead = tool({
  description:
    "Read line ranges from files, or walk the tree with grep to find and serve matching ranges. " +
    "One call serves every site. A read licenses a later write to the lines it served. " +
    "Addresses: path:12-45, path:12, path:A,+N, path:$, path:/regexp/, path:/from/,/to/. " +
    "With grep, specs name the directories or files to search.",
  args: {
    specs: z.array(z.string()).optional(),
    grep: z.string().optional(),
    astGrep: z.string().optional(),
    exclude: z.array(z.string()).optional(),
    stat: z.boolean().optional(),
    context: z.number().optional(),
    maxLines: z.number().optional(),
    filesFrom: z.string().optional(),
    root: z.string().optional().describe("Another checkout to read; mrw takes it before the subcommand."),
  },
  async execute(args, ctx) {
    const mrwArgs: string[] = [];
    if (args.root) mrwArgs.push("--root", args.root);
    mrwArgs.push("read");
    if (args.grep) mrwArgs.push("--grep", args.grep);
    if (args.astGrep) mrwArgs.push("--ast-grep", args.astGrep);
    for (const e of args.exclude ?? []) mrwArgs.push("--exclude", e);
    if (args.stat) mrwArgs.push("--stat");
    if (args.context) mrwArgs.push("-C", String(args.context));
    if (args.maxLines !== undefined) mrwArgs.push("--max-lines", String(args.maxLines));
    if (args.filesFrom) mrwArgs.push("--files-from", args.filesFrom);
    mrwArgs.push("--", ...(args.specs ?? []));
    return report("mrw read", await runMrw(ctx.worktree, mrwArgs, ctx.abort));
  },
});

const toolWrite = tool({
  description:
    "Apply an edit plan across files. Every hunk gets a verdict. A plan that fails validation " +
    "writes nothing. Addresses resolve against the ORIGINAL file. " +
    "Ops: replace, insert-after, insert-before, delete, create, unlink, rename. " +
    "A new file is '@@ path 0 create'. A multi-line replace needs anchor=. " +
    "mrw will not edit a line it has not served — read it first. " +
    "After a code write it runs the project's check unless noCheck.",
  args: {
    plan: z.string().describe(
      "The plan document. Each hunk: '@@ <path> <address> <op> [guards]' + body lines.\n" +
        "Example:\n" +
        '@@ internal/apply/apply.go 42-58 replace anchor="func Apply" lines=17\n' +
        "        ... new lines ...\n" +
        "@@ cmd/mrw/main.go 12 insert-after\n" +
        '        "sort"',
    ),
    dryRun: z.boolean().optional(),
    noCheck: z.boolean().optional(),
    check: z.boolean().optional(),
    json: z.boolean().optional(),
    format: z.enum(["plan", "apply_patch", "search_replace"]).optional(),
  },
  async execute(args, ctx) {
    const mrwArgs = ["write"];
    if (args.dryRun) mrwArgs.push("--dry-run");
    if (args.noCheck) mrwArgs.push("--no-check");
    if (args.check) mrwArgs.push("--check");
    if (args.json) mrwArgs.push("--json");
    if (args.format) mrwArgs.push("--format", args.format);
    mrwArgs.push("-"); // the plan arrives on stdin; nothing is written into the checkout
    return report("mrw write", await runMrw(ctx.worktree, mrwArgs, ctx.abort, args.plan));
  },
});

const toolCheck = tool({
  description:
    "Run the project's declared check, scoped to the working set or to the paths you name. " +
    "NOT read-only: the declared command may generate code or write fixtures.",
  args: {
    paths: z.array(z.string()).optional(),
  },
  async execute(args, ctx) {
    return report("mrw check", await runMrw(ctx.worktree, ["check", ...(args.paths ?? [])], ctx.abort));
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
    return report("mrw stats", await runMrw(ctx.worktree, args.json ? ["stats", "--json"] : ["stats"], ctx.abort));
  },
});

const toolSeen = tool({
  description:
    "Print the read-before-modify ledger: which lines have been served, which is what licenses " +
    "a write. Use when a write is refused for a line you believe you read. prune removes the " +
    "state of checkouts that are gone; dryRun with prune shows what it would remove.",
  args: {
    prune: z.boolean().optional(),
    dryRun: z.boolean().optional(),
  },
  async execute(args, ctx) {
    const mrwArgs = ["seen"];
    if (args.prune) mrwArgs.push("--prune");
    if (args.dryRun) mrwArgs.push("--dry-run");
    return report("mrw seen", await runMrw(ctx.worktree, mrwArgs, ctx.abort));
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
    return report("mrw iter", await runMrw(ctx.worktree, ["iter", ...(args.args ?? [])], ctx.abort));
  },
});

const toolVersion = tool({
  description: "Print mrw's version string.",
  args: {},
  async execute(_args, ctx) {
    return report("mrw version", await runMrw(ctx.worktree, ["version"], ctx.abort));
  },
});

const toolInstructions = tool({
  description:
    "Print the contract from the binary: use mrw always and plan the activity, the two rules " +
    "that produce most refusals, the traps that make a red run look green, the plan ops, and " +
    "the read side.",
  args: {},
  async execute(_args, ctx) {
    return report("mrw instructions", await runMrw(ctx.worktree, ["instructions"], ctx.abort));
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
