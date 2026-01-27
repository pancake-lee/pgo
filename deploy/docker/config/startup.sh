#!/bin/sh
set -e

/usr/sbin/sshd

# check bootCheck result
chmod +x /backend/bootCheck
if ! /backend/bootCheck; then
    echo "Boot check failed, exiting..."
    exit 1
fi

# TODO 临时：build新的镜像应该包含进去
pm2 install pm2-prom-module

pm2 start /backend/pm2.config.js

tail -f /dev/null
