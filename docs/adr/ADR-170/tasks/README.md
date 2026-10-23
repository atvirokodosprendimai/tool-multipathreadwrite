# ADR-170 tasks

Source of truth: the task files. This index is derived from them.

| Task | Goal | Depends-on | Status |
|------|------|------------|--------|
| T1 | `mrw mcp` serves no tools until `--enable` or `MRW_MCP=on` | none | done |
| T2 | the opencode plugin enables it, and the help, README and AGENTS.md say so | T1 | done |

Execution order: T1, T2. One pull request.

Status: `pending` | `partial` | `blocked` | `done`.
