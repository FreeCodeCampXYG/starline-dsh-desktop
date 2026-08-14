# 已知问题与平台支持边界

本文是平台缺陷和验证边界的唯一说明。README、构建和排错文档只保留摘要并链接到这里，避免把推测、CI 启动成功和真实设备功能验证混在一起。

## 状态定义

| 状态 | 含义 |
| --- | --- |
| 已确认缺陷 | 已从正式 Release 归档内容或实际功能测试确认 |
| 同构建逻辑推断 | 已在一个架构确认，另一个架构使用相同打包逻辑；尚未单独执行设备测试 |
| 缺少设备验证 | 构建或 Web 启动检查成功，但没有代表性设备上的安装和功能证据 |
| 产品限制 | 当前设计或分发方式明确不覆盖的能力，不等同于回归 Bug |

## v0.2.4 平台矩阵

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

## 当前 main 的修复状态

当前源码已经加入修复候选，但不反向改变 v0.2.4 的已发布资产：

1. 仍使用 `npm ci --ignore-scripts` 默认禁止全部依赖脚本；
2. 在执行任何白名单脚本前，核对 `node-pty@1.1.0` 与 `@deepseek-ai/dsh-subprocess-local@0.1.0-rc.6` 的锁文件 integrity、已安装版本、生命周期命令和已审查脚本 SHA-256；
3. 只允许 `node-pty` 的 install/postinstall 重建，以及 DSH 官方 `ensure-spawn-helper.mjs` 权限修复；
4. 在 Windows x64/ARM64、macOS Intel/ARM64、Linux x64/ARM64 原生 runner 上真实调用 `node-pty.spawn()`，启动平台 Shell、核对唯一输出标记和退出码；
5. macOS 明确检查 `spawn-helper` 可执行位，Linux 明确检查并加载本机架构构建出的 `build/Release/pty.node`；
6. 最终 ZIP/TAR.GZ 重新解包后，使用归档内 Node 再执行同一功能测试；
7. DSH 改为接收 `--port 0`，宿主只接受其日志中公布并通过 loopback 校验的实际 URL，不再自行预占后释放端口。

只有新版本的六平台 CI 和最终归档检查全部成功后，Release notes 才能把 macOS/Linux 离线 PTY 标记为已修复。

## v0.2.4 使用建议

- Windows x64 离线环境可以优先使用 `offline-full`，仍应核对 Release 的 SHA-256。
- macOS 或 Linux 若需要终端、Shell、Bash 等依赖 PTY 的能力，请使用普通包，并准备兼容的系统 Node 和原生构建工具链。
- macOS 或 Linux 若同时要求完全离线和 PTY 功能，v0.2.4 没有可靠产物；应等待修复后的新版本。不要把 Web 页面能打开当作功能完整的证据。
- macOS 遇到 Gatekeeper 拦截时，只在已从本仓库 Release 下载并核对哈希后，通过 Finder 的“打开”或系统“隐私与安全性”确认；不要永久关闭 Gatekeeper。
- Linux 普通包在 Ubuntu/Debian 系统上的运行与原生模块构建依赖可参考：

  ```bash
  sudo apt-get install python3 make g++ libgtk-3-0 libwebkit2gtk-4.1-0
  ```

  其他发行版请使用等价软件包，并在 Issue 中注明发行版、版本、架构和 WebKitGTK 版本。

## 跨平台产品限制与风险

- **未签名**：Windows 无代码签名；macOS 无 Developer ID 签名和 notarization。
- **Linux 分发范围有限**：仅提供动态链接 TAR.GZ，没有 AppImage、DEB、RPM 或软件仓库更新通道。
- **没有自动更新器**：新版本需要用户自行下载并校验。
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

## v0.3.0 离线包候选方案

当前离线运行时包含约三万个松散依赖文件。v0.3.0 可以评估在构建阶段把经过功能验证的运行时封装为单一 tarball，并在首次启动时：

1. 校验固定 SHA-256 和运行时版本；
2. 解压到用户缓存目录中的唯一临时目录；
3. 完整验证 Node、DSH 入口和关键原生依赖；
4. 使用同卷原子 rename 切换到最终目录；
5. 失败时保留旧的已验证运行时并清理临时目录。

这能减少安装器和杀毒软件处理数万个文件的成本，但会增加首次启动时间、缓存占用、并发锁、磁盘空间检查、损坏恢复和第三方许可证分发复杂度，因此不并入当前缺陷修复。

构建侧检查见 [开发与跨平台构建](BUILDING.md)，发布阻断条件见 [发布流程](RELEASING.md)，具体症状见 [故障排查](TROUBLESHOOTING.md)。
