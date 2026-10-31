$scriptPath = Join-Path $PSScriptRoot "start-listingkit-local-ui.ps1"

function Get-StartUiScriptAst {
    $tokens = $null
    $errors = $null
    $ast = [System.Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$tokens, [ref]$errors)
    if ($errors.Count -gt 0) {
        throw "Unable to parse ${scriptPath}: $($errors[0].Message)"
    }
    return $ast
}

function Import-StartUiScriptFunctions {
    $ast = Get-StartUiScriptAst
    $functionNames = @(
        "Import-DotEnvFile",
        "Set-EnvIfMissing",
        "Initialize-UiLaunchEnvironment",
        "Assert-LoopbackApiBase",
        "Set-UiApiBases",
        "Get-ListeningProcessIds",
        "Get-ListeningConnections",
        "Assert-PortAvailable",
        "Assert-LoopbackListener",
        "Wait-ForUiReady",
        "Stop-VerifiedUiProcessTree",
        "Resolve-IsolatedRuntimeDirectory"
    )
    $functions = $ast.FindAll({
        param($node)
        $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and
            $functionNames -contains $node.Name
    }, $true)
    foreach ($function in $functions) {
        $definition = $function.Extent.Text -replace "^function\s+$([regex]::Escape($function.Name))", "function global:$($function.Name)"
        Invoke-Expression $definition
    }
}

Describe "start-listingkit-local-ui authentication" {
    BeforeEach {
        Import-StartUiScriptFunctions
    }

    It "does not expose a local auth gate bypass" {
        $ast = Get-StartUiScriptAst
        $paramBlock = $ast.ParamBlock
        $parameter = $paramBlock.Parameters |
            Where-Object { $_.Name.VariablePath.UserPath -eq "BypassAuthGate" } |
            Select-Object -First 1

        $parameter | Should Be $null
    }

    It "keeps hostile repository auth env values out of isolated acceptance" {
        $repoRoot = Join-Path $TestDrive "repo"
        New-Item -ItemType Directory -Path $repoRoot -Force | Out-Null
        Set-Content -LiteralPath (Join-Path $repoRoot ".env") -Value @(
            "ZITADEL_ISSUER_URL=https://production.example.invalid",
            "ZITADEL_CLIENT_ID=production-client"
        )
        $previousKubeconfig = $env:KUBECONFIG
        $previousIssuer = $env:ZITADEL_ISSUER_URL
        $previousClientID = $env:ZITADEL_CLIENT_ID
        try {
            $env:KUBECONFIG = "C:\hostile\kubeconfig"
            $env:ZITADEL_ISSUER_URL = "http://localhost:8080"
            $env:ZITADEL_CLIENT_ID = "local-client"

            Initialize-UiLaunchEnvironment -RepoRoot $repoRoot -IsolatedAcceptance

            [string]$env:KUBECONFIG | Should Be ""
            $env:ZITADEL_ISSUER_URL | Should Be "http://localhost:8080"
            $env:ZITADEL_CLIENT_ID | Should Be "local-client"
        } finally {
            $env:KUBECONFIG = $previousKubeconfig
            $env:ZITADEL_ISSUER_URL = $previousIssuer
            $env:ZITADEL_CLIENT_ID = $previousClientID
        }
    }

    It "overrides inherited API bases in isolated acceptance" {
        $names = @(
            "LISTINGKIT_SERVICE_API_BASE"
        )
        $previous = @{}
        try {
            foreach ($name in $names) {
                $previous[$name] = [Environment]::GetEnvironmentVariable($name)
                [Environment]::SetEnvironmentVariable($name, "https://deployed.example.invalid/$($name.ToLowerInvariant())")
            }

            Set-UiApiBases `
                -ServiceApiBase "http://127.0.0.1:18085/api/v1" `
                -IsolatedAcceptance

            $env:LISTINGKIT_SERVICE_API_BASE | Should Be "http://127.0.0.1:18085/api/v1"
        } finally {
            foreach ($name in $names) {
                [Environment]::SetEnvironmentVariable($name, $previous[$name])
            }
        }
    }

    It "rejects non-loopback API bases in isolated acceptance" {
        $message = ""
        try {
            Set-UiApiBases `
                -ServiceApiBase "https://deployed.example.invalid/api/v1" `
                -IsolatedAcceptance
        } catch {
            $message = $_.Exception.Message
        }
        $message | Should Match "loopback"
    }

    It "preserves normal explicit auth inputs without reading deployed secrets" {
        $repoRoot = Join-Path $TestDrive "normal-repo"
        New-Item -ItemType Directory -Path $repoRoot -Force | Out-Null
        Set-Content -LiteralPath (Join-Path $repoRoot ".env") -Value @(
            "AUTH_SECRET=synthetic-dotenv-auth",
            "ZITADEL_CLIENT_SECRET=synthetic-dotenv-client",
            "ZITADEL_CLIENT_ID=synthetic-dotenv-id",
            "LISTINGKIT_SERVICE_API_BASE=http://127.0.0.1:18085/api/v1"
        )
        $names = @("AUTH_SECRET", "ZITADEL_CLIENT_SECRET", "ZITADEL_CLIENT_ID", "LISTINGKIT_SERVICE_API_BASE", "KUBECONFIG")
        $previous = @{}
        $script:clusterReads = 0
        function global:kubectl { $script:clusterReads++; throw "Synthetic test must not read a cluster" }
        try {
            foreach ($name in $names) { $previous[$name] = [Environment]::GetEnvironmentVariable($name) }
            $env:AUTH_SECRET = "synthetic-explicit-auth"
            $env:ZITADEL_CLIENT_SECRET = "synthetic-explicit-client"
            $env:ZITADEL_CLIENT_ID = ""
            $env:LISTINGKIT_SERVICE_API_BASE = "http://127.0.0.1:28085/api/v1"
            $env:KUBECONFIG = "synthetic-no-such-kubeconfig"

            Initialize-UiLaunchEnvironment -RepoRoot $repoRoot
            Set-UiApiBases -ServiceApiBase "http://127.0.0.1:38085/api/v1"

            $script:clusterReads | Should Be 0
            $env:AUTH_SECRET | Should Be "synthetic-explicit-auth"
            $env:ZITADEL_CLIENT_SECRET | Should Be "synthetic-explicit-client"
            $env:ZITADEL_CLIENT_ID | Should Be "synthetic-dotenv-id"
            $env:LISTINGKIT_SERVICE_API_BASE | Should Be "http://127.0.0.1:28085/api/v1"
        } finally {
            foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $previous[$name]) }
            Remove-Item Function:\global:kubectl -ErrorAction SilentlyContinue
        }
    }

    It "has no retired Task API or cluster-secret entry" {
        $ast = Get-StartUiScriptAst
        @($ast.ParamBlock.Parameters | Where-Object { $_.Name.VariablePath.UserPath -eq "ApiBase" }).Count | Should Be 0
        $content = Get-Content -LiteralPath $scriptPath -Raw
        foreach ($retired in @('LISTINGKIT_API_BASE', 'NEXT_PUBLIC_LISTINGKIT_API_BASE', 'Import-DeployedListingKitAuthSecrets', 'kubectl')) {
            $content | Should Not Match ([regex]::Escape($retired))
        }
    }

    It "rejects an empty Service API base in isolated acceptance" {
        $message = ""
        try {
            Set-UiApiBases -ServiceApiBase "" -IsolatedAcceptance
        } catch {
            $message = $_.Exception.Message
        }
        $message | Should Match "explicit local API base"
    }

    It "refuses an occupied port in isolated acceptance" {
        function global:Get-ListeningProcessIds { @(4343) }
        $message = ""
        try {
            Assert-PortAvailable -ListenPort 3000
        } catch {
            $message = $_.Exception.Message
        }
        $message | Should Match "4343"
    }

    It "binds Next directly to loopback and cleans its process tree on failure" {
        $content = Get-Content -LiteralPath $scriptPath -Raw
        $content | Should Match '\$nextArguments \+= @\("-H", "127\.0\.0\.1"\)'
        $content | Should Match '-FilePath \$nodeExecutable'
        $content | Should Match 'Stop-VerifiedUiProcessTree -RootProcess \$process'
        $content | Should Match '\.IndexOf\(\$ExpectedCommandContains'
        $content | Should Not Match '\.Contains\(\$ExpectedCommandContains,\s*\[System\.StringComparison\]'
        $content | Should Not Match '-FilePath "powershell"'
    }

    It "cleans a verified listener with Windows PowerShell compatible matching" {
        $script:stoppedProcessID = $null
        function global:Get-ListeningProcessIds { @(4567) }
        function global:Get-CimInstance { [pscustomobject]@{ CommandLine = "NODE.EXE next\\dist\\bin\\next" } }
        function global:Stop-Process { param([int]$Id) $script:stoppedProcessID = $Id }
        try {
            Stop-VerifiedUiProcessTree -RootProcess $null -ListenPort 3000 -ExpectedCommandContains "node.exe NEXT"
            $script:stoppedProcessID | Should Be 4567
        } finally {
            Remove-Item Function:\global:Get-ListeningProcessIds -ErrorAction SilentlyContinue
            Remove-Item Function:\global:Get-CimInstance -ErrorAction SilentlyContinue
            Remove-Item Function:\global:Stop-Process -ErrorAction SilentlyContinue
        }
    }

    It "waits for the real UI HTTP endpoint" {
        $script:requestCount = 0
        function global:Get-Process { [pscustomobject]@{ Id = 1234 } }
        function global:Invoke-WebRequest {
            $script:requestCount++
            return [pscustomobject]@{ StatusCode = 200 }
        }
        try {
            Wait-ForUiReady -RootUrl "http://127.0.0.1:3000" -ProcessId 1234 -TimeoutSeconds 1
            $script:requestCount | Should Be 1
        } finally {
            Remove-Item Function:\global:Get-Process -ErrorAction SilentlyContinue
            Remove-Item Function:\global:Invoke-WebRequest -ErrorAction SilentlyContinue
        }
    }

    It "rejects a wildcard UI listener" {
        function global:Get-ListeningConnections { @([pscustomobject]@{ LocalAddress = "::"; OwningProcess = 1234 }) }
        $message = ""
        try {
            Assert-LoopbackListener -ListenPort 3000
        } catch {
            $message = $_.Exception.Message
        }
        $message | Should Match "not bound exclusively to loopback"
    }
}
