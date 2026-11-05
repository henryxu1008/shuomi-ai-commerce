# SHEIN 单页试取账本与运行依赖准备

## 产品范围与依据

[#1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073) / [#1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)，入口基线 `4d5bb30c94150dfffcdadd35b8107a23c3d8d159`。Design Basis 为 Independent Architecture。当前交付是让部署人员能核对原库的安装与权限，并用合成批准、密钥引用和原单证据验证入口。复用[业务首试范围](research-commercial-assessment-v1.md#53-shein-美国站商品卡片首试实施方案)、[默认关闭的入口](shein-us-trial-entry.md)、[单次发送](shein-us-one-page-trial-send.md)和[原单适配器](zenrows-shein-us-trial-adapter.md)。原工作区和唯一 Writer 续做。无 UI、Figma、旧库兼容或数据迁移。本轮不读真实凭据、不建库或安装到统一实例、不更新或启用实例、不请求供应商。SHEIN 书面回复并行。仍只取 1727 第一页、最多前 20 张、一次发送、25 credits、USD 0；两项 AI 各 ¥0.30 / 20 点且关闭。

## 现有 Product 安装步骤与权限

1. 复用 `product-acquisition-init` 的独立功能选项 `-enable-shein-us-trial -confirm-database product_acquisition -config <absolute-private-owner-manifest>`。目标必须为原 `product_acquisition`，显式私密配置要求现有 `collections` 和 `dataServices`。仅由有权的现有 Product 表 owner 在单独授权后执行。用既有数据库锁和事务，确认当前用户、目标库、`public` schema、现有 jobs/items 表 owner 以及已安装的 Data Services/Collection 当前结构与严格授权。先拒绝既有同名试取账本和不完整安装，再建一张 `public.data_acquisition_shein_us_trials`，保持原事务、行和其他 ACL。失败回滚。无运行时 DDL、无角色或口令变更。
2. 账本继续由 Product Data Acquisition 拥有。现有 `data_services_runtime` 只增加表 `SELECT`、三列 `INSERT(approval_id,binding,claimed_at)` 与两列 `UPDATE(request_id,credits)`。不给可转授权权限、DELETE、DDL、其他列更新、其他角色或 PUBLIC 访问。沿用原 Catalog、Collection 和 Data Services 权限。不调用全表重新授权。现有完整角色检查按两个当前可用状态验证：账本无角色权限，或账本存在并具备完整精确权限。缺表、部分或越界授权均不能装配试取；部分或越界授权不会被原 Data Services 许可检查接受。试取显式装配要求安装态，而原服务关闭时可验证已安装的精确授权。仅验证读取，不修复漂移。
3. 旧运行候选 `f50d7d35e` 的源码没有 `trial_binding`、`source_position`、`evidence_checksum` 及源序唯一约束，并会拒绝试取账本的新增访问权。安装前须按实际原库验证当前 base，旧形状应拒绝并保持原库。现有空表是否可按当前合同重建、更新顺序和回退另行审批；若有原记录或不满足条件，停在只读报告。实际切换时先确定候选能承接当前 base 与 ACL，不能假设旧版本能带新授权重启。若后续获准回退 ACL，只考虑撤销试取表本次新增授权，保留账本事实；不在本轮执行。

## 单次批准、凭据与原单

- 装配入口消费唯一可信、不可变的 `SHEINUSTrialApproval` 和一位明确的同企业核账者 scope。复用当前 approval ID、企业/操作者/成员、私有项目、账号/专用 key 引用、有效期、funding 和固定请求的绑定。提交、查看、发布限原成员；核账限指定核账者并与原项目读写分离。每次动作额外读取当前可信访问证据，撤权或不可用拒绝。Send/Publish 核对现行来源/使用范围，Send 核对未过期的同绑定账户额度证明。预算证明须有有效期、包内至少 25、单次最高 25、专用 key 独占窗口和零新增现金。View 和核账在发送批准过期后可凭当前相应权限处置原尝试。运行注入为可信服务侧参数；没有提交批准、改授权或接受证据的公开入口，没有第二批准库。
- ZenRows 专用 key 只由受控 transport 在发送前按原批准的账号和 key 引用解析。固定 resolver 拒绝其他账号、其他 key、过期或空/不合格值，不找主密钥或其他默认值。底层使用现有注入读取口；本轮只有合成值，不引入密钥表、凭据安装或实际私密文件。原请求核账复用 `zenrows.NewLookup`，只接受现有可信逐笔证据 reader，不调用真实账户、不添加私有 Activity Log HTTP 接口。原 ID、账号和可用的 key/目标字段须一致；无 ID 时须原占用与完整接受窗口及唯一终态。部分证据、过期窗口、错单或多个匹配均保留 UNKNOWN 和 25 预留，不补发。实际可信批准、实时证据和密钥来源接入仍需另行验证。
- `httpapi` 只提供显式的内部装配构造，复用既有 Trial authority port、ZenRows Fetch、Lookup 和 Product ledger。构造本身不读取密钥或请求供应商，不挂到 `cmd/current-application`、配置或 LAC。只在合成环境主动装配。实际发送及发布的租户、项目、资料和资金权限仍在原动作重复检查，端口不接管授权或记账事实。所有错误使用既有固定响应，原密钥、URL 查询和证据正文不入业务日志。

## 实施准入与验证

独立 R1 已核对设计 `c114f2f0d43740d85572059ddebe77d40ec47701`，结论 **IMPLEMENTATION_READY（仅离线）**。

试取表的列级 INSERT/UPDATE 由窄检查核验。全量角色拒绝仍覆盖其他表，账本另查 `relacl`、`attacl` 以拒绝 PUBLIC、其他角色、可转授权和越界列。安装事务用显式 `data_services_runtime` 目录 subject 验证未提交的 DDL/ACL，不切换身份或建立运行连接。发送预算证明必须覆盖有界 `sendCtx` 的实际截止时点，短于该截止时拒绝发送。

先经独立 Architecture Review 再实施。用现有隔离 Product PostgreSQL 验证旧形状拒绝、安装事务和精确列授权，以及新候选默认关闭时仍能验证精确 ACL、越权或缺项拒绝。用现有合成路由与 loopback TLS 验证固定批准、过期或撤权、错项目、错账号/密钥、重复发送门禁、原请求唯一核账及未知回执保留。实际账本安装、启用及真实试取均 `NOT_RUN`。
