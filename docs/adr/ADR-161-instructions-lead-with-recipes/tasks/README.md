# ADR-161 Tasks

Implementation tasks for ADR-161: mrw writes the plan itself, keeps the agent's context clean, and its instructions lead with recipes. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T6 | none |
| 4 | T7 | T6 |
| 5 | T8 | none |
| 6 | T3 | T1, T2, T6, T8 |
| 7 | T4 | T3 |
| 8 | T5 | T3 |
| 9 | T9 | T1, T2, T3, T6, T7 |

## Waves

| Wave | Tasks |
|------|-------|
| 1 | T1 |
| 2 | T2, T6, T8 |
| 3 | T7, T3 |
| 4 | T4, T5, T9 |

T1, T6, T7 and T8 all edit `internal/read/read.go` and `cmd/mrw/main.go`; they run one at a time, whatever the wave.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | read --sub prints a plan; write --grep --sub applies it | done | — | `go test …` and contract §263 |
| T2 | --rewrite with --ast-grep | pending | — | `go test …` and contract §264 |
| T6 | read --quiet licenses what it serves and prints no line | pending | — | `go test …` and contract §266 |
| T7 | a ranged read serves the line after each range | pending | — | `go test …` and contract §267 |
| T8 | mrw read - reads standard input | pending | — | `go test …` and contract §268 |
| T3 | the recipes in mrw instructions and --core | pending | — | `go test …` and contract §265 |
| T4 | AGENTS.md, the docs, the skill and the agent definitions carry the recipes | pending | — | the binary's recipe block in AGENTS.md; recipe 1 in the others |
| T5 | a paired B pilot | pending | — | `go run ./cmd/bench verify --stage pilot …` ×2 |
| T9 | the MCP surface takes the same tools | pending | — | `go test ./internal/mcp/ …` and contract §269 |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `--sub`, the plan source on write | T2, T3, T5 | T1 first |
| T2 | `--rewrite` | T3, T5 | before the recipes |
| T6 | `read --quiet` | T7, T3 | before the recipes |
| T8 | `read -` | T3 | before the recipes |
| T3 | the `Recipes:` block | T4, T5, T9 | before the mirrors, the pilot and the MCP handshake |
