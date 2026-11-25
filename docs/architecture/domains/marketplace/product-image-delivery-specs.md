# 商品图片交付规格候选

[执行 Issue #1643](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1643) · Independent Architecture · **NOT_READY（待独立 Architecture Review）**。现状核对基线为 `5edddae5834842e6c6d1aec4e654a2afa1405d93`。

## 用户结果与首批规格

运营从商品当前采用的图片中选图和交付用途，查看实际输出，保存并下载指定版式。首个可接的目标是 Shopify 草稿图库。现有草稿路径已经支持为变体从同一图库显式选一张图，所以变体配图共用图库方图规格及原配图关系。第二个用途是单张竖版详情文件下载，覆盖非方图交付需求。依据 [#137 的阶段决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)、[商品图片 V1 的内容和位置合同](../ai/product-image-agent.md#71-商品级计划与图片级配方)、[#1634 的三图六变体证据](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1634#issuecomment-7153241534)和 [#1637 的整组采用证据](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1637#issuecomment-7154687005)。#1637 的 Shopify 追加经原生后台完成。

| 候选规格及版本 | 用途 | 精确输出 | 依据 |
| --- | --- | --- | --- |
| `shopify-gallery-square-v1` | Shopify 草稿图库的一张图，可作为变体显式引用的图库图 | 1024 × 1024，1:1，PNG，每张 ≤ 3,145,728 字节 | 延续当前生成结果的尺寸，适配仓库 [Shopify 已采用图片暂存合同](shopify-draft-approved-image-transfer.md)。1024 是本项目选择，PNG 和上限来自当前项目适配器。 |
| `product-detail-portrait-v1` | 通用商品详情单张下载 | 960 × 1280，3:4，JPEG，每张 ≤ 3,145,728 字节 | 项目候选。用可复核的竖版详情文件验证构图和转换。预览和下载显示「竖版详情图」，不展示平台合规字样。 |

当前 [SHEIN owner 的尺寸函数](../../../../internal/marketplace/shein/goods/image_requirements.go) 要求 type 7 宽高均大于 900 且为 3:4。960 × 1280 是满足这条仓库几何判定的例子。真实站点、类目、格式、数量、位置及其他规则继续沿该 owner 的完整校验。当前阶段无 SHEIN 交付承诺，也无需重新核定官方规则。两种规格均为单文件；[#1644](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1644) 负责新草稿整套 12 图，[#1639](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1639) 负责已有草稿同步。当前 10 图与传输预算沿 [本地适配器合同](shopify-draft-approved-image-transfer.md#暂存与重试)。模型原生尺寸、报价、平台写权限、图内多语言排版、任意尺寸编辑及新素材库不在本批。

## 现状与责任

- [生成报价](../../../../internal/imageagent/image_set.go)固定 1024 × 1024；[生成物规范化](../../../../internal/app/worker/imageagent/generation_output.go)只在适用通道处理方形 PNG；[SHEIN 生成目标准备](../../../../internal/app/listing/image_set_target.go)用 1024 方图检查生成可用的位置。画面生成、用于展示的内容图位、交付文件规格是三个独立事实。已有 `carousel/detail`、顺序、显式变体关联和 Marketplace 官方位置各沿原合同。平台视觉模板只描述视觉；晚选择交付规格不会修改已冻结的生成计划或报价。
- [Product Asset](../../../../internal/product/asset/image_set.go)拥有按企业、商品、来源版本和原批准回执绑定的采用集合。现 [GeneratedMaterialReader](../../../../internal/app/imageagent/material_objects.go) 校验发布对象键、原 Run/Plan/Slot/Attempt 以及生成摘要。它仅可用于读父图；修改原 `URL` 或 `GenerationEvidence.ArtifactHash` 来指向派生文件会使获准字节读取及原证据失真。
- 现有 Artifact 基础设施托管不可变原图，原 [PublishedAssetRef](../../../../internal/imageagent/artifact.go) 特指一次生成尝试的产物。Product Asset 的选择回执持有「这张父图采用哪份交付文件」的业务事实。原图沿现有 Artifact 读取。派生文件使用自己的私有引用和摘要，原生成/结算回执、对象路径和 Asset 均保留。首版成品落点见下方事务核对。
- 通用详情规格由 Product 图片合同提供。Shopify 交付规格作为 Shopify Marketplace owner 的本地约束投影，由既有 adapter 的发送校验共享传输限制。交付页面和保存接口消费同一版本规格引用，包含 ID、版本、用途、宽高、MIME、字节上限和当期适用规则摘要。UI 自身不维护规格副本。规格变化要求重新生成预览并确认当前交付，历史批准文件仍可凭原授权查看和下载。

## 加工与预览

1. 商品当前图片入口以原批准身份和图位准备单张交付。首批输入限定为当前可授权读回并验证摘要的已采用生成 PNG。普通 URL、未采用候选、新上传、其他格式或递归加工均不受理。按文件字节核对实际 PNG 格式和静态、正向、8 位 sRGB 适用输入，非规范文件以明确状态拒绝。候选输入上限 8 MiB、单边 4096、总像素 8,388,608，先限字节和解码头再完整解码。首版逐张加工，沿请求 deadline 和已有图片资源边界做有界并发，避免多张图同时全量解码。
2. 同比例完整画面保持原像素尺寸或等比缩小。异比例时给出「保留完整画面，补白边」和「裁切到所选比例」的有效预览，先选再确认。用图内现有像素缩放，裁切保留区域以源图像素计，补白区域以输出像素计。首版居中放置，整数取整后有奇数余量时将多出的一个像素留在右侧或下侧。所用矩形在界面预览中准确呈现。禁止拉伸和放大，只有尺寸可达且无需放大的方案才可选。例如现有 1024 × 1024 方图做 960 × 1280 竖图可缩到 960 × 960、上下各补 160 像素。透明素材转 JPEG 时同样预览白底合成并显式确认。成品按所选规格编码，保留原文件备查。
3. 候选实现复用仓库已有 Go `image/png`、`image/jpeg`、`image/draw` 和 `golang.org/x/image/draw`，复用现有图片识别入口读取头，输出必须重新完整解码。默认无损 PNG 至多进行一次约定压缩；JPEG 按 90、85、80 的候选顺序尝试，在下限以上取第一个合规结果以供预览。最终字节必须同时符合精确尺寸、比例、MIME 和字节上限。不达标时提示「文件超过大小限制，请换一张图」。避免静默降分辨率、更换格式或把预览的 JPEG 重新整体编码一次。
4. 预览显示原图和实际待交付文件，标明用途、构图处理、成品尺寸、格式和文件大小。服务端将预览绑定到输入摘要、父审批及版本、目标图位、规则版本、实际裁切/补边参数、工具版本与输出 SHA-256。确认先核对仍有权限且版本未变化；仅保存预览所绑定的字节。若临时预览失效，允许同版本同参数重算后核对预览摘要；摘要不同则重新预览。取消退出后可从原图片入口返回，原批准文件保持可读。

## 保存、读取与失败处理

| 阶段 | 责任与合同 |
| --- | --- |
| 准备和预览 | 复用商品采用入口的企业、成员、商品、来源版本和图位校验。服务端按有权读取的原回执解析父 Asset，拒绝客户端直接提供对象键、源 URL 或审批证明。注入现批准源读取能力、确定性加工和规格读取。下载需要相同边界的实时授权。 |
| 确认与存储 | 在原选择合同中增加可选的交付文件引用，最多为每种首批规格各一份。元数据包含父批准 action/Asset 和原始字节摘要、确认成员和商品有效版本、图位与规格/规则版本、裁切矩形/缩放/画布位置及颜色、编码参数及工具版本、成品字节引用/MIME/宽高/字节数/SHA-256。交付引用及参数进入原选择摘要和不可变回执；当前集合沿原整组提交及 expected-head CAS 更替。保留未编辑图位、顺序和原生成事实。 |
| 字节与回执一致性 | 首版建议将经过确认的成品字节、Asset 回执和集合更新放在同一个 Asset 事务。预览在有界请求内生成，不作外部持久化。资产提交响应丢失按原 action 读取回执，同键同载荷重放原事实，同键不同选择拒绝。并发且旧 head 的确认返回冲突。容量与权限决策见下文。 |
| 交付消费者 | 通用规格由有权下载入口读取精确回执所指字节并复核摘要及规格。Shopify 草稿计划冻结用户选中的交付引用、图库顺序和变体映射，发送前验证准确回执、字节和 Marketplace 当前规则，再走原开发店草稿暂存/发送许可。 |
| 副作用与恢复 | 加工/保存不调模型、不调用平台、不触发新增供应商费用。既有生成成功、费用、未知状态及原回执不变；Shopify 授权、确认、外部写入与 UNKNOWN 核实继续由 Store/Listing 既有 owner 负责。Legacy decision：N/A。 |

### 采用确认与私有字节

1. **原事务可复用。** [ImageSetService](../../../../internal/product/asset/image_set_service.go) 的 `Preview` 和 `Select` 固定确认摘要，先查同 action 收据，再校验当前集合头。[`commitApproval` 和 `ReadApprovalCommit`](../../../../internal/integration/persistence/product/asset/repository.go) 按企业/action 处理幂等，事务内写回执、锁定并核对 [整组采用 head 的 action 和 payload hash](../../../../internal/integration/persistence/product/asset/image_set.go)，再写批准行和新 head。并行的同键请求等待既有事务并比对 payload hash。回执读回核验按序的批准行和完整摘要。增加派生字段时需要同时纳入请求摘要、选择摘要和资产持久化的 canonical serializer 及深拷贝。无派生字段的历史记录保留原序列化和摘要。
2. **复用事务，新增窄的人工确认合同。** 当前 [imageSelectionModeAdmission](../../../../internal/app/httpapi/image_agent_execution_mode.go) 要求当前 actor 的 `ApprovingResult` 投影，生成选择的 [Skill 栅栏](../../../../internal/imageagent/assetpublication/image_set_candidate.go) 只处理新生成候选。运营加工已采用图可能没有该 Run 的当前操作者身份。候选是 Product Asset 的人工确认命令，复用 [现来源和成员读取](../../../../internal/app/imageagent/source_selection.go)、原商品图片编辑权限及事务，先核验企业/成员、准确商品/版本、当前集合和指定父回执，再绑定格式、预览输出摘要和确认人。实施前由权限 owner 明确该命令接到哪个现有人工编辑准入及提交时重核入口。父 Run/Slot/Attempt、`GenerationEvidence`、`URL` 和原采用证明均从原行继承。重选会产生新 Asset ID 并更新 `SelectionReceipt`，所以派生引用单独记录首次确认 action/Asset 及父回执，对相同来源与图位的重选保留原派生字节引用；来源/版式变化要求重新确认。ACK 丢失后，在实时读取权限下先按原 action 核实原回执，再决定是否需要重新预览。此核实不以当前 head 或工具版本为准。
3. **现 Artifact 能写准确字节，缺少安全清理接点。** [不可变对象端口](../../../../internal/imageagent/objectstore/durable.go) 与 [S3 bridge](../../../../internal/app/worker/imageagent/artifact_store_bridge.go) 提供有界读取、校验和不可变 PUT/COPY，没有与 Product Asset 提交互斥的删除授权。现 [生成 staging 生命周期](../ai/image-agent-recovery.md) 清理的是生成尝试的临时前缀；正式批准引用要求长期可读。[容量合同](../../enterprise-storage-capacity.md) 明确图片批准目前不提供安全 purge。对象先写后提交时，查不到回执可能是尚在提交、提交已回滚，或读回暂不可用。固定 TTL、一次查空、请求超时、原生成 staging 生命周期均不能区分这些情况，也没有防止晚到 PUT 的证明。
4. **最小候选是同库原子保存。** 首版每次确认最多新增一个 ≤ 3 MiB 成品。沿 Asset 数据库事务写一条受限私有二进制记录，以企业/原确认 action/Asset/规格定位；精确关联该次批准行，同事务保存规范元数据与 SHA-256。普通库存及原回执批量查询只加载元数据，独立有界读取准确成品；同图重选引用原记录，不复制历史字节。只有在提交前从内存中读或按固定参数重算、且摘要等于用户确认的预览时才写入。回滚连同成品字节一起撤回，同键重放不产生第二份字节，也就没有需要外部清扫的未获回执对象。该方案需新增同 owner 的字节持久化和最小角色授权，现有 [schema/权限核验](../../../../internal/integration/persistence/product/asset/source_runtime.go) 只覆盖当前批准表及 head。由 Asset/存储 owner 决定是否接受首批字节写入及相应容量、备份与恢复边界，并确保旧读取者和写入者能够处理派生元数据后才开放。若选择对象落点，须先明确能让写入、采用提交与回收互斥的原 owner 合同，并给出在途、失联与历史回执仍引用时的证明；目前没有可直接套用的执行路径。

### Shopify 草稿读取接点

1. **保存与确认。** 现 [BuildDraft](../../../../internal/marketplace/shopify/draft.go) 仅保存批准图片 ID 对应的图库位置、原 URL 和变体索引。[目标记录](../../../../internal/listing/record/target/target.go) 固定 `InventoryHash` 和保存输入，[PrepareShopifyDraftIntent](../../../../internal/listing/submission/shopify.go) 再固定该记录的 `RecordHash`，原 wire payload 摘要独立计算。最小增量是在 Shopify 目标输入为所选图库 ID 增加显式、可省略的成品选择，服务端用准确确认 action/Asset/规格版本/摘要和当前库存核验并冻结。成品元数据进入库存摘要和目标记录；其私有定位、字节和 Shopify 暂存 URL 不进入稳定 wire payload。后台重新读 `shopifyDraftFacts` 时核对同一记录、权限、商品、库存及变体次序。历史计划缺派生选择时沿原图路径执行，传输前不把后来新增的成品自动套到已确认的草稿。
2. **读取与复验。** [readShopifyDraftImages](../../../../internal/app/listing/shopify_draft_images.go) 当前公共 URL 优先，只有非公网项才走 `ShopifyGeneratedImages`。在公共 URL 分支前处理显式成品选择。由 Asset owner 的窄 reader 以已授权企业、商品/版本及已核对的库存行对照精确批准回执，验证派生和父图的来源链，再有界读取所指成品，完整解码验证尺寸、PNG、大小及摘要。父图的 `GeneratedMaterialReader` 原键校验继续适用于父图；成品专用读取按私有引用查字节。缺引用、错位、权限撤回、规则版本或字节不符时停止，不回退父图或公开 URL，也不重新加工。
3. **发送与失败。** 既有 [uploadShopify](../../../../internal/app/listing/shopify_upload.go) 在发送租期内从已复核库存读取，暂存前后重验事实及 [Store 当前成员、应用安装和权限](../../../../internal/app/storecenter/shopify_product.go)，经 [原发送许可](../../../../internal/listing/submission/tiktok_no_send.go) 才创建草稿。保留现流程，读取的每张成品仍按图库原位置替换为暂存引用，变体索引保持对应。需在现 [开发店接线](../../../../internal/app/httpapi/current_application.go) 配置独立的私有成品 reader，沿既有隔离条件和 [PNG 暂存限制](../../../../internal/integration/shopify/staged_draft_images.go)。已明确选择成品时，接线缺失或字节核验失败要在暂存前沿 `NoSendImageContentChanged` 收束；进入远端创建后的 UNKNOWN 按原草稿身份核实。#1639 的已有草稿路由和 #1644 的整套图数量为独立交付任务。

## 证据与准入

[本地可复现记录](product-image-delivery-specs-offline.md)包含完整解码后的尺寸、实际格式、字节数和 SHA-256。两条文件规格在合成输入上成立。相同尺寸的裁切与补白结果在构图上明显不同，支持预览和显式选择。离线结果不覆盖应用保存、读取和平台操作。

本次风险范围是跨企业/商品错用图片、未获授权读取成品、不可控解码资源、批准和交付字节错配、历史证据丢失，以及错误触发生图或远端操作。使用既有企业授权、原回执及版本栅栏、限额和输出校验处理。无新增 Accepted Risk。

原 [候选与文件复核](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1647#pullrequestreview-6057559631) 已完成。后续只读准备核对了上述事务、读取与授权入口，沿用同一份离线字节证据。正式实现前需决定成品落点及其容量/备份责任、确认新人工编辑命令复用的权限入口，并独立复核准确回执和 Shopify 显式读取的合同增量。实现阶段验证确认/下载、图位与变体绑定、权限撤回、规则与版本冲突、摘要错配、并行确认与提交应答丢失。真实平台和用户验收待对应明确授权及执行。
