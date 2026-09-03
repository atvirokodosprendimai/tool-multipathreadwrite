# Contributing

## Prerequisites

| for | you need |
|---|---|
| building and testing | **Go 1.26.6 or newer** (the version in `go.mod`). One dependency, no cgo. |
| `scripts/measure.sh`, `scripts/contract.sh` | **bash**, **git**, **bc**. `bc` is absent from Alpine and most slim images: `apk add bc`. |
| either script on Windows | **WSL** or **Git Bash**. They are POSIX shell, not PowerShell. The binary itself is native. |

## The gates

CI runs these on every push and pull request, and a release publishes nothing
until they are green. Run them locally in the same order:

```sh
gofmt -l .            # must print NOTHING; `gofmt -l` exits 0 even when it lists files
go vet ./...
go test ./...
go test -race ./...
./scripts/contract.sh
```

Go builds and tests run on **Linux and Windows** in CI; `contract.sh` runs on
Linux, because it drives a POSIX shell.

**Never read an exit code through a pipe.** `./scripts/contract.sh | tail -5`
reports `tail`'s status, not the script's — a red run reads as green. Capture the
output and check `$?`, or let the command stand alone.

## What a change needs

- **A new promise needs a contract row.** `scripts/contract.sh` asserts each
  claim by making it go wrong on purpose. A row that passes without exercising
  anything is the defect class this file exists to prevent — see the `skip()`
  helper, used where the trigger is a permission bit that root ignores.
- **A durable decision needs an ADR** under `docs/adr/`: a change to a public
  contract, an exit code, persistent state, or a trust boundary. Bounded
  implementation does not.
- **Doc comments on exported identifiers**, starting with the name; a package
  comment per package; a short *why* comment on anything non-obvious. Comments
  that restate the code are noise.
- **One logical change per commit.** The message says why, not what.

## Reproducing the numbers in the README

```sh
./scripts/measure.sh
```

It builds its own binary from the working tree and stamps the commit, so the
figures always name what produced them. **Re-run it rather than quoting the
table** — the ratios track how large this repository's own files are, and they
have moved by a third within a day. It measures input bytes and round trips, not
time; there are no `go test -bench` benchmarks.

## Releasing

Push a strict `vX.Y.Z` tag:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The tag filter on `push` is a glob and cannot express "digits only", so a `check`
job re-matches with a regex and everything downstream gates on it —
`v1.2.3-rc1` builds nothing. Five targets cross-compile (linux and darwin on
amd64 and arm64, windows on amd64) and publish as raw binaries, conventional
archives, and a `SHA256SUMS.txt`.

## Licence

MIT — see `LICENSE`. By contributing you agree your contribution is licensed
under the same terms.
