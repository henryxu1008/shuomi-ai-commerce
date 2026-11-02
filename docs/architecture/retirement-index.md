# 按领域查找退休边界

返回[文档首页](../README.md) · [领域合同导航](topics.md) · [架构完整索引](README.md)

这里整理仍需引用的源码退休矩阵、KEEP / EXTRACT 映射、批准边界及原有效安全断言。
这些文档仍有当前约束作用，不能仅因标题含 `legacy` 或设计日期较早就当作普通过期计划删除。
每份文档中的旧 owner、数量、批准/失败与运行态结论仅对其原范围成立；目录分类不提升待确认设计，也不恢复旧兼容或迁移授权。

先读[Hard-Cut 规则](../refactoring/legacy-hard-cut-policy.md)和[Legacy Register](../refactoring/legacy-register.md)，
再按下面的分类找到具体矩阵。历史阶段报告与原始验证另见[归档索引](../archive/README.md)。

## 选择范围

- [总规则与收口](#overview)
- [采集、商品与 Store 来源](#product-source)
- [Agent、模型与图像](#agent-image)
- [平台刊登、店铺与 SDS](#platform)
- [身份、资源与计量](#identity-resource)
- [运行宿主、配置、发布与工具](#runtime)

<a id="overview"></a>

## 总规则与收口

只从当前 Hard-Cut / Register 和本组明列替代读取退出条件。

- [Compatibility Retirement](compatibility-retirement.md)
- [#29 / #301 源码退休收口核对与关闭边界设计](legacy-retirement-parent-closure-boundary.md)

<a id="product-source"></a>

## 采集、商品与 Store 来源

按具体路径核对 EXTRACT / RETIRE、原断言映射和有效当前 owner。

- [旧1688验证码试验与轨迹录制/回放完整退休闭包](legacy-1688-captcha-experiment-closure-retirement-boundary.md)
- [#301 旧 Amazon 来源适配层完整退休闭包设计](legacy-amazon-source-adapter-closure-retirement-boundary.md)
- [旧资产配方、来源发布包装与 OpenAI 迁移尾部完整退休边界](legacy-asset-source-openai-tail-closure-retirement-boundary.md)
- [#301 旧采集外观、兼容适配与诊断统计尾部完整退休边界](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md)
- [#301 旧 LocalAgent、SDS 商品源与孤立资产策略完整退休设计](legacy-localagent-source-policy-closure-retirement-boundary.md)
- [#301 旧数字采集与平台身份尾链完整退休边界](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)
- [旧数字内容配置、注入与持久化：完整退休闭包边界](legacy-numeric-content-management-closure-retirement-boundary.md)
- [#301 旧商品筛选与价格/库存推断策略完整退休闭包设计](legacy-product-policy-closure-retirement-boundary.md)
- [#301 旧商品来源归一化与中间事实合同完整退休设计](legacy-source-normalization-facts-closure-retirement-boundary.md)
- [#301 旧全局 Amazon 结果合同与 SHEIN 孤立属性 DTO 完整退休设计](legacy-source-result-contract-closure-retirement-boundary.md)
- [#301 旧采集 URL／站点便利接口与闲置超时管理完整退休边界](legacy-source-url-timeout-closure-retirement-boundary.md)
- [Store 旧来源准入与审计尾部完整退休边界](legacy-store-source-admission-audit-tail-retirement-boundary.md)

<a id="agent-image"></a>

## Agent、模型与图像

旧执行协议退出不取消当前预算、权限、effect、UNKNOWN 和恢复合同。

- [#301 旧 AI 路由与图像策略支线完整退休边界](legacy-ai-routing-image-policy-closure-retirement-boundary.md)
- [旧图片异步协议、GRSAI 通用执行与模型审核尾部完整退休设计](legacy-image-async-review-closure-retirement-boundary.md)
- [旧 ImageAgent 方法策略与历史快照兼容链完整退休设计](legacy-image-method-snapshot-closure-retirement-boundary.md)
- [旧 ImageAgent 业务 Signal 控制链完整退休设计](legacy-imageagent-business-signal-closure-retirement-boundary.md)
- [ImageAgent 旧控制、摘要合同与测试尾部完整退休边界](legacy-imageagent-control-summary-tail-retirement-boundary.md)
- [旧 ImageAgent 通用执行与 OpenAI 装配完整退休边界](legacy-imageagent-generic-execution-closure-retirement-boundary.md)
- [旧 ImageAgent 历史执行协议与便利合同完整退休设计](legacy-imageagent-history-protocol-closure-retirement-boundary.md)
- [#301 旧 ImageAgent 隔离 HTTP 与闲置启动装配完整退休设计](legacy-imageagent-http-runtime-closure-retirement-boundary.md)
- [旧 ImageAgent v2/v3 手工图像任务运行链完整退休设计](legacy-imageagent-manual-runtime-closure-retirement-boundary.md)
- [旧 ImageAgent V2 合同、批准链与默认装配残留完整退休设计](legacy-imageagent-v2-contract-tail-closure-retirement-boundary.md)
- [#301 旧生成链与审核候选合同完整退休设计（#953）](legacy-openai-adapter-closure-retirement-boundary.md)
- [旧 listing/task 与 OpenAI 便利调用链完整退休设计](legacy-task-openai-convenience-closure-retirement-boundary.md)
- [#301 旧租户 Prompt 管理与运行整链退休设计](legacy-tenant-prompt-closure-retirement-boundary.md)
- [#301 旧水印检测/去除与专属配置完整退休设计](legacy-watermark-config-closure-retirement-boundary.md)

<a id="platform"></a>

## 平台刊登、店铺与 SDS

旧数字 Task / Store / 平台编排与当前平台纯规则分别核对。

- [旧 Amazon 刊登整链与专属闭包退休边界](legacy-amazon-listing-closure-retirement-boundary.md)
- [旧 SHEIN/TEMU 自动核价链退休边界](legacy-auto-pricing-retirement-boundary.md)
- [旧刊登 CLI 与调度链退休边界设计](legacy-listing-cli-retirement-boundary.md)
- [旧 Listing control-plane 与数字任务派发/恢复链退休设计](legacy-listing-control-plane-retirement-boundary.md)
- [旧数字刊登运维源码：完整退休闭包设计](legacy-listing-operations-closure-retirement-boundary.md)
- [#301 旧 Task 预览与刊登素材辅助尾部完整退休设计](legacy-listing-preview-download-closure-retirement-boundary.md)
- [旧数字刊登共享管理与 Task/Workspace 尾部：完整退休边界](legacy-numeric-listing-support-closure-retirement-boundary.md)
- [旧 SHEIN/TEMU 数字商品与库存同步链退休边界](legacy-numeric-product-inventory-sync-retirement-boundary.md)
- [旧数字 Store 管理与刊登配置：完整退休闭包](legacy-numeric-store-management-closure-retirement-boundary.md)
- [#301 旧平台 API 调用实现完整退休闭包设计](legacy-platform-api-closure-retirement-boundary.md)
- [#301 旧 Publishing 公共层、闲置 SHEIN 策略与 JSON 尾部完整退休设计](legacy-publishing-support-policy-closure-retirement-boundary.md)
- [#301 旧 SDS 登录、商品同步与专属尾部完整退休设计](legacy-sds-runtime-closure-retirement-boundary.md)
- [旧 SHEIN 活动报名、促销与限时折扣编排退休边界](legacy-shein-activity-retirement-boundary.md)
- [#301 旧 SHEIN 品牌查询与错误分类整链退休设计](legacy-shein-brand-error-closure-retirement-boundary.md)
- [#301 旧 SHEIN 草稿装配包及专属扩展尾部完整退休设计](legacy-shein-draft-assembly-closure-retirement-boundary.md)
- [#301 旧 SHEIN 草稿 JSON 别名兼容整链退休设计](legacy-shein-draft-json-alias-retirement-v1.md)
- [#301 旧 SHEIN 草稿支撑层完整退休设计](legacy-shein-draft-support-closure-retirement-boundary.md)
- [旧 SHEIN 数字刊登整链：完整退休边界设计](legacy-shein-listing-closure-retirement-boundary.md)
- [独立 SHEIN 登录与身份回写完整退休设计](legacy-shein-login-identity-sync-closure-retirement-boundary.md)
- [旧 SHEIN 闲置策略与辅助合同完整退休闭包设计](legacy-shein-policy-contract-closure-retirement-boundary.md)
- [旧 SHEIN Publishing 运行适配、解析器与提交恢复完整闭包退休边界](legacy-shein-publishing-runtime-closure-retirement-boundary.md)
- [旧 SHEIN 敏感词整商品处理、反馈学习与词表管理完整退休闭包](legacy-shein-sensitiveword-closure-retirement-boundary.md)
- [旧 TEMU 批量重新上架链退休边界](legacy-temu-bulk-relist-retirement-boundary.md)
- [旧 TEMU 类目与模板查询编排退休边界设计](legacy-temu-category-template-retirement-boundary.md)
- [旧 TEMU 名称、描述与品牌清理编排退休边界设计](legacy-temu-content-handler-retirement-boundary.md)
- [旧 TEMU 变体筛选与 AI SKU 映射前置编排退休边界](legacy-temu-filter-mapping-retirement-boundary.md)
- [旧 TEMU 图片 API 与 vision prompt 残留退休边界设计](legacy-temu-image-api-prompt-retirement-boundary.md)
- [旧 TEMU 图片初始化、校验与并发上传编排退休边界设计](legacy-temu-image-orchestration-retirement-boundary.md)
- [剩余旧 TEMU 图片构建、标注与上传子链退休边界设计](legacy-temu-image-processing-retirement-boundary.md)
- [旧 TEMU 数字刊登剩余整链退休边界](legacy-temu-listing-closure-retirement-boundary.md)
- [旧 TEMU 刊登保存、提交与结果落库链退休边界](legacy-temu-listing-submit-retirement-boundary.md)
- [旧 TEMU 刊登前置查重、原始数据读取与缓存编排退休边界](legacy-temu-preflight-retirement-boundary.md)
- [旧 TEMU 默认属性处理管线完整闭包退休边界](legacy-temu-property-pipeline-retirement-boundary.md)
- [旧 TEMU SKU/SKC 构建链退休边界](legacy-temu-sku-skc-retirement-boundary.md)
- [旧 TEMU SPU 构建与最高零售价查询编排退休边界](legacy-temu-spu-price-query-retirement-boundary.md)
- [旧 TEMU 店铺上下文与商品初始化链退休边界](legacy-temu-store-init-retirement-boundary.md)
- [完整旧 TEMU 文本检查与要点闭包退休边界](legacy-temu-text-bullet-retirement-boundary.md)
- [旧 TEMU 变体读取、后处理与缓存编排退休边界](legacy-temu-variant-retirement-boundary.md)
- [#301 独立 SHEIN worker：旧数字账号、Store 与 Profile/Redis 完整退休闭包](shein-worker-numeric-persistence-retirement-v1.md)
- [Authorized empty Store observation retirement](store-observations-empty-retirement.md)

<a id="identity-resource"></a>

## 身份、资源与计量

既有身份、租户、账本和未决资源义务不因源码退休被推定已清理。

- [旧 ListingKit API 取 token、保存与认证诊断链：完整退休设计](legacy-api-token-tools-retirement-boundary.md)
- [#301 旧认证路由推断、权限尾部与 S3 对象操作完整退休设计](legacy-auth-route-s3-operation-closure-retirement-boundary.md)
- [#301 旧 HTTP 备用装配、白名单认证与辅助尾部完整退休设计](legacy-http-auth-support-closure-retirement-boundary.md)
- [#301 旧数字上下文与共享值合同完整退休设计](legacy-numeric-value-contract-closure-retirement-boundary.md)
- [旧资源通用事务与 Store 同库扣费链完整退休边界](legacy-resource-transaction-closure-retirement-boundary.md)
- [旧订阅、额度与影子计量完整退休设计](legacy-subscription-metering-closure-retirement-boundary.md)
- [#301 旧任务身份与数字映射合同完整退休闭包设计](legacy-task-identity-mapping-closure-retirement-boundary.md)
- [旧 Yudao 身份导入与 owner 恢复运维链：完整退休闭包](legacy-yudao-identity-operations-retirement-boundary.md)

<a id="runtime"></a>

## 运行宿主、配置、发布与工具

源码删除、保留配置、远端运行态与数据处理是不同边界。

- [旧浏览器预装与自下载整链退休设计](legacy-browser-install-closure-retirement-boundary.md)
- [#301 旧浏览器管理便利入口、配置旁路、实例诊断与闲置统计完整退休边界](legacy-browser-management-tail-closure-retirement-boundary.md)
- [#301 旧配置加载与校验门面完整退休闭包设计](legacy-config-facade-closure-retirement-boundary.md)
- [#301 旧配置别名、单项投影提交与 Dub 完整退休边界](legacy-config-projection-dub-closure-retirement-boundary.md)
- [旧全局配置与日志便利尾部完整退休闭包](legacy-global-config-logging-tail-closure-retirement-boundary.md)
- [#301 旧宿主配置管理、功能开关与 tracing 完整退休设计](legacy-host-config-featureflag-tracing-retirement-boundary.md)
- [#301 旧宿主、HTTP 与图像辅助完整退休闭包设计](legacy-host-image-helper-closure-retirement-boundary.md)
- [#301 旧宿主 HTTP、会话与运行辅助层完整退休设计](legacy-host-runtime-closure-retirement-boundary.md)
- [旧闲置 adapter/helper 完整退休闭包边界](legacy-idle-adapter-helper-closure-retirement-boundary.md)
- [#301 闲置 Gemini 客户端与宿主辅助入口完整退休设计](legacy-idle-provider-bootstrap-closure-retirement-boundary.md)
- [旧闲置配置、并行管理与 JSON 辅助合同完整退休闭包设计](legacy-idle-support-contract-closure-retirement-boundary.md)
- [五组闲置支持门面完整退休闭包](legacy-idle-support-facade-closure-retirement-boundary.md)
- [旧 Kubernetes API/Task 发布执行与 UI 专属接线：完整退休闭包](legacy-kubernetes-release-execution-closure-retirement-boundary.md)
- [旧 Kubernetes API/UI 发布身份与策略耦合：完整退休闭包](legacy-kubernetes-release-principal-retirement-boundary.md)
- [旧 ListingKit 上传配置与本地 UI 输入链完整退休设计](legacy-listingkit-input-config-closure-retirement-boundary.md)
- [旧 ListingKit 本地启动与 ImageAgent 验收编排：完整退休闭包](legacy-local-acceptance-closure-retirement-boundary.md)
- [#301 旧日志门面与全部消费接线完整退休设计](legacy-logger-facade-closure-retirement-boundary.md)
- [旧数字任务执行链：退休设计与持久化边界](legacy-numeric-task-execution-retirement-boundary.md)
- [旧数字任务持久化与恢复残留退休设计](legacy-numeric-task-persistence-retirement-boundary.md)
- [旧 ownership-controller 与 auto-shard 配置退休设计](legacy-ownership-controller-retirement-boundary.md)
- [#301 旧 RabbitMQ、数字 Store 代理运维及宿主配置/辅助尾部完整退休设计](legacy-queue-proxy-config-tail-closure-retirement-boundary.md)
- [#301 旧 DB 代理、Redis 封装与 shared helper 完整退休设计](legacy-redis-proxy-shared-closure-retirement-boundary.md)
- [旧 SHEIN／ListingKit 远程联调与专属镜像链完整退休设计](legacy-remote-debug-image-closure-retirement-boundary.md)
- [#301 恢复文件背后的旧调试与采集执行链退休设计](legacy-restored-debug-closure-retirement-boundary.md)
- [旧数字任务与刊登专属配置：完整退休闭包设计](legacy-runtime-config-closure-retirement-boundary.md)
- [#301 旧通用执行骨架与闲置宿主配置完整退休设计](legacy-runtime-skeleton-config-closure-retirement-boundary.md)
- [旧 schema CLI 与跨域聚合建表整链退休设计](legacy-schema-cli-owner-closure-retirement-boundary.md)
- [#301 旧日志控制、HTTP 无效选项与迁移状态尾部完整退休设计](legacy-support-control-tail-closure-retirement-boundary.md)
- [旧 Task 诊断与回归验收工具：完整退休闭包](legacy-task-diagnostics-retirement-boundary.md)
- [#301 旧工作池、SHEIN 敏感词文件尾部与恢复辅助层完整退休设计](legacy-workerpool-sensitiveword-recovery-closure-retirement-boundary.md)
- [发布身份预检与旧发布入口退休设计](release-identity-preflight-retirement.md)
