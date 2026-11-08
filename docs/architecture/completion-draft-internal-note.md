# 待补全草稿内部备注

执行 Issue [#1168](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1168)。

Design Basis: Independent Architecture
Admission Status: IMPLEMENTATION_READY / FROZEN。独立评审确认沿用草稿 owner、事务、权限与投影边界，明确 Walmart 与历史文案恢复语义。

依据用户批准的最小方案和[现有待补全草稿合同](supply-unbound-completion-drafts.md)、[结构化候选合同](completion-draft-structured.md)、[版本历史](completion-draft-history.md)及[旧版本恢复](completion-draft-restore.md)。

运营人员在待补全草稿页的「内部备注」纯文本框记录待核实事项。允许留空、编辑和清除，沿用「保存待补全草稿」。备注仅在工作台草稿和历史可见。第一版上限为 1024 UTF-8 字节，与现有标题上限一致。平台商品内容、原始资料、规格件数字段和既有上传任务保持独立。

## 保存合同

Listing CompletionDraft 是唯一 owner。内部备注作为草稿内容的可选 internalNote 属性；空字符串等价于省略，空值序列化时省略以维持历史内容和命令摘要。使用既有 content_json 与命令事务，数据库 schema 无变更，旧版本无该字段时显示为空。现有读取/管理权限、组织/操作者/成员作用域、固定来源、expectedRevision、同键同载荷重放和 UNKNOWN 原命令核实保持有效。备注只按纯文本展示，不支持 HTML，拒绝非法 UTF-8 和 NUL，仍受原请求大小上限约束。

调用链复用 Workbench/BFF → completiondraft HTTP 契约 → Service → 现有 Repository。草稿页填写、保存、刷新。属性/SKU 编辑器从当前已保存版本原样带回备注。历史查看和比较展示备注差异。现有旧版恢复只恢复标题和描述（含既有描述图片引用），沿用当前备注，不展示旧备注为恢复目标。仅备注不同时不允许触发文案恢复。

## 平台投影

现有 SHEIN、TikTok、Temu、Walmart 的预览与请求组装只能逐字段读取已批准的候选标题、描述、结构化属性和 SKU。备注不在任何投影白名单内。平台适配、上传任务、资产和发布接口无新参数。已绑定到具体草稿版本的目标或任务不会随新备注版本改绑。新增备注不触发任何外部操作。

## 验收与范围

用合成草稿验保存后刷新、清除、属性页保留、历史无备注、旧文案恢复保留当前备注和同键回放。用专门的合成备注检查平台预览请求、预览结果、最终平台提交载荷均没有备注，并确认原来的允许字段仍能投影。复用现有测试和模拟平台。真实草稿、实际平台提交、部署与生产验收均不属于本次执行。Legacy decision: N/A。
