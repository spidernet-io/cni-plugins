#!/bin/sh

function log()
{
    echo "$(date --iso-8601=seconds) ${1}"
}

cp /home/* /host/opt/cni/bin/
ls /host/opt/cni/bin

log "Entering sleep (success)..."
sleep infinity
