# PGO 项目专用运行时环境
# 
# 构建命令:
#   docker build -f dev.dockerfile -t pgo:latest .
#
# 说明：本镜像仅包含 PGO 运行所需的最小依赖

# --------------------------------------------------
# Build arguments
# --------------------------------------------------
ARG GO_DL_URL="https://go.dev/dl"
ARG GO_VERSION=1.24.4
ARG PROTOC_VERSION=34.1

# --------------------------------------------------
# Stage 1: base with Go runtime
# --------------------------------------------------
FROM rockylinux:9.2 AS base

SHELL ["/bin/bash", "-lc"]

ARG GO_DL_URL
ARG GO_VERSION

RUN set -euxo pipefail \
    && dnf install -y --setopt=install_weak_deps=False --setopt=tsflags=nodocs \
        wget \
        unzip \
        tzdata \
    # --------------------------------------------------
    && wget ${GO_DL_URL}/go${GO_VERSION}.linux-amd64.tar.gz -O /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm -f /tmp/go.tar.gz \
    && mkdir -p /root/.config/go \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    # --------------------------------------------------
    && dnf remove -y wget || true \
    && dnf clean all \
    && rm -rf /var/cache/dnf /tmp/* /var/tmp/*

ENV PATH=/usr/local/go/bin:$PATH
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

# --------------------------------------------------
# Stage 2: add protoc for code generation (optional)
# --------------------------------------------------
FROM base AS with_protoc

ARG PROTOC_VERSION

RUN set -euxo pipefail \
    && wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip -O /tmp/protoc.zip \
    && unzip /tmp/protoc.zip -d /usr/local \
    && rm -f /tmp/protoc.zip \
    && dnf clean all \
    && rm -rf /var/cache/dnf /tmp/* /var/tmp/*

ENV PATH=/usr/local/bin:$PATH

# --------------------------------------------------
# Stage 3: build PGO application
# --------------------------------------------------
FROM with_protoc AS builder

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build main binary
RUN set -euxo pipefail \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
       go build -ldflags="-s -w" -o /build/pgo ./cmd/pgo

# --------------------------------------------------
# Stage 4: minimal runtime image
# --------------------------------------------------
FROM rockylinux:9.2-minimal AS runtime

RUN set -euxo pipefail \
    && dnf install -y --setopt=install_weak_deps=False --setopt=tsflags=nodocs \
        tzdata \
    && dnf clean all \
    && rm -rf /var/cache/dnf /tmp/* /var/tmp/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/pgo /app/pgo

# Set timezone
ENV TZ=Asia/Shanghai

# Expose ports (if needed)
EXPOSE 8080 9000

# Default command
ENTRYPOINT ["/app/pgo"]
CMD ["--help"]
