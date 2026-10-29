param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GoTestArgs
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

$fastTestPackages = @(
    "./cmd/current-application"
    "./internal/app/httpapi"
    "./internal/crawler/alibaba1688"
    "./internal/listingadmin"
    "./internal/integration/orgresource"
)

$boundaryTestPackages = @("./tests/...")

Write-Host "Running fast Go test suite..." -ForegroundColor Cyan
& go test -v @GoTestArgs @fastTestPackages
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

& go test -v @GoTestArgs @boundaryTestPackages
exit $LASTEXITCODE
