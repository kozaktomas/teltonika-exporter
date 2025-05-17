#!/bin/sh

groupadd --system teltonika-exporter || true
useradd --system -d /nonexistent -s /usr/sbin/nologin -g teltonika-exporter teltonika-exporter || true

chown teltonika-exporter /etc/teltonika-exporter/config.yaml

mkdir /var/lib/teltonika-exporter
chown teltonika-exporter:teltonika-exporter /var/lib/teltonika-exporter

systemctl daemon-reload
systemctl enable teltonika-exporter
systemctl restart teltonika-exporter