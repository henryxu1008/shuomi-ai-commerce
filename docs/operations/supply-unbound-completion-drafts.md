# 未绑定店铺的待补全草稿

适用设计：[已冻结 Independent Architecture](../architecture/supply-unbound-completion-drafts.md)，产品决定 [PD-SUPPLY-UNBOUND-COMPLETION-2026-10-12](../product/supply-unbound-completion-drafts.md)。

当前阶段交付“已保存商品 → 原转入批次 → 独立候选标题/描述 → 保存版本 → 刷新后继续编辑”。普通 Application 仍复用既有身份、SourceAccount、Commercial/resource 和 Product 装配；该草稿消费者不需要 Store、SHEIN、Asset、采集器、模型或 Temporal。当前店铺同步 preparation worker 与其他已安装能力应保留。

## 明确启用

1. 使用当前 Product database 和原 runtime login，不能用 schema owner 启动服务。Manifest 增加 `"supplyCompletionDrafts":true`，保持 `"productCollections":true`；保留其他现有 manifest 字段和私密值。当前 live membership 配置必须已有，不能用本地假组织代替。
2. UI 增加 `LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED=true`，保持 ProductCollections 开关。后台只 VerifySchema / VerifyRuntimePermissions，不执行 DDL。缺 schema/grants 应拒绝启动，不能只打开前端。
3. 新安装可在原 Product acquisition owner manifest 中增加 `"supplyCompletionDrafts":true`，沿全新空库初始化合同安装。保留实例只能在取得该精确目标库的单独授权后执行下一节的 additive owner 操作。

## 保留实例的 owner 安装

操作前保存原 Product 备份、全部现有表计数、runtime table/column grants 和私有 manifest；核对准确实例、数据库、原角色及 serving 镜像。凭据只从受保护的本地绝对路径读取，不写聊天、日志或 Git。

private owner manifest 使用原 `product-acquisition-init` 格式，`schemaVersion:1`、`collections:true`、`supplyCompletionDrafts:true`，Database 指向精确本机 Product 目标和已有 schema owner。从候选源码或对应 schema-init 镜像执行：

```powershell
go run ./cmd/product-acquisition-init -config C:\private\completion-owner.json -enable-completion-drafts -confirm-database product_acquisition
```

该入口用原 owner transaction/advisory lock，验证现有 Collection schema，安装原 Preparation schema（已存在时不改写）和本 owner 三表：

- `listing_completion_draft_records`：SELECT/INSERT。
- `listing_completion_draft_heads`：SELECT/INSERT/UPDATE。
- `listing_completion_draft_commands`：SELECT/INSERT。

同时授予原 Preparation 所需最小权限：`listing_preparations` SELECT/INSERT/UPDATE，`listing_preparation_sources` SELECT/INSERT。它不 reset/revoke 原 Product grants，不更新或删除任何业务行，不开放正式执行表、不创建 runtime 用户或修改其凭据。已有 Preparation 消费者的权限仍必须保留。

启用后台和 UI 时沿原所有 Compose 层、命名卷和私有 manifest，只更新 serving 容器；不要启动旧全库初始化器。每次 restart 必须使用一致的 enabled capability union。Product acquisition、BrowserCapture、Collections、plans、完整 SupplyChain 和市场的 grant verifier 均消费本开关。

## 入口和使用

`/workbench/supply/mine` 是独立首页。待适配 `/workbench/supply/mine/waiting` 中选择真实批次及来源，点击“创建 / 编辑待补全草稿”。完整 SupplyChain 开启时可从“保存未绑定草稿”进入同一原来源入口。待补全列表 `/workbench/supply/mine/missing` 显示真实草稿数量、原来源版本和保存版本。

编辑器展示原始资料和独立候选标题、描述。空字段允许保存；候选标题最多 1024 UTF-8 字节，描述最多 32768 字节，完整请求最多 64KiB。列表复用 Supply 的 2MiB response budget，必要时提前分页，nextCursor 指向已返回的最后来源，不丢失后续记录。

保存结果不确定时保留原命令键及完整请求；点击“核实原操作”。NOT_FOUND 或网络失败不会允许换键覆盖；只有原请求恢复成功或确定拒绝后才能继续。恢复旧回执后重新读取当前 head，避免用旧版本继续覆盖。

以后配置真实店铺后，可在原平台编辑器显式选择“带入待补全草稿 Rn”；只带入文字，保留已有类目、图片、规格及其他语种。原 Store/category、权限、规则、资产批准、Target.Create 和上传门禁继续适用，候选草稿不代表平台就绪。

## 回退与数据保留

停新能力和 UI 开关，保留三表及全部保存记录。若旧镜像的严格 grant verifier 不接受新 grants，由授权 owner 仅撤三张 CompletionDraft 表的 SELECT/INSERT/UPDATE；不能撤原 Preparation grants，除非已核对其他 plans/full 消费者确实不使用。不要 drop 表、重置卷、恢复旧全库覆盖新业务事实或执行 `down -v`。

重新启用时只恢复三表最小 grants；用 enabled union 做 serving preflight。安装前后原表计数、事实身份、原 grants 和命名卷须一致。本说明不是当前本地实例的执行授权；真实环境安装、更新、合并和用户试用仍需各自现有授权。开发自检与 controlled fixtures 不等于用户验收，真实模型、支付及 SHEIN 调用保持 NOT_RUN。
