#!/bin/bash
STRINGI_DISABLE_PKG_CONFIG=1 GITLAB_TOKEN="" make build && ./dist/scribe_linux_amd64_v1/scribe

echo "=================$(date)================="
