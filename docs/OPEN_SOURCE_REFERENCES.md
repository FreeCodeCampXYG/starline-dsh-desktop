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

## 离线运行时再分发

`offline-full` 额外分发固定的 Node 24.19.0 可执行文件和 `offline-runtime/package-lock.json` 解析出的 DSH 生产依赖。Node 的 MIT 许可证复制为 `LICENSE-node.txt`，npm 包自身随发布 tarball 提供的许可证文件保留在 `node_modules` 中。升级 Node、DSH 或锁文件时必须重新检查许可证、平台可用性、依赖脚本策略和压缩体积。
