# 系统管理员模块展示修复

Design Basis: N/A。修复既有只读元数据投影，不改变权限词汇、实际授权、租户、成员编辑资格、状态机、持久化或外部副作用。准入依据先于生产代码修改记录于本文件；本轮用户仅授权本地源码、测试、必要说明与提交，因此不创建或修改外部 Issue / PR。关联既有成员与角色交付 #598，未重开其完整范围。

使用者：企业成员与管理员。场景：在“我的账户 → 企业空间 → 成员与权限 → 角色权限”查看管理员模块，在成员列表与详情查看同一角色定义。结果：只有固定 administrator 策略完整覆盖的可用模块显示“已包含”；其余显示“未包含”，未开放模块继续显示“尚未开放”。自定义角色保留原添加、移除和保存流程。

依据：[Enterprise custom roles v1](../architecture/enterprise-custom-roles-v1.md)、[当前 UI / IA](../product/final-ui-ia-authority.md)，沿用 Figma 文件 `tg48P46SSXl6TBy9lZwg63` 的角色权限 `1536:473`、成员列表 `1532:359`、成员详情 `1541:386`，不新增页面或调整布局。

根因：`Roles()` 把整个 available catalog 当成系统管理员权限，`project()` 却返回空模块，均未读取固定策略。authz 保持固定 Casbin 策略及模块映射 owner，Membership 仅复用统一系统角色定义。投影只用于元数据；业务请求与成员实际权限继续使用已注入的共享 / scoped authorizer。查看者自定义角色、配置的 protected subject 与平台角色不参与固定系统角色的计算。权限集合为空的模块不能作为完整覆盖。

明确不做：授予 `browser.use`、编辑受保护成员、修改 ZITADEL / IAM / 账号 / 配置、启动试用或共享服务、操作店铺环境、合并、推送或部署。受保护角色完整列表写回仍为 IMPLEMENTATION_NOT_READY / B1；本修复不解除该阻塞。

基线：最新 main `0aab8d5ebe03140e05c2a9c9716d3665da0a0ace`。交接补丁基于未合并组合 `a19cf48e892264cbe89cd997b59d76f88dcdb0dd`，SHA256 `4764FD0B1FDB89F95E96733BE3F7C4AB67AEB22214D8F86378ED7B2D70F0098A`。main 尚无 `store-browser` 模块；本次不复制该未合并功能，只对现有投影作语义适配，并用固定组合源隔离验证该窄模块仍不被管理员隐式继承。

Legacy decision: RETIRE（错误的全目录 / 空目录投影）。Reusable behavior: 现有 Casbin 固定策略和 authz 模块映射。Current owner: authz 固定策略投影 → Membership 系统角色定义 → 现有角色 UI。Cutover/deletion condition: 两个投影入口同时替换；无 fallback、双读写、迁移或第二事实源。

必要验证：先以成员目录 / 详情一致性与系统角色只读 UI 回归确认缺陷，再验证完整权限覆盖、空权限与 unavailable、查看者 / protected subject 隔离；运行 authz / Membership 相关测试及现有前端角色 / 成员回归、类型和 lint 检查。完整路径形成后做一次独立最终复核。开发自检不代替用户试用或上线验收。

本地开发自检：

- RED：成员查询测试准确复现角色目录有模块、列表 / 详情为空的矛盾；前端系统角色的两个只读状态用例因缺少“已包含”失败，已有自定义角色用例保持通过。
- GREEN：`go test -mod=readonly -count=1 ./internal/authz ./internal/organization/membership/...` 通过，包含既有请求 authorizer 调用边界、成员命令 / 保护资格和 HTTP 合同回归。
- 前端角色组件、成员页面、角色 API、成员 API 四个测试文件共 59 项通过；新增 fixture 补齐类型标注后，角色组件 5 项复查、完整 `pnpm typecheck` 与两个改动 TSX 文件的 ESLint 均通过。依赖使用 frozen lockfile 离线安装，未改依赖或 lockfile。
- 将固定 `a19cf48` 的相关源码与测试依赖归档到本工作树 `build/administrator-module-projection/a19-source`，叠加相同 Go 修改与交接浏览器回归，以上四个 Go 包均通过。证明该组合中管理员不隐含 `browser.use`，显式自定义角色仍赋予该权限，保护资格保持。此项为源码组合测试，未触碰组合分支原工作树或运行实例。
- 真实角色组件 / CSS / typed client 的隔离浏览器检查使用合成 GET 返回；1280 宽浅色和 390 宽深色中只读状态可见，自定义角色仍可编辑，无横向溢出、页面异常或写请求。工件保存在本工作树 `build/administrator-module-projection/preview`；这是组件视觉自检，不是完整应用 / Figma 像素验收。

检查期间的临时归档准备路径错误与测试 fixture 类型错误已修正；只将随后实际通过的执行记录为 GREEN。未执行完整 CI、生产构建、真实登录 / provider / 权限写入或部署，用户试用仍待独立确认。固定组合包含未合并依赖，不能把该组合的验证描述为 main 已具备店铺浏览器能力。
