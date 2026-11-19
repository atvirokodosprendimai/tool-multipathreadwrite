# Stage 9 builds (ADR-206 Decision 4, T2)

M and N are two builds of one tag. They differ in one refusal: N applies a number-only hunk that ADR-203 refuses, and logs it to `mrw/stage9-exposed.jsonl` under the state home. Neither is installed or released; the patched source is never compiled by this repository's build.

- Tag: `v1.93.0` (the newest `v*` tag reachable from `main` when T2 was written; `TestArmNDiffersFromMInOneRefusal` builds this tag, not a newer one)
- Toolchain: `go1.27.4`, the version `go.mod` names (`GOTOOLCHAIN=go1.27.4`), `CGO_ENABLED=0`
- Flags, both builds: `go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=v1.93.0" ./cmd/mrw`, run with `GOFLAGS= GOWORK=off GOPROXY=off`: inherited flags cannot change the binaries, and a missing toolchain or module fails the build instead of being fetched
- N is the tag with `bench/stage-9/no-refusal.patch` applied (`git apply`, in a temporary export that is its own repository) before its build
- Platform: `darwin/arm64`. Built with `bench/stage-9/build.sh v1.93.0 OUTDIR`, twice, with the same two hashes. Hashes are per platform: the test compares them only on this one; rebuild on the machine that runs the stage and record that.

```
m sha256: a51d6938442a0d8d851e7ad22d167422be2520cc2dc8f1119ff927111806c686
n sha256: 83e8d91d7304db696b6852af45e3597a0f17899b84bf85ba3f3a329b0c00a6c5
```
