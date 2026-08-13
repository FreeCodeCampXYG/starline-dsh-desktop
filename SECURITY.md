# Security Policy

## Supported versions

项目仍处于早期开发阶段。安全修复默认只面向最新发布版本和 `main` 分支，不承诺维护旧版本分支。

## Reporting a vulnerability

请不要为安全漏洞创建公开 Issue。使用 GitHub 仓库的 **Security → Report a vulnerability** 私密提交，并尽量提供：

- 受影响版本和操作系统；
- 可重复的最小步骤；
- 可能影响的数据、权限或进程边界；
- 去除 API Key、Token、Cookie 和私人路径后的日志；
- 如已知，可提供缓解建议。

本项目不会要求你发送模型密钥或 GitHub 凭据。

## Security boundary

Starline DSH Desktop 负责启动本机 `dsh web`，并在系统 WebView 中打开 loopback 页面。它不会实现 DSH 的认证、模型调用或插件权限模型。与官方 DSH 内核相关的漏洞也应同步报告给 [deepseek-ai/deepseek-harness](https://github.com/deepseek-ai/deepseek-harness/security)。
