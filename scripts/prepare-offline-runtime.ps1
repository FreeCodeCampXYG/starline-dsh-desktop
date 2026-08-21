param(
    [Parameter(Mandatory = $true)]
    [string]$DSHVersion,
    [switch]$SkipLockRefresh,
    [switch]$SkipInstall
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeRoot = Join-Path $repositoryRoot 'offline-runtime'
$packageJson = Get-Content -LiteralPath (Join-Path $runtimeRoot 'package.json') -Raw | ConvertFrom-Json
$lockedVersion = $packageJson.dependencies.'@deepseek-ai/dsh'
$nodeVersion = (Get-Content -LiteralPath (Join-Path $runtimeRoot 'node-version.txt') -Raw).Trim()
$verifier = Join-Path $PSScriptRoot 'verify-offline-runtime.mjs'

if ($lockedVersion -ne $DSHVersion) {
    throw "offline-runtime/package.json pins DSH $lockedVersion, expected $DSHVersion."
}

$nodeSource = (Get-Command node -ErrorAction Stop).Source
$actualNodeVersion = (& $nodeSource --version).Trim().TrimStart('v')
if ($actualNodeVersion -ne $nodeVersion) {
    throw "Node $actualNodeVersion is active, expected pinned Node $nodeVersion."
}

if (-not $SkipLockRefresh) {
    # lock 只在单独的 runner 上解析一次；平台任务复用它，避免并发解析造成重复下载。

    for ($attempt = 1; $attempt -le 3; $attempt++) {
        npm --prefix $runtimeRoot install --package-lock-only --ignore-scripts --workspaces=false --fetch-retries=2 --fetch-retry-mintimeout=5000 --fetch-retry-maxtimeout=30000 --fetch-timeout=120000
        if ($LASTEXITCODE -eq 0) {
            break
        }
        if ($attempt -eq 3) {
            throw "Offline package-lock resolution failed after 3 attempts (last exit code $LASTEXITCODE)."
        }
        $delay = $attempt * 15
        Write-Warning "Package registry resolution failed; retrying in ${delay}s (attempt $($attempt + 1)/3)."
        Start-Sleep -Seconds $delay
    }
}

if (-not $SkipInstall) {
    npm --prefix $runtimeRoot ci --omit=dev --ignore-scripts --workspaces=false --fetch-retries=2 --fetch-retry-mintimeout=5000 --fetch-retry-maxtimeout=30000 --fetch-timeout=120000
    if ($LASTEXITCODE -ne 0) {
        throw "npm ci failed with exit code $LASTEXITCODE."
    }
}
elseif (-not (Test-Path -LiteralPath (Join-Path $runtimeRoot 'node_modules') -PathType Container)) {
    throw "-SkipInstall requires an existing offline-runtime/node_modules cache."
}

& $nodeSource (Join-Path $repositoryRoot 'scripts\sync-offline-runtime-metadata.mjs') $runtimeRoot
if ($LASTEXITCODE -ne 0) {
    throw "Offline approval metadata synchronization failed with exit code $LASTEXITCODE."
}

& $nodeSource $verifier preflight $runtimeRoot
if ($LASTEXITCODE -ne 0) {
    throw "Offline lifecycle-script preflight failed with exit code $LASTEXITCODE."
}

# npm ci keeps every lifecycle script disabled. Only the pinned and hash-checked
# node-pty install/postinstall pair is allowed to run here; it may select a
# reviewed platform prebuild instead of compiling from source.
npm --prefix $runtimeRoot rebuild node-pty --foreground-scripts --ignore-scripts=false --workspaces=false
if ($LASTEXITCODE -ne 0) {
    throw "Approved node-pty lifecycle preparation failed with exit code $LASTEXITCODE."
}

$permissionFix = Join-Path $runtimeRoot 'node_modules\@deepseek-ai\dsh-subprocess-local\scripts\ensure-spawn-helper.mjs'
& $nodeSource $permissionFix
if ($LASTEXITCODE -ne 0) {
    throw "Approved node-pty permission repair failed with exit code $LASTEXITCODE."
}

$nodeRoot = Split-Path -Parent $nodeSource
$licenseSource = Join-Path $nodeRoot 'LICENSE'
Copy-Item -LiteralPath $nodeSource -Destination (Join-Path $runtimeRoot 'node.exe') -Force
$licenseDestination = Join-Path $runtimeRoot 'LICENSE-node.txt'
$expectedLicenseHash = '148eacf7863ef4329224a29398623077200a27194aa075569faf4a0a85566ca5'
if (Test-Path -LiteralPath $licenseSource -PathType Leaf) {
    Copy-Item -LiteralPath $licenseSource -Destination $licenseDestination -Force
}
elseif (-not (Test-Path -LiteralPath $licenseDestination -PathType Leaf)) {
    $licenseURL = "https://raw.githubusercontent.com/nodejs/node/v$nodeVersion/LICENSE"
    Invoke-WebRequest -UseBasicParsing -Uri $licenseURL -OutFile $licenseDestination -TimeoutSec 60
}
$licenseText = [IO.File]::ReadAllText($licenseDestination).Replace("`r`n", "`n")
[IO.File]::WriteAllText($licenseDestination, $licenseText, [Text.UTF8Encoding]::new($false))
$licenseHash = (Get-FileHash -LiteralPath $licenseDestination -Algorithm SHA256).Hash.ToLowerInvariant()
if ($licenseHash -ne $expectedLicenseHash) {
    throw "Node license SHA-256 mismatch: $licenseHash"
}
[IO.File]::WriteAllText(
    (Join-Path $runtimeRoot 'dsh-version.txt'),
    "$DSHVersion`n",
    [Text.UTF8Encoding]::new($false)
)

$dshEntry = Join-Path $runtimeRoot 'node_modules\@deepseek-ai\dsh\lib\bin.js'
& (Join-Path $runtimeRoot 'node.exe') $dshEntry --version
if ($LASTEXITCODE -ne 0) {
    throw "Bundled DSH smoke test failed with exit code $LASTEXITCODE."
}

& (Join-Path $runtimeRoot 'node.exe') $verifier verify $runtimeRoot
if ($LASTEXITCODE -ne 0) {
    throw "Bundled node-pty functional test failed with exit code $LASTEXITCODE."
}

$payloadBytes = (
    Get-ChildItem -LiteralPath $runtimeRoot -Recurse -File |
        Measure-Object -Property Length -Sum
).Sum
Write-Output "Prepared Windows offline runtime: $payloadBytes bytes"
