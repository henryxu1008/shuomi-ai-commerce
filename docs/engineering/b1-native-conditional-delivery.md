# B1 配套源码交付

执行 Issue：[954](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954)。
Design Basis：Reuse Existing Architecture / IMPLEMENTATION_READY，复用
[冻结的原生条件合同](../architecture/protected-member-role-native-cas.md)。
用户于 2026-10-30 授权配套源码推送、PR、必要 CI 与独立复核，合并、发布、
再次部署和现有实例操作不在本轮范围。源码整理不增加产品能力或恢复协议。

## 使用者与调用路径

当前企业具有 MemberManage 的管理员，在成员详情中，对符合资格的受保护成员
增减一个允许的同企业业务角色。追加仅接受 store-browser 窄角色；系统角色只读，
其他原生角色保留。页面先显示目标与角色，再由用户明确提交一次。

成员抽屉 → 同源 BFF → Membership 当前准入/回执 → ZITADEL 条件接口。
命令重新核对当前 actor、企业、角色定义、原生作用域及 sequence；页面资格投影
不是写权限。ZITADEL 仍为唯一授权事实 owner，应用不直接写 provider 数据库。
明确零写入的原生冲突在同一事务保存证据、审计并释放 reservation；响应或提交
不明继续保留原 key/UNKNOWN，不自动重发、不退回完整角色列表写回。

## 配套 provider

Provider 源码位于 [henryxu1008/zitadel](https://github.com/henryxu1008/zitadel)，
补丁 PR 固定官方 v4.17.1 / `a9311b8c702531832575351a663e98a2242778e5`
基线，原生实现提交为 `82ff277bc320d9b0da51d9a987271962841d753a`。
本仓库的后端与 UI 使用同一个主要 PR；准确最终 HEAD、配套 PR 和 CI 留在 PR。

必须具备 `GetAuthorizationRevision` 和 `ChangeAuthorizationRoleConditional`。
没有这两个原生接口的官方版本不能提供此入口；保持现有 fail-closed 行为，
不得以重读、完整列表写回或本地锁代替条件合同。未来 provider 升级须保留补丁并
复核实际改变的原生 command/pusher 边界；版本号相同本身不证明包含该能力。

## 验证与证据复用

- 原生并发撤销、提交后通知、grant 发现、错误作用域及禁止重试的有效证据，
  绑定原实现提交与文件 postimage；未变化代码不重复整套验收。
- 当前 PR 的原必需 CI 保持，并显式运行已有 `b1_local` PostgreSQL 回执测试。
  GitHub 临时数据库由测试自身的 schema installer 初始化，仅含合成数据，
  验证零写入证据/audit/reservation 原子性和审计失败回滚。
- [独立配置的隔离 GUI 实测](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954#issuecomment-6501221843)
  已完成追加、实际浏览器刷新读回原回执、撤销及再刷新；管理员角色和原回执保持。
- [保留实例维护](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954#issuecomment-6498833232)
  已完成真实备份、断网恢复校验及 provider 普通 start 配套切换；
  [单店试用](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954#issuecomment-6512557750)
  已完成普通账号打开、正常关闭及双端报告保存。这些是历史运行证据，
  不冒充本 PR 新 HEAD 的实际部署，也不授权重复运行。

本轮保留现有实例、数据、管理员授权、模块配置、店铺及登录状态。现有实例的
真实受保护成员角色变更和商家网站内操作未执行；旧 UNKNOWN 保持原事实。
没有数据迁移、schema/ACL 变更、初始化、第二授权事实源或旧列表 fallback。
Legacy decision：N/A，复用当前 ZITADEL 与 Membership owner。
