# 已拒绝的需求/方案

> 明确放弃的事项记录，防止未来重复提出。包含拒绝理由和重新考虑的触发条件。

## Sponge 代码生成工具

- **提出时间**：2024 年
- **拒绝时间**：2024 年
- **拒绝理由**：生成的代码完全是固定的，使用 Gin 而非 Kratos，业务代码中包含 SQL 字符串（非类型安全的 ORM），无法替换组件。详见 [`docs/design/2024-10-25-sponge-evaluation.md`](../design/2024-10-25-sponge-evaluation.md)。
- **重新考虑触发条件**：Sponge 支持完全自定义模板，且能生成 Kratos 风格代码；或者项目决定切换到 Gin 生态。

## [拒] make api 命令参数可配置化 (2026-07-08)

**提议**：将 protoc 的路径、输出路径等参数抽取为 Makefile 变量支持覆盖，genCURD 中 `make api` 改为可选。

**拒绝理由**：一个项目的文件结构一般很稳定，无需提供太多变量，反而让日常使用变得复杂。稳定的参数写死在 Makefile 即可。

> 来源：photo-agent backend 重构方案 5.6

## [拒] pconfig 支持 SQLite 路径的自动目录创建 (2026-07-08)

**提议**：在 `pconfig.Scan` 或 `pconfig.PostLoad` 钩子中，支持 `default:""` 路径的自动父目录创建（通过 struct tag 如 `mkdir:"true"` 控制）。

**拒绝理由**：运维逻辑报错/崩溃才是更好的方式，静默创建目录可能掩盖配置错误。

> 来源：photo-agent backend 重构方案 5.9

## [拒] pgo 增加 Proto-only 生成模式（不依赖 Kratos）(2026-07-08)

**提议**：增加轻量的 `make proto` 目标，只生成 pb.go 不做 HTTP/gRPC 代码生成，供非 Kratos 项目使用。

**拒绝理由**：如果只需要 proto 编译，用户自己执行 protoc 命令即可，不需要 pgo 提供额外封装。

> 来源：photo-agent backend 重构方案 5.10
