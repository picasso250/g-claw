$ErrorActionPreference = "Stop"

$OutputPath = Join-Path (Get-Location) "probe-shuyao-before-upgrade.txt"
$RepoDir = Join-Path $HOME "glaw"

function Write-Section {
    param([string]$Title)
    ""
    "===== $Title ====="
}

try {
    $OutputPath = Join-Path (Get-Location) "probe-shuyao-before-upgrade.txt"

    @(
        "GeneratedAt: $(Get-Date -Format o)"
        "UserProfile: $HOME"
        "PWD: $((Get-Location).Path)"
    ) | Set-Content -LiteralPath $OutputPath -Encoding UTF8

    Write-Section "PWD" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    (Get-Location).Path | Add-Content -LiteralPath $OutputPath -Encoding UTF8

    Write-Section "Repo Exists" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    "RepoDir: $RepoDir" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    "Exists: $(Test-Path -LiteralPath $RepoDir)" | Add-Content -LiteralPath $OutputPath -Encoding UTF8

    if (Test-Path -LiteralPath $RepoDir) {
        Write-Section "Git Status" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
        & git -C $RepoDir status --short 2>&1 | Add-Content -LiteralPath $OutputPath -Encoding UTF8

        Write-Section "Git HEAD" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
        & git -C $RepoDir rev-parse HEAD 2>&1 | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    }

    Write-Section "Commands" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    foreach ($name in @("python", "git", "go")) {
        "[$name]" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
        $cmd = Get-Command $name -ErrorAction SilentlyContinue
        if ($null -eq $cmd) {
            "missing" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
        } else {
            $cmd | Format-List * | Out-String | Add-Content -LiteralPath $OutputPath -Encoding UTF8
        }
    }

    Write-Section "Relevant Processes" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    Get-CimInstance Win32_Process |
        Where-Object { $_.Name -match 'glaw|kilocode|gemini|node' } |
        Select-Object ProcessId, Name, CommandLine |
        Format-List |
        Out-String |
        Add-Content -LiteralPath $OutputPath -Encoding UTF8

    Write-Section "Rescue Files" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    foreach ($path in @(
        "$HOME\claw-life-saver\INIT.md",
        "$HOME\claw-life-saver\SOUL.md",
        "$HOME\claw-life-saver\USER.md",
        "$HOME\claw-life-saver\MEMORY.txt",
        "$HOME\claw-life-saver\.env"
    )) {
        "$path :: $(Test-Path -LiteralPath $path)" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    }

    "OK: probe finished" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    Write-Host "Probe finished: $OutputPath"
} catch {
    "ERROR: $($_.Exception.Message)" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    throw
}
