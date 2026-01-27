# Grafana预设配置

[Provisioning功能](https://grafana.com/docs/grafana/latest/administration/provisioning/)

入口是yml配置文件，而不是grafana web上导出的json文件

其中datasource直接用yml配置完成了
使用容器内部网络，不需要安全验证等配置，只配置url即可

dashboard首先需要yml配置文件，然后设置options.path为json文件位置

- host.json基于1860，只改了名字
- docker.json基于14282，只改了名字
- pm2.json自研，数据上报使用pm2的[pm2-prom-module]模块
