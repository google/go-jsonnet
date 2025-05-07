#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

JSONNET_VERSION="$(grep -Ee '^\s*const\s+version\s*=\s*"[^"]+"\s*$' vm.go | sed -E -e 's/[^"]+"([^"]+)".*/\1/')"

# GITHUB_REF is set by GH actions, see
# https://docs.github.com/en/actions/learn-github-actions/environment-variables#default-environment-variables

VERSION_SUFFIX=
if [[ ( "${GITHUB_REF_TYPE}" != 'branch' || "${GITHUB_REF_NAME}" != "prepare-release" ) &&
      ( "${GITHUB_REF_TYPE}" != 'tag' || "${GITHUB_REF_NAME}" != "${JSONNET_VERSION}" ) ]]; then
    >&2 echo 'WARNING: Jsonnet library version in header does not match release ref. Adding commit suffix.'
    VERSION_SUFFIX="-${GITHUB_SHA:0:9}"
fi

# A prefix is added to better match the GitHub generated archives.
PREFIX="go-jsonnet-${JSONNET_VERSION}${VERSION_SUFFIX}"
ARCHIVE="go-jsonnet-${JSONNET_VERSION}${VERSION_SUFFIX}.tar.gz"
git archive --format=tar --prefix="${PREFIX}"/ "${GITHUB_SHA}" | gzip > "$ARCHIVE"
ARCHIVE_SHA=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')

echo "archive_sha256=${ARCHIVE_SHA}" >> "$GITHUB_OUTPUT"
echo "go_jsonnet_version=${JSONNET_VERSION}" >> "$GITHUB_OUTPUT"
echo "go_jsonnet_version_permanent=${JSONNET_VERSION}${VERSION_SUFFIX}" >> "$GITHUB_OUTPUT"
