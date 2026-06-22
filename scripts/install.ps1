[CmdletBinding()]
param(
    [switch]$SkipLaunch,
    [switch]$NoDeps
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$AppName = "subweazl"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$InstallRoot = if ($env:SUBWEAZL_HOME) { $env:SUBWEAZL_HOME } else { Join-Path $env:APPDATA "subweazl" }
$BinDir = Join-Path $InstallRoot "bin"
$BinPath = Join-Path $BinDir "$AppName.exe"
$GoCache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $RepoRoot ".gocache" }
$GoModCache = if ($env:GOMODCACHE) { $env:GOMODCACHE } else { Join-Path $RepoRoot ".gomodcache" }

function Test-Command($Name) {
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Update-SessionPath {
    $machine = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $user = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = @($machine, $user) -join ";"
}

function Add-UserPath($PathToAdd) {
    $current = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = @()
    if ($current) {
        $parts = $current -split ";" | Where-Object { $_ }
    }
    if ($parts -contains $PathToAdd) {
        return
    }
    $next = (@($parts) + $PathToAdd) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $next, "User")
    Update-SessionPath
    Write-Host "Added $PathToAdd to your user PATH"
}

function Find-Executable($Name) {
    $found = Get-Command $Name -ErrorAction SilentlyContinue
    if ($found) {
        return $found.Source
    }

    $candidates = @()
    if ($env:LOCALAPPDATA) {
        $candidates += Join-Path $env:LOCALAPPDATA "Programs\mpv\$Name"
        $candidates += Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
    }
    if ($env:ProgramFiles) {
        $candidates += Join-Path $env:ProgramFiles "mpv\$Name"
    }
    if (${env:ProgramFiles(x86)}) {
        $candidates += Join-Path ${env:ProgramFiles(x86)} "mpv\$Name"
    }
    foreach ($candidate in $candidates) {
        if (Test-Path $candidate -PathType Leaf) {
            return $candidate
        }
        if (Test-Path $candidate -PathType Container) {
            $hit = Get-ChildItem -Path $candidate -Filter $Name -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($hit) {
                return $hit.FullName
            }
        }
    }
    return $null
}

function Ensure-CommandPath($Command, $ExecutableName) {
    if (Test-Command $Command) {
        return $true
    }
    $exe = Find-Executable $ExecutableName
    if (-not $exe) {
        return $false
    }
    Add-UserPath (Split-Path -Parent $exe)
    return (Test-Command $Command)
}

function Install-Winget($Ids) {
    if (-not (Test-Command "winget")) {
        return $false
    }
    foreach ($id in $Ids) {
        & winget install --id $id --exact --accept-package-agreements --accept-source-agreements
        if ($LASTEXITCODE -eq 0) {
            return $true
        }
        Write-Host "winget could not install $id"
    }
    return $false
}

function Install-Choco($Name) {
    if (-not (Test-Command "choco")) {
        return $false
    }
    & choco install $Name -y
    if ($LASTEXITCODE -eq 0) {
        return $true
    }
    Write-Host "choco could not install $Name"
    return $false
}

function Install-Scoop($Name) {
    if (-not (Test-Command "scoop")) {
        return $false
    }
    & scoop install $Name
    if ($LASTEXITCODE -eq 0) {
        return $true
    }
    Write-Host "scoop could not install $Name"
    return $false
}

function Ensure-Dependency($Command, $WingetIds, $ChocoName, $ScoopName) {
    if (Test-Command $Command) {
        return
    }
    if ($NoDeps) {
        throw "$Command was not found on PATH. Rerun without -NoDeps or install it manually."
    }

    Write-Host "$Command was not found. Trying to install it..."
    $installed = (Install-Winget $WingetIds) -or (Install-Choco $ChocoName) -or (Install-Scoop $ScoopName)
    Update-SessionPath
    if (-not $installed -or -not (Ensure-CommandPath $Command "$Command.exe")) {
        throw "Could not install or locate $Command. Install it manually, then rerun this script."
    }
}

function Get-GoVersion {
    $goExe = (Get-Command "go" -ErrorAction Stop).Source
    $raw = (& $goExe "version")
    if ($raw -match "go([0-9]+\.[0-9]+)") {
        return [version]$Matches[1]
    }
    throw "Could not parse Go version from: $raw"
}

function Check-GoVersion {
    $requiredText = (Select-String -Path (Join-Path $RepoRoot "go.mod") -Pattern "^go\s+([0-9]+\.[0-9]+)" | Select-Object -First 1).Matches.Groups[1].Value
    $required = [version]$requiredText
    $current = Get-GoVersion
    if ($current -lt $required) {
        throw "Go $required or newer is required. Found Go $current."
    }
}

Ensure-Dependency "go" @("GoLang.Go") "golang" "go"
Ensure-Dependency "mpv" @("9P3JFR0CLLL6") "mpv" "mpv"
Ensure-Dependency "ffmpeg" @("Gyan.FFmpeg.Essentials", "BtbN.FFmpeg.GPL", "Gyan.FFmpeg") "ffmpeg" "ffmpeg"
Check-GoVersion

New-Item -ItemType Directory -Force -Path $BinDir, $GoCache, $GoModCache | Out-Null

Write-Host "Building $AppName..."
Push-Location $RepoRoot
try {
    $env:GOCACHE = $GoCache
    $env:GOMODCACHE = $GoModCache
    $goExe = (Get-Command "go" -ErrorAction Stop).Source
    & $goExe "build" "-buildvcs=false" "-o" $BinPath "./cmd/subweazl"
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }
} finally {
    Pop-Location
}

Add-UserPath $BinDir

Write-Host "Installed $AppName to $BinPath"
Write-Host "If your current shell cannot find it yet, open a new terminal or run:"
Write-Host "  `$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')"

if ($SkipLaunch -or $env:SUBWEAZL_SKIP_LAUNCH -eq "1") {
    Write-Host "Skipping first launch"
} else {
    Write-Host ""
    Write-Host "Launching $AppName..."
    & $BinPath
}
