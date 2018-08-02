FROM nvidia/cuda:9.0-base-ubuntu16.04@sha256:ba1c9865dcafe8af90e60869f94acda6ca6b74981d59b63cff842be284ab2aed as base
LABEL maintainer="William Minshew <w@emrys.io>"

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    cuda-command-line-tools-9-0 \
    cuda-cublas-9-0 \
    cuda-cufft-9-0 \
    cuda-curand-9-0 \
    cuda-cusolver-9-0 \
    cuda-cusparse-9-0 \
    libcudnn7=7.0.5.15-1+cuda9.0 \
    python3-pip \
    && \
    apt-get clean && \
    apt-get autoremove && \
rm -rf /var/lib/apt/lists/*

RUN pip3 install virtualenv

ARG HOME="/home/user"
RUN useradd --create-home --home-dir $HOME user \
&& chown -R user:user $HOME
WORKDIR $HOME
RUN virtualenv venv


FROM base
ARG DEVPI_HOST="http://127.0.0.1:3141"
ARG REQS
COPY $REQS $REQS
RUN ./venv/bin/pip --no-cache-dir install -i $DEVPI_HOST/root/pypi/+simple/ -r $REQS
USER user

ARG MAIN
ENV MAIN $MAIN
COPY --chown=user:user $MAIN $MAIN
CMD ./venv/bin/python $MAIN
