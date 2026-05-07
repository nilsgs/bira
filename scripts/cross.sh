#!/usr/bin/env sh
set -eu

repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
src_dir="$repo_dir/src"
dist_dir="$repo_dir/dist"
version="$(tr -d '\r\n' < "$repo_dir/VERSION")"
commit="$(git -C "$repo_dir" rev-parse --short HEAD 2>/dev/null || printf 'unknown')"
ldflags="-s -w -X bira/cmd.version=${version} -X bira/cmd.commit=${commit}"

echo "Cross-building bira v${version}+${commit}..."
mkdir -p "$dist_dir"

cd "$src_dir"
for spec in \
  "linux amd64 bira-linux-amd64" \
  "linux arm64 bira-linux-arm64" \
  "darwin amd64 bira-darwin-amd64" \
  "darwin arm64 bira-darwin-arm64" \
  "windows amd64 bira-windows-amd64.exe" \
  "windows arm64 bira-windows-arm64.exe"
do
  set -- $spec
  GOOS="$1" GOARCH="$2" go build -ldflags "$ldflags" -o "$dist_dir/$3" .
done
