$ErrorActionPreference = "Stop"

$OutputPath = Join-Path (Get-Location) "ps-claw-executor-check.txt"

@(
    "GeneratedAt: $(Get-Date -Format o)"
    "PWD: $((Get-Location).Path)"
) | Set-Content -LiteralPath $OutputPath -Encoding UTF8

Get-CimInstance Win32_Process |
    Where-Object { $_.Name -match 'pwsh|powershell|python|node|glaw|claw' } |
    Select-Object ProcessId, Name, CommandLine |
    Sort-Object Name, ProcessId |
    Format-List |
    Out-String |
    Add-Content -LiteralPath $OutputPath -Encoding UTF8

Write-Host "wrote $OutputPath"
