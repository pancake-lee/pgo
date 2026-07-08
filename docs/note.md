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
