# 文档组织与分批迁移规则

> 本页用于**定位现行资料和规划文档位置**，不创设新的产品要求、架构准入、运行权力或验收门禁。实际产品、领域与开发决策仍以各自当前权威为准。执行依据：[Issue #1340](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1340)。

## 1. 目标与当前基线

- 从 [文档首页](README.md) 查找产品、业务流程、领域 owner、实现入口、运行说明与任务证据，而不是从 200 多份扁平架构文件中猜测。
- 减少在多个文档复制同一业务规则；架构所有权参考 [当前领域边界](architecture/project-boundaries.md)，源码与命令参考 [当前源码地图](development/repository-structure.md)。
- **清单基线：** `main` 提交 `b67c8d7ee5c6bf543d4723c3e768219fa314fd35`；`docs/` 共 **350 份 Markdown**、**3 份其他资产**，其中 `architecture/` 根层 242 份 Markdown。详细逐文件候选动作在 [document-inventory.csv](document-inventory.csv)。
- 已识别 **26 组跨分类同名（不含 README）**。名称相同不等于内容重复，产品决定、领域合同和运行手册可以合法同名。

## 2. 各层唯一职责

| 文档层 | 回答的问题 | 当前入口 | 不承接 |
| --- | --- | --- | --- |
| **Product** | 使用者从哪里开始、为什么做、业务状态/人工确认/输出是什么？ | [产品索引](product/README.md)、[业务流程参考](product/final-ui-ia-authority.md) | Schema、API 细节、CI、实例证据 |
| **Architecture** | 哪个领域拥有事实？合同、授权、版本、状态、事务与恢复如何约束？ | [架构索引](architecture/README.md)、[领域边界](architecture/project-boundaries.md) | 单次操作日志、试用环境的临时状态 |
| **Development** | 到哪里改代码，怎样配置本地开发、运行测试？ | [开发索引](development/README.md)、[源码地图](development/repository-structure.md) | 第二套业务/领域合同 |
| **Engineering** | 如何按 Issue 接单、做评审、组织当前消费者接线？ | [工程索引](engineering/README.md)、[Issue 驱动规则](engineering/issue-driven-development.md) | 长期领域事实的重复 owner |
| **Operations** | 使用者/运维人员如何安装、配置、恢复、核对实际结果？ | [运行索引](operations/README.md) | 新产品授权、未经批准的真实操作 |

`AGENTS.md` 保持 AI Agent 开发与准入规则的仓库级唯一入口；不把它复制到各领域。单次 Issue/PR 负责准确提交、测试、CI、部署和试用的**时点证据**，不能由文档中的 `READY` 或历史 `PASS` 自动推断生产可用。

## 3. 目标目录（渐进式，不立即迁移）

```text
docs/
├── README.md                     # 角色入口与权威来源
├── document-organization.md      # 本页：分类、迁移、检查
├── document-inventory.csv        # 逐文件候选清单
├── product/
│   ├── README.md
│   ├── flows/                    # 端到端业务操作与交接
│   └── decisions/                # 现行产品决定（按需引入）
├── architecture/
│   ├── README.md
│   ├── project-boundaries.md     # 保留路径：领域权威
│   ├── project-target-architecture.md
│   ├── domains/
│   │   ├── sourcing/
│   │   ├── catalog/
│   │   ├── research/
│   │   ├── ai/
│   │   ├── workbench/
│   │   ├── listing/
│   │   ├── marketplace/
│   │   ├── store/
│   │   ├── commercial/
│   │   └── data-services/
│   └── platform/
│       ├── identity/
│       ├── runtime/
│       └── integrations/
├── development/
│   ├── README.md
│   ├── repository-structure.md   # 保留路径：源码入口
│   ├── guides/
│   └── ui/
├── engineering/
│   ├── README.md
│   ├── issue-driven-development.md
│   └── integration/
└── operations/
    ├── README.md
    ├── installation/
    ├── runbooks/
    └── user-guides/
```

以上是**候选目录**，空目录不预先创建。领域分类最终以 [project-boundaries.md](architecture/project-boundaries.md) 与文档内容确认，不凭文件名前缀确定 fact owner。跨领域消费者可以链接多个合同，不复制它们。

## 已核实迁移与 CSV 使用说明（2026-11-17）

原 353 条清单以 `b67c8d7ee5c6bf543d4723c3e768219fa314fd35` 为**盘点基线**，不是以后新增导航文件的完整动态文件目录。除已复核的迁移文件外，`PROPOSED_ONLY` 的候选领域、路径和大小仍为当时的快照，不得当作实时源码状态或直接派工的授权。

已完成且确认源链接的路径迁移：
- [#1358](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1358)：`issue132-title-evaluation-admission.md` 与 `issue132-title-pilot-local-entry.md` 从架构根目录迁入 `architecture/domains/ai/`；仍是未准入的设计候选。
- [#1359](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1359)：`completion-draft-internal-note.md` 迁入 `architecture/domains/listing/`，保留 Listing CompletionDraft 已冻结合同。
- [#1361](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1361)：两份 Listing CompletionDraft 历史/恢复架构合同成组迁入 `architecture/domains/listing/`，原 Operations 使用说明继续独立维护。

对应 CSV 三十九行更新为新的真实 `path`、`KEEP_PATH`、`MOVED_VERIFIED`，`size_bytes` 为迁移后 UTF-8 正文字节数。其余行保留原盘点视角、`PROPOSED_ONLY` 与原候选路径；迁移后**不得仅据旧 CSV 行推断某文件还在原位置**，应重新检查当前 main。

- [#1363](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1363)：`CompletionDraft` 结构化候选冻结合同由架构根层迁至 `architecture/domains/listing/`；相邻 Product 与 Operations 同名文档各自保留业务/使用说明职责，不进行内容合并。
- [#1366](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1366)：C3-2 SHEIN 候选类目规则查询与只读解释合同归入 `architecture/domains/listing/`；Marketplace RuleReader/Provider 仍提供原规则，不变更 C3-2 R1 准入与不可采用边界。
- [#1369](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1369)：将平台上架资料智能体 v1 的 Listing materials 冻结合同迁入 `architecture/domains/listing/`；保留 Marketplace 官方规则、当前 Agent 版本与独立准入范围，不声称自动发布或运行验收。
- [#1372](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1372)：SHEIN native Validator 与 submit readiness action policy 两份 Marketplace 协议/校验合同成组迁入 `architecture/domains/marketplace/`；Listing 仍持有 Target/Submission 事实与原键恢复，保留原版本/历史准入。
- [#1374](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1374)：现行 SHEIN 单商品多颜色/SKU 销售属性映射合同归入 `architecture/domains/listing/`；明确保留原根层的颜色拆稿旧设计为已被 2026-11-09 业务决定替代的历史资料，不迁成当前流程，不解除稀疏规格发布阻断。
- [#1378](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1378)：TEMU、TikTok Shop US、Walmart US 三份**官方合同证据**归入 `architecture/domains/marketplace/`；这些是既有公开接口调查的有界快照，不改变原产品设计准入、商家授权范围、UNKNOWN/NOT_RUN 或真实发布资格。
- [#1380](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1380)：TEMU、TikTok Shop US、Walmart US 三份跨 Store、Listing、Marketplace 的平台专项总设计统一移至 Marketplace 阅读目录；各原事实 Owner、实施许可、未解决真实商家授权/官方规格缺口及历史 NOT_RUN 均保持，不据目录改变权威。
- [#1383](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1383)：SHEIN 首次上架/状态回读、TikTok 发送前 no-send 与显式重试、Walmart Feed 拒绝修复/有界续读三份原 Submission/结果恢复合同迁入 `architecture/domains/listing/`，保留 Marketplace 官方协议来源、原权限/Unknown/人工确认和历史适用边界；不得把移动误认为产品实际已发布或原恢复门禁放宽。

- [#1386](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1386)：Provider-neutral AI 合同与关闭新执行/保留历史读取合同归入 `architecture/domains/ai/`；保留原状态、权限、预算和运行 NOT_RUN。

- [#1390](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1390)：官方技能模板自动准备与准确技能版本已有商品入口两份 AI 架构合同归入 `architecture/domains/ai/`；保持原权限/版本/人工确认与历史试用限制，未触碰 Skill 或商品执行。

- [#1392](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1392)：通用图片模板与 Image Agent Workflow 合同归入 `architecture/domains/ai/`，保留现有技能版本、Temporal/effect owner、UNKNOWN、预算及人工批准边界。

- [#1394](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1394)：图片目标语言/人工确认与 API 事件读取权限合同迁入 `architecture/domains/ai/`；保留原预算、权限、UNKNOWN 和历史运行边界。

- [#1398](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1398)：Image Agent 效果终态与 external-effect finalization 两份历史合同迁入 `architecture/domains/ai/`；保留既有版本与两份合同的不同范围，不将目录移动解释成协议收敛或恢复放宽。

- 本批 Image Agent 外部效果恢复与兼容协议归入 `architecture/domains/ai/`；保留原 Temporal 版本、fence、UNKNOWN 和历史准入，不因路径移动开放重试。

- 本批 Image Agent 预算准入和候选 worker 发布保护两份合同归入 `architecture/domains/ai/`，原有预算/UNKNOWN 与候选隔离保护不变；路径归类不代表运行验收或发布准入。

- 本批将 DeepSeek 共享文字参考价准入及 GRSAI VIP 图片绑定合同归入 `architecture/domains/ai/`；保持准确凭据、预算、旧版历史准入与模型调用限制，路径移动不代表生产开放。

- 本批：智能体定制人工流程与 VIP 已付费原图恢复合同迁入 `architecture/domains/ai/`；原权限、报价/预算、人工处理、UNKNOWN 及历史状态保持，不开放真实调用。

- 本批 Image Agent v3 Effect Policy 冻结设计合同归入 `architecture/domains/ai/`；保持原领域策略、CAS、UNKNOWN、租约与历史准入，迁移不构成运行/实现许可。

## 4. 逐文件清单的解释

[CSV 清单](document-inventory.csv) 每行包括：

- `path`：已核实迁移的行写入当前真实路径；其余行保留原基线快照，使用前必须再次核对 main。`layer`、`domain`：原为候选领域，已迁移行按核实的领域校准。
- `action`：`KEEP_PATH`（保留关键路径/当前路径）、`KEEP_ASSET`（非 Markdown 资产）、`REVIEW_MOVE`（待阅读后判断是否搬迁）、`REVIEW_CROSS_LAYER`（跨层同名关联，先核实不同职责）、`REVIEW_LIFECYCLE`（试用、候选、特定实例或历史时期名称，先核实适用性）。
- `candidate_path`：**提议路径，不是已批准的迁移操作**；`related_paths`：跨分类同名关联；`size_bytes`：GitHub 当前文件大小。
- `review_status=PROPOSED_ONLY` 适用于仍未逐篇核实的 333 个原基线行；20 行 `MOVED_VERIFIED` 表示仅**准确路径与相对引用迁移已核实**，不代表设计本身已准入、产品已验收或允许关闭 Issue。不得按表直接删除、搬移或关闭任务。

任何新一轮文件检查应重新取当时主线与相关文档版本；旧行可能因正常开发变动而过期。若准确 owner/内容与候选目录不一致，应修改清单后再实施。

## 5. 执行时每份文件的决定

1. **KEEP**：当前有效且位置合理（或有稳定深链接），原路径保留。
2. **UPDATE**：原位置保留，仅改过期决定、错误链接和职责说明；修改需遵循最新用户决定，历史证据不倒签。
3. **MOVE**：已核对新目录与 owner，检查所有 Markdown、源码、测试、脚本、CI、外部文档入口和锚点引用后，在同一 PR 内移动并同步引用。跨领域入口仍可链接原权威合同。
4. **CONSOLIDATE**：确认文档确实维护了相同业务事实、且合并不会抹去产品/架构/操作层独立职责时，选一个权威位置，其余消费者改为链接。不得因同名自动合并。
5. **RETIRE**：只有确证已退出且当前无需读取的文档，在将仍有效的合同和使用入口归位、修复引用后才删除；历史通过 Git 追溯。**不新建 archive 或 retired 文档堆放区**。

搬迁中的高引用文件（`project-boundaries.md`、`project-target-architecture.md`、`repository-structure.md`、各层 `README.md`、`AGENTS.md`）默认暂不移动，除非有单独批准的必要理由。

## 6. 典型跨层同名文件的内容抽查

为避免将“相同文件名”误判为内容重复，已抽查以下实际正文。以下结论只覆盖列出的文档，不等于全部 26 组已完成语义审查：

| 主题 | 抽查结果 | 当前建议 |
| --- | --- | --- |
| `selected-application-multi-product.md`（Product / Architecture / Operations） | 分别是多商品申请产品决定、逐商品幂等与状态的冻结合同、卖家进入和操作说明 | **KEEP_SEPARATE**，三层互相引用；不按同名合并 |
| `supply-unbound-completion-drafts.md`（Product / Architecture / Operations） | 分别说明不绑定 Store 的业务许可、Listing CompletionDraft 事实 owner/事务、安装与保存恢复步骤 | **KEEP_SEPARATE**；核对旧 UI/Figma 权威表述，但不取消独立草稿 |
| `skill-capability.md`（Architecture / Operations） | 前者是 Skill 身份、版本与模型/脚本准入合同，后者是安装和操作交接 | **KEEP_SEPARATE**；其中旧阶段“未启动”等时点描述要与最新 Issue/PR 分开表示 |
| `my-supply-chain-shein.md`（Architecture / Operations） | 旧五阶段技术设计已在文首声明被四阶段增量替代；操作说明已转向「商品刊登」四阶段 | **UPDATE_REVIEW**：优先固定现行四阶段权威链接，审查原合同仍被消费的部分后再决定是否精简 |
| `ecosystem-services.md`（Architecture / Operations） | 冻结合同含完整支付/结算，而现行有界资格接线与运行说明强调“非支付阶段” | **KEEP_SEPARATE**，进一步标清两阶段适用条件；不可因为代码或目录存在就宣布真实支付开放 |

上表的 `KEEP_SEPARATE` / `UPDATE_REVIEW` 是**人工抽查建议**；除前述已核实迁移行外，CSV 原基线仍保持 `PROPOSED_ONLY`，这些抽查不授权修改上述领域合同。后一批需核实当前产品决定、准确发布/试用范围和反向引用后再具体实施。

## 7. 推荐推进与验收

| 批次 | 改动 | 验收 |
| --- | --- | --- |
| **1：本 PR** | 清单、导航、职责定义；不移动任何业务合同 | 353 行清单对应准确文件，当前导航链接可达，业务/运行行为零变化 |
| **2：领域架构** | 按 Sourcing → Catalog → Research → Listing/Marketplace → AI/Workbench 等真实 owner 分批，逐批处理 | 每份迁移前有原文/引用检查，当前 owner 只保留一个规则定义，受影响路径全部更新 |
| **3：跨层材料** | 核对 26 组同名、环境试用和候选文档 | 不误删独立 Product/Architecture/Operations 内容，失效文档有具体证据 |
| **4：轻量检查** | 先评估现有仓库 CI 的链接/索引检查，再最小增量复用 | 不产生新的通用治理平台、重型门禁或与开发无关的检查 |

本批 **Design Basis: N/A**：只引入索引与候选清单，不修改产品能力、领域合同、权限、状态机、持久化、计量、外部副作用或运行环境。不代表任何实例的新验收。
