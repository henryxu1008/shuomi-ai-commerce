> #301 / #972 窄范围替代：本文涉及旧1688 LocalAgent PoC CLI/client/deviceauth/未挂载HTTP/内存Job/Runner、旧integration/crawler/a1688及父crawler空壳、sds/adapter/product_source、asset/policy和两个专属Prepare的KEEP/实施/操作要求，已由本批完整退休设计替代。这不取消当前公共/扩展/批量/原浏览器采集、SDS/POD、资产审批/有效规则、共享身份权限、独立SHEIN登录或Store同步；其它产品边界与永久禁令保持。以下起始Git blob、历史批准/失败/数字/勾选逐字保留，不恢复迁移或试用/生产前置，不授权任何既有资源操作。

# 1688 Local Agent POC

This POC lets an authenticated Windows CLI run the existing public 1688
crawler in the user's local browser environment. The API owns tenant identity,
job leases, result validation, and `SourceEnvelope` reconstruction.

## Run

Start the API locally with the normal development configuration. Then run the
confirmation-gated script from the repository root:

```powershell
pwsh -NoProfile -File .\scripts\1688-local-agent-acceptance.ps1 `
  -ApiBaseUrl http://127.0.0.1:18086 `
  -IssuerURL $env:LISTINGKIT_ZITADEL_ISSUER_URL `
  -ClientID $env:LISTINGKIT_ZITADEL_CLIENT_ID `
  -ProjectID $env:TASK_PROCESSOR_LISTINGKIT_ZITADEL_PROJECT_ID `
  -Url https://detail.1688.com/offer/1052008074197.html `
  -Confirm CREATE-LOCAL-AGENT-JOB
```

The CLI performs device authorization, claims one tenant-scoped job, runs the
public processor in a short-lived browser context, and submits either a
narrow product snapshot or a sanitized failure category. The access token is
kept in process memory only. No cookies, source-account IDs, browser profiles,
proxy credentials, target stores, ListingKit drafts, or publish calls are part
of this flow.

The CLI automatically uses an installed Chrome executable when one is found on
Windows. Pass `-BrowserPath` to the acceptance script (or `-browser-path` to
the CLI) to select a specific executable.

The default device-auth scopes include the supported ListingKit roles and the
`admin` alias. For a deployment with custom platform-admin roles, pass the
complete scope string with `-Scopes`/`-scopes` or set `ZITADEL_SCOPES`.

Omit `-Url` only when a pending job was created separately through the API.
The POC stores jobs in API process memory; restarting the API drops them.
