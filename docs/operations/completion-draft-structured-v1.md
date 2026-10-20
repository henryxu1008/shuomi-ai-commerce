# 编辑待补全草稿的商品属性和 SKU

Refs #718。范围与权限见 [产品决定](../product/completion-draft-structured-v1.md) 和 [冻结合同](../architecture/completion-draft-structured-v1.md)。开发自检、CI、独立检查均不等于用户或生产验收。

## 正常入口

启用原 ProductCollections / supplyCompletionDrafts 的当前应用，正常登录并选择企业，从已有待补全草稿选择“编辑商品属性和 SKU”。独立路由：

`/workbench/supply/mine/missing/<sourceId>/structure?preparation=<preparationId>&item=<ownItemId>`

需要原供应链查看权限；保存仍需管理权限和实时成员授权。来源及导航必须匹配。返回草稿、商品、批次和保存历史均使用页面实际读取的固定来源。

1. 页面显示固定 Vn 和当前保存 Rm。首次来源预填明确未保存，之后准确读回已保存候选，空属性不会被来源属性重新填入。
2. 添加/编辑/删除候选商品属性；现有 SKU 仅编辑币种、价格和库存，编码/规格/图片/行数不变。留空表示未填写，0 为明确零。源商品不变。
3. 保存完整候选快照追加 Rm+1，保留当前标题/描述。原文字页保存同样保留当前结构化内容，两个页面共享同一版本。
4. 冲突后先重新读取保存版本，再确认修改；页面不会自动合并或覆盖其他页的新资料。结果待核实时按原操作键核实/重试，不能另建操作绕过。
5. 历史详情只读；比较覆盖属性、币种、价格和库存，旧无结构化记录明确未保存，不补来源值。**历史恢复只采用标题/描述，保留当前候选属性、价格、库存**；相同文字不因其他字段不同而启用恢复。

价格/库存是未绑定候选资料；当前各平台“带入草稿”只采用文字，结构化字段没有自动成为平台资料、真实库存或发布依据。

## 现有运行方式和安装边界

后台沿当前 `cmd/current-application` 和已验证私有配置运行，能力与原三张 CompletionDraft 表/最小权限不变；此变更无 DDL、grants 或新服务。不要用全库初始化、重复 schema 安装或共享测试数据执行开发检查。

本地前端在 `web/listingkit-ui` 执行 `pnpm build` 后 `pnpm start`（或既有 `pnpm dev`）；沿现有 OIDC/current-application 接线和 `LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true`、`LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED=true`，不复制或公布凭据。

后续真实更新需要同候选后台和前端一起交付；旧严格 DTO/完整内容 hash 不设计混合兼容路径。写入结构化记录后不要直接退回旧二进制、忽略新字段或重算旧 hash；保留数据并修复前进。原文字记录和旧 pending 命令在新程序中保持原 hash/原键，不能把旧 pending 改写成当前 head 的内容。

候选属性最多256项、SKU最多256行、整个保存仍64KiB（包括 JSON 转义），文字和单值限制沿合同。超过预算时不能保存，先减少候选内容；来源超过SKU编辑上限仍可查看/编辑文字。IEEE signed zero规范为JSONB数值0，不做金额舍入/换算。

## 验证与限制

使用原 Go domain / HTTP / fresh PostgreSQL testcontainers 和 Vitest 测试，不接保留实例数据库。验证SKU成员、复制/hash/CAS/原键回执、旧记录、0/null、历史比较以及文字保存/恢复保留当前结构化内容。类型、生产 lint/build 和最终准确 HEAD CI另记PR。

本功能开发不授权更新 own675 或其他实例、写保留业务数据、迁移、清理、合并或关闭Issue。试用实例须在候选完成后另获用户具体授权；未实际执行的恢复写入与用户/生产验收保留NOT_RUN。
