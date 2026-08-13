## 变更说明

<!-- 说明改了什么、为什么改，以及没有改什么。 -->

关联 Issue：Closes #

## 变更类型

- [ ] Bug fix
- [ ] Feature
- [ ] Build / packaging
- [ ] Documentation
- [ ] Refactor without behavior change
- [ ] Breaking change

## 验证

<!-- 列出实际执行的命令和结果，不要只写“测试通过”。 -->

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `npm --prefix frontend run typecheck`
- [ ] `npm --prefix frontend run build`
- [ ] 相关平台的 Wails 生产构建

已验证平台：

- [ ] Windows x64
- [ ] Windows ARM64
- [ ] macOS Intel
- [ ] macOS Apple Silicon
- [ ] Linux x64
- [ ] Linux ARM64

## 界面或行为证据

<!-- 涉及 UI 时请提供前后截图或短视频；不涉及则说明原因。 -->

## 边界与风险

- [ ] 没有把 DSH Agent/会话/插件内核复制进桌面壳
- [ ] 没有提交 API Key、Token、Cookie、私有配置或构建产物
- [ ] 已考虑子进程回收、loopback 安全和代理变量影响
- [ ] 已同步相关文档和发布说明
