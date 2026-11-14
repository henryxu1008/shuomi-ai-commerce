#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$SourceRef,
    [Parameter(Mandatory = $true)][string]$OutputDirectory,
    [string]$ImageReceipt
)
$ErrorActionPreference = 'Stop'
$repository = Split-Path -Parent $PSScriptRoot
$revision = & git -C $repository rev-parse --verify --end-of-options "${SourceRef}^{commit}"
if ($LASTEXITCODE -ne 0 -or $revision -notmatch '^[0-9a-f]{40}$') { throw 'SourceRef must resolve to an exact local commit.' }
$destination = [IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $destination) { throw 'Choose a new package directory. Existing packages and instance data are not overwritten.' }
New-Item -ItemType Directory -Path $destination | Out-Null
if (-not $ImageReceipt) {
    & (Join-Path $PSScriptRoot 'build-listingkit-images.ps1') -SourceRef $revision -OutputDirectory (Join-Path $destination 'images') -Component All
    if ($LASTEXITCODE -ne 0) { throw 'Exact-source image build failed.' }
    $ImageReceipt = Join-Path $destination "images/build-$revision-All.json"
}
$receipt = Get-Content -LiteralPath $ImageReceipt -Raw | ConvertFrom-Json
if ($receipt.schemaVersion -ne 1 -or $receipt.sourceCommit -cne $revision -or $receipt.component -cne 'All') { throw 'Image receipt must contain both images from the exact selected source.' }
$backend = @($receipt.images | Where-Object { $_.component -ceq 'Backend' })
$ui = @($receipt.images | Where-Object { $_.component -ceq 'UI' })
if ($backend.Count -ne 1 -or $ui.Count -ne 1 -or $backend[0].target -cne 'data-services-runtime') { throw 'The prepared native module runtime and UI are required.' }
foreach ($item in @($backend[0], $ui[0])) {
    $inspection = @((& docker image inspect $item.imageId) | ConvertFrom-Json)[0]
    if ($LASTEXITCODE -ne 0 -or $inspection.Id -cne $item.imageId -or $inspection.Config.Labels.'org.opencontainers.image.revision' -cne $revision) { throw 'Prepared image identity does not match its source receipt.' }
}
$source = Join-Path $destination 'source'
New-Item -ItemType Directory -Path $source | Out-Null
$archive = Join-Path $destination 'source.tar'
& git -C $repository archive --format=tar --output=$archive $revision
if ($LASTEXITCODE -ne 0) { throw 'Source archive failed.' }
& tar -xf $archive -C $source
if ($LASTEXITCODE -ne 0) { throw 'Source extraction failed.' }
Remove-Item -LiteralPath $archive
$files = [ordered]@{}
Get-ChildItem -LiteralPath $source -Recurse -File | ForEach-Object {
    $relative = [IO.Path]::GetRelativePath($source, $_.FullName).Replace('\','/')
    $files[$relative] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
}
$frontend = Join-Path $source 'web/listingkit-ui'
Push-Location $frontend
try {
    & pnpm.cmd install --frozen-lockfile --offline
    if ($LASTEXITCODE -ne 0) { throw 'Prepare the existing pnpm dependencies before packaging.' }
    & pnpm.cmd build:setup
    if ($LASTEXITCODE -ne 0) { throw 'Setup UI build failed.' }
} finally { Pop-Location }
$builtUI = [IO.Path]::GetFullPath((Join-Path $frontend '.setup-dist'))
$uiDestination = [IO.Path]::GetFullPath((Join-Path $destination 'ui'))
if ([IO.Path]::GetDirectoryName($builtUI) -cne [IO.Path]::GetFullPath($frontend) -or
    [IO.Path]::GetDirectoryName($uiDestination) -cne $destination) { throw 'Unsafe package UI move target.' }
Move-Item -LiteralPath $builtUI -Destination $uiDestination
Push-Location $source
try {
    $binary = if ($IsWindows) { 'listingkit-setup.exe' } else { 'listingkit-setup' }
    & go build -trimpath -o (Join-Path $destination $binary) ./cmd/listingkit-setup
    if ($LASTEXITCODE -ne 0) { throw 'Local setup command build failed.' }
} finally { Pop-Location }
$release = [ordered]@{schemaVersion=1;sourceCommit=$revision;backendImage=$backend[0].imageId;uiImage=$ui[0].imageId;files=$files}
[IO.File]::WriteAllText((Join-Path $destination 'release.json'), ($release | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
Write-Output "Prepared local setup package: $destination"
Write-Output "Start: $(Join-Path $destination $binary)"
Write-Output 'This builds a package only. No retained instance is changed; no project is initialized.'
