#!/bin/bash
set -e

if [ "$NOTEBOOK" = 'true' ]; then
  if [ "$HAS_MAIN"  ]; then
    exec jupyter notebook --ip=0.0.0.0 --no-browser "$MAIN"
  fi

  rm -f "$MAIN"
  exec jupyter notebook --ip=0.0.0.0 --no-browser
fi

exec python3 "$MAIN"
