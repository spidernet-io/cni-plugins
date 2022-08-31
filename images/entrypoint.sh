#!/bin/sh

function log()
{
    echo "$(date +%Y-%m-%d" "%H:%M:%S) ${1}"
}

HOST_BIN_PATH=/host/opt/cni/bin

cd /home/cnibin
ALL_BIN=`ls ./ `
for BIN in $ALL_BIN ; do
  log "install $BIN"
  if [ -f "${HOST_BIN_PATH}/${BIN}" ] ; then
    rm -f ${HOST_BIN_PATH}/${BIN}.old
    mv ${HOST_BIN_PATH}/${BIN}  ${HOST_BIN_PATH}/${BIN}.old
  fi
  cp $BIN ${HOST_BIN_PATH}/${BIN}
  # try to remove if existed, ignore failure if file busy
  rm -f ${HOST_BIN_PATH}/${BIN}.old &>/dev/null
done

ls /host/opt/cni/bin

log "Entering sleep (success)..."
sleep infinity
