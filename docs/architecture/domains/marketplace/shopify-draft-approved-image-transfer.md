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
