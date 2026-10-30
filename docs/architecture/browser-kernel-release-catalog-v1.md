# 平台浏览器内核发行目录 V1

2026-10-30。Issue #1013。Status: **IMPLEMENTATION_READY**。首轮独立检查绑定 c73b22421ac51c3601a19567c64a5be3f27446f1；同一 Reviewer 对跨端增量 3c9e47d328038cc917f8009af156662651341108 和桌面依据 eee5f60e30a30abf15aa03a282c3e98e4b82ea80 确认保持准入，无架构 BLOCKER。正式实现晚于这两次检查；分页/版本校验在实施层验证。

## 产品依据、范围与 owner

用户决定：内核由平台通过接口发行可用版本，普通用户只有选择功能。浏览器 PRODUCT-CONTRACT.md（文档提交2ecc218）是已确认产品方向与草案；本文件确认后端合同。遵守 PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08、现有 CurrentIdentity/平台权限/严格路由及单 Writer。

本批 Design Basis: **Independent Architecture**，因为新增平台发行事实、发布/下架状态及跨后端/原生边界。用户结果是 Native 登录后可选择兼容版本，原生凭 releaseId 获得可信固定制品元数据；维护者通过 API 发布、下架及设置推荐。后端无管理页面、不代理或下载大文件、不读上游 latest、不上传或实际发布制品。桌面 Go/页面由原浏览器聊天负责。

发行目录是新的窄 domain `internal/browserkernel`，全平台统一，不持有企业/Store 字段，不放入 Store、Membership、通知或商品库。元数据唯一事实 owner 为该 domain 的独立 PostgreSQL `browser_kernel` 数据库/schema，适配器在 `internal/integration/persistence/browserkernel`。复用 GORM/PostgreSQL事务、既有连接配置和schema-owner模式。制品仍在已有成熟存储/CDN/发布资产中；本 API 只记录平台确认的固定 HTTPS 资产引用，不成为另一套对象存储。

Legacy decision: N/A。复用合格基础设施，无迁移、旧 ID 映射、fallback、双写或旧 Service 包装。现有环境、数据、权限、EXE/内核和店铺全部不操作；本批只源码、必要离线/任务独立 PostgreSQL 夹具及一个 Draft PR，不合并/部署/实际写发行目录或下载/安装内核。

## 精确 HTTP 合同

所有响应是直接 JSON（无 data 包装）；`Cache-Control: no-store`。GET 禁止请求体和未知/重复 query。请求deadline15秒（含认证），JSON体最多16KiB、读取5秒，复用 httproute.DecodeJSON 的重复键/深度/未知字段拒绝。错误只返回 `{"code":"固定码"}`，不回显URL、输入、数据库错误或凭据。

| Method / path | 请求 | 成功 |
| --- | --- | --- |
| GET `/api/v1/browser/kernel-releases` | 必需 `os`, `arch`, `clientVersion`；可选 `cursor`, `limit` | 200 `{ "releases":[ReleaseSummary], "nextCursor":"" }` |
| GET `/api/v1/browser/kernel-releases/:releaseId/artifact` | 相同三个必需兼容query，无分页参数 | 200 `Artifact` |
| PUT `/api/v1/platform/browser/kernel-releases/:releaseId` | `Publish` JSON | 200 `{ "releaseId":"uuid", "state":"published|withdrawn" }` |
| POST `/api/v1/platform/browser/kernel-releases/:releaseId/withdraw` | 空体 | 同上，state=withdrawn |
| PUT `/api/v1/platform/browser/kernel-releases/:releaseId/recommendation` | `{ "recommended":true|false }`，字段必需 | 同上 |

`releaseId` 必须为规范小写UUID。`os` 仅 windows/linux/darwin，`arch` 仅 amd64/arm64；客户端传原生运行时 OS/arch。`clientVersion` 为不带v前缀的严格 SemVer（如1.8.1），使用已有 `golang.org/x/mod/semver` 比较；版本参数是兼容筛选，不是客户端远程证明或授权。显示版本允许 Chromium 四段版本，不作为 SemVer 解析。

`ReleaseSummary` 的字段固定为：`releaseId`, `version`, `os`, `arch`, `minClientVersion`, `maxClientVersionExclusive`（空串表示无上限）, `recommended`。目录不含地址、路径或发布者身份。

`Artifact` 为 Summary 相同字段加 `archiveFormat`（zip/tar.gz/tar.xz）、`size`（JSON整数、1..2147483648）、`sha256`（64位小写hex）、`downloadUrl`、`downloadOrigins`（1..8个规范HTTPS origin，取部署 artifactOrigins 的副本，不由 Publish 输入）。原生必须重新绑定当前 releaseId/OS/arch/兼容版本并校验大小/哈希；页面只拿 Summary、releaseId 和准备进度，不透传 Artifact。releaseId 即制品身份，不另设可变 artifactId。

`Publish` 包含：`version`, `os`, `arch`, `minClientVersion`, `maxClientVersionExclusive`, `archiveFormat`, `size`, `sha256`, `downloadUrl`；全部必需（上限字段允许空串）。推荐初始false，推荐修改走单独API；不存在隐式latest或自动推荐。版本字符串最多64字符，无控制字符/首尾空格。URL最多2048字符，仅HTTPS、无userinfo/query/fragment，origin精确命中部署提供的批准资产origin（1..8项，无通配），用稳定版本路径，不持久化会过期的预签URL。URL仅由平台发行身份提供，普通query/body不得替换。

分页limit默认50、1..100；按不可变UUID字典序升序。cursor是上页实际消费的最后UUID，不含租户/秘密。数据库每页最多limit+1行，按OS/arch/已发布筛选后，只消费前limit行并用SemVer筛兼容范围；额外行仅判断hasMore，不能消费或推进cursor。hasMore时nextCursor为最后实际消费行，否则空。因此允许本页releases为空但nextCursor非空。客户端继续读取直到nextCursor空；不把空中间页误认为目录结束。推荐不改变分页顺序，也不自动改写店铺绑定。SemVer先校验完整三段且无v前缀，再调用现有库比较；非空max必须严格大于min，允许合法预发行/build metadata。

错误：认证401沿当前middleware；写入权限403沿当前平台middleware。域校验400 `INVALID_REQUEST`；不存在/已下架/OS架构版本不匹配均404 `KERNEL_RELEASE_NOT_FOUND`；同ID不同固定载荷409 `KERNEL_RELEASE_CONFLICT`；下架后请求推荐409 `KERNEL_RELEASE_WITHDRAWN`；存储/未知内部错误503 `KERNEL_CATALOG_UNAVAILABLE`。平台写API不存在时404，永不以不可用库生成临时目录或内存成功。

## 权限和可信边界

读路由复用 `AuthPolicyCurrentIdentity` + OrganizationNone + 空permission：任一当前已认证身份可读，不需要企业上下文、Store grant、browser.use或平台权限。

发行身份定义为现有 `listingkit.platform_admin` 判定通过者；写路由复用 `AuthPolicyCurrentIdentityWithVerifiedRoles` + OrganizationNone + PermissionListingKitPlatformAdm（与既有全局通知/生态平台维护相同）。只消费已验证全局role或已配置全局账号/role，不接受企业admin/listingkit_admin、组织grant或伪造header替代。复用现有权限撤销/有效期边界，不新增本地认证或原子角色写承诺。不增加IAM角色/模块/授权分配，不改白名单。

backend不从制品URL取内容，因此无后端资产网络访问或URL代理。Artifact只经原生Go客户端读取；该端须用与API请求分离的下载客户端，不把API bearer/登录cookie传到CDN或重定向目标。初始URL与每次重定向都必须HTTPS、无userinfo/fragment且origin精确命中downloadOrigins；最多3次重定向，禁止降级或越界。为支持现成CDN/GitHub release assets，重定向目标允许临时签名query，但不得存入缓存回执、摘要或日志；原始downloadUrl仍禁止query。部署origin须为规范小写host、无默认443端口/路径/query/fragment，输出即精确比较值。即便允许源被替换为不同字节，原生仍须大小/SHA校验失败，不能登记可用缓存。这是完整性保护，不承诺上游availability。

配套桌面依据为 ant-browser/design/kernel-distribution/DESKTOP-IMPLEMENTATION-BASIS.md（2026-10-30）：原生复用严格公共API origin、登录generation及AuthorizeRequest，不带Store/组织上下文；只有releaseId准备入口，独立下载client，固定哈希缓存，归档路径/链接检查，不在identity lock内联网或拿browser锁；旧任意URL/本地目录入口在原生边界收口。普通下架禁止后续目录/元数据读取和新选择，不作安全kill/下载租约：已发出的metadata和已验证缓存不会远程失效，已有固定绑定不升级/删除，保存新选择须重新核对目录。跨端最终检查验证这些既定边界，不另造跨库事务。

Threat Must：普通用户/企业管理员不可发版；跨企业不形成另一目录owner；同ID不能替换二进制或兼容身份；下架不可通过重试恢复；请求输入不变成任意服务器访问；不泄露代理/登录/存储秘密；目录/GET不能成为店铺启动授权。Out of Scope：原生下载实现、制品构建/签名/上传、存储管理页面、审批平台、按企业发行策略、安全禁用/强制更新、在运行进程替换、商家登录及真实发行。既有全局平台授权时点是本轮复用边界，不新增上游权限CAS/恢复平台。

## 持久化、幂等和状态

schema `browser_kernel` 仅一张 `releases` 表：UUID PK；固定metadata列；recommended；published_at/by；withdrawn_at/by；recommendation_updated_at/by。原生目录及CDN不写此表；无客户端任意本地目录注册事实。发布者必须先在既有制品存储准备并验证资产，随后API只提交这些固定事实；后端不假称已验证真实包。

发布在单个PostgreSQL事务 INSERT ON CONFLICT DO NOTHING 后读取同ID行；首次写入含发布者/时间。完全相同固定字段重复发布返回既有行（包括已下架state），不同固定字段409。所有固定字段（含显示版本、兼容范围、URL和格式）发布后不可改。无重发上游副作用，无补偿、双写/outbox需求。

下架只从published变为withdrawn，同一SQL条件更新withdrawn_at/by并recommended=false；重复下架返回原结果，不覆写原审计时间/发布者。下架单向；同ID重发不恢复，新制品需新ID。推荐set只对未下架行更新bool和最后修改者/时间，同值重试不改审计；允许多个推荐（无唯一latest承诺）。推荐/下架串行锁同一行，使withdraw后不可能再recommend=true。已有店铺/运行内核和本机缓存完全不写。

唯一索引+事务处理并发同ID发布；DB runtime仅SELECT/INSERT以及三个状态/审计组的UPDATE列，无固定字段UPDATE、DELETE、DDL、TRUNCATE、grant option、表owner或角色继承。schema-owner显式 fresh-init CLI在独立库安装表并给预创建runtime role最小权限，不由runtime执行。元数据发布/撤销与审计字段同一行原子提交，不另造审核/命令平台。

连接失败/超时/提交响应丢失返回503，不宣称没有提交；维护者只可用相同releaseId和相同固定载荷重读/重提，服务端自然幂等；不自动下载、反向修改制品或切店铺。读取失败保持不可用，不回退upstream latest。重启/多进程依赖同一PostgreSQL事实与约束，无进程锁或文件替代owner。

## 装配与部署界限

新增可选 `browserKernel` manifest项：`database`（canonical DB browser_kernel、独立runtime user browser_kernel_runtime、pool<=4）、`artifactOrigins`。nil默认不创建pool、不挂路由；不改既有模块状态或企业32模块合同。配置存在但不完整/共享错误DB/pool/不合格schema权限时启动拒绝。currentapplication连接pool生命周期→WithBrowserKernelCatalog→domain/persistence→五条descriptor→当前精确route guard（新增五条仅在明确启用时），不关guard。

runtime只读验schema/权限，不建表、不调用初始化器。仅operator后续获准时用独立schema-owner CLI初始化新库；本轮不初始化任何现有库，不写真实发行目录，不启动新服务。不建立通用制品、发布工作流、同步器、下载代理或新存储服务。

## 实现验证和交付

TDD先用现有Go/httptest/临时PostgreSQL捕获：匹配和分页兼容过滤；GET无URL/拒绝替换URL；登录/平台/企业身份边界；相同发布幂等/不同载荷冲突/撤销后重提；推荐与撤销并发；超时错误映射；DB独立pool、schema最小权限/固定列不可改、真实持久化重开读取及并发；nil配置路由默认关闭，启用五条routes受原精确准入保护。

复用现有testcontainers integration fixture和schema-owner模式，所有数据仅合成且资源任务隔离；不碰保留库，不新增runner/故障平台。只运行相关必要检查和稳定候选CI；已有代码有效证据复用。完整用户跨端交付后做一次最终独立差异/合同复核，不能以CI或实现者报告签发产品/发行/商家验收。当前目标为一个Draft PR，真实API发行、CDN下载、原生安装/开关与部署保持NOT_RUN。
