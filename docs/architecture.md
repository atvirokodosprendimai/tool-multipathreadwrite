# Architecture: tool-multipathreadwrite (mrw)

**Status:** Living — every ADR that changes structure updates this doc in the same commit.
**Repo:** github.com/atvirokodosprendimai/tool-multipathreadwrite
**Tier:** library (one Go module, four binaries: `mrw`, `bench`, `curve`, `nativebench`)
**Gate command:** `go test ./internal/adversarial/ ./cmd/mrw/ ./internal/mcp/ -count=1 -run 'TestTheEngineImportsNoSurface$|TestEachSurfaceHasOneRoot$|TestOnlyTheseStartAProcess$|TestEveryModulePackageIsInTheModuleMap$|TestOnlyTheCheckLatestFileImportsTheNetwork$|TestNoShippedReceiptFieldDisappears$|TestNoShippedMCPReceiptFieldDisappears$|TestTheTwoSurfacesCountAWriteAlike$'`
**Last full audit:** 2026-10-20, scouted from `go list ./...` at b89dafa7

The current state the ADRs in `docs/adr/` are deltas against. AGENTS.md says how to use and change
the code; this says what is where and which direction it may depend. Every rule row names the
check that fails when the rule breaks.

For existing write guarantees, focused tests, binary contract rows and limits,
see [Safety evidence for the write path](notes/safety-evidence.md).

## Module Map

The engine is the six packages AGENTS.md's drill names (`read`, `apply`, `plan`, `seen`, `check`,
`state`); the surfaces are the two composition roots that serve it (`cmd/mrw` for the CLI,
`internal/mcp` for MCP) and `internal/writer`, the write flow both of them call.

| Module | Layer | One reason to change | Owner |
|--------|-------|----------------------|-------|
| `cmd/mrw` | api | the CLI's commands, flags, help and human receipts | ADR-161, ADR-069, ADR-092 |
| `internal/mcp` | api | the MCP tools, their schemas, checkpoints and the result ceiling | ADR-023, ADR-031, ADR-113 |
| `internal/writer` | domain | the write flow shared by both surfaces: lock, apply, check, steps, drift | ADR-075, ADR-112, ADR-127 |
| `internal/plan` | domain | the plan grammar: headers, guards, bodies | ADR-013, ADR-060, ADR-109 |
| `internal/apply` | domain | validating a whole plan, staging writes and reporting what the filesystem commit landed | ADR-001, ADR-066, ADR-132 |
| `internal/read` | domain | serving ranges, `--grep`, `--ast-grep`, and building the `--sub`/`--rewrite` plans | ADR-007, ADR-104, ADR-161 |
| `internal/seen` | domain | the read-before-modify ledger and the write lock | ADR-002, ADR-075, ADR-150 |
| `internal/check` | domain | the project check and declared steps | ADR-003, ADR-054, ADR-092 |
| `internal/state` | infra | where mrw's own state lives outside the tree | ADR-004, ADR-077, ADR-110 |
| `internal/rooted` | infra | the root boundary: what a path names and whether it may be touched | ADR-007, ADR-071, ADR-081, ADR-143 |
| `internal/links` | infra | following symlinks and junctions | ADR-071, ADR-103 |
| `internal/subproc` | infra | child processes and their process group | ADR-095, ADR-120 |
| `internal/regular` | infra | opening only regular files, without blocking | ADR-109 |
| `internal/addr` | domain | the address grammar shared by read specs and plan hunks | ADR-026 |
| `internal/lines` | domain | line splitting and numbering for every surface | ADR-065, ADR-073 |
| `internal/ingest` | adapter | compiling `apply_patch` and `search_replace` to a native plan | ADR-069 |
| `internal/iter` | domain | the working set | ADR-079 |
| `internal/authoring` | domain | plan outcomes and the advisory pattern (`mrw stats`) | ADR-055, ADR-056, ADR-102 |
| `internal/guide` | domain | the teaching sentences both surfaces print | ADR-037, ADR-063, ADR-141 |
| `internal/preview` | domain | a plan maker's dry-run display | ADR-164 |
| `internal/statehint` | domain | naming a failure inside the state directory | ADR-164 |
| `internal/refusal` | domain | refusal kinds shared by plan, apply and mcp | ADR-087, ADR-118 |
| `internal/speclist` | adapter | `--files-from` and `files_from` lists | ADR-117 |
| `internal/adversarial` | test | tests that hold the promises, not the code | ADR-015 |
| `cmd/bench`, `internal/bench` | tooling | the agent benchmark | ADR-156, ADR-162 |
| `cmd/curve`, `internal/curve` | tooling | the read/write curve harness | ADR-020 |
| `cmd/nativebench` | tooling | the native performance baseline driver, run as `docs/bench/native-baseline/PREREGISTRATION.md` registers | #558, #565 |
| `cmd/opencode/mrw-plugin` | adapter | the opencode plugin, which spawns the `mrw` binary | ADR-089 |

## Dependency Contracts

| # | Rule | Check |
|---|------|-------|
| D1 | No engine package reaches a command, `internal/mcp`, `internal/writer`, `internal/bench` or `internal/curve` | `go test ./internal/adversarial/ -run 'TestTheEngineImportsNoSurface$'` in `internal/adversarial/architecture_test.go` |
| D2 | `internal/mcp` is imported only by `cmd/mrw`, `internal/bench` only by `cmd/bench`, `internal/curve` only by `cmd/curve`; the `mrw` binary reaches neither benchmark | `go test ./internal/adversarial/ -run 'TestEachSurfaceHasOneRoot$'` in `internal/adversarial/architecture_test.go` |
| D3 | Inside the `mrw` binary only `internal/check`, `internal/read` and `internal/subproc` import `os/exec` | `go test ./internal/adversarial/ -run 'TestOnlyTheseStartAProcess$'` in `internal/adversarial/architecture_test.go` |
| D4 | Only `cmd/mrw/latest149.go` imports the network, and only for `version --check-latest` | `go test ./cmd/mrw/ -run 'TestOnlyTheCheckLatestFileImportsTheNetwork$'` in `cmd/mrw/latest149_test.go` |
| D5 | Unused code, and production code only a test reaches, is deleted | `./scripts/static.sh` (deadcode, U1000 with `-tests=false`) |
| D6 | Every package `go list ./...` finds has a Module Map row, and every `cmd/` or `internal/` path a row names is in the tree | `go test ./internal/adversarial/ -run 'TestEveryModulePackageIsInTheModuleMap$'` in `internal/adversarial/architecture_test.go` |

## Concept Ownership (DRY)

| Concept | Single source | Consumers | Divergence check |
|---------|---------------|-----------|------------------|
| Receipt keys (`--json`, `mrw_write`) | `docs/receipts.txt` | `cmd/mrw`, `internal/mcp` | `go test ./cmd/mrw/ -run 'TestNoShippedReceiptFieldDisappears$'` in `cmd/mrw/receipts111_test.go`, and `go test ./internal/mcp/ -run 'TestNoShippedMCPReceiptFieldDisappears$'` in `internal/mcp/receipts111_test.go` |
| How a write is counted | `internal/authoring` | `cmd/mrw`, `internal/mcp` | `go test ./cmd/mrw/ -run 'TestTheTwoSurfacesCountAWriteAlike$'` in `cmd/mrw/surfaces113_test.go` |
| Teaching sentences | `internal/guide` | `mrw instructions`, the MCP instructions | `./scripts/contract.sh` (§43 holds the two to one text) |
| Exit codes | AGENTS.md §Exit codes | every command | `./scripts/contract.sh` |
| The write receipt struct | two copies: `cmd/mrw/main.go` `receipt`, `internal/mcp/tools.go` `writeReceipt` | the two surfaces | (deferred: docs/adr/BACKLOG.md "Architecture audit 2026-10-20") |
| A gate's refusal text | two copies of `gateRefusal`: `cmd/mrw/main.go`, `internal/mcp/tools.go` | the two surfaces | (deferred: docs/adr/BACKLOG.md "Architecture audit 2026-10-20") |
| Case folding of a path | two copies of `foldKey`: `internal/read/walk.go`, `internal/apply/apply.go` | walk, apply | (deferred: docs/adr/BACKLOG.md "Architecture audit 2026-10-20") |

## Composition Root

| Root | Constructs | Check |
|------|------------|-------|
| `cmd/mrw/main.go` `main` | the CLI command tree; the MCP server (`mcp.Serve`, with the `MaxResultChars` and `Version` globals set first) | `go test ./internal/adversarial/ -run 'TestEachSurfaceHasOneRoot$'` in `internal/adversarial/architecture_test.go` |
| `cmd/bench/main.go` | the agent benchmark | `go test ./internal/adversarial/ -run 'TestEachSurfaceHasOneRoot$'` in `internal/adversarial/architecture_test.go` |
| `cmd/curve/main.go` | the curve harness | `go test ./internal/adversarial/ -run 'TestEachSurfaceHasOneRoot$'` in `internal/adversarial/architecture_test.go` |
| `cmd/nativebench/main.go` | the native baseline driver, which runs the two `mrw` binaries it is handed | `go test ./internal/adversarial/ -run 'TestEveryModulePackageIsInTheModuleMap$'` in `internal/adversarial/architecture_test.go` (D6: its Module Map row; it imports no surface package, so D2 has nothing to say about it) |

## Test Doubles

| Fake | Stands in for | Contract (what must hold) | Check |
|------|---------------|---------------------------|-------|
| `fakeLinks` in `internal/rooted/links_test.go` | Windows links and junctions (`links.FS`) | each entry kind (`dir`, `link`, `placeholder`, `unreadable`, `denied`, `brokensymlink`, `invalid`) answers as Win32 does, so `throughLinks` is driven on every platform | `go test ./internal/rooted/` on the Windows CI job, against real junctions |
| a fake `ast-grep` binary built in `internal/adversarial` | the `ast-grep` binary | hangs, exits and prints as the real one can, for the timeout and refusal paths | `go test ./internal/adversarial/` |

## Trust & Data Boundaries

| Boundary | Crossing rule | Check |
|----------|---------------|-------|
| The root | every path a caller names resolves through `rooted.Resolve`; a path outside the root, through a link out of it, or by a Win32 alias is refused | `go test ./internal/rooted/ ./internal/adversarial/` and `./scripts/contract.sh` |
| mrw's state directory | nothing inside it, or hard-linked to it, is served or written | `go test ./internal/rooted/` and `./scripts/contract.sh` (ADR-077, ADR-134 rows) |
| `.git` | a write through any `.git` component is refused; a read is served | `./scripts/contract.sh` (ADR-143 rows) |
| Windows device names | a reserved name in any component is refused on every build | `go test ./internal/rooted/` (ADR-081) |
| Shell execution | only the project's declared check and steps run, through `internal/check` with `sh -c`, and `--then-sh`, which runs what the caller gave it | D3, and `./scripts/contract.sh` (ADR-092, ADR-095 rows) |
| The network | none, except `version --check-latest` when asked | D4: `go test ./cmd/mrw/ -run 'TestOnlyTheCheckLatestFileImportsTheNetwork$'` in `cmd/mrw/latest149_test.go` |

## Superseded

None yet
