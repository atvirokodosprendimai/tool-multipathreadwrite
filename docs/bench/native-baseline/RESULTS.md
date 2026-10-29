# Native performance baseline — results, 2026-10-29

This is the first run of [PREREGISTRATION.md](PREREGISTRATION.md), agreed by Zy on 2026-10-29 and merged in #558 before any measurement. It runs on macOS arm64 only.

It says nothing about agent task time, tokens or cost; that is item 4.

## What ran

| | |
|---|---|
| A | release **v1.78.0 (06075f7)** `mrw-darwin-arm64`, checked against the release's `SHA256SUMS.txt`: `cf4050beacb4390b539fbeb89087875b8468e2a3d97cb2aca20ee10116e0bf38` |
| B | a source build of `fc70c69c` with the release flags (`CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=v1.78.0-dev+fc70c69c"`): `d5c89c7ce69f93a5a5c8e2d732b17e2c0855472d008795a85f8c1fb68ae61733` |
| driver | `cmd/nativebench` at `e36be25f`; `main.go` sha256 `2d61ade7bec70e8e0ec178022ff18719885d56cea481c61673f7dfbfbfe149e1`. Fixtures come from seed 20261029 |
| R3 "repo" tree | `git archive v1.78.0`, 4,554 files, 32 MB |
| machine | Mac17,2, Apple M5, 10 cores, 32 GB, on AC power; macOS 27.0.1 (26A434); go1.27.3 |
| when | 2026-10-29 09:58:42Z – 10:02:45Z (4 min 3 s of the 15-minute budget) |
| load | every cell started below the 2.0 gate: 1-minute load 1.80–1.97 (`load_before` in each sample). The load was 1.82 at the end |
| samples | 1,080: 18 cells × 2 binaries × 30 timed runs, after 5 discarded warm-ups. All are kept in [raw/samples.jsonl](raw/samples.jsonl) |

The load gate first kept the run waiting from about 09:00Z to 09:58Z, at a load average of 5–10. The cause was a Cursor renderer process at 300–500% CPU, plus another session's Go tests. The wait in front of the driver was bounded at 2 hours; the gate itself was not changed.

## Timings (ms; RSS from rusage)

| workload | size | binary | n | failed | p25 ms | p50 ms | p75 ms | p90 ms | max ms | cpu p50 ms | max rss MB | stdout B |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| R1 | 100 | A | 30 | 0 | 9.75 | 9.96 | 10.03 | 10.06 | 10.97 | 4.35 | 14.3 | 1266 |
| R1 | 100 | B | 30 | 0 | 9.04 | 9.95 | 10.04 | 10.13 | 10.21 | 4.33 | 13.8 | 1266 |
| R1 | 10000 | A | 30 | 0 | 9.92 | 10.01 | 10.09 | 10.14 | 13.75 | 4.60 | 15.2 | 1275 |
| R1 | 10000 | B | 30 | 0 | 9.88 | 9.93 | 10.03 | 10.07 | 13.44 | 4.57 | 14.7 | 1275 |
| R1 | 1000000 | A | 30 | 0 | 32.03 | 32.90 | 33.07 | 33.23 | 34.07 | 29.37 | 106.8 | 1330 |
| R1 | 1000000 | B | 30 | 0 | 32.77 | 32.87 | 32.93 | 33.05 | 33.23 | 29.66 | 107.1 | 1330 |
| R2 | 1 | A | 30 | 0 | 9.06 | 9.71 | 10.01 | 10.05 | 11.88 | 4.33 | 14.6 | 1278 |
| R2 | 1 | B | 30 | 0 | 9.02 | 9.84 | 9.98 | 10.37 | 15.04 | 4.36 | 14.1 | 1278 |
| R2 | 10 | A | 30 | 0 | 9.95 | 10.09 | 10.97 | 11.03 | 11.29 | 5.22 | 15.5 | 12782 |
| R2 | 10 | B | 30 | 0 | 9.94 | 10.08 | 10.90 | 11.06 | 11.83 | 5.23 | 15.1 | 12782 |
| R2 | 100 | A | 30 | 0 | 17.91 | 18.01 | 18.14 | 18.97 | 20.26 | 13.31 | 20.5 | 127888 |
| R2 | 100 | B | 30 | 0 | 17.85 | 17.96 | 18.05 | 18.18 | 20.97 | 13.32 | 20.0 | 127888 |
| R3 | repo | A | 30 | 0 | 99.89 | 100.62 | 101.97 | 104.21 | 153.18 | 371.14 | 26.2 | 638 |
| R3 | repo | B | 30 | 0 | 99.88 | 100.63 | 102.14 | 103.65 | 150.25 | 372.69 | 25.8 | 638 |
| R3 | deep3000 | A | 30 | 30 | 49.71 | 50.00 | 50.23 | 51.15 | 52.01 | 280.28 | 22.0 | 0 |
| R3 | deep3000 | B | 30 | 30 | 49.98 | 50.18 | 50.38 | 50.50 | 55.66 | 283.07 | 21.5 | 0 |
| W1 | 1h/1f | A | 30 | 0 | 11.70 | 11.86 | 11.99 | 12.13 | 14.79 | 6.02 | 15.0 | 133 |
| W1 | 1h/1f | B | 30 | 0 | 11.64 | 11.82 | 11.86 | 11.90 | 13.77 | 6.04 | 14.6 | 133 |
| W1 | 50h/10f | A | 30 | 0 | 14.93 | 15.93 | 16.45 | 16.89 | 19.05 | 9.80 | 18.9 | 2139 |
| W1 | 50h/10f | B | 30 | 0 | 15.02 | 15.86 | 16.03 | 16.69 | 17.70 | 9.64 | 19.1 | 2139 |
| W1 | 500h/50f | A | 30 | 0 | 31.47 | 32.23 | 33.21 | 34.24 | 36.34 | 30.70 | 23.3 | 18710 |
| W1 | 500h/50f | B | 30 | 0 | 31.35 | 33.09 | 33.94 | 35.13 | 41.14 | 30.71 | 23.1 | 18710 |
| W2 | 10 | A | 30 | 0 | 15.91 | 16.03 | 16.90 | 17.08 | 21.26 | 7.54 | 16.6 | 387 |
| W2 | 10 | B | 30 | 0 | 15.88 | 16.70 | 16.98 | 17.92 | 18.37 | 7.48 | 16.0 | 387 |
| W2 | 1000 | A | 30 | 0 | 22.88 | 23.11 | 23.72 | 24.19 | 26.09 | 15.88 | 23.0 | 30993 |
| W2 | 1000 | B | 30 | 0 | 22.72 | 22.85 | 23.33 | 24.53 | 28.62 | 15.79 | 22.5 | 30993 |
| L1 | 0 | A | 30 | 0 | 9.91 | 10.00 | 10.07 | 10.13 | 10.34 | 4.61 | 15.2 | 1276 |
| L1 | 0 | B | 30 | 0 | 9.91 | 9.96 | 10.04 | 10.16 | 11.82 | 4.63 | 14.7 | 1276 |
| L1 | 1000 | A | 30 | 0 | 9.91 | 10.00 | 10.09 | 10.93 | 11.22 | 5.55 | 18.1 | 1273 |
| L1 | 1000 | B | 30 | 0 | 9.97 | 10.04 | 10.11 | 10.74 | 11.76 | 5.51 | 17.6 | 1273 |
| L1 | 10000 | A | 30 | 0 | 13.92 | 14.11 | 14.89 | 15.04 | 18.41 | 11.03 | 24.6 | 1277 |
| L1 | 10000 | B | 30 | 0 | 13.92 | 14.02 | 14.24 | 14.92 | 16.01 | 10.80 | 24.4 | 1277 |
| C1 | first | A | 30 | 0 | 9.66 | 10.43 | 10.56 | 10.63 | 11.63 | 4.82 | 15.1 | 1278 |
| C1 | first | B | 30 | 0 | 9.68 | 9.83 | 10.63 | 10.77 | 13.30 | 4.76 | 14.6 | 1278 |
| C1 | warm | A | 30 | 0 | 9.95 | 10.04 | 10.12 | 10.15 | 11.16 | 4.62 | 15.4 | 1278 |
| C1 | warm | B | 30 | 0 | 9.88 | 9.94 | 10.01 | 10.05 | 10.17 | 4.60 | 14.7 | 1278 |

R3 "deep3000" exits 1 in every run, by design. The tree is BenchmarkWalkDeep131's shape and matches nothing, so `mrw read --grep` reports `no match`. As registered, each non-zero exit is counted in `failed` and not dropped.

A and B agree within noise in every cell: the commits after v1.78.0 changed no served code path.

## Against the registered bar

**Candidate bottlenecks.** A candidate is a cell whose p50 exceeds 100 ms, or whose time grows faster than linearly between adjacent sizes (time ratio over size ratio above 1.5).

- **R3 "repo" is the one candidate.** Its p50 is 100.62 ms (A) and 100.63 ms (B), just over the line.
- **Nothing grows super-linearly.** The largest time ratio over size ratio is W1 (1,1)→(50,10): 1.34× the time for 50× the hunks. Every other adjacent pair is far below 1.5:
  - R1: 10,000→1,000,000 lines, 3.3× the time for 100× the size;
  - L1: 1,000→10,000 entries, 1.41× for 10×;
  - W2: 10→1,000 sites, 1.44× for 100×.
- **Everything else** has a p50 at or below 33 ms.

**The candidate, profiled before any code is changed** ([raw/r3.cpu.pprof](raw/r3.cpu.pprof), [raw/r3.mem.pprof](raw/r3.mem.pprof)). The profile is of `BenchmarkR3Repo` (`cmd/nativebench/bench_test.go`), which runs the same walk and `--count` read in process. It was taken at a 1-minute load of 4.0, above the gate, so its timings are not results. Its shape is the finding:

- **75% of CPU is in syscalls.** `regular.Open` (`os.OpenFile`, from `readBudget`) takes 48% of all samples: one open per walked file, 4,554 of them.
- `Lstat` takes 9%, and `rooted.(*Resolver).Resolve` 9% (overlapping).
- `io.ReadAll` takes 5.5%.
- The regexp is under 1%.
- The walk is parallel: CPU p50 is 371 ms against a wall p50 of 100 ms.

The cost is the filesystem's per-file open, not mrw's own code. No code change is made here. Any optimisation goes back to Zy first, and is kept only if the registered bar holds: a p50 gain of 20% or more with non-overlapping p25–p75 intervals, every gate passing, and no guard, licence or bound changed.

## Allocations (`go test -bench -benchmem -count 5`)

These were run straight after the timed cells, at a 1-minute load of 2.6–4.7, above the gate. The pre-registration gates workloads, not these benchmarks, and B/op and allocs/op do not depend on load. Their ns/op are reported, not judged.

```
BenchmarkWalkDeep131-10    	      26	  50658521 ns/op	 9809263 B/op	   78721 allocs/op
BenchmarkWalkDeep131-10    	      16	  64113523 ns/op	 9814257 B/op	   78715 allocs/op
BenchmarkWalkDeep131-10    	      26	  42107099 ns/op	 9710311 B/op	   78703 allocs/op
BenchmarkWalkDeep131-10    	      26	  42511428 ns/op	 9685917 B/op	   78702 allocs/op
BenchmarkWalkDeep131-10    	      26	  42787580 ns/op	 9677348 B/op	   78701 allocs/op
BenchmarkResolveADeepFile-10    	   33744	     34931 ns/op	   16784 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   34210	     35664 ns/op	   16912 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   34263	     35334 ns/op	   16784 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   33662	     35081 ns/op	   16912 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   33836	     35438 ns/op	   16784 B/op	     146 allocs/op
BenchmarkR1-10    	    4671	    249785 ns/op	  889576 B/op	     239 allocs/op
BenchmarkR1-10    	    4869	    509785 ns/op	  889602 B/op	     239 allocs/op
BenchmarkR1-10    	    4708	    263013 ns/op	  889588 B/op	     239 allocs/op
BenchmarkR1-10    	    4605	    260289 ns/op	  889593 B/op	     239 allocs/op
BenchmarkR1-10    	    4740	    256147 ns/op	  889589 B/op	     239 allocs/op
BenchmarkW1-10    	     361	   3372565 ns/op	 4053095 B/op	    6462 allocs/op
BenchmarkW1-10    	     356	   3446429 ns/op	 4059367 B/op	    6464 allocs/op
BenchmarkW1-10    	     345	   3448778 ns/op	 4058817 B/op	    6467 allocs/op
BenchmarkW1-10    	     334	   3305389 ns/op	 4055056 B/op	    6470 allocs/op
BenchmarkW1-10    	     340	   3390711 ns/op	 4060008 B/op	    6469 allocs/op
```

## Where this run departs from the text

- **Peak RSS** comes from `ProcessState.SysUsage()`'s `Maxrss`, the kernel rusage figure `/usr/bin/time -l` prints, not from running `/usr/bin/time` itself.
- **The R3 profile and the allocation benchmarks** ran after the timed cells, above the load gate (stated above). No timing from them is used for a verdict.
- **Two one-run trials of the driver**, made before this run to check that it worked, were discarded. The first exposed a driver bug: a prepare-step read sent to the null device licenses nothing (ADR-133), so W1's writes were refused. It was fixed before the commit that ran.
