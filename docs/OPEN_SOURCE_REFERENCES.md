# 开源工程参考

本项目没有直接复制其他仓库的工作流；以下项目用于校准构建矩阵、故障信息和维护入口的组织方式。

## Go/Wails 结构

- [Wails v2.14.0 NSIS 模板](https://github.com/wailsapp/wails/tree/v2.14.0/v2/pkg/buildassets/build/windows/installer)：保留 `build/windows/installer/project.nsi` 作为可维护安装器源码，并由 Wails 生成工具宏；
- [October](https://github.com/marcus-crane/october)：根入口负责嵌入前端和组装 Wails，普通 Go 行为放入独立 backend 包；
- [go-stock](https://github.com/ArvinLovegood/go-stock)：大型 Wails 项目按 agent、data、db、logger 等变化轴拆分后端，说明领域增长后不能继续把逻辑堆在根 `app.go`；
- [ChYing](https://github.com/yhy0/ChYing)：使用独立配置、数据库和代理包，但其当前主分支已是 Wails v3，本项目只参考职责边界，不混用 v3 API。

本项目没有照搬这些仓库的目录名。结合 Go 的 `internal` 可见性、本项目“薄宿主”的规模和 Wails `embed` 限制，采用根组合入口 + `internal/application`、`internal/config`、`internal/launcher`。这比通用大仓库模板更符合 KISS/YAGNI，也避免根 `app.go` 随功能增长失控。

## Windows 安装路径

- [NSIS InstallDirRegKey](https://nsis.sourceforge.io/Reference/InstallDirRegKey)：注册表中存在有效路径时覆盖默认 `InstallDir`；
- [NSIS Unicode](https://nsis.sourceforge.io/Reference/Unicode)：`Unicode true` 生成 Unicode 安装器；
- Wails 默认 NSIS 模板提供目录选择页，本项目补充 `InstallLocation` 写入/复用、引号保护，以及中文和空格路径的隔离安装测试。

## GitHub Actions

- [Wails](https://github.com/wailsapp/wails)：按操作系统使用原生 runner 构建，避免把 GUI 应用的交叉编译结果当作完整验证。
- [LocalSend](https://github.com/localsend/localsend)：按平台和架构拆分产物，再由独立发布任务汇总 Release。
- [GitHub runner-images](https://github.com/actions/runner-images)：核对当前 runner 标签、处理器架构和预装工具。

据此，本仓库使用六个原生目标：Windows x64/ARM64、macOS Intel/Apple Silicon、Linux x64/ARM64。构建任务 `fail-fast: false`，便于一次看到全部平台状态；Release 只有在所有目标完成后才发布，并为每个文件生成 SHA-256。

## Issue 与 Pull Request

- [LocalSend issue forms](https://github.com/localsend/localsend/tree/main/.github/ISSUE_TEMPLATE)：把 Bug 与功能建议分开，要求版本、平台和复现步骤。
- [AppFlowy issue forms](https://github.com/AppFlowy-IO/AppFlowy/tree/main/.github/ISSUE_TEMPLATE)：用结构化表单降低维护者补问成本。
- [Wails pull request template](https://github.com/wailsapp/wails/blob/master/.github/PULL_REQUEST_TEMPLATE.md)：明确关联 Issue、测试范围和平台信息。

Starline 的表单额外要求先判断问题属于桌面宿主还是 DSH 上游，并提醒提交者删除 API Key、Token、用户名和工作目录等敏感信息。PR 模板列出六个平台，但只要求贡献者勾选实际验证过的目标；未验证项交由 CI 给出结果。

## 版本策略

工作流中的官方 GitHub Actions 使用当前稳定主版本；Wails CLI 与应用依赖固定为明确版本。Dependabot 每周检查 Go、npm 与 Actions 更新，避免在构建时隐式追随 `latest`。

## 同类 DSH Desktop Issue 经验

[anywhere-labs/deepseek-harness-desktop Issues](https://github.com/anywhere-labs/deepseek-harness-desktop/issues) 被用作跨平台故障样本，不作为可直接复制的实现来源。审计后的主要决策是：

- [运行时边界讨论 #62](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/62) 支持继续保持“薄壳 + 独立 DSH 子进程”，让插件故障、退出码、日志和重启保持清晰边界；
- [打包漏传递依赖 #57](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/57) 说明开发环境启动成功不能证明发布闭包完整，因此本项目保留真实 npm 依赖树、最终归档复测和固定版本入口；
- [Electron runner 误用 `process.execPath` #71](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/71) 与 [原生目录选择 FFI #29](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/29) 属于 Electron/native capability 架构路径。本项目启动真实 Node、保留 Web 目录浏览，不移植 Electron 环境变量补丁或 SSH 环境伪装；但会在六平台离线门禁中真实加载 Koffi 动态库和调用本机函数；
- [右键菜单 #12](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/12) 与 [剪贴板权限/降级 #14](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/14) 促使本项目在正式 Wails 构建中启用默认 WebView 右键菜单，并继续保留 iframe 剪贴板权限和浏览器回退；
- [Windows 黑框 #15](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/15) 与 [ACL 沙箱 PowerShell 失败 #84](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/84) 位于 DSH 更深层子进程，外壳隐藏自己直接创建的进程但不篡改上游沙箱；用户可显式选择 DSH `danger-full-access`，宿主不会在失败时静默降级权限；
- [MCP readiness 超时 #45](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/45) 与 [重复插件注册 #8](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/8) 说明启动失败必须保留原始日志和有界重试。本项目不自动修改用户 profile 或插件图；
- [macOS 全屏/自绘标题栏 #49](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/49) 与 [透明/vibrancy 性能 #79](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/79) 进一步支持使用默认原生标题栏和不透明 WebView，不复制装饰性平台特效。

这些结论只说明架构适用性；外部 Issue 的设备结果不能替代本项目自身的发布资产审计和设备测试。

## 离线运行时再分发

`offline-full` 额外分发固定的 Node 24.19.0 可执行文件和 `offline-runtime/package-lock.json` 解析出的 DSH 生产依赖。Node 的 MIT 许可证复制为 `LICENSE-node.txt`，npm 包自身随发布 tarball 提供的许可证文件保留在 `node_modules` 中。升级 Node、DSH 或锁文件时必须重新检查许可证、平台可用性、依赖脚本策略和压缩体积。
