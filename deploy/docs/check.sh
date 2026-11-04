#!/bin/sh
# Checks a running docs image at $1 (default http://localhost:8080): the health
# check, the redirect to the guide, its pages, the live examples' no-cache and
# their gzipped builds served without a Content-Encoding.
set -eu
base=${1:-http://localhost:8080}
fail=0

header() { curl -sI "$base$1" | tr -d '\r' | grep -i "^$2:" | head -1 | cut -d' ' -f2-; }
status() { curl -s -o /dev/null -w '%{http_code}' "$base$1"; }
expect() {
	if [ "$2" = "$3" ]; then echo "ok   $1"; else echo "FAIL $1: got '$2', want '$3'"; fail=1; fi
}

wasm=
for f in public/live/*.wasm.gz; do
	[ -f "$f" ] && wasm=${f##*/} && break
done
expect "healthz" "$(status /healthz)" 200
expect "/ redirects" "$(status /)" 302
expect "/ redirects to the guide" "$(header / location)" /ember/
expect "the guide" "$(status /ember/)" 200
expect "a missing page" "$(status /ember/no-such-page/)" 404
expect "live page not cached" "$(header /ember/live/index.html cache-control)" no-cache
if [ -n "$wasm" ]; then
	expect "$wasm" "$(status "/ember/live/$wasm")" 200
	expect "$wasm not re-encoded" "$(header "/ember/live/$wasm" content-encoding)" ""
	expect "$wasm not cached" "$(header "/ember/live/$wasm" cache-control)" no-cache
fi
exit $fail
