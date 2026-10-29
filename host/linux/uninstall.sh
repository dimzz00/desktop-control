#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

keep_config=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep-config) keep_config=1; shift ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
done

if [[ "$(id -u)" -ne 0 ]]; then
  echo "uninstall.sh: must run as root" >&2
  exit 1
fi

# Only remove what kendali installed.
if [[ -f /opt/kendali/.installed-by-kendali ]]; then
  rm -rf /opt/kendali
fi
rm -f /opt/kendali/current

if [[ "$keep_config" -eq 0 ]]; then
  rm -f /etc/kendali/rc.yaml
fi
echo "uninstall: done"
