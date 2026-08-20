# DEV_STATE

更新时间：2026-08-20。`v0.6.2` 已由提交 `64069e3` 推送到 `main`，Windows x64 失败 job 重跑后通过，Release 正在汇总；当前工作改动准备发布为 `v0.6.3`，增加 Wails CLI 按平台缓存、Windows NSIS 缓存与 NSIS 重试校验。普通包依赖系统 Node/npm 及当前 npm 缓存路径，离线包才自带 Node 与 DSH 依赖；用户配置已切为禁用代理并恢复到含 rc.7 的本机 npm 缓存，`D:\winsofts\dsh-desktop` 已用本地 rc.7 离线运行时启动验证。工作流文本 YAML 解析和差异检查通过；正式缓存命中率与节省时间需由后续 GitHub Actions 运行确认。Go 六平台构建仍由 GitHub Actions 验证，不移动已公开 tag。
