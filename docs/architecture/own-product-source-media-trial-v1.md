# 自有商品来源图片的本地隔离试用

Execution Issue: [#675](https://github.com/qq550723504/task-processor/issues/675)

Design Basis: **Independent Architecture**。Admission Status: **IMPLEMENTATION_READY / FROZEN**；2026-10-14独立Reviewer `own_entry_review`准入绑定design-only HEAD `728c7ab249daef41aae23d207aefc14afc88124b`，无未解决设计BLOCKER，先于生产实现。精确URL/配置与容量注入、production代理/TLS/loopback/无redirect、原同文件恢复、namespace启动复用归IMPLEMENTATION_TEST，在本轮实现及启用试用前收敛，不重开全局设计。

## 产品结果与依据

用户2026-10-14在自有商品独立页面、变体、通用属性和详情试用确认后，明确“可以继续”，继续图片上传与详情展示。当前保留实例缺少来源图片存储；现有 SourceMedia 要求公网HTTPS地址，现有 ImageAgent 隔离URL政策明确不授权来源图片，不能伪装公网域名或直接放宽通用下载校验。

本轮在已有独立试用实例上传JPEG/PNG，加入公共图片池，明确分配给SKU，保存后刷新并切换SKU回看准确图片。Product/Figma Authority沿[自有商品录入设计](own-product-entry-v1.md)、[Final UI/IA](../product/final-ui-ia-authority.md)及其已读Figma添加/自有库节点；不新增页面、模态框或布局，复用原上传区和图库。

不做：已保存原始资料编辑、平台发布/图片批准、AI、真实云供应商或付款、共享/生产实例改动、新schema/数据迁移、对象清理、通用本地文件服务或验收平台。只准备与保留当前独立实例、图片和测试商品；本轮新PR不继承#676合并授权。

## Owner与实际调用

继续使用原SourceMedia（实时身份/企业授权、内容hash、确定性object key、不可变PUT和内容读回）；S3 SDK/MinIO拥有对象；原storagecapacity ledger拥有容量namespace/预占/占用；原Source/Catalog/Collection UoW拥有商品证据/版本/引用。无第二事实源。

`Config.sourceMediaIsolatedTrial`（默认false、服务端私有配置）→ `ValidateSourceMediaStorage`与CoreConfig → 当前HTTP Collection SourceMedia装配（含原人工图片reader）→ `SourceMedia`的具体、仅来源图片试用URL政策。S3构造、容量装配及原scope权限继续通过原路径注入。政策不进入浏览器请求，不提供通用allow-private callback，不被通用HTTP下载器、ImageAgent生成图或平台适配消费。

现录入/详情图片沿原`/api/image-proxy`展示；生产UI拒绝localhost，dev或通用private开关不能作为本轮方案。单独默认关闭的服务器`IMAGE_PROXY_SOURCE_MEDIA_TRIAL_BASE`仅准入相同固定origin/bucket/key语法的原图片展示GET；保持原TLS校验/受信任CA、无Cookie/Authorization，地址解析只允许loopback，试用对象不接受redirect，仍只输出限长JPEG/PNG。其它URL仍走原allowlist与公网地址检查，不扩到生成图、任意私网或API请求。

## 新增边界与不变量

- 正式模式继续要求原公网HTTPS publicBase；通用`ValidatePublicHTTPSURL`、DNS/IP/redirect与网络下载校验完全不变。原`isolatedTrialGeneratedURLs`仍不授权SourceMedia。
- 试用必须显式开启`sourceMediaIsolatedTrial`，同时配置SourceMedia、ProductCollections、原Commercial/Money owner和`storageCapacity:true`。缺失/错误配置拒绝启动，不静默切换供应商或未计量模式。
- 试用publicBase只能是规范的`https://localhost:<明确端口>/source-media-assets/<有效固定bucket>`；S3 endpoint只能是规范的HTTP/HTTPS `127.0.0.1:<明确端口>`、无其它path/query/fragment/userinfo，artifactMode固定aws、path-style固定开启。不能与真实云endpoint混用。
- 新政策只接受服务端根据原org/actor/content hash构造的精确key：`product-source/<64位scope digest>/<64位content hash>`；返回URL必须逐字等于固定base加该key。上传前校验URL；成功PUT后按原对象读取验证大小/MIME/hash/图片内容，再回读相同URL和实时scope。任意URL、其它bucket/origin/port/key/query/编码路径均不能借此准入。
- 保留原3MiB/JPEG/PNG/像素限制、同内容重放、容量Reserve-before-PUT与ConfirmStored、UNKNOWN同文件/同key核实；不新增状态机、事务、重试或后台恢复。原商品保存事务与图片来源绑定不变。
- 本地匿名图片读取是原公开商品图片语义的本机范围：沿既有受信任localhost TLS与loopback namespace，只转发精确对象路径GET/HEAD到本项目MinIO，清除Cookie/Authorization，不开放目录、bucket列举、写入或控制台。使用原SDK与仓库已用的MinIO/mc版本；保留本项目独立对象/密钥命名卷，不访问其他项目存储。
- 试用URL只供本机图片展示；正式Source/Asset批准、适配和公网下载仍按各自原政策拒绝私网引用。本轮保存不承诺可发布。

## 持久化、启动和恢复

复用[企业容量批准合同](enterprise-storage-capacity-v1.md)：原商业owner已初始化容量表及runtime权限，先只读核实本独立实例的实际schema/ACL；不跑非空Product初始化、不重置商业价格/权益，不改权限或schema。正式S3仍按bucket Owner.ID登记；本机MinIO按下方已核实的provider deployment ID依据登记空`product-source/` prefix。后续启动沿原namespace登记，不因图片已保存重新要求空库。

容量/对象与商品UoW保持既有分离：上传成功而未保存商品，图片仍是已计量候选；重复上传同内容不产生新身份/双占用。不自行清理未引用文件。

原GET `SourceMedia.Read`只核实文件可读，不Confirm容量；核实成功不等于原Upload完成。原UI当前会在GET成功后清pending，这处消费与已批准恢复合同不一致，分类IMPLEMENTATION_TEST：核实成功仍保留原scope/hash/bytes意图，允许重新选择同一文件，沿原Upload重放确认后才清意图并将图片加入表单。不让GET写容量，不新建恢复协议、平台或后台扫描；正常一次Upload成功路径不增加操作。

记录启用前私有manifest，保留原镜像/数据库/命名卷。开始写容量namespace后不恢复`storageCapacity:false`、换空owner或删除账本。修复/重启保留同owner、true和对象卷；必要回退只可关闭新上传、仍保留容量true和全部事实，不回未计量writer。普通start不重复初始化，不删除数据。凭据不入Git/日志。

## Legacy与验证

Legacy decision: N/A；只消费当前SourceMedia/S3/容量/UoW，不包裹ImageAgent来源禁令、不引入RETIRE消费者或兼容路径。

TDD先捕获：默认模式拒绝localhost；仅正确显式试用配置准入；精确已读回对象URL成功，错origin/key/编码/query/flag/容量配置拒绝；生产图片代理只接纳固定来源对象、拒redirect/任意private目标；GET核实不丢Upload pending，同文件重放后容量恰好占一次；正式公网模式不受影响。复用原签名不可变S3读回与容量/权限测试，验证实际Core→HTTP消费，不建新runner。

正常浏览器完成两张不同图片上传、分配不同SKU、保存、刷新、切换准确图片，并保留此前六SKU/两属性商品。复用相关Go/frontend检查与稳定候选PR CI，最终独立复核仅覆盖本轮增量；实现者验证不代签用户/生产验收。

## 真实启动发现的MinIO身份BLOCKER增量

Admission Status: **REOPENED / REVIEW_PENDING**（仅本节新增依据；未修改部分沿原728c7ab准入）。50f507f06真实启动无法完成核心happy path：签名GetBucketAcl成功但Owner.ID为空，原容量合同拒绝；签名ACL/list/versioning响应也无deployment header。MinIO固定版本[官方ACL实现](https://raw.githubusercontent.com/minio/minio/RELEASE.2025-04-22T22-12-26Z/cmd/acl-handlers.go)返回dummy ACL，不能伪造owner、凭据hash、改匿名政策或跳过容量作为修复。

有界只读实测：本项目provider卷`.minio.sys/format.json.id`与原官方`mc admin info`的`deploymentID`同为非空规范UUID；后者是有签名认证的MinIO只读ServerInfo，不是S3响应header。增量仅为默认关闭、明确来源试用的物理实例身份准入，不建设通用provider平台、账本或恢复协议。

私有`sourceMediaIsolatedTrialDeploymentID`只与原sourceMediaIsolatedTrial一起使用，必填且规范非零UUID；正式模式出现该字段、trial缺字段或其它endpoint均拒绝。setup从本项目已有provider对象卷只读取得ID，首次pin进原runtime manifest；重配若已有pin与卷ID不等则拒绝，不能重pin掩盖实例替换。

复用官方MinIO madmin-go SDK的只读ServerInfo，沿相同loopback endpoint/现有本项目凭据、不用环境proxy、不redirect、保留正常TLS。构造API与HTTP容量meter前均比较认证返回deploymentID与pin；失败/空/不匹配拒绝。namespace=Digest(["isolated-source-minio",endpoint,region,bucket,deploymentID])，不依赖可轮换凭据；第一次仍要求空prefix、后续仍原Ensure/RequireNamespace。S3 aws/cos正式owner证明不增加空owner fallback。对象不可变PUT、内容读回、scope授权、容量/商品事实owner和schema全部不变。

调用：Config校验及Core字段→runtime prepareStorageCapacity、HTTP collection source-media容量接线→同一个来源专用resolver→原namespace仓储/meter。原生成图/Knowledge/Ecoservices继续原S3 resolver。只修改本次来源接线，避免把试用policy扩到其它消费者。

增量验证先RED：仅完整显式trial及规范pin准入；真实形状ServerInfo认证只读且正确ID成功、空/错/无权限/redirect拒绝；相同实例凭据变化namespace稳定；正式空ACL仍拒绝；runtime/HTTP消费同resolver；实际MinIO首次与已有图片重启都可用。该BLOCKER影响图片试用启动，不授权新schema/ACL或接受容量风险。

同时发现新main未开放Store模块与保留实例旧schema不一致。该模块无应用/观测/运行任务，不属于本轮自有商品结果；本项目私有配置关闭未开放Store模块并保留其数据库，产品owner与原六SKU商品不变。没有执行旧数据迁移/非空Product初始化。正常保留start采用分步--no-deps启动已准备服务，避免重新运行schema/bootstrap setup。
