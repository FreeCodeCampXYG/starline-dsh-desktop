# DEV_STATE

## 2026-09-01 Issue #6 npm registry 启动回退

- 在线普通包启动改为先探测官方 `https://registry.npmjs.org` 的 DSH dist-tags 端点；官方不可达时才使用 `https://registry.npmmirror.com`，避免假定用户无法访问官方 npm。
- 探测沿用用户选择的 HTTP(S) 代理环境；本机 1080 可作为自定义代理填写 `http://127.0.0.1:1080`，未将该端口硬编码为全局默认值。
- 新增 registry 顺序与环境选择单测；定向 Go 测试通过。完整 launcher 测试受当前机器缺少 `pwsh.exe` 的既有 Windows 兼容入口测试阻断；GitHub Actions 仍需完成全量验证。

## 2026-08-31 online Linux DEB 首次启动超时

- Issue #5 的 Linux/KDE 日志显示 online DEB 的 npx 准备阶段在 90 秒内未输出 DSH 监听地址；现有片段不足以断言无网络，仍需 npm debug log 区分 registry、代理和首次依赖准备问题。
- 默认 online 就绪上限改为 5 分钟，设置范围仍为 30–600 秒；在线超时会明确提示检查 npm registry/代理或改用 `offline-full`。`--no-open` 已回读精确的 `@deepseek-ai/dsh-web-app@0.1.1-rc.2` 发行包确认受支持，必须保留以避免 DSH 自动交接到系统浏览器。
- 本地通过文档链接检查和差异空白检查；当前机器未安装 Go，因此 Go 格式化、单测、Wails/DEB 构建仍待 CI/Linux 设备验证。Issue 已回复，建议使用匹配版本和架构的 `offline-full` DEB，并补充完整宿主/npm 脱敏日志。

## 2026-08-23 V2 浏览器回归测试

- 使用 ChromeGo CDP 9222，以工号 10999、演示机构、管理员只读边界执行本地 V2 与公司旧版回归；本地前端 `127.0.0.1:4200`、后端 `127.0.0.1:9200`，公司旧版 `10.1.50.47`。
- 交付物位于 `v2-test-run/`：测试计划、Markdown/HTML 报告、V2/旧版对比、问题清单、接口证据草案、截图和局部 outerHTML。
- 已确认：V2 query/page、query/audit、字段/模板接口可达并返回 HTTP 200；本地自造模板保存与删除成功且已清理。
- 已确认问题：组合查询字段选择嵌套抽屉移出视口（P0）；空值条件先保存/查询后提示（P1）；应用与主查询重复 query/page（P1）；导出前提示残留（P2）；1366 结果区横向滚动（P2）；旧版导出以 HTTP 200 携带业务错误（P1）。
- 未验证：真实有数据结果、导出大于 5000 条、双击病案打开、完整字段类型矩阵和后端 SQL 日志根因；需要修复 P0 后继续浏览器回归。

## 2026-08-22 v0.6.12 DSH Web 浏览器交接拦截

- 根因已确认：`@deepseek-ai/dsh-web-app@0.1.1-rc.2` 的 Web 启动默认开启系统浏览器交接，并提供 `--no-open`；Wails 宿主此前未传该参数，所以启动时会额外拉起浏览器。
- `internal/launcher/launcher.go` 将宿主启动参数集中为 `dshWebArgs()`，固定追加 `--no-open`；loopback URL 仍由宿主校验后交给内嵌 iframe，手动“在浏览器中打开”未移除。`internal/launcher/launcher_test.go` 新增参数契约回归；README、架构和排错文档已同步。
- 本地已通过 `node scripts/check-doc-links.mjs`（15 个 Markdown）和 `git diff --check`；当前机器没有 Go/gofmt，未执行 Go 单测。待 `v0.6.12` GitHub Actions 完成跨平台构建与 Release 验证。
- 代码提交为 `8485649`；发布准备提交为 `0e06220`，状态提交为 `1c19064`；annotated `v0.6.12` 已通过 `127.0.0.1:1080` 推送并回读确认。按用户要求未推送 `main`，远端 `main` 仍停在 `v0.6.11`；tag 工作流的跨平台构建与 Release 结果待回读。

## 2026-08-22 GitHub Actions offline packaging optimization

- Remote run `32520690184` confirmed the main delay was the `resolve-offline-lock` job: repeated `npm install --package-lock-only` took about 13 minutes. The v0.6.7 macOS OOM also occurred in that step, not while creating the final archives.
- `.github/workflows/release.yml` now removes npm download caching, the per-platform offline `node_modules` cache, and the separate lock-resolution job. Each runner prepares its own runtime from the committed lock, avoiding repeated compression/restoration and registry resolution.
- `.github/workflows/refresh-offline-lock.yml` is the only online lock refresh path. It runs on GitHub, validates the pinned DSH version, and commits `offline-runtime/package-lock.json` to `main`; release and CI workflows fail loudly if the tracked lock is stale.
- Release artifacts keep their existing zip/tar/deb formats, while `actions/upload-artifact` uses `compression-level: 0` to avoid compressing already-compressed files a second time. Split-volume archives were not introduced because they would add multi-file download/extraction requirements without addressing the lock-resolution OOM.
- Local verification: workflow YAML parsed successfully and `git diff --check` passed. GitHub lock refresh run `32561978384` succeeded and committed `1bba8dc`; CI run `32562729911` then passed its quality gates and all six native-build jobs. No local npm/Go build or artifact download was performed.

## 2026-08-22 v0.6.10 旧 DSH 在线覆盖修复

- `v0.6.8` Windows 构建日志已确认通过 `-X main.defaultDSHVersion=0.1.1-rc.2` 嵌入默认版本，并在 runner 上输出 `Synced offline DSH 0.1.1-rc.2`；截图中的 `0.1.0-rc.7 · 系统 Node / npx` 来自用户配置保留的旧在线版本，离线包不匹配时按设计切换到系统 npx。
- `internal/application` 现在比较保存版本与包内版本：包内版本更新时优先使用离线闭包，保留旧配置供“恢复默认”清理；仅保存版本高于包内版本时继续使用在线 npx。新增回归测试覆盖 `0.1.0-rc.7` 配置被 `0.1.1-rc.2` 离线包覆盖的场景。
- 提交 `18a51e8` 已进入 `v0.6.10` annotated tag；本机 `git diff --check` 通过，未执行 Go test（当前机器没有 Go），Release run `32559326375` 和 CI `32559324126` 正在验证。

## 2026-08-22 v0.6.8 Release runner 修复

- `v0.6.7` 的三个失败均发生在离线 runtime 准备：Linux x64 为临时 npm `ETARGET`，两个 macOS 为 Node 24.19.0 默认 heap OOM；未发现 DSH/Node 引擎不兼容。
- Release workflow 现在只解析一次 lock 并以 artifact 复用，按平台缓存 `node_modules`，Node heap 上限 4096 MiB；准备脚本提供缓存跳过选项和 npm 有界重试/超时，同时保留元数据同步、原生模块重建和冒烟测试。
- 发布脚本允许 DSH 版本保持 `0.1.1-rc.2`、仅递增 Desktop 版本的修复发布；当前本地验证已通过 PowerShell、Git Bash、YAML 与 `git diff --check`。
- `v0.6.8` 的六个平台构建、离线运行时准备和冒烟测试均成功；原发布说明步骤因下载了内部 `package-lock.json` artifact，触发资产清单不一致。常规 Release workflow 已改为只下载 `starline-dsh-desktop-*` artifact，线上复用 workflow 已用历史 run `32520690184` 成功创建 `v0.6.8` 正式 Release（33 个资产）。
- 为避免重复构建，新增 `.github/workflows/publish-existing-release.yml`：可在 GitHub runner 上按历史 Release run ID 下载并过滤平台 artifact，直接为现有 tag 生成 notes 和 Release；本机不搬运构建包。

当前状态：`v0.6.8` Release 已公开，标签未移动，包含六平台 online/offline-full 资产与 SHA-256 清单；`v0.6.9` 修复 tag 已推送但其重复构建已取消，未创建对应 Release。

## 2026-08-22 标签驱动的 DSH 发布准备脚本

- 新增 `scripts/prepare-dsh-release.ps1` 和 `docs/RELEASING.md` 使用说明。脚本要求显式 DSH 版本，只同步 DSH/Desktop 版本、文档和 CHANGELOG，不在本机查询 npm、下载依赖或构建离线包；tag runner 通过 `scripts/sync-offline-runtime-metadata.mjs` 刷新 `package-lock.json`、依赖 integrity 和 `ensure-spawn-helper.mjs` SHA-256。
- `-Commit -Tag -Push` 会在同一次运行中逐步确认提交、annotated tag 和 `main`/tag 推送；GitHub Actions 从新 tag 负责六平台 online/offline-full 原生构建与归档。省略这些开关只生成待审阅改动，不能在工作区变脏时重复运行。
- 当前脚本、runner 同步逻辑和文档尚未推送到远端；上一项在线更新回退修复仍在本地提交 `55c51fd`，官方 DSH dist-tag 已确认 `0.1.1-rc.2`。待用户在具备 starline GitHub 凭据的机器上运行脚本并完成 CI/设备验证。

## 2026-08-22 DSH 在线更新回退

- 官方 npm registry 与国内镜像当前均返回 `@deepseek-ai/dsh` 的 `latest=0.1.1-rc.2`、`next=0.1.1-rc.2`；之前用户看到的 `0.1.1-rc.1` 已不是当前 dist-tag。
- 修复 `offline-full` 检查到新版本后无应用按钮的问题：包内版本仍默认离线启动；确认新版本后，如果系统存在 Node.js 22.19+ 或 24+ 及 npm/npx，宿主改用系统在线运行时并沿用代理降级链下载，不改写包内依赖闭包；失败仍可通过恢复默认回到内置版本。缺少系统 Node/npm 时，界面明确提示需要安装运行时或下载新离线资产。
- 关键代码为 `internal/launcher/runtime.go`、`internal/application/app.go`；回归覆盖离线包版本选择、在线运行时回退及 `0.1.1-rc.2` 跨 minor 版本检查。已通过前端构建、文档链接检查、JavaScript 语法检查和 `git diff --check`；本机没有 Go/Wails，Go 测试、原生构建、系统 Node/npm 在线下载和真实 Windows UAC/火绒行为仍待 CI/设备验证。

## 2026-08-20 Windows 管理员权限修复

- Windows 主程序 manifest 已显式声明 `requestedExecutionLevel=requireAdministrator`；正式 EXE 启动会请求 UAC 管理员令牌，Node/DSH 子进程继承该令牌，macOS/Linux 不变。
- 同步更新 `CHANGELOG.md`、`README.md` 和 `docs/TROUBLESHOOTING.md`，明确 UAC 行为及火绒信誉/行为防护仍需单独放行，管理员权限不等于绕过安全软件。
- 静态 XML 解析确认权限级别为 `requireAdministrator`；Wails web-host 结构审计 20 pass/0 warn/0 fail。GitHub Release 已生成 Windows x64/ARM64、macOS Intel/Apple Silicon、Linux x64/ARM64 资产及各自校验文件，Windows 离线 web runtime 冒烟测试通过。当前机器未安装 Go/Wails，未执行本地原生构建；真实设备上的最终 EXE 嵌入 manifest、UAC 弹窗、火绒策略和日志权限仍待验证。

## 2026-08-20 实机 Profile bundle 与 task-board 锁

- Windows 实机曾出现 `web` Profile 声明 `@linxin666/dsh-web-ui-all` 但 `node_modules` 缺包；确认 Profile 路径和离线运行时均存在，问题来自插件安装后的不一致状态，不是路径被删除。
- 通过 `127.0.0.1:1080` 代理补装 `@linxin666/dsh-web-ui-all@0.2.5`；pnpm 因原生构建脚本门禁返回非零，随后用 `dsh plugin --profile web install --ignore-scripts` 完成依赖树收尾。`--dump-config` 已能解析该 bundle 及其 loader 层。
- 启动测试留下的 `C:\Users\xiaoy\.dsh\task-board\ledger-v2.lock` 记录已退出的 PID 17076，导致重启时报 `task-board ledger is already owned`；已只删除该陈旧锁，保留 `ledger-v2.json`。下一步由用户再次重试 Desktop；若仍失败，需提供新的完整日志。

更新时间：2026-08-20。`v0.6.5` 目标已完成：提交 `33c6ab4` 与 annotated tag 已推送，GitHub CI `32353799488` 的 Go 格式、Go test/vet、前端、离线依赖闭包和 Windows x64/ARM64、macOS Intel/Apple Silicon、Linux x64/ARM64 六个平台原生构建全部成功；Release `32353813467` 的六平台打包和发布成功，正式页面包含 33 个资产、本版变更和固定系统要求。已确认原故障来自现有 DSH `web` Profile 记录的 pnpm Store 与另一项目误写的全局 `store-dir` 不一致；该全局误配置已恢复为 pnpm 默认行为，项目自己的缓存脚本和项目级配置未改动，Profile 离线锁文件一致性检查通过且插件清单未变化。代码现在只在 DSH 子进程内复用 Profile `.modules.yaml` 已记录的 Store，不修改全局 pnpm 配置；启动与手动 DSH 版本检查按可达的自定义代理、应用启动时继承的 HTTP(S) 环境代理、国内镜像直连依次降级，网络请求保持有界，DSH 新版本启动失败继续恢复旧版本配置和进程。桌面工具、macOS 菜单和帮助页已增加 GitHub 项目主页、Desktop 最新 Release 手工入口、版权及 MIT License 信息；Desktop、DSH、插件和离线包更新边界已在 `README.md`、`CHANGELOG.md`、`docs/ARCHITECTURE.md`、`docs/TROUBLESHOOTING.md` 中统一。核心文件为 `internal/launcher/environment.go`、`internal/launcher/dsh_command_shim.go`、`internal/updater/dsh.go`、`internal/application/app.go`、`frontend/src/main.ts`。已知边界：系统代理仅指进程继承的 HTTP(S) 环境变量，不读取 Windows 系统代理面板；DSH Market 自身事务语义仍属于上游插件；Actions 仅报告既存的 `actions/cache@v4` Node 20 弃用警告，不影响本次结果；代表性设备上的实际插件升级仍待用户反馈。下一步只需在真实应用中重试 DSH Market 插件更新；若仍失败，保留错误开头和完整脱敏日志，按单个失败点处理，不移动已公开 tag。
