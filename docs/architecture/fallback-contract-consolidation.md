# Fallback 合同收敛

准入状态 `IMPLEMENTATION_READY`，执行事项 [#1101](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1101)。独立架构复核同意图片范围，并将包装缺项通过已有人工备注表达归为 `IMPLEMENTATION_TEST`。同一 Delivery Batch 按图片、SHEIN、S3 和清理分提交，最终在一个 PR 评审。用户已批准收敛范围和统一交付。产品基线沿用 [全新系统开发基线](../product/greenfield-no-legacy-migration.md)。

## 图片恢复与商品来源

用户可继续查看原生成结果及结算证据。套图和 POD 的恢复命令只接受当前计划中有明确 `RecoverableEffects` 归属且与阻塞槽位、尝试序号及原因匹配的项。列表无效时拒绝推导。已有普通图片计划仍可使用其空列表加匹配 `Run.Block` 的合同。列表非空但没有目标项时，以列表为准。

`Service.RecoverEffect` 和 Temporal client 复用同一归属判定。恢复活动继续核对并处理原 effect，持久化和信号沿现有 fence 及 idempotency key。恢复协议仍使用原 `tenant + run + plan revision + slot + attempt`，原 action 重试沿原命令。UNKNOWN 继续保留，恢复不得重新生成、再次扣费或合成 effect。授权、跨企业边界、事务、状态转换、worker wire、资源上限和 deadline 沿 [Product Image Agent](product-image-agent.md) 和 [原产出恢复](vip-original-output-recovery.md)。

只读结算摘要仍可为合法空列表且原 `Run.Block` 与阻塞槽位严格匹配的历史投影查原始 `GenerationFact`，沿既有 intent、成员、报价和结算证明校验。它只说明已结算或已释放。套图页面从 API 的 `recoverableEffects` 呈现恢复动作，已结算且没有命令归属的原结果显示只读状态。最近任务摘要仅展示已证明的结算状态。页面沿 [#840 已准入的原产出恢复界面](vip-original-output-recovery.md) 复用现有区块与控件。

商品套图准备在解析完资产清单、生成配置快照或报价前，校验商品 ID 和有效商品版本与清单 `ProductContext` 精确对应，缺失和漂移拒绝。现有读取、确认和执行的原 owner 校验继续生效。POD 明确无源，沿现有空清单身份。旧清单的持久化字节与哈希算法继续用于原事实读取，商品套图的新准备没有无上下文准入。

验证使用现有域、Temporal、HTTP 和界面测试入口，覆盖当前有归属可恢复、缺归属或坏列表拒绝、普通计划路径、历史结算只读、商品版本缺失或漂移、POD 原无源身份。

## SHEIN 离线草稿

用户可生成并检查不完整的离线草稿。Catalog 未提供变体时保留实际商品、已批准素材及文案，草稿、预览和包内 SKU 列表为空，由原 `variants` 诊断提示缺规格。已有变体只使用各自的来源 SKU、规格和价格，缺少变体重量或尺寸时留空。离线包通过既有 `ReviewNotes` 记录一条包装缺项说明，保存诊断显示既有 `manual_notes` 通用提醒。发布载荷准备对诊断副本所做的默认值填充维持现有规则版本。

`catalog.ProjectCanonical` 和 ApprovedAsset 仍是来源，`draft.Builder` 构建离线包，现有 validator 和 Listing Record 负责诊断与保存。保存草稿仍可携带阻断项。现有平台提交策略、价格和图片整理规则、发布权限、报告绑定与 owner 校验沿 [离线包读取与诊断](report-center-shein-offline-package-source.md) 和 [Listing Record 合同](../../internal/listing/record/README.md)。离线包修改限定在新构造，同一幂等请求回读原已保存的包和诊断，严格解码仍接受既有格式。原事务及失败恢复保持原合同，离线构造没有平台写入。

验证空变体输入仍可构造与解码，原诊断报告缺规格，SKU 专属包装事实保留，商品级包装值不进入缺失 SKU 字段。逐项核对新字节与样例的预期差异。

## S3 与代码清理

S3 地址复用显式 `PublicBase` 和准确对象 key。缺配置或空 key 返回不可用。URL 推导与 S3 SDK 的传输 endpoint、寻址、对象校验和恢复分开，图片与来源素材入口继续要求显式配置。私有存储可以省略公开地址。删除只有测试引用的前端本地 JSON helper 及其路径 helper。SHEIN 固定词表沿现有匹配语义清理，移除正则编译失败时改用更宽匹配的分支。持久化包字段和仍有消费者的协议解析保留。

Legacy decision 为 `RETIRE`，移除未经事实授权的推导及无消费者的工具。此批范围限代码、设计、离线回归和 PR CI。运行配置、真实实例、数据迁移、真实模型及平台操作、合并和发布按各自授权执行。
