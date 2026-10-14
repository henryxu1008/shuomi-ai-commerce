# 自有商品单件转入与独立待补全入口 v1

Refs #694；录入、编辑与历史来源 #675；供应链与未绑定草稿 #605。
Design Basis: Reuse Existing Architecture。当前已批准合同可完整覆盖，未新增公共领域输入、事务、权限、事实 owner、schema 或恢复协议。

## 1. 用户结果与依据

2026-10-14 用户明确：“先明确单件转入和来源版本固定合同，再做自有商品到待补全草稿的入口，继续用独立页面，复用现有供应链和草稿能力。”

企业当前操作人在有效自有商品详情进入独立确认页，确认一件商品及其来源版本，转入后进入独立待补全编辑页，填写候选标题/描述并保存，刷新后继续编辑。原商品图片、属性、多SKU和0价格/库存准确查看。

依据：[供应链 §5/5.1](my-supply-chain-shein-v1.md)、[未绑定草稿](supply-unbound-completion-drafts.md)、[自有编辑](own-product-edit-v1.md)、[原产品决定](../product/supply-unbound-completion-drafts.md)。Figma沿 [Final UI/IA](../product/final-ui-ia-authority.md)，`tg48P46SSXl6TBy9lZwg63 / 31:463`，自有库3500:1100、待适配3507:1965/3143、待补全2667:1948。用户本次独立页面决定替代本路径弹窗；没有新增frame，不伪称已有，不增加一级中心。

## 2. 已有合同与 owner

- `collection.BatchSelectionInput.ItemIDs` 已支持精确子集，单件为 `[itemId]`；不是新建TransferItem owner。
- `collection.Service.ReadBatch/ReadItem` 均消费当前 actor-private owner 与实时权限。只增加ReadBatch的HTTP/BFF消费者，返回现有Batch DTO。
- `preparation.Service.Transfer` 沿原key、输入hash、scope和短效Collection proof；持久层在Product事务中锁batch，检查expectedRevision，读取选定成员并固定SourceItem。
- 自有编辑、成员增减、移动和归档在原Product事务内递增batch revision；Catalog正文不从客户端复制。准确来源为 `productKey/publicationId/source.version`，不用Catalog latest head。
- `preparation.SourceSelector` 读取已固定source与准确不可变Catalog版本，保留原owner/member及当前data/supply权限。既有来源在Collection后续移动/归档后按原retained-source合同读取。
- `completiondraft.Service` 唯一持有候选标题/描述、revision与命令回执；Source/Catalog仍持原正文，Preparation仍持固定来源，正式Target/Asset/Store/Submission不变。

调用链：Console详情 → 独立确认页 → Collection读 → 原Transfer → 原ListSources/SourceSelector → 独立CompletionDraft页 → 原Save/Read/ReadCommand/List。App仅装配，既有 opt-in `supplyCompletionDrafts` 及Preparation-only/full两种组合保持，不引入新的能力开关。

## 3. 确认与来源固定

确认页 `/workbench/data/mine/own/<itemId>/prepare`：

1. 读取当前item，仅确定其batchId和own类型。
2. 读取该batch，取得当前batch revision。
3. 再读item作为展示的确认资料；要求其batchId仍匹配、来源仍own。批次版本读取必须先于确认资料读取，不能把旧商品资料与后来取得的batch revision拼接。
4. 确认操作原样发送 `TransferInput{batchId,expectedRevision:batch.revision,itemIds:[itemId]}`。服务器不信任客户端的正文、productKey或任意来源版本；现事务锁batch并检查CAS后固定准确成员来源。

商品在步骤2之后被编辑、移动、归档，或同批其它成员变化，旧revision均由原CAS拒绝，用户重新读取后再次确认。变化发生在步骤2之前时，步骤3展示当前资料。变化发生在成功转入之后时，已有SourceItem/草稿保持原版本。接受原聚合合同的保守冲突：同批其它商品变化也需重新确认，不引入新细粒度锁或协议。

默认只转入当前声明版本；历史页面仍只读，不增加选择旧版本或自动升级草稿。UI显示实际固定source.version、原始资料与候选内容两个层级；0按现自有资料合同保留，缺少事实不伪造。

## 4. 两次明确操作与恢复

转入提交Preparation及一条SourceItem；成功回执的count必须1，source必须属于该preparation且collectionItemId匹配所选item。当前确认仍对应原input时，读回source tuple与collection revision还须匹配展示资料；恢复旧key时读取其原固定事实，不能和新确认混用。之后跳转 `/workbench/supply/mine/missing/<sourceId>` 独立编辑页，通过原SourceSelector重新读取固定资料。成功后source读取失败保留回执，只单独重读来源或从原待适配继续，不重复转入。

转入成功不声称草稿已保存，不自动执行Save。独立页显示“尚未保存”，候选文字默认来自固定来源；用户保存才由原CompletionDraft事务写入。离开/取消编辑只保留已转入事实，之后可从原待适配继续填写；不补偿删除Preparation。

同key同input重放原Preparation；不同input冲突。UNKNOWN沿 `useSupplyCommand` 保存原key/body/scope，核实原Transfers/by-key后读取该回执的固定source，不用当前item重造来源；pending时禁止新确认。页面只恢复属于当前item的单件transfer，其他pending操作不在本页执行。当前item后来移动/归档，不阻止沿原key核实既有成功结果；访问仍由原实时权限和member边界控制。

明确开始的新操作使用新key，可建立另一份独立Preparation/草稿；不新增跨key去重事实或静默替换旧草稿。CompletionDraft沿原revision CAS、同key旧record、异载荷冲突及UNKNOWN恢复。恢复后重新读取实际head，未保存输入不冒充保存结果。

## 5. 页面、边界与预算

商品详情入口仅在Collection数据与供应链/CompletionDraft能力已开放、当前read/manage权限具备时展示；服务端仍按原verified identity/live grants执行。独立编辑页沿同一能力与权限门禁，草稿列表及待适配本路径链接转到独立页，原query入口也使用相同独立页面内容，不保留弹窗实现。

复用OwnProductContents只读展示、现Console/Card/Input等theme tokens、原SupplyCommandFeedback和pending intent。SourceDetail没有提供创建时间时显示未提供，不编造商品日期。切换org/actor/item/source取消读取，keyed内容和命令不能迟到显示到新scope；当前member撤换及撤权继续由已有owner拒绝。

新Batch读取采用原GET严格body/query/no-store、10s和2MiB边界、现Batch JSON/schema；BFF和客户端核对响应id。转入、SourceDetail及草稿请求沿原10s domain/20s HTTP、2MiB读取、64KiB候选保存、title1024/description32768 UTF-8预算。无外部URL下载、AI或平台副作用。

## 6. 不变范围与验证

Out of Scope：新增表/权限模型/领域owner/状态机/恢复协调器、复制来源正文、自动绑定或发布、历史版本转入、自动草稿升级、平台字段编辑、AI/支付/provider、迁移/清理、通用验证工具、合并/关单/部署和真实数据操作。

Legacy decision: N/A。复用当前合格Collection/Preparation/CompletionDraft和Console，不调用RETIRE抽象，不增加兼容、双写或第二事实源。

必要TDD：Batch GET与BFF精确绑定；确认读顺序与改变batch后的冲突；单件请求、count/source校验、旧key恢复而非新item基准；独立页保存刷新/迟到响应/权限；原多SKU、图片、属性和0显示。复用已有真实PostgreSQL Transfer子集/CAS/重放及自有编辑后retained-source测试，新增一条实际own→Transfer→CompletionDraft→own编辑后保持原来源的相关回归；不扩展全量故障矩阵。

该Design Basis准入覆盖完整既有合同调用，Issue进入Ready后实现；最终候选运行适用CI和一次独立交付检查。若真实实现必须改变上述不变边界，停止该部分并升级Independent Architecture，不能在页面实现里发明协议。

交付已有正常启动方式及新入口，用户/生产验收不代签。现保留实例若尚未安装CompletionDraft，按 [原owner安装合同](../operations/supply-unbound-completion-drafts.md) 完成具体目标授权后才安装既有schema/minimal grants及更新serving；不运行非空库全量初始化，不改Store或其它会话环境，数据、图片、账本、命名卷保留。
