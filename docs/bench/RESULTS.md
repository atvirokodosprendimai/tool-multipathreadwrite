| stage | model | source | verdict | tasks | attempts | excluded | cost (B−A)/A [95%] | solved B−A [95%] | silent mis-edits A / B | B shell reads | B slipped writes |
|---|---|---|---|---|---|---|---|---|---|---|---|
| stage-3-swe-bench | claude-opus-5-5 | swe-bench | INCONCLUSIVE | 20 | 40 | 0 | -0.053 [-0.229, +0.148] | +0.000 [+0.000, +0.000] | not measured | 47 | 0 |
| stage-3-swe-bench | claude-sonnet-5-5 | swe-bench | INCONCLUSIVE | 20 | 40 | 0 | -0.061 [-0.170, +0.046] | +0.000 [-0.150, +0.150] | not measured | 23 | 0 |
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
