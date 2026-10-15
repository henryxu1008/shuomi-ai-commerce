# 本地 MinIO 企业容量身份准入 V1

Execution Issue: [#659](https://github.com/qq550723504/task-processor/issues/659)

Design Basis: **Independent Architecture**。Admission Status: **DESIGN_REVIEW_REQUIRED**。
本稿为 [企业存储 V1](enterprise-storage-capacity-v1.md) 的有界增量；正式生产路径必须在独立评审明确 `IMPLEMENTATION_READY` 后修改。

## 1 用户结果、依据与范围

2026-10-15 用户明确授权：先补齐 MinIO 身份准入设计和正式评审，再复用现有 SDK 与部署身份校验修复；验证通过、满足启动条件后启用本地 35444 容量功能。保留现有数据、41444 素材试用和 42444 边界验证实例，不部署生产、不操作真实支付，不关闭 Issue。

结果：35444 当前企业管理员可沿原“资源与额度”查看真实容量；当前已启用的 Knowledge 上传/processor 与 Ecoservices 业务附件在同一原 Commercial 容量 owner 下计量。默认 5 GiB、50 GiB/19.90 元/30×24小时、超限/到期保留原数据、原权限/钱包/报价/订单合同全部不变。

当前产品与 Figma authority 继续采用企业存储 V1 第 2/3 节及其实际节点：资源页 1627:359、存储 5872:440、购买状态 5872:464、overlay 5875:437。本增量没有 UI、IA、命名、交互或新入口变化，不重画 Figma。

本次只覆盖两个已启用文件能力的显式本地 MinIO 身份和必要的原 owner 安装入口；不启用 SourceMedia、图片 Worker、模型或支付，不把源码合并、技术自检签为用户验收。本地启用允许使用经复核候选；PR 合并仍需相应授权。

用户本次决定替代旧文档对“历史业务实例一律拒绝”的表述：保留业务数据库不是迁移或回填的授权；只有**所有当前计量 prefix 全新/为空、原容量六表及容量订单为空、旧未计量 writer 已停止**时，可在原实例第一次启用。发现未计量文件即停止、反馈，不复制、搬迁、删对象或建立隐藏基线。

## 2 已复现的 BLOCKER 与 Threat Model

35444 当前 MinIO 固定版本 RELEASE.2025-04-22T22-12-26Z。原 runtime 仅对应 prefix GetObject/PutObject；两桶实物及原 SDK root 只读核对均为空、versioning 未启用。原 runtime bucket 准入返回 403；root GetBucketAcl 成功但 Owner.ID 为空，通用 resolver 仍拒绝。MinIO 拒绝 IAM action `s3:GetBucketAcl`；其 [ACL handler](https://github.com/minio/minio/blob/RELEASE.2025-04-22T22-12-26Z/cmd/acl-handlers.go#L109-L152) 返回 dummy ACL，不提供 canonical owner，不能把扩大 ACL 权限作为修复。

Finding classification: **BLOCKER**。受影响 Must：真实物理存储身份 + 空 namespace 准入。命中“核心 happy path 按当前设计无法完成”，仅重新打开本增量设计；不重审已冻结计量、购买或恢复合同。

Must 防护：错误/空/匿名身份、部署更换、配置拼写错误、凭据撤销、HTTP redirect 或异常响应不得登记 namespace；云 provider 不得降级到 MinIO；不能以凭据或 endpoint 四元组替代物理身份；不能损坏原文件、旧权益、Money 或其他表/ACL；切换期间不得有未计量 writer。

当前边界：私密运行配置和本机 Docker/安装管理员为原可信部署边界，终端用户不能设置 endpoint、pin 或 namespace。仅信该边界明确选择的本地 MinIO，authenticated ServerInfo 的稳定 DeploymentID 是此部署的身份；不把它宣传为 AWS canonical account ID。外部/生产 MinIO、恶意宿主/服务端、云 IAM 改造、任意网络 discovery、通用多存储框架不在本轮范围。现有 operator 拥有的本机存储管理员权限不交给应用。

Should：共享现有签名 SDK/有界 transport，保持配置和执行路径小且可审查。现阶段不追加全量故障矩阵、验收 runner 或第二计量平台。

## 3 显式 provider 合同与身份

在 KnowledgeStorageConfig（Knowledge/Ecoservices 共用）新增两个可选私密配置字段：

```json
{"capacityIdentityProvider":"minio-deployment","capacityMinIODeploymentID":"<trusted installation UUID>"}
```

两个字段必须成对配置。provider 未指定且 pin 为空时完全沿原 AWS/COS GetBucketAcl owner 路径，identity hash 不变；未知 provider、pin 单独出现、错误/全零/非 canonical UUID 均拒绝配置。新显式类型只支持 `mode=aws`、path-style、本地 canonical endpoint；不通过探测 403/空 Owner.ID 自动 fallback。不得给 Knowledge/Ecoservices 借用 source-only isolated-trial 开关。

本地 endpoint 只允许 HTTP/HTTPS、明确合法端口、无 userinfo/path/query/fragment、严格 canonical URL；host 必须是 loopback IP 或原 Compose 的 `knowledge-objects` / `ecoservices-objects` 两个固定服务名。不接受任意公网/私网 host、localhost 别名、隐式端口、URL 规范化绕过或代理。两服务名沿现有内部 Docker network 解析，不新增公网监听或证书信任。

trusted operator 用现有 mc/MinIO SDK 的原管理员只读 ServerInfo 获取 pin，私密交接后由运行 manifest 持有。应用每次启动用**原窄 runtime 凭据**经现有 `madmin-go/v3` SDK 签名执行 `GET /minio/admin/v3/info`，只消费返回 DeploymentID，必须与 pin 完全一致。沿原 source trial 提取共享有界读取行为：固定 origin/path/method，无 cookie/redirect/proxy；上下文最多 10 秒且不扩 caller deadline，响应最多 1 MiB，非 200、超界、无法解码、缺少/不匹配 ID 均失败。继续用 SDK 的签名/解析，不实现认证或 MinIO 协议。

身份为 `Digest(JSON(["local-minio-deployment-v1", endpoint, region, bucket, deploymentID]))`。tag 与原 cloud/source-trial identity 明确分离，不改变旧实例已登记 hash。region/bucket 继续为 trusted config 中的非空、合法值。凭据轮换仍需重新签名验证同一部署，但不改变 namespace；实际部署 ID 变更而 pin 未变时失败，不能静默接受新部署。经明确部署操作换 pin 后按新 namespace 原首次空 prefix 准入；非空不允许回填。

本次原 namespace 表继续存不可逆 digest 与 prefix，无新表、字段或事实 owner。原 SourceMedia trial 的 UUID/loopback/隔离配置约束、identity tag 和 PublicBase/URL 合同不变，只复用其合格 SDK/transport 行为。当前 Image/SourceMedia 云路径不增加此配置消费端，未配置的 writer 不提前实现。

## 4 原 owner、接线与权限

| Contract | Implementation | Injection | Consumer |
| --- | --- | --- | --- |
| 显式本地部署 identity 解析 | `integration/s3`，复用 madmin SDK/有界 info transport | 原 currentapplication 的 Knowledge/Ecoservices 私密存储字段 | 原 prepareStorageCapacity |
| 空 prefix/版本检查 | 原 VerifyEmptyCapacityPrefix，AWS SDK | 原 bucket/prefix/runtime credential | 原 EnsureNamespace 首次准入 |
| namespace/占用/预占/lot | 原 storagecapacity repository + Commercial DB | 原 Meter | 原 Knowledge/Ecoservices，不新增生命周期 owner |
| 小范围容量启用安装 | 原 commercial-owner-schema-migrate 的专用 flag | 原 Commercial schema-owner config | 原六容量表 GRANT 与原 InitialStorageOffer |

MinIO runtime 只新增 `admin:ServerInfo`（只读部署信息，Resource `*` 为此 API 的供应商语义）、对应 bucket 的 `s3:GetBucketVersioning`，以及限定原计量 prefix 的 `s3:ListBucket` / `s3:ListBucketVersions`；原 GetObject/PutObject policy 和 credentials 保留。不授予 admin:*、IAM mutation、桶 ACL 写、删除或其它 bucket 的对象读写。ServerInfo 元数据仅在可信本机进程内消费，不投影到业务 API、浏览器或日志。依据：[固定版本 ServerInfo handler](https://github.com/minio/minio/blob/RELEASE.2025-04-22T22-12-26Z/cmd/admin-handlers.go#L2816-L2839)。

原商业 schema 已完整，runtime 当前只有六容量表 SELECT、无扩容 offer。全量初始化会消费 Money/OrgResource 并改动无关授权，不允许在本次保留库重跑。沿原 CLI 新增 `-storage-enable-empty-only`，只接受原 `-config`，拒绝 money-config（包括显式空值）、其它安装 mode 与位置参数。

此入口只打开原 Commercial DB，以原 schema owner 在一个最多 1 分钟事务内操作；固定 public search_path，5秒锁等待、30秒语句。先对六容量表与四张相关商业表取得 SHARE ROW EXCLUSIVE 锁；要求全部现有且原 owner/模型匹配、七个容量列已存在，不做 DDL/创建角色。复用原 qualified schema 检查及 disabled guard：六表必须为空且不能有任何容量订单。任意部分/不同模型/额外约束或字段缺失拒绝，不自动修复。

同一事务仅给既有 commercial_owner_runtime 六表 SELECT/INSERT/UPDATE，调用原 `CreateOfferIfAbsent(InitialStorageOffer())` 安装 50GiB/1990分/2592000秒 offer；不覆盖已有 owner 管理的有效价格/禁用状态，不操作 Money、其他商业表事实或既有角色授权。失败整事务回滚；原价格/offer 冲突沿现有 owner 拒绝。重复执行在仍无容量事实/订单时核对，不重复安装；有任何容量事实后入口拒绝，正常重启不运行安装命令。无 DELETE/TRUNCATE 新授权，不撤销旧 ACL。

该 flag 是本次启用缺少的原 owner 最小安装入口，不是验证 runner、迁移平台或新事实 owner。真实安装只按用户本次本地授权执行；生产安装保持未授权。

## 5 状态、事务、副作用与恢复

本次不新增业务状态机/订单状态、写入恢复协议、幂等身份或释放逻辑。MinIO identity 与原空 prefix 检查均在 SQL 事务外，身份验证失败不得产生 namespace；数据库故障不改存储/IAM。容量安装的六表授权与 offer 共用一个 Commercial 事务；MinIO policy 安装与配置/启动是原部署操作，不冒充跨库原子事务。

操作顺序：记录准确代码/评审、备份原配置/Commercial、快照原 IAM/业务事实/容器/对象；先只读解析两个 trusted 部署并验证空目录和所有 writer；经评审候选用原 SDK 验证已补的最小 runtime IAM；运行受限容量安装并验证原 runtime 权限。只短暂停止原 current-application（其 Knowledge processor 与附件端均随之停止），不停止其它实例/数据库/对象存储；复查全部 prefix 为空、部署匹配，无其它未计量 writer。仅在原 manifest 增加 storageCapacity=true 和两个存储 identity 字段，保留其它配置，使用包含现行 serving 改动的经复核候选启动；不替换 UI、不运行 Compose init dependencies、不扩大业务能力。

启动沿原 EnsureNamespace 登记原两个 prefix，随后恢复服务和原 processor。检查身份在查询既有注册之前，所以以后启动不能用旧注册掩盖部署更换。注册存在后的重启沿原 owner 使用旧 namespace，不扫描/重算已有文件。多个首次注册发生中断时保留已经写入的原事实，使用同一 true 配置修复后重启；不删除注册、换库或回到未计量 writer。

IAM/安装失败或 commit ACK 不明先核对**同一** policy/原表授权/原 offer，不创建替代数据或重买。尚无任何容量 namespace/订单/占用/预占时，可保留新增 IAM/空表授权/offer并恢复原配置/镜像；一旦有容量事实必须保持 true 并修复前进，不能关闭开关回退或覆盖旧备份。原 UNKNOWN/取消/并发/到期/购买恢复全部沿冻结 V1，无后台扫描、TTL 清零或第二重试 owner。

## 6 必需验证与交接

按 TDD 先复现 MinIO dummy ACL 不可满足原准入及显式 pin 接线缺失，再最小实现。使用已有单元/SQLite/独立 PG/真实 MinIO fixtures，不建立测试平台。

| Must | 风险匹配证据 |
| --- | --- |
| 显式可信物理身份，无云降级 | 签名 ServerInfo、正确/缺失/异值 ID、匿名/403/redirect/超界/取消；云缺 Owner.ID 继续失败；轮换 key identity 相同 |
| 正确 consumer 与配置 | Knowledge/Ecoservices 原装配消费配置；非法 provider/pin/endpoint 拒绝；两个 prefix 首次为空、有文件拒绝，既有 namespace 重启不重扫 |
| 保留商业事实与权限 | 独立 PG 原 CLI：仅六表 GRANT/原 offer、无 Money 连接；有事实/订单、字段缺失、冲突与安装失败拒绝/回滚；仍空时重放不覆盖原 offer |
| 真实 MinIO 可执行 | 独立 MinIO 的窄 runtime 凭据通过 SDK身份/空目录准入，错误部署 pin 拒绝；不把假 HTTP 或 root 调用当作 runtime PASS |
| 原 trial 不受影响 | 原 source-only trial SDK/URL测试、AWS/COS空目录/owner测试；41444/42444运行ID/镜像/配置/卷与容量事实保持 |
| 本地启用没有损坏原数据 | 安装及切换前后原非容量商业/钱包/Knowledge/Ecoservices事实一致；原旧 offer 保留；新变化仅预期容量 offer/授权/namespace及正常原业务读取 |
| 用户能查看 | 35444正常原登录、企业上下文、“资源与额度”读原容量；正确5GiB真实零或实际值；原页面继续可读，支付不开启 |

上述是本批必要开发自检和有界运行核对，不作为新增人工验收流程。交接保留35444、41444、42444入口和原启停方式，明确技术实测、PR状态与用户试用分别报告；CI或独立代码评审不签发用户/生产验收。

## 7 Legacy 与评审

```text
Legacy decision: EXTRACT
Reusable behavior: 现有 madmin SDK签名、源素材trial的有界只读DeploymentID验证、原 Commercial/容量schema核对与报价安装。
Current owner: integration/s3、currentapplication、Commercial schema-owner、storagecapacity。
Cutover/deletion condition: 新显式配置直接调用现行 owner；现有 trial/cloud身份保持，不建立旧账户/endpoint fallback、双读写或迁移。
```

当前 round 0，等待独立评审。只审核本增量身份/权限/受限安装与本地切换边界；按 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE 分类，最多两轮正常架构评审。正式准入绑定 design-only commit/blob，并先于实现记录到本稿和 Issue。
