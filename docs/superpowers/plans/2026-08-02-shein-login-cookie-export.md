> #301 / #985 **SUPERSEDED：独立 SHEIN 登录与身份回写完整退休**。按[用户已确认的完整边界](../../architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md) §1–§7/§4.2/§6.1，本文件涉及独立登录 CLI/runtime/service、Cookie/Profile/RedisConfig、Store 执行归属/身份回写及旧 session transport/商家仓库查询的 KEEP、安装或运行指引仅为历史；这些专属闭包已从源码退出，不留 alias/fallback/替代 owner。当前 Store 观察同步、官方授权及其它当前能力保留。既有表/记录/权限/Redis/Profile/凭据/进程无操作，运行停用 NOT_RUN/NOT_AUTHORIZED；不增加试用前置，#301 OPEN。以下起点 main 完整原正文、历史、批准、失败、数字与勾选逐字保留。

# SHEIN Login Cookie Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent the SHEIN Worker from treating the SSO login page as authenticated or persisting an empty Cookie payload after login.

**Architecture:** Mirror the verified Python flow: prioritize a visible login form over generic success selectors, complete the login or verification flow, visit the account-specific seller target route, then export browser state. A non-empty Cookie list is required before the automation returns success; otherwise it captures an artifact and returns a failed login result.

**Tech Stack:** Go, Playwright, Redis/miniredis, Vitest, GitHub Actions, Kubernetes.

## Global Constraints

- Preserve the existing Cookie availability UI/backend changes in this isolated worktree.
- Do not delete or rewrite Redis state while implementing or deploying the fix.
- Do not classify `.main-content` alone as a reason to bypass a visible login form; the Python reference retains that selector as a fallback after form detection.
- The production release must use immutable Worker, API, and UI image tags built from the same committed source revision.

---

### Task 1: Model login-surface precedence and target route

**Files:**
- Modify: `internal/sheinlogin/automation.go:148-260,1030-1055`
- Test: `internal/sheinlogin/automation_test.go`

**Interfaces:**
- Produces: a pure login-surface decision that prefers `login_form` over a generic `logged_in` signal, and `postLoginTargetURL(Account) string`.
- Consumes: `Account.LoginURL` and the existing form/verification/login page probes.

- [ ] **Step 1: Write failing tests**

  Add table tests proving that a visible form plus a generic logged-in selector resolves to `login_form`, and that SHEIN SellerHub and SSO accounts resolve to `https://sellerhub.shein.com/#/spmp/commdities/list` and `https://sso.geiwohuo.com/#/spmp/commdities/list` respectively.

- [ ] **Step 2: Run the focused test and verify RED**

  Run: `go test ./internal/sheinlogin -run 'TestLoginSurfaceDecision|TestPostLoginTargetURL' -count=1`

  Expected: FAIL because the decision/target helpers do not exist.

- [ ] **Step 3: Implement the minimal surface decision**

  Add a small pure decision helper and update `waitForLoginSurface`/`StartLogin` so a visible username/password form takes precedence over generic success selectors. Add the account-specific target URL helper using the same routes as the Python reference.

- [ ] **Step 4: Verify GREEN**

  Run: `go test ./internal/sheinlogin -run 'TestLoginSurfaceDecision|TestPostLoginTargetURL' -count=1`

  Expected: PASS.

### Task 2: Require exported Cookie state before success

**Files:**
- Modify: `internal/sheinlogin/automation.go:187-260`
- Test: `internal/sheinlogin/automation_test.go`

**Interfaces:**
- Produces: `exportAuthenticatedBrowserState(...)` returning a normal browser state only when `cookieCount(...) > 0`; an empty state produces an artifact-backed failed `AutomationResult`.
- Consumes: the existing `cookieOnlyBrowserState`, `cookieCount`, `artifactResult`, and `ErrNoUsableCookie` helpers.

- [ ] **Step 1: Write failing tests**

  Add tests for the pure exported-state validation helper: empty cookies must return `ErrNoUsableCookie`; a non-empty cookie list must be preserved as cookie-only state.

- [ ] **Step 2: Run the focused test and verify RED**

  Run: `go test ./internal/sheinlogin -run TestValidatedCookieOnlyBrowserState -count=1`

  Expected: FAIL because the validation helper does not exist.

- [ ] **Step 3: Implement the minimal export path**

  Navigate to `postLoginTargetURL(account)` before each successful browser-state export. Use the validation helper for the already-logged-in, recovery, normal-login, and verify-code completion paths. On an empty export, capture an artifact and return failure rather than success.

- [ ] **Step 4: Verify GREEN**

  Run: `go test ./internal/sheinlogin -run TestValidatedCookieOnlyBrowserState -count=1`

  Expected: PASS.

### Task 3: Verify and release the complete fix

**Files:**
- Inspect: `internal/sheinlogin/automation.go`, `internal/sheinlogin/service.go`
- Inspect: `.github/workflows/shein-login-worker-deploy.yml`, `.github/workflows/listingkit-deploy.yml`, `.github/workflows/listingkit-ui-deploy.yml`

- [ ] **Step 1: Run backend and UI verification**

  Run: `go test ./internal/sheinlogin -count=1`; `pnpm --dir web/listingkit-ui test`; `pnpm --dir web/listingkit-ui run typecheck`; `git diff --check`.

- [ ] **Step 2: Commit and push the isolated branch**

  Stage only the SHEIN automation, Cookie status, UI status, regression tests, and plan files. Commit with a focused message and push `codex/cookie-availability-fix`.

- [ ] **Step 3: Deploy immutable images from the same commit**

  Dispatch `SHEIN Login Worker Deploy`, `ListingKit API Deploy`, and `ListingKit UI Deploy` with `source_ref` set to the committed SHA and immutable `image_tag` set to its first eight characters. Wait for each rollout.

- [ ] **Step 4: Production acceptance test**

  Re-login store 985, then verify Worker logs include a non-zero exported Cookie count, Redis DB 9 key `shein:cookie:246:985` has a non-empty Cookie array, `has_cookie=true`, and the store pause key no longer reports `auth_expired`.
