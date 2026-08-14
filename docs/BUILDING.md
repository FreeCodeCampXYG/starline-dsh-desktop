# 开发与跨平台构建

## 工具链

- Go 1.25+
- Node.js 24.19.0（Release 离线运行时固定版本；普通包运行时也接受 Node 22.19+）
- npm
- Wails CLI 2.14.0
- 平台原生 C/C++ 工具链与 WebView 开发包

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
wails doctor
```

## 安装前端依赖

```bash
npm --prefix frontend ci
```

仓库使用 `package-lock.json`。CI、Release 和正式构建都使用 `npm ci`，不在构建时更新依赖解析结果。

离线运行时有独立锁文件：

```bash
npm --prefix offline-runtime ci --omit=dev --ignore-scripts --workspaces=false
node offline-runtime/node_modules/@deepseek-ai/dsh/lib/bin.js --version
```

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

Linux 使用 WebKitGTK 4.1 build tag：

```bash
go test -tags webkit2_41 ./...
go vet -tags webkit2_41 ./...
```

## Windows

安装 Go、Node.js、Wails、WebView2 和 NSIS，并确保 `makensis` 在 PATH：

```powershell
.\scripts\build-windows.ps1 -Version 0.2.3
```

额外构建 Windows x64 离线便携包：

```powershell
.\scripts\build-windows.ps1 -Version 0.1.1 -OfflineFull
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
wails build -clean -trimpath -platform windows/amd64 -nsis -installscope user -ldflags "-s -w -H=windowsgui -X main.version=0.2.3"
```

Windows ARM64 必须在 ARM64 Windows 或受支持的原生 runner 构建。仓库 Release 使用 `windows-11-arm`，不把 x64 交叉编译结果当作完整平台验证。

## macOS

安装 Xcode Command Line Tools、Go、Node.js 和 Wails：

```bash
npm --prefix frontend ci
wails build -clean -trimpath -platform darwin/arm64 -ldflags "-s -w -X main.version=0.2.3"
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
wails build -clean -trimpath -platform linux/amd64 -tags webkit2_41 -ldflags "-s -w -X main.version=0.2.3"
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
