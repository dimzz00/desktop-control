#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd "$(dirname "$0")/.."
test -f docs/INSTALL.md
test -f docs/CONNECT.md
test -f docs/TROUBLESHOOT.md
test -f docs/SECURITY.md
grep -q "tidak terhubung" docs/CONNECT.md
! grep -rq "TODO" docs/*.md
echo "docs: ok"
