# --------------------------------------------------
# 全局参数定义
# --------------------------------------------------
ARG GO_DL_URL="https://go.dev/dl/"
ARG GO_VERSION=1.24.4
ARG NODE_VERSION=16.15.0
ARG FileServer="http://127.0.0.1:9000/download/"
ARG PROTOC_VERSION=30.2

# --------------------------------------------------
# 第一阶段：基础环境与维护工具 (Base)
# 功能：提供OS基础、SSH服务、以及生产和开发都需要的日常运维工具
# --------------------------------------------------
FROM rockylinux:9.2 AS base

# 配置环境变量
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/lib/:/usr/lib64/:/usr/local/lib/:/usr/local/lib64/:/usr/local/samba/

# 安装基础仓库和运维工具 (合并指令以减少层数)
RUN echo "Setup Repositories and Install Maintenance Tools" \
    && echo 'alias ll="ls -la"' >> /etc/bashrc \
    && dnf install -y epel-release \
    && dnf config-manager --set-enabled crb \
    && dnf install -y --nogpgcheck https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm \
    && dnf install -y --nogpgcheck https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm \
    && dnf install -y \
        dmidecode nginx wget \
        procps iputils net-tools vim tar xz zip \
        openssh-server \
        ImageMagick \
    # SSH 配置
    && ssh-keygen -A \
    && echo 'root:root' | chpasswd \
    && echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config \
    && echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config \
    # 清理缓存
    && dnf clean all && rm -rf /var/cache/dnf

# --------------------------------------------------
# 第二阶段：开发环境
# --------------------------------------------------
FROM base AS dev
# 记得声明一下
ARG FileServer
ARG GO_DL_URL 
ARG GO_VERSION
ARG NODE_VERSION
ARG PROTOC_VERSION

ENV NVM_DIR=/root/.nvm
# 应该使用nvm管理node版本，所以不应该直接设置某个版本的路径
# ENV NODE_PATH=$NVM_DIR/versions/node/v${NODE_VERSION}/lib/node_modules \
#     PATH=$NVM_DIR/versions/node/v${NODE_VERSION}/bin:$PATH

# 安装 各种语言开发环境
RUN echo "Installing development and runtime environments" \
    && dnf install -y make git mysql\
    # --------------------------------------------------
    && dnf remove -yq golang go \
    && wget ${GO_DL_URL}/go${GO_VERSION}.linux-amd64.tar.gz -O go.tar.gz \
    && tar zxvf go.tar.gz -C /usr/local/ \
    && rm -rf go.tar.gz \
    && echo "export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin" >> /etc/profile \
    && echo "export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin" >> /etc/bashrc \
    && export PATH=$PATH:/usr/local/go/bin \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    # --------------------------------------------------
    && dnf install -y python3-pip \
    && pip install usd-core -i https://pypi.tuna.tsinghua.edu.cn/simple/ \
    && rm -rf /root/.cache/pip/ \
    # --------------------------------------------------
    && git clone https://gitee.com/mirrors/nvm.git $NVM_DIR \
    && cd $NVM_DIR \
    && git checkout v0.39.5 \
    && . $NVM_DIR/nvm.sh \
    && echo '[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"' >> /etc/bashrc \
    && export NVM_NODEJS_ORG_MIRROR=https://npmmirror.com/mirrors/node \
    && nvm install ${NODE_VERSION} \
    && npm config set registry https://registry.npmmirror.com \
    && npm install -g pm2@4.5 yarn pnpm \
    && pm2 install pm2-prom-module \

# --------------------------------------------------
# 第三阶段：额外的开发运行环境扩展
# --------------------------------------------------
FROM dev AS dev_ex
# 记得声明一下
ARG FileServer
ARG PROTOC_VERSION
# --------------------------------------------------
# 一些运行库和工具
RUN echo "Installing Runtime Binaries" \
    && dnf install -y python3-gpg libtasn1 libxslt libcom_err jansson-devel \
    # --------------------------------------------------
    && wget ${FileServer}/ffmpeg -O /usr/bin/ffmpeg \
    && chmod +x /usr/bin/ffmpeg \
    && wget ${FileServer}/ffprobe -O /usr/bin/ffprobe \
    && chmod +x /usr/bin/ffprobe \
    # --------------------------------------------------
    && wget ${FileServer}/minio-20211229064906.0.0.x86_64.rpm -O /tmp/minio.rpm \
    && dnf install -y /tmp/minio.rpm \
    && rm -f /tmp/minio.rpm \
    # --------------------------------------------------
    && wget ${FileServer}/hiredis.tar.gz -O /tmp/hiredis.tar.gz \
    && tar zxvf /tmp/hiredis.tar.gz -C /usr/local/ \
    && rm -f /tmp/hiredis.tar.gz \
    # --------------------------------------------------
    && wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip -O protoc.zip \
    && unzip protoc.zip -d /usr/local/ \
    && rm -rf protoc.zip
