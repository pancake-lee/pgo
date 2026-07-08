# 已拒绝的需求/方案

> 明确放弃的事项记录，防止未来重复提出。包含拒绝理由和重新考虑的触发条件。

## Sponge 代码生成工具

- **提出时间**：2024 年
- **拒绝时间**：2024 年
- **拒绝理由**：生成的代码完全是固定的，使用 Gin 而非 Kratos，业务代码中包含 SQL 字符串（非类型安全的 ORM），无法替换组件。详见 [`docs/design/sponge-evaluation.md`](../design/sponge-evaluation.md)。
- **重新考虑触发条件**：Sponge 支持完全自定义模板，且能生成 Kratos 风格代码；或者项目决定切换到 Gin 生态。
