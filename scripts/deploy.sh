#!/bin/bash
set -e

APP_DIR="/home/ubuntu/gugu-admin-api"
APP_NAME="gugu-admin-api"

cd "$APP_DIR"

# Replace binary
mv "${APP_NAME}.new" "$APP_NAME"
chmod +x "$APP_NAME"

# Restart via systemd
sudo systemctl restart gugu-admin-api

sleep 2

# Health check
if systemctl is-active --quiet gugu-admin-api; then
  echo "Deploy successful - gugu-admin-api is running"
else
  echo "Deploy failed - gugu-admin-api is not running"
  sudo journalctl -u gugu-admin-api --no-pager -n 20
  exit 1
fi
