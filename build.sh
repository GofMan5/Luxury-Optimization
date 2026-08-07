#!/usr/bin/env sh
set -eu

version="${1:-1.0.4}"
if ! printf '%s\n' "$version" | grep -Eq '^1\.0\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
  printf '%s\n' 'version must stay in the 1.0.x release line' >&2
  exit 2
fi

project_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
dist="$project_root/dist"
mkdir -p -- "$dist"
staging=$(mktemp -d "$dist/.build-XXXXXXXX")
previous="$staging/previous"

cleanup() {
  if [ -d "$staging" ]; then
    rm -f -- "$previous"/* 2>/dev/null || true
    rmdir -- "$previous" 2>/dev/null || true
    rm -f -- "$staging"/* 2>/dev/null || true
    rmdir -- "$staging" 2>/dev/null || true
  fi
}
trap cleanup EXIT HUP INT TERM

cd "$project_root/backend"
go mod verify
go test -mod=readonly ./...
go vet -mod=readonly ./...

mkdir -p -- "$previous"
targets='windows amd64 Luxury-Optimization-windows-amd64.exe
windows arm64 Luxury-Optimization-windows-arm64.exe
windows 386 Luxury-Optimization-windows-386.exe
linux amd64 Luxury-Optimization-linux-amd64
linux arm64 Luxury-Optimization-linux-arm64'

printf '%s\n' "$targets" | while read -r goos goarch name; do
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -mod=readonly -trimpath -ldflags "-s -w -X github.com/GofMan5/Luxury-Optimization/internal/optimizer.version=$version" -o "$staging/$name" ./cmd/luxury-optimization
done

: > "$staging/SHA256SUMS.txt"
printf '%s\n' "$targets" | while read -r _ _ name; do
  hash=$(sha256sum "$staging/$name" | cut -d ' ' -f 1)
  printf '%s  %s\n' "$hash" "$name" >> "$staging/SHA256SUMS.txt"
done

artifact_names=$(printf '%s\n' "$targets" | awk '{print $3}')
for name in $artifact_names SHA256SUMS.txt; do
  if [ -f "$dist/$name" ]; then mv -- "$dist/$name" "$previous/$name"; fi
done

published=''
rollback() {
  for name in $published; do rm -f -- "$dist/$name"; done
  for name in $artifact_names SHA256SUMS.txt; do
    if [ -f "$previous/$name" ]; then mv -- "$previous/$name" "$dist/$name"; fi
  done
}

for name in $artifact_names SHA256SUMS.txt; do
  if ! mv -- "$staging/$name" "$dist/$name"; then rollback; exit 1; fi
  published="$published $name"
done

if ! (cd "$dist" && sha256sum -c SHA256SUMS.txt); then rollback; exit 1; fi
rm -f -- "$dist/GofMan3-Optimizer-amd64.exe" "$dist/GofMan3-Optimizer-arm64.exe" "$dist/GofMan3-Optimizer-386.exe"
find "$dist" -maxdepth 1 -type f -printf '%f %s bytes\n' | sort
