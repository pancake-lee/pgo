FROM alpine:latest
MAINTAINER zhufuyi "g.zhufuyi@gmail.com"

# set the time zone to Shanghai
RUN apk add tzdata  \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

# add curl, used for http service checking, can be installed without it if deployed in k8s
RUN apk add curl

COPY configs/ /app/configs/
COPY service_user /app/service_user
RUN chmod +x /app/service_user

# http port
EXPOSE 8080


WORKDIR /app

CMD ["./service_user", "-c", "configs/service_user.yml"]
# if you use the Configuration Center, service_user.yml is changed to the Configuration Center configuration.
#CMD ["./service_user", "-c", "configs/service_user.yml", "-enable-cc"]
