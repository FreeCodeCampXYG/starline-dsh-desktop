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

const packageJSON = readJSON(packagePath)
const dshVersion = packageJSON.dependencies?.['@deepseek-ai/dsh']
const lock = readJSON(lockPath)
const lockRoot = lock.packages?.['']?.dependencies?.['@deepseek-ai/dsh']
if (!dshVersion || lockRoot !== dshVersion) {
  throw new Error(`offline DSH lock mismatch: package.json=${dshVersion}, package-lock.json=${lockRoot}`)
}

const localKey = 'node_modules/@deepseek-ai/dsh-subprocess-local'
const localPackage = lock.packages?.[localKey]
if (!localPackage?.version || !localPackage.integrity) {
  throw new Error(`missing lock metadata for ${localKey}`)
}

const helperPath = join(runtimeRoot, 'node_modules', '@deepseek-ai', 'dsh-subprocess-local', 'scripts', 'ensure-spawn-helper.mjs')
const helperHash = createHash('sha256').update(readFileSync(helperPath)).digest('hex')
let verifier = readFileSync(verifierPath, 'utf8')
verifier = replaceOnce(
  verifier,
  /(node_modules\/@deepseek-ai\/dsh-subprocess-local'\s*:\s*\{\s*version:\s*')[^']+('\s*,\s*integrity:\s*')[^']+(')/s,
  (match, prefix, integrityPrefix, suffix) => `${prefix}${localPackage.version}${integrityPrefix}${localPackage.integrity}${suffix}`,
  `${localKey} approval metadata`,
)
verifier = replaceOnce(
  verifier,
  /(\['node_modules\/@deepseek-ai\/dsh-subprocess-local\/scripts\/ensure-spawn-helper\.mjs',\s*')[^']+('])/,
  (match, prefix, suffix) => `${prefix}${helperHash}${suffix}`,
  'ensure-spawn-helper approval hash',
)
writeFileSync(verifierPath, verifier, 'utf8')
console.log(`Synced offline DSH ${dshVersion}; ${localKey} ${localPackage.version}; helper SHA-256 ${helperHash}`)
