#!bin/bash
# inspiration: https://github.com/muccg/docker-devpi/blob/master/docker-entrypoint.sh

if [ "$1" = 'devpi' ]; then
  if [ ! -f ${DEVPISERVER_SERVERDIR:-"/devpi/server"}/.serverversion  ]; then
    /usr/bin/devpi-server --init
  fi

  exec /usr/bin/devpi-server --replica-max-retries=${DEVPISERVER_MAX_RETRIES:-3} --host 0.0.0.0 --port 3141 ${DEVPISERVER_DEBUG:+"--debug"} 2>&1
fi

exec "$@"
