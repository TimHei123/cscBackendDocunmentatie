# ﻿#!/bin/bash

sudo systemctl start ovsdb-server
sudo systemctl enable ovsdb-server

STUDENTNR="$1"
STUDENTNAME="$2"
IP="$3"
PASSWORD="$4"
PROJECTNAME="$5"

useradd -m --shell /bin/bash "$STUDENTNAME"
usermod -aG sudo "$STUDENTNAME"

echo "$STUDENTNAME:$PASSWORD" | chpasswd

# sed -i "s/145.89.192.x/${IP}/g" /etc/netplan/*
# netplan apply


cat <<EOF > /etc/netplan/50-cloud-init.yaml
network:
    ethernets:
        ens18:
            addresses:
            - $IP/24
            nameservers:
                addresses:
                - 1.1.1.1
                - 1.0.0.1
                search: []
            routes:
            -   to: default
                via: 145.89.192.1
    version: 2
EOF

netplan apply

hostnamectl set-hostname "OICT-AUTO-$STUDENTNR-$PROJECTNAME"

echo "ubuntu:Operator2022!" | chpasswd

# rm /home/ubuntu/startup.sh