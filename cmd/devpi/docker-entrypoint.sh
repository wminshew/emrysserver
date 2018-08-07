#!bin/bash
# inspiration: https://github.com/muccg/docker-devpi/blob/master/docker-entrypoint.sh

function defaults {
  : ${DEVPI_SERVERDIR="/devpi/server"}
  export DEVPI_SERVERDIR
}

defaults

if [ "$1" = 'devpi' ]; then
  if [ ! -f $DEVPI_SERVERDIR/.serverversion  ]; then
    /usr/bin/devpi-server --init
  fi

  exec /usr/bin/devpi-server --host 0.0.0.0 --port 3141
fi

exec "$@"
