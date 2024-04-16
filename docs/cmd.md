# 命令行相关

## 随手记
* 初始化/编译/运行
    * go run .\main.go
    * go mod init gogogo
    * go mod tidy
    * go build
    * .\gogogo.exe

- make
    - https://gnuwin32.sourceforge.net/packages/make.htm
    - 下载：Complete package, except sources
    - 安装
    - 设置环境变量如： C:\Program Files (x86)\GnuWin32\bin 到 path
    - 要重启 vscode 以应用新环境变量。是指所有 vscode 窗口。

- make api 
    - 这个命令运行没问题，因为加了 --proto_path=./third_party
    - vscode 提示错误，需要在配置中添加
    ```json
    "protoc": {
        "options": [
            "--proto_path=./third_party",
        ]
    }
    ```