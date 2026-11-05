# A valid sibling expression triggers the closer hint

**Status: open.** Reproduced on 2026-10-30 with installed mrw v1.79.0.
The fixture is synthetic; private source identities are kept outside the repository.

Replacing one RSpec expectation with the same expectation split over several lines correctly
changes the file. The hint nevertheless matches the closing `)` of a separate expectation,
three non-blank lines later. This is the sibling-closer limitation in the
[shape-hint measurements](README.md#what-this-does-not-measure), not edit damage.

Run the Python block below from any directory with `mrw` on PATH. Set `MRW` to an executable
path to check a different binary. It uses disposable files and a private state directory,
bounds each CLI call at five seconds, and runs no Ruby, RSpec, project suite or history replay.
Every fixture write goes through mrw.

This is a standalone regression target, outside the normal test suite. It requires no hint
for the valid Ruby edit and a hint for a genuine duplicate Markdown fence. Suppressing every
hint fails the second case. Exit 0 means both expectations hold; exit 1 means a hint expectation
failed; exit 2 means the fixture could not be set up or its resulting content was wrong.
On v1.79.0 the Ruby case fails, the duplicate-fence control passes, and the block exits 1.

````python
import os
from pathlib import Path
import subprocess
import tempfile

binary = os.environ.get("MRW", "mrw")
source = (
    'RSpec.describe X do\n'
    '  it "a" do\n'
    '    expect(A).to(receive(:call).and_call_original)\n'
    '\n'
    '    expect { run }.to(\n'
    '      change { y }\n'
    '    )\n'
    '  end\n'
    'end\n'
)
expanded = (
    '    expect(A).to(\n'
    '      receive(:call).\n'
    '        and_call_original\n'
    '    )\n'
)
cases = [
    (
        "s_spec.rb",
        source,
        "@@ s_spec.rb 3 replace\n" + expanded,
        source.replace(
            "    expect(A).to(receive(:call).and_call_original)\n",
            expanded,
            1,
        ),
        False,
    ),
    (
        "orphan.md",
        "intro\n```\nbody\n```\nend\n",
        "@@ orphan.md 3 replace\nchanged\n```\n",
        "intro\n```\nchanged\n```\n```\nend\n",
        True,
    ),
]

with tempfile.TemporaryDirectory(prefix="mrw-sibling-closer-") as directory:
    root = Path(directory)
    environment = dict(os.environ, XDG_STATE_HOME=str(root / "state"))

    def call(*args, data=None):
        try:
            result = subprocess.run(
                [binary, "--root", str(root), *args],
                input=data,
                capture_output=True,
                text=True,
                env=environment,
                timeout=5,
                check=False,
            )
        except (OSError, subprocess.TimeoutExpired) as error:
            print(f"Fixture error: {error}")
            raise SystemExit(2) from error
        if result.returncode != 0:
            print(result.stdout + result.stderr)
            raise SystemExit(2)
        return result

    print(call("version").stdout.strip())
    failures = 0
    for name, original, plan, expected, want_hint in cases:
        call("write", "--no-check", "--create", name, data=original)
        call("read", name)
        result = call("write", "--no-check", "-", data=plan)
        if (root / name).read_text(encoding="utf-8") != expected:
            print(f"Fixture error: {name} content differs from the expected edit")
            raise SystemExit(2)
        output = result.stdout + result.stderr
        got_hint = "closer line " in output
        passed = got_hint == want_hint
        print(f"{'PASS' if passed else 'FAIL'} {name}: closer hint={got_hint}, expected={want_hint}")
        if not passed:
            print(output)
            failures += 1
    raise SystemExit(1 if failures else 0)
````

The Ruby write on v1.79.0 reports:

```text
closer line 10 repeats the body's last line: ) (the closer the file already had? read past the body)
```

The changed Ruby file is still the expected file: the new expectation ends at line 6,
and the second expectation's own closing parenthesis is at line 10. A future fix must keep
the duplicate-fence control and ADR-119's true-positive fixtures effective. The
[backlog entry](../../adr/BACKLOG.md#from-adr-119-sibling-closer-false-positive)
owns that follow-up; these fixtures change no heuristic or measurement threshold.
