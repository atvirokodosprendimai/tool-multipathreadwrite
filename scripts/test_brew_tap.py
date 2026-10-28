"""Exercise the release publisher against a real local Git remote."""

import contextlib
import copy
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import brew_tap as brew

ROOT = Path(__file__).resolve().parents[1]
TAG = "v1.77.1"
SHA = "a" * 40
SUMS = {f"mrw_{target}.tar.gz": str(i) * 64
        for i, target in enumerate(brew.TARGETS, 1)}


def formula():
    lines = ['class Mrw < Formula', '  version "1.77.1"']
    for system, os_name in (("macos", "darwin"), ("linux", "linux")):
        lines.append(f"  on_{system} do")
        for cpu, arch in (("intel", "amd64"), ("arm", "arm64")):
            name = f"mrw_{os_name}_{arch}.tar.gz"
            lines += [f"    if Hardware::CPU.{cpu}?",
                      f'      url "https://github.com/{brew.REPO}/releases/download/{TAG}/{name}"',
                      f'      sha256 "{SUMS[name]}"', "    end"]
        lines.append("  end")
    return ("\n".join(lines) + "\nend\n").encode()


class BrewTapTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.remote = self.root / "remote.git"
        self.git_env = dict(os.environ, GIT_AUTHOR_NAME="fixture",
                            GIT_AUTHOR_EMAIL="fixture@example.test",
                            GIT_COMMITTER_NAME="fixture",
                            GIT_COMMITTER_EMAIL="fixture@example.test",
                            GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull)
        self.real_run = brew.run
        self.real_run(["git", "init", "--bare", "--initial-branch=main", str(self.remote)])
        seed = self.root / "seed"
        self.real_run(["git", "clone", str(self.remote), str(seed)])
        (seed / "README").write_text("tap fixture\n", encoding="utf-8")
        self.real_run(["git", "add", "."], cwd=seed)
        self.real_run(["git", "commit", "-m", "seed"], cwd=seed, env=self.git_env)
        self.real_run(["git", "push", "origin", "HEAD:main"], cwd=seed)
        self.before = self.head()
        self.ci = {"head_branch": TAG, "head_sha": SHA, "event": "push",
                   "path": ".github/workflows/ci.yml", "conclusion": "success",
                   "head_repository": {"full_name": brew.REPO}}
        self.release = {"tag_name": TAG, "draft": False, "prerelease": False,
                        "assets": [{"name": n, "size": 1}
                                   for n in [*SUMS, "SHA256SUMS.txt"]]}
        self.latest = copy.deepcopy(self.release)
        self.manifest = "".join(f"{sha}  {name}\n" for name, sha in SUMS.items()).encode()
        self.rendered = formula()
        self.bad_readback = False
        self.api_error = False
        self.calls = []
        self.addCleanup(patch.stopall)
        patch.dict(os.environ, {"PUSH_TOKEN": "fixture-secret"}).start()
        patch.object(brew, "run", side_effect=self.fake_run).start()

    def head(self):
        return self.real_run(["git", "--git-dir", str(self.remote), "rev-parse", "HEAD"])

    def fake_run(self, argv, *, cwd=None, env=None):
        argv = [str(a) for a in argv]
        self.calls.append(argv)
        if argv[:2] == ["gh", "api"]:
            if self.api_error:
                raise subprocess.CalledProcessError(1, argv)
            endpoint = argv[2]
            if "/actions/runs/" in endpoint:
                data = self.ci
            elif "/actions/workflows/ci.yml/runs?" in endpoint:
                data = {"workflow_runs": [self.ci]}
            elif "/commits/refs%2Ftags%2F" in endpoint:
                data = {"sha": SHA}
            elif "/releases/tags/" in endpoint:
                data = self.release
            elif endpoint.endswith("/releases/latest"):
                data = self.latest
            elif "/contents/tools/gen-brew-formula.ts?" in endpoint:
                self.assertIn(brew.GENERATOR_REF, endpoint)
                return b"// pinned renderer fixture\n"
            elif "/contents/Formula/mrw.rb?" in endpoint:
                if self.bad_readback:
                    return b"stale formula\n"
                return self.real_run(["git", "--git-dir", str(self.remote),
                                      "show", "main:Formula/mrw.rb"])
            elif endpoint == f"repos/{brew.TAP}":
                data = {"default_branch": "main"}
            else:
                self.fail(f"unexpected GitHub endpoint: {endpoint}")
            return json.dumps(data).encode()
        if argv[:3] == ["gh", "release", "download"]:
            Path(argv[argv.index("--output") + 1]).write_bytes(self.manifest)
            return b""
        if argv[:2] == ["deno", "run"]:
            self.assertIn("--allow-net", argv)
            self.assertNotIn("-A", argv)
            return self.rendered
        if "clone" in argv and f"https://github.com/{brew.TAP}.git" in argv:
            argv = [str(self.remote) if a == f"https://github.com/{brew.TAP}.git" else a
                    for a in argv]
        return self.real_run(argv, cwd=cwd, env=env)

    def invoke(self, *args):
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            return brew.main(list(args))

    def test_publish_and_retry(self):
        self.assertEqual(self.invoke("--run-id", "123"), 0)
        self.assertNotEqual(self.head(), self.before)
        served = self.real_run(["git", "--git-dir", str(self.remote),
                               "show", "main:Formula/mrw.rb"])
        self.assertEqual(served, self.rendered)
        first = self.head()
        self.assertEqual(self.invoke("--run-id", "123"), 0)
        self.assertEqual(self.head(), first)
        self.assertEqual(sum("push" in call for call in self.calls), 1)
        self.assertNotIn("fixture-secret", json.dumps(self.calls))

    def test_manual_replay_requires_release_ci(self):
        self.assertEqual(self.invoke("--tag", TAG), 0)
        self.ci["head_sha"] = "b" * 40
        self.assertEqual(self.invoke("--tag", TAG), 1)

    def test_untrusted_runs_refused(self):
        cases = {"event": "pull_request", "conclusion": "failure",
                 "path": ".github/workflows/other.yml", "head_sha": "b" * 40,
                 "head_repository": {"full_name": "fork/repo"},
                 "head_branch": "v1.77.1-rc1"}
        original = copy.deepcopy(self.ci)
        for key, value in cases.items():
            with self.subTest(key=key):
                self.ci = {**original, key: value}
                self.assertEqual(self.invoke("--run-id", "123"), 1)
                self.assertEqual(self.head(), self.before)
        self.assertEqual(self.invoke("--run-id", "../123"), 1)
        self.assertEqual(self.invoke("--tag", "v1.77.1;echo injected"), 1)

    def test_incomplete_or_older_release_refused(self):
        original = copy.deepcopy(self.release)
        for key, value in (("draft", True), ("prerelease", True),
                           ("tag_name", "v1.77.0"), ("assets", [])):
            with self.subTest(key=key):
                self.release = {**original, key: value}
                self.assertEqual(self.invoke("--run-id", "123"), 1)
                self.assertEqual(self.head(), self.before)
        self.release = original
        self.latest["tag_name"] = "v1.78.0"
        self.assertEqual(self.invoke("--run-id", "123"), 1)
        self.assertEqual(self.head(), self.before)

    def test_manifest_and_formula_pairs_refused(self):
        original = self.rendered
        for damaged in (original.replace(b"1" * 64, b"f" * 64),
                        original.replace(b"darwin_amd64.tar.gz", b"linux_amd64.tar.gz"),
                        original.replace(b"on_macos", b"on_windows"),
                        original.replace(b'1.77.1"', b'1.77.0"')):
            with self.subTest(damaged=damaged):
                self.rendered = damaged
                self.assertEqual(self.invoke("--run-id", "123"), 1)
                self.assertEqual(self.head(), self.before)
        self.rendered = original
        valid_manifest = self.manifest
        for damaged in (b"", valid_manifest + valid_manifest.splitlines(keepends=True)[0]):
            with self.subTest(manifest=damaged):
                self.manifest = damaged
                self.assertEqual(self.invoke("--run-id", "123"), 1)
                self.assertEqual(self.head(), self.before)

    def test_push_failure_not_success(self):
        hook = self.remote / "hooks" / "pre-receive"
        hook.write_text("#!/bin/sh\nexit 1\n", encoding="utf-8")
        hook.chmod(0o755)
        self.assertEqual(self.invoke("--run-id", "123"), 1)
        self.assertEqual(self.head(), self.before)

    def test_readback_must_match(self):
        self.bad_readback = True
        self.assertEqual(self.invoke("--run-id", "123"), 1)
        self.assertNotEqual(self.head(), self.before)

    def test_api_failure_refused(self):
        self.api_error = True
        self.assertEqual(self.invoke("--run-id", "123"), 1)
        self.assertEqual(self.head(), self.before)

    def test_workflow_reaches_publisher(self):
        workflow = (ROOT / ".github/workflows/brew-tap.yml").read_text(encoding="utf-8")
        trigger = workflow.split("permissions:", 1)[0]
        self.assertIn("workflow_run:", trigger)
        self.assertNotIn("branches:", trigger)
        self.assertIn("workflow_dispatch:", trigger)
        self.assertIn("github.event.workflow_run.event == 'push'", workflow)
        self.assertIn("github.event.workflow_run.head_repository.full_name == github.repository", workflow)
        self.assertIn("ref: ${{ github.event.repository.default_branch }}", workflow)
        self.assertIn("python3 scripts/brew_tap.py", workflow)
        self.assertIn('args=(--tag "$TAG")', workflow)
        self.assertIn('args=(--run-id "$RUN_ID")', workflow)

    def test_ci_and_harness_select_tests(self):
        ci = (ROOT / ".github/workflows/ci.yml").read_text(encoding="utf-8")
        self.assertIn("python3 -m unittest discover -s scripts -p test_brew_tap.py -v", ci)
        harness = json.loads((ROOT / ".quality-harness.json").read_text(encoding="utf-8"))
        self.assertEqual(harness["steps"]["brew-tap"],
                         "python3 -m unittest discover -s scripts -p test_brew_tap.py -v")


    def test_target_outside_os_scope_is_refused(self):
        self.rendered = self.rendered.replace(
            b"    end\n    if Hardware::CPU.arm?\n",
            b"    end\n  end\n  if Hardware::CPU.arm?\n", 1).replace(
                b"    end\n  end\n  on_linux", b"    end\n  on_linux", 1)
        self.assertEqual(self.invoke("--run-id", "123"), 1)
        self.assertEqual(self.head(), self.before)

    def test_subprocess_stderr_is_bounded_and_redacted(self):
        hook = self.remote / "hooks" / "pre-receive"
        hook.write_text("#!/bin/sh\necho 'tap fixture rejected' >&2\nexit 1\n", encoding="utf-8")
        hook.chmod(0o755)
        output = io.StringIO()
        with contextlib.redirect_stderr(output):
            self.assertEqual(brew.main(["--run-id", "123"]), 1)
        self.assertIn("tap fixture rejected", output.getvalue())
        error = subprocess.CalledProcessError(
            1, ["git", "push"], stderr=b"x" * 5000 + b" permission denied fixture-secret")
        output = io.StringIO()
        with patch.object(brew, "publish", side_effect=error), contextlib.redirect_stderr(output):
            self.assertEqual(brew.main(["--run-id", "123"]), 1)
        self.assertIn("permission denied", output.getvalue())
        self.assertNotIn("fixture-secret", output.getvalue())
        self.assertLess(len(output.getvalue()), 4500)


if __name__ == "__main__":
    unittest.main()
