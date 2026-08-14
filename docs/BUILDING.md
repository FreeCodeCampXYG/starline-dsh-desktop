# 开发与跨平台构建

## 工具链

- Go 1.25+
- Node.js 24.19.0（Release 离线运行时固定版本；普通包运行时也接受 Node 22.19+）
- npm
- Wails CLI 2.14.0
- 平台原生 C/C++ 工具链与 WebView 开发包
- Python 与固定 Pillow 版本（仅重新生成应用图标时需要，不是程序运行依赖）

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
wails doctor
```

## 安装前端依赖

```bash
npm --prefix frontend ci
```

仓库使用 `package-lock.json`。CI、Release 和正式构建都使用 `npm ci`，不在构建时更新依赖解析结果。

离线运行时有独立锁文件。使用平台准备脚本，不要手工只运行 `npm ci --ignore-scripts`：

```powershell
.\scripts\prepare-offline-runtime.ps1 -DSHVersion 0.1.0-rc.6
```

```bash
bash scripts/prepare-offline-runtime.sh 0.1.0-rc.6
```

准备脚本采用显式白名单，而不是开放所有依赖脚本：

1. `npm ci --ignore-scripts` 安装锁定依赖；
2. `verify-offline-runtime.mjs preflight` 核对包版本、lockfile integrity、生命周期命令和已审查脚本 SHA-256；
3. 只重建 `node-pty`，再执行锁定 DSH 提供的 `ensure-spawn-helper.mjs`；
4. 使用包内 Node 真实启动平台 Shell，并核对输出和退出码。

新增或升级白名单包时必须重新审查脚本并显式更新校验值。CLI `--version` 成功只证明 JavaScript 入口可启动，不能替代功能测试。v0.2.4 已发布资产的影响范围见 [已知问题与平台支持边界](KNOWN_ISSUES.md)。

`offline-runtime/node_modules`、Node 可执行文件、Node 许可证副本和 `dsh-version.txt` 都是构建产物，不提交到仓库。

## 本地开发与 DevTools

```bash
wails dev
```

开发模式会启用 WebView 检查器，但默认不弹窗。Windows/Linux 按 `Ctrl+Shift+F12`，macOS 按 `Fn+Command+Shift+F12` 手动打开；就绪后可在检查器中选择对应 frame 查看 DSH 页面。前端开发服务器只服务宿主启动/设置界面。

如果需要调试独立可执行文件，构建 Wails 调试版本：

```powershell
wails build -clean -trimpath -debug -platform windows/amd64 -ldflags "-H=windowsgui -X main.version=dev"
.\build\bin\starline-dsh-desktop.exe
```

macOS 和 Linux 将 `-platform` 分别改为当前机器的 `darwin/amd64`、`darwin/arm64`、`linux/amd64` 或 `linux/arm64`；Linux 继续添加 `-tags webkit2_41`。

项目不设置 `options.Debug.OpenInspectorOnStartup`，避免本地调试程序每次启动都弹出检查器。正式构建和 Release 工作流均不使用 `-debug` 或 `-devtools`，因此 Setup.exe、便携包和离线包不能在安装后临时开启检查器；需要排查时应从同一提交构建调试版本。项目不启用远程调试端口。

## 基础验证

必须先生成 `frontend/dist`，因为 Go 的 `embed` 在编译测试包时要求资源存在：

```bash
npm --prefix frontend run docs:check
npm --prefix frontend run typecheck
npm --prefix frontend run build
go test ./...
go vet ./...
```

上述验证覆盖 Go 宿主和前端，不覆盖离线 Node 依赖的运行时完整性。离线包还必须在目标平台验证：

1. 包内 Node 的平台和架构正确；
2. DSH CLI 与 Web 页面能够启动；
3. `sharp`、`koffi`、`node-pty` 等原生依赖能够从包内路径加载；
4. `node-pty` 实际启动平台 Shell，输出唯一测试标记并以退出码 0 结束；
5. macOS helper 保留可执行位，Linux 归档包含当前架构的 `pty.node`；
6. 测试对象是重新解开的最终 ZIP/TAR.GZ/安装包，而不只是打包前目录。

只检查 HTTP 200、页面标题或 `dsh --version` 属于启动证据，不能替代功能证据。

Linux 使用 WebKitGTK 4.1 build tag：

```bash
go test -tags webkit2_41 ./...
go vet -tags webkit2_41 ./...
```

## 重新生成应用图标

图标设计、权利边界和各文件用途见 [品牌与应用图标](BRANDING.md)。生成器会同时写入 Wails 主 PNG 和 Windows 多尺寸 ICO：

```powershell
python -m pip install -r scripts/requirements-icons.txt
python scripts/generate-app-icons.py
```

生成后至少检查 16/24/32 像素可读性，并执行一次 Windows Wails 构建；仅替换 `appicon.png` 不会覆盖已经存在的 `build/windows/icon.ico`。

## Windows

安装 Go、Node.js、Wails、WebView2 和 NSIS，并确保 `makensis` 在 PATH：

```powershell
.\scripts\build-windows.ps1 -Version 0.2.4
```

额外构建 Windows x64 离线便携包：

```powershell
.\scripts\build-windows.ps1 -Version 0.2.4 -OfflineFull
```

脚本依次执行：

1. 工具检查；
2. `npm ci`、typecheck、前端构建；
3. Go test/vet；
4. Wails 生产构建；
5. 当前用户级 NSIS Setup；
6. 便携 ZIP；
7. SHA-256 文件。

输出位于 `dist/`。脚本会临时把版本写进 Wails 安装包元数据，并在结束或失败时恢复 `wails.json`。

安装器目录行为可用独立测试验证。它会生成一个名称和卸载键都隔离的临时测试产品，在系统临时目录的中文、空格路径中完成安装、再次安装和卸载，不覆盖正式安装记录：

```powershell
.\scripts\test-windows-installer-path.ps1
```

测试同时确认 `InstallLocation` 被记录、第二次安装复用自定义目录，以及卸载清理成功。脚本要求 `build/bin/starline-dsh-desktop.exe`、生成过的 `wails_tools.nsh` 和 `makensis` 已可用。

手动构建：

```powershell
wails build -clean -trimpath -platform windows/amd64 -nsis -installscope user -ldflags "-s -w -H=windowsgui -X main.version=0.2.4"
```

Windows ARM64 必须在 ARM64 Windows 或受支持的原生 runner 构建。仓库 Release 使用 `windows-11-arm`，不把 x64 交叉编译结果当作完整平台验证。

## macOS

安装 Xcode Command Line Tools、Go、Node.js 和 Wails：

```bash
npm --prefix frontend ci
wails build -clean -trimpath -platform darwin/arm64 -ldflags "-s -w -X main.version=0.2.4"
```

Intel 使用 `darwin/amd64`。GitHub Actions 分别使用 `macos-15-intel` 与 `macos-15` 原生 runner。

本地未签名 `.app` 只适合开发测试。公开发布前还需要 Developer ID 签名、hardened runtime 和 notarization；它们不能由普通 CI 开关代替。

## Linux

Ubuntu 24.04 开发依赖：

```bash
sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

构建：

```bash
npm --prefix frontend ci
wails build -clean -trimpath -platform linux/amd64 -tags webkit2_41 -ldflags "-s -w -X main.version=0.2.4"
```

ARM64 使用 `linux/arm64`，GitHub Actions 在 `ubuntu-24.04-arm` 原生 runner 上构建。

当前 Linux 产物是动态链接的 TAR.GZ，不宣称跨所有发行版通用。发布前应至少在 Ubuntu 24.04 上验证，并记录其他发行版反馈。

## GitHub Actions 构建矩阵

| OS | Runner | Wails platform | 产物 |
| --- | --- | --- | --- |
| Windows x64 | `windows-latest` | `windows/amd64` | Setup.exe + ZIP + offline-full ZIP |
| Windows ARM64 | `windows-11-arm` | `windows/arm64` | Setup.exe + ZIP + offline-full ZIP |
| macOS Intel | `macos-15-intel` | `darwin/amd64` | app ZIP + offline-full app ZIP |
| macOS Apple Silicon | `macos-15` | `darwin/arm64` | app ZIP + offline-full app ZIP |
| Linux x64 | `ubuntu-24.04` | `linux/amd64` | TAR.GZ + offline-full TAR.GZ |
| Linux ARM64 | `ubuntu-24.04-arm` | `linux/arm64` | TAR.GZ + offline-full TAR.GZ |

各目标在对应架构的原生 runner 上构建。矩阵 `fail-fast: false`，一个平台失败时其他平台仍给出结果，但 tag Release 必须等待全部目标成功。

Release 脚本在每个原生 runner 上用同一锁文件安装平台适配的可选依赖，并复制该 runner 的 Node 24.19.0。普通包不受体积影响；离线包按平台单独下载，不把六个平台运行时合并在一个文件中。

v0.2.4 的六个平台构建和 Web 启动检查虽为绿色，但当时没有白名单原生准备和 PTY 功能检查，因此 macOS/Linux 离线包不满足当前发布门槛。当前 `main` 已在六个原生 runner 增加真实 `node-pty.spawn()`；修复必须由递增版本交付，不能移动已公开 tag。
