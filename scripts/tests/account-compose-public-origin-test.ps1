$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$script = @'
set -eu
. /src/deployments/docker/account-compose/deployment-origins.sh
unset ACCOUNT_PUBLIC_TRIAL ACCOUNT_PUBLIC_APPLICATION_ORIGIN ACCOUNT_PUBLIC_IDENTITY_ORIGIN
account_resolve_origins
test -z "$account_public_app"
expect_rejected() {
 if (account_resolve_origins) >/dev/null 2>&1; then echo 'invalid public origins accepted' >&2; exit 1; fi
}
ACCOUNT_PUBLIC_APPLICATION_ORIGIN=https://trial.example.com
expect_rejected
ACCOUNT_PUBLIC_TRIAL=1
expect_rejected
ACCOUNT_PUBLIC_IDENTITY_ORIGIN=https://auth-trial.example.com
account_resolve_origins
test "$account_public_app" = https://trial.example.com
test "$account_identity_host" = auth-trial.example.com
state=$(mktemp -d)
account_pin_origins "$state" fresh
account_pin_origins "$state" retained
ACCOUNT_PUBLIC_IDENTITY_HOST=wrong.example.com
expect_rejected
unset ACCOUNT_PUBLIC_IDENTITY_HOST
ACCOUNT_PUBLIC_APPLICATION_ORIGIN=https://another.example.com
if account_pin_origins "$state" retained >/dev/null 2>&1; then echo 'retained origin drift accepted' >&2; exit 1; fi
ACCOUNT_PUBLIC_APPLICATION_ORIGIN=https://trial.example.com
for bad in http://auth.example.com https://localhost https://127.0.0.1 https://auth.example.com/path https://user@auth.example.com https://auth.example.com:8443 'https://auth.example.com?x=1' https://auth..example.com; do
 ACCOUNT_PUBLIC_IDENTITY_ORIGIN="$bad"
 expect_rejected
done
ACCOUNT_PUBLIC_IDENTITY_ORIGIN=$ACCOUNT_PUBLIC_APPLICATION_ORIGIN
expect_rejected
unset ACCOUNT_PUBLIC_TRIAL ACCOUNT_PUBLIC_APPLICATION_ORIGIN ACCOUNT_PUBLIC_IDENTITY_ORIGIN
account_resolve_origins
if account_pin_origins "$state" retained >/dev/null 2>&1; then echo 'public project silently changed to local' >&2; exit 1; fi
echo 'PASS: explicit HTTPS origins and retained origin pinning'
'@
$script.Replace("`r`n", "`n") | & docker run --rm -i --mount "type=bind,src=$root,dst=/src,readonly" alpine:3.22 sh
if ($LASTEXITCODE -ne 0) { throw 'Public origin contract failed' }

# Render without creating or inspecting any running Compose project.
$directory = Join-Path $root 'deployments/docker/account-compose'
$settings = @{
    COMPOSE_PROJECT_NAME = 'public-origin-contract'
    ACCOUNT_TOOL_RELEASE_DIRECTORY = $root
    ACCOUNT_PUBLIC_APPLICATION_ORIGIN = 'https://trial.example.com'
    ACCOUNT_PUBLIC_IDENTITY_ORIGIN = 'https://auth-trial.example.com'
    ACCOUNT_PUBLIC_APPLICATION_HOST = 'trial.example.com'
    ACCOUNT_PUBLIC_IDENTITY_HOST = 'auth-trial.example.com'
}
$saved = @{}
try {
    foreach ($key in $settings.Keys) {
        $saved[$key] = [Environment]::GetEnvironmentVariable($key)
        [Environment]::SetEnvironmentVariable($key, $settings[$key])
    }
    $arguments = @('compose', '--env-file', (Join-Path $directory '.env.example'))
    foreach ($file in @('docker-compose.yml','docker-compose.knowledge.yml','docker-compose.data-services.yml','docker-compose.modules.yml','docker-compose.public-trial.yml')) {
        $arguments += @('-f', (Join-Path $directory $file))
    }
    $rendered = & docker @arguments config --format json
    if ($LASTEXITCODE -ne 0) { throw 'Public Compose configuration did not render' }
    $model = ($rendered | Out-String) | ConvertFrom-Json
    if ($model.services.'listingkit-ui'.environment.AUTH_URL -cne $settings.ACCOUNT_PUBLIC_APPLICATION_ORIGIN -or
        $model.services.'zitadel-api'.environment.ZITADEL_EXTERNALDOMAIN -cne $settings.ACCOUNT_PUBLIC_IDENTITY_HOST -or
        $model.services.'zitadel-api'.environment.ZITADEL_EXTERNALPORT -cne '443') { throw 'Public issuer/application configuration mismatch' }
    foreach ($service in $model.services.PSObject.Properties.Value) {
        foreach ($port in $service.ports) {
            if ($port.host_ip -cne '127.0.0.1') { throw 'A private Compose service was exposed' }
        }
    }
    Write-Output 'PASS: public issuer wiring and loopback-only service ports (render only)'
} finally {
    foreach ($key in $saved.Keys) { [Environment]::SetEnvironmentVariable($key, $saved[$key]) }
}
