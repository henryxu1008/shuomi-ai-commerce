# Product Domain Contracts

> 归类说明（2026-11-20）：Catalog 目录是不可变商品版本这一交接结果的导航入口，不改变本文件作为 Product 跨域总合同的效力。Sourcing、Catalog、Collection、Asset、Review、ImageAgent 与下游 Listing/Marketplace 各自保留原文所列权责。此前 [#1343](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1343) / [#1344](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1344) 批次只建阅读导航、未迁移根合同；本批仅在核对消费者后调整这一个合同的位置，不复制、重启或改变任何能力。

Product owns source publications, canonical catalog versions, current collections and approved product assets. Read [Project Boundaries](../../project-boundaries.md) for dependencies and [source handoff](../../../product/product-sourcing-handoff.md) for the admitted import path.

## 领域导航

本文件仍是 Sourcing/Catalog/Collection/Asset 的现行权威总合同。按任务查阅可先进入 [Sourcing](../sourcing/README.md)、[Catalog](README.md)、[Research](../research/README.md) 三份导航；它们只链接原合同和源码，不新增任何第二事实 owner。对应实际代码应以 [Project Boundaries](../../project-boundaries.md) 和当前源代码为准。

## Current owners

| Fact or behavior | Owner |
| --- | --- |
| Source identity, envelope and publication contract | `internal/product/sourcing` |
| Admitted acquisition and durable operation composition | `internal/app/productsourcing` and current acquisition repositories |
| Canonical immutable product versions | `internal/product/catalog` |
| User/Organization collections and explicit transfers | current Product collection owner |
| Approved assets and approval provenance | `internal/product/asset` |
| Deterministic review/proposal values | `internal/product/review` and `internal/product/review/proposal` |
| Image execution and external effect facts | `internal/imageagent`; Product receives explicit candidates and approved results |

## Publication and versioning

Source platform/native identity, canonical URL when available, raw reference, warnings and lineage follow the current SourceEnvelope / SourceIdentity contract. Missing source facts stay unknown; a provider DTO or image candidate is not a canonical Product fact.

The admitted InternalProducer / SRC transaction publishes through Catalog's existing publisher. The same explicit publication identity and same payload replay the original result; changed payload conflicts. Canonical versions and publication identity are not inferred from an execution task, browser profile or a checksum fallback.

Read and apply operations bind ProductKey, catalog version, actor and Organization. Later source observations do not silently rewrite a frozen proposal or an approved platform draft.

## Assets, review and downstream use

Source images, generated outputs and uploads remain candidates until explicit approval. ApprovedAsset preserves source/approval provenance and immutable content identity. Browser-supplied URLs, dimensions or a model's claim do not prove an approved asset or a charge outcome.

Marketplace adaptation and deterministic readiness consume the current Product/Asset contracts without taking ownership of canonical facts. Listing material, draft and submission status stay at their current Listing/Marketplace owners.

## Safety and recovery

- Product code uses neutral values and domain ports; it does not import provider SDKs or integration implementations.
- Keep verified effective Organization and original actor/member scope across submit, lookup, replay and explicit transfer.
- Persist the original operation identity before side effects. Response loss or a scoped 404 does not prove that an operation was never submitted.
- Use the existing same-owner transaction boundary for publication and collection effects. Preserve original idempotency, immutable version and cancellation behavior.
- Readiness, approval, resource admission and real-platform authorization remain independent gates. A saved source product or successful generation does not authorize publishing.

The [greenfield baseline](../../../product/greenfield-no-legacy-migration.md) applies: no old execution/task/ID/profile migration, compatibility pipeline or second source of truth.
