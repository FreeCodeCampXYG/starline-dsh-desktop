# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 的组织方式，并使用语义化版本号。

## [Unreleased]

### 计划

- Windows 代码签名与 macOS Developer ID 签名、公证。
- 多实例工作区锁与 MCP 编排服务。

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

[Unreleased]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.5.0...HEAD
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
