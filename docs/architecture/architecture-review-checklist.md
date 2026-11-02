# Architecture Review Checklist

本清单服务当前交付，准入与 finding 分类遵守 [AGENTS.md](../../AGENTS.md)。先读当前 Issue 的 Product Decisions、Must / Should / Out of Scope、Accepted Risks 和 Design Basis；不自行扩展产品需求或 Threat Model。

## 检查当前改动

- 用户能完成什么操作，范围与明确不做是什么？页面是否符合当前 Product / Figma Authority？
- 是否复用正确的 domain / fact owner？contract → implementation → injection → consumer 是否完整？应用装配是否越权持有领域事实？
- actor、企业、成员和权限是否来自当前可信边界？请求、缓存和重试是否保持原作用域？
- 事务、幂等键及同键不同载荷如何处理？是否仍只有一个余额、资源、商品、批准和业务状态 owner？
- 超时、响应丢失、重启、撤权、取消与并发是否可能造成重复不可恢复副作用？UNKNOWN 由谁负责，恢复依据是什么？
- 持久化或外部调用是否复用现有能力与成熟组件？是否引入了未经授权的迁移、兼容、fallback 或第二事实源？
- 当前 Must 有什么直接证据？源码、CI、部署和用户验收是否分别表述？未改变代码已有证据能否复用？

真实 finding 按 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE 分类。只有命中仓库明确的 Blocker 后果才阻塞相应阶段；普通有界修改不重复全局架构评审。

## 验证入口

按风险运行已有相关测试。包依赖和源码结构规则由 `.golangci.yml` 及 `tests/` 中的源码检查直接执行；本页不复制测试名称清单或冻结文档字节。

## Review References

Every review reference must resolve to an existing repository document.

- `project-boundaries.md`
- `project-target-architecture.md`
- `auth-and-tenancy.md`
- `app-assembly-boundaries.md`
- `httpapi-assembly-boundaries.md`
- `temporal-boundaries.md`
- `commerce-tool-foundation.md`
- `../development/repository-structure.md`

专项合同从[架构索引](README.md)选择；未参与本次改动的领域不机械重复审核。
