#!/bin/sh
# Fetches the wgpu-native library matching the vendored header into
# internal/wgpu/lib/<os>-<arch>/, which git ignores. A game run from the
# source tree loads it from there.
#
#	internal/wgpu/scripts/fetch.sh [goos goarch]
set -eu
here=$(cd "$(dirname "$0")/.." && pwd)
version=$(tr -d '[:space:]' < "$here/include/VERSION")
goos=${1:-$(go env GOOS)}
goarch=${2:-$(go env GOARCH)}
case "$goos-$goarch" in
	darwin-arm64) asset=macos-aarch64; lib=libwgpu_native.dylib ;;
	darwin-amd64) asset=macos-x86_64; lib=libwgpu_native.dylib ;;
	linux-amd64) asset=linux-x86_64; lib=libwgpu_native.so ;;
	linux-arm64) asset=linux-aarch64; lib=libwgpu_native.so ;;
	windows-amd64) asset=windows-x86_64-gnu; lib=wgpu_native.dll ;;
	windows-arm64) asset=windows-aarch64-msvc; lib=wgpu_native.dll ;;
	*) echo "fetch: no wgpu-native build for $goos-$goarch" >&2; exit 1 ;;
esac
out="$here/lib/$goos-$goarch"
mkdir -p "$out"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
url="https://github.com/gfx-rs/wgpu-native/releases/download/$version/wgpu-$asset-release.zip"
echo "fetch: $url"
curl -fsSL -o "$tmp/wgpu.zip" "$url"
mkdir -p "$tmp/x"
if command -v unzip >/dev/null 2>&1; then
	unzip -q -o "$tmp/wgpu.zip" -d "$tmp/x"
else
	# Windows has no unzip, but its tar (bsdtar) reads zips.
	tar -xf "$tmp/wgpu.zip" -C "$tmp/x"
fi
found=$(find "$tmp/x" -name "$lib" | head -1)
if [ -z "$found" ]; then
	echo "fetch: $lib not in the release" >&2
	exit 1
fi
cp "$found" "$out/$lib"
echo "fetch: $out/$lib"
