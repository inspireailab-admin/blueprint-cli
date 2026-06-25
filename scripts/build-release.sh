#!/usr/bin/env sh
# Cross-compile the Blueprint CLI for all release platforms.
# Output â†’ cli/dist/blueprint-<os>-<arch>[.exe]
#
# Usage:
#   ./scripts/build-release.sh            # version "dev"
#   ./scripts/build-release.sh v0.1.0     # ldflags-injected version

set -e

VERSION="${1:-dev}"
OUT="dist"
LDFLAGS="-s -w -X github.com/inspireailab-admin/blueprint-cli/internal/cmd.Version=${VERSION}"

rm -rf "$OUT"
mkdir -p "$OUT"

# os arch [ext]
targets="
darwin   amd64
darwin   arm64
linux    amd64
linux    arm64
windows  amd64 .exe
windows  arm64 .exe
"

echo "$targets" | while read -r goos goarch ext; do
  [ -z "$goos" ] && continue
  out="$OUT/blueprint-${goos}-${goarch}${ext}"
  echo "â†’ $out"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "$LDFLAGS" -o "$out" .
done

echo ""
echo "Built:"
ls -la "$OUT/"

echo ""
echo "Release with:"
echo "  gh release create ${VERSION} ${OUT}/blueprint-* --title \"${VERSION}\" --notes \"Blueprint CLI ${VERSION}\""
