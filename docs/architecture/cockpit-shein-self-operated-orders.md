# SHEIN US 普通自运营消费者订单

Design Basis: Independent Architecture。Admission Status: IMPLEMENTATION_READY。执行 #1163；当前用户批准按公开文档核查后的建议推进，只做 US 普通自运营消费者订单，保留业务类型及销售范围，特殊 POP/SFS 单独核对，不增加仅本土销售筛选。

## 用户结果与范围

运营人员在新的独立本地实例选 SHEIN US 自运营测试店，使用官方形状离线回执，经原同步、保存、GET/BFF 和首页验证日期、店铺、订单去重、缺金额及覆盖不足；原订单详情可见销售范围。页面使用“自运营消费者订单”，不把自运营应用或 US 站点改称本土身份。普通自运营范围同时包含本土、跨境和未知销售范围。

不接真实店铺，不交换真实令牌，不调用平台业务 API；只允许公开官方文档读取。全托、POP/SFS 的业务接入、半托已有行为、全平台成功率、GMV 卡片、原合成预览、其他保留实例、生产部署、合并和 Issue 关闭不在本轮范围。无新同步框架、表、迁移或兼容层。

## 公开合同与产品依据

2026-11-09 用户已允许只查公开官方文档。源文档说明而非真实授权回执，是本轮合同依据；实际应用权限及真实字段覆盖仍为 NOT_RUN。

- [名词说明](https://open.sheincorp.com/zh/documents/system/4f5b2c46-839d-4dd2-80e5-65722a060759)：自运营为商家负责定价、运营、履约；应用类型决定授权范围，POP/SHEIN自营等另有类型。
- [授权手册](https://open.sheincorp.com/zh/documents/system/2169474d-1d4a-41a9-b9fd-427f63f54a63)、[密钥交换](https://open.sheincorp.com/zh/documents/apidoc/detail/3001235)：原 tempToken → openKeyId/secretKey/supplierId 协议；supplierBusinessMode 是可选特殊商家业务类型，如 POP-US、POP-GLOBAL、SFS。缺失不是本土身份证明。
- [店铺信息](https://open.sheincorp.com/documents/apidoc/detail/3000908)：storeInfo 下可选 supplierBusinessMode。查询可用于核对授权身份，不用缺失响应覆盖原授权事实。
- [消费者订单列表](https://open.sheincorp.com/documents/apidoc/detail/3000301-2000001)、[详情](https://open.sheincorp.com/zh/documents/apidoc/detail/3001563)：自运营、半托可用；salesArea=1 本土销售、2 跨境销售，是订单范围，不是商家注册地。orderCreateTime 是 UTC+8 下发时间。
- 复用 [Store Observations](store-center-platform-observations.md)、[SHEIN 单店金额](cockpit-shein-order-amount.md)、[当前领域 owner](project-boundaries.md) 及当前全新系统产品基线。

## Owner、合同和调用路径

Store 继续唯一拥有应用/连接、密钥和授权身份；原 OfficialMerchantCredential 增加 `SheinBusinessMode string`，表示原授权交换回执的 supplierBusinessMode，空串表示源未提供。SHEIN 专属 sealedCredential 加密保存这个字段，绑定原 attempt/AAD；不新增明文表、第二身份事实或 schema。该字段不对浏览器开放写入，不接受页面指定商家类型。

Integration.Exchange 读取 info.supplierBusinessMode；缺失/null/空串统一为未提供。字符串必须 UTF-8、有界64字节、无首尾空白和控制字符；未来非空合法代码原样保留，不用枚举重写为普通。格式错误沿原 exchange UNKNOWN 责任处理。Seal/Open 使用同一约束。原 supplierId/openKeyId/app/revision/组织/店铺/连接版本不变，商家身份 hash 不重新定义。

QueryStore 读取 storeInfo.supplierBusinessMode 并检查格式；若返回非空且与原授权模式不一致（含原授权未提供而查询返回特殊类型），返回原 unavailable，不把查询值写回授权凭据或降为普通。查询未提供时不覆盖原授权事实。仍由原 Complete/Observe 保存连接状态，重新授权沿原流程，不新增 retry、状态或自动恢复。

独立只读 OfficialObservationAccess 仅对 ApplicationSelfOperated + orders 增加准入：原加密授权模式非空即 ErrUnsupported，涵盖 POP-US/POP-GLOBAL/SFS 及未来未知特殊代码；不发送订单请求。模式未提供时，依据服务端已注册 self_operated 应用允许普通路径，但不推断本土。半托/全托现有准入不变。原私有 handle 的 cipher/attempt/connection pin 随每次 Check 重读，模式因换绑/重授权改变时按原绑定漂移规则失效。不得让“已授权”自动代表普通自运营或订单权限已实测。

Store Observations 继续唯一拥有消费者订单。Order 增加可选 `SalesArea *int`（json salesArea）；原 order-detail 解析、原 JSON record 保存与 GET/BFF 严格 schema 保留。1/2 按原含义展示；缺失/null 为未知，合法未识别整数保留原码并展示未识别；格式错误沿原不可用处理。不用 site、stockMode、仓库或履约资格推断销售范围；不按该字段过滤金额或推断店铺模式。不新增 storeCode 等本轮不消费的字段。

contract → implementation → injection → consumer：
原授权字段 → SHEIN Exchange/加密保护 → Store 原 credential/connection material → 原 OfficialObservationAccess 的 self_operated orders 准入 → 原 Service.Begin/Step/CommitPage/head → 原金额 GET/BFF/首页；salesArea → 原 Order JSON → 原订单 GET/BFF/详情展示。现有 currentapplication 注册原 provider/access，无新线上 reader 或接口代理。

## 不变量、状态与失败

- 当前组织、actor/member、Store grants、orders.read/sync、连接状态/应用版本/服务权益与每次调用前后重检保持原合同；不得借用半托密钥或用浏览器覆盖模式。
- 业务模式与 SecretKey 同一 Store 加密事实/事务保存，不双写业务类型表；模式缺失表示官方未提供，不是兼容分支。Legacy decision: N/A，全新安装，无迁移/fallback/旧ID映射。
- 不新增状态机或事务。授权 exchange 的 claim/UNKNOWN、凭据 Install、Complete/Observe 和订单 CommitPage 仍由原 owner/事务处理；响应丢失不重建授权或重发，未知不计为成功。
- 原授权事实只随原重新授权产生新凭据；QueryStore 的缺失不清空已知值，不一致 fail closed 并沿原状态查询/重授权路径处理。
- 数据窗口、分页上限、deadline、每页事务、原 key 相同载荷返回原命令、不同载荷冲突、current head 和不拼历史 generation 均不变。SHEIN 单页重复身份/详情关联不一致沿原规则拒绝；跨闭窗相同 orderNo 沿原 ON CONFLICT 更新单条保存记录，不新增跨窗口载荷冲突或恢复协议。同窗跨页的覆盖不足提示仍沿原 owner；不得套用其他平台的 typed 冲突分支重写 SHEIN 同步。
- 日期仍按列表 orderCreateTime → IssuedAt，UTC+8 完整日；订单号去重，不按商品/包裹加总。金额只取 productTotalPrice/orderCurrency，各币精确求和；不使用成本或预计收入，缺字段和覆盖不足隐藏金额。salesArea 未知不会补零或被归为本土。
- 无真实外部副作用；离线 RoundTripper 没有默认网络传输。测试内 IAM/Store 材料、调度替身清楚标明，不当作真实授权/Temporal 集群证据。

## 验证与交付

TDD 先捕获：授权业务模式解析/密文往返与格式边界；店铺信息非空冲突；普通 self_operated orders 允许、特殊/未知模式拒绝且订单传输0；原连接/密文变更在调用前后失效。验证同根因 sibling 字段及半托既有行为，不扩新平台。

订单测试验证 salesArea 1/2/null/未知码经 parser → 原 PostgreSQL → GET/BFF 保留；详情展示准确，不把缺失当本土。独立新 PostgreSQL + 两家普通自运营店及有界特殊类型测试材料，使用原 Service/保存/统计和现有本地承载验证日期、店铺、原键重放/跨窗口去重、原generation、缺金额/币种、覆盖不足及特定来源排除。只用本轮必要测试及稳定候选 CI，不新增通用 runner 或专项工具平台。

运行入口、准确候选与结果写 PR；新实例与测试数据保留供用户试用。完成实现进入 In Review，独立最终检查与用户产品验收分别记录；本轮不合并、不改原半托实例/原合成预览。

独立架构复核：设计 c09e1f34b442e766431c5085373819fd58c92c7c 由 /root/architecture_review 确认 IMPLEMENTATION_READY，无 BLOCKER。6374a4666 已校准原 SHEIN 跨窗保存事实；该设计事实澄清及其余格式/准入/销售范围保留/半托回归归 IMPLEMENTATION_TEST，在本批实现验证中收敛，不新增同步协议。
