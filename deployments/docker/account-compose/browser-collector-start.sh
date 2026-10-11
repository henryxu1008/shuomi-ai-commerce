#!/bin/sh
set -eu
export A1688_BROWSER_COLLECTOR_CREDENTIAL="$(cat /collector-private/credential)"
exec /usr/local/bin/collector -listen 0.0.0.0:19545 -browser /usr/local/bin/collector-chromium
