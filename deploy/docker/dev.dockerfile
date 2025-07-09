# docker build -f dev.dockerfile -t dr9:latest .

FROM rockylinux:9.2

RUN echo 'alias ll="ls -la"' >> /etc/bashrc \
    && echo 'export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/lib/:/usr/lib64/:/usr/local/lib/:/usr/local/lib64/' >> /etc/bashrc \
    && echo 'export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/lib/:/usr/lib64/:/usr/local/lib/:/usr/local/lib64/' >> /etc/profile \
    \
    && echo "dnf install with epel-release" \
    && dnf install -y epel-release \
    && dnf config-manager --set-enabled crb \
    && dnf install -y --nogpgcheck https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm \
    && dnf install -y --nogpgcheck https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm

RUN dnf install -y python3-pip \
    && dnf install -y nodejs npm \
    && npm config set registry https://registry.npmmirror.com \
    && npm install -g pm2@4.5

RUN echo 'set up ssh' \
    && dnf install -y openssh-server \
    && echo 'docker setup sshd' \
    && ssh-keygen -A \
    && echo 'root:root' | chpasswd \
    && echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config \
    && echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config

RUN echo "dnf install for developer" \
    && dnf install -y dmidecode nginx wget \
    && dnf install -y procps iputils net-tools vim mysql tar git

RUN dnf remove golang go \
    && wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz -O go.tar.gz \
    && tar zxvf go.tar.gz -C /usr/local/ \
    && rm -rf go.tar.gz \
    && echo "export PATH=$PATH:/usr/local/go/bin" >> /etc/profile \
    && echo "export PATH=$PATH:/usr/local/go/bin" >> /etc/bashrc \
    && export PATH=$PATH:/usr/local/go/bin \
    && go env -w GOPROXY=https://goproxy.cn,direct

RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v30.2/protoc-30.2-linux-x86_64.zip -O protoc.zip \
    && unzip protoc.zip -d /usr/local/ \
    && rm -rf protoc.zip
