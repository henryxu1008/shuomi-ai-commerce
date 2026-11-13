param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]] $CommandArguments
)

$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
Push-Location -LiteralPath $repoRoot
try {
    & go run ./cmd/ai-workbench-retained-prepare @CommandArguments
    if ($LASTEXITCODE -ne 0) {
        throw "ai-workbench-retained-prepare failed (exit $LASTEXITCODE)"
    }
} finally {
    Pop-Location
}
