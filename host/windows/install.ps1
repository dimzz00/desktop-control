# SPDX-License-Identifier: AGPL-3.0-or-later
[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$Archive,
    [string]$Url,
    [string]$Sha256
)

$ErrorActionPreference = "Stop"

if (-not $Archive -and -not $Url) {
    throw "either -Archive or -Url is required"
}
if ($Archive -and $Url) {
    throw "-Archive and -Url are mutually exclusive"
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    throw "Administrator privileges required"
}

$base = if ($env:ProgramData) { $env:ProgramData } else { "C:\ProgramData" }
$root = Join-Path $base "kendali"
$releases = Join-Path $root "releases"

if ($Archive) {
    if (-not (Test-Path $Archive)) { throw "archive not found: $Archive" }
    if ($Sha256) {
        $actual = (Get-FileHash -Path $Archive -Algorithm SHA256).Hash
        if ($actual -ne $Sha256) { throw "checksum mismatch" }
    }
    if ($PSCmdlet.ShouldProcess($root, "install")) {
        New-Item -ItemType Directory -Force -Path $releases | Out-Null
        $version = "0.1.0"
        $target = Join-Path $releases $version
        New-Item -ItemType Directory -Force -Path $target | Out-Null
        Expand-Archive -Path $Archive -DestinationPath $target -Force
        & (Join-Path $target "rc.exe") install
    }
    return
}

if ($Url) {
    throw "url mode not implemented in this bootstrap"
}
