# Changelog

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