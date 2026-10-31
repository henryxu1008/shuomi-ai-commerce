> #301 / #1029 [用户确认的完整输入链退休](../architecture/legacy-listingkit-input-config-closure-retirement-boundary.md)：当前 UI launcher 仅保留 ServiceApiBase/LISTINGKIT_SERVICE_API_BASE；旧 ApiBase/Task API 两 env 与自动读取集群 Secret 已退出。原显式进程 env/本地 .env 和 Auth.js handoff 保留，不新增登录步骤；下文旧 API/dev/token 操作仍为历史。 两组30路径在同一 #1029 整批实施，用户已确认冻结设计 e343a825b 的§1–§7（含§4.2/§6.1），IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW。退出旧手工上传配置及旧 UI Task API/隐式集群 Secret 导入；当前 UI/Auth.js/Service API、图像/采集、imgproxy 两有效 Secret 键与原安全检查、Store 观察保留。workflow/inventory/既有实例数据凭据进程不动，无试用前置，#301 OPEN。以下原文、历史、计数和勾选完整保留；源码 CI/最终复核证据另见同一 PR，不代表产品或运行验收。

# Historical ListingKit Local Debug Reference

> #852 / #29 / #301 [用户确认的 API token 工具完整退休](../architecture/legacy-api-token-tools-retirement-boundary.md)，冻结设计 `e6ac780d7`：旧取 token、保存与 auth-check 三脚本已从仓库删除；下文旧 API token 文件/环境约定及脚本命令仅为历史，不是当前操作或试用要求。当前 Auth.js handoff、原生身份及 Login V2 探针保持原 owner；当前采集、Store 和独立 SHEIN 登录/Store 同步保留。历史正文原样，未读取现有凭据或操作实例/数据，不增加试用前置。

> #846 / #29 / #301 [已确认完整退休](../architecture/legacy-local-acceptance-closure-retirement-boundary.md)：旧 API/dev launcher 源及其专属验收编排已删除，下文旧命令不能启动当前应用。当前 UI launcher/测试、token handoff、原生身份 CLI/库和通用工具保留；既有 Account Compose / 私密 current-application manifest 入口、#529 身份与发布边界及其余历史正文保持。未操作实例或数据，不增加试用前置。

> #529 / #301 retire the old API/local-stack source and coordinated release
> identity-preflight. The startup and token-helper examples below describe the
> historical ListingKit stack, not a current-application delivery procedure.
> Current internal-trial delivery follows [Account Compose](../../deployments/docker/account-compose/README.md)
> and its existing private current-application manifest. The current command is
> `go run ./cmd/current-application -config <absolute-private-manifest>`.
> Existing instances and data remain with their current runtime/domain owners.
> Production release is **NOT_READY**; this reference grants no environment,
> credential, deployment or data operation.

ListingKit local scripts require the same ZITADEL authentication flow as every
other environment. There is no local identity, trusted identity header, or
authentication-disable mode.

## Start the local stack

Run the local stack normally:

```powershell
.\scripts\start-listingkit-local-dev.ps1
```

Before starting, provide a valid local ZITADEL issuer/client configuration in
`.env` (or inherited process environment). The API returns `503` for protected
routes until `ZITADEL_ISSUER_URL` and `ZITADEL_CLIENT_ID` are configured, and
then requires a verified bearer token. The UI redirects unauthenticated users
through the configured ZITADEL login flow.

Start each side separately when needed:

```powershell
.\scripts\start-listingkit-local-api.ps1
.\scripts\start-listingkit-local-ui.ps1
```

## Direct API Calls

For authenticated API calls, prefer the local machine-token helpers instead of
repeatedly copying browser tokens:

```powershell
.\scripts\listingkit-fetch-machine-token.ps1 -ApiBaseUrl http://localhost:8085
.\scripts\listingkit-auth-check.ps1
```

The helper writes the token under `.local\listingkit-api-token.txt` and can export `LISTINGKIT_API_TOKEN` for scripted checks.

## Retired coordinated release entry

The [approved retirement boundary](../architecture/release-identity-preflight-retirement.md)
removes the old identity-preflight command, metadata readers, resolver, image
and Job. The API/UI workflows and PowerShell release wrapper now only reject
before credentials, build/push or mutation, including rollback and historical
receipt inputs. Do not run the removed preflight or prepare old Organization
metadata, mappings or directory grants to enable current delivery.

The old protocol remains in Git history and the explicitly historical
[release reference](../../deployments/kubernetes/listingkit-workbench/README.md).
Current authentication, scoped roles, worker/schema and data-protection
contracts remain with their existing owners; source retirement does not revoke
historical runs, cluster permissions or stored instances.

## Safety

Do not use caller-supplied `X-User-*` or tenant headers as identity. ListingKit
derives identity from the verified ZITADEL bearer token in every environment.

Current Organization and scoped-role access use their admitted owners and
verified identity contracts. Historical package-local legacy-data fixtures do
not authorize metadata mapping, a trusted header or an authentication bypass
for the current application.
