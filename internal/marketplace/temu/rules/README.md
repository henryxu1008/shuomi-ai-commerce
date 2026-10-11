# TEMU official specification facts

This is the pure TEMU rule owner admitted by
[the frozen #31 design](../../../../docs/architecture/temu-platform-v1.md).
It has no HTTP, credentials, database, workflow or mutable authority.

`NewOfficialCatalog` loads the checked-in official E23, E24 and E30 snapshots.
`Sources` exposes the official links, timestamps and hashes of the included
TSV rows; `Revision` binds all three source snapshots. The E30 snapshot includes
all 87 mapped rows of the 235-row document. The other 148 rows have no site ID;
unknown regions never receive a guessed site. Windows line-ending conversion
does not change rule identity.

`SemiSites` retains all 96 PA entries, including 26 closed sites. The regional
SEMI column does not replace this PA dictionary or alias its IDs.
`SemiSelection` checks documented AU/NZ associations. A Pan-European entry
keeps eligibility unresolved until current official merchant evidence supplies
its actual group and constraints; this package does not force all geographic
European sites or manufacture merchant permission.

`LocalTarget` joins the exact original region to site ID and the 39 official
specification rows. Deployment partition and the specification's historical
partition label remain separate: Belgium is EU in E30 and GLO in E24.
GE/AM stay visible with `local_specifications_missing`; SA/LU and other blank
opening fields remain `local_opening_unverified`. No result declares readiness
for upload, active authorization, or eligibility for every merchant.

`NumberRule.Allows` uses Go's standard `math/big` exact arithmetic and the
site's original range and precision. Raw languages and range text are retained,
including source inconsistencies. Missing or invalid quantities remain missing
or invalid; there is no price inferred from cost, float rounding, clamping,
default package size, exchange rate, or implicit unit conversion.

`CheckLocalNumbers` applies E17/E24 to the exact original partition and region.
It checks explicit package units, per-SKU currency/range/lexical precision,
retail strictly above base price with exact arithmetic, and supplied inventory
within the documented bounds. It never changes a fact. A zero inventory is
distinct from missing inventory. The returned numerical report has no overall
publish-ready verdict: current category, template, assets, consent and Store
authority still require the shared preparation and Submission path.

Current consumers are the remaining TargetRecord/rule preparation and typed
TEMU integration work in this same delivery batch. These interfaces are not yet
wired into the shared editor or a publishing path. The full #31 scope, current
merchant rule checks and GE/AM delivery gate remain required.
