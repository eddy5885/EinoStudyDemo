#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

export GODEBUG=x509negativeserial=1
export http_proxy=http://127.0.0.1:8899
export https_proxy=http://127.0.0.1:8899

exec go run .
