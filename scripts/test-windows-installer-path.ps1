param(
    [string]$BinaryPath = 'build\bin\starline-dsh-desktop.exe'
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$binary = [IO.Path]::GetFullPath((Join-Path $repositoryRoot $BinaryPath))
$installerScript = Join-Path $repositoryRoot 'build\windows\installer\project.nsi'
$testID = Get-Date -Format 'yyyyMMddHHmmssfff'
$projectName = "starline-dsh-path-test-$testID"
$productName = "Starline DSH Path Test $testID"
$customInstall = Join-Path ([IO.Path]::GetTempPath()) "星痕 安装路径测试\$testID"
$defaultInstall = Join-Path $env:LOCALAPPDATA "Programs\$productName"
$testInstaller = Join-Path $repositoryRoot "build\bin\$projectName-amd64-installer.exe"
$uninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\StarlineTest$productName"

if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) {
    throw "Test binary not found: $binary"
}
if ((Test-Path -LiteralPath $customInstall) -or (Test-Path -LiteralPath $defaultInstall)) {
    throw 'Generated installer test directories must not exist before the test.'
}

function Invoke-Installer {
    param(
        [Parameter(Mandatory)][string]$Path
    )
    $arguments = '/S'
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $Path
    $startInfo.Arguments = $arguments
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $process = [Diagnostics.Process]::Start($startInfo)
    $process.WaitForExit()
    if ($process.ExitCode -ne 0) {
        throw "Installer exited with code $($process.ExitCode)."
    }
}

Push-Location (Split-Path -Parent $installerScript)
try {
    & makensis.exe `
        "-DARG_WAILS_AMD64_BINARY=$binary" `
        "-DINFO_PROJECTNAME=$projectName" `
        '-DINFO_COMPANYNAME=StarlineTest' `
        "-DINFO_PRODUCTNAME=$productName" `
        '-DINFO_PRODUCTVERSION=0.1.1' `
        '-DINFO_COPYRIGHT=Copyright StarlineTest' `
        '-DWAILS_INSTALL_SCOPE=user' `
        '-DREQUEST_EXECUTION_LEVEL=user' `
        $installerScript
    if ($LASTEXITCODE -ne 0) {
        throw "makensis failed with exit code $LASTEXITCODE."
    }

    New-Item -Path $uninstallKey -Force | Out-Null
    New-ItemProperty -LiteralPath $uninstallKey -Name InstallLocation -Value $customInstall -PropertyType String -Force | Out-Null
    Invoke-Installer -Path $testInstaller
    $installedBinary = Join-Path $customInstall "$projectName.exe"
    $uninstaller = Join-Path $customInstall 'uninstall.exe'
    $installedLicense = Join-Path $customInstall 'LICENSE.txt'
    $installedNotice = Join-Path $customInstall 'NOTICE.md'
    $installedAuthors = Join-Path $customInstall 'AUTHORS.md'
    $recordedPath = (Get-ItemProperty -LiteralPath $uninstallKey -Name InstallLocation).InstallLocation
    if (
        -not (Test-Path -LiteralPath $installedBinary -PathType Leaf) -or
        -not (Test-Path -LiteralPath $uninstaller -PathType Leaf) -or
        -not (Test-Path -LiteralPath $installedLicense -PathType Leaf) -or
        -not (Test-Path -LiteralPath $installedNotice -PathType Leaf) -or
        -not (Test-Path -LiteralPath $installedAuthors -PathType Leaf)
    ) {
        $actualFiles = @(
            Get-ChildItem -LiteralPath $recordedPath -File -ErrorAction SilentlyContinue |
                Select-Object -ExpandProperty Name
        ) -join ', '
        throw "Installer files mismatch. Recorded path: $recordedPath. Files: $actualFiles"
    }
    if ([IO.Path]::GetFullPath($recordedPath) -ne [IO.Path]::GetFullPath($customInstall)) {
        throw "InstallLocation mismatch: $recordedPath"
    }

    $expectedBinaryHash = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash
    [IO.File]::WriteAllText($installedBinary, 'upgrade-overwrite-marker', [Text.UTF8Encoding]::new($false))
    Invoke-Installer -Path $testInstaller
    if (
        -not (Test-Path -LiteralPath $installedBinary -PathType Leaf) -or
        (Get-FileHash -LiteralPath $installedBinary -Algorithm SHA256).Hash -ne $expectedBinaryHash
    ) {
        throw 'The second installation did not overwrite the existing binary in the recorded custom directory.'
    }
    if (Test-Path -LiteralPath $defaultInstall) {
        throw "The second installation unexpectedly used the default directory: $defaultInstall"
    }

    [pscustomobject]@{
        CustomInstallPath = $customInstall
        UnicodeAndSpaces  = 'passed'
        RememberedPath    = 'passed'
        InPlaceOverwrite  = 'passed'
        MetadataFiles     = 'passed'
    }
}
finally {
    foreach ($candidate in @(
        (Join-Path $customInstall 'uninstall.exe'),
        (Join-Path $defaultInstall 'uninstall.exe')
    )) {
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            $cleanup = Start-Process -FilePath $candidate -ArgumentList '/S' -Wait -PassThru -WindowStyle Hidden
            if ($cleanup.ExitCode -ne 0) {
                Write-Warning "Test uninstaller failed with code $($cleanup.ExitCode): $candidate"
            }
        }
    }
    if (Test-Path -LiteralPath $testInstaller -PathType Leaf) {
        Remove-Item -LiteralPath $testInstaller -Force
    }
    if (Test-Path -LiteralPath $uninstallKey) {
        Remove-Item -LiteralPath $uninstallKey -Force
    }
    Pop-Location
}
