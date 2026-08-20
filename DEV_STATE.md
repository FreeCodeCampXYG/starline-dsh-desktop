# DEV_STATE

更新时间：2026-08-20。`v0.6.2` Release 工作流 `32338925807` 首次仅因 Windows x64 下载 NSIS 时 Chocolatey 返回 HTTP 408 失败，失败 job 重跑后六个平台和 Release 均成功。`v0.6.3` 提交 `c346784` 与 annotated tag 已推送，增加 Wails CLI 按平台缓存、Windows NSIS 3.11.0 安装目录缓存及三次安装重试/`makensis` 校验；Go/npm 继续使用 setup actions 的既有缓存，缓存不进入安装包或离线运行时。前端 typecheck/build、文档链接、Release notes 测试、版本元数据、workflow YAML 解析和差异检查通过；`v0.6.3` CI/Release 正在运行，缓存命中率与节省时间等待线上日志确认。普通包和离线包边界、rc.7 本机离线启动验证保持不变，不移动已公开 tag。
