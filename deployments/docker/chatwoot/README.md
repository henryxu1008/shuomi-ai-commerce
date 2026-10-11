# 平台人工客服（Chatwoot Community Edition）

当前状态：**集成草稿，未开放**。隔离试用未满足“用户只能看到自己的客服记录”，Issue #650 保持 Blocked；下方命令是保留的接线说明，不构成启动、合并或上线批准。用户已要求停止进一步专项复核。当前专用测试实例已停止，命名卷保留；正式开放前必须解决已确认的记录隔离问题。

适用于 Issue #650：平台已登录用户的使用咨询、报错截图、人工回复和离线留言。客服会话与附件只由 Chatwoot 保存；当前版本没有 AI、店铺买家客服、平台业务操作或匿名官网聊天。设计与身份边界见 [platform-support-chatwoot.md](../../../docs/architecture/platform-support-chatwoot.md)。

复用 [Chatwoot 官方 Docker 部署](https://developers.chatwoot.com/self-hosted/deployment/docker)，固定 `chatwoot/chatwoot:v4.18.0-ce`。这是独立可选服务，不会自动部署到现有平台实例。准备 Docker Linux containers；正式使用前需要独立 HTTPS 支持域名、实际坐席和备份位置。HTTP 仅用于 loopback 隔离试用。

## 启动一个新的隔离实例

在仓库根目录运行 PowerShell。复制 `.env.example` 到仓库外的私密目录，替换 `SECRET_KEY_BASE`、`POSTGRES_PASSWORD`、`REDIS_PASSWORD`；不要把密钥提交到 Git。`SECRET_KEY_BASE` 应为至少 64 位随机 hex。生产 `FRONTEND_URL` 使用真实 HTTPS 支持域名。私密文件仅允许运行用户/服务账号读取。

```powershell
$env:COMPOSE_PROJECT_NAME = 'platform-support-trial'
$env:CHATWOOT_ENV_FILE = 'C:\private\support\chatwoot.env'
$env:CHATWOOT_PORT = '36500' # 选未占用端口
$supportCompose = @('-f', 'deployments/docker/chatwoot/docker-compose.yml')
docker compose @supportCompose config --quiet
docker compose @supportCompose up -d postgres redis
docker compose @supportCompose --profile init run --rm prepare
# prepare 成功后再启动；它负责 db:chatwoot_prepare。
docker compose @supportCompose up -d rails sidekiq
docker compose @supportCompose ps
```

打开 `http://localhost:36500`，完成首次安装向导并使用自己的管理员凭据。若不希望向官方订阅安装信息，取消勾选向导的订阅复选框。注册默认关闭；首次安装向导不需要开启公开注册。邀请必要坐席、建立专用 **Website** inbox，并将坐席加入该 inbox。

在 inbox 设置中：

1. 将 Website URL / Allowed domains 设为实际平台 origin（包括协议和端口）。
2. 开启 **Enforce identity validation**。复制网站 token 和该 inbox 原生身份校验密钥（通常为 24 字符）；密钥保存到单独的私密文件，例如 `C:\private\support\identity-secret`，不加 JSON 引号。
3. 启用附件；关闭不需要的开聊邮件收集/邮件续聊。没有 SMTP 时，离线回复由用户重新打开平台客服查看，不承诺邮件送达。
4. 根据真实坐席安排配置工作时间、离线文案和回复时间，不沿用与实际服务不符的默认承诺。

本接线不使用 Chatwoot 管理员 API token，不需要开启 API CORS，不把平台/OIDC token 发送给 Chatwoot。

## 接通平台现有入口

Next BFF 在运行时读取以下配置，全部是服务端变量；不要设置为 `NEXT_PUBLIC_*` 或镜像构建参数：

```text
CHATWOOT_ORIGIN=https://support.your-domain.example
CHATWOOT_WEBSITE_TOKEN=<专用 Website inbox 的公开 token>
CHATWOOT_IDENTITY_SECRET_FILE=C:\private\support\identity-secret
```

也支持服务端 `CHATWOOT_IDENTITY_SECRET`，但不能与 `…_FILE` 同时配置。支持 origin 必须与平台 origin 不同，必须是完整 origin、无路径/查询/凭据；非 loopback HTTP 被拒绝。平台现有 `LISTINGKIT_PUBLIC_BASE_URL` 和 ZITADEL 配置保持正确。没有有效配置时“联系客服”按钮禁用；配置正确但服务不可达时显示连接失败，不伪造成功。

使用 account-compose 的获准新试用实例时，额外加入 opt-in overlay：

```powershell
$env:CHATWOOT_ORIGIN = 'https://support.your-domain.example'
$env:CHATWOOT_WEBSITE_TOKEN = '<公开 Website token>'
$env:CHATWOOT_IDENTITY_SECRET_HOST_FILE = 'C:\private\support\identity-secret'
# 保留原实例的 COMPOSE_PROJECT_NAME、所有 env 和全部既有 overlays。
# 只在该实例已获接线授权后执行：
docker compose -f deployments/docker/account-compose/docker-compose.yml `
  -f deployments/docker/account-compose/docker-compose.chatwoot.yml config --quiet
# 在实际完整 Compose 文件列表末尾加此 overlay，按原启动路径重建 listingkit-ui。
```

overlay 只把专用密钥文件只读挂载给 UI BFF，不改其他 owner、业务表或共享数据。不要把省略现有 overlays 的示例当成原实例重启命令。HTTPS 平台使用 HTTPS 客服域名；浏览器或 Node 不信任证书时修正域名/证书配置，不关闭校验或跳过警告。生产 TLS 反代必须支持 `/cable` WebSocket；该独立 Compose 仅发布 loopback Rails 端口。

平台用户登录 → 工作台右上角 **联系客服** → 原生聊天窗口 → 开始会话、发送文字或截图。坐席登录 Chatwoot → inbox 查看对话、回复。关闭后重新打开会按同一服务器身份显示历史；切换用户/企业会销毁旧窗口。客服记录是个人记录，不向企业成员共享，也不会自动附带当前页面 URL 或业务事实。

## 数据保留、备份与恢复

数据库、Redis、附件使用该独立 Compose project 的三个命名卷。记录 `COMPOSE_PROJECT_NAME` 和 env/密钥文件位置；以后仍使用同一 project、端口和完整配置。

```powershell
docker compose @supportCompose stop
docker compose @supportCompose start
```

`stop/start` 保留数据。不要用 `down -v` 重启或清理需要保留的会话/附件。升级前备份，在获准窗口停止 Rails/Sidekiq 写入，再备份 DB 和 attachments；保存版本、project 名、私密配置及恢复说明。PowerShell 不用文本重定向保存二进制 dump：

```powershell
docker compose @supportCompose stop rails sidekiq
docker compose @supportCompose exec -T postgres pg_dump -U chatwoot -d chatwoot -Fc -f /tmp/chatwoot.dump
$supportDb = docker compose @supportCompose ps -q postgres
docker cp "${supportDb}:/tmp/chatwoot.dump" 'C:\private\support\backup\chatwoot.dump'
$supportRails = docker compose @supportCompose ps --all -q rails
# 从停止的 Rails 容器导出附件，保存同一时间的 DB/附件配对。
docker cp "${supportRails}:/app/storage" 'C:\private\support\backup\storage'
docker compose @supportCompose exec -T redis /bin/sh -ec 'export REDISCLI_AUTH="$REDIS_PASSWORD"; redis-cli SAVE'
docker compose @supportCompose stop redis
# AOF 也需要停止写入/rewrite 后完整导出。
$supportRedis = docker compose @supportCompose ps --all -q redis
docker cp "${supportRedis}:/data" 'C:\private\support\backup\redis-data'
docker compose @supportCompose start redis rails sidekiq
```

在**新隔离恢复 project** 使用相同版本、对应配置和空卷；先启动 Postgres，保持 Rails/Sidekiq 停止。拷回 dump 后使用 `pg_restore -U chatwoot -d chatwoot --exit-on-error /tmp/chatwoot.dump`。创建但不启动 Rails/Redis 容器，分别拷回完整 `storage` 内容到 `/app/storage/` 和 `redis-data` 内容到 `/data/`。确认归属/权限后启动 Redis/Rails/Sidekiq，验证恢复的历史及截图可读。不要对仍有业务数据的数据库执行清空/覆盖；真实恢复/数据操作须单独授权。Redis 用于队列，升级/恢复时先停写且检查待处理任务，不把丢弃 Redis 队列当作正常无损恢复。

## 检查与限制

开发自检：相关 Vitest、typecheck、lint；隔离真实 CE 集成与模拟登录必须分别记录。健康检查和 fixture 通过不等于真实平台 ZITADEL 用户验收。人工运营、SMTP 邮件、正式域名/TLS、备份目标及生产接线要在对应开放范围确定。

Chatwoot 原生访客 token 由 provider 管理；本接线只在内存中临时使用，关闭窗口不等于撤销已被捕获的 provider 凭据。身份端点不可用于平台业务授权。升级 Chatwoot 时复核 native `loaded/config-set/toggle-open` 和 contact/set_user 协议，并验证原生握手及本人历史隔离后再切换版本。
