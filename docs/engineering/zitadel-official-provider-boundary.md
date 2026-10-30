# 官方 ZITADEL 适配与受保护成员管理边界

日期：2026-10-30。执行 Issue：[954](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954)。
用户通过侧聊明确转达：仅使用官方 ZITADEL 发布版本与公开 API，不长期维护 fork。
本文是正式源码调整前的轻量 Design Basis；原定制增强及其通过证据保留为历史，
不再作为当前产品交付要求，不表述为上游已确认漏洞。

## Design Basis

**Reuse Existing Architecture / IMPLEMENTATION_READY。** 复用当前成员详情/目录、
受保护成员禁止整组角色替换与移除、Membership 原回执、Store C1/C3/C4 以及
既有 OIDC/实时权限 owner。本次撤销未合并的定制接口消费者与受保护成员新写入口，
不新增授权对象、权限词汇、事实 owner、表、状态、恢复协议或外部副作用。
UI 沿用既有成员抽屉及只读提示位置，不新增页面或要求新的 Figma 设计。

### 当前用户结果

1. 正常登录、成员/权限读取和普通业务账号的单店配置、打开、正常关闭、报告保持。
2. 受保护成员在应用内只读，不再提供单业务角色追加/撤销。说明由具有相应权限的
   管理员通过 ZITADEL 原生管理端维护，不自动授予原生管理权限或 browser.use。
3. 生产代码不调用 `GetAuthorizationRevision` / `ChangeAuthorizationRoleConditional`，
   不以探测这些接口是否存在来决定是否开放写入口。
4. 已取消命令不通过普通 UpdateAuthorization、重读、锁、自动重试或写后补偿执行。

## Owner 与保留项

- 登录、原生授权与角色分配：官方 ZITADEL；应用不成为第二授权事实源。
- 企业业务角色定义：现有 Membership RoleStore，原授权/版本规则保持。
- 命令和结果事实：原 Membership ReceiptStore，旧 key、payload、reservation、
  UNKNOWN 与审计保留。真实存在的隔离追加/撤销回执见
  [页面刷新读回证据](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954#issuecomment-6501221843)。
- UI 与 BFF：既有成员抽屉、同源代理和严格回执解析；不新建适配层/事实库。

已保存的定制业务角色操作只允许原主体、原企业范围内读取原回执；继续/核实路径
不得再次 dispatch、调用定制接口或修改历史 phase/reservation。页面不提供该类
操作的“继续原操作”，只读核对不得把后来观察伪造成旧请求 ACK。保留原 JSON
字段/枚举是原 canonical receipt 的只读事实合同，不提供旧执行器或新旧 fallback。
不迁移、映射、清理或回填历史记录。其他现有邀请/普通成员操作的恢复保持原合同。

Legacy decision: RETIRE（定制 reader/writer、资格探测、单角色执行器和新写入口）；
EXTRACT（原 ReceiptStore 已持久化事实的只读解析/呈现，停止旧执行能力）。

## 官方 API 写入影响核对

官方 [UpdateAuthorization](https://zitadel.com/docs/reference/api/authorization/zitadel.authorization.v2.AuthorizationService.UpdateAuthorization)
以完整 `roleKeys` 更新；请求遗漏的原角色被撤销，没有本次扩展的 expected-sequence。
这是本次产品边界选择的依据，不是对上游漏洞或所有并发安全的结论。

| 现有通路 | 官方合同与本次处理 |
| --- | --- |
| 登录、目录、权限读取 | 保持官方 OIDC/公开读取 API，去掉成员详情的定制 revision 调用 |
| 受保护成员单角色追加/撤销 | 取消入口、BFF/HTTP 新命令与恢复执行；零 provider 写入 |
| 普通成员角色替换 | 原 UpdateAuthorization 完整角色集与 live admission/observedVersion 保持；observedVersion 不是 provider CAS，不新增原子并发承诺 |
| 普通成员移除 | 原 DeleteAuthorization 与受保护目标拒绝保持；没有 expected-sequence，不宣称能原子覆盖管理端同时修改 |
| 邀请/创建授权 | 原 CreateUser/CreateAuthorization、单次 dispatch、原 key/UNKNOWN 保持；不因此开放对受保护成员的追加 |
| 企业角色目录定义 | 沿用原 RoleStore/native project-role owner 与既有准入；与成员角色分配写入分开，不迁移角色 |

应用可在下发前拒绝观察到的版本/权限变化，但不能通过官方非条件 API 证明整个
读取到写入窗口内不存在原生管理端的并发修改。本轮不新增“所有冲突已解决”承诺，
不扩大普通成员已有操作；重新开放受保护成员写入须另行设计与产品决定。

## 必要实现与验证

- TDD 验证受保护详情只读且无定制请求、提示准确；移除按钮不替代后端保护。
- 已取消 POST 入口与 BFF 不接受新命令；原键继续/Verify 和底层 writer 均不能
  重发该 kind，旧回执/UNKNOWN 可读且持久化记录不变。
- 原生 `platform_admin` 也按受保护角色处理；保留原内置管理员权限与禁止普通
  全量写回受保护目标的既有判断。
- 官方公开 API 的 HTTP 合同测试使用严格模拟服务；复用现有隔离初始化/登录能力
  和官方发行镜像做有界身份及普通账号 Store 路径验证，失败/未执行如实记录。
- 原定制 provider CI/单角色写入测试不是本方向通过证据。取消专属执行测试与其
  新增 CI 准入项，仓库原有必要 CI/保护保持；只读历史回执测试仍按现有事实保留。
- 最终同一 PR #1006 做增量独立复核；不重复已未变化的 Store/浏览器故障矩阵。

## 现有实例的未来切换与回滚边界

本轮只交付源码及可审查方案，不读取私密配置建立 fixture，不操作现有实例、
登录、数据、管理员权限、模块配置、店铺或信任设置，不创建远端 fork 新路线。
原 fork/PR #1/源码/证据先保存，不合并、发布、关闭或删除。

后续切回官方镜像必须另获部署授权，并先完成以下具体核对：

1. 固定已审官方镜像 digest、适配后应用镜像及原实际 Compose 配置栈；核对官方
   基线与已部署补丁具有相同 schema/ACL 预期，不执行 init/setup/迁移。
2. 真实备份和恢复能力按授权方案核对；原操作、角色、配置、Store 和会话作基线。
   特别核对是否有未终结的定制角色操作：存在则停止切换，保留原事实并报告；
   不通过释放 UNKNOWN/reservation 或删除记录强行通过。
3. 先部署不调用定制 API 的应用/页面，再以普通 start 切官方 provider；具体依赖
   与健康检查顺序在获准维护前绑定准确镜像，不重新配置角色/账号。
4. 只读验证登录、权限、受保护说明与 Store 可读；实际店铺开关如需重跑另按授权。
5. 回滚保留原镜像/配置与当前数据，普通 start 恢复；不恢复旧数据库/角色快照去
   覆盖切换后合法写入，不在未经确认时恢复已取消的应用内写入口。

合并、发布、部署、Issue 关闭和远端资源删除均另行确认。
