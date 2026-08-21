#!/usr/bin/env node

import { createHash } from 'node:crypto'
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'

const runtimeRoot = resolve(process.argv[2] ?? 'offline-runtime')
const repositoryRoot = dirname(runtimeRoot)
const packagePath = join(runtimeRoot, 'package.json')
const lockPath = join(runtimeRoot, 'package-lock.json')
const verifierPath = join(repositoryRoot, 'scripts', 'verify-offline-runtime.mjs')

function readJSON(path) {
  return JSON.parse(readFileSync(path, 'utf8'))
}

function replaceOnce(source, pattern, replacement, label) {
  let count = 0
  const updated = source.replace(pattern, (...args) => {
    count += 1
    return replacement(...args)
  })
  if (count !== 1) {
    throw new Error(`${label} expected one match, received ${count}`)
  }
  return updated
}

function sha256(path) {
  return createHash('sha256').update(readFileSync(path)).digest('hex')
}

function syncApprovedPackage(verifier, key, locked) {
  if (!locked?.version || !locked.integrity) {
    throw new Error(`missing lock metadata for ${key}`)
  }
  const escapedKey = key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return replaceOnce(
    verifier,
    new RegExp(`(${escapedKey}'\\s*:\\s*\\{\\s*version:\\s*')[^']+('\\s*,\\s*integrity:\\s*')[^']+(')`, 's'),
    (match, prefix, integrityPrefix, suffix) => `${prefix}${locked.version}${integrityPrefix}${locked.integrity}${suffix}`,
    `${key} approval metadata`,
  )
}

const packageJSON = readJSON(packagePath)
const dshVersion = packageJSON.dependencies?.['@deepseek-ai/dsh']
const lock = readJSON(lockPath)
const lockRoot = lock.packages?.['']?.dependencies?.['@deepseek-ai/dsh']
if (!dshVersion || lockRoot !== dshVersion) {
  throw new Error(`offline DSH lock mismatch: package.json=${dshVersion}, package-lock.json=${lockRoot}`)
}

// 先从 runner 实际安装结果重建白名单，避免新 DSH 闭包沿用旧的 integrity 或脚本哈希。
const approvedPackages = [
  'node_modules/node-pty',
  'node_modules/@deepseek-ai/dsh-subprocess-local',
]
let verifier = readFileSync(verifierPath, 'utf8')
for (const key of approvedPackages) {
  verifier = syncApprovedPackage(verifier, key, lock.packages?.[key])
}

const approvedScripts = [
  'node_modules/node-pty/scripts/prebuild.js',
  'node_modules/node-pty/scripts/post-install.js',
  'node_modules/@deepseek-ai/dsh-subprocess-local/scripts/ensure-spawn-helper.mjs',
]
for (const relativePath of approvedScripts) {
  const hash = sha256(join(runtimeRoot, ...relativePath.split('/')))
  const escapedPath = relativePath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  verifier = replaceOnce(
    verifier,
    new RegExp(`(\\['${escapedPath}',\\s*')[^']+('\\])`),
    (match, prefix, suffix) => `${prefix}${hash}${suffix}`,
    `${relativePath} approval hash`,
  )
}
writeFileSync(verifierPath, verifier, 'utf8')
console.log(`Synced offline DSH ${dshVersion}; approval metadata and ${approvedScripts.length} script hashes from runner install`)
