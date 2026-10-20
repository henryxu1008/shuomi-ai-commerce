# 模型与费率管理 V1

依据：[冻结设计](../architecture/model-route-pricing-management-v1.md)、#830。这里描述未来经授权后的准备和使用入口；实现 PR 不代表保留实例已安装、已切模或真实模型可用。

## 使用入口

我的账户 → 企业空间 → 平台维护 → 模型与费率。只有注入平台维护 allowlist 且有当前企业成员资格者可读取，发布额外复用 `listingkit.admin.write`。入口仅在管理 API 返回当前企业实际接线用途后显示。直接访问 `/workbench/account/organization/model-route-pricing` 仍须服务端登录、当前企业和后端实时授权。

工作台上下文的当前企业 `permissions` 须回读既有 authorizer 已授予的 `listingkit.platform_admin`；仅有 `platform_admin` 角色展示或企业管理员权限不能替代此项回读及后端校验。补齐权限回读不执行安装或启用管理：未接线服务、无实际用途仍显示不可用，企业自定义模块不因此获得平台权限。

每次选择一个用途，读取当前版本和合格绑定，填写费率，校验预览，再确认发布。文本成本是原币种/百万 Token；AI 点数是输入/输出/百万 Token；图片为共享 factory 的点数/张，影响商品图片和 artwork，供应商成本尚未接入。各用途和两个 owner DB 独立发布。充值售价仍在原 Commercial 入口维护。

发布后查看准确回执和重新读取的实际当前版本。返回历史不等于回退；“以历史配置创建新预览”也必须重新校验并发布新版本。字段错误和 CAS/凭据冲突保留编辑；结果待核实时保留原请求键，恢复页面后只查询该回执，未核实前不新键发布。会话存储只保留未完成发布的原输入和请求键，发布事实仍在数据库。

## 显式准备与运行接线

以下步骤均需单独环境操作授权，不能把文档命令当作执行授权。

1. 原 ProductAgent/Run DB 和 ImageAgent DB、原消费依赖及凭据须已安装。复用原 schema owner；预先准备各库独立的 `model_route_config_writer` LOGIN 角色（有界连接，非 owner/super/DDL，不能读 Key 或改凭据）。不用 HTTP/worker pool 写 current。
2. 对每个启用库用原 `agent-configuration-schema-init` 的 `--model-route-pricing-only --runtime-role <原读取角色> --model-route-writer-role model_route_config_writer`。DSN 仍从私密 `AGENT_CONFIGURATION_SCHEMA_DSN` 提供。显式安装只增加 revision/current、去密 view 和窄 grants，不写凭据、seed current、导入旧 map 或修改历史。若同库有第二原读取角色，沿同一显式入口授 SELECT；不使用 owner pool 运行服务。
3. 私密 current-application manifest 中按需增加 `modelRoutePricing.textWriter` / `imageWriter`（复用原 DatabaseConfig 形状，host/port/database 必须匹配原执行 owner，user 固定上述窄角色，maxConnections ≤ 2）。省略某个 family 使它保持原批准的未管理运行方式；纳入管理后没有 manifest/member/global-Key fallback。
4. `modelRoutePricing.bindings` 是私密部署准入描述数组（`routeconfiguration.Definition`），不是当前选择。文本原各用途描述作为基础候选，额外候选必须属于已启用组织/用途，prompt/schema/currency 一致，token/bytes/deadline 不扩大，沿原 adapter/usage 校验。选品/诊断保留原单次半程成本上界，其它文本保留原运行成本上界。图片每个原允许组织须显式给出 `imageagent.generation` 候选，provider/model/endpoint/version 均精确绑定到原 adapter；不能推断 provider。
5. 文本额外绑定沿原私密 `product-agent-credential-provision -config <private.json> -input <private.json>`，consumer 为 title/content/materials/selection/diagnosis/planning；clientName 必须匹配本用途候选，仍使用 `title_credential_writer`，只写组织行，返回去密 credentialVersion/endpoint digest。先固定可信 endpoint，准备独立新 clientName，取得版本后绑定准入描述；切换不用 upsert 覆盖原任务凭据。图片沿原 Image credential owner 的私密维护方式准备独立组织绑定；管理页不录入 Key。
6. 启动仍用原 `current-application -config <private.json>`。startup 只验证 schema/角色并注入 fresh reader，不执行 DDL/seed。文本用原 Run pool，图片 API quote 与 worker 用同一原 Image worker pool/current reader。缺 current 时显示“尚未发布”并拒绝新执行，第一次管理发布用 NONE→新 revision。

在 managed image 模式，原 `imageAgent.generation.pointsPerImage` 仅作为部署硬预算上界（候选最大点数不得超出），不再选择当前价格；原 `priceVersion` 不进入新 quote/intent。实际 points/price/configurationVersion 全来自已发布 current。两份原 API/worker配置的一致性检查仍保留，不能以改费率为由扩大 hard limits。

数据库 writer 与原 reader 的连接生命周期由现有 currentapplication 管理；启动失败停止开放管理，不自动修表或回落。schema 检查拒绝缺失 scoped PK/唯一键/FK；serving reader 不能写配置表，writer 不能改历史、删 current、读 Key 或写 credential。

## 任务影响与验证

发布不是取消已通过最后发送门禁的请求。未发送旧报价需重新准备，多步下次调用遇到变化会停；已经发送者按冻结模型和费率完成、结算；UNKNOWN/预留、历史账单、结果、Review/Apply 不重算或转移 owner。恢复旧未决事实不依赖新 current，也不允许自动重发。

开发验证：`go test ./internal/aicapability/routeconfig ./internal/integration/aicapability/einomodel ./internal/app/worker/imageagent ./internal/app/httpapi ./internal/app/runtime/currentapplication`；事务测试可设置仅指向任务隔离 PG 的 `ISSUE830_TEST_DSN`。CI 的 routeconfig 测试复用 testcontainers 自动创建和清理独立 PostgreSQL。前端测试 `pnpm exec vitest run src/components/workbench/account/model-route-pricing.test.tsx src/lib/server/model-route-pricing-proxy.test.ts`、typecheck/build。所有配置/模型试验均使用 fixture/fake transport；无 provider 在线探测。

独立检查、自检、CI 与用户实际试用分开记录在 #830/实现 PR。保留环境安装、真实模型、真实费用及用户验收保持 NOT_RUN，待相应授权。
