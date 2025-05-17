#!/bin/sh

if [ "$1" != "remove" ]; then
	exit 0
fi

systemctl disable teltonika-exporter || true
systemctl stop teltonika-exporter    || true
