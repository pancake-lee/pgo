#!/bin/sh
set -e

/usr/sbin/sshd

/usr/sbin/tailscaled --state=/var/lib/tailscale/tailscaled.state

tail -f /dev/null
