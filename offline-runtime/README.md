# Offline runtime payload

This directory defines the pinned DSH production dependency closure used only by `offline-full` release artifacts.

Release jobs install it with the committed lock file and all lifecycle scripts disabled. They then verify the approved package versions, lockfile integrity values, lifecycle commands, and script hashes before rebuilding only `node-pty` and running DSH's reviewed spawn-helper permission repair. Each native runner uses the Node version pinned in `node-version.txt`, performs a real PTY shell spawn, and repeats the functional check after unpacking the final archive.

Generated files are intentionally ignored:

- `node_modules/`
- `node.exe` on Windows or `node` on macOS/Linux
- `LICENSE-node.txt`
- `dsh-version.txt`

The normal Setup/portable artifacts do not include this payload. Offline-full avoids npm access at startup, but model providers, remote MCP servers, web tools, updates, and other network-backed DSH features may still require network access.
