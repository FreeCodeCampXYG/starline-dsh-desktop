# 故障排查

## 日志位置

在应用中选择 **桌面工具 → 打开日志目录**。默认位置由系统用户缓存目录决定：

- Windows：通常为 `%LOCALAPPDATA%\starline-dsh-desktop\logs`；
- macOS：通常位于用户 Library/Caches；
- Linux：通常位于 `~/.cache/starline-dsh-desktop/logs`。

应用最多保留最近 10 份日志。分享日志前删除 API Key、Token、Cookie、用户名和私人路径。

## 找不到 Node.js

终端执行：

```bash
node --version
npx --version
```

需要 Node `22.19+` 或 `24+`。Node 23 不在 DSH 支持范围内。图形应用继承的 PATH 可能与终端不同；macOS/Linux 从桌面启动时尤其需要确认 Node 安装在系统可见位置。

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
