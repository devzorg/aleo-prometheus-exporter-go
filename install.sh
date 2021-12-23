#!/usr/bin/env bash

go build

sudo echo "[Unit]
Description=Aleo Prometheus Exporter
After=network-online.target
StartLimitIntervalSec=0
[Service]
Type=simple
User=$USER
ExecStart=/usr/bin/aleo_exporter
Restart=always
RestartSec=3
LimitNOFILE=10000
[Install]
WantedBy=multi-user.target
EnvironmentFile=/etc/sysconfig/aleo_exporter
" > /etc/systemd/system/aleo-prometheus-exporter.service
sudo cp aleo_exporter /usr/bin/aleo_exporter
sudo systemctl daemon-reload
sudo systemctl enable aleo-prometheus-exporter
sudo systemctl start aleo-prometheus-exporter
