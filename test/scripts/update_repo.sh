#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

source_list="deb http://mirrors.aliyun.com/ubuntu/ bionic main restricted universe multiverse \n
deb http://mirrors.aliyun.com/ubuntu/ bionic-security main restricted universe multiverse \n
deb http://mirrors.aliyun.com/ubuntu/ bionic-updates main restricted universe multiverse \n
deb http://mirrors.aliyun.com/ubuntu/ bionic-proposed main restricted universe multiverse \n
deb http://mirrors.aliyun.com/ubuntu/ bionic-backports main restricted universe multiverse \n
deb-src http://mirrors.aliyun.com/ubuntu/ bionic main restricted universe multiverse \n
deb-src http://mirrors.aliyun.com/ubuntu/ bionic-security main restricted universe multiverse \n
deb-src http://mirrors.aliyun.com/ubuntu/ bionic-updates main restricted universe multiverse \n
deb-src http://mirrors.aliyun.com/ubuntu/ bionic-proposed main restricted universe multiverse \n
deb-src http://mirrors.aliyun.com/ubuntu/ bionic-backports main restricted universe multiverse"

echo -e $source_list >> /etc/apt/source_list
apt-get update
apt-get install -y inetutils-ping
apt-get install -y vim
apt-get install -y net-tools
apt-get install -y tcpdump
