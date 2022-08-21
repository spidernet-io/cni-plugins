#!/bin/bash

function log()
{
    echo "$(date --iso-8601=seconds) ${1}"
}

log "Entering sleep (success)..."
sleep infinity

