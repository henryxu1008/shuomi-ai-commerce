# SHEIN 美国站一页响应到原版发布的离线接线

## 范围

[#1074](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074) 由原 #1097 Writer 续做。设计基于 [商品卡片首试 §5.3](research-commercial-assessment-v1.md#53-shein-美国站商品卡片首试实施方案)、[单次发送](shein-us-one-page-trial-send.md)、[ZenRows 原单适配器](zenrows-shein-us-trial-adapter.md) 和 [现有 Product owner](project-boundaries.md)。起点 `ec5f6fb3b13efcc0f85da492e17f5c23d87977f9`。私有研究项目成员的目标是把一个合格的美国站卡片从同一批准单保存进现有 Collection，并由 Research 原版回读。保持类目 1727 第一页、按源顺序只处理前 20、一次提交、25 credits、零增量现金。只实现内部离线组合。当前 Figma 页面及 #1075 原版读取路径无 UI 改动。真实账号、供应商、实例、AI、付费比较、购买与合并均在范围外。

## 合同与事实 owner

- 保留现有 `SHEINUSTrialAuthority`、一次占用和 `SHEINUSTrialLookup`。发送结果拆为原单用量与从同一及时完整响应取得的合格卡片。核账结果只含用量，不能携带卡片或触发发送。使用现有 Product 调研项目和执行权校验。批准绑定含私有 scope、既有资源资金归属和固定查询，资金归属用现有 `LiveAccess.CheckExecution`，不增加金额或资源扣减。正式接线仍待授权来源、账号和准备能力另行准入。
- ZenRows 仍采用有界 HTTP/1.1 回执路径。供应商账务可信且正文完整时，单独核对 `Zr-Final-Url` 严格为获准 US 类目 1727 第一页，解析 `parsed.products` 的数组前 20 个原位置。按 `goods_id`、非空标题、HTTPS 原商品 US 主机及路径 ID 验证，去掉跟踪参数，以原序保留首个合格商品 ID。金额以原 JSON 十进制文本保留，只有金额和明确币种同时合法才输出显示价。缺价、变体和未证实的时间与优惠按既有缺项合同。解析不合格时不产生卡片，可信的原费用仍可结账。原 HTML 和不准入字段不保存或记录日志。
- 数据 owner 仍为现有 Product Data Acquisition。及时完整的同一响应、当前权限、确切原 ID 与已确认 0–25 credits 才可准备卡片。一次同步 Product 事务提交原页用量及规范化准备记录。页级试取行持有唯一实际 credits 和逻辑预留，`data_acquisition_jobs/items` 持有唯一观察字节和准备/发布状态。以原批准 ID 为 SHEIN job ID，按冻结查询派生 queryRef，条目及发布 ID 由原 job 与原商品 ID 稳定派生。保存源序号以原顺序交接。全新安装 schema 明确 Amazon 继续按条目预留与原计费，SHEIN 页级试取的卡片不填虚构 Resource 预留。禁止按卡重复扣费。
- 继续使用现有 `PublishSHEINUS` 的 Product UoW、Sourcing、Catalog、Collection，发布前从原私有 job、原条目和已确认合规的原页账务校验冻结绑定和 SHA-256。每张卡复用原事务。重放验证准确已有发布；不重新解析、发送或重复发布。发布适用当前权限和原 Product 作业期限，撤权即拒绝新发布。Research 继续用已有 `ReadSHEINUSProvenance`，按原四元组选择、回读字节和当前授权。

## 失败和准入

| 事件 | 权威结果 |
| --- | --- |
| 获准单已占用，网络失败、超时、取消、失权、响应写入不明 | 原单不重发，保持已知 ID 和 UNKNOWN/25 预留，无新准备卡。按现有只读原单查证。若用量已由同步事务确认则保持其原值。 |
| 完整可信但非合格类目、无合格卡、404/410、超过批准费用 | 费用按回执实记。无新准备卡，超额标超限。 |
| 原页确认及准备已同步提交，后续发布失败或进程重启 | 原账与规范化观察保留。只重试同一原准备条目的 Product 发布，部分已保存卡回读原回执。原外部请求次数不变。 |
| 超时后账户原单核账确认了用量 | 只更新原页用量，缺失的页面内容无自动回填。 |
| 重放、并发或权限变化 | 原一次发送门禁不变。业务发布检查当前授权，基于 scope + 原 job/条目版本及字节绑定，在事务内保证幂等。 |

独立 Architecture Review 确认新增的同 Product 准备事务、页账/卡片约束、撤权与重放，再进入 `IMPLEMENTATION_READY`。采用现有本地 TLS 与空 Product PostgreSQL 测试入口，合成权限经原 authority、实时执行权和真实占用事务通过，禁止直接插入假预留或伪造原页回执。使用 21 张卡的样例，检查前 20、坏字段、缺价、最终 URL 跨区、收费空结果、重放、发布失败恢复和超时只核原单。源发布和原版读回通过后，#1075 仍由既有消费版本核对。真实首试要件独立保留，许可、账户和万邦并行。
