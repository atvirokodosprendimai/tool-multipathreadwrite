# ADR-037 Tasks

Implementation tasks for ADR-037: The binary teaches the format it demands. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Shared sentences exist and MCP contains them | done | — | `go test ./internal/guide/ ./internal/mcp/ -run 'TestEverySurfaceContainsTheSharedSentences\|TestMCPInstructionsContainShared\|TestTheInstructionsTellAHostHowToAuthorAPlan'` |
| T2 | `mrw instructions` prints the CLI pamphlet | done | — | `go test ./cmd/mrw/ ./internal/guide/ -run 'TestInstructionsCommand\|TestEverySubcommandReachesTheAgentFacingGuide' && grep -q '^# 75\. ' scripts/contract.sh` |
| T3 | README and the skill name the command | done | — | `go test ./cmd/mrw/ -run 'TestReadmeAndSkillNameInstructions\|TestEverySubcommandReachesTheAgentFacingGuide'` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `guide.Shared()` | T2, T3 | T1 before T2 |
| T2 | `mrw instructions` subcommand | T3 | T2 before T3 |
