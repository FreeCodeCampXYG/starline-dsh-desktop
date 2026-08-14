# 故障排查

## 日志位置

在应用中选择 **桌面工具 → 打开日志目录**。默认位置由系统用户缓存目录决定：

- Windows：通常为 `%LOCALAPPDATA%\starline-dsh-desktop\logs`；
- macOS：通常位于用户 Library/Caches；
- Linux：通常位于 `~/.cache/starline-dsh-desktop/logs`。

应用最多保留最近 10 份日志。分享日志前删除 API Key、Token、Cookie、用户名和私人路径。

## Windows 自定义安装目录

Setup 向导的“选择安装位置”页面可以改目录。当前用户安装默认位于 `%LOCALAPPDATA%\Programs\Starline DSH Desktop`，也可以选择其他当前用户可写位置。安装器会记录所选 `InstallLocation`，以后再次安装或升级时继续使用。

中文和空格路径受支持。若自定义路径启动失败，请检查：

1. 目录是否允许当前用户写入；`Program Files` 通常需要管理员权限，不适合当前用户安装包；
2. 安全软件是否阻止该目录中的未签名程序；
3. 离线便携版的 `offline-runtime/` 是否仍与 EXE 相邻；
4. 是否只移动了 EXE，而没有移动完整离线目录；
5. 日志中的 Node/DSH 路径是否完整，分享前先脱敏。

配置和日志保存在系统用户目录，不依赖安装位置；更换安装目录不会自动搬动 DSH 自己的工作区数据。

当前自动化证据覆盖本地 NTFS 中文和空格路径；UNC 网络共享、映射盘断连和超长路径尚未作为支持目标验证。

## 找不到 Node.js

终端执行：

```bash
node --version
npx --version
```

需要 Node `22.19+` 或 `24+`。Node 23 不在 DSH 支持范围内。图形应用继承的 PATH 可能与终端不同；macOS/Linux 从桌面启动时尤其需要确认 Node 安装在系统可见位置。

如果电脑不能安装 Node 或访问 npm，请下载与操作系统及架构一致的 `offline-full` 包。离线包正常启动时，桌面顶部会显示“离线运行时”。

## 离线运行时损坏或版本不一致

不要只复制桌面可执行文件。Windows/Linux 需要让 `offline-runtime/` 与可执行文件保持在同一目录；macOS 运行时位于应用包的 `Contents/Resources/offline-runtime`。

出现缺少 `dsh-version.txt`、Node、DSH 入口或版本不匹配时，请重新解压完整离线资产。宿主会明确失败，不会静默回退到 npm 下载。设置 `DSH_DESKTOP_DSH_VERSION` 覆盖版本时，也必须与离线包版本完全一致。

## npm 下载失败

1. 检查 npm registry：`npm config get registry`；
2. 在终端直接运行固定版本 DSH；
3. 若使用 VPN 本机端口，在桌面工具中选择自定义 HTTP 代理；
4. 不要把 SOCKS5 地址填入当前 HTTP(S) 代理输入框；
5. 查看 npm 自己的调试日志路径，Starline 日志会打印该位置。

```bash
npx --yes --package=@deepseek-ai/dsh@0.1.0-rc.6 dsh web
```

## DSH 启动后页面不就绪

宿主要求页面返回 HTTP 200 且包含 DSH 标题。常见原因：

- npm 进程仍在首次安装依赖；
- DSH 上游版本改变了启动行为；
- 安全软件阻止 Node 监听 loopback；
- 选中的端口在关闭临时 listener 后被其他进程抢占；
- DSH 进程提前退出。

请保留完整启动日志并重试一次。如果浏览器直接运行 DSH 正常而桌面端失败，向本仓库报告。

## 代理改完没有生效

保存设置会重启 DSH。确认日志时间已经变化，并检查：

- 模式是否为“自定义代理”；
- 地址是否是 HTTP(S)，例如 `http://127.0.0.1:10808`；
- 代理客户端是否允许来自 Node.js 的连接；
- npm registry 与模型服务是否都能通过该代理访问。

`offline-full` 不访问 npm registry，但代理设置仍会传递给 DSH，用于模型 provider、远程 MCP、Web 工具等网络功能。

## Windows 黑框闪烁

当前版本使用 `CREATE_NO_WINDOW` 和隐藏窗口标志运行 Node 检查、DSH 与进程树回收。如果仍出现：

1. 确认使用最新 Release；
2. 说明是启动、重启还是关闭时出现；
3. 提供 Windows 版本和安装方式；
4. 检查是否由杀毒软件、终端配置或其他 Node shim 创建。

## macOS 无法打开

未签名/未 notarize 的构建可能被 Gatekeeper 阻止。开发者可在本机重新构建。不要建议普通用户永久关闭 Gatekeeper；正式分发应完成签名和公证。

## Linux 缺少共享库

检查：

```bash
ldd ./starline-dsh-desktop | grep 'not found'
```

安装 GTK3 和 WebKitGTK 4.1 的运行库。不同发行版包名不同，Issue 中请注明发行版、版本和架构。

## 应该向哪个仓库报告

| 问题 | 仓库 |
| --- | --- |
| Node 检测、代理、窗口、日志、安装包、进程残留 | Starline DSH Desktop |
| Agent 回复、模型 provider、会话、轨迹、插件、官方 UI | deepseek-harness |
| Wails 本身的 WebView/原生窗口最小复现 | wails |

提交前尽量在浏览器直接运行 `dsh web`，这一步能快速判断问题属于宿主还是上游。
