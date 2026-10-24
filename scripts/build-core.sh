#!/usr/bin/env bash
# Copyright (c) 2026 AllIsA1.
# SPDX-License-Identifier: MIT
# Builds the OpenFlux exit from a reviewed upstream commit, for both architectures.
#
#   scripts/build-core.sh v0.13.0 <40-hex commit> <out-dir>
#
# The upstream tag must point at exactly that commit, so the version this plugin names is what the source
# says it is. Writes <out-dir>/openflux-linux-{amd64,arm64} and a .sha256 file beside each. The binaries are
# static (CGO off) and the source is fetched, never vendored. CORE_REPO_URL replaces the upstream address
# (tests use a local repository); CI never sets it.
set -euo pipefail

die() { echo "build-core: $*" >&2; exit 1; }

version=${1:-}
commit=${2:-}
out=${3:-}
[[ -n $out ]] || die "usage: build-core.sh <version> <commit> <out-dir>"
repo=${CORE_REPO_URL:-https://github.com/p1neappleXpress/OpenFlux.git}
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "the version must look like v1.2.3, got '$version'"
[[ $commit =~ ^[0-9a-f]{40}$ ]] || die "the commit must be a full lowercase SHA, got '$commit'"

# The tag must name the commit (an annotated tag is peeled with ^{}).
refs=$(git ls-remote "$repo" "refs/tags/$version" "refs/tags/$version^{}")
peeled=$(awk '$2 ~ /\^\{\}$/ {print $1}' <<< "$refs")
plain=$(awk '$2 !~ /\^\{\}$/ {print $1}' <<< "$refs")
[[ ${peeled:-$plain} == "$commit" ]] || die "$version in $repo is ${peeled:-${plain:-missing}}, not $commit"

mkdir -p "$out"
out=$(cd "$out" && pwd)
src=$(mktemp -d)
trap 'rm -rf "$src"' EXIT
git init -q "$src"
git -C "$src" remote add origin "$repo"
git -C "$src" fetch -q --depth 1 origin "$commit"
git -C "$src" checkout -q FETCH_HEAD
[[ $(git -C "$src" rev-parse HEAD) == "$commit" ]] || die "fetched a different commit than $commit"

for arch in amd64 arm64; do
  file=openflux-linux-$arch
  (cd "$src" && GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$out/$file" .)
  (cd "$out" && sha256sum "$file" | cut -d' ' -f1 > "$file.sha256")
done
echo "built OpenFlux $version ($commit) in $out"
