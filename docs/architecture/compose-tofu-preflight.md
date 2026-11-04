# Compose OpenTofu preflight boundary

## Product outcome and authority

Execution: #1090, authorized by the user on 2026-11-04 after the #1085
empty-install failure. A package or provider download failure before identity
creation must permit the same fresh project to be retried after connectivity
recovers. This is an empty-install change under the
[greenfield baseline](../product/greenfield-no-legacy-migration.md); no UI or
Figma change, retained-project repair, migration, deployment or cleanup is in scope.

Design Basis: Independent Architecture. Admission is recorded in the execution
Issue after the bounded independent review, before production script changes.

## Existing owner and path

The existing account/acquisition/referrals Compose `tofu-init` services invoke
their original POSIX scripts and official OpenTofu/ZITADEL provider. Terraform
resources, outputs, provider versions, credentials and Compose injection remain
unchanged. The [native initialization contract](enterprise-custom-roles.md#concrete-initialization-and-permission-mapping)
continues to own native roles, grants and completion.

Current Terraform roots use local state and have no remote backend or migration.
[OpenTofu init](https://opentofu.org/docs/cli/commands/init/) prepares the working
directory and downloads providers; it can be repeated. In these checked-in local
configurations it does not create the identity resources that `apply` owns.

## Preparation and durable boundary

Keep the original completed/incomplete checks at entry. For a fresh project,
prepare packages, private local configuration, the existing TLS-verified OIDC
discovery read and the **first root** `tofu init` before writing `.terraform-started`.
Then recheck both markers, refuse if another initializer changed them during
preparation, and write/chmod the existing started marker immediately before the
first `tofu apply`. No new marker, retry owner, schema or shared helper is needed.

| Existing durable state | Action |
| --- | --- |
| complete | Existing exit/backfill path; never reapply |
| started without complete | Existing fail-closed refusal before any preparation |
| neither marker; preparation fails or is interrupted | No apply; leave local preparation files, permit the same command to retry |
| either marker appears during preparation | Refuse before apply; do not modify the marker |
| started written; apply/output/later probe fails or is interrupted | Keep started/state/outputs; refuse replay, even when the external result is unknown |
| all original work succeeds | Original started-to-complete rename |

Referrals performs a main apply before its probe init/apply/destroy. The probe
stays **inside** the started boundary; its download failure cannot be retried as
if no identity mutation happened. Never remove a started marker on any error.
The original single-initializer Compose ownership and its narrow marker-write
sequence remain; this does not introduce a multi-writer installation protocol.

## Security, persistence and external effects

Only existing private volumes receive local preparation writes; serving gains no
credentials or access. Keep umask, mode 600 markers/outputs, TLS CA/hostname checks,
fixed public identity validation and official login/authorization unchanged.
Preparation never receives a new authority to mutate cloud or identity resources.
The native state and original output materialization remain the sole facts after
apply. No cleanup, automatic rollback, fallback, second fact owner or retries of
unknown external effects are introduced. Existing failed projects are not repaired.

## Validation and delivery limits

Use the existing Go/POSIX temporary-directory and PATH-stub style to execute each
real script with remapped private paths. First demonstrate RED for failed package
and root-provider preparation retaining started; then verify recovery in the same
directory without resetting files. Verify marker appearance during preparation,
apply failure after a state write, output failure, complete replay, and referrals
probe-init failure after main apply. All apply stubs require started to exist.
Reuse the existing account completed-state/backfill tests. No real provider,
business credentials, retained volumes or external service is used.

One final independent diff/affected-path check and applicable exact-head CI follow
implementation. These checks do not grant merge, deployment, Issue closure or
product/production acceptance.
