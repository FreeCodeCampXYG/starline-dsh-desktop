# 已知问题与平台支持边界

本文是平台缺陷和验证边界的唯一说明。README、构建和排错文档只保留摘要并链接到这里，避免把推测、CI 启动成功和真实设备功能验证混在一起。

## 状态定义

| 状态 | 含义 |
| --- | --- |
| 已确认缺陷 | 已从正式 Release 归档内容或实际功能测试确认 |
| 同构建逻辑推断 | 已在一个架构确认，另一个架构使用相同打包逻辑；尚未单独执行设备测试 |
| 缺少设备验证 | 构建或 Web 启动检查成功，但没有代表性设备上的安装和功能证据 |
| 产品限制 | 当前设计或分发方式明确不覆盖的能力，不等同于回归 Bug |

## v0.6.0 发布候选边界

本版加入单调阶段百分比，并把默认在线版本与 `offline-full` 锁定到 DSH rc.7。rc.7 同时把 `node-pty` 提升到 `1.2.0-beta.15` 并改用新的预构建布局；源码已同步生命周期白名单、完整性、脚本 SHA-256 和结构检查，但六平台原生安装、真实 PTY、最终归档与 Setup 覆盖证据必须来自 `v0.6.0` tag 的 Release 工作流。阶段百分比不是 npm 下载字节百分比，停在 65% 仅表示 npx 子进程已启动且仍在准备依赖；代表性设备升级验证仍然缺失。

当前 main 还加入了在线启动网络边界：npm 单次请求默认等待 10 秒并最多重试 1 次；普通包准备、自启动版本检查固定优先直连国内 npm 镜像，自定义代理端口不可达时普通包快速回退镜像；手动刷新/应用版本才按用户代理设置访问。普通包本地 DSH 页面默认最多等待 90 秒，可在设置中配置为 30–600 秒。镜像和宿主等待只覆盖 npm 运行时准备与本地 Web 就绪，不覆盖 DeepSeek Harness 内部模型/API 请求；后者的连接、流式响应和超时仍由 DSH 上游决定。`latest`/`next` 切换如果新版本下载或本地页面校验不完整，会尝试恢复更新前的精确版本和配置；若用户在切换期间从其他实例修改了配置，则为避免覆盖会报告冲突而不强行回退。上述改动尚未形成新的公开 tag，仍需线上 CI 验证。

## v0.5.2 发布候选边界

本版增加启动后自动查询 npm `latest`/`next`、显式通道切换，并移除在线启动强制的 `--prefer-offline`。本地仅做静态检查，六平台编译、原生依赖、归档和 DEB 安装证据必须来自对应 tag 的 Release 工作流；工作流完成前不得把候选实现描述为已通过跨平台验证。自动检查直连国内镜像且不阻塞 DSH，手动通道切换会回收当前应用持有的进程树，但仍缺少代表性设备上的升级体验验证。

## v0.5.1 当前发布证据

提交 `299578c` 的 [CI 运行 32023493664](https://github.com/FreeCodeCampXYG/starline-dsh-desktop/actions/runs/32023493664) 已通过质量门和六平台原生构建；[v0.5.1 Release 工作流 32026664082](https://github.com/FreeCodeCampXYG/starline-dsh-desktop/actions/runs/32026664082) 已完成最终打包与发布。新增的进程级兼容入口只在 Agent 执行 `dsh plugin` 且完全遗漏 `--profile` 时补为当前 `web` profile；显式 profile 与其他 DSH 命令保持不变，不修改全局 PATH 或 npm shim。真实设备权限、插件网络和 pnpm 可用性仍不由该修复保证。

## v0.5.0 当前发布证据

提交 `70c805a` 的 [CI 运行 31928652053](https://github.com/FreeCodeCampXYG/starline-dsh-desktop/actions/runs/31928652053) 已通过质量门和六平台原生构建；[v0.5.0 Release 工作流 31929012111](https://github.com/FreeCodeCampXYG/starline-dsh-desktop/actions/runs/31929012111) 已通过六平台打包并发布 [33 个正式资产](https://github.com/FreeCodeCampXYG/starline-dsh-desktop/releases/tag/v0.5.0)。发布工作流在各原生 runner 上准备离线运行时，真实调用 `node-pty.spawn()`，并在最终 ZIP/TAR.GZ 解包后复测；这属于原生 CI、归档和公开 Release 证据，不属于代表性设备测试。

| 平台与产物 | 已确认 | 仍缺少的证据 |
| --- | --- | --- |
| Windows x64/ARM64 | 原生 runner 完成应用构建、NSIS/便携包生成；最终离线 ZIP 实际运行 Node、DSH、PTY、ripgrep，并加载 Sharp/Koffi | ARM64 实体设备；干净机器安装、首次启动、升级、卸载；SmartScreen 与安全软件差异 |
| macOS Intel/Apple Silicon | 原生 runner 完成 `.app`、在线/离线 ZIP 和最终归档 PTY/helper 权限复测 | 代表性 Mac 首次启动、工作区、插件和升级；Developer ID 签名与公证 |
| Linux x64/ARM64 | Ubuntu 24.04 原生 runner 完成动态链接应用、在线 DEB、在线/离线 TAR.GZ、最终归档原生工具复测，以及 DEB 的 control/架构/ELF/动态库检查和 apt 安装/卸载 | 代表性 Ubuntu 桌面设备的 GUI、应用菜单、首次启动、升级和卸载；ARM64 实体设备 |

v0.5.0 已增加 Ubuntu 24.04 x64/ARM64 在线 DEB；正式工作流确认 control/架构/ELF/动态库检查、apt 安装/卸载及六平台打包通过，但仍缺少代表性 Ubuntu 设备的 GUI、应用菜单、首次启动、升级和卸载验证。v0.4.0 历史 Release 不包含 DEB。支持边界固定为 [系统要求与兼容基线](SYSTEM_REQUIREMENTS.md)，不再把旧 Ubuntu 或其他发行版列为理论兼容目标。

## v0.3.3 发布加固

- 正式 WebView 启用默认右键菜单，复制按钮受上游权限或实现影响时仍可手工选择、复制和粘贴；尚缺六平台 GUI 设备验证。
- 离线检查不再只 `require('sharp')` / `require('koffi')`：Sharp 必须实际生成 PNG，Koffi 必须加载本机动态库并调用进程 ID 函数；Windows 还会专门加载 `ole32.dll`，覆盖同类桌面端曾出现的 FFI 加载崩溃路径。
- 这些改动已随 v0.3.3 完成原生 CI、打包和 Release；它们不反向改变 v0.3.2 资产。

## v0.2.4 历史平台矩阵

| 平台与产物 | 当前状态 | 缺陷、限制与证据边界 |
| --- | --- | --- |
| Windows x64 普通 ZIP / Setup | 可用，但未签名 | 自动化覆盖当前用户安装、中文和空格路径；SmartScreen 仍可能显示“未知发布者”。`Program Files` 等需管理员权限的目录不属于当前用户安装包的支持范围。 |
| Windows x64 `offline-full` | 已完成功能抽查 | 正式归档中的 Node、DSH、`sharp`、`koffi`、ripgrep 和 `node-pty` 已实际加载；PTY 已启动 `cmd.exe` 并返回测试标记。离线包仍不能让模型服务、远程 MCP 或 Web 工具在无网络时工作。 |
| Windows ARM64 全部产物 | 缺少设备验证 | 原生 runner 构建和 Web 启动检查已通过，归档结构包含 ARM64 PTY 预构建文件；尚无代表性 Windows on ARM 设备上的 Setup、升级、卸载和真实 PTY 启动证据。 |
| macOS 普通 ZIP | 有条件可用 | 依赖系统 Node/npx；未做 Developer ID 签名和公证，Gatekeeper 可能阻止首次打开。当前只提供 ZIP 中的 `.app`，没有 DMG。 |
| macOS ARM64 `offline-full` | **已确认缺陷** | v0.2.4 正式 ZIP 中 `node-pty` 的 `spawn-helper` 权限为 `0644`，缺少可执行位。Web 页面能够启动不代表终端、Shell 或依赖 PTY 的工具可用；当前不把这些功能列为受支持能力。 |
| macOS Intel `offline-full` | **同构建逻辑推断存在缺陷** | 与 ARM64 使用同一条跳过依赖安装脚本的打包路径，预计存在相同的 `spawn-helper` 权限问题；尚未在 Intel Mac 上单独执行 PTY 测试。 |
| Linux x64 普通 TAR.GZ | 有条件可用 | 动态链接并在 Ubuntu 24.04 构建，需要 GTK3、WebKitGTK 4.1；首次由 npx 安装 DSH 时，`node-pty` 还可能需要 Python 3、make 和 C/C++ 编译器。未承诺兼容所有发行版。 |
| Linux ARM64 普通 TAR.GZ | 缺少设备验证 | 原生 runner 构建和 Web 启动检查已通过；运行库、发行版兼容性及首次安装 `node-pty` 的工具链要求与 x64 相同，尚缺少代表性 ARM64 Linux 设备反馈。 |
| Linux x64 `offline-full` | **已确认缺陷** | v0.2.4 正式 TAR.GZ 缺少 Linux 可加载的 `node-pty` 原生绑定，调用终端、Shell 或其他 PTY 功能时会失败。 |
| Linux ARM64 `offline-full` | **同构建逻辑推断存在缺陷** | 与 x64 使用相同的 `npm ci --ignore-scripts` 打包路径，预计同样没有为 Linux 构建 `node-pty` 原生绑定；尚未单独下载归档并执行设备测试。 |

上述 `offline-full` 缺陷来自跳过 npm 生命周期脚本：macOS 没有执行权限修复，Linux 没有执行原生模块构建。DSH CLI `--version`、Web HTTP 200 或页面标题检查只能证明“能够启动”，不能证明 PTY、浏览器自动化或其他原生扩展能工作。

## v0.2.5 起的 PTY 修复状态

v0.2.5 起已经加入以下修复，并在 v0.3.2 的六平台构建与最终归档检查中通过；这仍不反向改变 v0.2.4 的已发布资产。当前 main 又针对 rc.7 的依赖变化更新了门禁，但需要新的线上证据：

1. 仍使用 `npm ci --ignore-scripts` 默认禁止全部依赖脚本；
2. 在执行任何白名单脚本前，核对锁文件 integrity、已安装版本、生命周期命令和已审查脚本 SHA-256；tag runner 根据本次 DSH 锁文件精确批准 `node-pty` 与 `@deepseek-ai/dsh-subprocess-local`，不会沿用旧版本白名单；
3. 只允许 `node-pty` 的 install/postinstall 重建，以及 DSH 官方 `ensure-spawn-helper.mjs` 权限修复；
4. 在 Windows x64/ARM64、macOS Intel/ARM64、Linux x64/ARM64 原生 runner 上真实调用 `node-pty.spawn()`，启动平台 Shell、核对唯一输出标记和退出码；
5. macOS 明确检查 `spawn-helper` 可执行位；当前 main 按新依赖布局检查 Linux `prebuilds/linux-<arch>/pty.node`，Windows 检查 ConPTY 绑定和 helper，再由真实 PTY 测试证明可加载；
6. 最终 ZIP/TAR.GZ 重新解包后，使用归档内 Node 再执行同一功能测试；
7. DSH 改为接收 `--port 0`，宿主只接受其日志中公布并通过 loopback 校验的实际 URL，不再自行预占后释放端口。

v0.3.2 已达到六平台原生 CI 和最终归档检查门槛；设备安装与真实用户工作流证据仍按上表保留为缺口。

## 同类 DSH Desktop Issue 的适用边界

对 [anywhere-labs/deepseek-harness-desktop Issues](https://github.com/anywhere-labs/deepseek-harness-desktop/issues) 的审计按架构归属处理，不把 Electron 补丁直接套进 Wails：

- **本项目已规避的 Electron 专属故障**：本项目启动真实 Node，不会把 Electron `process.execPath` 当成 Node，因此不适用 [#71](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/71) 的 Electron runner 根因；离线包使用完整 npm 依赖树并做功能复测，避免 [#57](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/57) 的 staging 漏传递依赖；宿主不注入 Electron 原生目录选择能力，保留 DSH Web 浏览路径，不采用 [#29](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/29) 的 `SSH_CONNECTION=1` 伪装绕过；窗口使用原生标题栏和不透明 WebView，不复制 [#49](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/49)、[#79](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/79) 的自绘标题栏、透明和 vibrancy 组合。
- **本项目可以加固的 WebView 行为**：iframe 已显式声明剪贴板权限；v0.3.3 额外启用 Wails 默认右键菜单，为 [#12](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/12)、[#14](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/14) 一类复制/粘贴失败提供手工回退。DSH 自身复制按钮是否在每个平台成功仍需设备验证，不能仅由静态配置推断。
- **仍可能影响本项目的 DSH 上游问题**：Windows ACL 沙箱内部创建 PowerShell 的黑框和 `0xC0000142` 发生在 DSH 子进程内部，外壳只能隐藏自己直接启动的 Node/回收命令，参见 [#15](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/15)、[#84](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/84)；用户可以在 DSH 中显式选择 `danger-full-access` 作为当前用户权限下的诊断或高权限模式，但宿主不会在失败后自动放宽权限，也不会因此获得管理员令牌。MCP 延迟、重复插件注册和 profile 错误仍可能导致 readiness 前失败，参见 [#45](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/45)、[#8](https://github.com/anywhere-labs/deepseek-harness-desktop/issues/8)；本项目提供五分钟有界等待、原始日志、重试和浏览器回退，不修改 DSH 插件图。

## v0.2.4 使用建议

- Windows x64 离线环境可以优先使用 `offline-full`，仍应核对 Release 的 SHA-256。
- macOS 或 Linux 若需要终端、Shell、Bash 等依赖 PTY 的能力，请使用普通包，并准备兼容的系统 Node 和原生构建工具链。
- macOS 或 Linux 若同时要求完全离线和 PTY 功能，v0.2.4 没有可靠产物；应等待修复后的新版本。不要把 Web 页面能打开当作功能完整的证据。
- macOS 遇到 Gatekeeper 拦截时，只在已从本仓库 Release 下载并核对哈希后，通过 Finder 的“打开”或系统“隐私与安全性”确认；不要永久关闭 Gatekeeper。
- Linux 普通包在 Ubuntu/Debian 系统上的运行与原生模块构建依赖可参考：

  ```bash
  sudo apt-get install python3 make g++ libgtk-3-0t64 libwebkit2gtk-4.1-0
  ```

  其他发行版请使用等价软件包，并在 Issue 中注明发行版、版本、架构和 WebKitGTK 版本。

## 跨平台产品限制与风险

- **未签名**：Windows 无代码签名；macOS 无 Developer ID 签名和 notarization。
- **Linux 分发范围有限**：v0.5.0 增加 Ubuntu 24.04 x64/ARM64 在线 DEB，并继续提供动态链接 TAR.GZ；仍没有 AppImage、RPM 或软件仓库更新通道，也不支持旧 Ubuntu/其他发行版。
- **没有 Desktop 二进制自动更新器**：DSH 在线依赖可以在界面确认切换；没有系统 Node/npm 的 `offline-full` 用户，以及所有新的桌面安装包，仍需自行下载并校验。
- **v0.2.4 端口交接存在小窗口**：该版本让系统选择空闲端口后先关闭临时 listener，再把端口交给 DSH；当前 `main` 已改为 `--port 0` 并解析 DSH 实际公布的地址。
- **DSH 用户数据不是便携隔离数据**：宿主当前不覆盖 `DSH_HOME`，桌面端和命令行 DSH 使用上游默认用户目录；移动便携包不会自动搬走工作区和会话状态。
- **上游 Web UI 兼容性**：宿主通过 loopback iframe 承载 DSH。若上游将来增加禁止嵌入的 CSP 或 `X-Frame-Options`，内嵌页面可能失效；“在浏览器中打开”是当前回退入口。
- **离线包文件很多**：`offline-full` 包含数万依赖文件，解压、杀毒扫描、复制和首次启动会比普通包慢，占用空间也显著更大。

## 维护者的修复与发布门槛

1. 不移动已经公开的 `v0.2.4` tag；修复应发布递增的 patch 版本。
2. 默认继续跳过全部依赖脚本，只对白名单中经过版本、integrity、命令和脚本哈希审查的原生准备步骤开放执行。
3. 在每个原生 runner 上实际 `require` 并调用原生依赖；PTY 测试必须启动平台 Shell 并核对输出和退出码。
4. 测试打包前的 staging 目录后，还要重新解开最终 ZIP/TAR.GZ/安装包进行同样检查，确认权限和原生文件没有在归档阶段丢失。
5. Windows ARM64、macOS Intel/ARM64、Linux x64/ARM64 的发布说明必须分别标注设备验证证据；绿色 CI 不自动等于完整支持。

## 后续离线包候选方案

当前离线运行时包含约三万个松散依赖文件。后续版本可以评估在构建阶段把经过功能验证的运行时封装为单一 tarball，并在首次启动时：

1. 校验固定 SHA-256 和运行时版本；
2. 解压到用户缓存目录中的唯一临时目录；
3. 完整验证 Node、DSH 入口和关键原生依赖；
4. 使用同卷原子 rename 切换到最终目录；
5. 失败时保留旧的已验证运行时并清理临时目录。

这能减少安装器和杀毒软件处理数万个文件的成本，但会增加首次启动时间、缓存占用、并发锁、磁盘空间检查、损坏恢复和第三方许可证分发复杂度，因此不并入当前缺陷修复。

构建侧检查见 [开发与跨平台构建](BUILDING.md)，发布阻断条件见 [发布流程](RELEASING.md)，具体症状见 [故障排查](TROUBLESHOOTING.md)。
