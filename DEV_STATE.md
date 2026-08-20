# DEV_STATE

更新时间：2026-08-20。待发布基线为 `v0.6.2`，上一公开版本 `v0.6.1` 核心提交为 `55b8110`；本次增加自动检查直连国内镜像、手动更新按代理、普通包自定义代理不可达回退、Windows 开始菜单卸载入口和跨平台卸载说明。普通包依赖系统 Node/npm 及当前 npm 缓存路径，离线包才自带 Node 与 DSH 依赖；用户配置已切为禁用代理并恢复到含 rc.7 的本机 npm 缓存，`D:\winsofts\dsh-desktop` 已用本地 rc.7 离线运行时启动验证。前端 typecheck、文档链接检查、前端构建、gofmt、application/config/updater 及 launcher 代理回退相关测试通过；完整 launcher 测试因当前环境没有 `pwsh.exe` 被仓库原有 Windows 兼容测试阻断。Go 六平台构建仍由 GitHub Actions 验证。下一步：通过 1080 推送 `main`、创建并推送递增 tag `v0.6.2`，再根据 CI 结果修复单个问题，不移动旧 tag。
