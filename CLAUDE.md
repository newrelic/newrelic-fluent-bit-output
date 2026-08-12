# newrelic-fluent-bit-output

## Overview
Go plugin for Fluent Bit that forwards logs to New Relic. Ships as a shared library (`.so` on Linux, `.dll` on Windows) and as Docker images.

## Key Files
- `Dockerfile` — Linux image
- `Dockerfile.windows` — Windows ltsc2022 image
- `Dockerfile.windows_older` — Windows ltsc2019 image (uses fluent-bit `4.0.3`, last version with Windows 2019 support)
- `out_newrelic.go` — main plugin entry point
- `version.go` — plugin version

## Windows Docker Images

### Important Context
The Windows runtime image uses a **multi-stage build** to override the base image:

```dockerfile
FROM fluent/fluent-bit:windows-${WINDOWS_VERSION}-${FLUENTBIT_VERSION} AS fluentbit
FROM mcr.microsoft.com/windows/servercore:ltsc${WINDOWS_VERSION} AS runtime
COPY --from=fluentbit /fluent-bit /fluent-bit
COPY --from=nrBuilder /build/out_newrelic.dll /fluent-bit/bin/out_newrelic.dll
RUN setx /M PATH "%PATH%;C:\fluent-bit\bin"
```

**Why this pattern:** `fluent/fluent-bit` Windows images bundle an outdated Windows Server base image with unpatched CVEs. Overriding with `mcr.microsoft.com/windows/servercore:ltsc2022` ensures a fresh patched base is used, which is required for Microsoft Azure Marketplace certification (Partner Center requires `OS build 10.0.20348.5386+`).

**This fix is temporary** — the image needs to be rebuilt periodically when Microsoft releases new Windows patches. A permanent fix requires the fluent-bit team to regularly rebuild their images.

### Windows 2019
`Dockerfile.windows_older` is used for Windows 2019. It is **not fixed** with the base image override because:
- fluent-bit stopped publishing Windows 2019 base images after `v4.0.3`
- AKS ended Windows Server 2019 node pool support on 2026-03-01

### Known Fluent Bit Bug
Fluent Bit `v5.0.6` crashes on Windows when `HTTP_Server` is enabled ([issue #11904](https://github.com/fluent/fluent-bit/issues/11904)). The crash manifests as:
```
[BUG !] Bug found in _mk_event_del() at mk_event_libevent.c:235
```
**Workaround:** Disable the HTTP server in the Fluent Bit config (`HTTP_Server Off`) and disable the liveness probe.

## CI/CD Pipelines
- `pr.yaml` — PR validation: builds Windows image (no push, no test)
- `merge-to-master.yml` — builds and pushes Windows images to Docker Hub on merge
  - `windows-docker-images` job — builds `Dockerfile.windows` for ltsc2022
  - `windows-docker-images-older-version` job — builds `Dockerfile.windows_older` for ltsc2019

## Testing Windows Image Locally

### Requirements
- Windows Server 2022 EC2 instance with Docker (use ECS-optimized AMI)
- Connect via SSM (no inbound rules needed)

### Build from Branch
```powershell
Invoke-WebRequest `
  -Uri "https://github.com/newrelic/newrelic-fluent-bit-output/archive/refs/heads/<branch>.zip" `
  -OutFile "C:\test\repo.zip"
Expand-Archive -Path "C:\test\repo.zip" -DestinationPath "C:\test\" -Force
cd "C:\test\newrelic-fluent-bit-output-<branch>"
docker build -f Dockerfile.windows --build-arg WINDOWS_VERSION=2022 -t newrelic-fluentbit-test:latest .
```

### Verify Base Image Version
```powershell
docker run newrelic-fluentbit-test:latest powershell -Command `
  "(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').CurrentBuildNumber + '.' + (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').UBR"
```
Expected: `20348.5386` or higher.

### Run with Full Test Config

This config mirrors what the Helm chart uses on AKS Windows nodes and covers:
- **Log forwarding** — tail input sending to New Relic
- **DB persistence** — tracks log offsets across restarts (same as `fluentBit.windowsDb` in Helm)
- **HTTP server + health check** — equivalent to liveness probe in Helm (tests fluent-bit [#11904](https://github.com/fluent/fluent-bit/issues/11904) fix in `5.0.9+`)
- **Windows log path** — uses same path pattern as Helm (`C:\var\log\containers\*.log`)

```powershell
New-Item -Path "C:\test\etc" -ItemType Directory -Force
New-Item -Path "C:\test\logs\containers" -ItemType Directory -Force
New-Item -Path "C:\test\logs\containers\fbtest.log" -ItemType File -Force

@'
[SERVICE]
    Flush         5
    Daemon        Off
    Log_Level     debug
    HTTP_Server   On
    HTTP_Listen   0.0.0.0
    HTTP_Port     2020
    Health_Check  On
    DB            C:\test\logs\flb_kube.db

[INPUT]
    Name              tail
    Path              C:\test\logs\containers\*.log
    DB                C:\test\logs\flb_kube.db
    Mem_Buf_Limit     7MB
    Skip_Long_Lines   On
    Refresh_Interval  10

[OUTPUT]
    Name        newrelic
    Match       *
    licenseKey  ${LICENSE_KEY}
    endpoint    https://log-api.newrelic.com/log/v1
'@ | Out-File -FilePath "C:\test\etc\fluent-bit.conf" -Encoding ascii

docker run `
  -e "LICENSE_KEY=<your-nr-license-key>" `
  -v "C:\test\etc:C:\fluent-bit\etc" `
  -v "C:\test\logs:C:\test\logs" `
  -p 2020:2020 `
  newrelic-fluentbit-test:latest `
  fluent-bit.exe -c C:\fluent-bit\etc\fluent-bit.conf -e C:\fluent-bit\bin\out_newrelic.dll
```

**Write test logs** (separate SSM session):
```powershell
"Hello from Windows container test $(Get-Date)" | Out-File "C:\test\logs\containers\fbtest2.log" -Encoding ascii
```

**Verify health endpoint** (liveness probe equivalent):
```powershell
Invoke-WebRequest -Uri "http://localhost:2020/api/v1/health" -UseBasicParsing
```

**Verify DB file created** (persistence working):
```powershell
Test-Path "C:\test\logs\flb_kube.db"  # Should return True
```

**Check New Relic Logs UI** for the test message.

**After 1 minute with no crash:**
- ✅ HTTP server bug ([#11904](https://github.com/fluent/fluent-bit/issues/11904)) is fixed in this version
- ✅ Liveness probe can be re-enabled in Helm chart

**If crashes with `[BUG !] Bug found in _mk_event_del()`:**
- Set `HTTP_Server Off` and remove `Health_Check On` from config
- Disable liveness probe in Helm (`livenessProbe.enabled: false`)
- Monitor upstream issue [#11904](https://github.com/fluent/fluent-bit/issues/11904)

## Helm Chart Testing (AKS)
Uses `newrelic/helm-charts` — `newrelic-logging` chart. Windows image tag is built as `{repository}:{tag}-{imageTagSuffix}`.

```yaml
# custom-values.yaml
licenseKey: <your-nr-license-key>
cluster: test-cluster
enableWindows: true
enableLinux: false
image:
  repository: newrelic/newrelic-fluentbit-output
  tag: "3.7.1"
livenessProbe:
  enabled: false
fluentBit:
  config:
    service: |
      [SERVICE]
          Flush 1
          Log_Level info
          Daemon off
          Parsers_File parsers.conf
          HTTP_Server Off
```

```bash
helm repo add newrelic https://helm-charts.newrelic.com
helm install newrelic-logging newrelic/newrelic-logging -f custom-values.yaml
```

## Related Links
- PR fixing Windows CVEs: https://github.com/newrelic/newrelic-fluent-bit-output/pull/263
- Fluent Bit Windows HTTP server crash: https://github.com/fluent/fluent-bit/issues/11904
- fluent-bit-package E2E tests: `../fluent-bit-package`
