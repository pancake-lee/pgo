# 调试相关

## vscode debug
* 没有配置的情况下直接使用debug功能，将直接调试当前正在编辑的文件
* 简单的配置.vscode/launch.json
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
