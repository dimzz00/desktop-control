# SPDX-License-Identifier: AGPL-3.0-or-later
Describe 'install.ps1' {
    BeforeAll {
        $script = Join-Path $PSScriptRoot '..\install.ps1'
    }
    It 'menolak tanpa Administrator' {
        { & $script -Archive 'x.zip' -ErrorAction Stop } | Should -Throw '*Administrator*'
    }
}
