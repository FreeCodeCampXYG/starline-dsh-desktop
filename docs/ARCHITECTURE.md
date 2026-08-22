# 架构说明

## 目标

Starline DSH Desktop 是 DeepSeek Harness 的宿主层，不是 DSH 的分支或替代实现。目标是让官方 Web UI 以桌面应用方式可靠启动，同时把平台相关工作限制在小而明确的范围内。

## 工程原则

- **关注点分离（Separation of Concerns）**：Wails 绑定、用户配置、DSH 进程和前端视图分别维护；
- **单一职责（SRP）**：文件按独立变化原因拆分，例如运行时发现、代理环境、日志和网络边界互不混放；
- **依赖方向**：根 `main.go` 只组合依赖，`internal/application` 调用 `config` 与 `launcher`，底层包不反向依赖 Wails 入口；
- **KISS / YAGNI**：没有为当前不存在的数据库、插件内核或自动更新建立空接口和仓储层；
- **Fail Loud**：离线运行时缺损、版本冲突、路径或配置不可用时明确失败，不静默改变运行模式。

## 项目结构

```text
main.go                         Wails 组合入口、前端嵌入、版本注入
internal/
├─ application/                Wails 生命周期、绑定方法、系统菜单、状态机
├─ config/                     用户设置、校验和持久化
└─ launcher/                   DSH 子进程边界
   ├─ launcher.go              启动、就绪、停止和进程状态
   ├─ runtime.go               包内运行时/系统 Node 解析
   ├─ environment.go           代理变量与 NO_PROXY
   ├─ network.go               loopback URL 和端口边界
   ├─ logs.go                  用户日志、轮转和文件管理器入口
   └─ process_*.go             平台进程组与隐藏窗口实现
frontend/                      桌面宿主自有 UI
offline-runtime/               离线依赖声明和锁文件；生成内容不入库
build/                         图标、manifest、NSIS 安装器源码
scripts/                       构建、离线运行时和验证脚本
docs/                          架构、构建、发布、排错与参考
```

根目录保留 `main.go` 是因为 Go `embed` 不能通过 `..` 嵌入上级目录的 `frontend/dist`。它只承担组合入口，不放业务规则。

## 组件

```text
Wails application
├─ internal/application lifecycle adapter
│  ├─ offline-runtime discovery or Node/npm validation
│  ├─ proxy environment composition
│  ├─ loopback port allocation
│  ├─ child process lifecycle
│  └─ log retention and diagnostics
├─ local frontend
│  ├─ startup/error screen
│  ├─ proxy settings dialog
│  ├─ help and desktop tools menu
│  └─ DSH iframe
└─ official @deepseek-ai/dsh process
   └─ dsh web on 127.0.0.1:<dynamic-port>
```

## 启动序列

1. Wails 创建隐藏的原生窗口并加载内嵌前端资源；800 毫秒兜底计时器保证前端异常或 DSH 较慢时窗口仍会主动显示。
2. Go 宿主读取用户代理配置。
3. 前端显示由 Go 报告的单调阶段百分比；百分比对应运行时检测、Node 校验、子进程启动、监听地址和 HTTP 指纹等可验证里程碑，不等同于 npm 下载字节。
4. 宿主优先查找与可执行文件同包的 `offline-runtime/`；macOS 在应用包 `Contents/Resources` 中查找。
5. 离线运行时必须同时包含匹配版本文件、Node 可执行文件和 DSH 入口，缺损或版本不一致时明确失败，不静默联网回退。
6. 未发现离线运行时时，宿主查找系统 Node.js 并验证版本；Windows 直接让 Node 执行 `npx-cli.js`，Unix 使用 PATH 中的 `npx`。npm 请求单次等待 10 秒、最多重试 1 次；普通包准备固定优先 `registry.npmmirror.com`，自定义端口不可达时快速直连镜像。
7. 宿主向 DSH 传入 `--port 0`，由 DSH 保持 listener 所有权并选择可用 loopback 端口；宿主不再预占后释放端口。
8. 子进程执行精确版本的 `@deepseek-ai/dsh`：默认使用 Desktop 发布时验证的固定版本；前端启动后通过 Go 后端直连国内镜像查询 `latest`/`next`，只显示通道状态；用户手动刷新或应用更新时按自定义代理、应用启动时继承的系统环境代理、国内镜像直连依次查询。普通包准备沿用同一有界降级，并在日志和状态中提示实际路径。在线包经用户确认后再次核对 dist-tag、保存对应精确版本，并可清除设置恢复默认；新版本在默认 90 秒（可在设置中调整为 30–600 秒）内无法完成本地页面指纹校验时，宿主恢复更新前的配置并自动重启旧版本。这个等待上限只保护 npm 运行时准备和本地 DSH Web 就绪，DeepSeek Harness 内部远程模型/API 请求仍由 DSH 自己处理。普通包使用：

   ```text
   npx --yes --package=@deepseek-ai/dsh@<version> dsh web --host 127.0.0.1 --port 0 --no-open
   ```

   宿主传入 `--no-open` 关闭官方 Web 表层默认的系统浏览器交接；页面 URL 仍会打印并由宿主校验后交给内嵌 iframe，用户需要时可通过“在浏览器中打开”手动回退。

   `offline-full` 直接执行 `offline-runtime/node[.exe] offline-runtime/node_modules/@deepseek-ai/dsh/lib/bin.js web ...`，不调用 npm registry，也不允许界面原地替换包内依赖闭包。

9. 宿主从 DSH 输出的 `dsh web: http://...` 行解析实际地址，只接受通过 loopback 安全校验的 HTTP URL，再使用禁用代理的 HTTP client 轮询并验证状态码和 DSH 页面标题。
10. 宿主只在该 DSH 子进程的 `PATH` 前置临时命令目录；Agent 调用 `dsh plugin` 且完全遗漏 `--profile` 时，兼容入口按当前桌面 profile 补为 `web`，再转发给同一固定版本运行时。显式 profile、其他 DSH 命令、系统 PATH 和全局 npm shim 均不修改，DSH 退出后临时目录删除。
11. 就绪后前端先在遮罩后让 iframe 导航到该 loopback URL；iframe `load` 后主动显示原生窗口并淡入页面。启动失败或进程意外退出时跳过等待，立即显示顶部错误状态栏。

## 安全边界

- iframe 只接收由宿主生成或从 DSH 日志中验证过的 loopback HTTP URL；
- iframe 只显式开放剪贴板读写权限，不开放任意外部页面的 Wails 绑定；
- 健康检查不使用系统代理，防止 loopback 请求泄漏；
- `NO_PROXY` 始终合并 `127.0.0.1`、`localhost`、`::1`；
- 自定义代理用于手动 registry 检查并传递给 DSH/npm 子进程；不可达时先尝试启动应用时继承的 HTTP(S) 环境代理，再直连国内 npm 镜像；镜像只影响 npm 运行时元数据和包下载，不替代 DeepSeek Harness 的模型/API 服务地址；
- 配置不存储模型 API Key；
- 日志文件权限使用用户私有权限，并最多保留最近 10 份；
- 退出时只回收宿主持有的子进程树，不按进程名全局扫描；
- 重启和进度事件共用 generation 标识，旧启动任务的迟到结果或百分比不能覆盖新状态；单个进程的阶段也只能单调前进。

## 窗口与托盘生命周期

- Windows 使用 `HideWindowOnClose`：窗口右上角 X 只隐藏主窗口，应用和 DSH 子进程继续运行。
- `internal/application/tray_windows.go` 仅在 Windows 编译第三方托盘依赖，提供“显示窗口”“重启 DSH”和“退出”。
- Windows 只有显式“退出”会设置 `quitRequested` 并调用 `runtime.Quit`，随后进入 `OnShutdown`，确保 DSH/Node 进程树和托盘句柄一起回收。
- macOS/Linux 使用 `tray_unix.go` 空实现并允许原生关闭窗口直接退出，避开 macOS `AppDelegate` 重名和 Linux 无显示 CI 的 GTK 初始化问题。后续如引入与 Wails 原生菜单兼容的托盘实现，再恢复对应平台的隐藏行为。
- 正式构建启用系统 WebView 默认右键菜单，为选择、复制和粘贴提供平台原生回退；不注入自定义菜单脚本到跨源 DSH iframe。

## Web 与原生能力边界

宿主刻意把 DSH 当作普通 loopback Web 应用，不设置 Electron 平台查询参数，不向页面注入 Electron preload，也不声明 DSH 的 native directory-picker capability。因此工作区选择、插件 UI 和剪贴板按钮继续遵循上游 Web profile；Starline 只提供 iframe 剪贴板权限、默认右键菜单和“在浏览器中打开”回退。这样避免把 Electron `process.execPath` 当 Node、原生目录选择 FFI 崩溃、自绘标题栏/插件按钮重叠等同类桌面端问题引入 Wails，同时意味着无法由外壳直接修补跨源 iframe 内的 DSH 业务代码。

## 进程回收

- Windows：DSH 使用独立 process group 和 `CREATE_NO_WINDOW`；关闭时对已记录 PID 执行隐藏的 `taskkill /T /F`。
- macOS/Linux：子进程进入独立 process group，关闭时只向该进程组发送终止信号。

回收策略的目标不是优雅管理所有 DSH 实例，而是确保当前窗口拥有的子树不残留、不误伤其他实例。

## 配置与日志

| 类型 | 位置 |
| --- | --- |
| 配置 | `os.UserConfigDir()/starline-dsh-desktop/settings.json` |
| 日志 | `os.UserCacheDir()/starline-dsh-desktop/logs/` |
| DSH/npm 数据 | 普通包由 DSH/npm 管理；离线包运行时只读随包提供，用户数据仍在 DSH 自身目录 |

配置写入先完成临时文件，再替换目标。Windows 替换旧配置前需要显式移除旧文件，这是标准库跨平台语义差异。

配置文件旁边保留 `settings.json.lock` 作为跨进程原子写锁：Windows 使用本机文件锁，macOS/Linux 使用 `flock`。保存时会在锁内重新读取磁盘配置并与当前实例的旧值比较；如果其他实例已经保存，当前写入返回冲突而不是覆盖。该锁只保护共享配置，不限制多个 DSH Web 实例同时运行。

宿主当前不设置 `DSH_HOME`。桌面端与命令行 DSH 因而遵循上游默认用户数据目录，共享状态；`offline-full` 的“便携”只表示程序和运行时可以整体移动，不表示工作区、会话和账户配置也随包移动。

启动时若现有 `web` Profile 的 `node_modules/.modules.yaml` 已记录 pnpm Store，宿主只在该 DSH 子进程内设置对应的 `npm_config_store_dir`。这不会修改全局 pnpm 配置；目的是防止其他项目改变全局 `store-dir` 后，DSH Market 对既有 Profile 执行插件安装或更新时触发 Store 位置冲突。

## DSH 更新边界

- `internal/updater` 在前端启动后直连国内镜像读取固定 DSH dist-tags 端点；用户手动刷新时按自定义代理、系统环境代理和直连依次尝试国内镜像与官方 registry。请求限制响应大小、校验 `latest`/`next` SemVer、拒绝非受信 registry 重定向，并在结束后关闭 HTTP 空闲连接；检查本身不运行 npm、不下载软件包，失败也不阻塞 DSH。
- 在线包应用更新时由后端再次查询指定的 `latest` 或 `next`，只把返回的精确版本写入用户配置，然后同步回收宿主持有的旧 DSH 子进程树并通过既有 npx 启动链重启；前端不能提交任意包名、版本或其他通道。
- 环境变量 `DSH_DESKTOP_DSH_VERSION` 继续作为显式开发覆盖且优先级最高；存在覆盖时，界面不修改实际版本。
- `offline-full` 默认以包内 `dsh-version.txt` 为准；用户确认在线新版本后，设置中的精确版本会让宿主改用系统 Node/npm，通过既有代理降级链下载而不改写包内依赖闭包，失败时可清除设置回到离线版本。没有系统 Node/npm 时，离线升级仍是新的 Desktop Release 和六平台原生依赖门禁。
- 当前 main 把下一轮离线闭包固定为 rc.2；“最新”表示发布时审查并锁定的精确版本，不表示离线包在用户设备上跟随 npm `latest` 漂移。
- 仓库 Dependabot 每周只检查 `offline-runtime` 的官方 DSH 直接依赖并提出 PR；它不自动合并、不发布，也不能代替原生 CI、最终归档和设备验证。

## 安装目录与路径语义

- Windows Setup 的目录页允许选择任意当前用户可写目录，并通过注册表 `InstallLocation` 记住选择；同一应用身份和架构的后续 Setup 复用该目录并覆盖程序文件，用户数据仍保留在安装目录之外；
- NSIS 源文件使用 `Unicode true`，安装验证覆盖中文和空格目录；
- Go 文件路径全部通过 `os.Executable`、`filepath` 和参数数组处理，不拼接 `cmd /c` 字符串；
- 普通包的配置和日志位于用户目录，不随安装目录移动；
- Windows/Linux 离线包要求 `offline-runtime/` 与可执行文件相邻，macOS 放在 `.app/Contents/Resources`；
- `offline-full` 当前是便携归档而不是独立安装器；升级采用关闭旧进程、解压新目录、验证后移除旧目录的边界，不对运行中的依赖树做原地覆盖；
- 便携包可以整体移动，但不要只移动离线版的可执行文件。

## 不在本项目中的内容

- DSH Agent 和函数式核心；
- 会话、轨迹、工具审批与插件协议；
- 模型 provider、API Key 与账号系统；
- DSH 工作区迁移；
- 无人值守地自动安装或自动降级 npm 包。

如果功能需要复制上述模块，应先判断它是否应提交到 DSH 上游。

## 当前兼容性边界

- 官方 DSH Web UI 通过 iframe 承载；若上游将来启用禁止嵌入的 CSP 或 `X-Frame-Options`，需要改用浏览器回退或与上游协调；
- Web/CLI 启动检查不等于 Node 原生扩展或工具调用检查；平台缺陷、离线包 PTY 状态和设备证据统一记录在 [已知问题与平台支持边界](KNOWN_ISSUES.md)。
- 离线门禁实际生成一张 Sharp PNG、通过 Koffi 加载平台动态库并调用本机进程 ID 函数、运行 ripgrep，并启动真实 PTY；Windows 还加载 `ole32.dll`，防止只 `require` 模块却遗漏运行时 FFI 崩溃。

## 工具与操作系统权限

- Wails 宿主只负责启动 DSH，不为 DSH 工具提权，也不移除上游审批或工作区策略；
- `node-pty` 提供终端传输能力，不等同于管理员、UAC、root、全盘文件访问或沙箱逃逸能力；
- DSH 子进程继承当前桌面用户的操作系统权限和经过代理设置处理的环境变量；系统 ACL、macOS 隐私权限、Linux 文件模式和安全软件仍然生效；
- 官方 DSH 页面仍可由用户显式选择 `danger-full-access`，以当前操作系统用户权限执行更广泛的 PowerShell/文件操作；宿主不自动切换权限模式，也不会把该模式等同于 Windows UAC 管理员权限；
- 宿主只回收自己启动的 DSH 进程树，不以更高权限扫描或终止其他用户进程。
