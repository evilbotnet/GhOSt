#!/usr/bin/env bash
# Install CryptPad as the local office suite (run as root on the device/VM).
# CryptPad runs as the ghost user, on-demand via ghost-cryptpad.service.
# Its sandbox needs a second browser origin (localhost:3001) — ghostd serves
# that as a TCP proxy (GHOST_OFFICE_SAFE_PROXY in ghost-daemon.service).
set -euo pipefail

echo "==> node + git"
export DEBIAN_FRONTEND=noninteractive
apt-get install -y --no-install-recommends nodejs npm git ca-certificates fontconfig
# Debian ships npm 9.2, too old for CryptPad's overrides syntax — take npm 10
npm install -g npm@10 >/dev/null

echo "==> clone"
if [[ ! -d /opt/cryptpad ]]; then
  install -d -o ghost -g ghost /opt/cryptpad
  # TODO: pin to a release tag once we settle on one (plan: track releases only)
  su -s /bin/bash ghost -c "git clone --depth 1 https://github.com/cryptpad/cryptpad.git /opt/cryptpad"
fi

echo "==> deps (this takes a while)"
su -s /bin/bash ghost -c "cd /opt/cryptpad && /usr/local/bin/npm install --no-audit --no-fund && /usr/local/bin/npm run install:components"

echo "==> config (origins: 3000 main, 3001 sandbox via ghostd proxy)"
su -s /bin/bash ghost -c "
cd /opt/cryptpad
cp -n config/config.example.js config/config.js
sed -i \"s|// httpSafeOrigin:.*|httpSafeOrigin: 'http://localhost:3001',|\" config/config.js
sed -i \"s|^    //httpAddress:.*|    httpAddress: '127.0.0.1',|\" config/config.js
sed -i \"s|^    //httpPort: 3000,|    httpPort: 3000,|\" config/config.js
"

systemctl daemon-reload
echo "==> done — the Office app starts CryptPad on demand"
