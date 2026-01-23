#!/bin/sh
set -e

/usr/sbin/sshd

pm2 start /backend/pm2.config.js

tail -f /dev/null
