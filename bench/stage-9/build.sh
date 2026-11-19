#!/bin/sh
# Builds the two binaries of ADR-206 Decision 4 from one tag: mrw-m as the tag
# builds, and mrw-n with bench/stage-9/no-refusal.patch applied. Usage:
#   bench/stage-9/build.sh TAG OUTDIR
# Prints `m sha256: <hex>` and `n sha256: <hex>`. The tag is exported into a
# temporary directory, so the caller's checkout is not touched; the patched
# source is never compiled by this repository's own build.
set -eu

[ $# -eq 2 ] || { echo "usage: build.sh TAG OUTDIR" >&2; exit 2; }
tag=$1
mkdir -p "$2"
out=$(cd "$2" && pwd)
here=$(cd "$(dirname "$0")" && pwd)
repo=$(git -C "$here" rev-parse --show-toplevel)

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM
git -C "$repo" archive "$tag" | tar -x -C "$tmp"

# The toolchain go.mod names, the one the release is built with.
ver=$(sed -n 's/^go //p' "$tmp/go.mod" | head -n 1)
[ -n "$ver" ] || { echo "no go directive in go.mod of $tag" >&2; exit 2; }

build() {
	(cd "$tmp" && GOTOOLCHAIN="go$ver" CGO_ENABLED=0 go build -trimpath -buildvcs=false \
		-ldflags "-s -w -X main.version=$tag" -o "$out/$1" ./cmd/mrw)
}

sha() { shasum -a 256 "$1" | cut -d ' ' -f 1; }

build mrw-m
(cd "$tmp" && git apply "$here/no-refusal.patch")
build mrw-n

echo "m sha256: $(sha "$out/mrw-m")"
echo "n sha256: $(sha "$out/mrw-n")"
