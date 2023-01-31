#!/usr/bin/env bash
set -e

ensure_supervisord.sh

sc restart app
