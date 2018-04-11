# TODO: minimize size... alpine? multi-stage with scratch?
# TODO: add cuda/gpu support
# TODO: does access to tensorflow etc come from the image or venv?
FROM ubuntu:16.04
# TODO: switch to emrys email
MAINTAINER William Minshew <wminshew@gmail.com>

# TODO: should be able to select version of python to run; maybe
# should be handled by base image or multi-build...
# TODO: order packages by alphanumeric
RUN apt-get update; \
    apt-get install -y \
    python3-pip \
    ; \
    rm -rf /var/lib/apt/lists/*
RUN pip3 install virtualenv

# TODO: separate base-build-image from $USER arg-build-image
ARG USER
# TODO: think about how to partition this properly so multiple miners
# can benefit from a single build, i.e. if the user hasn't changed
# his requirements.txt but runs 8 emrys commands (or if 8 gpus are
# all working on the same command) they shouldn't have to replicate
# image build work
# TODO: data will likely have to be mounted (vs added?)
COPY user-upload/$USER $USER

WORKDIR $USER
RUN virtualenv venv && \
    venv/bin/pip3 install -r requirements.txt && \
    venv/bin/python train.py
