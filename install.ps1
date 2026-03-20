$ErrorActionPreference = "Stop"

$PythonWingetId = "Python.Python.3.13"

function Write-Step {
    param(
        [string]$Message
    )

    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Test-WingetAvailable {
    return $null -ne (Resolve-WingetCommand)
}

function Resolve-WingetCommand {
    $candidates = @(
        "winget.exe",
        "winget",
        (Join-Path $env:LOCALAPPDATA "Microsoft\WindowsApps\winget.exe")
    ) | Where-Object { $_ }

    foreach ($candidate in $candidates) {
        $command = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($command) {
            return $command.Source
        }

        if (Test-Path $candidate) {
            return $candidate
        }
    }

    return $null
}

function Test-CommandExists {
    param(
        [string[]]$Names
    )

    foreach ($name in $Names) {
        if (Get-Command $name -ErrorAction SilentlyContinue) {
            return $true
        }
    }

    return $false
}

function Test-WingetPackageInstalled {
    param(
        [string]$WingetCommand,
        [string]$Id
    )

    $output = & $WingetCommand list --exact --id $Id --accept-source-agreements 2>$null | Out-String
    return $LASTEXITCODE -eq 0 -and $output -match [regex]::Escape($Id)
}

function Install-WingetPackageIfMissing {
    param(
        [string]$WingetCommand,
        [string]$Name,
        [string]$WingetId,
        [string[]]$CommandNames
    )

    if ($CommandNames -and (Test-CommandExists -Names $CommandNames)) {
        Write-Host "[skip] $Name is already available in PATH."
        return
    }

    if (Test-WingetPackageInstalled -WingetCommand $WingetCommand -Id $WingetId) {
        Write-Host "[skip] $Name is already installed via winget."
        return
    }

    Write-Host "[install] $Name"
    & $WingetCommand install --exact --id $WingetId --accept-package-agreements --accept-source-agreements
}

function Resolve-NpmCommand {
    $candidates = @(
        "npm.cmd",
        "npm",
        (Join-Path $env:ProgramFiles "nodejs\npm.cmd"),
        (Join-Path ${env:ProgramFiles(x86)} "nodejs\npm.cmd"),
        (Join-Path $env:LocalAppData "Programs\nodejs\npm.cmd")
    ) | Where-Object { $_ }

    foreach ($candidate in $candidates) {
        $command = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($command) {
            return $command.Source
        }

        if (Test-Path $candidate) {
            return $candidate
        }
    }

    throw "npm was not found after Node.js installation. Restart the shell and run the script again."
}

function Test-NpmGlobalPackageInstalled {
    param(
        [string]$NpmCommand,
        [string]$PackageName
    )

    & $NpmCommand list -g $PackageName --depth=0 *> $null
    return $LASTEXITCODE -eq 0
}

function Install-NpmGlobalPackageIfMissing {
    param(
        [string]$NpmCommand,
        [string]$PackageName
    )

    if (Test-NpmGlobalPackageInstalled -NpmCommand $NpmCommand -PackageName $PackageName) {
        Write-Host "[skip] npm package $PackageName is already installed globally."
        return
    }

    Write-Host "[install] npm package $PackageName"
    & $NpmCommand install -g $PackageName
}

if (-not (Test-WingetAvailable)) {
    throw "winget is required but was not found. Install App Installer from Microsoft Store first."
}

$wingetCommand = Resolve-WingetCommand

$packages = @(
    @{
        Name = "curl"
        WingetId = "cURL.cURL"
        CommandNames = @("curl.exe", "curl")
    },
    @{
        Name = "PowerShell 7"
        WingetId = "Microsoft.PowerShell"
        CommandNames = @("pwsh.exe", "pwsh")
    },
    @{
        Name = "Git"
        WingetId = "Git.Git"
        CommandNames = @("git.exe", "git")
    },
    @{
        Name = "Python"
        WingetId = $PythonWingetId
        CommandNames = @("python.exe", "python")
    },
    @{
        Name = "Node.js"
        WingetId = "OpenJS.NodeJS"
        CommandNames = @("node.exe", "node", "npm.cmd", "npm")
    },
    @{
        Name = "Go"
        WingetId = "GoLang.Go"
        CommandNames = @("go.exe", "go")
    }
)

Write-Step "Installing base dependencies from INSTALL.md"
foreach ($package in $packages) {
    Install-WingetPackageIfMissing -WingetCommand $wingetCommand -Name $package.Name -WingetId $package.WingetId -CommandNames $package.CommandNames
}

Write-Step "Installing global npm CLIs"
$npmCommand = Resolve-NpmCommand
Install-NpmGlobalPackageIfMissing -NpmCommand $npmCommand -PackageName "@google/gemini-cli"
Install-NpmGlobalPackageIfMissing -NpmCommand $npmCommand -PackageName "@kilocode/cli"

Write-Step "Done"
