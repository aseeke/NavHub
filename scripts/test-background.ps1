param([string]$Binary = (Join-Path $PSScriptRoot '../release/navhub-windows-amd64.exe'))

$ErrorActionPreference = 'Stop'
$sourceBinary = (Resolve-Path -LiteralPath $Binary).Path
$testDirectory = Join-Path ([IO.Path]::GetTempPath()) ('NavHub background test-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $testDirectory | Out-Null
$launcher = Join-Path $testDirectory 'navhub-background.ps1'
$serviceProcess = $null
try {
    Copy-Item -LiteralPath $sourceBinary -Destination (Join-Path $testDirectory 'navhub.exe')
    Copy-Item (Join-Path $PSScriptRoot '../deploy/windows/*') -Destination $testDirectory
    @'
host: 127.0.0.1
port: 3139
data_dir: data
open_browser: false
log_file: true
'@ | Set-Content -LiteralPath (Join-Path $testDirectory 'config.yaml') -Encoding ascii

    # Exercise the actual double-click entry point, including paths with spaces.
    $wrapper = Start-Process -FilePath wscript.exe -ArgumentList ('"{0}"' -f (Join-Path $testDirectory 'start-background.vbs')) -WindowStyle Hidden -PassThru
    $wrapper.WaitForExit()
    $status = $null
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        try { $status = Invoke-RestMethod 'http://127.0.0.1:3139/api/status' -TimeoutSec 1; break } catch { Start-Sleep -Milliseconds 250 }
    }
    if (-not $status -or $status.initialized) { throw 'Background launcher did not start a fresh isolated instance.' }
    $testExe = Join-Path $testDirectory 'navhub.exe'
    $matches = @(Get-Process -Name navhub -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $testExe })
    if ($matches.Count -ne 1) { throw 'Expected one background process.' }
    $serviceProcess = $matches[0]
    & $launcher -Action Start -NoDialog
    $matches = @(Get-Process -Name navhub -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $testExe })
    if ($matches.Count -ne 1) { throw 'Repeated startup created another process.' }

    $body = @{ username = 'background-test'; password = 'background-test-password-123' } | ConvertTo-Json
    Invoke-RestMethod 'http://127.0.0.1:3139/api/setup' -Method Post -ContentType 'application/json' -Body $body | Out-Null
    $wrapper = Start-Process -FilePath wscript.exe -ArgumentList ('"{0}"' -f (Join-Path $testDirectory 'stop-background.vbs')) -WindowStyle Hidden -PassThru
    $wrapper.WaitForExit()
    if (-not $serviceProcess.WaitForExit(10000)) { throw 'Stop launcher did not stop its process.' }
    & $launcher -Action Stop -NoDialog
    & $launcher -Action Start -NoDialog
    $status = Invoke-RestMethod 'http://127.0.0.1:3139/api/status' -TimeoutSec 5
    if (-not $status.initialized) { throw 'Restart lost the existing database.' }
    if (-not (Test-Path -LiteralPath (Join-Path $testDirectory 'data/navhub.log'))) { throw 'File log was not created.' }
    Write-Output 'PASS: hidden VBS start/stop, paths with spaces, repeated start, restart, database retention and file logging.'
} finally {
    if (Test-Path -LiteralPath $launcher) { & $launcher -Action Stop -NoDialog }
    $resolvedTestDirectory = [IO.Path]::GetFullPath($testDirectory)
    $expectedPrefix = Join-Path ([IO.Path]::GetTempPath()) 'NavHub background test-'
    if (-not $resolvedTestDirectory.StartsWith($expectedPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Unexpected temporary cleanup path.' }
    Remove-Item -LiteralPath $resolvedTestDirectory -Recurse -Force
}
