#!/usr/bin/env node

import { createHash } from 'node:crypto'
import { spawnSync } from 'node:child_process'
import { existsSync, readFileSync, statSync } from 'node:fs'
import { createRequire } from 'node:module'
import { join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

const APPROVED_PACKAGES = {
  'node_modules/node-pty': {
    version: '1.1.0',
    integrity: 'sha512-20JqtutY6JPXTUnL0ij1uad7Qe1baT46lyolh2sSENDd4sTzKZ4nmAFkeAARDKwmlLjPx6XKRlwRUxwjOy+lUg==',
    scripts: {
      install: 'node scripts/prebuild.js || node-gyp rebuild',
      postinstall: 'node scripts/post-install.js',
    },
  },
  'node_modules/@deepseek-ai/dsh-subprocess-local': {
    version: '0.1.0-rc.6',
    integrity: 'sha512-D2daTRaprE25ti1Ra69eURPwLULpnRyUHqAGtcMHddcjNIsZ4yR0DGinYOsgoRymzu3t02qLxyoio05atSViRw==',
    scripts: {
      postinstall: 'node scripts/ensure-spawn-helper.mjs',
    },
  },
}

const APPROVED_SCRIPT_HASHES = new Map([
  ['node_modules/node-pty/scripts/prebuild.js', '7e604b10f7769d7dc95947d3481c00513b9e3bb6c561d5264506350a16a0381a'],
  ['node_modules/node-pty/scripts/post-install.js', 'b15bb24cd3388b05392eeb200aa805e2827356818d111b93847e1b0f6a6ada46'],
  ['node_modules/@deepseek-ai/dsh-subprocess-local/scripts/ensure-spawn-helper.mjs', 'ca5509febf1e6ec1356df121835ebe5ed2f9cace4bdc2ba6d83d41c7e45e0f1b'],
])

function fail(message) {
  throw new Error(message)
}

function readJSON(path) {
  return JSON.parse(readFileSync(path, 'utf8'))
}

function assertEqual(actual, expected, label) {
  if (actual !== expected) {
    fail(`${label} mismatch: expected ${JSON.stringify(expected)}, received ${JSON.stringify(actual)}`)
  }
}

function verifyApprovedMetadata(runtimeRoot) {
  const lock = readJSON(join(runtimeRoot, 'package-lock.json'))
  for (const [lockKey, approved] of Object.entries(APPROVED_PACKAGES)) {
    const locked = lock.packages?.[lockKey]
    if (!locked) {
      fail(`Approved package is absent from package-lock.json: ${lockKey}`)
    }
    assertEqual(locked.version, approved.version, `${lockKey} locked version`)
    assertEqual(locked.integrity, approved.integrity, `${lockKey} locked integrity`)

    const packageRoot = join(runtimeRoot, lockKey)
    const packageJSON = readJSON(join(packageRoot, 'package.json'))
    assertEqual(packageJSON.version, approved.version, `${lockKey} installed version`)
    for (const [scriptName, command] of Object.entries(approved.scripts)) {
      assertEqual(packageJSON.scripts?.[scriptName], command, `${lockKey} ${scriptName} script`)
    }
  }

  for (const [relativePath, expectedHash] of APPROVED_SCRIPT_HASHES) {
    const scriptPath = join(runtimeRoot, ...relativePath.split('/'))
    const actualHash = createHash('sha256').update(readFileSync(scriptPath)).digest('hex')
    assertEqual(actualHash, expectedHash, `${relativePath} SHA-256`)
  }
}

function requireFile(path, label) {
  if (!existsSync(path) || !statSync(path).isFile()) {
    fail(`${label} is missing: ${path}`)
  }
}

function verifyNativePayload(runtimeRoot) {
  const nodePtyRoot = join(runtimeRoot, 'node_modules', 'node-pty')
  if (process.platform === 'win32') {
    requireFile(join(nodePtyRoot, 'prebuilds', `win32-${process.arch}`, 'pty.node'), 'Windows node-pty binding')
    return
  }
  if (process.platform === 'darwin') {
    requireFile(join(nodePtyRoot, 'prebuilds', `darwin-${process.arch}`, 'pty.node'), 'macOS node-pty binding')
    const helper = join(nodePtyRoot, 'prebuilds', `darwin-${process.arch}`, 'spawn-helper')
    requireFile(helper, 'macOS node-pty spawn-helper')
    if ((statSync(helper).mode & 0o111) === 0) {
      fail(`macOS node-pty spawn-helper is not executable: ${helper}`)
    }
    return
  }
  if (process.platform === 'linux') {
    requireFile(join(nodePtyRoot, 'build', 'Release', 'pty.node'), 'Linux node-pty binding')
    return
  }
  fail(`Unsupported offline runtime platform: ${process.platform}`)
}

async function smokeTestPTY(runtimeRoot) {
  const runtimeRequire = createRequire(join(runtimeRoot, 'package.json'))
  const nodePty = runtimeRequire('node-pty')
  const marker = `STARLINE_DSH_PTY_OK_${process.pid}`
  const shell = process.platform === 'win32' ? (process.env.ComSpec || 'cmd.exe') : '/bin/sh'
  const input = process.platform === 'win32'
    ? `echo ${marker} & exit\r`
    : `printf '${marker}\\n'; exit\n`

  await new Promise((resolvePromise, rejectPromise) => {
    let terminal
    try {
      terminal = nodePty.spawn(shell, [], {
        name: 'xterm-color',
        cols: 80,
        rows: 24,
        cwd: runtimeRoot,
        env: process.env,
      })
    } catch (error) {
      rejectPromise(error)
      return
    }

    let output = ''
    let settled = false
    const finish = (error) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      if (error) rejectPromise(error)
      else resolvePromise()
    }
    const timer = setTimeout(() => {
      try {
        terminal.kill()
      } catch {
        // The verifier will fail immediately below even if cleanup is already complete.
      }
      finish(new Error(`node-pty shell smoke test timed out; output=${JSON.stringify(output)}`))
    }, 15_000)

    terminal.onData((data) => {
      output += data
    })
    terminal.onExit(({ exitCode, signal }) => {
      if (exitCode !== 0 || !output.includes(marker)) {
        finish(new Error(`node-pty shell smoke test failed: exit=${exitCode}, signal=${signal}, output=${JSON.stringify(output)}`))
        return
      }
      finish()
    })
    terminal.write(input)
  })
}

async function smokeTestNativeTools(runtimeRoot) {
  const runtimeRequire = createRequire(join(runtimeRoot, 'package.json'))
  runtimeRequire('sharp')
  runtimeRequire('koffi')

  const ripgrepEntry = runtimeRequire.resolve('@vscode/ripgrep')
  const { rgPath } = await import(pathToFileURL(ripgrepEntry).href)
  requireFile(rgPath, 'ripgrep executable')
  const result = spawnSync(rgPath, ['--version'], {
    encoding: 'utf8',
    timeout: 10_000,
    windowsHide: true,
  })
  if (result.error || result.status !== 0) {
    fail(`ripgrep smoke test failed: ${result.error || result.stderr || `exit ${result.status}`}`)
  }
  console.log(`Verified sharp, koffi, and ripgrep: ${process.platform}/${process.arch}`)
}

async function main() {
  const [mode, runtimeArgument] = process.argv.slice(2)
  if (!['preflight', 'verify'].includes(mode) || !runtimeArgument) {
    fail('Usage: verify-offline-runtime.mjs <preflight|verify> <offline-runtime-directory>')
  }
  const runtimeRoot = resolve(runtimeArgument)
  verifyApprovedMetadata(runtimeRoot)
  if (mode === 'preflight') {
    console.log(`Approved offline lifecycle scripts: ${process.platform}/${process.arch}`)
    return
  }

  verifyNativePayload(runtimeRoot)
  await smokeTestNativeTools(runtimeRoot)
  await smokeTestPTY(runtimeRoot)
  console.log(`Verified offline node-pty spawn: ${process.platform}/${process.arch}`)
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error instanceof Error ? error.stack : error)
    process.exit(1)
  },
)
