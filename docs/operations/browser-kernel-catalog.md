# 浏览器内核目录源码交接

Issue #1013。设计与精确 HTTP DTO 见 [发行目录 V1](../architecture/browser-kernel-release-catalog.md)。普通用户使用桌面版本选择器；#1056 在 Web 工作台补充平台维护页面，后端继续提供同一发行 API。源码交付或构建成功不表示已更新任何现有实例或完成真实发行。

## 工作台维护入口

已登录且持有既有全局 `listingkit.platform_admin` 权限的维护者，从「我的账户 → 企业空间 → 内核发行管理」进入 `/workbench/account/organization/kernel-releases`，查询指定 OS、arch 和客户端版本的兼容目录。列表按 `nextCursor` 翻页，包括本页为空但仍有下一页的情况。登记表单位于其 `/new` 子页面，与列表分开。

浏览器只调用同源 `/api/account/kernel-releases` BFF。BFF 沿用服务端 Auth.js 会话和当前 Workbench authorizer 权限，不接受浏览器提供的 bearer、角色或企业权限替代；发布仍由原 platform handler 再次校验。BFF 不转发企业选择、用户声明或 Cookie 到发行 owner，不开放新的公共 platform 写接口。下载 URL 只作为已授权维护者的固定元数据，不在此下载或运行制品。

登记先校验固定九字段、编号及读回客户端版本，保留原编号和载荷，再调用原 PUT。只有同一编号的 artifact 读回与全部固定元数据一致且推荐为 false 时，页面显示「已登记并读回确认」。服务端明确拒绝或确认尚未发送时保留可编辑输入；发送后结果不确定、响应丢失、读回失败或不匹配时保留待核实请求，不会自动重发、换编号或把查询不到当作未写入。待核实期间可以显式查询本次登记，不能创建另一条记录。这个非凭据请求快照按用户保留在当前标签页 sessionStorage 中；重开同一页面仍需先核实。

部署前核实 Web 服务到同一后端的既有 `LISTINGKIT_SERVICE_API_BASE` 接线、kernel 模块和现有管理员权限。最小候选是更新 Web UI（页面与同源 BFF）；若现有 ingress 未转发该 BFF 路径，应由部署 owner 核实并单独批准必要接线，不泛转发 `/api/v1/platform/*`。本源码任务不执行部署、权限变更或真实目录写入。登录或管理员权限不足时按原身份流程处理，不通过改角色完成验收。

原 PUT 合同还允许同编号的幂等响应为 `state=withdrawn`。仅当严格响应中的编号匹配本次请求时，页面将其作为明确的非成功终态：解除该请求的 pending、显示原编号及已撤回状态，允许显式登记另一条，不再查询必定不可见的 artifact，也不显示成功。编号不匹配、响应结构无效或网络结果未知时仍保留原 pending。

## 接线与默认行为

`cmd/current-application` 读取可选 manifest `browserKernel`。省略时不打开目录数据库、不挂五条路由，不改变其他模块或账号权限。启用时要求独立 `browser_kernel` 数据库、既有 `browser_kernel_runtime` 角色、1..4连接和1..8个规范HTTPS资产origin。origin不含路径、默认443端口、凭据、query或通配符。

示例结构只用于后续获准的独立部署准备，不能原样写入现有私有配置：

```json
{
  "browserKernel": {
    "database": {
      "host": "127.0.0.1",
      "port": 15432,
      "user": "browser_kernel_runtime",
      "password": "PRIVATE_VALUE_SUPPLIED_BY_OPERATOR",
      "database": "browser_kernel",
      "maxConnections": 4
    },
    "artifactOrigins": ["https://assets.example.test", "https://cdn.example.test"]
  }
}
```

runtime只验证现有schema/列/主键及最小权限，失败即拒绝服务，不自动初始化或修复ACL。新的空库与受限角色须由operator另行获准准备；独立初始化源码入口是 `go run ./cmd/browser-kernel-schema-init --runtime-role browser_kernel_runtime`，schema-owner连接仅从私有环境变量 `BROWSER_KERNEL_SCHEMA_DSN` 读取。它拒绝其他数据库名，事务内建表与授权，不创建数据库/角色、不产生发行记录，也不在已有schema上做迁移。启动应用不调用此命令。

## 消费和维护接口

- 普通当前登录身份：`GET /api/v1/browser/kernel-releases?os=windows&arch=amd64&clientVersion=1.0.0`。遍历 `nextCursor` 到空；中间页可能为空，不能提前结束。目录仅含版本摘要。
- 原生：`GET /api/v1/browser/kernel-releases/{releaseId}/artifact`，带相同三项兼容参数。响应直接JSON，额外含固定格式/size/SHA256/URL和 `downloadOrigins`。先绑定当前releaseId/原生平台和版本；下载client不带API bearer/cookie，每跳在HTTPS白名单内、最多3跳。大小和摘要通过前不得解压/登记，归档路径/链接校验和本地提交由桌面owner实现；后端不取制品网络内容。
- 维护者：`PUT /api/v1/platform/browser/kernel-releases/{releaseId}` 发布完整固定metadata。`POST .../{releaseId}/withdraw` 空体下架；`PUT .../{releaseId}/recommendation` 发送 `{"recommended":true}` 或false。只复用现有全局 `listingkit.platform_admin` 权限，不赋新角色、不接受企业admin或伪造header。

首次发布的全部固定字段不可修改；相同ID/载荷重试返回原事实，变更返回409。推荐初始false，不自动latest。下架单向且不清理已有固定绑定/缓存或切换运行内核。503可能发生在提交响应丢失后，维护者使用相同ID与相同载荷查询/重试；不能把503当作未写入再换ID重复发行。当前不提供安全kill、强制升级或跨端下载租约。

API token只进入API请求头，不能发给资产存储，metadata/下载URL不能进入renderer或摘要。源metadata必须是稳定固定版本URL；CDN跳转中的临时签名query仅内存使用，不记录/保存。现成CDN或GitHub资产须把初始和实际跳转域名都列入部署origin。

桌面首版发行包必须自包含，仅普通文件和目录；ZIP、tar.gz、tar.xz均先检查归档边界，再复用既有解压库，拒绝symlink、hardlink和特殊文件。含链接的常规macOS bundle需要发行方制作自包含包；目录支持darwin元数据不代表这些包已实机兼容。实际包/系统兼容性仍待后续获准验证，本轮不实际发行或下载CDN包。

## 已有检查入口与限制

```text
go test ./internal/browserkernel/... ./internal/app/runtime/currentapplication ./internal/app/httpapi ./cmd/browser-kernel-schema-init -count=1
go test -race -tags integration ./internal/integration/persistence/browserkernel ./internal/app/httpapi -run '^(TestKernelCatalogPostgres|TestCurrentApplicationKernelCatalogPostgres)$' -count=1 -v
```

后一个入口复用既有testcontainers，仅创建/终止任务隔离的PostgreSQL合成夹具；覆盖固定事实并发/回执、推荐下架、受限ACL、连接重开、默认关闭及真实HTTP接线。开发自检/CI并非用户验收；跨端真实发行、真实CDN包准备、现有实例部署和真实商家使用继续为 `NOT_RUN`。
