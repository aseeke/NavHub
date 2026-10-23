$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
Push-Location $taskRoot
try {
    Push-Location web
    try { npm ci; if ($LASTEXITCODE) { throw 'npm ci failed' }; npm test; if ($LASTEXITCODE) { throw 'Frontend tests failed' }; npm run build; if ($LASTEXITCODE) { throw 'Frontend build failed' } } finally { Pop-Location }
    go test ./...
    if ($LASTEXITCODE) { throw 'Go tests failed' }
    New-Item -ItemType Directory -Force release | Out-Null
    $env:CGO_ENABLED = '0'
    foreach ($target in @(@('windows', 'amd64', 'navhub-windows-amd64.exe'), @('linux', 'amd64', 'navhub-linux-amd64'), @('linux', 'arm64', 'navhub-linux-arm64'))) {
        $env:GOOS = $target[0]; $env:GOARCH = $target[1]
        go build -trimpath -ldflags '-s -w' -o (Join-Path release $target[2]) .
        if ($LASTEXITCODE) { throw "Build failed: $target" }
    }
    Copy-Item config.example.yaml, README.md, LICENSE -Destination release
    foreach ($target in @(@('windows-amd64', 'navhub-windows-amd64.exe', 'navhub.exe'), @('linux-amd64', 'navhub-linux-amd64', 'navhub'), @('linux-arm64', 'navhub-linux-arm64', 'navhub'))) {
        $packageDir = Join-Path release $target[0]
        New-Item -ItemType Directory -Force $packageDir | Out-Null
        Copy-Item (Join-Path release $target[1]) -Destination (Join-Path $packageDir $target[2])
        Copy-Item README.md, config.example.yaml, LICENSE -Destination $packageDir
        Copy-Item docs, deploy -Destination $packageDir -Recurse -Force
        Compress-Archive -Path (Join-Path $packageDir '*') -DestinationPath (Join-Path release ("navhub-1.0.0-" + $target[0] + '.zip')) -Force
    }
    Get-FileHash release/navhub-* -Algorithm SHA256 | ForEach-Object { "$($_.Hash.ToLower())  $(Split-Path -Leaf $_.Path)" } | Set-Content release/SHA256SUMS -Encoding ascii
} finally { Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue; Pop-Location }
