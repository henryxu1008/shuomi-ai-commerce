# Enterprise capacity in a new unified installation

This composes the existing [capacity owner](../../../docs/architecture/enterprise-storage-capacity.md)
and [MinIO admission](../../../docs/architecture/minio-capacity-admission.md).
It is for a **new empty** project using base + Knowledge + Data Services + native
modules + non-payment Ecoservices. Runtime still owns empty-prefix admission,
namespace registration, accounting and the existing 5 GiB entitlement. No new
schema, price, wallet funding or storage purchase is added by this layer.

From this directory, prepare the ordinary private `.env` and exact-origin plugin
as in [modules.md](modules.md). The recommended POSIX entrypoint includes capacity:

First prepare the backend/UI with the repository's
[selected-Git image build entry](../../../docs/operations/application-image-build.md).
Build bootstrap/schema and MinIO client/server from that same clean source
archive using the standard targets, and retain their source/image records.
The deployment's private `docker-compose.images.yml` selects these prepared
images without changing service commands, entrypoints or mounts. Preserve it on
restart; image building and deployment remain separate operations.

Map every selected build-only service in that same image-only overlay:

| Standard Dockerfile target / image | Selected Compose services |
| --- | --- |
| Standard selected-Git Backend image (`data-services-runtime`) / UI image | `current-application` / `listingkit-ui` |
| `Dockerfile.account-compose`, `bootstrap` | `bootstrap` |
| `Dockerfile.account-compose`, `schema-init` | `schema-init`, `native-modules-init`, `ecoservices-init`, `storage-capacity-init`; selected `chat-init`, `acquisition-plans-init`, `store-observations-init` |
| `Dockerfile.knowledge-storage`, `server` | `knowledge-objects`, `qualification-objects`, `ecoservices-objects` |
| `Dockerfile.knowledge-storage`, `client` | `knowledge-storage-init`, `qualification-storage-init`, `ecoservices-storage-init` |
| `deployments/docker/Dockerfile.browser-collector`, `runtime` (same selected-Git clean source; only with acquisition plans or Store observations) | `browser-collector` |

Leave unselected services out of the image overlay; an image entry itself adds
that service to Compose. Keep source bind paths from the same selected release.

```sh
sh ./compose-modules.sh -- -f docker-compose.images.yml up -d --no-build --wait
# Optional model-free conversations; requires CHAT_ALLOWED_ORGANIZATION_IDS.
sh ./compose-modules.sh --chat -- -f docker-compose.images.yml up -d --no-build --wait
# Add --public for the fixed HTTPS origins described in public-trial.md.
sh ./compose-modules.sh --chat --public -- -f docker-compose.images.yml up -d --no-build --wait
```

Use the **same options** for `config --quiet`, `ps`, `stop`, and
`up -d --no-build --wait`. Image-only release overrides must retain the command,
entrypoint, environment and mounts selected by these layers. Never use `down -v`.
The entrypoint does not create `.env`, run acceptance fixtures, perform provider
calls, publish images, or authorize deployment by itself.

For direct Docker Compose use (including PowerShell), append
`-f docker-compose.storage-capacity.yml` after the Ecoservices layer. If selecting
Chat, append its original layer and then `docker-compose.storage-capacity-chat.yml`.
These companion layers select the final source manifest and dependency, so a later
module cannot silently replace the metered configuration. Explicit
`--acquisition-plans` and `--store-observations` options likewise compose their
original layers plus capacity companions; the latter implies acquisition plans
and Chat. They remain opt-in and retain their original installation authority,
provider restrictions and preparation-only behavior. Do not append an unhandled
manifest-producing layer after these files.

## Installation and retention

1. Bootstrap records the capacity choice before publishing its complete marker.
   Adding this layer to an existing installation, or dropping it from a retained
   capacity installation, fails before provisioning. This entrypoint is not a
   retained-instance enablement or migration command.
2. The existing storage initializers attach the repository's existing capacity
   read policy to their original narrow runtime users. Their trusted administrative
   `mc admin info` reads the canonical DeploymentID; the same runtime user must
   successfully read that ID and list the exact metered prefix/versions. Only
   the non-secret identity file is published to each dedicated private identity
   volume. Subsequent installation runs reject an identity change.
3. `storage-capacity-init` has no network and no SQL or storage administrator
   secrets. It reads the selected original private manifest and identity files,
   sets `storageCapacity:true` and the two existing MinIO identity fields, and
   publishes a private 0600 final manifest. Other modules and credentials are
   preserved. Identical reruns keep the file; changed inputs fail for inspection.
4. The backend mounts the final manifest read-only. Its existing SDK identity,
   IAM, schema and empty-prefix checks must pass before it serves. Existing
   commercial schema installation remains in the original schema owner.

Retain the identity and final-runtime volumes with every original volume. Once
capacity facts exist, keep `true`, the original identity pins and Commercial
database. Do not clear namespaces, swap databases/buckets or restore an unmetered
manifest. Interrupted configuration can repeat from the same unchanged inputs;
errors do not authorize resetting installation state or data. Existing retained
environments follow the separately authorized
[capacity runbook](../../../docs/engineering/enterprise-storage-capacity-runbook.md).

## Other-module wiring check

The unified deployment was checked through installer -> private manifest ->
Compose dependency/mount -> current runtime consumer. This is a source/config
check, not business or provider acceptance.

| Module | Deployment result / remaining configuration |
| --- | --- |
| Identity, membership, roles, referrals, Commercial/Money, account audit | Base installer and original restricted pools are wired; normal native login/grants remain required. Online wallet payment and external invitation mail still need their actual provider setup. Mailpit is private capture only. |
| Knowledge | Original schema, Tika and private objects are wired; capacity identity/policy/flag were missing and are now composed. No official Knowledge catalog is fabricated. |
| Collections, acquisition and Data Services | Original shared Product installer, bounded roles, Temporal and matched browser remain wired. Optional scheduled acquisition has its own explicit layer and final-manifest companion. No acquisition request runs during installation. |
| Projects, Report reads, tools, notifications, Agent configuration/customization, Cockpit | Native installers/manifest fields and current consumers are wired. Tool packaging still requires the exact deployment receiver. Saving a new title report needs the existing same-instance Product Review configuration; this composition does not invent it. |
| Ecoservices | Non-payment owner and private object dependency are wired; its missing capacity identity/read policy is now included. Merchant/payment/settlement execution remains unconfigured. Qualification evidence is excluded from capacity under its original contract. |
| Chat | Optional conversation-only installer is wired. Its replacement manifest previously could bypass a final capacity overlay; the companion explicitly feeds Chat output into capacity assembly. Model execution still needs admitted credentials, scope, price and budget. |
| Store observations | Separate opt-in preparation layer, now with an explicit entrypoint and capacity companion. It reports provider unconfigured and admits no real product/order synchronization. |
| Source-media/product images and full image Agent | No qualified public storage/Asset/model/worker configuration is supplied by this unified profile. The separate local-only source/image trials cannot be reused as public deployment. Not enabled by the capacity flag. |
| Store sync, fulfillment, Supply Chain, POD, SMS | Require their existing official applications, channel/merchant credentials, Asset/Temporal or SMS setup. No automatic enabling, fabricated credentials or external calls. |

After normal login, check `/workbench/account/organization/resources` with the
original CommercialRead permission: total 5 GiB, real used/reserved/available
values. Check selected module capabilities separately; a healthy backend alone
does not establish that every business path or external service is usable.
