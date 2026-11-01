> **#1050 继续整理：保留合同参考与历史记录。** 保留已勾选的 ProductImage builder 抽取过程；不恢复旧 ProductImage owner 或默认装配。
>
> 当前阅读依据：[httpapi-assembly-boundaries.md](../../architecture/httpapi-assembly-boundaries.md)、[legacy-hard-cut-policy.md](../../refactoring/legacy-hard-cut-policy.md)。旧 owner、操作命令和 agent / sub-skill 指令不构成当前派工或真实操作授权；预期输出、勾选和实际验证继续区分。下方原 Git 正文完整保留。

# HTTPAPI Module Builder Phase 6B Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Continue shrinking `internal/app/httpapi/modules.go` by moving the `productimage` module builder and its private assembly helpers into the owning business package.

**Architecture:** `internal/productimage/httpapi` becomes the canonical home for its HTTP runtime builder, model provider wiring, image pipeline component resolution, repository bootstrap, and publisher selection. `internal/app/httpapi` keeps only a thin delegating `buildImageModule` entry point plus still-shared ListingKit-facing image helpers.

**Tech Stack:** Go, Gin, existing worker pool and product image pipeline

---

### Task 1: Move productimage module builder

**Files:**
- Create: `internal/productimage/httpapi/bootstrap.go`
- Modify: `internal/app/httpapi/modules.go`

- [x] **Step 1: Move productimage module builder and repository bootstrap**
- [x] **Step 2: Move model provider and asset publisher assembly needed only by productimage**
- [x] **Step 3: Rewire app/httpapi to delegate to the new builder**

### Task 2: Remove dead centralized helpers

**Files:**
- Modify: `internal/app/httpapi/modules.go`

- [x] **Step 1: Remove unused image-only helper functions from modules.go**
- [x] **Step 2: Keep still-shared ListingKit image helpers in app/httpapi for now**

### Task 3: Verify extracted builder

**Files:**
- Test: `internal/app/httpapi`
- Test: `internal/productimage/...`

- [x] **Step 1: Format changed files**
- [x] **Step 2: Run targeted tests**

```powershell
go test ./internal/app/httpapi ./internal/productimage/... -count=1
```

Expected: PASS

**Follow-up:** Extract `listingKit` builder and the remaining shared image/temporal/repository assembly helpers in a final phase.
