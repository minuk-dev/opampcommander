#!/usr/bin/env bash
# Prints the project version for the current commit. Output must stay in
# lockstep with .goreleaser.yaml's snapshot.version_template so the apiserver
# and web images, built in separate workflows, end up with identical tags.
#
# Release build (no flag, HEAD == v*): "<tag without leading v>"
#                                         e.g. "0.1.39"
# Snapshot build (--snapshot, OR no flag and HEAD is not a tag):
#                "<incpatch last_tag>-SNAPSHOT-<7-char commit>"
#                                         e.g. "0.1.40-SNAPSHOT-7452c0d"
#
# CI MUST pass --snapshot whenever it would also pass --snapshot to goreleaser
# (i.e. for non-tag refs). Auto-detecting from HEAD alone breaks when main
# happens to land on a tagged commit.
set -euo pipefail

snapshot=0
case "${1:-}" in
    --snapshot) snapshot=1 ;;
    "") ;;
    *) echo "usage: $0 [--snapshot]" >&2; exit 2 ;;
esac

if [[ "${snapshot}" -eq 0 ]]; then
    if tag=$(git describe --tags --exact-match HEAD 2>/dev/null); then
        echo "${tag#v}"
        exit 0
    fi
fi

last_tag=$(git describe --tags --abbrev=0 --match='v*' 2>/dev/null || echo "v0.0.0")
last_tag=${last_tag#v}
IFS='.' read -r major minor patch <<<"${last_tag}"
next_patch=$((patch + 1))
short_commit=$(git rev-parse --short=7 HEAD)
echo "${major}.${minor}.${next_patch}-SNAPSHOT-${short_commit}"
