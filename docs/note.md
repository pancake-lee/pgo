# 技术备忘

> 活跃技术备忘：当前在用的实现细节、问题修复记录、常用命令等。

## [commitlint](`https://github.com/conventional-changelog/commitlint`)

| prefix   | desc       |
| -------- | ---------- |
| build    | 构建相关   |
| chore    | 杂项       |
| ci       | CI/CD 相关 |
| docs     | 文档       |
| feat     | 功能       |
| fix      | 修复       |
| perf     | 性能       |
| refactor | 重构       |
| revert   | 回退       |
| style    | 代码风格   |
| test     | 测试       |
| gen      | 生成代码   |
| improve  | 优化代码   |
| tidy     | 整理、清理 |
| bak      | 备份       |

## 命令行备忘

### 初始化/编译/运行

```shell
go run .\main.go
go mod init pgo
go mod tidy
go build
.\pgo.exe
```

### Windows 上使用 make

- 下载 [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)：Complete package, except sources
- 安装后设置环境变量，如 `C:\Program Files (x86)\GnuWin32\bin` 到 PATH
- 重启 vscode 以应用新环境变量（所有 vscode 窗口）

### Windows 客户端运行模式

`make cli-win` 生成 GUI 子系统程序，无参数启动界面，`pgo.exe cli` 启动交互菜单，其他参数执行对应命令。CLI 分支附着父控制台，父进程无控制台时创建新控制台，以读写权限打开控制台设备，并恢复输入输出。随后启动继承有效标准句柄的 CLI 子进程并等待退出，让键盘库初始化时取得正确输入；有效文件重定向会保留。

如果终端在 GUI 程序启动后立即返回提示符，可显式等待，避免终端和交互菜单同时读取输入。PowerShell 使用：

```powershell
Start-Process -FilePath .\pgo.exe -ArgumentList cli -NoNewWindow -Wait
```

CMD 使用 `start "" /wait pgo.exe cli`。等待参数见 [Start-Process](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/start-process) 和 [start](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/start)。

### make api

- 命令运行没问题（加了 `--proto_path=./third_party`）
- vscode 提示错误时，在 `.vscode/settings.json` 中添加：

```json
"protoc": {
    "options": [
        "--proto_path=./third_party",
    ]
}
```

### Git 常用

```shell
git config --global credential.helper store
git config --global user.name xxx
git config --global user.email xxx

# 哪些分支还没有完全合并到 release 分支
git branch -r --no-merged=release

# b 中有哪些提交未合并到 a
git cherry -v a b

git commit --amend
```

## 调试备忘

### 持续 profiling 采集方式

- 2026-10-01 决定由 Grafana Alloy 拉取应用 pprof 并写入 Pyroscope，不在应用内集成 `pyroscope-go` 推送 SDK。这样 CPU、heap 与日志采集统一放在部署层，应用只维护标准且受控的诊断端点。
- performance 只制造负载并记录 Vegeta 的请求数、吞吐、成功率和延迟分位数；服务指标与 profiling 直接在 Grafana、Pyroscope 中按负载时间范围查看，不再导出到本地压测产物。

### 权限压测结果保留策略

- 2026-10-04 用户明确删除 repeat 参数与 round 层级，每次只执行一次负载流程。
- 采用方案 A，同目录测试清除旧负载报告，准备数据跨 RPS 复用；不增加手动运行历史目录，效果按监控时间窗口比较。

- 2026-10-04 用户澄清原需求为固定 mixed，删除自动 pure/mixed/recovery 编排；交互明确说明读写同时运行并提供管理员写入 RPS，保留热点组与对照组。

- 2026-10-04 权限测试直接实现读写并行方案，交互、日志和当前设计使用行为描述，报告按读组和管理员输出。

### vscode debug

- 没有配置的情况下直接 debug，将调试当前正在编辑的文件
- 简单配置 `.vscode/launch.json`：

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Current File",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${fileDirname}",
      "args": ["-port=8080"]
    }
  ]
}
```
