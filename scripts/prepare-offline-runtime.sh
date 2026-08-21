#!/usr/bin/env bash
set -euo pipefail

skip_lock_refresh=0
skip_install=0
dsh_version=""
for argument in "$@"; do
	case "$argument" in
		--skip-lock-refresh) skip_lock_refresh=1 ;;
		--skip-install) skip_install=1 ;;
		-h|--help)
			echo "Usage: $0 [--skip-lock-refresh] [--skip-install] <dsh-version>"
			exit 0
			;;
		*)
			if [ -n "$dsh_version" ]; then
				echo "Usage: $0 [--skip-lock-refresh] [--skip-install] <dsh-version>" >&2
				exit 2
			fi
			dsh_version="$argument"
			;;
	esac
done
if [ -z "$dsh_version" ]; then
	echo "Usage: $0 [--skip-lock-refresh] [--skip-install] <dsh-version>" >&2
	exit 2
fi
script_directory="$(dirname "$0")"
repository_root="$(cd "$script_directory/.." && pwd)"
runtime_root="$repository_root/offline-runtime"
locked_version="$(node -p "require('$runtime_root/package.json').dependencies['@deepseek-ai/dsh']")"
node_version="$(tr -d '[:space:]' <"$runtime_root/node-version.txt")"
verifier="$repository_root/scripts/verify-offline-runtime.mjs"

if [ "$locked_version" != "$dsh_version" ]; then
	echo "offline-runtime/package.json pins DSH $locked_version, expected $dsh_version." >&2
	exit 1
fi

node_source="$(command -v node)"
actual_node_version="$(node --version | sed 's/^v//')"
if [ "$actual_node_version" != "$node_version" ]; then
	echo "Node $actual_node_version is active, expected pinned Node $node_version." >&2
	exit 1
fi
node_directory="$(dirname "$node_source")"
node_root="$(cd "$node_directory/.." && pwd)"

fetch_options=(
	--fetch-retries=2
	--fetch-retry-mintimeout=5000
	--fetch-retry-maxtimeout=30000
	--fetch-timeout=120000
)

# lock 只在单独的 runner 上解析一次；平台任务复用它，避免并发解析造成重复下载。
if [ "$skip_lock_refresh" -eq 0 ]; then
	for attempt in 1 2 3; do
		if npm --prefix "$runtime_root" install --package-lock-only --ignore-scripts --workspaces=false "${fetch_options[@]}"; then
			break
		fi
		if [ "$attempt" -eq 3 ]; then
			echo "Offline package-lock resolution failed after 3 attempts." >&2
			exit 1
		fi
		delay=$((attempt * 15))
		echo "Package registry resolution failed; retrying in ${delay}s (attempt $((attempt + 1))/3)." >&2
		sleep "$delay"
	done
fi
if [ "$skip_install" -eq 0 ]; then
	npm --prefix "$runtime_root" ci --omit=dev --ignore-scripts --workspaces=false "${fetch_options[@]}"
elif [ ! -d "$runtime_root/node_modules" ]; then
	echo "--skip-install requires an existing offline-runtime/node_modules cache." >&2
	exit 1
fi
"$node_source" "$repository_root/scripts/sync-offline-runtime-metadata.mjs" "$runtime_root"
"$node_source" "$verifier" preflight "$runtime_root"

# npm ci keeps every lifecycle script disabled. Only the pinned and hash-checked
# node-pty install/postinstall pair is allowed to run here; it may select a
# reviewed platform prebuild instead of compiling from source.
npm --prefix "$runtime_root" rebuild node-pty --foreground-scripts --ignore-scripts=false --workspaces=false
"$node_source" "$runtime_root/node_modules/@deepseek-ai/dsh-subprocess-local/scripts/ensure-spawn-helper.mjs"

cp "$node_source" "$runtime_root/node"
chmod +x "$runtime_root/node"
license_destination="$runtime_root/LICENSE-node.txt"
expected_license_hash="148eacf7863ef4329224a29398623077200a27194aa075569faf4a0a85566ca5"
if [ -f "$node_root/LICENSE" ]; then
	cp "$node_root/LICENSE" "$license_destination"
elif [ ! -f "$license_destination" ]; then
	license_url="https://raw.githubusercontent.com/nodejs/node/v$node_version/LICENSE"
	curl --fail --location --proto '=https' --tlsv1.2 --connect-timeout 15 --max-time 60 "$license_url" --output "$license_destination"
fi
actual_license_hash="$(shasum -a 256 "$license_destination" | awk '{print $1}')"
if [ "$actual_license_hash" != "$expected_license_hash" ]; then
	echo "Node license SHA-256 mismatch: $actual_license_hash" >&2
	exit 1
fi
printf '%s\n' "$dsh_version" >"$runtime_root/dsh-version.txt"

"$runtime_root/node" "$runtime_root/node_modules/@deepseek-ai/dsh/lib/bin.js" --version
"$runtime_root/node" "$verifier" verify "$runtime_root"
echo "Prepared Unix offline runtime: $(du -sk "$runtime_root" | awk '{print $1}') KiB"
