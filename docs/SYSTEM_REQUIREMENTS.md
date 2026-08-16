# 系统要求与兼容基线

本文记录当前版本线明确支持的最低系统基线。它是产品支持边界，不是对所有“理论上可能启动”环境的承诺；每个 `v*` tag 的 Release 页面会从该 tag 内的 `release-requirements.json` 固定生成同一组要求，后续版本调整不会反向改变旧版本说明。

## 当前支持基线

| 平台 | 最低系统版本 | 内核或系统构建基线 | 必要组件 |
| --- | --- | --- | --- |
| Windows x64 / ARM64 | Windows 10 22H2 或 Windows 11 | Windows 10 OS build 19045；Windows 11 使用受支持版本 | Microsoft Edge WebView2 Runtime |
| macOS Intel / Apple Silicon | macOS 13.5 或更高版本 | macOS 13.5 对应的系统内核线或更新版本 | 系统 WebKit；在线包还需满足 Node.js 要求 |
| Linux x64 / ARM64 | 仅支持 Ubuntu Desktop 24.04 LTS | Ubuntu 24.04 GA 基线为 Linux 6.8；建议使用该发行版当前受支持内核 | glibc 2.39+、GTK3、WebKitGTK 4.1 |

明确不支持 Windows 10 22H2 之前版本、macOS 13.5 之前版本、Ubuntu 22.04 及更早版本。其他 Linux 发行版即使具有较新内核，也不在当前支持范围内；因为动态链接应用是否可运行还取决于 glibc、GTK/WebKitGTK 版本和发行版打包方式，不能只看 `uname -r`。

Ubuntu 24.04 的 Linux 6.8 是项目构建和支持基线，不是从 ELF 文件推导出的通用“硬内核下限”。旧发行版升级 HWE 内核也不会自动升级 glibc 和 WebKitGTK，因此不会变成受支持环境。

## 在线包与离线包

| 包类型 | Node.js | npm / npx | 网络与原生依赖 |
| --- | --- | --- | --- |
| `online` | `22.19+` 或 `24+` | 必需 | 首次启动可能访问 npm registry；首次准备 DSH 原生模块时可能需要 Python 3、make 和 C/C++ 编译器 |
| `offline-full` | 内置固定 Node 24.19.0 | 启动 DSH 时不使用 | 不访问 npm registry，但模型服务、远程 MCP、Web 工具和更新仍可能需要网络 |

Linux `.deb` 是在线小包。包管理器会强制检查 glibc、GTK3、WebKitGTK 4.1 等系统库；Node 可能来自 nvm、Volta 或其他用户级安装，无法由 dpkg 可靠识别，所以 DEB 将 Node/npm 标记为建议组件，应用启动时仍会执行真实版本检查。

## 证据边界

- Windows、macOS、Ubuntu 24.04 六目标在各自原生 GitHub Actions runner 构建，不等同于代表性设备测试。
- Linux `.deb` 必须在 Ubuntu 24.04 x64/ARM64 原生 runner 上检查包架构、控制字段、文件布局、ELF 架构和动态库解析，并实际完成 apt 安装/卸载后才进入 Release。
- 当前产物未完成 Windows/macOS 商业签名、公证，也未完成所有平台的干净设备安装、升级和卸载验证。

上游依据包括 [Node.js 24 支持平台](https://github.com/nodejs/node/blob/v24.19.0/BUILDING.md#supported-platforms) 与 [Ubuntu 24.04 Release Notes](https://discourse.ubuntu.com/t/noble-numbat-release-notes/39890)。本项目采用更保守的产品基线：即使某个依赖单独支持旧系统，也不据此扩大整个桌面包的支持范围。
