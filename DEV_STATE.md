# DEV_STATE

当前状态：`a5dea50` 的 Windows `requireAdministrator` 修复已由 `4c04b13` 同步为 `v0.6.6`，main 与 annotated tag 均已推送；GitHub CI `32368691430` 和 Release `32368740319` 已成功完成六平台构建、离线运行时冒烟测试与发布。

## 2026-08-20 Windows 管理员权限修复

- Windows 主程序 manifest 已显式声明 `requestedExecutionLevel=requireAdministrator`；正式 EXE 启动会请求 UAC 管理员令牌，Node/DSH 子进程继承该令牌，macOS/Linux 不变。
- 同步更新 `CHANGELOG.md`、`README.md` 和 `docs/TROUBLESHOOTING.md`，明确 UAC 行为及火绒信誉/行为防护仍需单独放行，管理员权限不等于绕过安全软件。
- 静态 XML 解析确认权限级别为 `requireAdministrator`；Wails web-host 结构审计 20 pass/0 warn/0 fail。GitHub Release 已生成 Windows x64/ARM64、macOS Intel/Apple Silicon、Linux x64/ARM64 资产及各自校验文件，Windows 离线 web runtime 冒烟测试通过。当前机器未安装 Go/Wails，未执行本地原生构建；真实设备上的最终 EXE 嵌入 manifest、UAC 弹窗、火绒策略和日志权限仍待验证。

## 2026-08-20 实机 Profile bundle 与 task-board 锁

- Windows 实机曾出现 `web` Profile 声明 `@linxin666/dsh-web-ui-all` 但 `node_modules` 缺包；确认 Profile 路径和离线运行时均存在，问题来自插件安装后的不一致状态，不是路径被删除。
- 通过 `127.0.0.1:1080` 代理补装 `@linxin666/dsh-web-ui-all@0.2.5`；pnpm 因原生构建脚本门禁返回非零，随后用 `dsh plugin --profile web install --ignore-scripts` 完成依赖树收尾。`--dump-config` 已能解析该 bundle 及其 loader 层。
- 启动测试留下的 `C:\Users\xiaoy\.dsh\task-board\ledger-v2.lock` 记录已退出的 PID 17076，导致重启时报 `task-board ledger is already owned`；已只删除该陈旧锁，保留 `ledger-v2.json`。下一步由用户再次重试 Desktop；若仍失败，需提供新的完整日志。

更新时间：2026-08-20。`v0.6.5` 目标已完成：提交 `33c6ab4` 与 annotated tag 已推送，GitHub CI `32353799488` 的 Go 格式、Go test/vet、前端、离线依赖闭包和 Windows x64/ARM64、macOS Intel/Apple Silicon、Linux x64/ARM64 六个平台原生构建全部成功；Release `32353813467` 的六平台打包和发布成功，正式页面包含 33 个资产、本版变更和固定系统要求。已确认原故障来自现有 DSH `web` Profile 记录的 pnpm Store 与另一项目误写的全局 `store-dir` 不一致；该全局误配置已恢复为 pnpm 默认行为，项目自己的缓存脚本和项目级配置未改动，Profile 离线锁文件一致性检查通过且插件清单未变化。代码现在只在 DSH 子进程内复用 Profile `.modules.yaml` 已记录的 Store，不修改全局 pnpm 配置；启动与手动 DSH 版本检查按可达的自定义代理、应用启动时继承的 HTTP(S) 环境代理、国内镜像直连依次降级，网络请求保持有界，DSH 新版本启动失败继续恢复旧版本配置和进程。桌面工具、macOS 菜单和帮助页已增加 GitHub 项目主页、Desktop 最新 Release 手工入口、版权及 MIT License 信息；Desktop、DSH、插件和离线包更新边界已在 `README.md`、`CHANGELOG.md`、`docs/ARCHITECTURE.md`、`docs/TROUBLESHOOTING.md` 中统一。核心文件为 `internal/launcher/environment.go`、`internal/launcher/dsh_command_shim.go`、`internal/updater/dsh.go`、`internal/application/app.go`、`frontend/src/main.ts`。已知边界：系统代理仅指进程继承的 HTTP(S) 环境变量，不读取 Windows 系统代理面板；DSH Market 自身事务语义仍属于上游插件；Actions 仅报告既存的 `actions/cache@v4` Node 20 弃用警告，不影响本次结果；代表性设备上的实际插件升级仍待用户反馈。下一步只需在真实应用中重试 DSH Market 插件更新；若仍失败，保留错误开头和完整脱敏日志，按单个失败点处理，不移动已公开 tag。
