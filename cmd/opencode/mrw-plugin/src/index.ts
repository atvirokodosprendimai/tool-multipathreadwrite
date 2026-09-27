/**
 * mrw-opencode-plugin — spawns the mrw binary and exposes its CLI commands
 * as opencode tools. Each command becomes a tool with a Zod schema, proper
 * description, and structured output.
 */

import { Plugin, tool } from "@opencode-ai/plugin";
import { z } from "zod";
import { spawn } from "node:child_process";
import path from "node:path";
import fs from "node:fs";

// ---------------------------------------------------------------------------
// mrw subprocess runner
// ---------------------------------------------------------------------------

function resolveBinary(directory: string): string {
  const local = path.join(directory, "bin", "mrw.exe");
  return local;
}

function runMrw(binary: string, args: string[], cwd: string): Promise<{
  stdout: string;
  stderr: string;
  exitCode: number;
}> {
  return new Promise((resolve, reject) => {
    const proc = spawn(binary, args, {
      cwd,
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env, CLAUDE_PROJECT_DIR: cwd },
    });

    const stdoutChunks: Buffer[] = [];
    const stderrChunks: Buffer[] = [];

    proc.stdout.on("data", (chunk: Buffer) => stdoutChunks.push(chunk));
    proc.stderr.on("data", (chunk: Buffer) => stderrChunks.push(chunk));

    proc.on("close", (code: number | null) => {
      resolve({
        stdout: Buffer.concat(stdoutChunks).toString("utf8"),
        stderr: Buffer.concat(stderrChunks).toString("utf8"),
        exitCode: code ?? -1,
      });
    });

    proc.on("error", (err: Error) => {
      reject(err);
    });
  });
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

const toolRead = tool({
  description:
    "Read line ranges from files, or walk the tree with --grep to find and serve matching ranges. " +
    "One call serves every site. Use before editing to license writes. " +
    "Addresses: path:12-45, path:12, path:A,+N, path:$, path:/regexp/, path:/from/,/to/. " +
    "With --grep, specs become directories/files to search. Returns served content plus a receipt.",
  args: {
    specs: z.array(z.string()).optional(),
    grep: z.string().optional(),
    astGrep: z.string().optional(),
    exclude: z.array(z.string()).optional(),
    stat: z.boolean().optional(),
    context: z.number().optional(),
    maxLines: z.number().optional(),
    filesFrom: z.string().optional(),
    root: z.string().optional(),
  },
  async execute(args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const mrwArgs: string[] = ["read"];

    if (args.specs?.length) {
      for (const s of args.specs) mrwArgs.push(s);
    }
    if (args.grep) mrwArgs.push("--grep", args.grep);
    if (args.astGrep) mrwArgs.push("--ast-grep", args.astGrep);
    if (args.exclude?.length) {
      for (const e of args.exclude) mrwArgs.push("--exclude", e);
    }
    if (args.stat) mrwArgs.push("--stat");
    if (args.context) mrwArgs.push("-C", String(args.context));
    if (args.maxLines !== undefined) mrwArgs.push("--max-lines", String(args.maxLines));
    if (args.filesFrom) mrwArgs.push("--files-from", args.filesFrom);
    if (args.root) mrwArgs.push("--root", args.root);

    const result = await runMrw(binary, mrwArgs, ctx.worktree);
    const header = `exit: ${result.exitCode}\n${result.stderr ? "stderr: " + result.stderr + "\n" : ""}`;
    return {
      title: "mrw read",
      output: header + (result.stdout || "(no output)"),
      metadata: { exitCode: result.exitCode },
    };
  },
});

const toolWrite = tool({
  description:
    "Apply an edit plan across files. Every hunk gets a verdict. All-or-nothing: " +
    "if any hunk fails, nothing is written. Addresses resolve against the ORIGINAL file. " +
    "Ops: replace, insert-after, insert-before, delete, create, unlink, rename. " +
    "A new file is '@@ path 0 create'. Requires anchor= on multi-line replace. " +
    "mrw will not edit a line it has not served — read it first.",
  args: {
    plan: z.string().describe(
      "The plan document. Each hunk: '@@ <path> <address> <op> [guards]' + body lines.\n" +
      "Example:\n" +
      '@@ internal/apply/apply.go 42-58 replace anchor="func Apply" lines=17\n' +
      "        ... new lines ...\n" +
      '@@ cmd/mrw/main.go 12 insert-after\n' +
      '        "sort"'
    ),
    dryRun: z.boolean().optional(),
    noCheck: z.boolean().optional(),
    json: z.boolean().optional(),
    format: z.enum(["plan", "apply_patch", "search_replace"]).optional(),
    check: z.boolean().optional(),
  },
  async execute(args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const mrwArgs: string[] = ["write"];

    if (args.dryRun) mrwArgs.push("--dry-run");
    if (args.noCheck) mrwArgs.push("--no-check");
    if (args.json) mrwArgs.push("--json");
    if (args.check) mrwArgs.push("--check");
    if (args.format) mrwArgs.push("--format", args.format);

    const tmpFile = path.join(ctx.worktree, ".mrw-plan-input");
    fs.writeFileSync(tmpFile, args.plan, "utf8");

    try {
      const result = await runMrw(binary, mrwArgs, ctx.worktree);
      const header = `exit: ${result.exitCode}\n${result.stderr ? "stderr: " + result.stderr + "\n" : ""}`;
      return {
        title: "mrw write",
        output: header + (result.stdout || "(no output)"),
        metadata: { exitCode: result.exitCode },
      };
    } finally {
      fs.unlinkSync(tmpFile);
    }
  },
});

const toolCheck = tool({
  description:
    "Run the project's check (go test, pytest, etc.) scoped to the working set " +
    "or to paths you name. NOT read-only: the declared command may generate code. " +
    "Exit 0 = passed, 1 = failed, 2 = miss (no check found).",
  args: {
    paths: z.array(z.string()).optional(),
  },
  async execute(args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const mrwArgs = ["check"];
    if (args.paths?.length) {
      for (const p of args.paths) mrwArgs.push(p);
    }

    const result = await runMrw(binary, mrwArgs, ctx.worktree);
    return {
      title: "mrw check",
      output: `exit: ${result.exitCode}\n${result.stderr || ""}${result.stdout || "(no output)"}`,
      metadata: { exitCode: result.exitCode },
    };
  },
});

const toolStats = tool({
  description:
    "Print what became of the plans this checkout has been given — applied, " +
    "refused, failed_check, check_not_run, plus landed writes and failed checks. " +
    "Also prints recent-window pattern and strict-balance pricing.",
  args: {
    json: z.boolean().optional(),
  },
  async execute(args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const mrwArgs = ["stats"];
    if (args.json) mrwArgs.push("--json");

    const result = await runMrw(binary, mrwArgs, ctx.worktree);
    return {
      title: "mrw stats",
      output: result.stdout || "(no output)",
      metadata: {},
    };
  },
});

const toolSeen = tool({
  description:
    "Print the read-before-modify ledger: which lines have been served to callers, " +
    "which is what licenses a write. Also shows the state directory path and count. " +
    "Use when a write is refused for a line you believe you read.",
  args: {
    prune: z.boolean().optional(),
    dryRun: z.boolean().optional(),
  },
  async execute(args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const mrwArgs = ["seen"];
    if (args.prune) mrwArgs.push("--prune");
    if (args.dryRun) mrwArgs.push("--dry-run");

    const result = await runMrw(binary, mrwArgs, ctx.worktree);
    return {
      title: "mrw seen",
      output: result.stdout || "(no output)",
      metadata: {},
    };
  },
});

const toolIter = tool({
  description:
    "Show or edit the working set: the specs mrw is currently carrying. " +
    "The working set is the index that write resolves hunk paths against.",
  args: {
    set: z.array(z.string()).optional(),
  },
  async execute(args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const mrwArgs = ["iter"];
    if (args.set?.length) {
      for (const s of args.set) mrwArgs.push(s);
    }

    const result = await runMrw(binary, mrwArgs, ctx.worktree);
    return {
      title: "mrw iter",
      output: result.stdout || "(no output)",
      metadata: { exitCode: result.exitCode },
    };
  },
});

const toolVersion = tool({
  description:
    "Print mrw's version string. Extra arguments are usage errors.",
  args: {},
  async execute(_args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const result = await runMrw(binary, ["version"], ctx.worktree);
    return {
      title: "mrw version",
      output: result.stdout.trim() || result.stderr,
      metadata: {},
    };
  },
});

const toolInstructions = tool({
  description:
    "Print the contract from the binary: use mrw always and plan the activity, " +
    "the two rules that produce most refusals, the traps that make a red run " +
    "look green, the plan ops, and the read side. Exit 0. No flags.",
  args: {},
  async execute(_args, ctx) {
    const binary = resolveBinary(ctx.worktree);
    const result = await runMrw(binary, ["instructions"], ctx.worktree);
    return {
      title: "mrw instructions",
      output: result.stdout || result.stderr,
      metadata: {},
    };
  },
});

// ---------------------------------------------------------------------------
// Plugin entry point
// ---------------------------------------------------------------------------

export const mrwPlugin: Plugin = async (input) => {
  return {
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
    async dispose() {
      // Nothing to clean up
    },
  };
};
