# 百炼商品图片接线 V1

Execution: [#774](https://github.com/qq550723504/task-processor/issues/774)。Design Basis: **Independent Architecture**；Admission: **DESIGNING，等待适用独立检查**。本设计先于正式生产代码。

## 产品结果与依据

2026-10-18，用户要求查清保留实例图片执行未接入的原因，补齐必要接线，验证三个官方图片方法的选择入口、权限和明暗主题，保留已有数据与配置、不重复派工；随后明确“先使用百炼”。服务商选择替代原运行接入对 GRSAI 的假设，但不撤销点数、权限、人工确认、UNKNOWN 恢复或素材保存合同。

继续沿用 [完整图片 V1](product-image-agent-v1.md)、[官方方法 V1](official-skills-and-image-method-v1.md)、[企业智能体配置](organization-agent-configuration-v1.md) 及其 Figma `tg48P46SSXl6TBy9lZwg63` / `31:463` / 图片方法节点 `6861:1826`、`6861:1864`。不新增页面、技能类别或付费调用入口；本增量改变 provider 接线，选择器仍使用已批准设计。

使用者从已有合法商品素材上下文打开整套图片面板，选择标准商品套图、场景化商品展示或卖点详情图编排的准确版本；准备阶段显示百炼实际路由和明确点数，只有原人工确认才能开始生成。未完成配置时显示具体不可用原因，不伪造选择成功或执行可用。

Must：显式百炼路由；真实授权素材/准确方法进入原指纹；计划、报价、生成意图和实际 dispatch 的 provider/model/protocol 完全一致；原权限、企业凭据、点数及成员月限复验；一次派发与原 UNKNOWN/结果恢复；现存数据、配置、凭据、任务与卷保留。

Out of Scope：真实模型调用、另设点数账本或默认价格、跨企业/全局密钥 fallback、个人图片直接执行、新任务调度、异步 provider 协议、平台发布、供应链门控变更、技能创作助手、迁移/清理、通用 provider 框架或新验收工具。无新增 Accepted Risk。用户/生产验收继续 NOT_RUN。

## 当前根因与最小改动

保留实例启动清单没有 `imageAgent`，故完整 HTTP/Worker/Asset 未装配；Image DB 尚未安装当前运行表和企业图片凭据表；缺少私有 worker、图片存储和每张点数价格。原 source acquisition pool 已有必要的标题 proposal SELECT，不扩大该读取权限。

代码在 `generation_provider.go`、`generation_quote.go`、`generation_execution.go`、`generation_fact.go` 与 `image_set.go` 多处直接固定 GRSAI。把供应商身份写死在通用意图/报价校验里造成接线耦合，单改 URL 会被拒绝；逐处放开任意字符串又会削弱原准入。

最小处理是明确支持第二个有限路由配置，统一验证其 provider/model/protocol 元组，由受信配置元数据向原报价及意图传播。现有 GRSAI 路由仍是当前已批准能力，不能把其历史回执改写为百炼；本轮不设计运行数据兼容/迁移，不引入未知 provider 或自动 fallback。

## Owner 与接线

| 合同 / owner | 实现与注入 / 消费者 |
| --- | --- |
| 当前企业 credential owner | 沿原 `image_gpt_image_2` 准确企业配置槽；百炼明确 style、model、HTTPS base 和 credential reference，不能复制标题密钥或依赖宿主 CLI 密钥 |
| ImageSet / Generation | 原 quote、plan、intent、effect 表及 JSON 字段；受信 factory 元数据新增或携带明确路由元组，原 fingerprint、确认、worker revalidate、候选采用逐项匹配；无新表/状态机 |
| 百炼 HTTP 集成 | 新窄 adapter，实现既有 route-bound image generator；复用现有 ProductImage adapter、prompt、素材验证和 SSRF/有界读取能力；不建另一套商品图片工作流 |
| Resource / 成员月限 | 原 orgresource reserve / observe / settle / release 与 UNKNOWN owner；价格仍由维护配置明确提供，不从人民币价格自行换算 |
| 字节和 Asset | 原不可变 artifact store、GenerationSuccess observation、物化及 Product Asset owner；不另存“百炼任务”或正式图片事实 |
| 当前应用装配 | 沿原 FullImageSet HTTP/worker/配置/Asset/Temporal 端口，不启用完整 SupplyChain 来间接打开图片 |

当前实例只有已保存集合/Preparation 来源，没有单商品采集 operation。运行前须确认现有 full image source reader 能消费其合法原来源；若需把通用 Preparation 来源从完整平台 Supply worker 中拆出，只提取既有来源读取/当前 IAM 端口，保留现有 `supply` context、原 source ID/固定版本与来源权限，不创建假采集 operation、重复商品或第二事实源。若无法在这些现有合同内完成，保持该入口 NOT_RUN 并报告，不临时新增 source kind/发布或授权模型。

## 百炼协议边界

依据阿里云[万相2.7同步图像接口](https://help.aliyun.com/zh/model-studio/wan-image-generation-and-editing-api-reference) 和本地实际原文 `bailian-docs-llm-wiki/raw/model-api-reference/image-generation/wan-image-api-reference/wan-image-generation-and-editing-api-reference.md`（2026-10-18核对）。先采用 `wan2.7-image` 的同步图像编辑，固定每次 `n=1`、关闭组图、请求 `1024*1024`、无水印；不接入异步查询/重试协议。候选模型仅此一个，变更须明确更新受信路由配置与校验。

百炼 region 与 credential 必须来自同一获准业务空间。只允许官方 HTTPS host：`dashscope.aliyuncs.com`、`dashscope-intl.aliyuncs.com`，或精确业务空间的 `*.cn-beijing.maas.aliyuncs.com` / `*.ap-southeast-1.maas.aliyuncs.com`；不接受用户信息、query、fragment、非默认端口、模糊后缀或转发到另一 host。POST 路径固定 `/api/v1/services/aigc/multimodal-generation/generation`。重定向禁用、transport 不自动重发 POST、timeout 最多5分钟；不使用 CLI 子进程代替应用 provider。

将已授权并冻结 hash 的真实 JPEG/PNG 字节按 data URL 传入 `input.messages[].content`，与冻结 prompt 同请求。参考图数量、尺寸和媒体类型先按官方限制校验；提示词超过5000个 Unicode 字符时在派发前明确拒绝，不能让 provider 截断准确方法或静默丢参考图。请求体/响应体有界，不把正文、base64、Authorization 写入日志/错误/API/checkpoint。

响应先有界解码并检查 request_id、精确一张结果、公开 HTTPS 输出 URL；沿原观察回调持久化 request ID、结果摘要与私有结果定位，再使用原下载/媒体/1024尺寸验证和不可变存储。provider 对尺寸允许误差，应用仍按现有实际输出尺寸合同拒绝不符结果，不伪造1024元数据或暗中调用放大模型。

百炼按张计费，token usage 缺失不能伪造0 token；Generation 的固定每张点数与实际 provider usage 两种事实继续分开。适配器向原 receipt 传递可证实的调用/图片数量和 request ID，token 字段无证据则维持未知。价格/credential reference/route reference/配置版本变化使原新派发 revalidate 失败，不能把新凭据用于旧已冻结意图。

## 副作用与恢复

沿原 ImageSet confirm → 原准入回执 → worker → Resource reserve → 原 unique effect claim → provider POST → GenerationSuccess observation → 原下载/物化/采用路径。原 Skill 方法锁和来源/当前成员权限栅栏保持；未选方法继续原合同。

POST 交给 transport 后的超时、连接断开、响应丢失、不完整 JSON、无法证明结果的非2xx均不能自动重新提交；保留原 UNKNOWN 事实/预留与核实入口。明确派发前的参数/权限/预算失败不消耗调用；不能仅凭收到错误字符串宣称 provider 没有执行。成功观察已保存但下载/进程退出失败时只沿原结果 URL 恢复下载/物化，不再次生成。观察回调失败也不能重发 POST。相同键换 payload/route/method 与原 digest 冲突；确认/采用回执仍优先恢复。

不新增自动 provider 对账、跨库事务、补偿系统或新的 UNKNOWN owner。供方没有可证实的读回入口时保留人工核实，不能以“没查到”释放未知成本或自动重发。

## 保留实例与停止条件

本设计和隔离实现不授权重跑 retained initializer。原 Compose/.env/私有文件不覆盖；有配置值后采用新的受保护配置文件与最小 override，保留原回退。当前空 Image owner 的显式 schema/窄角色准备、对象存储和企业凭据录入要分别记录真实授权、空库条件和前后证据；现有业务 DB 不跑迁移/DDL，也不重装官方技能。缺少这些前置时不能让 current application 启动时补表或自动启用企业。

**仍待用户确定每张 AI 点数；百炼地域/业务空间与现有授权凭据须在原配置入口确认。** 未给出前保持新生成关闭，不写入样例价格、不把供应商单价当产品点数。保留实例实际三方法执行选择/计划/生成/采用如未真实执行，一律 NOT_RUN。不得通过新造商品或伪 Run 提供验收截图。

Legacy decision: N/A；本次消费当前 ImageSet/Generation/Skill/Resource/Asset 合同，不依赖或恢复 legacy task/worker、旧试验 overlay、历史迁移或第二事实源。

## 验证与准入

独立检查只针对本次新增百炼路由、副作用和上述接线边界；两轮停止规则及 finding 分类按 AGENTS，不重审未变化的完整图片架构。达到 IMPLEMENTATION_READY 后才正式实现。

TDD：先以受控 HTTP fixture 捕获百炼 route 被拒、元组错配、n/size/原图/prompt 丢失、重复POST或错误结果观察；再补最小 adapter/元数据传播。覆盖正式host准入、超界零派发、只发一次、scope/配置变更拒绝、成功观察先于下载、响应丢失UNKNOWN、无 token evidence 不造usage、准确 Skill 及原 GRSAI 路由回归。复用已有 ImageSet/Resource/Asset fixture，不新增 runner。

实际实例验证三个兼容官方版本可选且保留 ID/version/digest；个人图片、跨企业、失权/停用版本拒绝；两种主题和不使用方法路径；停在准备/确认前，不调用真实模型、不产生真实点数预留或新 Run。若准备需要写 Run 则本轮不点击准备，只验选择入口与权限。缺少合法当前上下文或配置则如实记录具体阻碍。

本批一个主要 PR，原会话/owned worktree 一个 Writer。设计检查、隔离测试、PR CI、retained runtime 和用户验收分别记录；不自动合并/关闭 Issue，不把旧 #795 CI 冒充本增量证据。
