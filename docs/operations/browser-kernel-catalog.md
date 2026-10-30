# 浏览器内核目录源码交接

Issue #1013。设计与精确 HTTP DTO 见 [发行目录 V1](../architecture/browser-kernel-release-catalog-v1.md)。当前交付为源码/Draft PR；未启用任何现有实例，也未发行、下载或安装真实内核。普通用户使用桌面版本选择器；本后端不提供另一套管理页面。

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

## 已有检查入口与限制

```text
go test ./internal/browserkernel/... ./internal/app/runtime/currentapplication ./internal/app/httpapi ./cmd/browser-kernel-schema-init -count=1
go test -race -tags integration ./internal/integration/persistence/browserkernel ./internal/app/httpapi -run '^(TestKernelCatalogPostgres|TestCurrentApplicationKernelCatalogPostgres)$' -count=1 -v
```

后一个入口复用既有testcontainers，仅创建/终止任务隔离的PostgreSQL合成夹具；覆盖固定事实并发/回执、推荐下架、受限ACL、连接重开、默认关闭及真实HTTP接线。开发自检/CI并非用户验收；跨端真实发行、真实CDN包准备、现有实例部署和真实商家使用继续为 `NOT_RUN`。
