param(
    [string]$RunDir = (Join-Path $HOME "claw-executor"),
    [string]$WorkerUrl = "https://remote-executor.io99.xyz",
    [string]$AgentId = "claw-executor"
)

$ErrorActionPreference = "Stop"

$TokenFile = Join-Path $RunDir "glaw-executor-token.txt"
$ExecutorScript = Join-Path $RunDir "claw_executor.py"
$StartLogPath = Join-Path $RunDir "claw-executor-start.txt"
$RuntimeLogPath = Join-Path $RunDir "claw-executor-runtime.log"

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

@(
    "StartedAt: $(Get-Date -Format o)"
    "PWD: $((Get-Location).Path)"
    "ExecutorScript: $ExecutorScript"
    "WorkerURL: $WorkerUrl"
    "AgentId: $AgentId"
    "RuntimeLog: $RuntimeLogPath"
) | Set-Content -LiteralPath $StartLogPath -Encoding UTF8

$command = @"
`$env:EXECUTOR_TOKEN = '$env:EXECUTOR_TOKEN'
Set-Location '$RunDir'
Write-Host '===== claw-executor start ====='
Get-Content -LiteralPath '$StartLogPath'
Write-Host ''
Write-Host '===== claw-executor runtime ====='
python '$ExecutorScript' --worker-url '$WorkerUrl' --agent-id '$AgentId' 2>&1 | Tee-Object -FilePath '$RuntimeLogPath' -Append
"@

$proc = Start-Process -FilePath "pwsh" -ArgumentList @("-NoLogo", "-NoExit", "-Command", $command) -WorkingDirectory $RunDir -PassThru

$startLines = Get-Content -LiteralPath $StartLogPath
@($startLines + "PID: $($proc.Id)") | Set-Content -LiteralPath $StartLogPath -Encoding UTF8

Write-Host "claw-executor started (PID: $($proc.Id))"
Write-Host "start log: $StartLogPath"
