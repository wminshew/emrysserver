#!/bin/bash
set -e

CONDA_BASE=$(conda info --base)
source $CONDA_BASE/etc/profile.d/conda.sh

if [ -f "$CONDA_ENV" ]; then
  CONDA_ENV_NAME=`sed -n 's/name: //pI' "$CONDA_ENV"`
  conda env create -q -y -f "$CONDA_ENV"
else
  CONDA_ENV_NAME=user
  conda create -q -y -n $CONDA_ENV_NAME 1>/dev/null
fi
conda activate $CONDA_ENV_NAME

if [ -f "$PIP_REQS" ]; then
  if ! [ `conda list "^pip" | grep "^pip"` ]; then
    conda install -q -y -n $CONDA_ENV_NAME pip 1>/dev/null
  fi
  pip --no-cache-dir --timeout=30 --retries=10 install --progress-bar off -r "$PIP_REQS"
fi

if [ "$NOTEBOOK" = 'true' ]; then
  if ! [ `command -v jupyter` ]; then
    conda install -q -y -n $CONDA_ENV_NAME jupyter
  fi

  if [ -f "$MAIN" ]; then
    exec jupyter notebook --ip=0.0.0.0 --no-browser "$MAIN"
  fi

  exec jupyter notebook --ip=0.0.0.0 --no-browser
fi

exec python "$MAIN"
