#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.." 
buf lint
buf generate
echo "✅ proto generated into gen/go"
