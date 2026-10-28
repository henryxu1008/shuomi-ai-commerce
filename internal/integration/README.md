# External Integrations

This directory is the long-term home of external system adapters.

Owned external adapters:

- OpenAI
- GRSAI image providers
- S3
- remote image HTTP
- Playwright
- marketplace API clients
- crawler adapters
- Dub affiliate attribution
- ZITADEL
- Casbin
- domain persistence adapters

Dub is an attribution adapter only. Shuomi remains authoritative for subscription,
order, commission, refund, earnings-ledger, and payout state. The Dub adapter uses
the REST API directly instead of importing Dub's generated Go SDK.

Historical retirement: the unused native Gemini image adapter was retired in #958.
Current governed Google text execution remains owned by `googleinteractions` and
`aicapability/einomodel`; the active image adapters listed above remain unchanged.
