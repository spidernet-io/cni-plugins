#!/bin/sh

function log()
{
    echo "$(date +%Y-%m-%d" "%H:%M:%S) ${1}"
}

cp /home/* /host/opt/cni/bin/
ls /host/opt/cni/bin

log "Entering sleep (success)..."
sleep infinity
