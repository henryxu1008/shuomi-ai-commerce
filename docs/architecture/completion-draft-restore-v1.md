# 待补全草稿：恢复历史内容为新版本

## 产品结果与设计依据

2026-10-15 用户批准：先明确历史内容恢复、权限和并发冲突合同，再将旧版标题/描述恢复为新保存版本；独立页面，保留返回草稿、商品和转入批次，复用现有保存及幂等能力。

Design Basis: **Reuse Existing Architecture**。本功能是已批准历史精确读取与普通 Save 的有界 UI 组合，不新增恢复 API、领域命令、权限、事务、持久化事实或状态机。依据：[历史读取](completion-draft-history-v1.md)、[未绑定草稿及 Save](supply-unbound-completion-drafts.md)、[已有草稿及导航](own-product-existing-drafts-v1.md)、用户本次决定。用户结果是误改后可以预览旧内容，并明确保存为新版本。

Product/Figma Authority：[当前 UI/IA](../product/final-ui-ia-authority.md)，文件 tg48P46SSXl6TBy9lZwg63 / 31:463、待补全 2667:1948；本次独立页面决定沿用现有 ConsolePage、表格、按钮和主题 token，不新增一级菜单或宣称存在新 Figma frame。

Owner 仍为 CompletionDraft immutable records/head/commands、Preparation 固定 source、原权限及 Console/BFF。调用路径：独立恢复页 → 现有 read source / read current / read exact Rn → 现有 useSupplyCommand 与 save-completion-draft → 原 HTTP / CompletionDraft.Service.Save → 原同库事务 CAS / command receipt。无 Legacy 消费，Legacy decision: N/A。

## 恢复内容与显示

- 历史详情保持只读；有 supply.read 和 supply.manage 的用户可进入 `/workbench/supply/mine/missing/:sourceId/history/:revision/restore`。SSR 复用能力开关、UUID、规范正整数 Rn 和严格 preparation/item query 校验。
- 页面读取准确旧版、当前已保存 head 和固定 source。必须核对 sourceId、整个固定 source、draftId、准确旧 revision，以及导航 preparation/item 和 own 来源归属；任一不符、缺失或读取失败均不提供保存。
- 页面显示固定来源 Vn、当前保存 Rm、选中旧版 Rn，以及标题/描述逐字段当前值和将保存值。只采用旧版的两个精确字符串，保留空串、空白和换行；不提供恢复页面内的二次编辑。
- 仅允许 n < m；选中当前版本或两个字段均与当前相同，不追加冗余版本。相同内容提示无需恢复；普通编辑仍回原草稿。
- 用户明确点击“恢复为新版本”才调用普通 Save，载荷仅为 `{sourceId, expectedRevision: m, content: old.content}`。创建 R(m+1)，保留全部原 Rn、head 的正常历史、固定 Vn、商品及批次；不改变既有记录，不倒退 head。
- 这是普通候选内容保存，不新增 restoredFrom 等来源事实，不声称服务端执行了独立的历史回滚命令。成功仅在现有严格命令回执确认后显示实际保存的 Rk，并提供准确新版本、最新草稿、保存历史、原商品及原批次入口。

## 权限、并发与幂等

- 查看旧版继续沿原 supply/data read、org/actor/member 及 retained-source 权限；写入仍由原 Save 实时 supply.manage / source / owner 授权。仅查看用户没有恢复入口，直达恢复页不开放操作；权限撤销由原 API 拒绝，不以页面权限代替服务端权限。
- 以预览读到的当前 Rm 为 expectedRevision。其他窗口先保存后，原 Save 返回 REVISION_CONFLICT；不得自动更新 expectedRevision 或自动重试。页面锁住原确认，用户显式重新读取当前版本、复核差异后，才能再次确认。
- 同一次确认仍由原 useSupplyCommand 在现有浏览器 intent owner 保存 scope、key 和完整载荷；busy/pending/storage 未就绪时禁止新增确认。响应丢失、超时、COMMIT UNKNOWN、页面刷新或重新进入，沿原键核实/原载荷重试；不新建第二保存键、恢复器或 UNKNOWN 协议。
- 恢复页可能遇到其他供应链操作的全局 pending intent。仍用既有反馈核实，但只有同 source、同选中旧内容的普通保存回执可以显示本次内容已保存；其他操作完成后重新读取，不误报恢复成功。核实旧操作的回执允许是当前 head 之前的准确保存版本，不将其冒充最新 head。
- scope/source/revision/preparation/item 变化 remount + abort，读取失败和加载期间不展示旧可确认预览；同 scope 的 pending 命令不被页面切换丢弃。恢复本身不调用平台、模型或支付。

## 范围与验证

Must：准确旧内容、同草稿归属、当前权限、显式预览确认、新版本及原历史保留、CAS 冲突后显式重读、原键核实与幂等、上下文及迟到响应隔离。

Out of Scope：恢复 SKU/图片/属性/平台字段、跨草稿复制、来源 Vn 升级、自动恢复或自动覆盖、历史删除、独立恢复 provenance、新 schema/grants/公共 API/owner/事务/恢复协议、Store/AI/provider/发布、迁移/清理及验证平台。已批准历史读取的只读限制继续适用于原接口和详情；本次仅增加显式普通 Save 消费者。

验证采用 TDD：独立页/入口、内容及换行精确保留、跨 source/draft/navigation 拒绝、仅查看/撤权、当前或同内容不保存、竞争保存后冲突锁定及显式重读、原 intent 在重入后的核实/原键重试、scope/版本切换迟到隔离、SSR route guards。复用原 domain/真实 PostgreSQL Save CAS、幂等和 UNKNOWN 已有测试；未修改 owner 时不新建持久化协议或重写同等矩阵。运行相关 UI/API/保存回归、类型检查、构建、适用 CI 和一次最终独立差异检查。

准入：以上 Reuse Existing Architecture 不变量与依赖齐备，可进入 Ready 后实施；如实际需要改变写入合同/权限/持久化/恢复边界，停止该部分并升级 Independent Architecture。交付正常独立页入口与启动说明；own675/其他实例更新、业务数据写入、合并及 Issue 关闭另需对应授权。实现、CI 和独立代码检查不代签用户/生产验收。
