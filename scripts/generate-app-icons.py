#!/usr/bin/env python3
"""生成 Starline DSH Desktop 的原创跨平台应用图标。"""

from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw


SIZE = 1024
SUPERSAMPLE = 4
CANVAS_SIZE = SIZE * SUPERSAMPLE
ICO_SIZES = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]


def scaled(value: float) -> int:
    """把设计坐标转换为抗锯齿画布坐标。"""

    return round(value * SUPERSAMPLE)


def mix(start: tuple[int, int, int], end: tuple[int, int, int], ratio: float) -> tuple[int, int, int, int]:
    """在线性渐变两端之间计算一个不透明颜色。"""

    return tuple(round(a + (b - a) * ratio) for a, b in zip(start, end)) + (255,)


def horizontal_gradient(start: tuple[int, int, int], end: tuple[int, int, int]) -> Image.Image:
    """生成覆盖整个高分辨率画布的水平渐变。"""

    image = Image.new("RGBA", (CANVAS_SIZE, CANVAS_SIZE))
    draw = ImageDraw.Draw(image)
    for x in range(CANVAS_SIZE):
        ratio = x / (CANVAS_SIZE - 1)
        draw.line((x, 0, x, CANVAS_SIZE), fill=mix(start, end, ratio))
    return image


def vertical_gradient(start: tuple[int, int, int], end: tuple[int, int, int]) -> Image.Image:
    """生成覆盖整个高分辨率画布的垂直渐变。"""

    image = Image.new("RGBA", (CANVAS_SIZE, CANVAS_SIZE))
    draw = ImageDraw.Draw(image)
    for y in range(CANVAS_SIZE):
        ratio = y / (CANVAS_SIZE - 1)
        draw.line((0, y, CANVAS_SIZE, y), fill=mix(start, end, ratio))
    return image


def cubic_bezier(
    start: tuple[float, float],
    control1: tuple[float, float],
    control2: tuple[float, float],
    end: tuple[float, float],
    steps: int = 72,
) -> list[tuple[int, int]]:
    """采样一段三次贝塞尔曲线，供 Pillow 绘制平滑星轨。"""

    points: list[tuple[int, int]] = []
    for index in range(steps + 1):
        t = index / steps
        inverse = 1 - t
        x = (
            inverse**3 * start[0]
            + 3 * inverse**2 * t * control1[0]
            + 3 * inverse * t**2 * control2[0]
            + t**3 * end[0]
        )
        y = (
            inverse**3 * start[1]
            + 3 * inverse**2 * t * control1[1]
            + 3 * inverse * t**2 * control2[1]
            + t**3 * end[1]
        )
        points.append((scaled(x), scaled(y)))
    return points


def create_icon() -> Image.Image:
    """绘制星轨 S、终端提示符和四角星组成的主图标。"""

    transparent = Image.new("RGBA", (CANVAS_SIZE, CANVAS_SIZE), (0, 0, 0, 0))
    icon = transparent.copy()

    outer_mask = Image.new("L", icon.size, 0)
    ImageDraw.Draw(outer_mask).rounded_rectangle(
        (scaled(58), scaled(58), scaled(966), scaled(966)),
        radius=scaled(226),
        fill=255,
    )
    border = Image.composite(
        horizontal_gradient((92, 111, 255), (25, 211, 190)),
        transparent,
        outer_mask,
    )
    icon.alpha_composite(border)

    inner_mask = Image.new("L", icon.size, 0)
    ImageDraw.Draw(inner_mask).rounded_rectangle(
        (scaled(94), scaled(94), scaled(930), scaled(930)),
        radius=scaled(194),
        fill=255,
    )
    background = Image.composite(
        vertical_gradient((7, 15, 30), (10, 36, 49)),
        transparent,
        inner_mask,
    )
    icon.alpha_composite(background)

    path = []
    segments = [
        ((744, 270), (644, 198), (394, 188), (278, 310)),
        ((278, 310), (164, 430), (286, 510), (526, 516)),
        ((526, 516), (760, 522), (846, 616), (734, 744)),
        ((734, 744), (628, 864), (360, 848), (252, 754)),
    ]
    for segment in segments:
        points = cubic_bezier(*segment)
        path.extend(points if not path else points[1:])

    track_mask = Image.new("L", icon.size, 0)
    track_draw = ImageDraw.Draw(track_mask)
    track_width = scaled(112)
    track_draw.line(path, fill=255, width=track_width, joint="curve")
    cap_radius = track_width // 2
    for x, y in (path[0], path[-1]):
        track_draw.ellipse((x - cap_radius, y - cap_radius, x + cap_radius, y + cap_radius), fill=255)
    track = Image.composite(
        horizontal_gradient((103, 121, 255), (31, 216, 190)),
        transparent,
        track_mask,
    )
    icon.alpha_composite(track)

    prompt_draw = ImageDraw.Draw(icon)
    prompt_color = (241, 247, 255, 255)
    prompt_width = scaled(46)
    prompt_draw.line(
        [(scaled(386), scaled(428)), (scaled(480), scaled(512)), (scaled(386), scaled(596))],
        fill=prompt_color,
        width=prompt_width,
        joint="curve",
    )
    prompt_draw.line(
        [(scaled(548), scaled(602)), (scaled(688), scaled(602))],
        fill=prompt_color,
        width=prompt_width,
    )
    prompt_radius = prompt_width // 2
    for x, y in ((386, 428), (386, 596), (548, 602), (688, 602)):
        cx, cy = scaled(x), scaled(y)
        prompt_draw.ellipse(
            (cx - prompt_radius, cy - prompt_radius, cx + prompt_radius, cy + prompt_radius),
            fill=prompt_color,
        )

    star_center = (scaled(790), scaled(238))
    outer_radius = scaled(72)
    inner_radius = scaled(18)
    cx, cy = star_center
    star_points = [
        (cx, cy - outer_radius),
        (cx + inner_radius, cy - inner_radius),
        (cx + outer_radius, cy),
        (cx + inner_radius, cy + inner_radius),
        (cx, cy + outer_radius),
        (cx - inner_radius, cy + inner_radius),
        (cx - outer_radius, cy),
        (cx - inner_radius, cy - inner_radius),
    ]
    prompt_draw.polygon(star_points, fill=(239, 250, 255, 255))

    return icon.resize((SIZE, SIZE), Image.Resampling.LANCZOS)


def main() -> None:
    """生成 Wails 使用的 PNG 与 Windows 多尺寸 ICO 文件。"""

    repository_root = Path(__file__).resolve().parents[1]
    png_path = repository_root / "build" / "appicon.png"
    ico_path = repository_root / "build" / "windows" / "icon.ico"
    ico_path.parent.mkdir(parents=True, exist_ok=True)

    icon = create_icon()
    icon.save(png_path, format="PNG", optimize=True)
    icon.save(ico_path, format="ICO", sizes=ICO_SIZES, bitmap_format="png")
    print(f"已生成：{png_path}")
    print(f"已生成：{ico_path}")


if __name__ == "__main__":
    main()
