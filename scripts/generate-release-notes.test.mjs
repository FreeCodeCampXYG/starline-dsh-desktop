import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url))
const projectRoot = path.dirname(scriptDirectory)
const generatorPath = path.join(scriptDirectory, 'generate-release-notes.mjs')

test('generates a grouped release page from the exact changelog version', async (context) => {
  const root = await mkdtemp(path.join(tmpdir(), 'starline-release-notes-'))
  context.after(async () => {
    await rm(root, { recursive: true, force: true })
  })

  const version = '9.8.7'
  const prefix = `starline-dsh-desktop-v${version}-`
  const assetDirectory = path.join(root, 'release')
  await mkdir(assetDirectory)
  const suffixes = [
    'windows-x64-setup-online.exe',
    'windows-x64-portable-online.zip',
    'windows-x64-portable-offline-full.zip',
    'windows-arm64-setup-online.exe',
    'windows-arm64-portable-online.zip',
    'windows-arm64-portable-offline-full.zip',
    'macos-intel-x64-app-online.zip',
    'macos-intel-x64-app-offline-full.zip',
    'macos-apple-silicon-arm64-app-online.zip',
    'macos-apple-silicon-arm64-app-offline-full.zip',
    'linux-x64-portable-online.tar.gz',
    'linux-x64-portable-offline-full.tar.gz',
    'linux-arm64-portable-online.tar.gz',
    'linux-arm64-portable-offline-full.tar.gz',
  ]

  const manifest = []
  for (const suffix of suffixes) {
    const name = `${prefix}${suffix}`
    const contents = Buffer.from(`mock package: ${name}`)
    const hash = createHash('sha256').update(contents).digest('hex')
    const checksumLine = `${hash}  ${name}`
    await writeFile(path.join(assetDirectory, name), contents)
    await writeFile(path.join(assetDirectory, `${name}.sha256`), `${checksumLine}\n`, 'utf8')
    manifest.push(checksumLine)
  }
  await writeFile(path.join(assetDirectory, 'SHA256SUMS.txt'), `${manifest.join('\n')}\n`, 'utf8')

  const changelogPath = path.join(root, 'CHANGELOG.md')
  await writeFile(changelogPath, [
    '# 变更日志',
    '',
    '## [Unreleased]',
    '',
    '- 后续计划。',
    '',
    `## [${version}] - 2026-08-14`,
    '',
    '### 修复',
    '',
    '- 这是本次应显示的修改日志。',
    '',
    '## [9.8.6] - 2026-08-13',
    '',
    '- 这段旧日志不应显示。',
  ].join('\n'), 'utf8')

  const outputPath = path.join(root, 'release-notes.md')
  const result = spawnSync(process.execPath, [
    generatorPath,
    '--version', version,
    '--tag', `v${version}`,
    '--repository', 'FreeCodeCampXYG/starline-dsh-desktop',
    '--asset-dir', assetDirectory,
    '--output', outputPath,
    '--changelog', changelogPath,
  ], {
    cwd: projectRoot,
    encoding: 'utf8',
  })

  assert.equal(result.status, 0, result.stderr)
  const notes = await readFile(outputPath, 'utf8')
  assert.match(notes, /\| Windows \| x64（Intel \/ AMD） \|/)
  assert.match(notes, /windows-x64-setup-online\.exe/)
  assert.match(notes, /macos-apple-silicon-arm64-app-offline-full\.zip/)
  assert.match(notes, /这是本次应显示的修改日志/)
  assert.doesNotMatch(notes, /这段旧日志不应显示/)
  assert.match(notes, /1 KiB/)
})
