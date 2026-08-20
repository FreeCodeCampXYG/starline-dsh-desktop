# DEV_STATE

更新时间：2026-08-20。当前目标是维护 Go/Wails v2 的 DeepSeek Harness 薄宿主，并由 GitHub Actions 完成多平台在线构建与发布；本机不执行跨平台构建或离线运行时重建。

已完成：`v0.6.4` 已由提交 `fca1af0` 和 annotated tag 推送到远端。Release 工作流 `32341535288` 已成功，Windows x64、Windows ARM64、macOS Intel、macOS Apple Silicon、Linux x64、Linux ARM64 六个平台均完成构建、离线运行时准备、冒烟测试、归档和上传，Release `v0.6.4` 已创建并包含在线包、完整离线包、校验清单及 Linux DEB。Windows x64 的 `Ensure NSIS is available` 已通过；修复内容是 NSIS 命中或本地安装后同步当前 PowerShell PATH，并在冷启动时下载固定 `nsis.install 3.11.0`、校验 SHA-256 后安装。Wails CLI、Go、npm 和 NSIS 缓存均保留在 CI 工作流中，不进入发布资产。

已验证：workflow YAML、嵌入 PowerShell、版本元数据、Release notes、前端 typecheck/build、文档链接检查和 `git diff --check` 通过；仓库工作树干净，`main` 与远端同步。GitHub Actions 的 Node.js 20 弃用提示是非阻断警告。

已知限制：尚未取得代表性 Windows/macOS/Ubuntu 设备上的实测反馈；本机不宣称完成跨平台运行验证。后续若修改构建流程，应创建递增版本并只针对实际失败步骤回读 CI，不移动已公开 tag。

核心入口：`.github/workflows/ci.yml`、`.github/workflows/release.yml`、`CHANGELOG.md`、`README.md`、`docs/BUILDING.md`、`wails.json`、`frontend/package.json`。
