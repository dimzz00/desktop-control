# SPDX-License-Identifier: AGPL-3.0-or-later
[CmdletBinding()]
param(
    [switch]$KeepConfig
)

$ErrorActionPreference = "Stop"

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) { throw "Administrator privileges required" }

$base = if ($env:ProgramData) { $env:ProgramData } else { "C:\ProgramData" }
$root = Join-Path $base "kendali"

if (Test-Path $root) {
    Remove-Item -Recurse -Force $root
}
if (-not $KeepConfig) {
    $cfg = Join-Path $root "rc.yaml"
    if (Test-Path $cfg) { Remove-Item -Force $cfg }
}
Write-Host "uninstall: done"
