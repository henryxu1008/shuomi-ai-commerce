# External Integrations

This directory is the long-term home of external system adapters.

Owned external adapters:

- OpenAI-compatible source editing and exact organization credential lookup
- GRSAI image providers
- S3
- remote image HTTP
- Playwright
- marketplace API clients
- crawler adapters
- ZITADEL
- Casbin
- domain persistence adapters

Historical retirement: the unused native Gemini image adapter was retired in #958.
Current governed Google text execution remains owned by `googleinteractions` and
`aicapability/einomodel`; the active image adapters listed above remain unchanged.

#1033 retired the old OpenAI SDK Client/Manager/pool and generic model-review
assembly. Current source-edit adapters, governed text/Eino routes and published
credential/configuration versions remain under their current owners.
