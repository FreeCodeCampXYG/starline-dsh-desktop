param(
    [Parameter(Mandatory = $true)]
    [string]$DSHVersion
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeRoot = Join-Path $repositoryRoot 'offline-runtime'
$packageJson = Get-Content -LiteralPath (Join-Path $runtimeRoot 'package.json') -Raw | ConvertFrom-Json
$lockedVersion = $packageJson.dependencies.'@deepseek-ai/dsh'
$nodeVersion = (Get-Content -LiteralPath (Join-Path $runtimeRoot 'node-version.txt') -Raw).Trim()

if ($lockedVersion -ne $DSHVersion) {
    throw "offline-runtime/package.json pins DSH $lockedVersion, expected $DSHVersion."
}

npm --prefix $runtimeRoot ci --omit=dev --ignore-scripts --workspaces=false
if ($LASTEXITCODE -ne 0) {
    throw "npm ci failed with exit code $LASTEXITCODE."
}

$nodeSource = (Get-Command node -ErrorAction Stop).Source
$actualNodeVersion = (& $nodeSource --version).Trim().TrimStart('v')
if ($actualNodeVersion -ne $nodeVersion) {
    throw "Node $actualNodeVersion is active, expected pinned Node $nodeVersion."
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

$payloadBytes = (
    Get-ChildItem -LiteralPath $runtimeRoot -Recurse -File |
        Measure-Object -Property Length -Sum
).Sum
Write-Output "Prepared Windows offline runtime: $payloadBytes bytes"
