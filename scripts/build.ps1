$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
Push-Location $taskRoot
try {
    $releaseVersion = (Get-Content web/package.json -Raw | ConvertFrom-Json).version
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
    Copy-Item config.example.yaml, README.md, LICENSE, CHANGELOG.md -Destination release
    foreach ($target in @(@('windows-amd64', 'navhub-windows-amd64.exe', 'navhub.exe'), @('linux-amd64', 'navhub-linux-amd64', 'navhub'), @('linux-arm64', 'navhub-linux-arm64', 'navhub'))) {
        $packageDir = Join-Path release $target[0]
        New-Item -ItemType Directory -Force $packageDir | Out-Null
        Copy-Item (Join-Path release $target[1]) -Destination (Join-Path $packageDir $target[2])
        Copy-Item README.md, config.example.yaml, LICENSE, CHANGELOG.md -Destination $packageDir
        Copy-Item docs, deploy -Destination $packageDir -Recurse -Force
        $packageInputs = @($target[2], 'README.md', 'LICENSE', 'CHANGELOG.md', 'config.example.yaml', 'docs', 'deploy') | ForEach-Object { Join-Path $packageDir $_ }
        if ($target[0] -eq 'windows-amd64') {
            $launcherNames = @('start-background.vbs', 'stop-background.vbs', 'navhub-background.ps1')
            foreach ($launcherName in $launcherNames) {
                Copy-Item (Join-Path 'deploy/windows' $launcherName) -Destination $packageDir
                $packageInputs += Join-Path $packageDir $launcherName
            }
        }
        Compress-Archive -Path $packageInputs -DestinationPath (Join-Path release ("navhub-" + $releaseVersion + '-' + $target[0] + '.zip')) -Force
    }
    Get-FileHash ("release/navhub-" + $releaseVersion + '-*.zip') -Algorithm SHA256 | ForEach-Object { "$($_.Hash.ToLower())  $(Split-Path -Leaf $_.Path)" } | Set-Content release/SHA256SUMS.txt -Encoding ascii
} finally { Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue; Pop-Location }
