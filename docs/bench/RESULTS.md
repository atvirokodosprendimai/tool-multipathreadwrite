| stage | model | source | verdict | tasks | attempts | excluded | cost (B−A)/A [95%] | solved B−A [95%] | silent mis-edits A / B | B shell reads | B slipped writes |
|---|---|---|---|---|---|---|---|---|---|---|---|
| stage-3-swe-bench | claude-opus-5-5 | swe-bench | INCONCLUSIVE | 20 | 40 | 0 | -0.053 [-0.229, +0.148] | +0.000 [+0.000, +0.000] | not measured | 47 | 0 |
| stage-3-swe-bench | claude-sonnet-5-5 | swe-bench | INCONCLUSIVE | 20 | 40 | 0 | -0.061 [-0.170, +0.046] | +0.000 [-0.150, +0.150] | not measured | 23 | 0 |
| stage-4b-swe | claude-opus-5-5 | swe-bench | INCONCLUSIVE | 20 | 40 | 0 | -0.209 [-0.323, -0.086] | -0.050 [-0.200, +0.100] | not measured | 59 | 0 |
| stage-4b-swe | claude-opus-5-5 | swe-bench (only the instances whose problem statement names no patched file; descriptive, not the verdict) | (INCONCLUSIVE) | 15 | 30 | 0 | -0.229 [-0.345, -0.091] | -0.067 [-0.267, +0.133] | not measured | 53 | 0 |
| stage-4b-swe | claude-sonnet-5-5 | swe-bench | INCONCLUSIVE | 20 | 40 | 0 | -0.170 [-0.266, -0.076] | -0.150 [-0.300, +0.000] | not measured | 56 | 0 |
| stage-4b-swe | claude-sonnet-5-5 | swe-bench (only the instances whose problem statement names no patched file; descriptive, not the verdict) | (INCONCLUSIVE) | 15 | 30 | 0 | -0.199 [-0.306, -0.092] | -0.133 [-0.333, +0.000] | not measured | 39 | 0 |
| stage1 | claude-sonnet-5-5 | own | INCONCLUSIVE | 30 | 120 | 0 | +0.026 [-0.039, +0.090] | +0.050 [-0.083, +0.167] | 2 / 4 | 42 | 0 |
| stage1 | claude-sonnet-5-5 | own (without the tasks superseded after the stage ran; registered after it ran, descriptive, not the verdict) | (INCONCLUSIVE) | 27 | 108 | 0 | +0.010 [-0.055, +0.068] | +0.130 [+0.056, +0.222] | 0 / 0 | 34 | 0 |
| stage1 | claude-sonnet-5-5 | own (without the tasks whose own check is red at base; descriptive, not the verdict) | (INCONCLUSIVE) | 23 | 92 | 0 | +0.044 [-0.009, +0.103] | +0.043 [-0.130, +0.196] | 2 / 4 | 26 | 0 |
| stage1b | claude-sonnet-5-5 | own | INCONCLUSIVE | 40 | 160 | 0 | -0.004 [-0.050, +0.042] | -0.013 [-0.037, +0.000] | 0 / 0 | 59 | 0 |
| stage1b | claude-sonnet-5-5 | own (only the varied-multi-site tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 10 | 40 | 0 | -0.075 [-0.161, +0.010] | +0.000 [+0.000, +0.000] | 0 / 0 | 17 | 0 |
| stage1b | claude-sonnet-5-5 | own (without the tasks superseded after the stage ran; registered after it ran, descriptive, not the verdict) | (INCONCLUSIVE) | 39 | 156 | 0 | -0.007 [-0.055, +0.040] | +0.000 [+0.000, +0.000] | 0 / 0 | 58 | 0 |
| stage1b | claude-sonnet-5-5 | own (without the tasks whose own check is red at base; descriptive, not the verdict) | (INCONCLUSIVE) | 30 | 120 | 0 | +0.019 [-0.029, +0.068] | -0.017 [-0.050, +0.000] | 0 / 0 | 45 | 0 |
| stage2 | claude-opus-5-5 | own | INCONCLUSIVE | 40 | 160 | 0 | -0.027 [-0.078, +0.021] | -0.025 [-0.075, +0.000] | 0 / 2 | 48 | 0 |
| stage2 | claude-opus-5-5 | own (only the varied-multi-site tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 10 | 40 | 0 | -0.047 [-0.156, +0.062] | +0.000 [+0.000, +0.000] | 0 / 0 | 19 | 0 |
| stage2 | claude-opus-5-5 | own (without the tasks whose own check is red at base; descriptive, not the verdict) | (INCONCLUSIVE) | 30 | 120 | 0 | +0.001 [-0.045, +0.045] | -0.033 [-0.100, +0.000] | 0 / 2 | 20 | 0 |
| stage4 | claude-sonnet-5-5 | own | LOSES | 24 | 97 | 1 | +0.171 [+0.096, +0.247] | -0.042 [-0.104, +0.000] | 0 / 1 | 115 | 0 |
| stage4 | claude-sonnet-5-5 | own (only the check-coupled tasks; descriptive, not the verdict) | (LOSES) | 8 | 33 | 1 | +0.108 [+0.015, +0.214] | -0.125 [-0.312, +0.000] | 0 / 1 | 50 | 0 |
| stage4 | claude-sonnet-5-5 | own (only the single-line tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 6 | 24 | 0 | -0.007 [-0.103, +0.092] | +0.000 [+0.000, +0.000] | 0 / 0 | 1 | 0 |
| stage4 | claude-sonnet-5-5 | own (only the wide-rename tasks; descriptive, not the verdict) | (LOSES) | 10 | 40 | 0 | +0.328 [+0.263, +0.401] | +0.000 [+0.000, +0.000] | 0 / 0 | 64 | 0 |
| stage4 | claude-sonnet-5-5 | own (without the tasks whose own check is red at base; descriptive, not the verdict) | (LOSES) | 23 | 93 | 1 | +0.178 [+0.101, +0.256] | -0.043 [-0.109, +0.000] | 0 / 1 | 115 | 0 |
| stage4b | claude-opus-5-5 | own | INCONCLUSIVE | 24 | 96 | 0 | -0.016 [-0.046, +0.014] | -0.021 [-0.083, +0.042] | 2 / 2 | 37 | 0 |
| stage4b | claude-opus-5-5 | own (only the check-coupled tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 8 | 32 | 0 | -0.021 [-0.072, +0.028] | -0.062 [-0.188, +0.000] | 0 / 0 | 6 | 0 |
| stage4b | claude-opus-5-5 | own (only the single-line tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 6 | 24 | 0 | -0.044 [-0.097, +0.014] | +0.000 [+0.000, +0.000] | 0 / 0 | 0 | 0 |
| stage4b | claude-opus-5-5 | own (only the wide-rename tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 10 | 40 | 0 | +0.004 [-0.039, +0.054] | +0.000 [-0.150, +0.150] | 2 / 2 | 31 | 0 |
| stage4b | claude-opus-5-5 | own (without the tasks whose own check is red at base; descriptive, not the verdict) | (INCONCLUSIVE) | 23 | 92 | 0 | -0.012 [-0.042, +0.019] | -0.022 [-0.087, +0.043] | 2 / 2 | 37 | 0 |
| stage4b | claude-sonnet-5-5 | own | INCONCLUSIVE | 24 | 96 | 0 | -0.021 [-0.062, +0.018] | -0.083 [-0.208, +0.021] | 2 / 3 | 33 | 0 |
| stage4b | claude-sonnet-5-5 | own (only the check-coupled tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 8 | 32 | 0 | -0.017 [-0.108, +0.058] | -0.125 [-0.375, +0.125] | 2 / 1 | 16 | 0 |
| stage4b | claude-sonnet-5-5 | own (only the single-line tasks; descriptive, not the verdict) | (PAYS) | 6 | 24 | 0 | -0.103 [-0.170, -0.033] | +0.000 [+0.000, +0.000] | 0 / 0 | 0 | 0 |
| stage4b | claude-sonnet-5-5 | own (only the wide-rename tasks; descriptive, not the verdict) | (INCONCLUSIVE) | 10 | 40 | 0 | +0.025 [-0.001, +0.057] | -0.100 [-0.300, +0.000] | 0 / 2 | 17 | 0 |
| stage4b | claude-sonnet-5-5 | own (without the tasks whose own check is red at base; descriptive, not the verdict) | (INCONCLUSIVE) | 23 | 92 | 0 | -0.012 [-0.052, +0.025] | -0.087 [-0.217, +0.022] | 2 / 3 | 33 | 0 |

| stage | model | source | verdict | look | tasks | attempts | excluded | cost (X−A)/A [99%] | solved X−A [99%, Tango] | B shell reads |
|---|---|---|---|---|---|---|---|---|---|---|
| stage5 | claude-sonnet-5-5 | swe-bench, B against A | INCONCLUSIVE | open when the stage ended at look 5 | 191 | 383 | 1 | -0.070 [-0.121, -0.018] | -0.026 [-0.087, +0.028] | 392 |
| stage5 | claude-sonnet-5-5 | swe-bench, P against A | LOSES | closed at look 1 | 40 | 82 | 2 | +0.701 [+0.477, +0.926] | -0.025 [-0.182, +0.121] | 0 |
| stage5 | claude-opus-5-5 | swe-bench, B against A | INCONCLUSIVE | open when the stage ended at look 5 | 190 | 382 | 1 | -0.194 [-0.255, -0.127] | -0.005 [-0.067, +0.055] | 394 |
| stage5 | claude-opus-5-5 | swe-bench, P against A | LOSES | closed at look 1 | 39 | 83 | 4 | +0.486 [+0.312, +0.674] | +0.000 [-0.166, +0.166] | 0 |
| stage5 | claude-sonnet-5-5 | swe-bench, B against A (only the instances with 5 or more hunks; descriptive, not the verdict) | (INCONCLUSIVE) | look 5 | 31 | 63 | 1 | -0.090 [-0.225, +0.044] | +0.000 [-0.201, +0.201] | 103 |
| stage5 | claude-sonnet-5-5 | swe-bench, P against A (only the instances with 5 or more hunks; descriptive, not the verdict) | (LOSES) | look 1 | 7 | 14 | 0 | +0.553 [+0.079, +1.118] | +0.000 [-0.487, +0.487] | 0 |
| stage5 | claude-opus-5-5 | swe-bench, B against A (only the instances with 5 or more hunks; descriptive, not the verdict) | (INCONCLUSIVE) | look 5 | 31 | 62 | 0 | -0.226 [-0.339, -0.112] | -0.032 [-0.247, +0.177] | 85 |
| stage5 | claude-opus-5-5 | swe-bench, P against A (only the instances with 5 or more hunks; descriptive, not the verdict) | (LOSES) | look 1 | 7 | 15 | 1 | +0.377 [+0.226, +0.640] | +0.000 [-0.487, +0.487] | 0 |
