# Amazon Crawler Runtime Flow

#867 已按[用户确认的完整退休边界](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)退出旧数字 Task/HTTP Service、RawJSON cache/factory、regionGuard 与专属发布源。#901 按[完整来源适配层退休边界](legacy-amazon-source-adapter-closure-retirement-boundary.md)退出旧 integration/crawler/amazon 包、CreateProcessor 和孤立 Scraper 接口。本文仅描述保留源码路径，不代表运行环境或平台验收。

## 当前入口与 owner

当前应用在 `internal/app/runtime/currentapplication/runtime.go` 直接注入 `internal/integration/acquisition/amazon.New`，由其固定站点 client/parser 提供 Discover/Fetch 中立证据；`internal/product/dataacquisition` 消费这些证据，当前 Sourcing/Catalog/SRC/Collection 保存商品和 lineage。该业务路径不经过已退休的旧 Source/Processor 包装、CreateProcessor 或原浏览器池。

原浏览器调试入口 `hack/debug/test-amazon` 直接调用 `internal/crawler/amazon.NewAmazonProcessor` → `Process` → `Shutdown`。原处理器也保留单请求 `ProcessWithContext`、批请求 `ProcessBatchWithContext` → `BatchProcessor`，以及 `PoolManager` / `BrowserPool` 与 `InstanceProcessor` 的既有浏览器采集实现：

- `AmazonProcessor`：处理器生命周期、初始化错误、单个与批量请求及取消。
- `PoolManager` / `BrowserPool`：实例获取/归还、超时、风险阈值重建、代理、健康与池统计。
- `InstanceProcessor` / extractor：页面导航、目标邮编与货币、产品提取、质量校验和有限质量重试、失败证据。
- `qualityMetrics`：保留质量重试/恢复和目标上下文统计；既有 region normalization 与快照 map copy 归此消费者，旧 Service/数字 Task 指标不再存在。

`effectiveBrowserPoolSize` 仍由原 processor.go 持有并使用，`factory_test.go` 的两个有效池大小测试保持；删除旧 factory 不删除其原有效规则。HTTP 平台 API、域名/URL/产品检查、SPAPI 配置及当前有效 Queue/Retry/Redis 保持原 owner。本轮没有新建服务、结果缓存、恢复协议、当前业务库 owner 或远程调用。

## 持久化与运维边界

当前业务商品/lineage由 sourcing/Catalog/SRC owner 保存，不能恢复旧数字 Task 结果到 Redis/RawJSON 的第二事实源。旧 Dockerfile、Compose、push 脚本及 Kubernetes API/LB/监控声明已退出；没有替代发布命令。旧镜像、endpoint、缓存或监控对象的现实状态 UNKNOWN，本轮不查询或操作。

当前应用和独立 SHEIN 登录/Store 同步仍沿现有入口使用；源码退休不增加内部试用前置。当前 acquisition 的固定站点/解析/挑战/限制测试、原纯规则/处理器/批量/取消/浏览器池 mock 与共享 Sourcing/SourceIdentity/hash 测试继续验证保留行为；随旧 owner 退出的专属 source port/规划/回退/Envelope 包装测试不再保留。调试仍属于独立 hack/debug 模块，当前源码不代表该入口已部署；真实浏览器、provider与用户验收未执行。
