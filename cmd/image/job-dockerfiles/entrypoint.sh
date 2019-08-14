#!/bin/bash
set -e

CONDA_BASE=$(conda info --base)
source $CONDA_BASE/etc/profile.d/conda.sh

if [ -f "$CONDA_ENV" ]; then
  CONDA_ENV_NAME=$(sed -n 's/name: //pI' "$CONDA_ENV")
  conda env create -q -f "$CONDA_ENV"
else
  CONDA_ENV_NAME=user
  conda create -q -y -n $CONDA_ENV_NAME 1>/dev/null
fi
conda activate $CONDA_ENV_NAME

if [ -f "$PIP_REQS" ]; then
  if [[ -z $(conda list "^pip" | grep "^pip") ]]; then
    conda install -q -y -n $CONDA_ENV_NAME pip 1>/dev/null
  fi
  pip --no-cache-dir --timeout=30 --retries=10 install --progress-bar off -r "$PIP_REQS"
fi

if [ "$NOTEBOOK" = 'true' ]; then
  if [[ -z $(command -v jupyter) ]]; then
    conda install -q -y -n $CONDA_ENV_NAME jupyter 1>/dev/null
  fi

  exec jupyter notebook --ip=0.0.0.0 --no-browser --port=8888 --NotebookApp.custom_display_url=http://127.0.0.1:8888
fi

exec python "$MAIN"
