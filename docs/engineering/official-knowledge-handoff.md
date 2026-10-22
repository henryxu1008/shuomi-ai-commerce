# 官方知识库首版交接

当前源码增量执行 #884 / PR #887，依据[产品内依据V2冻结设计](../architecture/official-knowledge-provenance-v2.md)和用户「官方知识库不要使用github地址」决定。原 #632 / PR #633 的阅读范围、owner和权限沿用，外部来源URL合同已退出。

**V2尚未安装到保留实例。** 本轮仅实现、隔离检查与PR交付；下方2026-10-11的实例权限/交接只描述历史v1，不能授权本轮部署或证明当前实例已切换。

## 正常入口与操作

V2以后获准配套安装到正常 current-application / Console 后，使用原有 Auth.js / ZITADEL 登录，选择企业，通过 **AI工作台 → 知识库 → 官方知识库** 打开 `/workbench/ai/knowledge/official`，点击《AI电商应用指南》的“查看内容”。当前源码准确版本地址为 `/workbench/ai/knowledge/official/ai-commerce-guide/revisions/2`。

目录显示实际标题、摘要、维护方、版本和更新日期；正文页显示准确版本、SHA-256 和纯文本内容依据名称。API来源只有title，不提供URL字段或GitHub地址，页面不创建出处超链接。不存在的版本返回明确失败，不替换成最新版；新程序的v1准确路径返回404，不跳转v2。

读取使用当前企业既有 `workbench.knowledge.read` 权限：企业自定义角色的 `knowledge` 模块或既有受保护管理员策略。退出、企业切换、授权撤销或读取失败时不继续显示缓存正文。旧静态 operator/viewer 不是当前企业授权。菜单入口本身不授予权限。

## 正常启动方式

沿用 `cmd/current-application` 的私有绝对路径 manifest 和原有空安装/已安装 owner 配置。正常程序构建、启动命令：

```powershell
go build -o .local/current-application.exe ./cmd/current-application
& .local/current-application.exe -config 'C:\获准实例目录\current-application.json'
```

在另一个终端进入 `web/listingkit-ui`，使用原有私密 Auth.js / ZITADEL 配置，设置 `LISTINGKIT_SERVICE_API_BASE` 为该程序地址加 `/api/v1`、`LISTINGKIT_PUBLIC_BASE_URL` 为 Console 地址，执行 `pnpm install --frozen-lockfile`、`pnpm build`、`pnpm start --hostname 127.0.0.1 --port <已分配端口>`。服务地址和登录配置由该实例原有安装负责，本交付不重新安装或修改其数据。

官方模块随 `Workbench.Enabled` 的正常应用组合注册；没有新增 feature flag、schema、数据库权限、Tika、对象存储或模型依赖。企业知识库的 `LISTINGKIT_KNOWLEDGE_ENABLED` 与后端 Knowledge 配置仍独立控制“我的知识库”，不影响官方阅读。

历史v1交接：原 PR #633 消费 #631 的既有正常菜单与运行接线。当时用户授权接入保留实例 `task-processor-unified-20261010`，由原运行 owner 更新配套镜像并保留配置、全部命名卷和业务资料；该历史授权不覆盖本轮V2。

该历史实例入口为 [官方知识库](https://localhost:35444/workbench/ai/knowledge/official)。历史运行源码/镜像证据见 #632 / PR #633；本轮未更新，旧页面不证明V2已安装。以后需要更新须由原运行owner在新授权下使用配套backend/UI；正常停启仍由原私密启动脚本负责。

## 内容维护与限制

当前正文位于 `internal/knowledge/official/content/ai-commerce-guide-v2.txt`，由 `internal/knowledge/official` 校验并嵌入程序。发布修改经原PR审核，正文统一LF字节，版本、摘要、日期、依据名称由内容owner同步维护。v1保存在Git历史，未被原地改写；新程序只提供v2。新的准确版本更新digest/日期，重启读取同一内置内容，不写入企业知识库或建立第二事实源。

首版只有仓库事实支持的《AI电商应用指南》。其他平台规则、运营或合规栏目没有已批准资料，未开放。没有用户端编辑、上传、发布、收藏、“选择使用”或自动更新动作。**AI 引用属于 Later**：阅读不会创建任务、调用模型、计费、修改商品或发布平台。

开发自检、受控浏览器显示检查和代码评审分别记录在 PR。受控外部身份/权限与 HTTP 响应夹具只验证页面显示，不代表真实 IAM 或生产链路。用户试用、产品验收、真实 provider/payment/channel 和生产部署保持 `NOT_RUN`；不签发上线结论。
