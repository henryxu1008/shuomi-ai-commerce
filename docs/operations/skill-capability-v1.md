# Skill V1 交接（#688）

Design Basis：[已批准设计](../architecture/skill-capability-v1.md)。本文件交接正常安装和使用方式；不签发共享安装、真实模型或用户验收。

## 入口和操作

1. 登录后进入 **我的账户 → 账户资料**（`/workbench/account/profile`），点击“我的技能”。没有当前企业也能创建、导入、检查格式、保存版本、启停和导出本人技能。新建默认提供 `product-title-normalization` 方法文本；它只用已保存事实，不猜品牌、材质、认证、销量或利润。格式通过只表示包符合受支持格式。
2. “官方模板”只展示实际存储的可读取版本。已获现平台维护权限的人员可在此导入、新建、发布并启用官方方法；普通用户只能读取获开放的版本、按版本的复制/导出许可复制。空库不伪造官方模板。模板复制生成新的个人/组织私有草稿，来源 ID、版本和摘要保留；后续各自演进。
3. 编辑保存为新版本，不覆盖已经保存的正文。个人版本先发布再启用；共享/复制能力由保存时的明确许可确定，改变许可同样保存新版本。
4. 有当前企业时，可勾选同意并提交本人指定版本给该企业。组织维护者从 **我的智能体 → 商品内容与本地化配置**（`/workbench/agents/mine/product.content.agent`）的“团队技能”读取指定提交、批准并生成组织私有副本，随后发布/启用。批准不改变个人原件，也不赋予维护者查看其他个人材料的权限；批准后的副本不受个人后续撤销影响。未批准申请可由本人撤回。
5. “团队技能”中维护者明确允许个人技能准入；缺策略时默认不允许。组织成员只能依照现有 use/configure 权限创建/维护对应草稿，跨企业或他人私有技能不可读取。
6. 打开已保存商品的采集结果（`/workbench/supply/acquisition/operation/<operationId>`），选择现有“内容完善与本地化”。选择语言且仅勾选标题，在原执行确认中勾选“使用技能 / 方法”，选择来源和准确版本，再确认生成。一次只选一个；未选择技能的原路径不访问技能库。选中版本失效时停止并要求明确重新选择，不切换 latest。
7. 建议、任务和原内容审核页显示实际 Skill ID、归属、版本和摘要；正文按当前资源权限单独读取。原 default 人工审核、auto_approve 程序校验、read_only 仅建议、full_access 合法策略采用沿原规则。采用后在原 Review 回执和商品详情读回；技能不直接保存商品或发布平台。

## 受支持导入

- `SKILL.md`：严格 YAML frontmatter 的 `name`、`description`，可选 `license`、有界字符串 `metadata`；正文为 Markdown。
- 一层 `references/*.md|txt`；一层 `assets/*.md|txt|json`，JSON 模板须合法。
- 最多 16 文件，每文件 16 KiB，排序规范化后的完整文件 JSON 至多 48 KiB，HTTP 输入 96 KiB；完整材料在原报价前进入同一输入。
- 导出的 JSON 包可再导入；无自动下载、压缩包、脚本、Shell/Python、网络/图像/HTML、越界或缺失引用。不支持的 `allowed-tools`、model、context/fork 等字段明确拒绝，不当作宿主权限。

## 显式安装和正常启动

使用既有 ProductAgentDB / serving pool。仅管理技能时保留 current-application 的 `productAgent.database` 并设 `enabled:false`；不要求接入模型。新技能 schema 的未安装或不匹配状态返回不可用，应用启动不执行 DDL，也不影响未选技能的现有内容路径。

获环境安装授权的操作人员通过私密 `AGENT_CONFIGURATION_SCHEMA_DSN` 与已有普通 serving role 执行：

```powershell
# 已有合格当前配置库：仅增加技能存储和窄个人技能准入表。
go run ./cmd/agent-configuration-schema-init --runtime-role <existing_serving_role> --skills-only

# 全新空安装：原配置 + 技能 schema；原 Run/Review/Catalog 按各自入口安装。
go run ./cmd/agent-configuration-schema-init --runtime-role <existing_serving_role>

# 原正常应用入口，private manifest 复用当前模型、资源与数据库配置。
go run ./cmd/current-application -config <absolute_private_manifest.json>

# 在 web/listingkit-ui，使用原私密 ZITADEL/BFF 环境配置。
pnpm install --frozen-lockfile
pnpm build
pnpm start --hostname 127.0.0.1 --port <allocated_port>
```

`--skills-only` 与其他局部 installer 选项互斥，核对既有 schema owner/约束，不搬迁、回填、改属或清理数据。`skill_capability_owner` 与原 `agent_configuration_owner` 均为非登录 owner。Serving 只得范围内 SELECT/INSERT 和指定状态列 UPDATE；材料、摘要、owner、creator、复制许可及来源不可改写，无 DELETE/TRUNCATE。危险的有效继承授权使安装失败。安装命令不启用企业、准入个人方法、导入官方模板或调用模型。

实际商品执行还需原 Product/Run/Review/Catalog、企业启用、有效成员权限、contentTextPolicies、计量/额度/计价和已准入 provider 配置。UI/BFF 复用 `LISTINGKIT_SERVICE_API_BASE`、`PRODUCT_REVIEW_API_ORIGIN` 与现会话；不接收浏览器提交的 owner、角色、身份或凭据权威。模型真实费用和 provider 健康仍需另行确认。

## 商品内容模型凭据

个人技能管理完成后，真实标题优化使用商品内容智能体的独立 `product.content.decision` 路由。
部署者先保持执行关闭，在私密 manifest 的 `productAgent.contentTextPolicies` 中配置准确组织的
profile：独立 client、供应商、模型、endpoint 准入、版本化计价/点数费率，以及 token/字节/deadline
和运行费用上限。固定 `PromptVersion: product-content-agent-v1`、
`OutputSchemaVersion: product-content-action-v1`；首次凭据写入时
`AdmittedCredentialVersion` 和 `AdmittedEndpointIdentityDigest` 留空，导入后用真实返回值绑定。
凭据导入无需启用执行，也不借用 title/planning 策略。当前支持已准入的 OpenAI 兼容与 Claude
原生 route；仅标题能力准入的 Google Interactions adapter 不适用于商品内容。

复用 [受限凭据写入说明](product-agent-trial.md)：目标 ProductAgentDB 必须已具备合格的
`ai_client_credentials` schema 和独立受限 writer，serving role 保持只读。命令不会创建表、角色
或修补权限；缺少这些前置时先由原安装负责人提出独立安装方案，不能用 serving/管理员连接替代。
私密输入沿原字段使用 `action: "upsert"`、`consumer: "content"`、`organizationId`、准确
`clientName`、`apiKey`、`baseURL`、`model`、`apiStyle`、`timeoutSecond` 和 `writerDatabase`。
writer 必须连接同一 host/port/database 并使用独立身份，timeout 不短于 profile 的 deadline。
密钥和数据库密码只放在受限访问的本地私密文件中，不写入命令行、仓库或普通日志。
同一组织的 content、title、planning、materials、selection、diagnosis 策略须使用各自的凭据
client。原 owner 的键是组织/用户/client，不含消费者；命令在 upsert 和 disable 前按其
去首尾空白、空名映射为 default 的语义检查全部已配置引用，拒绝跨能力重名，防止覆盖或停用
另一能力的凭据。未启用策略仍参与检查；不同组织的同名 client 不冲突，不自动改名或迁移旧行。

```powershell
go run ./cmd/product-agent-credential-provision -config C:\private\current-application.json -input C:\private\content-credential.json
```

命令只返回非敏感的组织、content 供应商、凭据版本及 endpoint 摘要。将后两项准确填回本组织
content 策略；轮换后重新绑定，不自动启用 runtime。停用仍使用同一命令，私密输入仅保留
`action: "disable"`、`consumer: "content"`、组织、client 和 writerDatabase，不携带 provider 值。
它只停用该组织的准确 client 行，不回落成员或其他能力凭据。

真实试用前还需合格的 Run/调用计量表及原 Review/Asset/Catalog 角色与权限、允许当前服务模式
启用商品智能体、企业启用与成员额度、个人技能指定版本发布/启用及企业个人技能准入。用户须
确认商品/语言、向供应商发送商品信息、实际计价与单次/累计费用上限，并另行授权运行配置、
必要停启及真实付费调用。缺项时保留执行关闭；凭据写入、CI 或隔离测试不代表真实生成已可用。

## 保存、停用与恢复

Skill 定义、不可变版本、命令回执和共享申请在同 RunDB 的 `skill_capability`；个人技能准入在原 `agent_configuration.skill_admissions`。原配置、运行、预算/UNKNOWN、Review 和 Catalog 保持各自 owner。备份应一起覆盖 Skill、配置和 Run 事实，并保留原 ProductDB 备份；不通过重新导入构造运行历史。

管理写入复用准确 UUID key 和适用 revision；响应丢失时界面保留原命令，核实或重试原编号，拒绝同键异载荷。模型 UNKNOWN 仍由原运行/调用 owner 处理，不换编号或技能重发。版本撤销/准入关闭/企业停用会与当前有界 Skill use gate 排序：已经先准入的同步效果可以结束，后续新效果被拒绝。原已提交回执仍可读，不自动补偿、删除历史或修改组织副本。

重启/停止与删除数据分开。回滚应用不删除新表、命名 volumes、已保存版本或原业务事实。真实环境安装、服务重启、共享实例准备、真实模型、用户试用及生产验收在授权执行前均为 **NOT_RUN**。

## 开发验证入口

复用仓库 Go/React/Vitest 检查，无独立 runner 或验收平台。隔离 PostgreSQL：`ISSUE688_TEST_DSN` 运行 Skill 持久化/权限/并发测试；原 HTTP fixture 使用任务自己的 `ISSUE398_TEST_DSN`。安装本工作区前端依赖后，`ISSUE688_FRONTEND_TEST=1` 的 `TestSkillConsoleClientsUseActualBFFAndOwners` 执行真实 TypeScript client → Next route/BFF → mounted Go auth/Skill/config/Agent → 原 Review/Catalog；只有外部身份与模型响应受控替代。该测试不表示真实 ZITADEL、1688、模型质量或浏览器用户验收。滚动 SHA、CI 和独立评审证据写主要 PR。
