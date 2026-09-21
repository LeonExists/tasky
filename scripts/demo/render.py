"""Render model-captured Tasky frames into a compact, readable GIF.

The app text is read directly from frames.json. This file only adds a
presentation frame around it: terminal chrome, colors, and the caption strip.
"""

from __future__ import annotations

import argparse
import json
import re
import textwrap
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


WIDTH = 1000
HEIGHT = 740
OUTER = "#0b0e16"
TERMINAL = "#101826"
TERMINAL_EDGE = "#26324a"
TEXT = "#eef3fc"
MUTED = "#8792aa"
CYAN = "#66e3da"
PINK = "#e899c6"
GREEN = "#9be3b3"
YELLOW = "#f5c77b"

ANSI_SGR = re.compile(r"\x1b\[[0-9;]*m")


def find_font(*names: str) -> Path:
    candidates = [
        Path("C:/Windows/Fonts") / name
        for name in names
    ]
    candidates.extend(
        [
            Path("/System/Library/Fonts/Menlo.ttc"),
            Path("/System/Library/Fonts/Monaco.ttf"),
            Path("/usr/share/fonts/truetype/dejavu") / "DejaVuSansMono.ttf",
            Path("/usr/share/fonts/truetype/liberation2") / "LiberationMono-Regular.ttf",
        ]
    )
    for candidate in candidates:
        if candidate.exists():
            return candidate
    raise FileNotFoundError("Could not find a monospaced font")


def load_frames(path: Path) -> list[dict]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not data:
        raise ValueError(f"{path} contains no frames")
    for frame in data:
        if not frame.get("text"):
            raise ValueError("Every frame needs non-empty model text")
        if int(frame.get("duration_ms", 0)) <= 0:
            raise ValueError("Every frame needs a positive duration_ms")
    return data


def clean_ansi(value: str) -> str:
    return ANSI_SGR.sub("", value).replace("\r", "")


def wrapped_lines(value: str, font: ImageFont.FreeTypeFont, max_width: int) -> list[tuple[str, bool]]:
    """Return (line, is_help) pairs while preserving the model's visible text."""
    result: list[tuple[str, bool]] = []
    for original in clean_ansi(value).splitlines():
        is_help = original.startswith(("v:", "enter:", "Delete group"))
        if not original:
            result.append(("", is_help))
            continue
        if font.getlength(original) <= max_width:
            result.append((original, is_help))
            continue
        char_width = max(font.getlength("M"), 1)
        width_chars = max(int(max_width / char_width), 1)
        chunks = textwrap.wrap(
            original,
            width=width_chars,
            break_long_words=False,
            break_on_hyphens=False,
            replace_whitespace=False,
            drop_whitespace=True,
        )
        result.extend((chunk, is_help) for chunk in chunks)
    return result


def draw_segmented_line(
    draw: ImageDraw.ImageDraw,
    x: int,
    y: int,
    line: str,
    font: ImageFont.FreeTypeFont,
    help_line: bool,
) -> None:
    if help_line:
        draw.text((x, y), line, font=font, fill=MUTED)
        return

    fill = TEXT
    if line.strip() in {"Tasks", "Groups"} or line.strip() == "Tasky":
        fill = CYAN
    elif line.startswith("[") or "[Launch]" in line:
        fill = CYAN
    draw.text((x, y), line, font=font, fill=fill)

    leading = len(line) - len(line.lstrip(" "))
    if line.lstrip().startswith(">"):
        prefix_x = x + draw.textlength(line[:leading], font=font)
        draw.text((prefix_x, y), ">", font=font, fill=PINK)

    if "(disabled)" in line:
        disabled_x = x + draw.textlength(line[: line.index("(disabled)")], font=font)
        draw.text((disabled_x, y), "(disabled)", font=font, fill=PINK)

    if "[x]" in line:
        done_start = x + draw.textlength(line[: line.index("[x]")], font=font)
        done_end = done_start + draw.textlength("[x]", font=font)
        draw.line(
            (done_start, y + font.size // 2, done_end + draw.textlength(line[line.index("[x]") + 3 :], font=font), y + font.size // 2),
            fill=MUTED,
            width=max(font.size // 12, 1),
        )


def render_frame(
    model_text: str,
    caption: str,
    pressed: str,
    regular: ImageFont.FreeTypeFont,
    bold: ImageFont.FreeTypeFont,
) -> Image.Image:
    image = Image.new("RGB", (WIDTH, HEIGHT), OUTER)
    draw = ImageDraw.Draw(image)

    terminal_box = (24, 22, WIDTH - 24, 610)
    draw.rounded_rectangle(terminal_box, radius=16, fill=TERMINAL, outline=TERMINAL_EDGE, width=2)

    # Familiar terminal chrome, kept quiet so the real model output remains
    # the focus of the frame.
    draw.ellipse((44, 39, 56, 51), fill="#ff6b72")
    draw.ellipse((66, 39, 78, 51), fill=YELLOW)
    draw.ellipse((88, 39, 100, 51), fill="#59d28b")
    draw.line((24, 70, WIDTH - 24, 70), fill=TERMINAL_EDGE, width=1)
    draw.text((124, 34), "tasky", font=regular, fill=MUTED)

    content_font = regular
    content_bold = bold
    max_width = terminal_box[2] - 52 - 20
    lines = wrapped_lines(model_text, content_font, max_width)
    y = 92
    line_height = content_font.size + 9
    last_line_bottom = y + max(len(lines) - 1, 0) * line_height + content_font.size
    if last_line_bottom > terminal_box[3] - 18:
        raise ValueError(
            f"model text needs {last_line_bottom}px, beyond terminal content bottom {terminal_box[3] - 18}px"
        )
    for line, help_line in lines:
        selected_font = content_bold if line.strip() in {"Tasky", "Tasks", "Groups"} else content_font
        draw_segmented_line(draw, 52, y, line, selected_font, help_line)
        y += line_height

    # Annotation strip outside the terminal, so it cannot be mistaken for
    # text rendered by Tasky itself. Caption and key are deliberately on
    # separate rows so a long explanation never collides with the key pill.
    draw.rounded_rectangle((24, 630, WIDTH - 24, HEIGHT - 22), radius=12, fill="#0f1421", outline=TERMINAL_EDGE, width=1)
    caption_max_width = WIDTH - 92
    if draw.textlength(caption, font=bold) > caption_max_width:
        raise ValueError(f"caption is wider than the annotation strip: {caption!r}")
    draw.text((46, 646), caption, font=bold, fill=TEXT)
    if pressed:
        label = f"pressed  {pressed}"
        label_width = draw.textlength(label, font=regular)
        if label_width + 20 > WIDTH - 92:
            raise ValueError(f"pressed-key label is wider than the annotation strip: {label!r}")
        pill = (WIDTH - 54 - label_width, 680, WIDTH - 46, 712)
        draw.rounded_rectangle(pill, radius=8, fill="#1c2940", outline="#354665", width=1)
        draw.text((pill[0] + 10, 687), label, font=regular, fill=CYAN)

    return image


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, default=Path("scripts/demo/frames.json"))
    parser.add_argument("--output", type=Path, default=Path("assets/tasky-demo.gif"))
    args = parser.parse_args()

    frames = load_frames(args.input)
    regular_path = find_font("CascadiaMono.ttf", "consola.ttf")
    bold_path = find_font("CascadiaMono-Bold.ttf", "consolab.ttf", "CascadiaMono.ttf", "consola.ttf")
    regular = ImageFont.truetype(regular_path, 18)
    bold = ImageFont.truetype(bold_path, 18)

    images = [
        render_frame(
            frame["text"],
            frame.get("caption", ""),
            frame.get("pressed", ""),
            regular,
            bold,
        )
        for frame in frames
    ]

    args.output.parent.mkdir(parents=True, exist_ok=True)
    paletted = [image.quantize(colors=256, method=Image.Quantize.MEDIANCUT) for image in images]
    paletted[0].save(
        args.output,
        save_all=True,
        append_images=paletted[1:],
        duration=[int(frame["duration_ms"]) for frame in frames],
        loop=0,
        disposal=2,
        optimize=True,
    )
    duration = sum(int(frame["duration_ms"]) for frame in frames) / 1000
    print(f"wrote {args.output} ({WIDTH}x{HEIGHT}, {len(frames)} frames, {duration:.1f}s)")


if __name__ == "__main__":
    main()
