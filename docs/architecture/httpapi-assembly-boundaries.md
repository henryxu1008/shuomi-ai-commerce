# HTTP API Assembly Boundaries

当前链路为 `cmd/* -> internal/app/httpapi -> internal/*/httpapi -> domain service/runtime/contracts`。总规则见[系统边界](project-boundaries.md)，实际入口见[源码地图](../development/repository-structure.md#current-entrypoint-map)。

## 职责

- `cmd/*` 读取明确配置，调用应用入口，不持有路由、领域事实或服务组装细节。
- `internal/app/httpapi` 是当前 HTTP composition root，创建连接和当前模块，注入依赖、组合 middleware、路由描述符及生命周期。
- 领域自己的 `httpapi` 包拥有 handler 和 route registration。Domain / business packages 不导入 `internal/app/httpapi`；HTTP 请求 DTO 不成为领域事实。
- RunOptions、应用配置解析和运行状态属于 app。领域服务、查询、事务、授权语义与恢复属于对应 owner。
- Provider、持久化和 Marketplace client 由专属适配/装配文件接入，不集中进一个通用 adapters 文件，也不让领域实现依赖应用装配。
- Worker、登录、Product/Image、Store 等模块使用各自 owner 的输入、模块和结果类型；不在应用层复制同义别名或兼容 builder。

## 路由与权限

中间件解析可信 actor 与当前企业上下文，领域入口继续执行具体权限与原作用域校验。挂载路由不授予权限；未配置或未开放能力按原合同 denied/unavailable。

启动只验证当前 schema 与运行权限。安装和 schema 维护走明确命令，不把 DDL、真实平台连接或外部副作用藏进 HTTP startup。

## 验证

实际依赖和装配限制由 `tests/` 及 `depguard: domain_app_httpapi_boundaries` 检查，包括领域不得导入 app、模块 owner 类型、运行依赖和外部适配边界。本页只描述稳定规则，不复制测试名称清单或已删除文件的退休经过。
