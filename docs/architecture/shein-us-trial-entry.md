# SHEIN 美国站单页取数入口

## 设计依据和范围

[#1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073) / [#1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)，基线 `7e3c5dbe48d72135aae0a5c5a96e64133f17c2fb`。Design Basis 为 Independent Architecture。本轮用户确认离线完成入口，保持默认关闭，书面许可与真实试取独立推进。复用[已批准的一页业务范围](research-commercial-assessment-v1.md#53-shein-美国站商品卡片首试实施方案)、[单次发送](shein-us-one-page-trial-send.md)、[原单核账](zenrows-shein-us-trial-adapter.md)和[原版发布](shein-us-page-publication.md)。已有研究详情继续通过原 Collection 选择器关联、查看原版证据。仅增加后端内部试取 HTTP 合同，无新增页面、菜单、按钮或 Figma 改动。

用户结果是在受控装配中，有权的研究项目成员用事先获准的一份单子提交 SHEIN 美国站类目 `1727` 第一页，成功后可从原研究项目关联已发布资料；超时可以查看原单而不补发。按源序最多处理前 20 张，批准单最多发送一次，包内上限 25 credits，新增现金 USD 0。两项 AI 操作各 ¥0.30 / 20 点且执行关闭。真实批准/许可、ZenRows 专用 key 和实际 Lookup 接入、出口实测、生产装配、实例更新与 AI 调用不在本切片。1688 独立。全新系统，沿原 Product 库，不迁移、回填或兼容旧记录。

## 入口与权限

- 复用项目 HTTP 原身份绑定以及活跃企业/成员验证。入口挂在 `/api/v1/workbench/research/projects/:project_id/shein-us-trials/:approval_id`，`GET` 读取原尝试，`POST /submit` 提交或恢复原发布，`POST /reconcile` 只核对原账。无请求体、查询参数、可选目标、凭据、金额、批次或自动操作。两个标识须是规范 UUID，`approval_id` 也是唯一幂等键。拒绝扩展请求、设置 `private, no-store`，不回显原始异常。每条路由绑定现有当前身份、企业实时访问和所需权限。提交使用项目管理权限，读取及核账使用数据市场权限；读取另检查原项目与资料读取，核账权限仍由原账户权威确认。
- 当前生效身份提供企业、操作者与成员，调用者不能选择。`httpapi` 调用现有 Product Data Acquisition `SHEINUSPageTrial`。在 `SHEINUSTrialAuthority` 前加项目约束，逐次用已有项目 owner 校验该路径项目等于原批准单的项目。View 要求当前项目可读及 `LiveAccess.CheckRead`。Send/Publish 要求原成员对当前活跃 `PRODUCT_RESEARCH` 私有项目有写权限、批准权威仍允许并经过原 `LiveAccess.CheckExecution`。重新检查使用每次操作上下文的实时身份，沿旧授权及 funding 合同。跨项目/企业、失权或找不到批准单统一拒绝，不返回原单细节。仅由服务端可信 authority 读不可变批准和当前来源/账户证据，HTTP 不能创建或授予批准。
- Reconcile 可由可信 authority 明确授权的同企业账户核账者处理原尝试，允许原提交者失权或原项目归档。只校验同企业和批准单路径绑定，不以核账角色替代项目发布权限。既有 `SHEINUSTrialLookup` 只读查原账，账务更新归原 Product 账本。它不授予读取商品、保存、发送或发布权限，返回账务状态，不带商品、凭据或授权绑定。

## 装配、状态与恢复

- `current_application` 的内部装配选项只接受明确注入的 authority、sender、lookup，且要求现有 Market Research / Data Services / Collection。用原 Data Services Product pool、repository、live access 与发布方组合已评审的 `SHEINUSPageTrial`，禁止启动时创建 schema。显式接线时只读确认既有试取表权限，缺依赖、重复选项或缺表/权限拒绝装配。默认无该选项，路由注册和合同允许列表均无试取路由。该选项留在 `httpapi` 包内，当前配置、环境变量、LAC、运行主程序均无开启入口。离线测试以空的独立 Product 测试库、合成权威和本地替身显式装配。
- Product Data Acquisition 继续是批准关联、占用、逐笔账务和观察的唯一持久化路径。`GET` 经原 View 权威及项目可读校验后只读 Product 原尝试，确认可见性后返回最小状态。`submit` 沿原单占用事务和权限复核执行，未确认结果保留原预留；确认后才使用原准备记录的逐条发布，恢复时只回读或继续本地原发布。`reconcile` 沿同一原尝试核对终态，恢复已确认账务时仍不填造页面内容。入口不增加事实表、重试 worker、第二账本或跨域事务。
- 响应仅含批准 ID、原请求 ID、已确认 credits（未知为 null）、预留 credits 及明确状态。提交成功额外返回已发布数量，由原 Collection 读取资料。只有已占用且当前权限仍允许时才可披露 `UNKNOWN` 和预留；无可见原记录的异常只返回固定错误。用量确认和资料发布是两件事，原单已记账但发布失败时可再次提交同一批准单恢复本地发布，不会发新请求。有效但无合格卡的结果显示已确认、发布零张。实际超限用量保留原数，不发布。请求受路由期限约束，供应商发送仍服从原 30 秒和批准截止，取消后只保留原单核账必需事实。凭据、Cookie、原 HTML、URL 查询密钥和原始传输错误不进入响应或应用日志。

## 验证与阶段

独立 Architecture Review R1 核对设计 `dd2fc3238178f50fe988873a6d134d6753b8255b`，结论为离线 **IMPLEMENTATION_READY**。`GET` 在未占用时不能调用 Send。提交之后若发布授权被撤销，拒绝优先，响应不保留原单字段。核账者不能借原账访问商品或提交。响应超时由同一批准 ID 的只读查询恢复，不把空结果当成已占用或已扣点。服务器全局超时沿用现有合同。

数据市场权限独立于项目和 Collection 权限。原单 View（包括提交重放中的 View）和 Reconcile 在返回可见结果前通过现有企业实时授权复查当前调用者的数据市场权限。核账在查到外部结果后、写入账本前同样复核该权限。项目或账户权限继续按原动作分别检查。

独立架构审查通过且明确 `IMPLEMENTATION_READY` 后改生产路径。合成验证复用已隔离的 Product PostgreSQL 与原研究消费回归，覆盖默认无路由、注入缺项、同项目批准单、错项目/跨企业/撤权拒绝、拒绝输入扩展、单次发送及重复关联、超时后原单查看和只核原账、回执不泄漏，以及原商品发布后通过现有 Research 回读。生产安装权限、实际受权账户与出口、逐笔凭证和 SHEIN 书面许可分别核验。真实试取仍 `NOT_RUN`。
