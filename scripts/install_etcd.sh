#!/bin/sh
ETCD_VER=v3.5.18

GITHUB_URL=https://github.com/etcd-io/etcd/releases/download
DOWNLOAD_URL=${GITHUB_URL}

# LINUX
# curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o etcd-${ETCD_VER}-linux-amd64.tar.gz
# tar xzvf etcd-${ETCD_VER}-linux-amd64.tar.gz

# macOS M1
curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-darwin-arm64.zip -o etcd-${ETCD_VER}-darwin-arm64.zip
unzip etcd-${ETCD_VER}-darwin-arm64.zip
