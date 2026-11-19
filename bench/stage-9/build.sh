#!/bin/sh
# Builds the two binaries of ADR-206 Decision 4 from one tag: mrw-m as the tag
# builds, and mrw-n with bench/stage-9/no-refusal.patch applied. Usage:
#   bench/stage-9/build.sh TAG OUTDIR
# Prints `m sha256: <hex>` and `n sha256: <hex>`. The tag is exported into a
# temporary directory that is its own repository, so neither the caller's
# checkout nor an enclosing one is touched or consulted; the patched source is
# never compiled by this repository's own build. The builds run with no
# inherited Go flags or workspace and no network, so a missing toolchain or
# module fails instead of being fetched.
set -eu

[ $# -eq 2 ] || { echo "usage: build.sh TAG OUTDIR" >&2; exit 2; }
tag=$1
# A caller's git environment must not redirect the export or the patch.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
here=$(cd "$(dirname "$0")" && pwd)
repo=$(git -C "$here" rev-parse --show-toplevel)
# A tag, not a branch or any other ref of the same name.
git -C "$repo" rev-parse --verify --quiet "refs/tags/$tag^{commit}" >/dev/null \
	|| { echo "build.sh: no tag $tag in $repo" >&2; exit 2; }
mkdir -p "$2"
out=$(cd "$2" && pwd)

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT INT TERM
src=$work/src
mkdir "$src"
git -C "$repo" archive -o "$work/src.tar" "refs/tags/$tag^{commit}"
tar -x -C "$src" -f "$work/src.tar"
git -C "$src" init -q

# The toolchain go.mod names, the one the release is built with.
ver=$(awk '/^go [0-9]/ { print $2; exit }' "$src/go.mod")
[ -n "$ver" ] || { echo "build.sh: no go directive in go.mod of $tag" >&2; exit 2; }

build() {
	# GOENV=off ignores `go env -w` settings, which GOFLAGS= alone does not.
	(cd "$src" && env GOENV=off GOFLAGS= GOWORK=off GOPROXY=off GOPRIVATE= GONOPROXY= GONOSUMDB= \
		GOTOOLCHAIN="go$ver" CGO_ENABLED=0 \
		go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$tag" -o "$out/$1" ./cmd/mrw)
}

# sha prints the SHA-256 of a file, or stops the script when it cannot.
sha() {
	if command -v shasum >/dev/null 2>&1; then
		s=$(shasum -a 256 "$1")
	else
		s=$(sha256sum "$1")
	fi
	s=${s%% *}
	case $s in
	*[!0-9a-f]* | '') echo "build.sh: no hash for $1" >&2; exit 1 ;;
	esac
	[ ${#s} -eq 64 ] || { echo "build.sh: $s is not a SHA-256" >&2; exit 1; }
	printf '%s\n' "$s"
}

build mrw-m
git -C "$src" apply "$here/no-refusal.patch"
build mrw-n

m=$(sha "$out/mrw-m")
n=$(sha "$out/mrw-n")
[ "$m" != "$n" ] || { echo "build.sh: M and N are the same binary; the patch did not apply" >&2; exit 1; }
echo "m sha256: $m"
echo "n sha256: $n"
