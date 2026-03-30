$ErrorActionPreference = "Stop"

$RunDir = Join-Path $HOME "claw-executor"
$TokenFile = Join-Path $RunDir "glaw-executor-token.txt"
$ExecutorScript = Join-Path $RunDir "claw_executor.py"
$StartLogPath = Join-Path $RunDir "claw-executor-start.txt"
$RuntimeLogPath = Join-Path $RunDir "claw-executor-runtime.log"
$WorkerUrl = "https://remote-executor.io99.xyz"

New-Item -ItemType Directory -Force -Path $RunDir | Out-Null
Set-Location $RunDir

if (!(Test-Path -LiteralPath $TokenFile)) {
    throw "Missing token file: $TokenFile"
}
if (!(Test-Path -LiteralPath $ExecutorScript)) {
    throw "Missing executor script: $ExecutorScript"
}

$env:EXECUTOR_TOKEN = (Get-Content -LiteralPath $TokenFile -Raw).Trim()
if ([string]::IsNullOrWhiteSpace($env:EXECUTOR_TOKEN)) {
    throw "EXECUTOR_TOKEN is empty"
}

$command = @"
`$env:EXECUTOR_TOKEN = '$env:EXECUTOR_TOKEN'
Set-Location '$RunDir'
python '$ExecutorScript' --worker-url '$WorkerUrl' --agent-id 'claw-executor' *>> '$RuntimeLogPath'
"@

$proc = Start-Process -FilePath "pwsh" -ArgumentList @("-NoLogo", "-NoExit", "-Command", $command) -WorkingDirectory $RunDir -PassThru

@(
    "StartedAt: $(Get-Date -Format o)"
    "PWD: $((Get-Location).Path)"
    "ExecutorScript: $ExecutorScript"
    "WorkerURL: $WorkerUrl"
    "PID: $($proc.Id)"
    "RuntimeLog: $RuntimeLogPath"
) | Set-Content -LiteralPath $StartLogPath -Encoding UTF8

Write-Host "claw-executor started (PID: $($proc.Id))"
Write-Host "start log: $StartLogPath"
