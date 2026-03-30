#!/bin/sh
set -e

/usr/sbin/sshd

# check bootCheck result
chmod +x /backend/bootCheck
if ! /backend/bootCheck; then
    echo "Boot check failed, exiting..."
    exit 1
fi

pm2 start /backend/pm2.config.js

tail -f /dev/null
