# 发布流程

## 发布模型

- push/PR：运行质量门和六个原生平台构建；
- 手动运行 Release workflow：只生成 Actions artifacts，不创建 GitHub Release；
- 推送 `v*` tag：构建全部平台，成功后创建 GitHub Release；
- tag 不由 workflow 自动创建，避免错误版本触发不可逆发布。

## 版本规则

版本使用 SemVer：

- 稳定版：`1.2.3`，tag 为 `v1.2.3`；
- 预发布：`1.2.3-rc.1`，tag 为 `v1.2.3-rc.1`；
- 手动测试可使用 `1.2.3-dev`。

桌面界面版本通过 `-X main.version=...` 注入。Wails/Windows 安装包只接受数字三段元数据，因此预发布后缀会从 `productVersion` 中移除，但界面仍显示完整版本。

## 发布前检查

1. 确认工作区干净，依赖锁文件已提交；
2. 确认默认 DSH 版本确实能启动；
3. 本机至少执行基础测试与一个平台生产构建；
4. CI 六个平台全部通过；
5. 检查 README、变更记录和已知限制；
6. 确认没有 API Key、Token、Cookie、私有代理地址或构建缓存；
7. 确认签名状态。未签名时必须保留 Release 警告。

## 创建版本

版本提交和 tag 应分别审查。示例命令只作为维护说明，不代表可以跳过仓库的 Git 提交流程：

```bash
git tag -a v0.2.0 -m "Starline DSH Desktop v0.2.0"
git push origin refs/tags/v0.2.0:refs/tags/v0.2.0
```

Release workflow 会：

1. 从 tag 去掉前缀 `v`；
2. 校验版本格式；
3. 在六个原生 runner 构建；
4. 生成每个产物的 `.sha256`；
5. 汇总为 `SHA256SUMS.txt`；
6. 生成 GitHub Release notes；
7. tag 带 `-` 时标记为 prerelease。

## 产物清单

```text
starline-dsh-desktop-windows-amd64-setup.exe
starline-dsh-desktop-windows-amd64.zip
starline-dsh-desktop-windows-arm64-setup.exe
starline-dsh-desktop-windows-arm64.zip
starline-dsh-desktop-macos-amd64.zip
starline-dsh-desktop-macos-arm64.zip
starline-dsh-desktop-linux-amd64.tar.gz
starline-dsh-desktop-linux-arm64.tar.gz
*.sha256
SHA256SUMS.txt
```

## 签名与公证

当前 workflow 不包含秘密材料。正式签名需要单独设计：

- Windows：代码签名证书、受保护的签名服务或硬件密钥；
- macOS：Developer ID Application、临时 keychain、hardened runtime、notarytool；
- Linux：若提供仓库包，需要对应仓库签名，而不只是 TAR.GZ hash。

不要把证书、PFX 密码或 Apple 凭据写进仓库或普通 workflow 输出。

## 失败处理

- 单个 build job 失败：不发布 Release，先修复后重新推送新 tag；
- tag 内容错误：不要移动已公开 tag，应发布递增版本；
- 某平台暂时不支持：在代码和文档中明确移除该目标，不上传空壳产物；
- artifact 正常但 Release job 失败：可在相同 workflow run 重跑 publish job，避免重新编译；
- 发现安全问题：先将 Release 标记为 prerelease 或说明受影响范围，不自动删除历史产物。
