# Platform Runtime

Current runtime mechanisms include logging, database lifecycle and Goose, worker pool, Temporal dial and module registration.

Configuration loading and validation belong to `internal/core/config`. Current consumers use native Redis clients. Database shared-pool lifecycle belongs to `internal/platform/database`; platform registration does not create another resource, permission or business owner.

External protocols belong to `internal/integration`; current domain semantics remain in their owning modules. See [current boundaries](../../docs/architecture/project-boundaries.md).
