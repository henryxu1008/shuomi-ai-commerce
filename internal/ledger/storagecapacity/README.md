# Enterprise storage capacity

Canonical owner of retained physical bytes, finite write reservations and
wallet-funded capacity lots. Product/Asset and Knowledge retain ownership of
content, references, lifecycle and permissions. This package contains contracts
only; the PostgreSQL adapter lives in `internal/integration/persistence/storagecapacity`
on the existing commercial owner database.

Design basis: [Enterprise Storage Capacity V1](../../../docs/architecture/enterprise-storage-capacity-v1.md).

New writes reserve atomically before object I/O or paid dispatch. Unknown results
retain their original reservation. Object confirmation counts a physical key
once. Capacity expiry blocks new occupancy and never deletes retained data.
Storage purchases activate only after Money commit; activation and commercial
order completion share one commercial database transaction.
