#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$SourceRef,
    [Parameter(Mandatory = $true)][string]$OutputDirectory,
    [ValidateSet('Backend', 'UI', 'All')][string]$Component = 'All'
)
$ErrorActionPreference = 'Stop'
$repository = Split-Path -Parent $PSScriptRoot
$revision = & git -C $repository rev-parse --verify --end-of-options "${SourceRef}^{commit}"
if ($LASTEXITCODE -ne 0 -or $revision -notmatch '^[0-9a-f]{40}$') { throw 'SourceRef must resolve to a local Git commit.' }
$tree = & git -C $repository rev-parse "${revision}^{tree}"
if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve source tree.' }
$destination = [IO.Path]::GetFullPath($OutputDirectory)
$receiptPath = Join-Path $destination "build-$revision-$Component.json"
if (Test-Path -LiteralPath $receiptPath) { throw 'Build receipt already exists; choose a fresh output directory.' }
$tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$staging = Join-Path $tempParent ('listingkit-build-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $staging | Out-Null
$artifacts = @()
try {
    $archive = Join-Path $staging 'source.tar'
    $context = Join-Path $staging 'source'
    New-Item -ItemType Directory -Path $context | Out-Null
    & git -C $repository archive --format=tar --output=$archive $revision
    if ($LASTEXITCODE -ne 0) { throw 'Source archive failed.' }
    & tar -xf $archive -C $context
    if ($LASTEXITCODE -ne 0) { throw 'Source extraction failed.' }
    $components = if ($Component -eq 'All') { @('Backend', 'UI') } else { @($Component) }
    foreach ($item in $components) {
        $backend = $item -eq 'Backend'
        $image = if ($backend) { "local/current-application:git-$revision" } else { "local/listingkit-ui:git-$revision" }
        $dockerfile = if ($backend) { 'deployments/docker/Dockerfile.account-compose' } else { 'deployments/docker/Dockerfile.listingkit-ui' }
        $buildArguments = @('build', '--platform', 'linux/amd64', '--file', (Join-Path $context $dockerfile),
            '--tag', $image, '--label', "org.opencontainers.image.revision=$revision",
            '--label', "org.opencontainers.image.tree=$tree")
        if ($backend) { $buildArguments += @('--target', 'data-services-runtime') }
        & docker @buildArguments $context | Out-Host
        if ($LASTEXITCODE -ne 0) { throw "$item image build failed." }
        $rawImage = & docker image inspect $image
        if ($LASTEXITCODE -ne 0) { throw "$item image inspection failed." }
        $inspection = @($rawImage | ConvertFrom-Json)[0]
        if ($inspection.Config.Labels.'org.opencontainers.image.revision' -ne $revision -or
            $inspection.Config.Labels.'org.opencontainers.image.tree' -ne $tree) { throw 'Image source labels do not match.' }
        $rawHistory = & docker history --no-trunc --format '{{json .}}' $inspection.Id
        if ($LASTEXITCODE -ne 0) { throw "$item image history inspection failed." }
        $history = @($rawHistory | ForEach-Object { $_ | ConvertFrom-Json })
        $pattern = if ($backend) { '^COPY .* /usr/local/bin/current-application(?:\s|$)' } else { '^COPY .*\.next/standalone(?:\s|/)' }
        if (@($history | Where-Object { $_.CreatedBy -match $pattern }).Count -ne 1) {
            throw "$item image must contain exactly one application COPY; do not inherit a previous release."
        }
        if ($backend -and @($history | Where-Object { $_.CreatedBy -match '^COPY .* /app/current-application(?:\s|$)' }).Count) {
            throw 'Backend must use only /usr/local/bin/current-application.'
        }
        $artifacts += [ordered]@{component=$item;tag=$image;imageId=$inspection.Id;sizeBytes=$inspection.Size;
            filesystemLayers=@($inspection.RootFS.Layers).Count;dockerfile=$dockerfile}
    }
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
    $receipt = [ordered]@{schemaVersion=1;sourceCommit=$revision;sourceTree=$tree;platform='linux/amd64';
        component=$Component;images=$artifacts;builtUtc=[datetime]::UtcNow.ToString('o')}
    [IO.File]::WriteAllText($receiptPath, ($receipt | ConvertTo-Json -Depth 6), [Text.UTF8Encoding]::new($false))
    Write-Output "Build receipt: $receiptPath"
} finally {
    # Only the randomly-created source staging directory is eligible for removal.
    $resolved = [IO.Path]::GetFullPath($staging)
    if ([IO.Path]::GetDirectoryName($resolved) -ne $tempParent.TrimEnd([IO.Path]::DirectorySeparatorChar) -or
        [IO.Path]::GetFileName($resolved) -notmatch '^listingkit-build-[0-9a-f]{32}$') { throw 'Unsafe staging cleanup path.' }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
