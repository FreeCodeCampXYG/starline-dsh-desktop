param(
    [Parameter(Mandatory = $true)]
    [string]$DSHVersion,
    [string]$DesktopVersion,
    [switch]$Commit,
    [switch]$Tag,
    [switch]$Push
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
$semverPattern = '^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$'
$targetFiles = @(
    'CHANGELOG.md',
    'README.md',
    'docs/ARCHITECTURE.md',
    'docs/BUILDING.md',
    'docs/KNOWN_ISSUES.md',
    'docs/TROUBLESHOOTING.md',
    'frontend/src/main.ts',
    'main.go',
    'offline-runtime/README.md',
    'offline-runtime/package.json'
)

function Assert-Command {
    param([Parameter(Mandatory)][string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command not found: $Name"
    }
}

function Read-Utf8 {
    param([Parameter(Mandatory)][string]$Path)
    return [IO.File]::ReadAllText($Path, [Text.Encoding]::UTF8)
}

function Write-Utf8 {
    param([Parameter(Mandatory)][string]$Path, [Parameter(Mandatory)][string]$Content)
    [IO.File]::WriteAllText($Path, $Content, $utf8WithoutBom)
}

function Replace-Required {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$Old,
        [Parameter(Mandatory)][string]$New
    )
    $content = Read-Utf8 $Path
    if (-not $content.Contains($Old)) {
        throw "Expected text was not found in ${Path}: $Old"
    }
    Write-Utf8 $Path ($content.Replace($Old, $New))
}

function Confirm-Step {
    param([Parameter(Mandatory)][string]$Prompt)
    $answer = Read-Host "$Prompt [y/N]"
    return $answer -match '^(y|yes)$'
}

function Get-NextDesktopVersion {
    $latestTag = @(git tag --list 'v[0-9]*' --sort=-version:refname | Select-Object -First 1)
    if ($latestTag.Count -eq 0) {
        return '0.1.0'
    }
    $match = [regex]::Match($latestTag[0], '^v(\d+)\.(\d+)\.(\d+)$')
    if (-not $match.Success) {
        throw "Latest tag is not a stable semantic version: $($latestTag[0])"
    }
    return '{0}.{1}.{2}' -f [int]$match.Groups[1].Value, [int]$match.Groups[2].Value, ([int]$match.Groups[3].Value + 1)
}

Push-Location $projectRoot
$commitCreated = $false
$gitStageStarted = $false
$backupRoot = Join-Path ([IO.Path]::GetTempPath()) ('starline-dsh-release-' + [guid]::NewGuid().ToString('N'))
try {
    foreach ($command in @('git', 'node')) {
        Assert-Command $command
    }
    if ((git status --porcelain)) {
        throw "工作区不干净。请先提交或暂存现有改动，再运行发布准备脚本。"
    }

    $packagePath = Join-Path $projectRoot 'offline-runtime\package.json'
    $packageContent = Read-Utf8 $packagePath
    $package = $packageContent | ConvertFrom-Json
    $oldDshVersion = [string]$package.dependencies.'@deepseek-ai/dsh'

    if ($DSHVersion -notmatch $semverPattern) {
        throw "DSH 版本格式无效：$DSHVersion"
    }
    if ($DSHVersion -eq $oldDshVersion) {
        throw "目标 DSH 版本与当前版本相同：$DSHVersion"
    }
    if ([string]::IsNullOrWhiteSpace($DesktopVersion)) {
        $DesktopVersion = Get-NextDesktopVersion
    }
    if ($DesktopVersion -notmatch $semverPattern) {
        throw "Desktop 版本格式无效：$DesktopVersion"
    }
    $dshLabel = $DSHVersion
    $rcMatch = [regex]::Match($DSHVersion, '-rc\.(\d+)$')
    if ($rcMatch.Success) {
        $dshLabel = 'rc.' + $rcMatch.Groups[1].Value
    }
    $tagName = "v$DesktopVersion"
    if (@(git tag --list $tagName).Count -gt 0) {
        throw "Git tag 已存在：$tagName"
    }

    New-Item -ItemType Directory -Force -Path $backupRoot | Out-Null
    foreach ($relativePath in $targetFiles) {
        $sourcePath = Join-Path $projectRoot ($relativePath -replace '/', '\\')
        if (Test-Path -LiteralPath $sourcePath) {
            $backupPath = Join-Path $backupRoot ($relativePath -replace '/', '\\')
            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $backupPath) | Out-Null
            Copy-Item -LiteralPath $sourcePath -Destination $backupPath -Force
        }
    }

    $updatedPackage = [regex]::Replace(
        $packageContent,
        '("@deepseek-ai/dsh"\s*:\s*")[^"]+(")(,?)',
        {
            param($match)
            return $match.Groups[1].Value + $DSHVersion + $match.Groups[2].Value + $match.Groups[3].Value
        },
        1
    )
    if ($updatedPackage -eq $packageContent) {
        throw 'offline-runtime/package.json 中未找到 @deepseek-ai/dsh 版本字段。'
    }
    Write-Utf8 $packagePath $updatedPackage

    $oldDefaultVersionText = 'defaultDSHVersion = "' + $oldDshVersion + '"'
    $newDefaultVersionText = 'defaultDSHVersion = "' + $DSHVersion + '"'
    Replace-Required -Path (Join-Path $projectRoot 'main.go') -Old $oldDefaultVersionText -New $newDefaultVersionText
    $oldFrontendVersionText = 'dshVersion: "' + $oldDshVersion + '"'
    $newFrontendVersionText = 'dshVersion: "' + $DSHVersion + '"'
    Replace-Required -Path (Join-Path $projectRoot 'frontend\src\main.ts') -Old $oldFrontendVersionText -New $newFrontendVersionText
    Replace-Required -Path (Join-Path $projectRoot 'offline-runtime\README.md') -Old "@deepseek-ai/dsh@$oldDshVersion" -New "@deepseek-ai/dsh@$DSHVersion"
    Replace-Required -Path (Join-Path $projectRoot 'docs\BUILDING.md') -Old "prepare-offline-runtime.ps1 -DSHVersion $oldDshVersion" -New "prepare-offline-runtime.ps1 -DSHVersion $DSHVersion"
    Replace-Required -Path (Join-Path $projectRoot 'docs\BUILDING.md') -Old "prepare-offline-runtime.sh $oldDshVersion" -New "prepare-offline-runtime.sh $DSHVersion"
    Replace-Required -Path (Join-Path $projectRoot 'docs\TROUBLESHOOTING.md') -Old "@deepseek-ai/dsh@$oldDshVersion" -New "@deepseek-ai/dsh@$DSHVersion"

    $readmePath = Join-Path $projectRoot 'README.md'
    $readme = Read-Utf8 $readmePath
    $readme = $readme.Replace("@deepseek-ai/dsh@$oldDshVersion", "@deepseek-ai/dsh@$DSHVersion")
    $readme = $readme.Replace('下一轮 `offline-full` 也固定 rc.7', ('下一轮 `offline-full` 也固定 {0}' -f $dshLabel))
    $readme = $readme.Replace('当前 main 的下一轮 `offline-full` 已固定 rc.7', ('当前 main 的下一轮 `offline-full` 已固定 {0}' -f $dshLabel))
    Write-Utf8 $readmePath $readme

    Replace-Required -Path (Join-Path $projectRoot 'docs\ARCHITECTURE.md') -Old '当前 main 把下一轮离线闭包固定为 rc.7' -New "当前 main 把下一轮离线闭包固定为 $dshLabel"
    Replace-Required -Path (Join-Path $projectRoot 'docs\KNOWN_ISSUES.md') -Old '当前 main 又针对 rc.7 的依赖变化更新了门禁' -New "当前 main 又针对 $dshLabel 的依赖变化更新了门禁"

    $changelogPath = Join-Path $projectRoot 'CHANGELOG.md'
    $changelog = Read-Utf8 $changelogPath
    $entry = ('- `offline-full` 离线闭包和 Desktop 默认 DSH 版本更新为 `@deepseek-ai/dsh@{0}`；锁文件、依赖 integrity、原生模块和最终归档由本次 tag 的 GitHub Actions runner 下载并验证。' -f $DSHVersion)
    if (-not $changelog.Contains($entry)) {
        $replacement = '$1' + $entry + "`n"
        $changelog = [regex]::Replace($changelog, '(?s)(## \[未发布\].*?### 变更\r?\n)', $replacement, 1)
    }
    $versionHeading = '## [' + $DesktopVersion + '] - ' + (Get-Date -Format 'yyyy-MM-dd')
    if ($changelog.Contains('## [' + $DesktopVersion + ']')) {
        throw "CHANGELOG 已存在 Desktop 版本段：$DesktopVersion"
    }
    $unreleasedReplacement = "## [未发布]`n`n### 变更`n`n" + $versionHeading + "`n"
    $changelog = [regex]::Replace($changelog, '(?m)^## \[未发布\]\r?\n', $unreleasedReplacement, 1)
    Write-Utf8 $changelogPath $changelog

    Write-Output "已准备 DSH $DSHVersion；建议 Desktop tag：$tagName"
    Write-Output '本机未执行 npm 下载、npm pack 或离线包构建；GitHub Actions 将在 tag runner 上刷新 lock/approval 并构建六平台资产。'
    git diff --check
    node scripts/check-doc-links.mjs
    if ($LASTEXITCODE -ne 0) { throw '文档链接检查失败。' }
    git diff --stat

    if ($Commit) {
        if (-not (Confirm-Step "确认暂存并提交 $tagName")) { throw '用户取消提交。' }
        $gitStageStarted = $true
        git add -- $targetFiles
        git diff --cached --check
        git commit -m "release: prepare DSH $DSHVersion and Desktop $DesktopVersion" -m "Update the pinned offline DSH closure and default runtime to $DSHVersion; let the tagged GitHub Actions build perform native dependency preparation and packaging."
        if ($LASTEXITCODE -ne 0) { throw 'Git commit failed.' }
        $commitCreated = $true
    }
    if ($Tag) {
        if (-not $Commit) { throw '-Tag requires -Commit.' }
        if (-not (Confirm-Step "确认创建 annotated tag $tagName")) { throw '用户取消打 tag。' }
        git tag -a $tagName -m "Starline DSH Desktop ${tagName}: DSH $DSHVersion offline-full release"
        if ($LASTEXITCODE -ne 0) { throw 'Git tag failed.' }
    }
    if ($Push) {
        if (-not $Tag) { throw '-Push requires -Tag.' }
        if (-not (Confirm-Step "确认推送 main 和 $tagName 到 origin")) { throw '用户取消推送。' }
        git push origin 'refs/heads/main:refs/heads/main'
        if ($LASTEXITCODE -ne 0) { throw 'Git main push failed.' }
        git push origin "refs/tags/${tagName}:refs/tags/${tagName}"
        if ($LASTEXITCODE -ne 0) { throw 'Git tag push failed.' }
    }
    if (-not $Commit) {
        Write-Output "未执行 Git 提交。本次已生成版本改动；审阅后请手工提交/打 tag/推送，或先恢复工作区再用 -Commit -Tag -Push 一次性重跑。"
    }
}
catch {
    if (-not $commitCreated -and -not $gitStageStarted -and (Test-Path -LiteralPath $backupRoot)) {
        foreach ($relativePath in $targetFiles) {
            $backupPath = Join-Path $backupRoot ($relativePath -replace '/', '\\')
            $targetPath = Join-Path $projectRoot ($relativePath -replace '/', '\\')
            if (Test-Path -LiteralPath $backupPath) {
                Copy-Item -LiteralPath $backupPath -Destination $targetPath -Force
            }
        }
    }
    throw
}
finally {
    if (Test-Path -LiteralPath $backupRoot) {
        Remove-Item -LiteralPath $backupRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    Pop-Location
}
