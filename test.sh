#!/bin/bash
GITLAB_TOKEN="" make build && ./dist/scribe_linux_amd64_v1/scribe | tee "scribe_run_$(date +%Y%m%d%H%M%S).log"

echo "=================$(date)================="
