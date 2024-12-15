# 本地构建的用于开发环境的镜像
# docker build -f dev.dockerfile -t dr9:latest .

FROM rockylinux:9.2

COPY ./thirdparty/go1.23.4.linux-amd64.tar.gz /root/go1.23.4.linux-amd64.tar.gz

RUN echo 'alias ll="ls -la"' >> /etc/bashrc \
    && echo 'export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/lib/:/usr/lib64/:/usr/local/lib/:/usr/local/lib64/' >> /etc/profile \
    && echo 'export PATH=$PATH:/usr/local/go/bin/:/root/go/bin/' >> /etc/profile \
    && echo 'export PATH=$PATH:/usr/local/go/bin/:/root/go/bin/' >> /etc/bashrc \
    && echo "------------------------------------------------------------" \
    && echo 'dnf install' \
    && dnf install -y epel-release \
    && dnf config-manager --set-enabled crb \
    && dnf install -y --nogpgcheck https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm \
    && dnf install -y --nogpgcheck https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm \
    && dnf install -y dmidecode vim openssh-server procps nginx iputils wget unzip \
    && echo "------------------------------------------------------------" \
    && echo 'develop tools' \
    && dnf install -y git make postgresql \
    && git config --global credential.helper store \
    && tar zxvf /root/go1.23.4.linux-amd64.tar.gz -C /usr/local/ \
    && rm -f /root/go1.23.4.linux-amd64.tar.gz \
    && echo "------------------------------------------------------------" \
    && echo 'setup sshd' \
    && ssh-keygen -A \
    && echo 'root:root' | chpasswd \
    && echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config \
    && echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config \
    && echo "------------------------------------------------------------" \
    && echo 'clear cache' \
    && dnf clean all \
    && rm -rf /var/cache/dnf \
    && echo "------------------------------------------------------------" \
    && echo 'docker build done'

# 设置启动命令
# ENTRYPOINT ["/app/myapp"]
