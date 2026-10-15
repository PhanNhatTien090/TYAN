$ErrorActionPreference = 'Stop'
$taskRoot = $PSScriptRoot
$taskBinary = Join-Path $taskRoot 'tyan.exe'
$taskPidPath = Join-Path $taskRoot 'data\tyan.pid'
if (Test-Path -LiteralPath $taskPidPath) {
    $taskSavedPid = [int](Get-Content -LiteralPath $taskPidPath -Raw).Trim()
    $taskProcess = Get-Process -Id $taskSavedPid -ErrorAction SilentlyContinue
    if ($taskProcess -and $taskProcess.Path -eq $taskBinary) {
        Stop-Process -Id $taskSavedPid
        $taskProcess.WaitForExit(10000) | Out-Null
        Write-Output 'TYAN stopped.'
    }
    Remove-Item -LiteralPath $taskPidPath
}
$taskPgControl = Join-Path $taskRoot '.tyan-runtime\pgsql\bin\pg_ctl.exe'
$taskPgData = Join-Path $taskRoot '.tyan-runtime\postgres-data'
if ((Test-Path -LiteralPath $taskPgControl) -and (Test-Path -LiteralPath (Join-Path $taskPgData 'PG_VERSION'))) {
    & $taskPgControl -D $taskPgData status *> $null
    if ($LASTEXITCODE -eq 0) {
        & $taskPgControl -D $taskPgData -m fast -w stop
        if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL did not stop cleanly.' }
    }
}
