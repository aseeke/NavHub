param(
    [ValidateSet('Start', 'Stop')]
    [string]$Action = 'Start',
    [switch]$NoDialog
)

$ErrorActionPreference = 'Stop'
try {
    $exe = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot 'navhub.exe'))
    # Match the full path so another NavHub installation is never stopped.
    $running = @(Get-CimInstance Win32_Process -Filter "Name='navhub.exe'" | Where-Object {
        $_.ExecutablePath -and [string]::Equals($_.ExecutablePath, $exe, [StringComparison]::OrdinalIgnoreCase)
    })

    if ($Action -eq 'Stop') {
        foreach ($instance in $running) {
            $process = Get-Process -Id $instance.ProcessId -ErrorAction SilentlyContinue
            if ($process -and [string]::Equals($process.Path, $exe, [StringComparison]::OrdinalIgnoreCase)) {
                $process | Stop-Process -ErrorAction Stop
                $process.WaitForExit()
            }
        }
        Write-Output 'NavHub stopped. Navigation data was retained.'
    } elseif ($running.Count -gt 0) {
        Write-Output 'NavHub is already running from this folder.'
    } else {
        if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
            throw 'navhub.exe was not found. Keep the scripts in the same folder as navhub.exe.'
        }
        $config = Join-Path $PSScriptRoot 'config.yaml'
        $process = Start-Process -FilePath $exe -ArgumentList @('--config', ('"{0}"' -f $config), '--no-browser') -WorkingDirectory $PSScriptRoot -WindowStyle Hidden -PassThru
        Start-Sleep -Milliseconds 1500
        $process.Refresh()
        if ($process.HasExited) {
            throw 'NavHub exited during startup. Check data\navhub.log, the configured port, or run navhub.exe directly to see the error.'
        }
        Write-Output ('NavHub started in background. PID: {0}. Default URL: http://127.0.0.1:3002' -f $process.Id)
    }
} catch {
    if ($NoDialog) { throw }
    Add-Type -AssemblyName System.Windows.Forms
    [void][System.Windows.Forms.MessageBox]::Show($_.Exception.Message, 'NavHub', 'OK', 'Error')
    exit 1
}
