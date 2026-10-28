# Build ListingKit images without inheriting old releases

Run from a checkout containing the build entry below. Select an explicit local Git
commit/ref and a fresh receipt directory:

```powershell
pwsh -NoProfile -File scripts/build-listingkit-images.ps1 `
  -SourceRef <approved-commit> -Component All -OutputDirectory <receipt-directory>
```

`Backend` or `UI` can be selected independently. The command builds local
`local/current-application:git-<full-commit>` and
`local/listingkit-ui:git-<full-commit>` images for linux/amd64. It never starts
containers, reads runtime manifests, pushes images, applies Compose, initializes
schemas, prunes caches or deletes data. Existing release/deploy retirement gates
remain in force. Build success is not rollout or product acceptance.

The source context is a Git archive of the selected commit, including that
commit's standard Dockerfiles. Dirty files, untracked private manifests,
node_modules and local build directories are excluded unless tracked in Git.
Changes to Dockerfiles must therefore be committed before selecting them. The
receipt records commit, tree, image IDs, sizes and layer counts; it is written
only after all selected images build and pass the application-layer checks.
Existing receipts are never silently reused or overwritten. A failed validation
may leave an image tag for diagnosis, but produces no success receipt.

## Runtime construction

The canonical backend target is `data-services-runtime` in
`deployments/docker/Dockerfile.account-compose`. Its browser base contains Ubuntu,
system libraries and the driver/Chromium matching go.mod's Playwright SDK.
The installer is compiled separately from application source, so application-only
changes reuse the browser dependency layers. The serving target copies one
stripped current-application binary to `/usr/local/bin/current-application`.
Schema/acceptance executables are built only by their separate targets.

For an explicitly approved historical commit whose standard Dockerfile has the
`runtime` serving target, select it with `-BackendTarget runtime`. For example:

```powershell
pwsh -NoProfile -File scripts/build-listingkit-images.ps1 `
  -SourceRef 8478800e31bf2e126bd03fae5ec45536056d823e -Component All `
  -BackendTarget runtime -OutputDirectory <fresh-receipt-directory>
```

The default remains `data-services-runtime`; a missing selected target fails the
build without auto-detection or fallback. A non-default backend target adds
`-runtime` to the backend image tag and receipt filename, keeping it separate
from the default target's artifacts. The backend image label
`io.shuomi.build.backend-target` and receipt image entry's `target` record the
selection. UI tags and builds are unchanged. This option only builds the selected
commit's serving image; it does not authorize deployment or adapt application,
database, authentication or session contracts.

The UI uses the existing `Dockerfile.listingkit-ui`: a clean Node runtime and
only Next standalone/static/public output. Both paths reuse Docker's dependency
cache; compiler, module and package caches stay outside serving image layers.
The receipt identifies the actual images; upstream base tags can change, so it
does not claim bit-for-bit reproducibility across base-image updates.

Do not create release Dockerfiles with `FROM local/current-application:<old>` or
`FROM local/listingkit-ui:<old>`. Do not COPY the backend to two executable paths.
Removing old files in a later RUN does not remove the bytes from lower layers.
Build history checks reject repeated application COPY layers. The one-time
scratch/merged-filesystem rebuild used for disk recovery is not this source build
entry and must not become a chain of patch images.

## Retained instances

After separately authorized rollout, update only the intended service image in
the final retained image overlay. Preserve project/environment, private mounts,
network namespace, commands and named volumes. Never run `down -v`, old
initializers or generic prune as part of image building. Keep build and deploy
as separate operations; this change does not reopen production release.
