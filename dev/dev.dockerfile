# docker build -f dev.dockerfile -t dr9:2.0 .
# docker save dr9:2.0 -o dr9-2.0.tar
# 
# --------------------------------------------------
# Global build arguments
# --------------------------------------------------
ARG GO_DL_URL="https://go.dev/dl"
ARG GO_VERSION=1.24.4
ARG NODE_VERSION=22.22.1
ARG PROTOC_VERSION=34.1
ARG NVM_VERSION=v0.39.5

# --------------------------------------------------
# Stage 1: base OS and ops tools
# --------------------------------------------------
FROM rockylinux:9.2 AS base

SHELL ["/bin/bash", "-lc"]

RUN set -euxo pipefail \
    && dnf install -y --setopt=install_weak_deps=False --setopt=tsflags=nodocs dnf-plugins-core epel-release \
    && dnf config-manager --set-enabled crb \
    && dnf install -y --setopt=install_weak_deps=False --setopt=tsflags=nodocs --nogpgcheck \
        https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm \
        https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm \
    && dnf config-manager --add-repo https://pkgs.tailscale.com/stable/rhel/9/tailscale.repo \
    && dnf install -y --setopt=install_weak_deps=False --setopt=tsflags=nodocs \
        dmidecode \
        nginx \
        wget \
        procps-ng \
        iputils \
        net-tools \
        vim-enhanced \
        tar \
        xz \
        zip \
        unzip \
        openssh-server \
        ImageMagick \
        ffmpeg \
        redhat-rpm-config \
        psmisc \
        telnet \
        iotop-c \
        tailscale \
    && ssh-keygen -A \
    && echo 'root:root' | chpasswd \
    && dnf clean all \
    && rm -rf /var/cache/dnf /var/tmp/* /tmp/* /var/log/dnf* /var/log/yum.*

# --------------------------------------------------
# Stage 2: development runtime
# --------------------------------------------------
FROM base AS dev

ARG GO_DL_URL
ARG GO_VERSION
ARG NODE_VERSION
ARG NVM_VERSION

SHELL ["/bin/bash", "-lc"]

ENV NVM_DIR=/root/.nvm \
    NVM_NODEJS_ORG_MIRROR=https://npmmirror.com/mirrors/node \
    PATH=/root/.nvm/versions/node/v${NODE_VERSION}/bin:/usr/local/go/bin:/root/go/bin:${PATH}

RUN set -euxo pipefail \
    && dnf install -y --setopt=install_weak_deps=False --setopt=tsflags=nodocs \
        make \
        git \
        mysql \
        python3 \
        python3-pip \
        gcc \
        gcc-c++ \
        cmake \
        pkgconf-pkg-config \
    # --------------------------------------------------
    && (dnf remove -y golang go || true) \
    && wget ${GO_DL_URL}/go${GO_VERSION}.linux-amd64.tar.gz -O /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm -f /tmp/go.tar.gz \
    && mkdir -p /root/.config/go \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    # --------------------------------------------------
    && pip3 config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple \
    && pip3 install --no-cache-dir --disable-pip-version-check usd-core \
    # --------------------------------------------------
    && git clone --branch ${NVM_VERSION} --depth 1 https://gitee.com/mirrors/nvm.git ${NVM_DIR} \
    && rm -rf ${NVM_DIR}/.git \
    && . ${NVM_DIR}/nvm.sh \
    && nvm install ${NODE_VERSION} \
    && nvm alias default ${NODE_VERSION} \
    && nvm use default \
    && npm config set registry https://registry.npmmirror.com \
    && corepack enable \
    && npm install -g npm@10.9.4 pm2@4.5.6 \
    && pm2 install pm2-prom-module \
    && npm cache clean --force \
    # --------------------------------------------------
    && dnf clean all \
    && rm -rf /var/cache/dnf /root/.cache /root/.npm /root/.local/share/pnpm/store /tmp/* /var/tmp/* /var/log/dnf* /var/log/yum.*

# --------------------------------------------------
# Stage 3: extra runtime binaries
# --------------------------------------------------
FROM dev AS dev_ex

ARG PROTOC_VERSION

SHELL ["/bin/bash", "-lc"]

RUN set -euxo pipefail \
    && wget https://github.com/redis/hiredis/archive/refs/tags/v1.3.0.tar.gz -O /tmp/hiredis.tar.gz \
    && tar -C /usr/local -xzf /tmp/hiredis.tar.gz \
    && rm -f /tmp/hiredis.tar.gz \
    && wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip -O /tmp/protoc.zip \
    && unzip /tmp/protoc.zip -d /usr/local \
    && rm -f /tmp/protoc.zip \
    && rm -rf /tmp/* /var/tmp/*
