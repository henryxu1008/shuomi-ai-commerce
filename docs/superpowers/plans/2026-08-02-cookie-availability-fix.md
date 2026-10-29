> #301 / #985 **SUPERSEDED：独立 SHEIN 登录与身份回写完整退休**。按[用户已确认的完整边界](../../architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md) §1–§7/§4.2/§6.1，本文件涉及独立登录 CLI/runtime/service、Cookie/Profile/RedisConfig、Store 执行归属/身份回写及旧 session transport/商家仓库查询的 KEEP、安装或运行指引仅为历史；这些专属闭包已从源码退出，不留 alias/fallback/替代 owner。当前 Store 观察同步、官方授权及其它当前能力保留。既有表/记录/权限/Redis/Profile/凭据/进程无操作，运行停用 NOT_RUN/NOT_AUTHORIZED；不增加试用前置，#301 OPEN。以下起点 main 完整原正文、历史、批准、失败、数字与勾选逐字保留。

# Cookie Availability Status Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the SHEIN login status report a Cookie as available only when its stored payload contains usable Cookie entries.

**Architecture:** Keep Redis TTL as expiry metadata, but make `has_cookie` come from the existing payload validation path. The UI will use `has_cookie` as the source of truth and distinguish an existing-but-invalid payload from no login record.

**Tech Stack:** Go, Redis/miniredis, React/TypeScript, Vitest/Testing Library.

## Global Constraints

- Do not change Redis data, task status, or production configuration as part of this code change.
- Preserve the existing unrelated working-tree changes on `master`.
- Keep the fix limited to Cookie availability semantics and regression coverage.

---

### Task 1: Backend usable-Cookie status

**Files:**
- Modify: `internal/sheinlogin/service.go:117-165`
- Test: `internal/sheinlogin/service_test.go`

**Interfaces:**
- Consumes: `RedisStore.HasCookie` and `RedisStore.CookieTTL` for one tenant/store pair.
- Produces: `AccountStatus.HasCookie` representing parseable, non-empty Cookie payloads while `CookieTTL` continues to expose expiry metadata.

- [ ] **Step 1: Write the failing test**

  Add a test that stores `{"cookies":[]}` with a positive TTL, calls `Service.Status`, and asserts `HasCookie == false` while `CookieTTL > 0`.

- [ ] **Step 2: Run the focused backend test and verify it fails**

  Run: `go test ./internal/sheinlogin -run TestServiceStatusRejectsEmptyCookiePayload -count=1`

  Expected: FAIL because `Status` currently derives `HasCookie` from TTL alone.

- [ ] **Step 3: Implement the minimal backend fix**

  Keep the TTL lookup for display, but call `HasCookie` for the boolean field. Propagate malformed-payload errors instead of reporting a false success.

- [ ] **Step 4: Run the focused backend test and verify it passes**

  Run: `go test ./internal/sheinlogin -run TestServiceStatusRejectsEmptyCookiePayload -count=1`

  Expected: PASS.

- [ ] **Step 5: Run the full backend package tests**

  Run: `go test ./internal/sheinlogin -count=1`

  Expected: PASS with zero failures.

### Task 2: Frontend status presentation

**Files:**
- Modify: `web/listingkit-ui/src/components/listingkit/stores/store-login-status.tsx:6-24`
- Test: `web/listingkit-ui/src/components/listingkit/stores/store-login-status.test.ts`

**Interfaces:**
- Consumes: `SheinLoginAccountStatus.has_cookie` and `cookie_ttl`.
- Produces: `sheinLoginStatusLabel` returning `Cookie无效` for a positive TTL with `has_cookie=false`, and `已登录` only for `has_cookie=true`.

- [ ] **Step 1: Write the failing UI tests**

  Add cases for `{has_cookie:false,cookie_ttl:2592000}` returning `Cookie无效`, and `{has_cookie:true,cookie_ttl:2592000}` returning `已登录`.

- [ ] **Step 2: Run the focused UI test and verify it fails**

  Run: `pnpm --dir web/listingkit-ui exec vitest run src/components/listingkit/stores/store-login-status.test.ts`

  Expected: FAIL because the current fallback treats any positive TTL as logged in.

- [ ] **Step 3: Implement the minimal UI fix**

  Remove the TTL fallback from the success branch and add an invalid-cookie branch before the generic unauthenticated branch.

- [ ] **Step 4: Run the focused UI test and verify it passes**

  Run: `pnpm --dir web/listingkit-ui exec vitest run src/components/listingkit/stores/store-login-status.test.ts`

  Expected: PASS.

- [ ] **Step 5: Run the related UI test suite**

  Run: `pnpm --dir web/listingkit-ui exec vitest run src/components/listingkit/stores/tenant-store-directory-panel.test.tsx src/components/listingkit/stores/store-login-status.test.ts`

  Expected: PASS with zero failures.

### Task 3: Final verification and diff review

**Files:**
- Inspect: `internal/sheinlogin/service.go`
- Inspect: `web/listingkit-ui/src/components/listingkit/stores/store-login-status.tsx`

- [ ] **Step 1: Run backend and frontend focused tests together**

  Run the commands from Tasks 1 and 2 again from the isolated worktree.

- [ ] **Step 2: Check diff hygiene**

  Run: `git diff --check` and `git status --short`.

  Expected: only the plan, backend status/test changes, and frontend status/test changes are present; no unrelated production data or configuration changes.

- [ ] **Step 3: Review the final behavior**

  Confirm the empty-payload case cannot produce a green login status, while a valid non-empty payload remains green and still shows its TTL.
