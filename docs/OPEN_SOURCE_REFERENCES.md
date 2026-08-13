# 开源工程参考

本项目没有直接复制其他仓库的工作流；以下项目用于校准构建矩阵、故障信息和维护入口的组织方式。

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
