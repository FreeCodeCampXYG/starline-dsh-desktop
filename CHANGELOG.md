# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 的组织方式，并使用语义化版本号。

## [Unreleased]

### 计划

- Windows 代码签名与 macOS Developer ID 签名、公证。
- 可审计的可选离线运行时方案。

## [0.1.0] - 2026-08-14

### 新增

- 基于 Go 与 Wails 的 DeepSeek Harness 跨平台桌面宿主。
- 固定 DSH 版本启动、动态 loopback 端口、页面指纹健康检查。
- 继承环境、自定义地址和禁用三种代理模式。
- 启动诊断、日志目录、浏览器打开、重启与帮助入口。
- Windows 隐藏子进程与进程树回收，解决退出时控制台闪现。
- Windows Setup.exe/便携 ZIP、macOS ZIP、Linux TAR.GZ 打包流程。
- Windows、macOS、Linux 的 x64 与 ARM64 原生 CI/Release 构建矩阵。

[Unreleased]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/FreeCodeCampXYG/starline-dsh-desktop/releases/tag/v0.1.0
