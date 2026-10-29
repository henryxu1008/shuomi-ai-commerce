# 本机 TLS 撤销维护边界（#965）

Status: IMPLEMENTATION_READY. Independent architecture review round 1 completed before production implementation; 0 BLOCKER, two IMPLEMENTATION_TEST findings below.

2026-10-29 状态说明：以下第 1–7 节保留最初设计准入时点的范围和证据。
随后另行授权的隔离实测及 2026-10-27 本机 TLS/CRL/正常 Schannel 检查已记录在
[#965](https://github.com/qq550723504/shuomi-ai-commerce/issues/965)；当时的 NOT_RUN/NOT_AUTHORIZED 不能覆盖后来的准确执行记录。
冻结的维护、安全、失败停止和每次现场动作授权边界不变；本次仅同步 main 和整理文档，不重复现场操作，也不代表 Store 产品验收通过。

## 1. 用户结果与依据

硕米用户使用安装好的浏览器，经既有本机 HTTPS 入口访问店铺工作台。维护操作由部署操作者负责，不成为浏览器用户必须执行的脚本步骤。Refs [#965](https://github.com/qq550723504/shuomi-ai-commerce/issues/965)、[#954](https://github.com/qq550723504/shuomi-ai-commerce/issues/954)、[#956](https://github.com/qq550723504/shuomi-ai-commerce/pull/956)。用户已表示没有现成域名/可信证书方案；没有因此授权生成或安装任何真实凭据。

当前证据：唯一一次正常 Schannel 请求在 HTTP 前报 `CRYPT_E_NO_REVOCATION_CHECK (0x80092012)`；公开叶证书没有 CDP/AIA，链仅叶与根。内部一次匿名 401 JSON PASS 不覆盖此问题。现有 `account-compose/bootstrap.sh` 用临时 CA 私钥一次签发并清理，缺少可复用签名状态及撤销发布。不得重新执行含 IAM/DB 初始化的 bootstrap 来维护 TLS。

Must：维护入口独立；OpenSSL 原生 CA/CRL；签名状态与 serving 隔离；CRL 不经同一待验证 TLS 连接；失败/未知即停且留证；不改变身份/业务权限。Out of Scope：通用 PKI/OCSP、域名采购、自动续期 daemon、新服务/端口、IAM/DB、浏览器 UI、新登录协议、自动部署/信任安装、旧 CA 猜测性迁移。Figma N/A（无页面/交互改动）。

## 2. 最小承载与事实 owner

沿用现有 Traefik application entrypoint 与 `listingkit-ui` 服务（3001）。新增且仅新增一条非 TLS router：`Host(localhost) && (Method(GET) || Method(HEAD)) && Path(/.well-known/local-tls/root.crl)`；无重定向、认证、retry 或路径前缀通配。现有业务 router 继续显式 `tls: {}`。不改变监听/发布端口。CRL URL 在签发时固定为 `http://localhost:<application-port>/.well-known/local-tls/root.crl`；端口必须为 1..65535，默认 35444。该 URL 一旦写入证书不得随意变更。

Traefik 的有 `tls` router 只处理 TLS；一个 entrypoint 可以分别匹配 HTTP 与 HTTPS。现有 Next.js standalone 镜像已复制 `/app/public`，认证 proxy matcher 仅 `/workbench/:path*`。可使用单独只读目录 mount 到 `/app/public/.well-known/local-tls`，在 UI 启动前已有 `root.crl`，之后只更新同一名字的完整文件。**不依赖启动后新文件发现**，不改 frontend 生产代码。不加载此候选、不重启 UI/Traefik，是本批现场边界。

- 官方依据：[Traefik TLS router](https://doc.traefik.io/traefik/reference/routing-configuration/http/tls/overview/)、[Next.js public folder](https://nextjs.org/docs/app/api-reference/file-conventions/public-folder)、[OpenSSL ca](https://docs.openssl.org/3.0/man1/openssl-ca/)。
- CA/签发数据库、序列号、CRL number 与签名 key：仅显式离线维护入口 owner；不挂到 serving 容器。
- `public/`：仅公开 root certificate 与 DER CRL，可只读挂载；没有任何私钥。
- `releases/<serial>/`：localhost certificate + server private key；仅后续经授权提取到既有 TLS serving volume，不挂 CA state 根。
- 原 IAM/DB、授权、组织、Store 事实和业务 retry/UNKNOWN owner 不变。

## 3. 维护合同与调用路径

新增 POSIX shell 运维入口 `deployments/docker/account-compose/local-tls-maintain.sh`，只依赖既有 shell/coreutils/OpenSSL，不自建密码学实现。命令显式为 `init STATE PORT`、`issue STATE`、`refresh-crl STATE`、`revoke STATE SERIAL`。没有默认动作、远程下载、Docker/Compose、trust-store、API 或 bootstrap 调用，也没有 TLS bypass/test-key 开关。

`init` 只允许不存在的绝对新路径：创建 0700 state、0600 私有材料；生成新 root、OpenSSL 原生 CA index/serial/crlnumber 和首次空 CRL。根 basicConstraints CA:true/pathlen:0，keyUsage keyCertSign+cRLSign；根期限十年，服务器证书三十天，CRL 七天。旧根没有签名 key 的场景不猜测、不扫描、不复用：后续实际切换必须单独审批新根信任及所有消费者证书，并保留原工件。

`issue` 在已就绪 state 内生成一个新 serial release，经 `openssl ca` 签发；SAN 固定 localhost/127.0.0.1/::1，EKU serverAuth、CA:false、CDP 固定既有 HTTP URL。用 `openssl verify` 对链、purpose、DNS/IP 和 CRL 检查后才把 release 标为 complete；不写 current serving files、不吊销旧证书、不切换服务。

`refresh-crl` 使用原生 `openssl ca -gencrl`，原生验证签名，转换 DER 后原子替换同目录公开 `root.crl`；旧 CRL 留在 history。CA 证书剩余有效期不足一天则拒绝。更新频率由操作者负责，必须在 nextUpdate 之前；无后台 scheduler。过期/不可达的 CRL 应让系统验证失败，不能自动降低验证强度。

`revoke` 只接受属于本 CA 既有 `issued/<SERIAL>.pem` 的十六进制 serial，原生 CA 撤销后刷新 CRL。不是“撤销成功就继续使用该 leaf”：被撤销的现有部署必须由操作者另行处理；本命令不运行/替换产品。

contract → shell input validation/private-state ownership → OpenSSL CA → public CRL/release artifacts → **另行授权的**现有 volume/精确 router 接线 → Windows Schannel。签名状态本身是唯一 CA 事实，不写第二份自建证书数据库。

## 4. 持久化、失败与恢复

串行所有权：state 的 0700/current-uid 和固定目录/文件不为符号链接是前置；`mkdir .operation` 原子取得一次操作锁。OpenSSL 原生 index/serial 是既有成熟状态格式，不添加事务框架。每步 stdout/stderr/退出状态保留在操作目录；成功移入 history，失败、信号中断、未知保留 `.operation` 并拒绝后续命令。

不承诺跨多个文件的事务回滚。CA 签发/撤销或序列号可能已前进；CRL 可能尚未发布。任何异常不能盲重试或删除锁，必须保留目录、由操作者检查日志与原生 CA state 后提出新的精确恢复动作；本批不实现自动修复/补偿/重试。此时旧公开 CRL 仍可存在至其 nextUpdate，必须报告尚未发布的撤销，不能宣称即时失效。签发完成也不等于部署完成。

在生成新 CRL 之前保留旧公开 CRL。临时输出与最终 CRL 同目录，完整签名验证后 rename；公开工件转换/写入失败不覆盖旧工件。操作日志不打印 key 内容，不复制现有业务配置或凭据；私钥文件只会在将来明确运行相应签发命令时由 OpenSSL 使用。真实执行要求 POSIX 权限，不把 Windows chmod 当作私钥访问隔离。

## 5. 代码与测试切片

一个辅助 Draft PR：上述独立脚本、精确 router/公开目录的 opt-in Compose 模板、操作文档、原 tests 内的离线回归。现有 bootstrap 保持不动，避免未审批更改全量初装/身份初始化；旧一次性签发只作为已诊断且待切换的现场，不增设 fallback。Legacy decision: EXTRACT TLS-only maintenance behavior；不新增旧 CA 兼容桥。

先 RED 证明入口/缺失 CDP 合同及故障拒绝未满足，再 GREEN：

- 正确固定 SAN/CDP、CA/leaf usage、CRL 原生验证调用；公开目录没有私钥。
- 用假 OpenSSL 命令只产生非密码学文本 fixture；**不产生任何真实 key/CA/certificate/CRL**。验证实际 shell 的参数、路径、调用次数、stdout/stderr/退出与文件边界。
- 任一步失败或未完成 `.operation` 后拒绝后续调用；初始已存在 state、无就绪标记、错误端口/serial、symlink、不安全目录权限拒绝，且无 OpenSSL 副作用；旧 CRL 在转换失败前后不变。
- 不调用 bootstrap/IAM/DB、网络、Docker、trust-store；命令不自动执行下一类运维/运行操作。
- 正常 CRL 签名/Windows 获取与实际 Schannel 请求 **NOT_RUN**，fake executor 只证明编排/失败边界。

精确源码 HEAD CI、独立源码增量复核与冻结设计评审分开。CI 绿色不代替实际证书签发/信任/CRL 可达性或店铺产品验收。

## 6. 后续现场准入（本批不执行）

需用户在动作前批准：准确新 state 与影响范围、实际 CA/leaf/CRL 生成、私钥访问范围、所有使用旧根的消费者清单与信任切换、公开文件 mount 和单一 HTTP router、需要的准确一次重启/请求、证书正常验证及失败停止方案。不得用“没有现成证书”替代上述权限。

最终实际验收须包含：系统信任/SAN/期限/撤销检查全部开启的 Windows GET，401 与 JSON 的完整状态/头/正文；CRL 真实获取和有效期；实际服务/路由范围没有漂移。撤销负例必须使用隔离、另行授权的凭据与环境，不撤销现场正在使用证书。本机通过不代表公开 Internet 部署或 Store 授权/业务验收通过。

## 7. Architecture review

Round 1: 独立只读 reviewer `/root/store_design_review`，受审文档 SHA256 `EB95ED1433DC730B286A73C0D6971902C0292B84DD2ACD046D13F7079779911E`，main 基线 `6199063811dbb7904aa678735c1b7bb660b8d186`。结论 IMPLEMENTATION_READY，0 BLOCKER。

两项 IMPLEMENTATION_TEST 在既定合同内落实，不扩大架构：OpenSSL 3 的 CRL 验签可能打印 `verify failure` 却退出 0，须固定本 CA/关闭默认 trust sources，并要求明确 `verify OK`，空/未知输出失败留锁、不发布；CA 必须显式 `unique_subject=no`，使新旧 localhost 叶证书并存。离线回归须覆盖两者。依据：[OpenSSL crl 原生实现](https://raw.githubusercontent.com/openssl/openssl/openssl-3.0.18/apps/crl.c)、[原生 CA 配置](https://docs.openssl.org/3.0/man1/openssl-ca/)。

此准入只允许普通离线实现与无真实凭据 fixture。真实签发、信任安装、路由/服务变化及 Schannel 验收仍 NOT_RUN / NOT_AUTHORIZED。设计和 fake 测试不宣称部署就绪。
