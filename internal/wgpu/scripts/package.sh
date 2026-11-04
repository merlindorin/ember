#!/bin/sh
# Builds a game with the wgpu-native library it loads beside it, into
# $DIST (dist by default):
#
#	macOS           <name>-darwin-<arch>/<name>.app: the game in
#	                Contents/MacOS, the library in Contents/Frameworks, its
#	                licences in Contents/Resources, the whole signed ad hoc
#	Windows, Linux  <name>-<os>-<arch>/: the game, the library and its
#	                licences side by side
#
# The library is fetched first when task wgpu hasn't. Windows builds are
# windowed (no console) and cross compile from macOS or Linux with the C
# compiler in CC when it names one for Windows, else with mingw-w64 for
# amd64 (brew install mingw-w64) and zig for arm64 (brew install zig;
# CC="zig cc -target aarch64-windows-gnu"), which mingw-w64 doesn't cover.
# Linux builds need a Linux machine, for GLFW's X11 and Wayland headers,
# and build with GLFW's vulkan tag, so the game links libvulkan (which
# wgpu draws through there) rather than libGL, which it never calls: the
# build machine needs libvulkan-dev, a player's libvulkan1. Mac builds
# target macOS $MACOS_MIN (11.0 by default), the Info.plist's minimum
# too, rather than the build machine's own version, and the package fails
# when the game or the library still asks for a later one.
#
#	internal/wgpu/scripts/package.sh <package> <name> [goos goarch]
set -eu
if [ $# -lt 2 ]; then
	echo "usage: package.sh <package> <name> [goos goarch]" >&2
	exit 2
fi
pkg=$1
name=$2
goos=${3:-$(go env GOOS)}
goarch=${4:-$(go env GOARCH)}
here=$(cd "$(dirname "$0")/.." && pwd)
dist=${DIST:-dist}

case "$goos" in
	windows) lib=wgpu_native.dll; exe=$name.exe ;;
	darwin) lib=libwgpu_native.dylib; exe=$name ;;
	*) lib=libwgpu_native.so; exe=$name ;;
esac
src="$here/lib/$goos-$goarch/$lib"
[ -f "$src" ] || "$here/scripts/fetch.sh" "$goos" "$goarch"

ldflags="-s -w"
tags=
[ "$goos" = linux ] && tags=vulkan
if [ "$goos" = windows ]; then
	# zig ignores -H=windowsgui, so the subsystem goes to the linker too.
	ldflags="$ldflags -H=windowsgui -extldflags=-Wl,--subsystem,windows"
	if [ "$(go env GOHOSTOS)" != windows ]; then
		# The host's own compiler can't build for Windows: a CC that is
		# one of those is replaced, any other is the caller's choice.
		case "${CC:-}" in
			"" | cc | gcc | clang | */cc | */gcc | */clang)
				case "$goarch" in
					arm64) CC="zig cc -target aarch64-windows-gnu" ;;
					*) CC=x86_64-w64-mingw32-gcc ;;
				esac ;;
		esac
		export CC
	fi
fi

# minos is the macOS version the Mach-O file $1 says it needs.
minos() {
	otool -l "$1" | awk '/LC_BUILD_VERSION|LC_VERSION_MIN_MACOSX/ { found = 1 }
		found && ($1 == "minos" || $1 == "version") { print $2; exit }'
}

# later reports whether version $1 is later than $2, both major.minor[.patch].
later() {
	awk -v a="$1" -v b="$2" 'BEGIN {
		n = split(a, x, "."); m = split(b, y, ".")
		for (i = 1; i <= 3; i++) { if (x[i] + 0 != y[i] + 0) exit !(x[i] + 0 > y[i] + 0) }
		exit 1
	}'
}

if [ "$goos" = darwin ]; then
	min=${MACOS_MIN:-11.0}
	export MACOSX_DEPLOYMENT_TARGET=$min
	export CGO_CFLAGS="${CGO_CFLAGS:--O2 -g} -mmacosx-version-min=$min"
	export CGO_LDFLAGS="${CGO_LDFLAGS:--O2 -g} -mmacosx-version-min=$min"
	app="$dist/$name-$goos-$goarch/$name.app"
	rm -rf "$app"
	mkdir -p "$app/Contents/MacOS" "$app/Contents/Frameworks"
	CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build -trimpath -tags "$tags" -ldflags "$ldflags" -o "$app/Contents/MacOS/$exe" "$pkg"
	cp "$src" "$app/Contents/Frameworks/$lib"
	mkdir -p "$app/Contents/Resources"
	cp "$here"/include/webgpu/LICENSE-wgpu-native.* "$app/Contents/Resources/"
	for f in "$app/Contents/MacOS/$exe" "$app/Contents/Frameworks/$lib"; do
		v=$(minos "$f")
		if [ -z "$v" ] || later "$v" "$min"; then
			echo "package: $f needs macOS ${v:-unknown}, later than $min" >&2
			exit 1
		fi
	done
	cat > "$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key><string>$exe</string>
	<key>CFBundleIdentifier</key><string>dev.ember.$name</string>
	<key>CFBundleName</key><string>$name</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>${VERSION:-0.0.0}</string>
	<key>LSMinimumSystemVersion</key><string>$min</string>
	<key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF
	codesign --force --sign - "$app/Contents/Frameworks/$lib"
	codesign --force --sign - "$app"
	codesign --verify --strict "$app"
	echo "package: $app"
	exit 0
fi

out="$dist/$name-$goos-$goarch"
rm -rf "$out"
mkdir -p "$out"
CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build -trimpath -tags "$tags" -ldflags "$ldflags" -o "$out/$exe" "$pkg"
cp "$src" "$out/$lib"
cp "$here"/include/webgpu/LICENSE-wgpu-native.* "$out/"
echo "package: $out"
