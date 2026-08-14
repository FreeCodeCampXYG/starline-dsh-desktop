# Contributing

感谢参与 Starline DSH Desktop。这个项目的首要约束是保持“薄宿主”：解决跨平台桌面启动问题，但不复制 DeepSeek Harness 内核。

## 在开始编码前

1. 搜索现有 Issue 和 PR；
2. Bug 请给出最小复现；功能建议先说明它为什么属于桌面宿主；
3. 大功能先开 Issue 讨论边界；
4. DSH Agent、模型、会话、插件或官方 Web UI 问题应优先提交上游。

## 开发流程

1. 从 `main` 创建主题分支；
2. 保持改动小而聚焦，不重排无关代码；
3. 新增 Go 逻辑应包含针对行为的测试；
4. 面向用户的提示和错误优先使用中文，技术标识保持英文；
5. 更新受影响的 README 或 `docs/`；
6. 不提交 `dist/`、`build/bin/`、`frontend/dist/`、任何 `node_modules/`、生成的离线 Node 文件或 Wails bindings。

## 目录职责

| 目录 | 放什么 | 不放什么 |
| --- | --- | --- |
| `internal/application` | Wails 生命周期、绑定和桌面状态 | Node 命令解析、配置文件细节 |
| `internal/config` | 设置类型、输入校验、用户目录持久化 | Wails runtime 调用 |
| `internal/launcher` | DSH 命令、代理环境、loopback、日志和进程回收 | 前端展示逻辑 |
| `frontend` | 桌面宿主拥有的启动、错误、设置和帮助 UI | DSH Agent/会话/插件实现 |
| `build` / `scripts` | 原生资源、安装器源码和可复现构建工具 | 最终二进制和临时打包目录 |

新增代码先选择拥有该行为的包。只有出现新的独立变化原因时才创建新包；不要为了“看起来分层”增加只转发一次调用的空目录。

## 必须验证

```bash
npm --prefix frontend ci
npm --prefix frontend run docs:check
npm --prefix frontend run typecheck
npm --prefix frontend run build
npm --prefix offline-runtime ci --omit=dev --ignore-scripts --workspaces=false
node offline-runtime/node_modules/@deepseek-ai/dsh/lib/bin.js --version
go test ./...
go vet ./...
```

涉及平台行为时，还应运行对应 Wails 生产构建。Linux 使用 `-tags webkit2_41`。详细命令见 [BUILDING.md](docs/BUILDING.md)。

## Pull Request

- 使用 PR 模板；
- 关联 Issue，并清楚说明动机、范围和非目标；
- 列出实际执行过的命令与平台；
- UI 变化提供前后截图或短视频；
- 涉及子进程、代理或 loopback 时说明安全影响；
- 不把“CI 应该会过”当作本地验证结果。

维护者可能要求拆分同时修改宿主、文档和无关重构的超大 PR。

## Commit

提交主题应简短具体，正文说明主要变更和影响。避免只写 `update`、`fix`、`修改` 等无法审阅的记录。

## 安全与隐私

- 不在 Issue、PR、测试夹具或日志中提交模型密钥、Token、Cookie、代理凭据；
- 安全漏洞按 [SECURITY.md](SECURITY.md) 私密报告；
- 测试进程回收时只操作测试自己创建的 PID；
- 不增加扫描和终止全局 Node/DSH 进程的逻辑。

## 许可证

提交代码、文档或其他内容，即表示你确认有权提供该贡献，并同意按项目的 [MIT License](LICENSE) 授权。除非另有书面版权转让协议，你仍保留自己贡献部分的版权。第三方代码、素材或生成文件必须保留其原始版权和许可声明，并在 PR 中说明来源；详细边界见 [版权与第三方许可说明](NOTICE.md)。

项目维护者与贡献记录入口见 [AUTHORS.md](AUTHORS.md)。
