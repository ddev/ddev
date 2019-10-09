#!/usr/bin/env bash

backuptool=mariadb
if command -v xtrabackup >/dev/null; then backuptool="xtrabackup --datadir=/var/lib/mysql"; fi
echo $backuptool
