#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
(cd web && npm ci && npm test && npm run build)
go test ./...
mkdir -p release
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o release/navhub-windows-amd64.exe .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o release/navhub-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o release/navhub-linux-arm64 .
cp config.example.yaml README.md LICENSE release/
for platform in windows-amd64 linux-amd64 linux-arm64; do
  mkdir -p "release/$platform"
  if [ "$platform" = windows-amd64 ]; then
    cp release/navhub-windows-amd64.exe "release/$platform/navhub.exe"
  else
    cp "release/navhub-$platform" "release/$platform/navhub"
  fi
  cp README.md config.example.yaml LICENSE "release/$platform/"
  cp -R docs deploy "release/$platform/"
  tar -czf "release/navhub-1.0.0-$platform.tar.gz" -C "release/$platform" .
done
(cd release && sha256sum navhub-* > SHA256SUMS)
