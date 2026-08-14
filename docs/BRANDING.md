# 品牌与应用图标

## 设计含义

Starline DSH Desktop 使用项目自有的“星轨终端”图标：

- 主体的渐变 `S` 代表 **Starline**，曲线也表达任务轨迹和持续运行；
- 中央 `>_` 代表终端、开发工具和由桌面宿主启动的 DSH；
- 右上四角星代表 **Starline** 的“星线”含义；
- 深海蓝底与青蓝轨迹延续桌面宿主现有的深色界面。

图标优先保证 16、24、32 像素标题栏和任务栏尺寸下的轮廓清晰度，没有放入难以辨认的小字。

## 权利边界

图标由本项目使用几何图形和自有生成脚本原创绘制，没有复制或描摹 DeepSeek 鲸鱼、Wails `W`、Node.js 六边形或其他第三方徽标。星形、字母 `S` 和终端提示符属于通用视觉元素；当前组合、配色和图形文件作为项目自有资产随本项目按 [MIT License](../LICENSE) 提供。

这里的原创来源说明可以降低误用第三方素材的风险，但不构成对所有国家或地区商标注册结果的绝对保证。DeepSeek、Wails、Node.js 等名称与徽标仍归各自权利人所有，详细边界见 [版权与第三方许可说明](../NOTICE.md)。

## 资产与再生成

| 文件 | 用途 |
| --- | --- |
| `scripts/generate-app-icons.py` | 可审计的原创几何设计源与生成器 |
| `scripts/requirements-icons.txt` | 仅重新生成图标时需要的固定 Pillow 版本 |
| `build/appicon.png` | 1024×1024 RGBA；Wails/macOS 的主输入，并嵌入 Linux 窗口和 macOS About 面板 |
| `build/windows/icon.ico` | Windows 16/24/32/48/64/128/256 多尺寸图标 |

重新生成不会改变程序运行依赖：

```powershell
python -m pip install -r scripts/requirements-icons.txt
python scripts/generate-app-icons.py
```

Wails 在 `build/windows/icon.ico` 已存在时会直接使用它，不会根据 `appicon.png` 自动覆盖。因此生成器必须同时更新 PNG 和 ICO，提交前也必须构建一次 Windows 可执行文件验证嵌入资源。Go 宿主还会嵌入 PNG，向 Linux 窗口和 macOS About 面板传入同一图标。
