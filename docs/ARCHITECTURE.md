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

1. Wails 创建原生窗口并加载内嵌前端资源。
2. Go 宿主读取用户代理配置。
3. 宿主优先查找与可执行文件同包的 `offline-runtime/`；macOS 在应用包 `Contents/Resources` 中查找。
4. 离线运行时必须同时包含匹配版本文件、Node 可执行文件和 DSH 入口，缺损或版本不一致时明确失败，不静默联网回退。
5. 未发现离线运行时时，宿主查找系统 Node.js 并验证版本；Windows 直接让 Node 执行 `npx-cli.js`，Unix 使用 PATH 中的 `npx`。
6. 操作系统分配可用 loopback 端口，宿主关闭临时监听并将端口传给 DSH。
7. 子进程执行固定版本的 `@deepseek-ai/dsh`。普通包使用：

   ```text
   npx --yes --package=@deepseek-ai/dsh@<version> dsh web --host 127.0.0.1 --port <port>
   ```

   `offline-full` 直接执行 `offline-runtime/node[.exe] offline-runtime/node_modules/@deepseek-ai/dsh/lib/bin.js web ...`，不调用 npm registry。

8. 宿主使用禁用代理的 HTTP client 轮询本地地址，并同时验证状态码和 DSH 页面标题。
9. 就绪后前端 iframe 导航到该 loopback URL。

## 安全边界

- iframe 只接收由宿主生成或从 DSH 日志中验证过的 loopback HTTP URL；
- 健康检查不使用系统代理，防止 loopback 请求泄漏；
- `NO_PROXY` 始终合并 `127.0.0.1`、`localhost`、`::1`；
- 自定义代理仅传递给 DSH/npm 子进程；
- 配置不存储模型 API Key；
- 日志文件权限使用用户私有权限，并最多保留最近 10 份；
- 退出时只回收宿主持有的子进程树，不按进程名全局扫描；
- 重启使用 generation 标识，旧启动任务的迟到结果不能覆盖新状态。

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

## 安装目录与路径语义

- Windows Setup 的目录页允许选择任意当前用户可写目录，并通过注册表 `InstallLocation` 记住选择；
- NSIS 源文件使用 `Unicode true`，安装验证覆盖中文和空格目录；
- Go 文件路径全部通过 `os.Executable`、`filepath` 和参数数组处理，不拼接 `cmd /c` 字符串；
- 普通包的配置和日志位于用户目录，不随安装目录移动；
- Windows/Linux 离线包要求 `offline-runtime/` 与可执行文件相邻，macOS 放在 `.app/Contents/Resources`；
- 便携包可以整体移动，但不要只移动离线版的可执行文件。

## 不在本项目中的内容

- DSH Agent 和函数式核心；
- 会话、轨迹、工具审批与插件协议；
- 模型 provider、API Key 与账号系统；
- DSH 工作区迁移；
- npm 包本身的自动更新策略。

如果功能需要复制上述模块，应先判断它是否应提交到 DSH 上游。
