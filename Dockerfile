# Copyright © 2021-2023 The Go Service Components Authors. All rights reserved. Issued under the Apache 2.0 License.
FROM ubuntu:22.04

LABEL maintainer="karlmutch@gmail.com"

ENV LANG C.UTF-8

RUN \
    apt-get -y update

RUN \
    apt-get install -y language-pack-en && \
    update-locale "en_US.UTF-8" && \
    apt-get -y install git software-properties-common wget openssl ssh curl jq apt-utils source-highlight unzip && \
    apt-get clean && \
    apt-get autoremove

RUN \
    mkdir -p /usr/local/bin && \
    wget -O /usr/local/bin/semver https://github.com/karlmutch/duat/releases/download/0.17.0-rc.8/semver_0.17.0-rc.8_linux_amd64 && \
    wget -O /usr/local/bin/stencil https://github.com/karlmutch/duat/releases/download/0.17.0-rc.8/stencil_0.17.0-rc.8_linux_amd64 && \
    wget -O /usr/local/bin/github-release https://github.com/karlmutch/duat/releases/download/0.17.0-rc.8/github-release_0.17.0-rc.8_linux_amd64 && \
    chmod +x /usr/local/bin/semver && \
    chmod +x /usr/local/bin/stencil && \
    chmod +x /usr/local/bin/github-release

ENV GO_VERSION 1.21.1

ENV USER {{.duat.userName}}
ENV USER_ID {{.duat.userID}}
ENV USER_GROUP_ID {{.duat.userGroupID}}
ENV BUILD_LOG {{ env "BUILD_LOG" | default "build.log" }}

RUN groupadd -f -g ${USER_GROUP_ID} $USER} && \
    useradd -g ${USER_GROUP_ID} -u ${USER_ID} -ms /bin/bash ${USER}

USER ${USER}
WORKDIR /home/${USER}

ENV GOPATH=/project
ENV PATH=$GOPATH/bin:$PATH
ENV PATH=$PATH:/home/${USER}/.local/bin:/home/${USER}/go/bin
ENV GOROOT=/home/${USER}/go
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:

HEALTHCHECK NONE

RUN \
    mkdir -p /home/${USER}/go && \
    wget -O /tmp/go.tgz https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar xzf /tmp/go.tgz && \
    rm /tmp/go.tgz

CMD cd /project/src/github.com/karlmutch/go-service && \
    go install github.com/alvaroloes/enumer@455e9a94796c0e108c38e253b67307736fc4b200 && \
    go test -ldflags="-extldflags=-static" -tags="osusergo netgo" -v ./internal/test/...

# Done last to prevent lots of disruption when bumping versions
LABEL vendor="The Go Service Components authors" \
      ml.studio.module.version={{.duat.version}} \
      ml.studio.module.name={{.duat.module}}
