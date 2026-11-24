# Code Mode probe for issue #790

Measured 2026-11-24 on macOS with Claude Code 2.1.331 (model `claude-fable-5-2`) and mrw v1.94.1 (`c0563ea`).
ADR-218 cites these files. Nothing here is run by a gate.

**The fixture.** A scratch git checkout holding `a.txt`, `b.txt` and `c.txt`, three lines each (`header X`,
`value OLD in X`, `footer X`), with project hooks on `PreToolUse` and `PostToolUse`, matcher empty, running the
logger. The session was started interactively with `CLAUDE_CODE_EXPERIMENTAL_CODE_MODE=1`, project settings
only, in auto mode, and its opt-in dialog accepted. `$S` in the logs is the scratch directory; `session_id`
and `prompt_id` are the scratch sessions' own.

| File | What it is |
|---|---|
| `hookfull.py` | the logger: the whole payload, strings cut at 1,500 characters |
| `hooks-interactive2.jsonl` | event, tool and input only (an earlier logger). One script ran `mrw read a.txt:1`, `echo probe-forbidden` and `mrw read b.txt:1` and returned a status per call. A second hook denied the middle call with exit 2 and added a line of context after each read |
| `hooks4-full.jsonl` | one script, two `Bash` calls: the environment an inner call sees, and `mrw read a.txt:1-2` with only line 1 returned. Shows `tool_use_id` beginning `repl_` on inner calls and `toolu_` on `REPL` and `Poll`, and what `REPL` and `Poll` return |
| `hooks5-full.jsonl` | one script calling the host's `Read` on `c.txt`: the `PostToolUse` payload `mrw hook read` would be given, with a `repl_` id |

**What is not in a file.** The model's answer that line 1 of `a.txt` did not reach it, and `mrw seen` listing
lines 1-3 of `a.txt` and `b.txt` as shown afterwards, are quoted in the issue; the terminal they were read
from was not saved. The headless comparison (the same 25 tools with the variable set and unset under
`claude -p`) was read from two `stream-json` init events that were not kept.

**Limits.** One machine, one Claude Code version, one model, auto mode only. The claim that the read text did
not reach the model rests on the script's code, which returns statuses only, and on the model's own answer.

**Added for ADR-218, 2026-11-24.** `env1.jsonl` and `hookenv.py`: one Code Mode session in a folder whose
`.claude/settings.json` held `"env": {"MRW_SHOWN": "off"}`. The inner `Bash` call printed the variable, and the
hook process held it on all six events, `REPL` and inner calls alike: a session setting reaches both mrw and its
hook. `pilot/` holds the pilot that decides whether the refusals under that variable are affordable: its protocol
(`README.md`, frozen before the build and amended once before any run), its fixture, runner, logging hook and
summariser, and `results.md` once it has run.
