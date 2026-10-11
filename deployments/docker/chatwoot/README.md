# 平台人工客服（Chatwoot Community Edition）

按用户于 2026-10-11 选择，改用 **Chatwoot 官方 API 仅由服务器调用**：平台提供简洁聊天窗，人工使用原生后台。用户浏览器不嵌入 Widget SDK、不接收 Chatwoot API token 或附件签名 URL。开发/本地试用与正式上线分开；不进行专项复现或额外评审。

适用于 Issue #650：平台已登录用户的使用咨询、报错截图、人工回复和离线留言。客服会话与附件只由 Chatwoot 保存；当前版本没有 AI、店铺买家客服、平台业务操作或匿名官网聊天。设计与身份边界见 [platform-support-chatwoot.md](../../../docs/architecture/platform-support-chatwoot.md)。

复用 [官方 Docker 部署](https://developers.chatwoot.com/self-hosted/deployment/docker)及[官方 API channel 接入](https://www.chatwoot.com/hc/user-guide/articles/1677839703-how-to-create-an-api-channel-inbox)，固定 `chatwoot/chatwoot:v4.18.0-ce`。准备 Docker Linux containers。该服务不会自动接入共享平台；正式开放须确定私有服务网络、受控坐席入口、人工和备份位置。

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

打开 `http://localhost:36500` 完成首次安装向导，使用自己的管理员凭据；可取消向官方订阅安装信息。注册保持关闭。为平台客服使用**独立账户和 API inbox**，加入实际人工坐席；该账户不创建 Website inbox。API callback URL 留空，本轮通过认证 BFF 定期读取人工回复，不增加 webhook 服务。

按官方文档准备：

1. 记录专用 account ID 和 API inbox ID。
2. 建立只属于该客服账户的专用集成操作员，读取其 Profile → Access Token，存入仓库外私密文件，例如 `C:\private\support\api-token`。令牌授予此账户访问；不要使用拥有其他业务账户权限的通用管理员令牌。无需超级管理员 Platform API token。
3. 人工在该 inbox 回复；集成操作员不加入自动分配队列。图片最多 5 MB，支持 PNG/JPEG/WebP。
4. 客服记录归属由服务器登录身份和 conversation subject 确定，坐席不要改动平台创建的 identifier、source ID 或 `platform_support_subject`。后台内部笔记不发给平台用户。

令牌仅由 BFF 用于官方 Application API；浏览器不调用 Chatwoot 原站、公开客户 API 或 WebSocket。不需要 API CORS，也不把平台/OIDC token 发给 Chatwoot。无 SMTP 时，离线回复需重新打开平台客服查看，不承诺邮件通知。坐席后台仅向运维/客服在受控内部网络开放；远程访问使用现有访问控制，不能将 Chatwoot 原站整体公开转发给平台用户。

## 接通平台现有入口

Next BFF 在运行时读取以下配置，全部是服务端变量；不要设置为 `NEXT_PUBLIC_*` 或镜像构建参数：

```text
CHATWOOT_ORIGIN=http://localhost:36500
CHATWOOT_ACCOUNT_ID=<专用 account ID>
CHATWOOT_INBOX_ID=<专用 API inbox ID>
CHATWOOT_API_TOKEN_FILE=C:\private\support\api-token
```

也支持服务端 `CHATWOOT_API_TOKEN`，但不能与 `…_FILE` 同时配置。origin 无路径/查询/凭据，必须不同于平台 origin；HTTP 仅允许 loopback 和专用 Docker alias `chatwoot-rails`，其他连接使用 HTTPS。保留现有 ZITADEL 和平台配置。无配置时按钮禁用，服务不可达时显示失败，不伪造成功。每次读取/发送/下载均核对服务器身份、账户、inbox、contact 和会话归属；浏览器不能指定联系人。

使用 account-compose 的获准新试用实例时，额外加入 opt-in overlay：

```powershell
$env:CHATWOOT_ACCOUNT_ID = '<专用 account ID>'
$env:CHATWOOT_INBOX_ID = '<专用 API inbox ID>'
$env:CHATWOOT_API_TOKEN_HOST_FILE = 'C:\private\support\api-token'
$env:CHATWOOT_NETWORK_NAME = 'platform-support-trial_support-api'
# 保留原实例的 COMPOSE_PROJECT_NAME、所有 env 和全部既有 overlays。
# 只在该实例已获接线授权后执行：
docker compose -f deployments/docker/account-compose/docker-compose.yml `
  -f deployments/docker/account-compose/docker-compose.chatwoot.yml config --quiet
# 在实际完整 Compose 文件列表末尾加此 overlay，按原启动路径重建 listingkit-ui。
```

overlay 给既有 `runtime-network` 加专用内部网络，UI 保留其共享网络命名空间，仅 BFF 读取专用令牌只读挂载。DB/Redis 不加入该网络。不要把省略其他 overlays 的示例当成原实例重启命令；重建 network namespace 需按原完整实例启动路径。HTTPS 校验问题须修正证书，不能绕过。独立 Compose 仅发布 loopback 坐席端口；客户图片通过已登录 BFF 读取。

平台用户登录 → 工作台右上角 **联系客服** → 描述问题/添加截图 → 发送。人工登录 Chatwoot → 专用 API inbox 查看和回复。打开窗口约每 5 秒读取回复，也可点“刷新记录”。关闭重开显示个人历史，更多消息可点“查看更早消息”。切换用户/企业清空旧窗口。记录不向企业成员共享，不自动附带页面 URL/业务事实。发送超时后保留草稿，显示结果未确认；先看记录，再决定是否手动重新发送，不自动重发。

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

保留独立 Chatwoot API account、私有入口和服务器令牌边界；不要额外开放 Website widget 或 Client API。API token 轮换/撤销通过现有运维流程执行。升级前复核官方 API 响应及正常两用户个人历史、图片、人工回复；不恢复旧 Widget 路径。此接线不是平台业务授权接口。
