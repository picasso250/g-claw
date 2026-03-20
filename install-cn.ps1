$ErrorActionPreference = "Stop"

$NodeMsiUrl = "https://sw.pcmgr.qq.com/1238ec81f3baaabbe5c6d57d42c7cca6/69bcedd7/spcmgr/download/node-v24.14.0-x64.msi"
$NpmRegistry = "https://registry.npmmirror.com"

function Write-Step {
    param(
        [string]$Message
    )

    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
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

    throw "npm was not found. Restart the shell after Node.js installation and run the script again."
}

function Install-NodeIfMissing {
    if (Test-CommandExists -Names @("node.exe", "node")) {
        Write-Host "[skip] Node.js is already installed."
        return
    }

    $msiName = Split-Path $NodeMsiUrl -Leaf
    $msiPath = Join-Path $env:TEMP $msiName

    Write-Host "[download] $NodeMsiUrl"
    Invoke-WebRequest -Uri $NodeMsiUrl -OutFile $msiPath

    Write-Host "[install] Node.js"
    $process = Start-Process msiexec.exe -ArgumentList @("/i", $msiPath, "/qn", "/norestart") -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Node.js MSI installation failed with exit code $($process.ExitCode)."
    }

    $nodePaths = @(
        (Join-Path $env:ProgramFiles "nodejs"),
        (Join-Path ${env:ProgramFiles(x86)} "nodejs"),
        (Join-Path $env:LocalAppData "Programs\nodejs")
    ) | Where-Object { $_ -and (Test-Path $_) }

    foreach ($nodePath in $nodePaths) {
        if (($env:PATH -split ';') -notcontains $nodePath) {
            $env:PATH = "$env:PATH;$nodePath"
        }
    }
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
    & $NpmCommand install -g $PackageName --registry=$NpmRegistry
}

Write-Step "Installing Node.js from Tencent mirror"
Install-NodeIfMissing

Write-Step "Installing @google/gemini-cli from npmmirror"
$npmCommand = Resolve-NpmCommand
Install-NpmGlobalPackageIfMissing -NpmCommand $npmCommand -PackageName "@google/gemini-cli"

Write-Step "Done"
