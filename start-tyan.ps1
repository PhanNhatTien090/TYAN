$ErrorActionPreference = 'Stop'
$taskRoot = $PSScriptRoot
$taskRuntime = Join-Path $taskRoot '.tyan-runtime'
$taskPgBin = Join-Path $taskRuntime 'pgsql\bin'
$taskPgData = Join-Path $taskRuntime 'postgres-data'
$taskConfigPath = Join-Path $taskRoot 'config.json'
$taskBinary = Join-Path $taskRoot 'tyan.exe'
$taskLogDir = Join-Path $taskRoot 'data'

if (-not (Test-Path -LiteralPath $taskBinary)) { throw 'TYAN is not built. See README.vi.md for installation instructions.' }
if (-not (Test-Path -LiteralPath $taskConfigPath)) { throw 'Missing config.json. Configure PostgreSQL before starting TYAN.' }
New-Item -ItemType Directory -Force -Path $taskLogDir | Out-Null

# The portable database is local to this checkout and is not a Windows service.
if (Test-Path -LiteralPath (Join-Path $taskPgData 'PG_VERSION')) {
    & (Join-Path $taskPgBin 'pg_ctl.exe') -D $taskPgData status *> $null
    if ($LASTEXITCODE -ne 0) {
        & (Join-Path $taskPgBin 'pg_ctl.exe') -D $taskPgData -l (Join-Path $taskLogDir 'postgres.log') -w start
        if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL failed to start. See data/postgres.log.' }
    }
}

$taskPidPath = Join-Path $taskLogDir 'tyan.pid'
if (Test-Path -LiteralPath $taskPidPath) {
    $taskSavedPid = Get-Content -LiteralPath $taskPidPath -Raw
    $taskExisting = Get-Process -Id ([int]$taskSavedPid.Trim()) -ErrorAction SilentlyContinue
    if ($taskExisting -and $taskExisting.Path -eq $taskBinary) {
        Write-Output 'TYAN is already running: http://127.0.0.1:8787'
        exit 0
    }
}

$env:ARTEX_CONFIG = $taskConfigPath
$env:ARTEX_SKILL_DIR = Join-Path $taskRoot 'skills'
$taskGitBash = 'C:\Program Files\Git\bin'
if (Test-Path -LiteralPath (Join-Path $taskGitBash 'bash.exe')) { $env:PATH = $taskGitBash + ';' + $env:PATH }
$taskProcess = Start-Process -FilePath $taskBinary -ArgumentList @('-addr', '127.0.0.1:8787', '-data', ('"' + $taskLogDir + '"')) -WorkingDirectory $taskRoot -WindowStyle Hidden -RedirectStandardOutput (Join-Path $taskLogDir 'tyan.stdout.log') -RedirectStandardError (Join-Path $taskLogDir 'tyan.stderr.log') -PassThru
Set-Content -LiteralPath $taskPidPath -Value $taskProcess.Id

for ($taskAttempt = 0; $taskAttempt -lt 60; $taskAttempt++) {
    if ($taskProcess.HasExited) { throw 'TYAN exited. See data/tyan.stderr.log.' }
    try {
        $taskResponse = Invoke-WebRequest -Uri 'http://127.0.0.1:8787/api/health' -UseBasicParsing -TimeoutSec 2
        if ($taskResponse.StatusCode -eq 200) {
            Write-Output 'TYAN is running: http://127.0.0.1:8787'
            Write-Output 'On first use, open /setup to choose your administrator password.'
            exit 0
        }
    } catch { Start-Sleep -Milliseconds 500 }
}
throw 'TYAN startup timed out. See data/tyan.stderr.log.'
