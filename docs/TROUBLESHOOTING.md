# 故障排查

> 先查看 [已知问题与平台支持边界](KNOWN_ISSUES.md)。v0.3.2 已通过六平台原生 CI、PTY/原生依赖功能测试和最终归档复测，但这仍不能代替代表性设备上的安装、首次启动和真实工作流验证。

## 日志位置

在应用中选择 **桌面工具 → 打开日志目录**。默认位置由系统用户缓存目录决定：

- Windows：通常为 `%LOCALAPPDATA%\starline-dsh-desktop\logs`；
- macOS：通常位于用户 Library/Caches；
- Linux：通常位于 `~/.cache/starline-dsh-desktop/logs`。

应用最多保留最近 10 份日志。分享日志前删除 API Key、Token、Cookie、用户名和私人路径。

## Windows 自定义安装目录

Setup 向导的“选择安装位置”页面可以改目录。当前用户安装默认位于 `%LOCALAPPDATA%\Programs\Starline DSH Desktop`，也可以选择其他当前用户可写位置。安装器会记录所选 `InstallLocation`；关闭托盘中的旧程序后，再运行同一架构的新版 Setup 会继续使用该目录并覆盖程序文件，配置、日志和 DSH 工作区不在安装目录中。

中文和空格路径受支持。若自定义路径启动失败，请检查：

1. 目录是否允许当前用户写入；`Program Files` 通常需要管理员权限，不适合当前用户安装包；
2. 安全软件是否阻止该目录中的未签名程序；
3. 离线便携版的 `offline-runtime/` 是否仍与 EXE 相邻；
4. 是否只移动了 EXE，而没有移动完整离线目录；
5. 日志中的 Node/DSH 路径是否完整，分享前先脱敏。

配置和日志保存在系统用户目录，不依赖安装位置；更换安装目录不会自动搬动 DSH 自己的工作区数据。

当前自动化证据覆盖本地 NTFS 中文和空格路径；UNC 网络共享、映射盘断连和超长路径尚未作为支持目标验证。

## 找不到 Node.js

终端执行：

```bash
node --version
npx --version
```

需要 Node `22.19+` 或 `24+`。Node 23 不在 DSH 支持范围内。图形应用继承的 PATH 可能与终端不同；macOS/Linux 从桌面启动时尤其需要确认 Node 安装在系统可见位置。

如果电脑不能安装 Node 或访问 npm，可以下载与操作系统及架构一致的 `offline-full` 包。离线包正常启动时，桌面顶部会显示“离线运行时”。v0.5.0 已在原生 runner 和最终归档中复测 PTY 与关键原生依赖；未完成设备验证的平台仍应按 [平台矩阵](KNOWN_ISSUES.md#v050-当前发布证据) 谨慎使用。

## 离线运行时损坏或版本不一致

不要只复制桌面可执行文件。Windows/Linux 需要让 `offline-runtime/` 与可执行文件保持在同一目录；macOS 运行时位于应用包的 `Contents/Resources/offline-runtime`。

出现缺少 `dsh-version.txt`、Node、DSH 入口或版本不匹配时，请重新解压完整离线资产。宿主会明确失败，不会静默回退到 npm 下载。设置 `DSH_DESKTOP_DSH_VERSION` 覆盖版本时，也必须与离线包版本完全一致。

## npm 下载失败

1. 检查 npm registry：`npm config get registry`；
2. 在终端直接运行固定版本 DSH；
3. 若使用 VPN 本机端口，在桌面工具中选择自定义 HTTP 代理；
4. 不要把 SOCKS5 地址填入当前 HTTP(S) 代理输入框；
5. 如果选择“禁用代理”，Starline 会优先使用 `https://registry.npmmirror.com`；继承模式没有系统代理时也采用该镜像；
6. 查看 npm 自己的调试日志路径，Starline 日志会打印该位置。

在线包的单次 npm 网络等待为 10 秒，最多重试 1 次；在线 DSH 本地页面整体就绪等待默认上限为 90 秒，可在“代理与启动设置”中调整为 30–600 秒。自定义代理（例如 `127.0.0.1:1080`）不可达时会明确失败，不会偷偷改走直连或镜像；可切换到“禁用代理”后重试。这里的超时只控制 npm 运行时准备和本地 DSH 页面就绪，不控制 DeepSeek Harness 内部远程模型/API 请求；后者由 DSH 自身处理。

```bash
npx --yes --package=@deepseek-ai/dsh@0.1.0-rc.7 dsh web
```

## DSH 启动后页面不就绪

宿主要求页面返回 HTTP 200 且包含 DSH 标题。常见原因：

- npm 进程仍在首次安装依赖；
- DSH 上游版本改变了启动行为；
- 安全软件阻止 Node 监听 loopback；
- 旧版 v0.2.4 可能在临时 listener 释放后发生端口竞争；v0.2.5 起已改为 DSH `--port 0`；
- DSH 进程提前退出。

请保留完整启动日志并重试一次。如果浏览器直接运行 DSH 正常而桌面端失败，向本仓库报告。

## 代理改完没有生效

保存设置会重启 DSH。确认日志时间已经变化，并检查：

- 模式是否为“自定义代理”；
- 地址是否是 HTTP(S)，例如 `http://127.0.0.1:10808`；
- 代理客户端是否允许来自 Node.js 的连接；
- npm registry 与模型服务是否都能通过该代理访问。

`offline-full` 不访问 npm registry，但代理设置仍会传递给 DSH，用于模型 provider、远程 MCP、Web 工具等网络功能。

## 自动检查或更新 DSH 失败

1. 启动后的自动检查和手动刷新显示 `latest` 与 `next`；无代理时优先查询国内镜像，继承/自定义代理沿用用户选择。检查失败不会阻塞当前 DSH 启动。
2. 点击应用通道后，后端会再次核对 registry dist-tag，再保存精确版本并重启；`next` 属于预览通道。如果另一个桌面实例同时修改了设置，会返回配置冲突而不是覆盖。
3. 在线包更新后如果下载不完整、DSH 提前退出或页面指纹校验失败，应用会自动恢复更新前的版本和配置并重启旧运行时；只有旧版本也无法启动时才需要手动排查。
4. `offline-full` 会显示新版本但拒绝原地安装；请下载包含该 DSH 版本的新离线资产。当前离线资产是便携归档，应关闭旧程序后解压到新目录验证；手工覆盖运行中的 `node_modules` 会破坏完整性与原生依赖门禁。
5. 进度条显示真实启动阶段而非 npm 下载字节。停在 65% 表示 npx 子进程已启动、仍在解析或准备依赖；默认最多等待 90 秒（可在设置中改为 30–600 秒），查看 Starline 日志中的 registry、超时和 npm 调试日志判断是否仍有缓存命中或明确错误。
6. 使用 `DSH_DESKTOP_DSH_VERSION` 时，环境变量优先于界面设置；移除覆盖并重新启动桌面端后，更新按钮才能改变实际版本。
7. v0.5.1 及更早版本的在线命令强制 `--prefer-offline`，可能让刚发布的精确版本命中陈旧 npm packument 并误报 `ETARGET`；v0.5.2 起改用 npm 默认元数据校验和内容缓存策略。

## Windows 黑框闪烁

当前版本使用 `CREATE_NO_WINDOW` 和隐藏窗口标志运行宿主直接创建的 Node 检查、DSH 与进程树回收。DSH Windows ACL 沙箱可能在更深层重新创建 PowerShell/控制台进程，这不受外壳启动标志控制；同类报告见 [#15](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/15)。如果仍出现：

1. 确认使用最新 Release；
2. 说明是启动、重启还是关闭时出现；
3. 提供 Windows 版本和安装方式；
4. 区分是在 DSH 启动/关闭时闪一次，还是每次 Agent 执行 PowerShell 都闪；
5. 检查是否由杀毒软件、终端配置、Node shim 或 DSH 沙箱子进程创建。

## Windows PowerShell 返回 `0xC0000142`

十进制退出码 `3221225794` 即 `0xC0000142`（`STATUS_DLL_INIT_FAILED`）。同类 rc.6 报告表明，某些 PowerShell 7 自包含/便携安装在 DSH Windows ACL 受限令牌下无法初始化，而 `danger-full-access` 能运行；参见 [上游同类 Issue #84](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/84)。这发生在 DSH 沙箱内部，不等于 Starline 启动 Node 失败。

用户可以在官方 DSH 权限选择器中显式切换到 `danger-full-access`：它允许 PowerShell 使用当前 Windows 用户本来拥有的广泛文件/命令权限，适合确实需要大范围操作或诊断沙箱兼容性时使用，但会失去工作区隔离。宿主不会自动回退到该模式。它也不是“以管理员身份运行”，不会自动获得 UAC 管理员令牌。报告问题时请注明 Windows 版本、PowerShell 版本与实际路径、DSH 权限模式、十进制/十六进制退出码和脱敏日志。

## 复制、右键菜单或文件夹选择异常

- v0.3.3 在正式构建中启用 WebView 默认右键菜单，可用手工选择、复制和粘贴作为回退；v0.3.2 及更早资产不包含这一改动。
- iframe 已声明 `clipboard-read` / `clipboard-write`，但 DSH 自身复制按钮的降级逻辑仍属于上游页面。若按钮无反馈，先测试右键复制、键盘快捷键以及“在浏览器中打开”，再分别报告结果。
- Starline 不伪装 Electron Desktop，也不注入 DSH 原生目录选择器；工作区选择保留 Web 版浏览路径。不要设置 `SSH_CONNECTION=1` 伪装 SSH 环境作为长期修复。如果浏览器 `dsh web` 正常而内嵌页面失败，请附平台、WebView 版本和脱敏日志。

## MCP 或插件配置后启动很久

宿主最多等待五分钟，并在 DSH 提前退出或超时后显示日志入口。MCP 启动阻塞、profile 中重复插入已经由 bundle 自动注册的插件、无效 `cordis.patch.yml` 都可能让 DSH 在 readiness 前失败；同类案例见 [#45](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/45) 和 [#8](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/8)。先在浏览器直接运行同版本 `dsh web` 并检查同一份 profile；Starline 不自动删除或禁用用户插件。

## Agent 执行 `dsh plugin` 提示缺少 `--profile`

这说明 PowerShell 已经成功启动 DSH，失败点是官方 CLI 参数校验，不是 Shell 权限。官方语法要求 profile 写在 `plugin` 子命令后：

```powershell
dsh plugin --profile web --help
dsh plugin --profile web add <package>
```

v0.5.1 起会在 Starline 启动的 DSH 进程环境中提供临时兼容入口：只有 `dsh plugin` 完全没写 `--profile` 时才补为当前 `web` profile；`--profile tui` 等显式选择保持不变，其他 DSH 命令也不会被改写。该入口不修改系统 PATH 或全局 npm 文件，也不会绕过 DSH/PowerShell 权限；插件管理仍依赖官方要求的 pnpm、网络或本地缓存。v0.5.0 及更早版本可先按上面的完整语法执行。

## macOS 无法打开

未签名/未 notarize 的构建可能被 Gatekeeper 阻止。开发者可在本机重新构建。不要建议普通用户永久关闭 Gatekeeper；正式分发应完成签名和公证。

仅在确认文件来自本仓库 Release 并核对 SHA-256 后，使用 Finder 的“打开”或系统“隐私与安全性”确认。当前没有 DMG，也没有签名安装器。

## macOS 离线包的终端或 Shell 工具失败

v0.2.4 ARM64 `offline-full` 正式 ZIP 中，`offline-runtime/node_modules/node-pty/prebuilds/darwin-arm64/spawn-helper` 的权限是 `0644`，不能作为可执行文件启动；Intel 包使用相同安装逻辑。v0.3.2 已在两个 macOS 原生 runner 和最终 ZIP 中检查 helper 可执行位并真实启动 PTY。

如果 v0.3.2+ 仍出现终端/PTY 工具打不开或权限被拒绝，请提供完整资产名和 SHA-256；不要沿用 v0.2.4 的手工改权限结论，也不要全局关闭系统安全策略。

## Linux 缺少共享库

检查：

```bash
ldd ./starline-dsh-desktop | grep 'not found'
```

安装 GTK3 和 WebKitGTK 4.1 的运行库。不同发行版包名不同，Issue 中请注明发行版、版本和架构。

当前 Linux 产物在 Ubuntu 24.04 原生 runner 上动态链接构建，仅支持 Ubuntu Desktop 24.04 LTS。v0.5.0 增加 x64/ARM64 在线 DEB，并继续提供便携 TAR.GZ；没有 AppImage、RPM 或软件仓库更新通道。

DEB 应使用 apt 安装，以便解析系统库依赖：

```bash
sudo apt install ./starline-dsh-desktop-v<版本>-linux-x64-deb-online.deb
```

DEB 不内置 Node。安装后若提示找不到 Node，请安装 Node.js `22.19+` 或 `24+`，并确认桌面会话能够看到对应 PATH；Node 由 nvm/Volta 等用户级工具管理时，dpkg 无法代替应用做版本检查。

## Linux 提示找不到 `pty.node` 或加载 `node-pty` 失败

v0.2.4 x64 `offline-full` 正式 TAR.GZ 中没有 Linux 可加载的 `node-pty` 原生绑定；ARM64 使用相同打包逻辑。v0.3.2 已在 Ubuntu 24.04 x64/ARM64 原生 runner 和最终 TAR.GZ 中检查并加载 `pty.node`，再真实启动 PTY。

若机器可以联网，改用普通包，并先准备原生模块构建工具：

```bash
sudo apt-get install python3 make g++
```

若 v0.3.2+ 在目标发行版仍失败，优先检查 glibc、CPU 架构和动态库差异；Ubuntu 24.04 runner 成功不能证明所有发行版兼容。

## CI 显示成功，但功能仍失败

v0.3.2 Release 已增加真实 PTY/ripgrep 执行、Sharp/Koffi 加载和最终归档检查；v0.3.3 又把 Sharp/Koffi 提升为实际图像转换、动态库加载和本机函数调用。CI 仍不能证明 GUI 权限、WebView、安装器、安全软件和所有发行版表现。提交 Issue 时请同时说明：

1. 下载的完整资产名和 SHA-256；
2. 普通包还是 `offline-full`；
3. 系统版本、CPU 架构和 Node 版本；
4. 是启动页面失败，还是某个具体工具调用失败；
5. 脱敏后的 Starline 日志及 DSH 错误信息。

## 应该向哪个仓库报告

| 问题 | 仓库 |
| --- | --- |
| Node 检测、代理、窗口、日志、安装包、进程残留 | Starline DSH Desktop |
| Agent 回复、模型 provider、会话、轨迹、插件、官方 UI | deepseek-harness |
| Wails 本身的 WebView/原生窗口最小复现 | wails |

提交前尽量在浏览器直接运行 `dsh web`，这一步能快速判断问题属于宿主还是上游。
