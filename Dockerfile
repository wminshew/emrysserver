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
