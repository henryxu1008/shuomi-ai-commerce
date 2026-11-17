# Product Sourcing Handoff

Source work follows the [Product Domain contract](../architecture/product-domain-contracts.md), [current owners](../architecture/project-boundaries.md) and **PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08**: fresh installation and empty business data.

## 相关领域入口

- [Sourcing：来源证据与采集、发布](../architecture/domains/sourcing/README.md)
- [Catalog：版本、我的数据和素材批准](../architecture/domains/catalog/README.md)
- [Research：市场证据与人工选品](../architecture/domains/research/README.md)

本页是业务交接概览，不重新定义上述 owner 的授权、事务、发布与来源可用性。1688 新采集通道以已批准浏览器 collector 为准，历史静态 HTTP 来源证据不能被当作新的默认采集能力。

## Current path

Verified effective Organization / actor / source access -> current acquisition adapter -> SourceEnvelope -> admitted InternalProducer / SRC transaction -> Catalog publisher -> immutable ProductSnapshot -> explicit asset approval -> deterministic downstream readiness.

| Responsibility | Current owner |
| --- | --- |
| 1688 browser evidence | `internal/integration/acquisition/a1688/browser` and `internal/integration/acquisition/browsercollector` |
| Amazon acquisition | `internal/integration/acquisition/amazon` and the admitted browser/plugin entry |
| POD / SDS specialization | `internal/integration/sds` and `internal/app/pod` |
| Source identity and publication values | `internal/product/sourcing` |
| Authorized operation and transaction assembly | `internal/app/productsourcing` |
| Canonical facts and approved assets | `internal/product/catalog` and `internal/product/asset` |

## Contract

Preserve source platform/native ID, canonical URL when available, source version/fingerprint, raw reference, warnings and lineage. A source platform is not the target sales platform. Marketplace transport and payloads stay downstream.

Use the explicit publication identity and current publisher. Same key with the same effective payload replays; changed payload conflicts. No checksum fallback, old task result inference or adapter-owned canonical store is introduced.

New-system idempotency, authorization, immutable versions and explicit approval remain required. This guide does not authorize migration, profile reuse, wrappers, mappings or cutover machinery. Asset candidates are not ApprovedAsset, and a saved product is not permission to publish.

## Recovery and delivery

Lookup and recovery bind the original actor, Organization and key. A hidden or missing operation is not evidence that no submission happened. Keep UNKNOWN with its original owner; never create a new key to guess at recovery.

Normal configuration and user operations are documented in [operations](../operations/README.md#products), the [capture extension](../../extensions/product-capture/README.md) and [batch import contract](../architecture/1688-batch-import.md). Current execution scope, real-source availability and acceptance evidence belong to the current Issue / PR, not a static status snapshot.
