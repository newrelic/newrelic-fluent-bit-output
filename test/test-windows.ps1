<#
.SYNOPSIS
    Windows functional/runtime test for the newrelic-fluent-bit-output Windows image.
    Analogous to test.sh: starts a mock New Relic endpoint (the real MockServer, run as a
    standalone Windows process since no Windows-container image exists for it), runs the
    built plugin image against it, and verifies the logs actually arrive.

.PARAMETER Image
    The newrelic-fluent-bit-output Windows Docker image tag to test.

.PARAMETER MockServerVersion
    MockServer release version to download (tag is "mockserver-<version>" on
    https://github.com/mock-server/mockserver-monorepo/releases).
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Image,

    [string]$MockServerVersion = "7.5.0"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$TestDir = $PSScriptRoot
$WorkDir = Join-Path $TestDir ".windows-test-work"
$TestDataDir = Join-Path $TestDir "testdata-windows"
$NetworkName = "wintest-net"
$ContainerName = "fb-windows-test"
$ComposeProject = "fbwintest"
$ComposeFile = Join-Path $PSScriptRoot "docker-compose.windows.yml"
$FirewallRuleName = "newrelic-fb-mockserver-test-1080"
$MockServerPort = 1080
$HealthPort = 2020
$DbFileName = "flb.db"

$mockServerProcess = $null
$mockServerOutLog = Join-Path $WorkDir "mockserver-out.log"
$mockServerErrLog = Join-Path $WorkDir "mockserver-err.log"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message"
}

function Test-MockServerReady {
    try {
        $resp = Invoke-WebRequest -Method PUT -Uri "http://localhost:$MockServerPort/mockserver/status" `
            -UseBasicParsing -TimeoutSec 5
        return $resp.StatusCode -eq 200
    }
    catch {
        return $false
    }
}

function Test-LogsDelivered {
    param([string]$VerifyBody)
    try {
        $resp = Invoke-WebRequest -Method PUT -Uri "http://localhost:$MockServerPort/mockserver/verify" `
            -Body $VerifyBody -ContentType "application/json" -UseBasicParsing -TimeoutSec 5
        return $resp.StatusCode -eq 202
    }
    catch {
        return $false
    }
}

function Test-ContainerRunning {
    param([string]$Name)
    try {
        $status = docker inspect -f "{{.State.Running}}" $Name 2>$null
        return ($LASTEXITCODE -eq 0) -and ($status.Trim() -eq "true")
    }
    catch {
        return $false
    }
}

function Test-FluentBitHealthy {
    param([int]$Port)
    try {
        $resp = Invoke-WebRequest -Method GET -Uri "http://localhost:$Port/api/v1/health" -UseBasicParsing -TimeoutSec 5
        return $resp.StatusCode -eq 200
    }
    catch {
        return $false
    }
}

function Wait-Until {
    param(
        [scriptblock]$Condition,
        [string]$Description,
        [int]$MaxRetries = 10,
        [int]$DelaySeconds = 2
    )
    for ($i = 0; $i -le $MaxRetries; $i++) {
        if (& $Condition) {
            return $true
        }
        Write-Host "Waiting for $Description. Trying again in ${DelaySeconds}s. Try #$i"
        Start-Sleep -Seconds $DelaySeconds
    }
    return $false
}

function Remove-PreviousRunLeftovers {
    # Env vars for docker-compose.windows.yml aren't set yet this early, so fall back to a plain
    # container removal rather than `docker compose down` (which would warn about unset variables).
    docker rm -f $ContainerName 2>$null | Out-Null
    docker network rm $NetworkName 2>$null | Out-Null
    if (Test-Path $TestDataDir) { Remove-Item -Recurse -Force $TestDataDir -ErrorAction SilentlyContinue }
    if (Test-Path $WorkDir) { Remove-Item -Recurse -Force $WorkDir -ErrorAction SilentlyContinue }
    Remove-NetFirewallRule -DisplayName $FirewallRuleName -ErrorAction SilentlyContinue
}

function Invoke-Cleanup {
    Write-Step "Cleaning up"

    docker compose -f $ComposeFile -p $ComposeProject down --remove-orphans 2>$null | Out-Null
    docker rm -f $ContainerName 2>$null | Out-Null
    docker network rm $NetworkName 2>$null | Out-Null

    if ($mockServerProcess -and -not $mockServerProcess.HasExited) {
        Stop-Process -Id $mockServerProcess.Id -Force -ErrorAction SilentlyContinue
    }

    Remove-NetFirewallRule -DisplayName $FirewallRuleName -ErrorAction SilentlyContinue

    if (Test-Path $TestDataDir) { Remove-Item -Recurse -Force $TestDataDir -ErrorAction SilentlyContinue }
    if (Test-Path $WorkDir) { Remove-Item -Recurse -Force $WorkDir -ErrorAction SilentlyContinue }
}

try {
    Remove-PreviousRunLeftovers

    New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null

    Write-Step "Downloading MockServer $MockServerVersion (Windows bundle)"
    $zipUrl = "https://github.com/mock-server/mockserver-monorepo/releases/download/mockserver-$MockServerVersion/mockserver-$MockServerVersion-windows-x86_64.zip"
    $zipPath = Join-Path $WorkDir "mockserver.zip"
    Invoke-WebRequest -Uri $zipUrl -OutFile $zipPath

    Write-Step "Extracting MockServer bundle"
    Expand-Archive -Path $zipPath -DestinationPath $WorkDir -Force
    $mockServerDir = Get-ChildItem -Path $WorkDir -Directory | Where-Object { $_.Name -like "mockserver-*" } | Select-Object -First 1
    if (-not $mockServerDir) {
        throw "Could not find extracted MockServer directory under $WorkDir"
    }
    $mockServerBat = Join-Path $mockServerDir.FullName "bin\mockserver.bat"

    Write-Step "Starting MockServer on port $MockServerPort"
    $initJson = Join-Path $RepoRoot "test\expectations.json"
    $mockServerProcess = Start-Process -FilePath $mockServerBat `
        -ArgumentList @("run", "-p", "$MockServerPort", "--init", $initJson) `
        -RedirectStandardOutput $mockServerOutLog `
        -RedirectStandardError $mockServerErrLog `
        -WindowStyle Hidden -PassThru

    if (-not (Wait-Until -Condition { Test-MockServerReady } -Description "MockServer to be ready" -MaxRetries 15 -DelaySeconds 2)) {
        Write-Host "MockServer failed to start. stdout:"
        if (Test-Path $mockServerOutLog) { Get-Content $mockServerOutLog }
        Write-Host "MockServer failed to start. stderr:"
        if (Test-Path $mockServerErrLog) { Get-Content $mockServerErrLog }
        throw "MockServer did not become ready in time"
    }
    Write-Step "MockServer is ready"

    Write-Step "Opening firewall for inbound TCP $MockServerPort (needed for the Windows container to reach the host)"
    New-NetFirewallRule -DisplayName $FirewallRuleName -Direction Inbound -Protocol TCP -LocalPort $MockServerPort -Action Allow | Out-Null

    Write-Step "Creating a dedicated Docker NAT network"
    docker network create -d nat $NetworkName | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to create Docker network $NetworkName"
    }
    $networkInfo = docker network inspect $NetworkName | ConvertFrom-Json
    $gatewayIp = $networkInfo[0].IPAM.Config[0].Gateway
    if (-not $gatewayIp) {
        throw "Could not resolve gateway IP for network $NetworkName"
    }
    Write-Step "Resolved host gateway IP: $gatewayIp (this is how the container reaches the host-run MockServer)"
    $endpoint = "http://${gatewayIp}:${MockServerPort}/log/v1"

    Write-Step "Preparing test log file"
    New-Item -ItemType Directory -Force -Path $TestDataDir | Out-Null
    $logFile = Join-Path $TestDataDir "fbtest.log"
    New-Item -ItemType File -Force -Path $logFile | Out-Null

    Write-Step "Starting the newrelic-fluent-bit-output Windows container (via docker-compose.windows.yml)"
    # Windows containers only support directory-level bind mounts (single-file mounts fail with
    # "invalid mount config for type bind: source path must be a directory"), so stage the conf
    # in its own directory rather than mounting test/fluent-bit.windows.conf directly.
    $confDir = Join-Path $WorkDir "etc"
    New-Item -ItemType Directory -Force -Path $confDir | Out-Null
    Copy-Item -Path (Join-Path $TestDir "fluent-bit.windows.conf") -Destination (Join-Path $confDir "fluent-bit.conf") -Force

    # docker-compose.windows.yml only defines the plugin container, not MockServer - MockServer
    # isn't a container here (no Windows-container image exists for it), so unlike test.sh's
    # docker-compose.yml this can't be a symmetric two-service file. It attaches to the NAT
    # network created above via `external: true`, since ENDPOINT (derived from that network's
    # gateway IP) must be known before the container starts.
    $env:NR_FB_IMAGE = $Image
    $env:CONTAINER_NAME = $ContainerName
    $env:NETWORK_NAME = $NetworkName
    $env:HEALTH_PORT = "$HealthPort"
    $env:TESTDATA_DIR = $TestDataDir
    $env:CONF_DIR = $confDir
    $env:ENDPOINT = $endpoint

    docker compose -f $ComposeFile -p $ComposeProject up -d
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to start container $ContainerName from image $Image via docker-compose"
    }

    Write-Step "Sending test log lines"
    1..5 | ForEach-Object { Add-Content -Path $logFile -Value "Hello!" }

    $verifyBody = Get-Content (Join-Path $RepoRoot "test\verification.json") -Raw

    Write-Step "Waiting for logs to reach MockServer"
    if (-not (Wait-Until -Condition { Test-LogsDelivered -VerifyBody $verifyBody } -Description "logs to arrive at MockServer" -MaxRetries 10 -DelaySeconds 2)) {
        Write-Host "Logs did not reach MockServer in time. Diagnostics:"
        Write-Host "--- newrelic-fluent-bit-output container logs ---"
        docker logs $ContainerName
        Write-Host "--- MockServer stdout ---"
        if (Test-Path $mockServerOutLog) { Get-Content $mockServerOutLog }
        Write-Host "--- MockServer stderr ---"
        if (Test-Path $mockServerErrLog) { Get-Content $mockServerErrLog }
        throw "Functional test failed: logs never reached the mock New Relic endpoint"
    }
    Write-Step "Logs reached the mock New Relic endpoint."

    # Regression check for https://github.com/fluent/fluent-bit/issues/11904: Fluent Bit crashed
    # on Windows ("[BUG !] Bug found in _mk_event_del()") when HTTP_Server + Health_Check were
    # enabled. HTTP_Server/Health_Check are already on in fluent-bit.windows.conf, so watch the
    # container over a window to confirm it doesn't crash and the health endpoint responds.
    Write-Step "Watching for a regression of fluent/fluent-bit#11904 (HTTP_Server crash) for 60s"
    $crashCheckIterations = 12
    $crashCheckDelaySeconds = 5
    $sawHealthyResponse = $false
    for ($i = 0; $i -lt $crashCheckIterations; $i++) {
        if (-not (Test-ContainerRunning -Name $ContainerName)) {
            Write-Host "--- Container exited. Logs: ---"
            docker logs $ContainerName
            throw "Container exited while HTTP_Server was enabled - possible regression of fluent/fluent-bit#11904"
        }
        if (Test-FluentBitHealthy -Port $HealthPort) {
            $sawHealthyResponse = $true
        }
        Start-Sleep -Seconds $crashCheckDelaySeconds
    }
    if (-not $sawHealthyResponse) {
        throw "Health endpoint at port $HealthPort never responded with 200 - HTTP_Server may not have started correctly"
    }
    Write-Step "No crash after $($crashCheckIterations * $crashCheckDelaySeconds)s with HTTP_Server + Health_Check enabled"

    Write-Step "Checking DB persistence file was created (mirrors fluentBit.windowsDb in the Helm chart)"
    $dbFile = Join-Path $TestDataDir $DbFileName
    if (-not (Test-Path $dbFile)) {
        throw "Expected DB persistence file was not created at $dbFile"
    }
    Write-Step "DB persistence file confirmed at $dbFile"

    Write-Step "Success! Logs delivered, no HTTP_Server crash, DB persistence confirmed."
    exit 0
}
catch {
    Write-Host "Windows functional test failed: $_"
    exit 1
}
finally {
    Invoke-Cleanup
}
