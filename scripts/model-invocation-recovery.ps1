param(
    [Parameter(Mandatory = $true)][string]$Config,
    [ValidateSet('inspect', 'resolve')][string]$Action = 'inspect'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Push-Location (Join-Path $PSScriptRoot '..')
try {
    & go run ./cmd/model-invocation-recovery -config $Config -action $Action
    if ($LASTEXITCODE -ne 0) {
        throw "model-invocation-recovery failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}
