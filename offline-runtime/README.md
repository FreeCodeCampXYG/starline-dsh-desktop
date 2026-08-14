# Offline runtime payload

This directory defines the pinned DSH production dependency closure used only by `offline-full` release artifacts.

Release jobs install it with the committed lock file on each native runner, use the Node version pinned in `node-version.txt`, copy the matching Node executable and verified Node license into this directory, smoke-test the DSH CLI, and then place the directory next to the desktop executable or inside the macOS app bundle.

Generated files are intentionally ignored:

- `node_modules/`
- `node.exe` on Windows or `node` on macOS/Linux
- `LICENSE-node.txt`
- `dsh-version.txt`

The normal Setup/portable artifacts do not include this payload. Offline-full avoids npm access at startup, but model providers, remote MCP servers, web tools, updates, and other network-backed DSH features may still require network access.
