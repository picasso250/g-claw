$ErrorActionPreference = "Stop"

$OutputPath = Join-Path (Get-Location) "observe-shuyao-state.txt"
$WorkDir = Join-Path $HOME "g-claw"
$RepoDir = Join-Path $HOME "glaw"

function Add-Section {
    param([string]$Title)
    "" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
    "===== $Title =====" | Add-Content -LiteralPath $OutputPath -Encoding UTF8
}

@(
    "GeneratedAt: $(Get-Date -Format o)"
    "PWD: $((Get-Location).Path)"
    "WorkDir: $WorkDir"
    "RepoDir: $RepoDir"
) | Set-Content -LiteralPath $OutputPath -Encoding UTF8

Add-Section "WorkDir Exists"
"Exists: $(Test-Path -LiteralPath $WorkDir)" | Add-Content -LiteralPath $OutputPath -Encoding UTF8

Add-Section "Repo Exists"
"Exists: $(Test-Path -LiteralPath $RepoDir)" | Add-Content -LiteralPath $OutputPath -Encoding UTF8

if (Test-Path -LiteralPath $RepoDir) {
    Add-Section "Git Status"
    & git -C $RepoDir status --short 2>&1 | Add-Content -LiteralPath $OutputPath -Encoding UTF8

    Add-Section "Git HEAD"
    & git -C $RepoDir rev-parse HEAD 2>&1 | Add-Content -LiteralPath $OutputPath -Encoding UTF8

    Add-Section "go.mod Status"
    & git -C $RepoDir status --short -- go.mod 2>&1 | Add-Content -LiteralPath $OutputPath -Encoding UTF8

    Add-Section "go.mod Diff"
    & git -C $RepoDir diff -- go.mod 2>&1 | Add-Content -LiteralPath $OutputPath -Encoding UTF8
}

Add-Section "Relevant Processes"
Get-CimInstance Win32_Process |
    Where-Object { $_.Name -match 'pwsh|powershell|python|node|glaw|claw' } |
    Select-Object ProcessId, Name, CommandLine |
    Sort-Object Name, ProcessId |
    Format-List |
    Out-String |
    Add-Content -LiteralPath $OutputPath -Encoding UTF8

Write-Host "wrote $OutputPath"
