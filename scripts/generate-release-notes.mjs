import { readFile, readdir, stat, writeFile } from 'node:fs/promises'
import path from 'node:path'

const versionPattern = /^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/

function parseArguments(values) {
  const options = new Map()
  for (let index = 0; index < values.length; index += 2) {
    const key = values[index]
    const value = values[index + 1]
    if (!key?.startsWith('--') || value === undefined) {
      throw new Error(`Invalid argument near: ${key ?? '<end>'}`)
    }
    options.set(key.slice(2), value)
  }
  return options
}

function requireOption(options, name) {
  const value = options.get(name)
  if (!value) {
    throw new Error(`Missing required option: --${name}`)
  }
  return value
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function extractChangelogSection(markdown, version) {
  const lines = markdown.split(/\r?\n/)
  const heading = new RegExp(`^## \\[${escapeRegExp(version)}\\](?: - .+)?$`)
  const start = lines.findIndex((line) => heading.test(line))
  if (start === -1) {
    throw new Error(`CHANGELOG.md does not contain a [${version}] section.`)
  }

  let end = lines.length
  for (let index = start + 1; index < lines.length; index += 1) {
    if (lines[index].startsWith('## [') || /^\[[^\]]+\]:/.test(lines[index])) {
      end = index
      break
    }
  }

  const body = lines.slice(start + 1, end).join('\n').trim()
  if (!body) {
    throw new Error(`CHANGELOG.md section [${version}] is empty.`)
  }
  return body
}

function parseChecksumManifest(contents, source) {
  const checksums = new Map()
  for (const rawLine of contents.split(/\r?\n/)) {
    const line = rawLine.trimEnd()
    if (!line) {
      continue
    }
    const match = line.match(/^([0-9a-fA-F]{64})\s{2}(.+)$/)
    if (!match) {
      throw new Error(`Invalid checksum line in ${source}: ${line}`)
    }
    checksums.set(match[2], match[1].toLowerCase())
  }
  return checksums
}

function formatSize(bytes) {
  const mebibyte = 1024 * 1024
  if (bytes >= mebibyte) {
    return `${(bytes / mebibyte).toFixed(1)} MiB`
  }
  return `${Math.ceil(bytes / 1024)} KiB`
}

function assetUrl(repository, tag, fileName) {
  return `https://github.com/${repository}/releases/download/${encodeURIComponent(tag)}/${encodeURIComponent(fileName)}`
}

function linkFor(repository, tag, asset, label) {
  return `[${label} · ${formatSize(asset.size)}](${assetUrl(repository, tag, asset.name)})`
}

function parseReleaseRequirements(contents) {
  let requirements
  try {
    requirements = JSON.parse(contents)
  } catch (error) {
    throw new Error(`Invalid release requirements JSON: ${error.message}`)
  }

  if (requirements.schemaVersion !== 1 || !/^\d{4}-\d{2}-\d{2}$/.test(requirements.revision ?? '')) {
    throw new Error('release-requirements.json has an unsupported schema or revision.')
  }
  if (!Array.isArray(requirements.platforms) || requirements.platforms.length !== 3) {
    throw new Error('release-requirements.json must define exactly three platform groups.')
  }
  for (const platform of requirements.platforms) {
    for (const field of ['platform', 'supportedBaseline', 'requiredRuntime']) {
      if (typeof platform[field] !== 'string' || !platform[field].trim()) {
        throw new Error(`release-requirements.json has an invalid platform.${field}.`)
      }
    }
  }
  for (const field of ['onlineRuntime', 'offlineRuntime', 'unsupported']) {
    if (typeof requirements[field] !== 'string' || !requirements[field].trim()) {
      throw new Error(`release-requirements.json has an invalid ${field}.`)
    }
  }
  return requirements
}

const options = parseArguments(process.argv.slice(2))
const version = requireOption(options, 'version')
const tag = requireOption(options, 'tag')
const repository = requireOption(options, 'repository')
const assetDirectory = path.resolve(requireOption(options, 'asset-dir'))
const outputPath = path.resolve(requireOption(options, 'output'))
const changelogPath = path.resolve(options.get('changelog') ?? 'CHANGELOG.md')
const requirementsPath = path.resolve(options.get('requirements') ?? 'release-requirements.json')
const nodeVersionPath = path.resolve(options.get('node-version') ?? 'offline-runtime/node-version.txt')

if (!versionPattern.test(version)) {
  throw new Error(`Invalid version: ${version}`)
}
if (tag !== `v${version}`) {
  throw new Error(`Tag ${tag} does not match version ${version}.`)
}
if (!/^[^/\s]+\/[^/\s]+$/.test(repository)) {
  throw new Error(`Invalid GitHub repository: ${repository}`)
}

const rows = [
  {
    os: 'Windows',
    cpu: 'x64（Intel / AMD）',
    online: [
      ['windows-x64-setup-online.exe', 'Setup 安装版'],
      ['windows-x64-portable-online.zip', 'Portable 便携版'],
    ],
    offline: ['windows-x64-portable-offline-full.zip', '完整离线便携版'],
  },
  {
    os: 'Windows',
    cpu: 'ARM64（Windows on ARM）',
    online: [
      ['windows-arm64-setup-online.exe', 'Setup 安装版'],
      ['windows-arm64-portable-online.zip', 'Portable 便携版'],
    ],
    offline: ['windows-arm64-portable-offline-full.zip', '完整离线便携版'],
  },
  {
    os: 'macOS',
    cpu: 'Intel x64',
    online: [['macos-intel-x64-app-online.zip', '应用包']],
    offline: ['macos-intel-x64-app-offline-full.zip', '完整离线应用包'],
  },
  {
    os: 'macOS',
    cpu: 'Apple Silicon ARM64',
    online: [['macos-apple-silicon-arm64-app-online.zip', '应用包']],
    offline: ['macos-apple-silicon-arm64-app-offline-full.zip', '完整离线应用包'],
  },
  {
    os: 'Linux',
    cpu: 'x64（Intel / AMD）',
    online: [
      ['linux-x64-deb-online.deb', 'DEB 安装包'],
      ['linux-x64-portable-online.tar.gz', 'Portable 便携版'],
    ],
    offline: ['linux-x64-portable-offline-full.tar.gz', '完整离线便携版'],
  },
  {
    os: 'Linux',
    cpu: 'ARM64',
    online: [
      ['linux-arm64-deb-online.deb', 'DEB 安装包'],
      ['linux-arm64-portable-online.tar.gz', 'Portable 便携版'],
    ],
    offline: ['linux-arm64-portable-offline-full.tar.gz', '完整离线便携版'],
  },
]

const prefix = `starline-dsh-desktop-v${version}-`
const expectedNames = rows.flatMap((row) => [
  ...row.online.map(([suffix]) => `${prefix}${suffix}`),
  `${prefix}${row.offline[0]}`,
])
const directoryEntries = await readdir(assetDirectory, { withFileTypes: true })
const primaryNames = directoryEntries
  .filter((entry) => entry.isFile() && !entry.name.endsWith('.sha256') && entry.name !== 'SHA256SUMS.txt')
  .map((entry) => entry.name)
  .sort()
const expectedSorted = [...expectedNames].sort()
if (JSON.stringify(primaryNames) !== JSON.stringify(expectedSorted)) {
  throw new Error(`Release asset set mismatch.\nExpected: ${expectedSorted.join(', ')}\nActual: ${primaryNames.join(', ')}`)
}

const manifestPath = path.join(assetDirectory, 'SHA256SUMS.txt')
const combinedChecksums = parseChecksumManifest(await readFile(manifestPath, 'utf8'), 'SHA256SUMS.txt')
const assets = new Map()
for (const name of expectedNames) {
  const filePath = path.join(assetDirectory, name)
  const fileStat = await stat(filePath)
  if (!fileStat.isFile() || fileStat.size === 0) {
    throw new Error(`Release asset is missing or empty: ${name}`)
  }

  const sidecarName = `${name}.sha256`
  const sidecarChecksums = parseChecksumManifest(
    await readFile(path.join(assetDirectory, sidecarName), 'utf8'),
    sidecarName,
  )
  const sidecarHash = sidecarChecksums.get(name)
  const combinedHash = combinedChecksums.get(name)
  if (!sidecarHash || !combinedHash || sidecarHash !== combinedHash) {
    throw new Error(`Checksum mismatch for release asset: ${name}`)
  }
  assets.set(name, { name, size: fileStat.size })
}
if (combinedChecksums.size !== expectedNames.length) {
  throw new Error('SHA256SUMS.txt contains an unexpected number of assets.')
}

const changelog = extractChangelogSection(await readFile(changelogPath, 'utf8'), version)
const requirements = parseReleaseRequirements(await readFile(requirementsPath, 'utf8'))
const bundledNodeVersion = (await readFile(nodeVersionPath, 'utf8')).trim()
if (!/^\d+\.\d+\.\d+$/.test(bundledNodeVersion)) {
  throw new Error(`Invalid bundled Node.js version: ${bundledNodeVersion}`)
}
const tableRows = rows.map((row) => {
  const onlineLinks = row.online.map(([suffix, label]) => (
    linkFor(repository, tag, assets.get(`${prefix}${suffix}`), label)
  )).join('<br>')
  const [offlineSuffix, offlineLabel] = row.offline
  const offlineLink = linkFor(repository, tag, assets.get(`${prefix}${offlineSuffix}`), offlineLabel)
  return `| ${row.os} | ${row.cpu} | ${onlineLinks} | ${offlineLink} |`
})

const checksumUrl = assetUrl(repository, tag, 'SHA256SUMS.txt')
const requirementRows = requirements.platforms.map((platform) => (
  `| ${platform.platform} | ${platform.supportedBaseline} | ${platform.requiredRuntime} |`
))
const notes = [
  `本页下载文件已按 **系统、CPU、安装形态、联网模式** 命名，文件大小来自本次实际构建。`,
  '',
  '## 快速下载',
  '',
  '| 系统 | CPU | 在线小包 | 完整离线包 |',
  '| --- | --- | --- | --- |',
  ...tableRows,
  '',
  '## 本版本固定系统要求',
  '',
  `以下要求固定在本 tag 的 \`release-requirements.json\`（修订 ${requirements.revision}）中，后续版本不会反向改写本页。`,
  '',
  '| 平台 | 本版本支持基线 | 必要组件 |',
  '| --- | --- | --- |',
  ...requirementRows,
  '',
  `- 在线小包：${requirements.onlineRuntime}。`,
  `- \`offline-full\`：内置 Node.js ${bundledNodeVersion}；${requirements.offlineRuntime}。`,
  `- **旧系统边界：** ${requirements.unsupported}。`,
  '- Linux 内核号只是 Ubuntu 构建/支持基线；glibc 与 GTK/WebKitGTK 等用户态 ABI 同样是硬性条件，升级旧发行版内核不会自动获得支持。',
  '',
  '## 如何选择',
  '',
  '- 普通 Windows 10/11 电脑一般选择 **Windows x64 · Setup 安装版**；只有 Windows on ARM 设备才选择 ARM64。',
  '- Intel Mac 选择 `macos-intel-x64`，M1/M2/M3/M4 等 Apple Silicon Mac 选择 `macos-apple-silicon-arm64`。',
  '- `online` 是体积较小的普通包，需要系统 Node.js 22.19+ 或 24+，首次运行可能访问 npm registry。',
  '- `offline-full` 内置固定 Node.js 与 DSH 生产依赖，体积明显更大且包含大量文件；它不访问 npm，但模型服务、远程 MCP 和 Web 工具仍可能需要网络。',
  '',
  '## 本版变更',
  '',
  changelog,
  '',
  '## 校验与平台提示',
  '',
  `- 每个资产都有独立 \`.sha256\`，也可下载 [SHA256SUMS.txt](${checksumUrl}) 统一校验。`,
  '- 当前 Windows 与 macOS 产物尚未进行商业代码签名或 Apple 公证，系统可能显示未知发布者或 Gatekeeper 提示。',
  '- Linux DEB 与便携包均只支持上述 Ubuntu 24.04 基线；DEB 会声明 glibc、GTK3 与 WebKitGTK 4.1 依赖。',
  '- 本项目是独立社区项目，与 DeepSeek 官方无隶属、背书或商业关系。',
].join('\n')

await writeFile(outputPath, `${notes}\n`, 'utf8')
console.log(`Release notes written to ${outputPath}`)
