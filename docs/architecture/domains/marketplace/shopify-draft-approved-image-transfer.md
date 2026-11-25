# Shopify 草稿上传已采用商品图片

执行 [#1634](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1634)。Independent Architecture 增量，经独立增量复核达到 IMPLEMENTATION_READY。复用 [#1613 店铺授权、唯一创建目标及商品草稿合同](shopify-product-draft.md) 和 [企业商品图片合同](../ai/enterprise-grsai-product-image.md)。2026-11-25 用户授权在隔离环境修复直传，使用通用 Skill 发一件独立测试商品的带图六变体草稿。禁止重复创建原外套草稿 10419372589218。现有产品发布、订单、清除和正式环境不在范围内。

## 已知运行证据

当前 Store 本地授权依旧有效。隔离工作进程的应用令牌请求固定在 5 秒结束，原 DNS 路径和经核实的公开 Shopify 地址均重现。只读原协议探测分别在 5.708 秒和 5.539 秒获取 HTTP 200。后续 Shopify 确认店铺、同一安装和应用、开发店、币种、商品写入权限以及已有唯一私有元字段。故生产代码只对 Shopify 应用令牌申请设 10 秒有界客户端。继续服从已有上游 deadline、禁止跳转，一次申请失败后仍沿原业务流程反馈。安装、权限和 Admin API 的原检查条件保持有效。

## 业务、事实和授权

- 编辑者从当前商品的已批准图库选图，可为每个变体显式选择同一图库的一张图。沿现有 Shopify 售价表单、保存目标、准确核对和明确确认。绑定和图片顺序进入同一目标及摘要。没有显式变体配图时保持旧行为。拒绝空缺以外未选入图库的引用、不同版本和跨租户的库存。变体与图片不会从自由文本或颜色推断。
- Asset owner 持有商品、来源版本、采用清单及生成工件证据。Listing 继续读取当前库存、核对原目标和对应商品。选中的生成图通过当前 Image Agent 的 GeneratedMaterialReader 读取已有发布工件，核对准确对象键、租户、原任务、槽位、SHA-256。直接公开 URL 的既有产品媒体路径复用。生成图无法取得批准字节时停止，禁止把本机地址交给 Shopify。
- Listing → 既有 Store Center 实时核实同一企业、店铺、安装、代次、币种和权限 → Shopify adapter 暂存已批准字节 → 重核原业务资料与当前 Store 权限和 Shopify 安装 → Listing 原事务取一次性 SendPermit → 原 productSet 创建 DRAFT，并带同一图库对应的变体文件。最终令牌、回执、状态及幂等责任沿原合同。跨境数据只向该安装返回且限定为 Shopify 官方暂存端点，应用令牌仅发往已验证的 Shopify Admin 域名。
- 正式保存仍固定源图片身份及其内容摘要。供应商临时地址、签名、返回表单、字节及读取凭据不写持久状态或日志，不进入稳定创建意图。关闭业务模型与数据的旧路径保持旧 payload 形状。无需库迁移、新表、新权限、新后台任务或通用对象代理。
- 短期字节暂存只在 API 和 Worker 同时启用既有隔离生成地址、生成地址一致且经 Shopify 检查为开发店的组合注入。刊登和图片智能体使用同一有权的 Asset owner 和相同商品版本。合成隐私排空和真实清除组合均不接入暂存，这轮不将既有商品创建排空证据扩展成图片传输排空。复用现有生成工件 reader、应用配置和隔离店铺。禁止公共中转站、对 Shop 凭据放宽验证及其他试用的商品写入。

## 暂存与重试

- 只使用 Shopify 的 stagedUploadsCreate 和照片 multipart，与现有 Shopify productSet.files / variants.file。暂存仅使用实测的固定 Shopify 存储端点 https://shopify-staged-uploads.storage.googleapis.com。2026-11-25 已从原店官方接口验证根路径 POST、相同 host 的暂存引用及 Google V4 签名表单。要求精确文件名、签名表单 key 和返回引用路径一致。POST 目标和引用均做 HTTPS、准确 host、路径、返回表单及大小核对。拒绝非公网拨号、跳转、身份混淆、额外请求头和注册自定义 URL。临时传输使用独立无 Shopify token 的受限客户端。
- 提交一次暂存申请，对最多十张选中的生成图片使用受限并行，单张最多 3 MiB，整组最多 30 MiB。暂存必须在原一分钟发送租期内完成，暂存本身最多 22 秒，每张上传受 deadline 限制。取得最终写许可前重新核对原资料及授权。
- 暂存不创建商品、媒体文件条目或可售对象。失败或响应丢失不得进入 productSet，按原协议记录未发送。短期对象交由 Shopify 到期；下一次人工确认可创建新的短期暂存。已有退场和撤权语义不变。若进入 productSet 后结果不确定，继续使用原 Shopify 自定义 ID 查原草稿，禁止重发变体或草稿。已存在外套不作为本轮请求源。

## 验证

定向测试令牌上限、暂存响应安全、已批准生成图片来源与冻结索引、全组失败前置拒绝、已有公开图片、确切一次 productSet 以及已有查回合同。真实试用用新源商品与通用 Skill v3，通过正常工作台生成、采用、保存、送达，依 Shopify 后台刷新核对 DRAFT、六个规格、图库、变体图和工作台回执。未执行的检查标明状态。前后端用标准镜像构建，在现有隔离实例更新，不进入生产。

参考 [Shopify 暂存上传](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/stagedUploadsCreate) 与 [productSet](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/productSet)。

## #1644 新建草稿完整图组设计候选

[执行 Issue #1644](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1644) · Independent Architecture · **NOT_READY，待独立架构评审**。本节的候选参数和回读要求仅面向 #1644。上面的 #1634 独立准入和三图试用记录继续适用其原范围。代码核对基线为 `5edddae5834842e6c6d1aec4e654a2afa1405d93`。

### 用户结果与证据

商品运营从「商品刊登 → 待上传」选择一件商品和已连接的 Shopify 开发店。选中当前商品已采用的三色 12 张图，核对图库顺序和六个颜色／尺码规格各自的主图，保存资料后从原上传计划确认。一个原操作最多创建一件 Shopify 草稿。媒体处理完成后，从同一稿核实实际图片、顺序和六项关系。生成、采用与交付各有独立事实。此切片消费现有已采用图，范围限于新建草稿，既有草稿图库更新归 #1639。模型、计费、规格数量、商品发布、订单、正式部署、真实清除和历史迁移均在范围外。

| 原证据 | 已经证明 | #1644 仍需证明 |
| --- | --- | --- |
| [#1634 工作台上传](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1634#issuecomment-7153241534) | 开发店新稿 `10419475480738` 由工作台创建，后台刷新确认三张图、六个规格及配图；试用中一次暂存失败记为未发送，随后同商品有一个成功回执 | 一次创建中完整交付 12 张，且六项配图与冻结预览一致 |
| [#1637 分色图组](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1637#issuecomment-7154687005) | 工作台持久采用三色各四张；原外套 `10419372589218` 通过 Shopify 原生后台追加 12 张后共有 27 张 | 原生后台手工追加的记录仅作为采用和既有草稿证据。新建直传须从工作台验证 |
| [#1639 / Draft PR #1640](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1640) | 在既有外套草稿只读查到 12/12 原始字节摘要与 6/6 配图；原工作台只读候选见 PR 记录。PR 当前 `626c0cfcdb541bf73758c63790b6e3fe2fcee61e` 的合成同步写入与真实只读取证分别记录 | 原图摘要和媒体关系可以作为读取方法的证据。12 张一次创建后的排序、图库成员和变体关联仍需独立核验 |

### Shopify 2026-10 合同与代码边界

- [Admin client](../../../../internal/integration/shopify/admin.go) 实际调用 `/admin/api/2026-10/graphql.json`。Shopify 文档截至 2026-11-25 将该版本标为 latest。官方通用输入数组最多 250 项，单商品最多 250 个媒体。现有限制 10 张来自本应用，12 张低于上述平台公开边界。Shopify 还会按请求计算成本及节流，数组上限不代表同步处理时间、批次成功率或媒体可用时间。[API limits](https://shopify.dev/docs/api/usage/limits) · [商品媒体数量](https://help.shopify.com/en/manual/products/product-media/add-media)
- [现有暂存实现](../../../../internal/integration/shopify/staged_draft_images.go) 一次申请 `stagedUploadsCreate`，每张分别对已验证的 `url` 做带签名字段的 multipart POST，以 `resourceUrl` 作为文件来源。固定暂存主机、文件名／签名表单 key／引用路径核对和禁止重定向沿 #1634 实测合同。候选可核验完整返回并按唯一请求文件名恢复本地顺位，避免把目标数组或并发完成顺序当成图序；缺项、重复、签名与对象不符或任一上传出错均停在发送许可前。[stagedUploadsCreate](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/stagedUploadsCreate)
- [现有商品写入](../../../../internal/integration/shopify/products.go) 使用 `productSet(input:, synchronous: true)`，省略 `identifier`，写 `DRAFT` 与原私有来源字段。一次在 `ProductSetInput.files` 提交按预览顺序排列的完整图库，每个显式配图的 `variants[].file` 引用本请求图库里的同一文件输入。Shopify 要求每个变体文件也在产品 `files` 中。同步返回商品仍可能伴随媒体后台处理。按 ID 对已存在商品调用 `productSet` 会有列表替换语义，本切片仅用创建方式。[productSet](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/productSet) · [ProductSetInput](https://shopify.dev/docs/api/admin-graphql/2026-10/input-objects/ProductSetInput) · [ProductVariantSetInput](https://shopify.dev/docs/api/admin-graphql/2026-10/input-objects/ProductVariantSetInput)
- [源键查询](https://shopify.dev/docs/api/admin-graphql/2026-10/queries/productByIdentifier) 使用原 app-owned custom ID 与商品读取权限；已知商品凭原回执 GID 查询。[Product.media](https://shopify.dev/docs/api/admin-graphql/2026-10/objects/Product#field-Product.fields.media) 可按 `POSITION` 查询，[ProductVariant.media](https://shopify.dev/docs/api/admin-graphql/2026-10/objects/ProductVariant#field-ProductVariant.fields.media) 返回关联关系，[MediaImage](https://shopify.dev/docs/api/admin-graphql/2026-10/objects/MediaImage) 可提供处理状态和原图来源。未在官方 `productSet` / `files` 合同找到「输入位置即新建图库展示位置」保证。以实际 `POSITION` 读回对照预览，真实开发店验证前，顺序能否由单次创建满足仍是交付缺口。
- Shopify 提供另一次异步商品写入 `productReorderMedia`，也提供独立的文件创建和变体更新。本候选的产品路径只有现有暂存与一次原商品创建。如果准确回读证明图库顺序或变体关联无法由一次创建实现，则现方案不能完成当前 Must。补排序或补配图需另审同稿步骤的许可、远端并发变更、原响应丢失、查回与排空，当前候选不启用这些写入。[productReorderMedia](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/productReorderMedia) · [媒体管理指南](https://shopify.dev/docs/apps/build/product-merchandising/products-and-collections/manage-media)

代码需同时对齐四层。单改入口会在本地解析或读回再次截断：

| 层 | 当前实现 | 候选约束 |
| --- | --- | --- |
| 工作台 | [选图](../../../../web/listingkit-ui/src/components/workbench/listing/shopify-draft.tsx) 与 [请求／回执 schema](../../../../web/listingkit-ui/src/lib/contracts/supply-chain.ts) 各有 10 张限制；[草稿与上传预览 schema](../../../../web/listingkit-ui/src/lib/contracts/shopify-model.ts) 还限制 `images` 和 `imageIndex`；[计划详情](../../../../web/listingkit-ui/src/components/workbench/listing/upload-plan-panel.tsx) 已能列出图片与变体对应位置 | Web 的输入、保存结果、确认和回执统一读取同一 Web 上限，和 Go 合同用 12／13 张测试对齐。采用图刷新时，如已选项消失或映射失效，显式提示重新核对并阻止保存或确认，避免当前过滤把缺失项变成少图或无配图 |
| 域输入／观测 | [BuildDraft 与 ProductObservation](../../../../internal/marketplace/shopify/draft.go) 分别限制输入和媒体状态为 10；已按输入选择顺序保存图库，以一基索引保存规格图 | 新建草稿业务上限 12 张。保存的图序、资产引用及规格关系一致，索引按图库实际长度校验。商品身份回执与完整媒体对照使用各自有界证据 |
| 协议与暂存 | [draft input](../../../../internal/integration/shopify/draft_input.go) 的 10 同时影响暂存；变体引用同一图库位置的文件 | 与域的 12 张限制对齐，全组通过预算、签名、数量和身份校验才进入原创建。选择总量包含直接引用和暂存图。接口保留既有可公开读取图片路径 |
| 商品查回 | [products.go](../../../../internal/integration/shopify/products.go) 的创建响应、源键查回和已知 GID 查询只读取前 10 个状态，无 `pageInfo`、图库身份或变体关系 | 覆盖 12 张的结果摘要并显式识别不完整页。原商品身份合法时仍可展示一张稳定的创建回执，图库发生后续编辑或读取不完整则不给全组通过结论。专门的只读对照包含媒体 ID、顺位、就绪状态和规格关系 |

### 选择、发送与预算

- Product Asset 的同一企业、源商品和版本的正式采用清单是可选范围。Listing Target 保留用户选中的稳定资产 ID 和顺位，复用已有 `imageIds`／`variantImageIds` 与版本摘要。按采用记录提供的结构化颜色和图位展示整组，并让运营确认主图。当前 Supply 浏览器库存 schema 未保留 `presentation`；复用现有图片组库存的展示字段合同，从同一批准记录向选图页投影 `color`、`group` 和 `order`，不复制工件或靠文件名识别颜色。颜色、SKU 与图片关系以已有结构化事实及显式选择为据。黑、灰、白各四张的验收样本可按该顺序选择，三色主图若分别为图库第 1、5、9 位，则黑 S/M 都选 1，灰 S/M 都选 5，白 S/M 都选 9。具体顺位取自实际冻结预览。拒绝无权素材、重复项、超限或不在图库的配图，禁止截断。现有少图、无图、未单设配图仍按原规则工作。
- 工作台由原 Upload Plan 展示有序图片，规格显示 SKU、结构化选项、价格和对应主图。确认绑定原计划、来源、目标、采用库存与店铺版本。确认后 `UploadService` 读取已核对的 Target 与 Asset 库存，复用 `GeneratedMaterialReader` 检查原对象、字节和正式 SHA-256，再由 Store Center 验证成员、店铺、安装、代次、币种及权限。暂存引用只替换原冻结位置的传输 URL。完成暂存后重核原事实和 Store 绑定，再由 Listing Submission 在原事务取一次 `BeginShopifySend`，调用一次 `DraftWrite.Create`。
- 候选预算限制每张暂存 PNG 为 3 MiB，整组 12 张最多 36 MiB 原始字节。申请一个暂存批次、最多 12 次 POST。11–12 张暂存采用最多 4 个在途 POST，仍复用单次 10 秒网络上限及整批最多 22 秒（含申请和上传），实际以更早的父 deadline 为准。4 并发最多持有 4 份 multipart 图片正文，另有最多 36 MiB 已核验原字节，两者之外还有有界表单和运行开销。原 1–10 张维持 3 并发的预算。22 秒内无法完成四图一组的正常样本时须用实测调整候选预算，不能把多批或重试作为隐式降级。
- 整个发送准备继续服从原 60 秒租期和更早的父 deadline。12 图候选为最后一轮事实／权限复核、原发送许可和创建至少预留 20 秒，暂存截止取 `min(暂存开始+22 秒, 有效发送截止−20 秒)`，不足即停止。复用现有 10 秒令牌、5 秒 Admin 客户端和末次验证后短时 writer 一次性使用窗口；整个复核可能由多个调用组成，20 秒是候选预留量，实测应覆盖冷令牌、应用／定义查询和本地校验。权限验证、文件读取和最后创建耗时须在真实试用另行验证。到期或取消时停止等待和未发请求。若最终写许可已发放，按原协议保守留待核实，包括本地不能可信证明请求没有出站的情况。
- 暂存申请与临时 POST 沿当前隔离开发店装配；端点固定、拒绝跳转、无 Shopify 应用令牌随文件发送、临时 URL／表单／签名只留内存。短期对象由 Shopify 到期处理。读取失败、内容或权限漂移、某张上传失败及其响应丢失均阻止商品发送。只有原 Listing owner 持久确认 `NOT_SENT` 后，才允许用户重新核对确认；重复原确认按原结果重放。该判断针对商品创建，不能充作暂存请求已经退场的证据。沿 #1634 将图片暂存留在隔离开发店，隐私排空、清除和保留实例接线不继承本候选。

### 原操作与结果核对

`ShopifyDraftTarget`、`ShopifyDraftPayload` 和来源字段继续使用原企业、稳定店铺与源商品身份；保存的记录摘要和有序载荷指纹绑定原确认。签名引用不入原意图和结果摘要。相同确认、并发提交、成功后换本地资料版本或重新连接仍走原执行目标与已保存回执。发放创建许可后，HTTP 超时、断线、不完整 GraphQL 响应或本地未写成回执都由 Listing Submission 保留原尝试待核实。按同一安装可见性和原 custom ID 查回，正证据由原 `ResolveOfficial` 记录。查空、身份不符或访问失效只保留待核实。仅找到该商品不代表原 HTTP 请求已经退场，不据此解除既有排空约束。[原提交合同](shopify-product-draft.md) · [原结果核对](../../../../internal/app/listing/shopify_status.go)

创稿身份与图组交付各显示实际观察。取得有效 GID 后，沿当前有权刷新以该 GID 只读查询，无需另选远端商品或另发写入。对于 12 张生成图，以下为完整交付的最小证据：

1. 读同一已验证店铺的同一 `DRAFT` 的 `media(sortKey: POSITION, first: 12)`、变体列表和每个变体的媒体关系。图库上界 12、变体上界沿现有 30，单个变体只需判断是否有超过一个媒体关系；每个连接都检查 `pageInfo` 与边界，新的 12 图样本须有且仅有 12 个不同的目标媒体和六个对应变体。变体用结构化选项组合匹配，原规格有 SKU 时同时核对 SKU。
2. 由原 Target 的有序资产引用和冻结库存摘要取得正式 `GenerationEvidence.ArtifactHash`。查回时若原采用证据可用且摘要一致，复用 #1639 的受限 Shopify 原图读取方式，按原始字节 SHA-256 建立每张资产与唯一远端 MediaImage GID 的对应。沿原状态读取的一分钟截止，针对本稿最多读 12 个原图，单文件最多 3 MiB、整体最多 36 MiB，使用原固定 Shopify 主机、HTTPS、无跳转、无令牌及有限并发。原采用证据不可核对、原图域不符、重复字节无法唯一对应、媒体仍处理中或缺页，都保留需要核对。图库位置用媒体 GID 验证，规格主图也比较该图库里的 GID。签名地址、图片字节及凭据留在服务端，显示可来自当前已批准图及有权远端观察。无生成指纹的旧公开素材可继续使用原创建回执，只有得到等效的明确图像证据才可认定完整媒体匹配。
3. 结果页可显示「已创建 Shopify 草稿，图片处理中」或「草稿已创建，图片或配图需要核对」，媒体、顺序和全部映射均相符时再展示「12 张图片和 6 个规格配图已核对」。读取失败、媒体处理失败或图库顺序不符，可再查原稿，由原商品 ID 保持去重。媒体对照是受访问权限控制的当前读取结果，不重写创建回执，也不新增持久化状态、第二事实源、后台轮询或自动重发。退出或撤权后沿原 Store 可见性撤回当前观察。

### 与 #1639 共用位置及验证

#1640 的既有草稿流程保有其 Writer、PR、开发店和 E2E。它的最多 60 项现有图库观察与本项 12 张新建输入用途不同。可能需要双方对齐的是 `staged_draft_images.go` 的暂存安全校验／资源预算、[Store 商品权限接线](../../../../internal/app/storecenter/shopify_product.go) 与 #1640 的原图读取、`shopifyguard` 原发送护栏／回执存取、工作台刊登入口和 `supply-chain.ts` 合同。待 #1639 的可复用读取实现可消费时，经原 owner 对齐共用部分。合成证据可先复用现有 transport fixture 验证形状。既有草稿同步的操作键、允许写入步序、图库上界和增量变更权限独立保留。Legacy decision 为 N/A；无旧入口、迁移或第二事实源。

| 验证 | 使用现有入口及预期 | 当前阶段 |
| --- | --- | --- |
| 工作台、输入、冻结 | 复用 Shopify 编辑器和确认测试，核对同一批准组全部 12 张、预览顺序、六项映射（含大于 10 的合法配图索引）、第 13 张、素材或版本变化、从预览返回及重复点击。原三图测试作为回归 | NOT_RUN |
| 协议与有界暂存 | 复用 Admin transport fixture，核对一次暂存申请、12 次有界 POST、一条 `productSet` 的 12 个有序文件和六项同文件引用。模拟暂存重排、缺项、尾图超时、权限撤销、取消、截止时间不足，验证商品写入为零及原未发送结果。检查本地 reader、暂存和末次复核都在有效租期内 | NOT_RUN |
| 原键与媒体核对 | 复用 `TestPostgresShopifyDraftRetainsOriginalAttemptAndBlocksDuplicate` 和原应用 fixture。并发确认／一次创建响应丢失后按原来源键查；空读、错误店铺、权限失效均不得放开重传。正查只确认同一 GID。模拟 12 张和六项正确关系，以及媒体处理中、缺页、错序、同摘要歧义、错配并证明对应结果 | NOT_RUN |
| 开发店正常路径 | 单独获准后使用具有 12 张正式同源采用图、六个变体且从未在目标店成功创建的样本。在同一已授权开发安装和工作台，确认一次后读回同一新草稿实际顺序和六项关系。记录准确运行候选与预算实际耗时。不通过跨商品挪用原外套采用清单、绕过原创建目标或后台补图完成验证 | NOT_RUN，待开发店样本、授权及读回证据 |

设计评审核对三点。其一，原 60 秒租期内给 12 张暂存和末次复核的预算是否可验证。其二，单次创建后的真实 `POSITION` 和六项配图是否满足预览冻结合同。其三，真实试用能否合法消费已有同源采用素材，且创建目标此前没有成功或待核实记录。第二点可作为真实验证的发现项，出现真实差异后先按当前 Must 分类；第三点缺失时保留真实验证待办。合成证据只适用于准确候选。本阶段的授权限于设计与证据。
