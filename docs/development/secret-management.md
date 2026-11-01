# 凭据管理

生产凭据不得写入仓库，包括 `.env`、配置、Kubernetes Secret manifest、临时脚本、文档示例和调试输出。配置使用私密注入与准确作用域，不打印凭据或把它传入可观察的命令参数。

## 当前配置入口

- Go 应用使用显式绝对路径的私有 JSON manifest，见[当前入口地图](repository-structure.md#current-entrypoint-map)与[配置退休边界](listingkit-config-migration-checklist.md)。
- Console 的身份与 BFF 配置见[前端 README](../../web/listingkit-ui/README.md)和 [Account Compose](../../deployments/docker/account-compose/README.md)。UI 需要的 `ZITADEL_ISSUER_URL`、`ZITADEL_CLIENT_ID`、`ZITADEL_CLIENT_SECRET` 与 redirect URI 由相应私密环境提供；本目录没有需要复制的前端 `.env.example`。
- 当前 AI 读取已发布 route 所属组织的准确凭据。全局 `openai:`、仓库自有 `TASK_PROCESSOR_OPENAI_` 以及成员/全局凭据回退已按 [#1033](../architecture/legacy-imageagent-generic-execution-closure-retirement-boundary.md)退休；凭据物理表、明确作用域的 Get/Save、引用与版本合同不变。
- 旧 `listingkit.zitadel.*`、allowlist 和 RabbitMQ 配置示例不能恢复已退休授权或输入；拒绝旧字段的陷阱仍按[当前配置说明](listingkit-config-migration-checklist.md)保留。

## Kubernetes 的既有凭据约定

适用部署使用 ExternalSecret 生成运行时 Secret，消费方继续沿其准确 `secretRef` / `envFrom`。原约定为 `secretStoreRef.name=task-processor-secrets`、`kind=ClusterSecretStore`、远端 key `task-processor/<env>/<app>`；`target.name` 必须匹配实际 Deployment。配置是否已安装由对应运维 owner 核对。

原生产上线清单中的凭据轮转、ClusterSecretStore / 远端 key、旧明文 Secret 失效以及 Git 历史清理评估仍由生产 owner 负责；不把它们变为当前内部试用的新前置，也不由本说明授权执行。

删除当前文档中的明文不代表 Git 历史或既有凭据已失效。实际轮转、历史重写与实例配置变更分别沿明确范围处理。
