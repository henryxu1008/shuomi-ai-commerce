# Crawler Integrations

Owns source-specific crawling adapters that feed product sourcing pipelines.

Current approved children:

- `internal/integration/crawler/a1688`

Current Amazon acquisition belongs to `internal/integration/acquisition/amazon`.
The original browser processor remains in `internal/crawler/amazon`, with its
direct debugging entry in `hack/debug/test-amazon`; the unused Amazon source
adapter and its exclusive factory/interface tails have retired under #301/#901.

Use this area for:

- browser or fetch runtime concerns
- anti-bot and source-specific access adaptation
- raw extraction contracts

Do not use this area for:

- marketplace publishing rules
- normalized product facts
- listing-task orchestration

Normalized handoff belongs in `internal/product/sourcing`.

Guardrail:

- crawler integrations must not import `internal/listingkit`, marketplace/workspace/publishing packages, or `internal/product/sourcing`; source identity and normalized result alignment flow toward product sourcing from outside these adapters.
