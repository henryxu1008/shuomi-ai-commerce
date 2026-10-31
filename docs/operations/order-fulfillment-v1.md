# SHEIN 仓库履约首版运行与交接

执行 Issue：[#1037](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1037)。主要 PR：[#1038](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1038)。设计准入及事实 owner 见[冻结设计](../architecture/order-procurement-fulfillment-v1.md)。本说明不授权操作保留实例、真实订单或部署。

## 当前范围

用户 2026-11-01 批准先交付仓库履约：采购并合格收货，或预留已有库存，再分别执行美国站客单自发货和原授权供应商托管交仓。供应商直发延期、未开放。采购平台下单、付款、供应商运输均由用户在外部完成，内部登记不证明付款渠道确认。

消费者新执行只接受原单明确为 shein-us、stockMode=3、允许并已选商家物流、正常打印状态、已知订单和商品状态且未过平台原时限的订单；不自动切换物流。用户目前没有可验证的真实美国站自发货店铺/订单，真实平台联调为 **NOT_RUN / 待联调**。托管原采购单不能证明销售市场，显示“未知”，南北大区不当成美国。

托管首个可用运输路径限定原单全部剩余数量、一个实际包裹、shippingRoute=0，官方当前资料允许的自送/自提、上门查货或车队（deliveryType=4/21/2）。SHEIN 合作物流购买、货代、拆包路线、跨单合包及私有面单下载入口未开放。官方发货单列表文档 FAQ 说明不强制打印发货单；当前直送路径不伪造必须取得的面单或成功状态。未开放的运输方式不作为本入口可用方式展示。

## 部署准备（新隔离环境）

ERPNext 固定 v15.124.3、Frappe 15.125.0；官方基础镜像 digest 固定于扩展 Dockerfile。每企业独立 site、数据库、固定 HTTPS origin 与专用服务密钥。Company 标签不作为企业隔离边界。当前实验环境只在后端容器安装了扩展，不能当成完整已部署的服务。

从仓库根构建扩展镜像：

```powershell
docker build -f extensions/erpnext-fulfillment/Dockerfile -t local/commerce-erpnext:<reviewed-git-sha> extensions/erpnext-fulfillment
```

新 ERP 环境所有 Python 服务（backend、queue、scheduler）使用同一扩展镜像，沿用官方 frappe_docker 的持久卷与正常服务编排。新 site 首次安装前清模块缓存；`bench get-app` 正常安装通常会处理缓存，手动把代码加入已有 bench 不能省略此步骤。不要在 HTTP 启动时安装 schema。

```sh
bench --site <new-site> clear-cache
bench --site <new-site> install-app commerce_fulfillment
bench --site <new-site> migrate
```

创建专用 System User，只授予 `Commerce Fulfillment Service` 角色，生成独立 API key/secret。生产拒绝 Administrator；测试旁路仅限显式 `commerce_fulfillment_test_site=true` 的合成站点，不能带到试用/生产配置。角色仅对限定原生采购、库存和回执单据写入，并读取必要主数据/Account，不读取 Bank Account 或银行/付款交易。

在该新 site 配置 `commerce_fulfillment`（以下均为应替换的配置名，不是有效业务数据）：

```json
{
  "organization": "<current-organization-id>",
  "company": "<periodic-inventory-company>",
  "service_user": "<dedicated-service-user>",
  "customer": "<consumer-order-customer>",
  "internal_customer": "<same-company-internal-customer>",
  "source_warehouse": "<available-warehouse>",
  "transit_warehouse": "<non-reallocatable-in-transit-warehouse>",
  "rejected_warehouse": "<separate-quarantine-warehouse>",
  "selling_price_list": "<base-currency-selling-list>",
  "buying_price_list": "<base-currency-buying-list>",
  "supply_movement": "owned_transfer",
  "address_retention_seconds": 3600
}
```

Company 必须使用周期盘存，`enable_perpetual_inventory=0`；本入口不自动生成整套财务实施。启用 ERPNext `Stock Settings.enable_stock_reservation` 与 `allow_partial_reservation`。Item 必须是已启用库存商品、已配置准确单位和换算；供应商及公司基础币种价格表必须可用。UI 采购单价明确填写，不能因未填而默认为 0。首版采购使用基础币种；不以 rate=1 猜外币汇率。

`owned_transfer` 的内部 Customer 必须 `is_internal_customer=1` 且 `represents_company` 为同一 Company；原生内部 DN 同事务推进 SO/SRE，并把源仓货物移入固定在途仓。只有确认平台商业所有权已转移时才配置 `commercial_sale`，浏览器不能改写。拒收隔离仓与可用仓/在途仓必须不同，不参与新分配；缺少隔离仓配置时拒绝登记有拒收数量的收货。

地址需要 Frappe site 的正常 `encryption_key` 和显式 1–86400 秒保留期限；缺少时关闭地址能力。到期立即拒绝读取，清理复用正常 Frappe scheduler。不得把凭据、真实地址、site_config 或 API secret 放入 Git、浏览器配置或工作流参数。

硕米当前运行配置增加可选 `fulfillment`，保留已有 Store 官方应用、准确成员权限及 IAM token 配置：

```json
{
  "fulfillment": {
    "temporalAddress": "127.0.0.1:7233",
    "temporalNamespace": "<current-namespace>",
    "sites": [{
      "alias": "<fixed-site-alias>",
      "organization": "<current-organization-id>",
      "origin": "https://<fixed-private-erp-host>",
      "apiKey": "<dedicated-key-id>",
      "apiSecretFile": "<absolute-private-secret-file>",
      "supplyMovement": "owned_transfer"
    }]
  }
}
```

沿用当前应用正常启动命令和 ListingKit 镜像构建入口，不另建服务入口或临时认证：

```powershell
go run ./cmd/current-application -config <absolute-private-current-application.json>
```

UI 沿用当前登录配置和 `LISTINGKIT_SERVICE_API_BASE`，正常 `pnpm build` / `pnpm start`。以上命令只用于已准备的新隔离环境，不对现有实例执行。
ERP secret 文件必须满足当前私有文件 ACL 检查；固定 HTTPS、正常证书校验、禁止重定向。当前应用启动只核对站点配置并启动既有 Temporal worker，不执行安装、初始化或数据迁移。未配置该模块时没有履约命令路由。工作进程失败时能力关闭。

## 用户操作路径

1. 用当前硕米登录和企业选择进入 Console；成员需要 `workbench.store.read` 及相应 `workbench.fulfillment.read/purchase/stock/address/execute`。成员权限页通过现有模块授权分别配置 `fulfillment-purchase`、`fulfillment-stock`、`fulfillment-address`、`fulfillment-execute`；原 `store-orders` 仅带履约读取，不自动获得这四类用途。只读观察权限不自动授予采购、库存、地址或平台执行权；每次请求重新检查当前成员、企业、店铺连接和用途。
2. 独立一级“订单管理”页的“进入订单履约”，目标 `/workbench/store-orders/fulfill`；采购入口为“供应市场 → 我的供应链 → 采购管理”，目标 `/workbench/supply/mine/procurement`。两个入口关联同一原单与原生单据；工作页按原单关联、采购与收货、库存与装箱、平台发货、交接与物流分步。只有当前后端返回可用且当前权限允许才显示入口。
3. 选当前授权店铺和消费者/平台备货原单号。核对真实原行、市场、平台状态和原时限；读取、列表和刷新不接受订单、不采购、不发货。
4. 显式逐行关联原 SKU 与 ERP 商品/库存单位，建立原生 SO 需求。已有需求通过原平台身份找回；不另建重复需求。
5. 有现货则预留。缺货时生成内部 PO，在供应商平台人工下单/付款，登记外部单号。分批收货分别填合格、拒收隔离、报告短缺；合格 PR 通过 ERP 原生收货来源 SRE 绑定原需求，拒收只进入隔离仓。短缺是本次交付报告，不当成已收货或自动取消剩余。与供应商确认不再交付后，填原因、确认关闭内部剩余采购；不自动取消外部订单。
6. 用真实包裹编号和准确数量保存装箱。此时保存草稿 DN，不扣源仓。
7. 消费者地址为独立“准备接受并获取”或“准备仅读取”操作，核对后再明确确认。接受订单会改变平台状态、限制买家改地址。临时地址只在当前订单内存展示，到期、关闭、切原单/企业或离页后清除。实际运单使用官方当前返回渠道，准备后核对原包裹、运单和渠道再确认上传。
8. 托管选择官方当前地址、商家仓、运输方式及北京时间预约；核对原订单全部 SKU、数量和平台收货仓后准备。确认发送前再核对当前权限与完全一致的原内容。创建返回后，以原 deliveryCode 回读并校验原单/商品/数量/包裹数/仓库/方式；无完整证据保留 UNKNOWN。
9. 货物实际交给承运商/平台仓后，核对包裹、勾选实际交接并记录。托管必须绑定这张原装箱单的平台成功操作。原生 DN/SRE/SO/库存和命令回执在同一 ERP 事务提交；owned_transfer 源仓减、在途增，企业拥有总量不变。
10. 平台操作页查看原回执。消费者轨迹沿用订单管理；托管可查询该原 deliveryCode 的平台取件、预计到仓和实际到仓观察。采购在途由人工外部单号与分批实际收货核对；未接承运商的节点显示未知。原采购单累计实收、入库、次品单独显示，不能据此自动关闭某个包裹的 UNKNOWN 或把在途当可用库存。

## 原操作恢复与保存

ERP 原生 SO/PO/PR/SRE/DN 为唯一数量和库存事实 owner。ERP 扩展的 Operation 及子效果唯一键拥有原命令、确认/派发阶段及回执；Go、浏览器和 Temporal 不维护第二份库存账。Temporal 仅保存固定站点 alias、企业和原操作 ID，mutation activity 最多一次尝试，无地址、密钥或原始平台响应。

浏览器在发送前通过 Web Locks 与 localStorage 保存组织/用户限定的原操作 UUID 和非敏感命令载荷；无法可靠保存时不发送。恢复后不自动重放。收到可核对的原生回执后才清浏览器待核实提示；UNKNOWN 的原效果仍由 ERP 唯一键拦住，不因清提示而重发。

本地有意图但未取得回执时，可明确“核对并取消尚未生效的原操作”：原生唯一键插入取消回执与任何迟到原事务串行；若原事务已提交则保留其真实回执，不改成未生效。准备/确认尚未领取时可原子取消；CLAIMED/DISPATCHING/UNKNOWN 不被取消或改成功。新的 UUID、包裹名和工作流不会解锁原商品效果。仅 ERP 仍为 CONFIRMED、工作流在领取前失败时允许用户用原操作显式启动；不会自动重发 provider。

普通 HTTP/Frappe 417、超时或截断都不是无效果证明。已知原生校验失败只在本地整条业务命令成功回滚后，与原键持久化 DEFINITIVE_NO_EFFECT；未知异常保留原键核实。私有地址不进入普通结果、日志、Customer、localStorage 或工作流历史。原 site/Store 连接版本变化会停新执行，不自动重绑定已有需求。

## 验证状态与实际限制

适配原始记录见[适配报告](../architecture/erpnext-shein-fulfillment-fit-v1.md)，formal app、Go/BFF/UI 的当前检查与独立最终 Review 以 PR 准确候选记录为准。合成测试不是实际 SHEIN 试用，也不是用户验收。

本批不合并、不部署、不关闭 Issue、不修改现有实例和真实数据。SHEIN 权限、真实原单、真实仓库/运输资料、地址隐私配置、实际预约/上传/交仓均需后续单独联调与验收；没有证据保持 NOT_RUN。当前 API 没有可证明原效果的查询时，UNKNOWN 交平台核实，不提供本地强改成功入口。
