#!/bin/sh

if [ "$1" != "remove" ]; then
	exit 0
fi

rm -rf /var/lib/teltonika-exporter
systemctl daemon-reload
userdel  teltonika-exporter || true
groupdel teltonika-exporter 2>/dev/null || true