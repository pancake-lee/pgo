# Changelog

## [0.0.3] - 20240412 
### Added
- 初始化kratos官方模板工程
    - kratos new gogogo
        - 这命令还挺烦，会在当前目录创建gogogo，那我就不能在当前项目根目录执行。在上一层执行又会完全覆盖gogogo文件夹，而且是先删除再重建那种，连.git都没有了。
        - 所以在别的地方执行命令后，再把文件拷贝过来了。可能我打开方式不对吧。
    - go mod tidy
    - kratos run
    - 访问 http://127.0.0.1:8000/helloworld/pancake

### Removed
- 之前的 hello 程序放demo里了

## [0.0.2] - 20240411 
### Added
- hello 程序，提供为 http 服务，可通过3种方法进行测试
    - go test 通过 httptest 直接测试 服务处理函数
    - 先运行 http 服务程序，再运行 go test 通过真实的 http get 请求进行测试
    - 浏览器直接访问 http://127.0.0.1:8080/hello?user=pancake

## [0.0.1] - 20240411 
### Added
- hello 程序

## [0.0.0] - 20240411 
- 示例

### Added
- 新增了功能A

### Fixed
- 修复了错误B

### Removed
- 移除了功能Y

### Improved
- 改进了功能Z