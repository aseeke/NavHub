#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
exec ./release/navhub-linux-amd64 --config ./config.yaml --no-browser
