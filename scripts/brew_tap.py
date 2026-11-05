"""Publish the generated mrw Homebrew formula after a trusted release (ADR-172)."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from urllib.parse import quote

REPO = "atvirokodosprendimai/tool-multipathreadwrite"
TAP = "atvirokodosprendimai/homebrew-tap"
GENERATOR_REF = "a9c2d508f840694cbd601cdbb3b313661fd51bc9"
TARGETS = ("darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64")


def run(argv, *, cwd=None, env=None):
    return subprocess.run(argv, cwd=cwd, env=env, check=True,
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          timeout=90).stdout


def api(path, *, raw=False, env=None):
    args = ["gh", "api", path, "--hostname", "github.com"]
    if raw:
        args += ["-H", "Accept: application/vnd.github.raw+json"]
    result = run(args, env=env)
    return result if raw else json.loads(result)


def trusted(ci, tag, sha):
    return (ci.get("event") == "push" and ci.get("conclusion") == "success"
            and ci.get("path") == ".github/workflows/ci.yml"
            and ci.get("head_repository", {}).get("full_name") == REPO
            and ci.get("head_branch") == tag and ci.get("head_sha") == sha)


def release_tag(run_id, tag):
    if run_id is not None:
        if not re.fullmatch(r"[0-9]+", run_id):
            raise ValueError("run id must be numeric")
        ci = api(f"repos/{REPO}/actions/runs/{run_id}")
        tag = ci.get("head_branch", "")
    if not isinstance(tag, str) or not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+", tag):
        raise ValueError("not a strict vX.Y.Z tag")
    sha = api(f"repos/{REPO}/commits/refs%2Ftags%2F{quote(tag, safe='')}")["sha"]
    if run_id is None:
        # A manual repair must attest the same tag commit as the automatic path.
        runs = api(f"repos/{REPO}/actions/workflows/ci.yml/runs"
                   f"?event=push&status=success&branch={quote(tag)}&per_page=100")
        candidates = runs["workflow_runs"]
    else:
        candidates = [ci]
    if not any(trusted(candidate, tag, sha) for candidate in candidates):
        raise ValueError("no successful same-repository release ci matches the tag commit")
    release = api(f"repos/{REPO}/releases/tags/{tag}")
    if release.get("tag_name") != tag or release.get("draft") or release.get("prerelease"):
        raise ValueError("release is not a published stable tag")
    assets = {asset["name"] for asset in release["assets"] if asset["size"] > 0}
    required = {f"mrw_{target}.tar.gz" for target in TARGETS} | {"SHA256SUMS.txt"}
    if not required <= assets:
        raise ValueError("release is missing required archives or SHA256SUMS.txt")
    require_latest(tag)
    return tag


def require_latest(tag):
    if api(f"repos/{REPO}/releases/latest")["tag_name"] != tag:
        raise ValueError("refusing to replace the tap with an older release")


def validate_formula(formula, manifest, tag):
    text = formula.decode("utf-8")
    if re.findall(r'^\s*version "([^"]+)"$', text, re.M) != [tag[1:]]:
        raise ValueError("formula version differs from release tag")
    sums = {}
    for line in manifest.decode("utf-8").splitlines():
        match = re.fullmatch(r"([0-9a-f]{64})\s+\*?(mrw_(?:darwin|linux)_(?:amd64|arm64)\.tar\.gz)", line)
        if match:
            checksum, name = match.groups()
            if name in sums:
                raise ValueError(f"duplicate manifest checksum: {name}")
            sums[name] = checksum
    if set(sums) != {f"mrw_{target}.tar.gz" for target in TARGETS}:
        raise ValueError("manifest does not contain exactly the four required checksums")

    # Track the renderer's supported Ruby blocks, including closing ends.
    pairs = {}
    blocks = []
    lines = text.splitlines()
    for index, line in enumerate(lines):
        stripped = line.strip()
        os_match = re.fullmatch(r"on_(macos|linux) do", stripped)
        cpu_match = re.fullmatch(r"if Hardware::CPU\.(intel|arm)\?(?: && Hardware::CPU.is_64_bit\?)?", stripped)
        if stripped == "class Mrw < Formula":
            if blocks:
                raise ValueError("nested formula class")
            blocks.append("class")
        elif os_match:
            if blocks != ["class"]:
                raise ValueError("OS block is outside the formula class")
            blocks.append({"macos": "darwin", "linux": "linux"}[os_match[1]])
        elif cpu_match:
            if len(blocks) != 2 or blocks[1] not in {"darwin", "linux"}:
                raise ValueError("CPU block is outside its OS block")
            blocks.append({"intel": "amd64", "arm": "arm64"}[cpu_match[1]])
        elif stripped in {"define_method(:install) do", "test do"}:
            blocks.append("body")
        elif stripped == "end":
            if not blocks:
                raise ValueError("formula has an unmatched block end")
            blocks.pop()
        url = re.fullmatch(r'url "([^"]+)"', stripped)
        if not url:
            continue
        if len(blocks) != 3 or blocks[0] != "class":
            raise ValueError("formula URL is outside its OS/CPU blocks")
        target = f"{blocks[1]}_{blocks[2]}"
        sha = re.fullmatch(r'\s*sha256 "([0-9a-f]{64})"', lines[index + 1] if index + 1 < len(lines) else "")
        name = f"mrw_{target}.tar.gz"
        expected_url = f"https://github.com/{REPO}/releases/download/{tag}/{name}"
        if target not in TARGETS or target in pairs or not sha or url[1] != expected_url or sha[1] != sums[name]:
            raise ValueError("formula URL/checksum differs from its release target")
        pairs[target] = sha[1]
    if blocks or set(pairs) != set(TARGETS):
        raise ValueError("formula does not serve all four release targets in closed blocks")


def publish(*, run_id=None, tag=None):
    tag = release_tag(run_id, tag)
    token = os.environ.get("PUSH_TOKEN")
    if not token:
        raise ValueError("PUSH_TOKEN is required for the tap push")
    push_env = dict(os.environ, GH_TOKEN=token)
    branch = api(f"repos/{TAP}")["default_branch"]
    with tempfile.TemporaryDirectory(prefix="mrw-brew-") as directory:
        root = Path(directory)
        generator = root / "gen-brew-formula.ts"
        generator.write_bytes(api("repos/atvirokodosprendimai/skills/contents/"
                                  f"tools/gen-brew-formula.ts?ref={GENERATOR_REF}", raw=True))
        manifest = root / "SHA256SUMS.txt"
        run(["gh", "release", "download", tag, "--repo", REPO, "--pattern",
             "SHA256SUMS.txt", "--output", str(manifest)])
        render_env = {key: value for key, value in os.environ.items()
                      if key not in {"PUSH_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"}}
        formula = run(["deno", "run", "--allow-net", str(generator), REPO, tag, "mrw",
                       "Multi-path read and write: batch reads and batched, all-or-nothing edits"],
                      env=render_env)
        validate_formula(formula, manifest.read_bytes(), tag)
        tap = root / "tap"
        run(["git", "clone", "--depth", "1", "--branch", branch,
             f"https://github.com/{TAP}.git", str(tap)])
        destination = tap / "Formula" / "mrw.rb"
        destination.parent.mkdir(exist_ok=True)
        destination.write_bytes(formula)
        run(["git", "add", "Formula/mrw.rb"], cwd=tap)
        require_latest(tag)
        if run(["git", "diff", "--cached", "--name-only"], cwd=tap).strip():
            run(["git", "-c", "user.name=mrw-release-bot",
                 "-c", "user.email=actions@users.noreply.github.com",
                 "commit", "-m", f"mrw {tag}: regenerate formula from SHA256SUMS.txt"], cwd=tap)
            # Credentials stay in the environment; neither argv nor .git/config holds them.
            run(["git", "-c", "credential.helper=",
                 "-c", "credential.helper=!gh auth git-credential",
                 "push", "origin", f"HEAD:refs/heads/{branch}"], cwd=tap, env=push_env)
        served = api(f"repos/{TAP}/contents/Formula/mrw.rb?ref={quote(branch, safe='')}",
                     raw=True, env=push_env)
        if served != formula:
            raise ValueError("tap does not serve the exact checked formula")
    print(f"tap serves mrw {tag}")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--run-id")
    source.add_argument("--tag")
    args = parser.parse_args(argv)
    try:
        publish(run_id=args.run_id, tag=args.tag)
    except (ValueError, OSError, subprocess.SubprocessError) as exc:
        print(f"brew-tap: {exc}", file=sys.stderr)
        detail = getattr(exc, "stderr", None)
        if detail:
            detail = detail.decode("utf-8", "replace")
            for name in ("PUSH_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"):
                secret = os.environ.get(name)
                if secret:
                    detail = detail.replace(secret, "[redacted]")
            print(detail[-4096:].rstrip(), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
