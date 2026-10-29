# 店铺浏览器后端与固定角色展示组合

## 当前源码交付范围

2026-10-29 用户批准把已验证组合及必要的 #979 依赖交付回现有 Draft PR #956，沿用执行 Issue #954。本阶段允许本地提交、普通 fast-forward push 到既有 `codex/store-browser-v1`、维护 Issue / PR、必要 CI 与有界增量复核，替代下方初次集成阶段的“仅本地、不 push”限制。不创建重复主要 PR，不改 #979 的分支或关闭该 PR，不合并、不发布镜像，也不操作运行环境。

Design Basis 继续为 Reuse Existing Architecture / Ready，复用下述已经批准的 Store / authz / Membership / schema-owner 合同。发布前一次同步当前 main，保留 main 已退休的路径；已验证且未变化的生产代码、测试和独立审阅证据允许复用，新增组合风险按实际 diff 验证。准确最终 HEAD、基线、PR CI、增量复核和未执行项统一记录在 PR，不以历史构建或本地实例的旧 SHA 证明新候选已经运行。

只读桌面观察与动作验收分开：浏览器消费者的登录、企业、列表、详情、无权提示和禁用按钮证据由 openbrowser-platform#15 维护；它们不证明 browser.use 准入、配置 / 打开 / 关闭环境或结果报告已执行。受保护管理员完整 role_keys 写回的 B1 仍未解除，本阶段不修改权限或扩大试用范围。

## 历史：初次本地集成范围

本轮为 2026-10-29 获准的本地源码整合与离线验证，不是部署或现场验收。原执行上下文为 #954 / PR #956；不建立重复主要 PR，不推送、不更新外部 Issue / PR、不合并远端 PR。原工作树归档快照缺失，本聊天使用新建的独立 managed worktree，其他工作树与运行环境不改。

## Design Basis / 开工准入

**Reuse Existing Architecture / Ready。** 本记录先于本轮生产文件整合。复用原 #954 已 IMPLEMENTATION_READY 的 [Store 工作台设计](https://github.com/qq550723504/openbrowser-platform/tree/a42729e3a802cd7e2e972f67d937c654fe89c422/design/store-workbench)、C1/C3/C4 与其两轮独立架构评审；具体实施交接为 PR #956 的 `docs/architecture/store-browser-workbench-implementation.md`。固定角色显示复用已合并 #992 及 [说明](administrator-module-projection.md)。不重新定义状态机、持久化、权限语义或外部副作用。

使用者是等待浏览器试用的企业成员。当前操作阻碍是 main 缺少配套 Store browser 接口，而旧试用组合不是本轮最小源码版本。本轮可见交付是固定 Git SHA、可审查的最小组合差异、实际离线测试结果与下一运行阶段条件，不能把它描述为已安装或已经可用。

来源固定为刷新后的 main `e70f5ef6f894339a1850340c635f666538091a63` 与 PR #956 `88bfd9163c632dab48d097658c5fb55f8fd94066`，merge-base `dee5f3e4ed01f551b5e1852bfe0a7c461e24a941`。main 已包含 #992 merge `fbdd5bcea0399c63fdcf81934288650b12168c79` 和后来合并的 #990；继承当前 main，不从旧 `a19cf48` 追加采集/Amazon 等代码。

真实前端组合依赖为 #978 / PR #979 `26dc936eaddf47aa2e8f1857a3c20901b7a7849d`：browser 将目录从 32 增至 33，但 main 的只读响应解析仍限 32，导致角色页面拒绝合法目录。复用该已 Ready 的最小修复：只读目录/系统投影上限 64，自定义角色/create/save/持久化继续限 32。已有 API、模块格式、唯一性、scope、system/version 与可用目录交叉校验不变。不把本次本地复用当成远端 PR 合并或现场角色页通过。

## Owner、调用路径与不变量

- authz 仍拥有固定 Casbin 策略和模块映射；Membership 系统角色定义统一用于 Roles/List/Read。模块须完整覆盖全部非空权限才包含；查看者自定义角色、platform role 或 protected subject 不得污染固定元数据。#992 的系统角色只读“已包含/未包含”及自定义编辑行为保持。
- 实际请求仍由共享 scoped authorizer 及现有 CurrentIdentity/LiveWrite 处理；browser.use 不自动授给内置管理员/平台/受保护角色。新动作同时要求 store.read、browser.use、有效 Store grant/active，不隐含要求 store.update。角色元数据不能成为授权器。
- Store owner 继续负责 Store→grant 同事务 fence、首次单次准入、相同 key 不同载荷冲突、固定 issuer/subject 原回执与报告、原子审计及有界游标。回执/报告不产生执行能力，UNKNOWN 不自动重发。C3/C4 route descriptors、精确路由清单及默认关闭能力不放宽。
- browser 两表、Goose 1601→2501 与严格实际权限验证仍归原 schema owner；本轮仅合格临时 fixture 执行测试，不运行真实 schema CLI、不读真实库/凭据、不改真实 ACL。
- main 已退休的独立 SHEIN 登录/身份同步和审计字段不得通过旧分支冲突恢复；只保留本轮 browser 的当前固定字段。Legacy decision: RETIRE 已退休路径；复用当前 authz / Store / schema-owner，不新增 fallback、IAM、兼容层或第二事实源。

本次已有实现的三十余文件与约 1,900 行 Store 增量由上述批准架构覆盖；组合本身不增加新的高风险协议，不因文件阈值机械拆 PR 或重复全局设计。若实际测试证明需要改变上述不变量，停止该部分实现并报告具体缺口。

## 本地验证与停止点

先核对真实冲突及目录读边界的失败，再复用原回归。必要范围包括 authz、Membership、Store browser domain/HTTP、实际应用装配和 C3/C4 路由 guard、declared capabilities/core schema CLI；只在确有前端组合差异时运行角色解析/组件/类型检查。未变 #992 代码的原 CI 和独立复核只作历史证据，不冒充新 SHA 执行。

完整候选形成后按仓库规则做一次有界独立最终检查，仅关注组合差异、授权、持久化/审计和测试证据；不重新审核无关 main 或新建验收平台。执行结果与准确最终 SHA 保存在本地交接回执，不写进 AGENTS.md 或远端 Issue / PR。

受保护管理员完整 role_keys 写回仍 **IMPLEMENTATION_NOT_READY / B1**；不以锁、重读、冻结列表或事后校验替代原生条件/单角色合同。不授予当前账号 browser.use，不建/授权账号，不改 IAM/成员/角色/Store grant，不发真实业务写入。不操作 Native 登录、店铺环境或共享服务/Compose，不构建/运行服务镜像，不删除容器/卷。此前 issue413 Compose 删除提问不属于本轮。

交付状态仅为本地源码和离线自检，待独立验证/用户试用。后续运行阶段须单独明确部署版本/工件、适用安装状态和权限、运行操作与用户验收；本轮不获取这些环境事实或授权。
