#!/usr/bin/env bash
set -e

chroot $HOSTFS docker $@
