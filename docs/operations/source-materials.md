# 来源素材与资料检查

Refs #30 / #307 / PR #859。批准依据：[完整来源闭环 §1–§7](../architecture/source-business-closure-v1.md)、[Product Authority](../product/final-ui-ia-authority.md)。这是当前能力的可选装配说明，不是内部试用的新前置，也不改变 RUN-1 的验收范围。

## 启用与存储

普通入口仍为 `go run ./cmd/current-application -config <absolute-private-manifest>`。在任务/环境 owner 明确配置的当前 manifest 中启用已有 `productAcquisitionDatabase`、`productCollections: true`、`acquisitionPlans: true` 及公开 collector，并增加：

```json
{
  "sourceMaterials": {
    "assetDatabase": {
      "host": "127.0.0.1",
      "port": 5432,
      "user": "supply_asset_runtime",
      "password": "<privately-provisioned-password>",
      "database": "<current-asset-database>",
      "maxConnections": 2
    }
  }
}
```

该片段不是完整可运行 manifest；必须使用环境自己的身份、Product/Resource/collector 配置，不能复制占位值或读取既有凭据来拼装。缺省 `sourceMaterials` 时能力关闭。Asset 必须是当前唯一 owner；同时启用完整 SupplyChain/ImageAgent 时须指向同一 Asset 目标及配置，共用窄角色池。不能指向 Product、身份或计费库。

初始化继续由原 `cmd/supply-asset-init` 显式完成：私有绝对路径 `-dsn-file`、准确 `-confirm-database`，仅空库初始化时使用 `-install-empty-schema`。Product 仍沿现有采集初始化命令的 collections/acquisitionPlans 能力；本批不新增表或初始化命令。普通 serving 只验证 schema 和精确权限，不自动初始化或修复；不可变 assets/receipts 仍无 UPDATE/DELETE 权限。

UI 同时显式设置 `LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true`、`LISTINGKIT_ACQUISITION_PLANS_ENABLED=true`、`LISTINGKIT_SOURCE_MATERIALS_ENABLED=true`。与其它可选能力一样，UI flag 只选择入口，后端仍逐请求验证当前身份、企业、Collection owner 与 Preparation 权限。此配置不要求 Store、Agent、Temporal、图片生成或平台上传。

## 用户路径

1. 在 `/workbench/supply/acquisition` 用已有模式采集新来源；公开获取不要求源账号登录。成功结果按原 SRC/Catalog/Collection 事务保存，失败/UNKNOWN 沿原操作核实。
2. 到 `/workbench/data/mine` 查看已保存商品；刷新或重新打开仍读取该成员固定的 publication/version，来源、校验值、采集时间及警告来自已保存事实。查看原始商品保持只读。
3. 显式加入我的供应链后，在 `/workbench/supply/mine/waiting` 的来源条目点击「来源素材」。选择本次采用的完整图片集合及用途，再点击「确认采用所选图片」。勾选本身不批准图片。
4. 读到原确认回执后重读本版本 inventory 与 Product inputs；原来源 warning/review 仍显示。`本版本资料就绪` 仅表示商品资料输入检查通过，平台规则、店铺、图片远端字节、上传及发布未评估。

结果待核实期间保留原身份/企业/来源、key 和完整选择；「核实原确认」沿原 key/载荷安全重放，不换新 key。关闭面板不取消已提交动作。重新选图是新的明确批准；旧回执重放不会把后来集合改回旧集合。权限拒绝、未配置和依赖不可用不会被解释为未批准或 ready；切换身份/企业移除旧资料并中止旧读取。

商品与来源事实在 Product PostgreSQL，批准回执/图片集合在原 Asset PostgreSQL；浏览器只保留未完成操作意图，不是事实源。批准失败不删除已保存商品，诊断 GET 不执行采集、批准、收费或 Agent。

## 证据边界

准确实现 HEAD、原 PR CI、独立最终复核和任务自有 PostgreSQL/合成 fixture 结果维护在 #859。隔离代码/组合测试不是用户验收、真实身份或公开来源可用性证明；真实 collector 的 #848/#849 限制沿原 owner 记录。真实环境准备、公开来源请求、合并、部署与关单须分别授权。本轮没有这些操作，#301 保持 OPEN，独立 SHEIN 登录和 Store 同步保留。
