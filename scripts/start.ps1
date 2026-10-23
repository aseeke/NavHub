$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $PSScriptRoot)
& ./release/navhub-windows-amd64.exe --config ./config.yaml
