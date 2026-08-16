#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
	echo "Usage: $0 <version> <amd64|arm64> <binary> <output.deb>" >&2
	exit 2
fi

version="$1"
architecture="$2"
binary="$3"
output="$4"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
	echo "Invalid package version: $version" >&2
	exit 2
fi

case "$architecture" in
	amd64)
		expected_machine='x86-64'
		;;
	arm64)
		expected_machine='ARM aarch64'
		;;
	*)
		echo "Unsupported Debian architecture: $architecture" >&2
		exit 2
		;;
esac

if [[ ! -x "$binary" ]]; then
	echo "Linux application binary is missing or not executable: $binary" >&2
	exit 1
fi
if [[ "$output" != *.deb ]]; then
	echo "DEB output must end in .deb: $output" >&2
	exit 2
fi

script_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd "$script_directory/.." && pwd)"
requirements_path="$project_root/release-requirements.json"

for required_file in "$requirements_path" "$project_root/build/appicon.png" "$project_root/LICENSE" "$project_root/NOTICE.md" "$project_root/AUTHORS.md" "$project_root/CHANGELOG.md"; do
	if [[ ! -f "$required_file" ]]; then
		echo "Required package input is missing: $required_file" >&2
		exit 1
	fi
done

readarray -t package_fields < <(node -e '
const fs = require("node:fs")
const requirements = JSON.parse(fs.readFileSync(process.argv[1], "utf8"))
const architecture = process.argv[2]
const deb = requirements.linuxDeb
if (requirements.schemaVersion !== 1 || !deb?.architectures?.includes(architecture)) {
  throw new Error(`release-requirements.json does not permit ${architecture}`)
}
for (const field of ["depends", "suggests"]) {
  if (!Array.isArray(deb[field]) || deb[field].length === 0 || deb[field].some((value) => typeof value !== "string" || !value.trim())) {
    throw new Error(`release-requirements.json has an invalid linuxDeb.${field}`)
  }
}
console.log(deb.depends.join(", "))
console.log(deb.suggests.join(", "))
' "$requirements_path" "$architecture")

if [[ ${#package_fields[@]} -ne 2 ]]; then
	echo "Failed to resolve Linux package dependencies." >&2
	exit 1
fi
depends="${package_fields[0]}"
suggests="${package_fields[1]}"

output_directory="$(dirname "$output")"
mkdir -p "$output_directory"
output_directory="$(cd "$output_directory" && pwd)"
output="$output_directory/$(basename "$output")"

temporary_root="$(mktemp -d "${RUNNER_TEMP:-/tmp}/starline-dsh-deb.XXXXXX")"
trap 'rm -rf -- "$temporary_root"' EXIT
stage="$temporary_root/stage"
verify_root="$temporary_root/verify"
document_directory="$stage/usr/share/doc/starline-dsh-desktop"

install -Dm755 "$binary" "$stage/usr/bin/starline-dsh-desktop"
install -Dm644 "$project_root/build/appicon.png" "$stage/usr/share/pixmaps/starline-dsh-desktop.png"
install -Dm644 "$project_root/LICENSE" "$document_directory/copyright"
install -Dm644 "$project_root/NOTICE.md" "$document_directory/NOTICE.md"
install -Dm644 "$project_root/AUTHORS.md" "$document_directory/AUTHORS.md"
gzip -9n -c "$project_root/CHANGELOG.md" > "$document_directory/changelog.gz"

install -d "$stage/usr/share/applications" "$stage/DEBIAN"
cat > "$stage/usr/share/applications/starline-dsh-desktop.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Starline DSH Desktop
Comment=Desktop host for DeepSeek Harness
Exec=/usr/bin/starline-dsh-desktop
TryExec=/usr/bin/starline-dsh-desktop
Icon=starline-dsh-desktop
Terminal=false
Categories=Development;Utility;
StartupNotify=true
DESKTOP

deb_version="${version//-/\~}"
installed_size="$(du -sk "$stage/usr" | cut -f1)"
cat > "$stage/DEBIAN/control" <<CONTROL
Package: starline-dsh-desktop
Version: $deb_version
Section: devel
Priority: optional
Architecture: $architecture
Installed-Size: $installed_size
Maintainer: starline <1308947723@qq.com>
Homepage: https://github.com/FreeCodeCampXYG/starline-dsh-desktop
Depends: $depends
Suggests: $suggests
Description: Desktop host for DeepSeek Harness
 Starts the pinned DeepSeek Harness Web service and displays it in a Wails WebView.
 This online package requires a compatible user-installed Node.js and npm/npx.
CONTROL

dpkg-deb --root-owner-group --build "$stage" "$output"

test "$(dpkg-deb --field "$output" Package)" = 'starline-dsh-desktop'
test "$(dpkg-deb --field "$output" Version)" = "$deb_version"
test "$(dpkg-deb --field "$output" Architecture)" = "$architecture"
dpkg-deb --info "$output"
dpkg-deb --contents "$output"
dpkg-deb --extract "$output" "$verify_root"
test -x "$verify_root/usr/bin/starline-dsh-desktop"
test -f "$verify_root/usr/share/applications/starline-dsh-desktop.desktop"
test -f "$verify_root/usr/share/pixmaps/starline-dsh-desktop.png"
test -f "$verify_root/usr/share/doc/starline-dsh-desktop/copyright"

file_output="$(file "$verify_root/usr/bin/starline-dsh-desktop")"
echo "$file_output"
if [[ "$file_output" != *"$expected_machine"* ]]; then
	echo "Unexpected Linux binary architecture for $architecture." >&2
	exit 1
fi

ldd_output="$(ldd "$verify_root/usr/bin/starline-dsh-desktop")"
echo "$ldd_output"
if grep -q 'not found' <<<"$ldd_output"; then
	echo "The packaged Linux application has unresolved shared libraries." >&2
	exit 1
fi

echo "Verified Debian package: $output"
