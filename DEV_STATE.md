# DEV_STATE

更新时间：2026-08-20。`v0.6.2` Release 工作流 `32338925807` 经 failed-job 重跑后六个平台和 Release 均成功。`v0.6.3` 提交 `c346784` 引入 Wails/NSIS 缓存，但 Windows x64 冷缓存流程只写 `$GITHUB_PATH`、未更新当前 PowerShell `PATH`，同一步骤仍无法确认 `makensis`，因此该 tag 的 Release 失败且不移动。当前修复准备发布为 `v0.6.4`：NSIS 命中/安装后同步当前与后续步骤 PATH，首次未命中时下载固定 `nsis.install 3.11.0` 包、校验 SHA-256 后从本地安装；Go/npm/Wails 缓存和所有运行时安全校验保持。前端 typecheck/build、文档链接、Release notes 测试、版本元数据、workflow YAML 解析和差异检查需重新验证；正式六平台结果以 `v0.6.4` tag 为准。
