$ErrorActionPreference = 'Stop'

# Render only: do not contact a running service or create Compose resources.
$compose = Join-Path $PSScriptRoot '../../deployments/docker/account-compose/docker-compose.yml'
$envFile = Join-Path $PSScriptRoot '../../deployments/docker/account-compose/.env.example'
$project = 'task-processor-local-login-header-contract'
$previousProject = $env:COMPOSE_PROJECT_NAME

try {
    $env:COMPOSE_PROJECT_NAME = $project
    $rendered = & docker compose --project-name $project --env-file $envFile -f $compose config --format json 2>&1
    if ($LASTEXITCODE -ne 0) { throw 'Local Account Compose configuration did not render' }
    $model = ($rendered | Out-String) | ConvertFrom-Json
} finally {
    if ($null -eq $previousProject) {
        Remove-Item Env:COMPOSE_PROJECT_NAME -ErrorAction SilentlyContinue
    } else {
        $env:COMPOSE_PROJECT_NAME = $previousProject
    }
}

$login = $model.services.'zitadel-login'
$entrypoint = @($login.entrypoint)
if ($entrypoint -notcontains '--max-http-header-size=32768') {
    throw 'Local Login rebuild must retain the 32 KiB HTTP header budget'
}
if ($entrypoint.Count -ne 4 -or
    $entrypoint[0] -cne '/app/entrypoint.sh' -or
    $entrypoint[1] -cne 'node' -or
    $entrypoint[2] -cne '--max-http-header-size=32768' -or
    $entrypoint[3] -cne 'apps/login/server.js') {
    throw 'Local Login must retain the image entrypoint wrapper and server arguments'
}
if ($login.environment.PSObject.Properties.Name -contains 'NODE_OPTIONS') {
    throw 'Local Login must inherit the image CA/preload NODE_OPTIONS without replacing them'
}

Write-Output 'PASS: local Login rebuild retains 32 KiB and the image startup options (configuration only)'
