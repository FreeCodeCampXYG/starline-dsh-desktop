import { spawn } from 'node:child_process'

const commandPath = process.env.STARLINE_DSH_SHIM_COMMAND_PATH
const encodedPrefix = process.env.STARLINE_DSH_SHIM_COMMAND_PREFIX
const defaultProfile = process.env.STARLINE_DSH_ACTIVE_PROFILE

if (!commandPath || !encodedPrefix || !defaultProfile) {
  console.error('Starline DSH command shim is missing its runtime configuration.')
  process.exit(1)
}

let commandPrefix
try {
  commandPrefix = JSON.parse(Buffer.from(encodedPrefix, 'base64').toString('utf8'))
} catch {
  console.error('Starline DSH command shim received an invalid command prefix.')
  process.exit(1)
}
if (!Array.isArray(commandPrefix) || commandPrefix.some((value) => typeof value !== 'string')) {
  console.error('Starline DSH command shim received an invalid command prefix.')
  process.exit(1)
}

const forwardedArgs = process.argv.slice(2)
const hasProfile = forwardedArgs
  .slice(1)
  .some((value) => value === '--profile' || value.startsWith('--profile='))
if (forwardedArgs[0] === 'plugin' && !hasProfile) {
  forwardedArgs.splice(1, 0, '--profile', defaultProfile)
}

const child = spawn(commandPath, [...commandPrefix, ...forwardedArgs], {
  env: process.env,
  stdio: 'inherit',
  windowsHide: true,
})

child.once('error', (error) => {
  console.error(`Starline DSH command shim could not start DSH: ${error.message}`)
  process.exitCode = 127
})
child.once('exit', (code) => {
  process.exitCode = code ?? 1
})
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.once(signal, () => {
    if (!child.killed) child.kill(signal)
  })
}
