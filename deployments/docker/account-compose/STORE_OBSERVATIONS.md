# Store observation internal preparation

This optional final layer implements [the approved preparation stage](../../../docs/architecture/store-observations-preparation-runtime.md). Use the existing base, knowledge, data-services, modules, ecoservices, chat and acquisition-plans layers, then `docker-compose.store-observations.yml`. Backend uses the existing `data-services-runtime` image target; UI and support settings remain those of the current program.

The current Store schema must already be installed, `workbench_stores` empty, and `shein_observations` absent. This adds only the original four observation tables and minimal serving grants through `store-center-schema-init -observations-only`; it does not run base Store migrations or other initializers. Nonempty Store or an existing observation schema fails without deleting or changing facts. Obtain explicit authority for the named retained database before installation and save a private Store backup and original manifests/images first. A serving constructor never provisions schema.

For an authorized retained update, build the current backend/UI and schema-init images without changing source bind paths; apply a last private image overlay if needed. Using the same project/environment and complete Compose file set, first `docker compose … run --rm --no-deps store-observations-init`, then `docker compose … up -d --no-deps current-application listingkit-ui`. Do not run the previous initializers. Keep support private mounts/network/credentials and all named volumes. Restart the serving containers directly using the existing retained startup script; the worker lives inside the backend process.

The initializer verifies its source/output digests and completed marker on repetition; it performs no schema operation on a completed run. A partial run or changed base/output is an operator inspection stop, not permission to reinstall or clear state. Never print either manifest or owner credentials.

Normal entries: `/workbench/store-products` and `/workbench/store-orders`. Current enterprise read permissions still apply. Their capabilities report `runtimeReady=true`, `providerConfigured=false`, `available=false`, `canSync=false` once the real worker is started. No product/order/statistic/sync command read or write is admitted. Store catalog/Cockpit consumers receive no saved-observation service during preparation, preserving their existing unknown/unavailable projection. Queue `store-observations-current` registers the existing `StoreObservationsV1` workflow; startup/fatal/shutdown readiness remains the existing lifecycle. Real SHEIN, models, payments and user acceptance are NOT_RUN.

## Non-destructive rollback

Stop the backend/worker first. An old manifest with observations disabled cannot use newly enabled observation grants. Under the same explicit Store owner authority, perform this narrow transaction:

```sql
BEGIN;
REVOKE ALL ON shein_observations.commands, shein_observations.syncs,
  shein_observations.records, shein_observations.heads FROM store_center_runtime;
REVOKE USAGE ON SCHEMA shein_observations FROM store_center_runtime;
COMMIT;
```

Leave every table and business row intact. Run the old strict disabled permission preflight and start the saved old images/manifest only after it passes. An explicit later re-enable restores the existing observation grants through the original schema owner; it must not call InstallObservations again or discard the completed manifest volume. Retained observation facts prevent entering provider-free preparation. Restore full execution only with a separately validated real official application and Store connection configuration.
