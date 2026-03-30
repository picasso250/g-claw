param(
    [string]$RepoDir = (Join-Path $HOME "glaw"),
    [string]$ExePath = (Join-Path $HOME "bin\claw-life-saver.exe"),
    [string]$ExecSubjectKeyword = "claw-life-saver"
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "[dev-claw-life-saver] $Message" -ForegroundColor Cyan
}

function Require-Path {
    param(
        [string]$Path,
        [string]$Label
    )

    if (!(Test-Path -LiteralPath $Path)) {
        throw "Missing ${Label}: $Path"
    }
}

$RunDir = (Get-Location).Path
$EnvPath = Join-Path $RunDir ".env"
$MailFilterPath = Join-Path $RunDir "mail_filter_senders.txt"
$CronConfigPath = Join-Path $RunDir "cron.json"
$GatewayDir = Join-Path $RunDir "gateway"
$InitPath = Join-Path $RunDir "INIT.md"
$BinDir = Split-Path -Parent $ExePath

Require-Path -Path $RepoDir -Label "repo dir"
Require-Path -Path (Join-Path $RepoDir ".git") -Label "repo .git"
Require-Path -Path $EnvPath -Label ".env"
Require-Path -Path $MailFilterPath -Label "mail_filter_senders.txt"
Require-Path -Path $CronConfigPath -Label "cron.json"
Require-Path -Path $GatewayDir -Label "gateway dir"
Require-Path -Path $InitPath -Label "INIT.md"

Write-Step "Running git pull in $RepoDir"
Push-Location $RepoDir
try {
    & git pull
    if ($LASTEXITCODE -ne 0) {
        throw "git pull failed with exit code $LASTEXITCODE"
    }

    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

    Write-Step "Building $ExePath"
    & go build -buildvcs=false -o $ExePath .\cmd\glaw
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}

Write-Step "Starting in $RunDir"
Set-Location $RunDir
& $ExePath serve `
    --env $EnvPath `
    --mail-filter $MailFilterPath `
    --cron-config $CronConfigPath `
    --exec-subject-keyword $ExecSubjectKeyword
