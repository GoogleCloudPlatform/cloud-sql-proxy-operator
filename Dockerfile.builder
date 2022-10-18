FROM gcr.io/cloud-builders/docker

##
# Install build tools
RUN apt-get update && \
    apt-get install -y git build-essential curl unzip

##
# Install Go
RUN curl -L -o /tmp/go.tar.gz  https://go.dev/dl/go1.18.7.linux-amd64.tar.gz &&\
    mkdir -p /usr/local &&\
    tar -C /usr/local -xzf /tmp/go.tar.gz &&\
    rm -rf /tmp/go.tar.gz

##
# Install Google Cloud CLI
RUN curl -L -o /tmp/gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-405.0.0-linux-x86_64.tar.gz && \
    mkdir -p /usr/local &&\
    tar -C /usr/local -xzf /tmp/gcloud.tar.gz &&\
    rm -rf /tmp/gcloud.tar.gz

RUN mkdir -p /tools
ENV PATH=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/google-cloud-sdk/bin:/usr/local/go/bin:/tools/bin

##
# Cache project go dependencies
WORKDIR /tools
COPY go.mod go.sum ./
RUN go mod graph | awk '{if ($1 !~ "@") print $2}' | xargs go get

##
# Download tools needed during the build
COPY Makefile build.sample.env ./
RUN make download_tools

ENTRYPOINT /bin/bash

