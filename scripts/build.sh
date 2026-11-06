#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
release_version=$(node -p "require('./web/package.json').version")
(cd web && npm ci && npm test && npm run build)
go test ./...
mkdir -p release
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o release/navhub-windows-amd64.exe .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o release/navhub-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o release/navhub-linux-arm64 .
cp config.example.yaml README.md LICENSE CHANGELOG.md release/
for platform in windows-amd64 linux-amd64 linux-arm64; do
  mkdir -p "release/$platform"
  if [ "$platform" = windows-amd64 ]; then
    package_binary=navhub.exe
    cp release/navhub-windows-amd64.exe "release/$platform/navhub.exe"
  else
    package_binary=navhub
    cp "release/navhub-$platform" "release/$platform/navhub"
  fi
  cp README.md config.example.yaml LICENSE CHANGELOG.md "release/$platform/"
  cp -R docs deploy "release/$platform/"
  set -- "$package_binary" README.md LICENSE CHANGELOG.md config.example.yaml docs deploy
  if [ "$platform" = windows-amd64 ]; then
    cp deploy/windows/*.vbs deploy/windows/*.ps1 "release/$platform/"
    set -- "$@" start-background.vbs stop-background.vbs navhub-background.ps1
  fi
  tar -czf "release/navhub-$release_version-$platform.tar.gz" -C "release/$platform" "$@"
done
(cd release && sha256sum navhub-"$release_version"-*.tar.gz > SHA256SUMS.txt)
