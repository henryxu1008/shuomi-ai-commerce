$scriptPath = Join-Path $PSScriptRoot "build-push-deploy-listingkit-workbench.ps1"

Describe "retired ListingKit release entry" {
    It "contains only an unconditional literal rejection" {
        (Get-Content -LiteralPath $scriptPath -Raw).Trim() | Should -Be 'throw "Legacy ListingKit API/UI release and rollback are retired; current-application production release is NOT_READY."'
    }
    It "rejects repeated direct calls before operations" {
        { & $scriptPath } | Should -Throw "*release and rollback are retired*"
        { & $scriptPath } | Should -Throw "*release and rollback are retired*"
    }
    It "cannot use old build-only or nonproduction inputs to bypass retirement" {
        { & $scriptPath -SkipApply -SkipTests -Namespace listingkit-nonprod -Tag previous } | Should -Throw "*release and rollback are retired*"
    }
    It "rejects dot-sourcing without exporting old publishing functions" {
        { . $scriptPath } | Should -Throw "*release and rollback are retired*"
    }
}
