param(
    [ValidatePattern('^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$')]
    [string]$Version = '0.2.2',
    [switch]$OfflineFull
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$distDirectory = Join-Path $projectRoot 'dist'
$binaryPath = Join-Path $projectRoot 'build\bin\starline-dsh-desktop.exe'
$licensePath = Join-Path $projectRoot 'LICENSE'
$noticePath = Join-Path $projectRoot 'NOTICE.md'
$authorsPath = Join-Path $projectRoot 'AUTHORS.md'
$wailsConfigPath = Join-Path $projectRoot 'wails.json'
$originalWailsConfig = [IO.File]::ReadAllText($wailsConfigPath)
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)

function Assert-Command {
    param([Parameter(Mandatory)][string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command not found: $Name"
    }
}

Push-Location $projectRoot
try {
    foreach ($command in @('go', 'node', 'npm', 'wails', 'makensis')) {
        Assert-Command $command
    }

    $packageVersion = ($Version -split '[-+]')[0]
    $offlinePackage = Get-Content -LiteralPath (Join-Path $projectRoot 'offline-runtime\package.json') -Raw | ConvertFrom-Json
    $dshVersion = $offlinePackage.dependencies.'@deepseek-ai/dsh'
    $wailsConfig = $originalWailsConfig | ConvertFrom-Json
    $wailsConfig.info.productVersion = $packageVersion
    [IO.File]::WriteAllText($wailsConfigPath, ($wailsConfig | ConvertTo-Json -Depth 10), $utf8WithoutBom)

    npm --prefix frontend ci
    if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency installation failed.' }
    npm --prefix frontend run docs:check
    if ($LASTEXITCODE -ne 0) { throw 'Documentation link check failed.' }
    npm --prefix frontend run typecheck
    if ($LASTEXITCODE -ne 0) { throw 'TypeScript check failed.' }
    npm --prefix frontend run build
    if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed.' }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed.' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }

    wails build -clean -trimpath -platform windows/amd64 -nsis -installscope user -ldflags "-s -w -H=windowsgui -X main.version=$Version -X main.defaultDSHVersion=$dshVersion"
    if ($LASTEXITCODE -ne 0) { throw 'Wails build failed.' }

    New-Item -ItemType Directory -Force -Path $distDirectory | Out-Null
    $portablePath = Join-Path $distDirectory 'starline-dsh-desktop-windows-amd64.zip'
    $setupPath = Join-Path $distDirectory 'starline-dsh-desktop-windows-amd64-setup.exe'
    Compress-Archive -LiteralPath @($binaryPath, $licensePath, $noticePath, $authorsPath) -DestinationPath $portablePath -Force

    $installers = @(Get-ChildItem -LiteralPath (Join-Path $projectRoot 'build\bin') -Filter '*-installer.exe')
    if ($installers.Count -ne 1) {
        throw "Expected one NSIS installer, found $($installers.Count)."
    }
    Copy-Item -LiteralPath $installers[0].FullName -Destination $setupPath -Force

    $artifacts = @($portablePath, $setupPath)
    if ($OfflineFull) {
        & (Join-Path $PSScriptRoot 'prepare-offline-runtime.ps1') -DSHVersion $dshVersion
        $offlinePath = Join-Path $distDirectory 'starline-dsh-desktop-windows-amd64-offline-full.zip'
        if (Test-Path -LiteralPath $offlinePath) {
            Remove-Item -LiteralPath $offlinePath -Force
        }
        tar.exe -a -c -f $offlinePath -C (Split-Path -Parent $binaryPath) (Split-Path -Leaf $binaryPath) -C $projectRoot LICENSE NOTICE.md AUTHORS.md offline-runtime
        if ($LASTEXITCODE -ne 0) {
            throw "Offline ZIP creation failed with exit code $LASTEXITCODE."
        }
        $artifacts += $offlinePath
    }

    foreach ($artifact in $artifacts) {
        $hash = (Get-FileHash -LiteralPath $artifact -Algorithm SHA256).Hash.ToLower()
        $hashPath = "$artifact.sha256"
        Set-Content -LiteralPath $hashPath -Value "$hash  $([IO.Path]::GetFileName($artifact))" -Encoding ascii
    }

    Get-Item -LiteralPath $artifacts | Select-Object Name, Length, LastWriteTime
}
finally {
    [IO.File]::WriteAllText($wailsConfigPath, $originalWailsConfig, $utf8WithoutBom)
    Pop-Location
}
