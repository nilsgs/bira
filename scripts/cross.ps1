#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$RepoDir = Split-Path -Parent $PSScriptRoot
$SrcDir = Join-Path $RepoDir 'src'
$DistDir = Join-Path $RepoDir 'dist'
$Version = (Get-Content (Join-Path $RepoDir 'VERSION') -Raw).Trim()
$Commit = try { (git -C $RepoDir rev-parse --short HEAD 2>$null) } catch { 'unknown' }
if (-not $Commit) { $Commit = 'unknown' }
$Ldflags = "-s -w -X bira/cmd.version=$Version -X bira/cmd.commit=$Commit"
$Targets = @(
    @{ GOOS = 'linux'; GOARCH = 'amd64'; Output = 'bira-linux-amd64' },
    @{ GOOS = 'linux'; GOARCH = 'arm64'; Output = 'bira-linux-arm64' },
    @{ GOOS = 'darwin'; GOARCH = 'amd64'; Output = 'bira-darwin-amd64' },
    @{ GOOS = 'darwin'; GOARCH = 'arm64'; Output = 'bira-darwin-arm64' },
    @{ GOOS = 'windows'; GOARCH = 'amd64'; Output = 'bira-windows-amd64.exe' },
    @{ GOOS = 'windows'; GOARCH = 'arm64'; Output = 'bira-windows-arm64.exe' }
)

Write-Host "Cross-building bira v${Version}+${Commit}..."
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

Push-Location $SrcDir
try {
    foreach ($Target in $Targets) {
        $env:GOOS = $Target.GOOS
        $env:GOARCH = $Target.GOARCH
        & go build -ldflags $Ldflags -o (Join-Path $DistDir $Target.Output) .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}
