# Stage 4's pilot (ADR-160 T3)

Sonnet 5.5, stage `pilot`, never counted. Every receipt is kept as it ran; a later round goes into a new directory.

| Round | Built from | Directories | What it found |
|-------|-----------|-------------|---------------|
| 1 | `19a38aa7` | `calibration/`, `harness/`, `swe/`, `canary.json` | A: wide-rename 7/8, so its floor rose to 25 occurrences across 12 files; check-coupled 3/8, which measured `exact` refusing valid answers. B's hook turned back commands that wrote nothing. Both fixed in T6 (#434). |
| 2 | `883843c9` | `calibration-2/`, `harness-2/`, `swe-2/`, `canary-2.json` | A: wide-rename 6/8 at the raised floor (runs as drawn); check-coupled 8/8 graded by its tests, so its floor rose to a literal two or more test files hold. B: 4/4. |
| 3 | `6509c39d` | `calibration-3/`, `harness-3/`, `canary-3.json` | A: check-coupled 7/8 at the raised floor (runs as drawn, reported at ceiling). B: 2/2; one attempt set aside as setup (two mrw roots' ledgers) and rerun. |

The draws and the floors are in `docs/bench/stage-4-draw/`; T3's sign-offs, with the watch items for T4, are in its task file.
