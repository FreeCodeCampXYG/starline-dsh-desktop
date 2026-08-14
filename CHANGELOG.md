# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 的组织方式，并使用语义化版本号。

## [Unreleased]

### 计划

- Windows 代码签名与 macOS Developer ID 签名、公证。

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

[Unreleased]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.3...HEAD
[0.2.3]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/releases/tag/v0.1.0
