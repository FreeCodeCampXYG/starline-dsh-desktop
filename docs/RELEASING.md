# 发布流程

> v0.3.2 已通过六平台原生 CI、最终离线归档功能复测并公开发布，但仍不代表代表性设备上的安装、首次启动、升级和卸载全部通过。证据边界与 v0.2.4 历史缺陷见 [已知问题与平台支持边界](KNOWN_ISSUES.md)；任何修复都发布递增版本，不移动已有公开 tag。

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
3. 确认 `offline-runtime/package-lock.json` 与 `main.go` 默认 DSH 版本一致；
4. 在每个目标原生 runner 上让包内 `node-pty` 实际启动平台 Shell，并验证输出和退出码；仅有 CLI/Web smoke test 不通过此门槛；
5. 检查原生扩展平台/架构、macOS 可执行位以及 Linux `pty.node`，并对重新解开的最终归档重复检查；
6. 本机至少执行基础测试与一个平台生产构建；
7. CI 六个平台全部通过，且没有把“构建成功”代替上述功能测试；
8. 检查 README、变更记录和 [已知问题](KNOWN_ISSUES.md)，分别写明已确认、同逻辑推断和缺少设备验证的范围；
9. 确认没有 API Key、Token、Cookie、私有代理地址或构建缓存；
10. 确认签名状态。未签名时必须保留 Release 警告。

## 创建版本

版本提交和 tag 应分别审查。示例命令只作为维护说明，不代表可以跳过仓库的 Git 提交流程：

```bash
git tag -a v0.3.0 -m "Starline DSH Desktop v0.3.0"
git push origin refs/tags/v0.3.0:refs/tags/v0.3.0
```

Release workflow 会：

1. 从 tag 去掉前缀 `v`；
2. 校验版本格式；
3. 在六个原生 runner 构建；
4. 默认禁止全部依赖脚本，只在完整校验后执行白名单中的 `node-pty` 重建和 DSH 官方 helper 权限修复；
5. 使用包内 Node 真实启动平台 Shell，并执行 DSH CLI/Web smoke test；
6. 同时生成轻量普通包和独立 `offline-full` 包；
7. 重新解开最终离线 ZIP/TAR.GZ，再次检查原生文件、权限和真实 PTY；
8. 生成每个产物的 `.sha256`；
9. 确认普通包、离线包和安装目录均包含项目 `LICENSE`、`NOTICE.md` 与 `AUTHORS.md`（macOS 位于应用包 `Contents/Resources/licenses/`）；
10. 汇总为 `SHA256SUMS.txt`，校验预期资产、独立校验文件与合并清单完全一致；
11. 从 `CHANGELOG.md` 的当前版本段生成 Release 正文，并按平台显示下载用途与实际体积；缺少版本日志时停止发布；
12. 创建 GitHub Release；tag 带 `-` 时标记为 prerelease。

## 产物清单

```text
starline-dsh-desktop-v<version>-windows-x64-setup-online.exe
starline-dsh-desktop-v<version>-windows-x64-portable-online.zip
starline-dsh-desktop-v<version>-windows-x64-portable-offline-full.zip
starline-dsh-desktop-v<version>-windows-arm64-setup-online.exe
starline-dsh-desktop-v<version>-windows-arm64-portable-online.zip
starline-dsh-desktop-v<version>-windows-arm64-portable-offline-full.zip
starline-dsh-desktop-v<version>-macos-intel-x64-app-online.zip
starline-dsh-desktop-v<version>-macos-intel-x64-app-offline-full.zip
starline-dsh-desktop-v<version>-macos-apple-silicon-arm64-app-online.zip
starline-dsh-desktop-v<version>-macos-apple-silicon-arm64-app-offline-full.zip
starline-dsh-desktop-v<version>-linux-x64-portable-online.tar.gz
starline-dsh-desktop-v<version>-linux-x64-portable-offline-full.tar.gz
starline-dsh-desktop-v<version>-linux-arm64-portable-online.tar.gz
starline-dsh-desktop-v<version>-linux-arm64-portable-offline-full.tar.gz
*.sha256
SHA256SUMS.txt
```

命名语法固定为 `<产品>-v<版本>-<系统>-<CPU>-<形态>-<联网模式>.<扩展名>`。其中 `x64` 与 `arm64` 明确处理器架构，`setup`/`portable`/`app` 明确安装形态，`online` 与 `offline-full` 明确是否携带完整 Node/DSH 运行时。不要再用体积或模糊的 `amd64`/`arm64` 后缀让用户自行猜测用途。

Release 正文不是 GitHub 自动生成的单行 `Full Changelog`。`scripts/generate-release-notes.mjs` 必须读取与 tag 完全相同的 `CHANGELOG.md` 版本段，检查以上 14 个主资产及其校验文件，并以实际字节数生成分平台下载表。发布前应检查下载表、版本变化和签名警告是否完整显示。

普通包保持原有小体积。Windows x64 v0.2.4 参考值为普通 ZIP 约 4.3 MiB、Setup 约 6.0 MiB、完整离线 ZIP 约 113.6 MiB；完整离线包解压后超过 350 MiB。平台、架构与依赖选择不同，Release 前必须以实际资产大小为准。GitHub 单文件资产上限不是当前瓶颈，但六个平台会明显增加 Actions 时间、缓存和 Release 存储。

`offline-full` 包含数万个松散依赖文件。发布检查必须记录最终归档的文件数、解压后体积、关键原生文件和权限；不能只记录压缩包下载大小。普通包与离线包应分别说明网络、系统 Node 和原生构建工具链要求。

MIT License 要求分发副本保留版权与许可声明。Windows Setup 会把 `LICENSE.txt`、`NOTICE.md` 与 `AUTHORS.md` 安装到应用目录；Windows/Linux 压缩包在根目录携带 `LICENSE`、`NOTICE.md` 与 `AUTHORS.md`；macOS 应用包在 `Contents/Resources/licenses/` 中携带对应文件。`offline-full` 还必须保留 Node.js、DeepSeek Harness 和 npm 依赖自身随包提供的许可证文件。发布前应抽查解包内容，不能只检查仓库根目录。

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

## v0.3.0 离线运行时归档候选

把数万个依赖文件封装为单一 tarball 只能作为独立版本设计，不能与当前 PTY 缺陷修复混发。方案至少需要固定 SHA-256、同卷临时目录、并发锁、磁盘空间预检、原子 rename、失败回滚、旧缓存清理、首次启动进度和完整许可证保留。具体边界见 [已知问题与平台支持边界](KNOWN_ISSUES.md#v030-离线包候选方案)。
