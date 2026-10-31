> #301 / #1030 [用户确认的浏览器安装完整退休边界](architecture/legacy-browser-install-closure-retirement-boundary.md)仅替代下文属于该闭包的旧KEEP/安装指引；其它有效规则不变。用户已确认冻结设计 f5e560cd7 的 §1–§7（含 §4.2/§6.1），IMPLEMENTATION_READY；同一 #1030 按36路径整批实施，不机械拆批。退出旧预装基础镜像、两个仓库安装CLI及browserinstall/httpclient/timeout专属闭包；官方SDK标准安装、显式指纹浏览器/匹配Profile、当前能力与Store观察保持。无新安装器、alias或试用前置；workflow/所有inventory/既有实例数据凭据缓存进程零改，#301 OPEN。源码自检、原CI及最终复核另记同一PR，不代表产品或运行验收，合并另授。以下原Git正文、既有顶注、历史计数和勾选逐字保留。

> #301 / #1010 — USER_CONFIRMED / IMPLEMENTING. [完整退休边界](../docs/architecture/legacy-restored-debug-closure-retirement-boundary.md) §1–§7（含 §4.2/§6.1）替代下文仅属于本批的旧 KEEP：旧 Go Amazon/1688 手工浏览器执行树、五个 debug 入口、手机号探针、db-query/json-simplifier、公开 YAML 与隐式配置入口退出。Chromium 安装闭包归 `integration/browserinstall`，当前 Collector 配置合同原样保留。当前采集、图像、身份权限、Store 观察同步保持；实际消费配置的四个命令须显式 `-config`，worker canary 无需配置。tools module 仅保留原 CI lint 固定用途；test-all 仅测 root。下方原文保留为历史，不作为已退休入口的现行操作指引。源级/CI 不等于运行验收；workflow/完整 inventory/计数/ceilings/工具版本及既有资源零改，#301 OPEN，无新增试用前置。

# ListingKit Code Guide

> Status: current lightweight guide.
>
> This file replaces the old manually maintained code wiki. It should stay short and point to source-of-truth documents instead of duplicating package maps, generated baselines, or large code summaries.

## 1. Project summary

The product direction is AI Commerce Agent Platform. ListingKit remains an
existing execution surface, not final UI authority or a permanent package facade.
Use [final UI / IA](product/final-ui-ia-authority.md) for BusinessTask / AI工作台 /
店铺中心 projection, and current domain contracts for business facts.

Use these documents as the source of truth:

- [README.md](../README.md) for product positioning and runtime navigation.
- [Architecture index](architecture/README.md) for document reading order.
- [Project boundaries](architecture/project-boundaries.md) for ownership and dependencies.
- [Repository structure](development/repository-structure.md) for layout and local artifacts.
- [Current status](refactoring/current-refactoring-status.md) for maturity and gates.

## 2. Current official runtime entrypoints

The complete maintained product/operational command list and build/script owners
live only in [Repository Structure](development/repository-structure.md#顶层目录约定).
`TestCmdContainsOnlyOfficialEntrypoints` checks the actual tree and documentation.
Maintained does not mean production accepted.

## 3. Package ownership snapshot

ACTIVE target ownership:

```text
cmd/* -> internal/app/* runtime assembly
      -> internal/listing/*, internal/product/*, internal/marketplace/*
      -> narrow ports implemented by platform/integration adapters
```

- Product facts use `internal/product/catalog.ProductSnapshot`; approved assets
  use `internal/product/asset.ApprovedAsset`. Source images are candidates until
  explicit approval; ImageAgent owns image execution and approval.
- Sourcing owns neutral envelopes/lineage; concrete source DTOs belong to adapters.
  Follow [Sourcing Handoff](product/product-sourcing-handoff.md) for implemented,
  prepared-only and target flows.
- App owns assembly/lifecycle/route aggregation, not domain rules. Marketplace
  rules go to `internal/marketplace/*`; submission has one existing owner.
- CURRENT STATE at `cae67730c5c0e645d708cb2f6814f14781962bb1`: root ListingKit,
  compatibility shells and some historical marketplace/infra paths remain.
  They are drain debt under the [Legacy Register](refactoring/legacy-register.md),
  not new feature homes. [Module mapping](refactoring/module-target-mapping.md)
  specifies EXTRACT destinations followed by RETIRE; retired Product roots stay absent.
- Identity uses verified effective Organization; see [Auth and Tenancy](architecture/auth-and-tenancy.md).
  Do not add tenantbridge consumers or parallel Agent IAM.
- Execution order belongs to [#137](https://github.com/qq550723504/task-processor/issues/137)
  and the current execution Issue. [next-phase-plan.md](refactoring/next-phase-plan.md)
  is historical evidence only.

## 4. Generated baselines

Dependency baselines and package maps are generated evidence, not hand-maintained documentation.

Run the generator when fresh evidence is needed:

```powershell
./scripts/dependency-baseline.ps1
```

Do not treat old generated snapshots as architecture authority. Stable boundary rules belong in `docs/architecture/*`; current execution posture belongs in `docs/refactoring/current-refactoring-status.md` and the active decision documents.

## 5. Maintenance rule

Keep this guide concise. If a section starts duplicating file lists, package maps, or old implementation details, either move the rule into a stable architecture document or delete the duplication.
