# Platform Runtime

This directory is the long-term home of runtime infrastructure owned by the application itself.

The legacy RabbitMQ queue runtime is source-retired under the [confirmed #301 boundary](../../docs/architecture/legacy-queue-proxy-config-tail-closure-retirement-boundary.md). Shared broker declarations and diagnostic port-forwards remain with their existing owners.

The dormant host configuration, feature flags and tracing owners are source-retired under the [confirmed #301 boundary](../../docs/architecture/legacy-host-config-featureflag-tracing-retirement-boundary.md). Current configuration loading, environment overrides and validation remain with the existing `internal/core/config` entry points. No replacement runtime is introduced.

Owned runtime mechanisms:

- logging
- database runtime and Goose migration
- Redis
- worker pool
- Temporal dial
