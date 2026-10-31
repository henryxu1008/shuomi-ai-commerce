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

桌面首版发行包必须自包含，仅普通文件和目录；ZIP、tar.gz、tar.xz均先检查归档边界，再复用既有解压库，拒绝symlink、hardlink和特殊文件。含链接的常规macOS bundle需要发行方制作自包含包；目录支持darwin元数据不代表这些包已实机兼容。实际包/系统兼容性仍待后续获准验证，本轮不实际发行或下载CDN包。

## 已有检查入口与限制

```text
go test ./internal/browserkernel/... ./internal/app/runtime/currentapplication ./internal/app/httpapi ./cmd/browser-kernel-schema-init -count=1
go test -race -tags integration ./internal/integration/persistence/browserkernel ./internal/app/httpapi -run '^(TestKernelCatalogPostgres|TestCurrentApplicationKernelCatalogPostgres)$' -count=1 -v
```

后一个入口复用既有testcontainers，仅创建/终止任务隔离的PostgreSQL合成夹具；覆盖固定事实并发/回执、推荐下架、受限ACL、连接重开、默认关闭及真实HTTP接线。开发自检/CI并非用户验收；跨端真实发行、真实CDN包准备、现有实例部署和真实商家使用继续为 `NOT_RUN`。

## unified 共后端接线准备（#1042，不部署）

浏览器与 Web 的目标是复用同一 `current-application` 业务后端。日常候选为 A
`task-processor-unified-20261010`；B `listingkit-kernel-trial-20261030-01`
仅作隔离合成验收。保留 A 的项目名、全部卷、身份、业务数据和客户端；不从 B
复制账号、会话、配置或合成业务数据。此准备片不拥有 A 的部署权限。

kernel 后端已随 #1013 / PR #1014 进入 main：
`internal/browserkernel/httpapi/handler.go` 定义五条接口及原认证策略，
`internal/app/runtime/currentapplication/browser_kernel.go` 和
`internal/app/httpapi/browser_kernel.go` 负责可选装配与独立池校验，
`cmd/browser-kernel-schema-init` 是另行授权的 schema-owner 入口。
本片只复用这些代码；旧运行镜像是否包含它们仍须证明，补入口路由不等于补齐后端。

### 独立 opt-in ingress fragment

[`traefik-browser-kernel.fragment.yml`](../../deployments/docker/account-compose/traefik-browser-kernel.fragment.yml)
不是完整 dynamic 文件，也不由默认 Compose、overlay 或启动脚本加载。源码基础栈固定
`traefik:v3.6.8`，file provider 使用单个 `/etc/traefik/dynamic.yml` 且 `watch=false`。
规则采用 [Traefik v3.6 的 Host / Method / Path / PathRegexp 语法](https://doc.traefik.io/traefik/v3.6/reference/routing-configuration/http/routing/rules-and-priority/)，
不使用 v2 的命名正则占位符。

后续获准时，将 fragment 中两个 router 与一个 service 按键合并进**最终实际挂载文件的副本**：

- `browser-kernel-catalog`：仅 localhost、application HTTPS、GET 的精确目录路径。
- `browser-kernel-artifact`：仅相同入口下 GET、规范小写 UUID 的精确 `/artifact` 路径。
- `browser-kernel-api`：同网络命名空间 `http://127.0.0.1:8085`，即 Web 正在消费的后端。

优先级为 200，只覆盖这两类 GET。没有 strip/replace、query 过滤、header 改写、
forward-auth 或其他 middleware；原始 query 和 Authorization 交给后端严格校验，
复用 CurrentIdentity 鉴权，不能用入口匹配替代认证。HEAD/OPTIONS/PUT/POST、非规范 UUID、
尾斜杠/相邻路径、platform 发布/下架/推荐均不进入新路由，也没有 `/api/v1/*` 泛转发。
未匹配的请求继续沿既有路由处理；这不承诺它们在既有 Web BFF 上返回某个新状态码。

基础栈挂 `referrals-compose/traefik-dynamic.yml`；data-services、image trial、source-media
等 overlay 会按相同容器 target 替换完整 dynamic 文件。不能假设基础文件就是 A 当前文件，
也不能用 fragment 或某一个旧 overlay 文件覆盖最终组合。保留全部原 router、service、
middleware、TLS、entrypoint 和 mount/command 设置，包括 Web BFF、Store、身份、已有
Data Services/公开资产/CRL 路由。发现新增键冲突、未知最终挂载或版本不符即停止接线，
由部署 owner 提供获准的无秘密配置差异；本片不读取私有覆盖来推断。加载/重启须另获批准。

### 逐项切换与回退清单

以下各项均是后续部署 owner 的检查/批准项，**本片均未执行、NOT_AUTHORIZED**。
运行来源、schema 现状和现场组合未知时不能把准备片签为可用。

| 项目与责任 | 切换前必须具备的证据 / 单独授权 | 可逆范围与停止条件 |
| --- | --- | --- |
| 部署 owner：目标 | 确认 A 的原项目、原镜像、最终 overlay/mount/command 与所有卷；另行批准受影响服务及精确操作 | 保留项目名、端口、网络、身份、数据和全部卷；B 与真实客户端不操作。没有准确目标则停止 |
| 构建 owner：来源 | 从批准的固定源码提交复用 [现有构建工具](listingkit-image-build.md) 产生新 commit/tree/image-ID 收据；确认包含 kernel HTTP/runtime/schema 源码及所需 serving target | 构建与部署分别授权；不得把旧 revision 标签或容器 label 冲突猜成真实二进制来源。来源不完整标 `UNPROVEN` |
| 数据库 owner：独立库/角色/schema | 先确认当前 schema 现状、备份与恢复责任；单独批准独立 `browser_kernel` 库、既有受限 runtime role 和适用 fresh-init。没有批准不建库/授权/迁移 | 不共享其他 owner 的库/pool；不自动执行初始化。旧镜像与目标 schema 的兼容性未知时停止升级及镜像回退 |
| 配置 owner：模块与 origins | 在保留其他模块的前提下，单独批准 manifest `browserKernel`、独立池 1..4、最小权限/schema 校验及真实资产的精确 HTTPS origins | 原私有 manifest 的安全恢复副本由 owner 保管；不得提交连接串/秘密，不替换为 B 配置。缺模块/配置 fail closed |
| 部署 owner：路由信任 | 审查最终 dynamic 副本中仅新增本 fragment 三个键；确认 pinned v3 规则、原 Web/身份/TLS/Store 路由及原 Authorization 保持 | 只删除两个新增 router 与其新增 service 即可恢复原路由图；保留所有原键。单独批准最终文件加载，不能只改基础文件 |
| 身份/桌面 owner：公开身份 | 核对 A 的公开 API origin、issuer/audience 与既有 Native 合同；单独批准必要注册/授权、登录和端到端验证 | 本片不创建 Native client、不改证书/安全、不登录、不切换或启动真实客户端。身份不一致停止，不套用 B 身份 |
| 发行 owner：真实资产 | 单独批准资产来源、固定版本/大小/SHA、批准 origins 与真实发行；沿原平台鉴权入口操作 | 本 fragment 不暴露平台写入；不自动发行/推荐，不拿 B 合成记录证明真实发行，空目录不等于真实包已就绪 |
| 部署/验证 owner：应用与验收 | 前述 gates 齐备后，另行批准精确镜像/配置/路由切换、必要加载与原入口检查；记录当前 HEAD、image ID 与检查对象 | 本片不执行 apply、容器重启、网络健康探测、真实 Store/目录发行/下载。任何未知保持 `NOT_RUN` |
| 部署及数据库 owner：回退 | 切换前保管原精确 image ID/registry digest、原 manifest/dynamic 以及备份恢复方案；核验旧二进制与切换后 schema 的兼容性 | 获准后仅恢复目标 service 的原镜像和配置，保留全部卷。镜像回退不回滚 schema/数据，也不证明旧二进制兼容；恢复备份或 schema 操作需数据库 owner 另行批准，不自动执行 |

源码候选只能证明 opt-in 路由与收据边界。A 的后端来源、现场 API/身份/schema、真实发行、
CDN 下载和原生安装/切换仍为 `UNPROVEN` 或 `NOT_RUN`；当前进展与精确候选证据记在
Issue #1042 / 关联 Draft PR，不把任何历史合成验收签成 A 可用。

离线入口（不调用 Docker、不监听网络、不访问现场）：

```text
go test ./tests -run '^TestAccountComposeBrowserKernel' -count=1 -v
node --test scripts/build-listingkit-images.test.mjs
```

前者按既有 YAML 测试风格检查真实 fragment 的形状、Go 正则/方法/路径边界，以及与四种
现有 dynamic 文件在内存中合并/移除后原值保持；这是静态/受控证据，不是已运行 Traefik。
后者使用合成 Git 与 Docker 函数替身验证固定源码、标签、image ID、history 和收据。
