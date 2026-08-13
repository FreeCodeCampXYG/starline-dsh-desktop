# 架构说明

## 目标

Starline DSH Desktop 是 DeepSeek Harness 的宿主层，不是 DSH 的分支或替代实现。目标是让官方 Web UI 以桌面应用方式可靠启动，同时把平台相关工作限制在小而明确的范围内。

## 组件

```text
Wails application
├─ Go lifecycle host
│  ├─ Node/npm environment validation
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
3. 宿主查找 Node.js，执行 `node --version` 并验证受支持版本。
4. Windows 直接让 Node 执行 `npx-cli.js`，避免 `.cmd`/PowerShell shim 和控制台窗口差异；Unix 使用 PATH 中的 `npx`。
5. 操作系统分配可用 loopback 端口，宿主关闭临时监听并将端口传给 DSH。
6. 子进程执行固定版本的 `@deepseek-ai/dsh`：

   ```text
   npx --yes --package=@deepseek-ai/dsh@<version> dsh web --host 127.0.0.1 --port <port>
   ```

7. 宿主使用禁用代理的 HTTP client 轮询本地地址，并同时验证状态码和 DSH 页面标题。
8. 就绪后前端 iframe 导航到该 loopback URL。

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
| DSH/npm 数据 | 由 DSH 和 npm 自己管理 |

配置写入先完成临时文件，再替换目标。Windows 替换旧配置前需要显式移除旧文件，这是标准库跨平台语义差异。

## 不在本项目中的内容

- DSH Agent 和函数式核心；
- 会话、轨迹、工具审批与插件协议；
- 模型 provider、API Key 与账号系统；
- DSH 工作区迁移；
- npm 包本身的自动更新策略。

如果功能需要复制上述模块，应先判断它是否应提交到 DSH 上游。
