#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

archive="${1:-}"
if [[ -z "$archive" ]]; then
  echo "usage: idempotency.sh <archive>" >&2
  exit 2
fi

./host/linux/install.sh --archive "$archive"
snap1="$(find /opt/kendali /etc/kendali -type f -exec sha256sum {} + 2>/dev/null | sort)"
./host/linux/install.sh --archive "$archive"
snap2="$(find /opt/kendali /etc/kendali -type f -exec sha256sum {} + 2>/dev/null | sort)"
if [[ "$snap1" != "$snap2" ]]; then
  echo "idempotency FAILED: state changed on second install" >&2
  exit 1
fi
./host/linux/uninstall.sh
if [[ -d /opt/kendali ]] || [[ -f /etc/kendali/rc.yaml ]]; then
  echo "uninstall FAILED: leftovers remain" >&2
  exit 1
fi
echo "idempotency: ok"
