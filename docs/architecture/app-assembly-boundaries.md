# App Assembly Boundaries

[project-boundaries.md](project-boundaries.md) is the repository-wide entrypoint for default ownership and dependency direction. This contract narrows that rule to current `internal/app` composition; it does not create domain policy.

## Assembly Vocabulary

The application layer separates `build / initialize`, `register`, `start` and `coordinate`:

- Build / initialize reads explicit configuration, constructs current clients and services, and prepares dependency bundles.
- Register contributes ready components to module/lifecycle registries, without starting business effects.
- Start activates the configured lifecycle pieces; it does not install schema or invent facts.
- Coordinate handles supervision, cancellation, signals and orderly shutdown; domain recovery remains with its owner.

Where these package roles exist, `bootstrap` builds and registers, `runner` starts and supervises, and `consumer` assembles and coordinates messaging consumers. New work uses the existing current composition root; these terms are not a requirement to create missing packages or recreate deleted processors.

## Ownership

- `internal/app/httpapi` assembles current HTTP modules and injects trusted runtime dependencies. Each owning domain HTTP API registers its own routes.
- `internal/app/worker` assembles the corresponding current worker and native runtime.
- Product, Listing, Store, Agent, commercial and ledger modules keep their service, query, permission, transaction and recovery contracts.
- External clients and database pools stay in dedicated runtime assembly; domain packages cannot import app bootstrap/runtime to obtain them.
- A module registration descriptor is not a Tool Runtime, authorization decision or second service owner.

## Review Questions

Does the change belong to configuration/build, registration, lifecycle start or supervision? Is a domain policy leaking into the composition root? Are the actor, scope, fact owner, transaction and UNKNOWN responsibility still original? Does startup verify current prerequisites without DDL or unintended side effects?

Current source/import tests in `tests/` enforce the actual boundary. See [HTTP assembly](httpapi-assembly-boundaries.md) and [Temporal boundary](temporal-boundaries.md) for their specialized contracts.
