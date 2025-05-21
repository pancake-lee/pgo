# 命令行相关

## 随手记

- 初始化/编译/运行
  - go run .\main.go
  - go mod init pgo
  - go mod tidy
  - go build
  - .\pgo.exe

- make on windows
  - `https://gnuwin32.sourceforge.net/packages/make.htm`
  - 下载：Complete package, except sources
  - 安装
  - 设置环境变量如： C:\Program Files (x86)\GnuWin32\bin 到 path
  - 要重启 vscode 以应用新环境变量。是指所有 vscode 窗口。

- make api
  - 这个命令运行没问题，因为加了 --proto_path=./third_party
  - vscode 提示错误，需要在配置.vscode/settings.json 中添加

  ```json
  "protoc": {
      "options": [
          "--proto_path=./third_party",
      ]
  }
  ```

- git

  ```shell
  git config --global credential.helper store
  git config --global user.name xxx
  git config --global user.email xxx

  # 哪些分支还没有完全合并到release分支
  git branch -r --no-merged=release

  # b中有哪些提交未合并到a
  git cherry -v a b

  git commit --amend
  ```
