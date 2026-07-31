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
$FirewallRuleName = "newrelic-fb-mockserver-test-1080"
$MockServerPort = 1080

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
    docker rm -f $ContainerName 2>$null | Out-Null
    docker network rm $NetworkName 2>$null | Out-Null
    if (Test-Path $TestDataDir) { Remove-Item -Recurse -Force $TestDataDir -ErrorAction SilentlyContinue }
    if (Test-Path $WorkDir) { Remove-Item -Recurse -Force $WorkDir -ErrorAction SilentlyContinue }
    Remove-NetFirewallRule -DisplayName $FirewallRuleName -ErrorAction SilentlyContinue
}

function Invoke-Cleanup {
    Write-Step "Cleaning up"

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

    Write-Step "Starting the newrelic-fluent-bit-output Windows container"
    $confPath = Join-Path $TestDir "fluent-bit.windows.conf"
    docker run -d --name $ContainerName --network $NetworkName `
        -v "${TestDataDir}:C:\testdata" `
        -v "${confPath}:C:\fluent-bit\etc\fluent-bit.conf" `
        -e "FILE_PATH=C:\testdata\fbtest.log" `
        -e "API_KEY=some-insert-key" `
        -e "ENDPOINT=$endpoint" `
        $Image `
        fluent-bit.exe -c C:\fluent-bit\etc\fluent-bit.conf -e C:\fluent-bit\bin\out_newrelic.dll | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to start container $ContainerName from image $Image"
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

    Write-Step "Success! Logs reached the mock New Relic endpoint."
    exit 0
}
catch {
    Write-Host "Windows functional test failed: $_"
    exit 1
}
finally {
    Invoke-Cleanup
}
