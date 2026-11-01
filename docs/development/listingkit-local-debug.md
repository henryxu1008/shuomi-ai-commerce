# 当前本地开发与联调入口

当前后端从私有 JSON manifest 启动，命令与模块条件见[入口地图](repository-structure.md#current-entrypoint-map)；新建独立环境见 [Account Compose](../../deployments/docker/account-compose/README.md)。配置不得从旧 Task API 的 YAML 或全局环境变量推断。

Console 开发见[前端 README](../../web/listingkit-ui/README.md)。现有 UI launcher 使用 `ServiceApiBase` / `LISTINGKIT_SERVICE_API_BASE`；旧 `ApiBase`、Task API 环境变量和自动读取集群 Secret 已按 [#1029](../architecture/legacy-listingkit-input-config-closure-retirement-boundary.md)退休。

身份与组织授权继续使用当前 ZITADEL / Auth.js 与显式权限合同，不能信任调用方 `X-User-*` / tenant 头。凭据与注入边界见[凭据管理](secret-management.md)。

旧 API/dev launcher、六 mode ImageAgent 编排及 API token 文件工具分别按 [#846](../architecture/legacy-local-acceptance-closure-retirement-boundary.md)、[#852](../architecture/legacy-api-token-tools-retirement-boundary.md)退休；不再列出其运行命令。原说明见[固定历史版本](https://github.com/henryxu1008/shuomi-ai-commerce/blob/c0210ff34f44c371b5a1993aed89f439cfbdc1ff/docs/development/listingkit-local-debug.md)。

源码说明不证明已有实例状态。既有实例的升级、重启、凭据和数据操作另沿原 owner 及明确授权；生产 `NOT_READY` 与[旧发布退休边界](../architecture/release-identity-preflight-retirement.md)不变。
