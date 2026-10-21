# Platform Runtime

This directory is the long-term home of runtime infrastructure owned by the application itself.

The legacy RabbitMQ queue runtime is source-retired under the [confirmed #301 boundary](../../docs/architecture/legacy-queue-proxy-config-tail-closure-retirement-boundary.md). Shared broker declarations and diagnostic port-forwards remain with their existing owners.

Owned runtime mechanisms:

- config loading
- logging
- database runtime and Goose migration
- Redis
- worker pool
- Temporal dial
- feature flags
- tracing and technical observability
