# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 的组织方式，并使用语义化版本号。

## [0.6.6] - 2026-08-20

### 变更

- Windows 主程序 manifest 显式请求管理员权限；启动时可能显示 UAC 确认，Node/DSH 子进程继承该令牌，macOS/Linux 行为不变。

### 已知边界

- 管理员令牌不能绕过火绒等安全软件的信誉、行为或勒索防护策略；最终 EXE 的嵌入 manifest 和 UAC 行为以 Windows CI 与设备验证为准。

## [未发布]

### 变更

## [0.6.20] - 2026-09-02

### 修复

- DSH alpha.3 就绪后改由系统浏览器完成 loopback 认证；Desktop 保留启动、日志、代理和进程控制页，避免 Windows WebView2 显示认证黑屏。

## [0.6.19] - 2026-09-02

### 修复

- DSH alpha.3 就绪后改由 Desktop WebView 顶层导航到一次性认证 URL，避免 `wails://` iframe 跨站嵌入导致 `dsh web authentication required`。

## [0.6.18] - 2026-09-02

### 修复

- 健康检查不再消费 DSH alpha.3 的一次性 Web token，保留给内嵌 WebView 完成会话认证，避免出现 `dsh web authentication required`。

### 变更

- 启动检测到旧 Web Profile 插件与 DSH alpha.3 API 不兼容时弹框说明影响范围，要求先备份 Profile 再移除或升级插件；不会自动删除用户数据。

## [0.6.17] - 2026-09-01

### 修复

- 对齐 `@deepseek-ai/dsh@0.1.2-alpha.3` 的 Web profile 启动契约，宿主改用 `dsh --profile web`。
- 接受 DSH alpha.3 的 `DSH Local Build` 页面标题，并在健康检查中保留一次性启动 URL 换取的会话 cookie，避免服务已启动时被误判为超时。

### 安全

- Release 冒烟失败时保留各平台 DSH 启动日志供诊断，并对一次性 Web 启动 token 做脱敏后再上传。

## [0.6.16] - 2026-09-01

### 修复

- 修复升级到 DSH 0.1.2-alpha.3 后仍传递旧 `dsh web` 子命令导致的六平台离线 Web 启动超时。

### 构建

- 离线 Web 冒烟失败时上传 DSH 启动日志，避免只报告 HTTP 健康检查超时。

- `offline-full` 离线闭包和 Desktop 默认 DSH 版本更新为 `@deepseek-ai/dsh@0.1.2-alpha.3`；锁文件、依赖完整性、原生模块和最终归档由本次 tag 的 GitHub Actions runner 下载并验证。

## [0.6.15] - 2026-09-01

### 变更

- 将 `offline-runtime` 固定版本、启动默认版本和相关文档更新为 `@deepseek-ai/dsh@0.1.2-alpha.3`。
- GitHub Actions 将重新解析离线 lock，并执行六平台原生构建、离线闭包校验和最终归档检查。

- Issue #6：普通包启动先探测官方 npm registry，官方不可达时回退国内镜像；沿用用户代理设置，不再假定用户无法访问 npm。

## [0.6.14] - 2026-09-01

### 修复

- 在线普通包启动先探测官方 npm registry，官方不可达时回退国内镜像；沿用用户代理设置，不再假定用户无法访问 npm。

## [0.6.13] - 2026-08-31

### 变更

- 在线普通包首次通过 npx 准备 DSH 依赖的默认就绪上限从 90 秒调整为 5 分钟；仍可在设置中限制为 30–600 秒。

### 修复

- 在线 npx 在就绪期限内未输出监听地址时，错误信息明确提示检查 npm registry/代理或使用 `offline-full`，不再笼统归因为监听地址故障。

## [0.6.12] - 2026-08-22

### 修复

- 桌面宿主启动 DSH Web 时显式传入 `--no-open`，避免官方 Web 表层把内嵌页面重复交给系统浏览器；仍保留菜单中的手动浏览器回退入口。

## [0.6.11] - 2026-08-22

### 修复

- 离线运行时版本高于用户保存的旧 DSH 覆盖版本时，启动优先使用包内版本，避免界面继续显示旧版本。
- CI 与 Release 只消费已在 GitHub runner 上生成并校验的离线依赖锁，避免每个平台重复解析 registry。

### 构建

- 移除离线 `node_modules` 与 npm 下载缓存，关闭发布包的二次压缩，降低 GitHub runner 的 CPU、内存和缓存占用。

## [0.6.10] - 2026-08-22

### 变更

## [0.6.9] - 2026-08-22

### 变更

## [0.6.8] - 2026-08-22

### 变更
- Desktop 发布流程与构建门禁更新；离线包继续固定 `@deepseek-ai/dsh@0.1.1-rc.2`，本次 tag 由 GitHub Actions 重新解析并验证跨平台依赖。

## [0.6.7] - 2026-08-22

### 变更
- `offline-full` 离线闭包和 Desktop 默认 DSH 版本更新为 `@deepseek-ai/dsh@0.1.1-rc.2`；锁文件、依赖 integrity、原生模块和最终归档由本次 tag 的 GitHub Actions runner 下载并验证。

- Windows 主程序 manifest 显式请求管理员权限，便于需要提升令牌的 DSH 子进程和本地运行时在火绒等安全软件策略下完成受控操作；每次启动可能显示 UAC 确认，macOS/Linux 行为不变。
- 修复 `offline-full` 检查到新 DSH 版本后没有可应用按钮的问题：检测到系统 Node.js/npm 时，确认更新会通过现有代理降级链使用 npx 下载精确版本，不修改包内离线依赖，并支持失败后恢复默认离线版本。

### 已知边界

- 管理员令牌不能绕过火绒的信誉、行为或勒索防护策略；若仍被拦截，应在火绒中对经过校验的 Desktop、`offline-runtime\\node.exe` 和 DSH 入口分别允许，并保留脱敏日志。
- `offline-full` 仍需要系统 Node.js/npm 才能切换到在线 DSH 版本；没有系统运行时的机器应下载包含目标版本的新离线资产。

## [0.6.5] - 2026-08-20

### 修复

- DSH 子进程优先使用可达的自定义代理，自定义代理失效时依次尝试应用启动时继承的系统环境代理和国内 npm 镜像直连；手动 DSH 版本检查使用相同的有界降级顺序。
- 读取现有 DSH `web` Profile 的 pnpm Store 记录并仅在该子进程内固定 `npm_config_store_dir`，避免其他项目修改全局 `store-dir` 后造成 DSH Market 插件安装或更新失败。

### 新增

- 桌面工具和帮助页增加 GitHub 项目主页与 Desktop 手工更新入口；帮助页显示作者版权和 MIT License 信息，并区分 Desktop、DSH 与插件更新边界。

### 未验证

- 本机没有 Go 工具链；Go 测试和六平台构建结果以 `v0.6.5` tag 触发的 GitHub Actions 为准。

## [0.6.4] - 2026-08-20

### 修复

- NSIS 恢复或安装后同时更新当前 PowerShell 进程的 `PATH`，避免只写 `$GITHUB_PATH` 导致同一步骤仍找不到 `makensis`。
- 首次缓存未命中时直接下载固定的 `nsis.install 3.11.0` 包并校验 SHA-256，再从 runner 临时目录安装，绕过 Chocolatey OData 查询超时。

### 未验证

- `v0.6.4` 的 Windows x64 缓存冷启动和六平台 Release 结果以 tag 工作流为准。

## [0.6.3] - 2026-08-20

### 变更

- GitHub Actions 增加按平台复用的 Wails CLI 编译缓存和 Windows NSIS 安装目录缓存；Go/npm 原有缓存继续由官方 setup actions 管理。
- NSIS 准备增加缓存恢复后的 `makensis` 校验和三次 Chocolatey 重试，降低临时网络超时导致的 Windows x64 发布失败。

### 未验证

- 缓存命中率和节省时间需要下一次 GitHub Actions 运行后根据实际日志确认。

## [0.6.2] - 2026-08-20

### 修复

- 默认设置不再假设本机存在代理；自动版本检查固定优先直连国内 npm 镜像。
- 自定义代理端口不可达时，普通包快速回退到国内镜像直连，避免旧端口阻塞 DSH 启动。
- 手动刷新与应用 DSH 版本时，才按当前代理设置访问 registry；DSH 模型/API 代理边界在界面和日志中明确提示。
- Windows NSIS 安装器增加开始菜单卸载入口；帮助页补充 Setup、便携包、macOS 和 Ubuntu 的卸载/重装路径。

### 变更

- 普通包准备、自动检查和手动更新的网络路径分层，减少离线或未启动代理电脑的启动阻塞。

### 未验证

- 当前主机没有 Go 工具链；Go 测试与六平台构建仍由 GitHub Actions 验证。

## [Unreleased]

### 计划

- Windows 代码签名与 macOS Developer ID 签名、公证。
- 多实例工作区锁与 MCP 编排服务。

## [0.6.1] - 2026-08-20

### 修复

- 在线 npx 启动限制 npm 单次网络等待和重试；无代理时优先使用国内 npm 镜像，继承或自定义有效代理时不偷偷绕过代理。
- `latest`/`next` 更新改为失败可回滚：新版本无法完成 DSH 页面校验时恢复旧版本配置并自动重启旧运行时。

### 变更

- 在线 DSH 本地页面整体等待上限默认为 90 秒，并可在“代理与启动设置”中配置为 30–600 秒；离线运行时仍使用较长的本地启动等待。该上限不改变 DeepSeek Harness 内部模型/API 请求策略。
- 无代理时版本检查优先国内 npm 镜像，镜像不可用再尝试官方 registry；自定义代理不可用时不会静默切换网络路径。

## [0.6.0] - 2026-08-17

### 新增

- 启动页和 DSH 通道切换增加可验证阶段百分比，覆盖运行时检测、Node 校验、子进程启动、监听地址解析和本地页面指纹检查；不把不可测的 npm 依赖解析伪装成下载字节百分比。

### 变更

- 默认在线版本和下一轮 `offline-full` 固定依赖闭包提升到 `@deepseek-ai/dsh@0.1.0-rc.7`；同步审查 `node-pty@1.2.0-beta.15` 与 `dsh-subprocess-local@0.1.0-rc.7` 的生命周期命令、完整性和脚本 SHA-256。
- Windows 隔离安装测试增加已存在二进制的原地覆盖校验；Setup 继续复用记录的自定义安装目录，便携 `offline-full` 明确采用新目录解压升级。

### 待验证

- rc.7 离线闭包及新的 `node-pty` 预构建布局仍需未来递增 tag 的六平台原生构建、真实 PTY、最终归档和设备升级证据。

## [0.5.2] - 2026-08-17

### 新增

- 应用启动后使用当前代理模式在后台查询 npm 官方 `latest` 与 `next`，在顶栏和设置中同时显示两个通道；检查失败不阻塞当前 DSH 启动。
- 在线包可分别确认应用 `latest` 或 `next`，后端会再次核对官方 dist-tag 并只保存对应的精确 SemVer；预览通道不会静默安装。

### 修复

- 在线 npx 启动移除强制 `--prefer-offline`，避免刚发布的精确版本因陈旧 npm packument 被误报为 `ETARGET`，同时保留 npm 默认内容缓存。
- 版本检查完成后主动关闭 HTTP 空闲连接；更新重启继续同步回收本应用持有的 DSH 子进程树，避免反复检查或切换累积资源。

### 安全

- 更新查询固定到 npm 官方 DSH dist-tags 端点，限制响应大小、禁止非官方重定向，并复用 inherit/custom/disabled 代理策略；前端只能选择 `latest` 或 `next`，不能提交任意包名或版本。

## [0.5.1] - 2026-08-17

### 修复

- Starline 启动的 DSH 环境增加进程级命令兼容入口：Agent 执行 `dsh plugin` 时若遗漏官方必需的 `--profile`，默认补为当前 `web` profile；显式 profile 和其他 DSH 命令保持不变。

### 验证

- 质量门与 Windows x64/ARM64、macOS Intel/Apple Silicon、Linux x64/ARM64 原生构建全部通过；Windows 同时验证 PowerShell 与 CMD 的命令补全行为。

## [0.5.0] - 2026-08-16

### 新增

- Linux x64 与 ARM64 增加 Ubuntu 24.04 在线 DEB，安装应用菜单、图标与许可证，并在原生 Actions 中检查包元数据、架构、ELF 和动态库解析。

### 文档

- 明确 Windows 10 22H2、macOS 13.5、Ubuntu Desktop 24.04 的保守支持基线，旧系统和其他 Linux 发行版不再列为支持目标。
- Release 页面从 tag 内版本化的系统要求文件自动生成固定要求，避免后续 README 变化反向影响历史版本说明。

## [0.4.0] - 2026-08-16

### 新增

- 桌面设置增加 npm 官方 DSH 手动更新检查：在线包明确确认后保存精确版本并重启，可恢复 Desktop 内置兼容版本；离线包只提示下载匹配的新资产。
- Dependabot 每周检查离线运行时固定的官方 DSH 直接依赖，以受审查 PR 和六平台 CI 代替用户端后台自动升级。

### 安全

- 更新检查限制为 npm 官方 registry、校验返回包名和 SemVer、限制响应大小并拒绝非官方重定向；前端不能指定任意 npm 包或版本。

## [0.3.3] - 2026-08-16

### 改进

- 正式 Wails 构建启用系统 WebView 默认右键菜单，为选择、复制和粘贴提供平台原生回退。
- 离线运行时门禁增加真实 Sharp 图像转换、Koffi 动态库加载和本机函数调用；Windows 额外加载 `ole32.dll`，避免只加载模块却遗漏 FFI 运行失败。

### 文档

- 审计同类 DSH Desktop 的跨平台 Issue，明确 Electron 专属故障、本项目已规避路径、DSH 上游沙箱限制及 `danger-full-access` 与管理员权限的区别。
- 历史证据说明对齐已发布的 v0.3.2，发布元数据升级到 v0.3.3，并保留原生 CI、公开 Release 和设备验证之间的证据边界。

## [0.3.2] - 2026-08-14

### 修复

- 配置保存增加跨进程原子锁和版本检查，多开实例不会静默覆盖彼此的代理设置。
- Windows 使用本机文件锁，macOS/Linux 使用 `flock`，并在过期实例写入时返回明确冲突提示。

## [0.3.1] - 2026-08-14

### 修复

- 托盘实现改为仅在 Windows 编译第三方原生依赖，避免 macOS 与 Wails `AppDelegate` 符号冲突。
- Linux 构建不再在 Wails 绑定生成阶段加载 GTK 托盘依赖，修复无显示 CI runner 上的构建失败。
- macOS/Linux 关闭窗口时正常退出；Windows 继续通过托盘隐藏窗口，并从托盘菜单显式退出。

## [0.3.0] - 2026-08-14

### 改进

- 窗口右上角 X 改为隐藏到系统托盘，托盘菜单提供显示窗口、重启 DSH 和显式退出；显式退出才会进入 Wails shutdown 并回收宿主创建的子进程。
- 运行状态栏常驻显示 Desktop 与 DSH 版本号，便于确认实际启动的桌面端和上游运行时版本。
- Linux 构建补充 `libayatana-appindicator3-dev`，为通知区域托盘提供原生依赖。
- 首次启动增加短暂隐藏预热：DSH 很快就绪时直接显示已加载页面，超过 800 毫秒则显示稳定的后台准备界面，避免空白窗口和整页切换闪动。
- DSH iframe 在遮罩后加载并淡入；启动失败或进程意外退出时立即显示窗口，并在顶部状态栏提供错误摘要、详情、重试、代理和日志入口。
- 普通包通过 npx 启动固定版本 DSH 时增加 `--prefer-offline`，优先复用 npm 内容缓存，只有缓存缺失时才下载。

## [0.2.5] - 2026-08-14

### 修复

- 离线运行时继续默认禁用全部依赖脚本，只对白名单校验通过的 `node-pty` 构建和 DSH 官方 helper 权限修复开放执行。
- 六个原生平台增加真实 `node-pty.spawn()` Shell 测试，并在最终离线归档重新解包后复测原生文件、权限、输出和退出码。
- DSH Web 改用 `--port 0` 并解析其实际公布的 loopback URL，移除宿主预占后释放端口的竞争窗口。

### 改进

- 发行资产名统一包含版本、系统、CPU、安装形态和联网模式，明确区分 x64、ARM64、在线小包与完整离线包。
- GitHub Release 自动生成分平台快速下载表，显示实际文件体积，并从当前版本的变更日志精确生成“本版变更”。
- 发布任务会校验全部预期资产、独立校验文件和合并 SHA-256 清单；资产或版本日志缺失时直接停止发布。

### 文档

- 新增各平台已确认缺陷、同构建逻辑推断、设备验证缺口和临时规避方案的统一说明。
- 明确 v0.2.4 macOS/Linux `offline-full` 的 PTY 原生依赖问题，以及 CLI/Web 启动检查不能代替真实工具功能测试。
- 补充 Windows ARM64、Linux 发行版兼容性、签名、公证、安装路径和便携数据边界。

## [0.2.4] - 2026-08-14

### 新增

- 使用项目原创“星轨终端”图标替换 Wails 默认 `W`/旧图标，表达 Starline、终端与 DSH 宿主含义。
- 提供 1024×1024 RGBA 主图、Windows 七档多尺寸 ICO、可审计生成脚本及品牌/权利边界文档。
- Go 宿主显式嵌入主图，并用于 Linux 窗口与 macOS About 面板，避免平台回退到默认图标。

## [0.2.3] - 2026-08-14

### 修复

- 调试构建不再在每次启动时自动弹出 WebView 检查器；DevTools 仍可通过平台快捷键按需打开。

## [0.2.2] - 2026-08-14

### 新增

- 新增 `AUTHORS.md`，公开项目作者、主要维护者、联系方式、维护职责与贡献记录入口。
- README、Wails、前端和离线运行时补充统一的 starline 作者元数据。
- Windows、macOS、Linux 的普通包、安装包与离线包随包提供作者信息。
- Wails 调试构建自动打开 WebView 检查器，并补充正式 Release 默认关闭 DevTools 的调试说明。

## [0.2.1] - 2026-08-14

### 修复

- Windows runner 下载 Node.js LICENSE 时先统一为 UTF-8 LF，再执行固定哈希校验，避免 PowerShell 换行转换导致离线包构建误报。

## [0.2.0] - 2026-08-14

### 新增

- 独立的六平台 `offline-full` 便携包，内含固定 Node 与锁定的 DSH 生产依赖。
- 启动器优先使用完整且版本匹配的包内运行时，缺损时明确报错，不静默访问 npm。
- 桌面 UI 显示当前使用包内离线运行时还是系统 Node。
- Windows 安装器记住自定义安装目录，并增加中文、空格路径的安装/升级/卸载验证。
- Go 宿主按 application、config、launcher 职责分层，启动器进一步拆分运行时、代理、网络和日志模块。
- 新增统一文档导航与失效本地链接检查。
- 明确 MIT 版权所有者、贡献版权与第三方许可边界，并让发行产物携带许可证说明。

## [0.1.0] - 2026-08-14

### 新增

- 基于 Go 与 Wails 的 DeepSeek Harness 跨平台桌面宿主。
- 固定 DSH 版本启动、动态 loopback 端口、页面指纹健康检查。
- 继承环境、自定义地址和禁用三种代理模式。
- 启动诊断、日志目录、浏览器打开、重启与帮助入口。
- Windows 隐藏子进程与进程树回收，解决退出时控制台闪现。
- Windows Setup.exe/便携 ZIP、macOS ZIP、Linux TAR.GZ 打包流程。
- Windows、macOS、Linux 的 x64 与 ARM64 原生 CI/Release 构建矩阵。

[Unreleased]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.6...HEAD
[0.6.6]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.5...v0.6.6
[0.6.5]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.4...v0.6.5
[0.6.4]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.3...v0.6.4
[0.6.3]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.5.2...v0.6.0
[0.5.2]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.3.3...v0.4.0
[0.3.3]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.5...v0.3.0
[0.2.5]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/releases/tag/v0.1.0
