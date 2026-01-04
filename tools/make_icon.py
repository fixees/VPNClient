import sys
from PIL import Image


def main() -> int:
    if len(sys.argv) != 3:
        print("Usage: python tools/make_icon.py <input_png> <output_ico>")
        return 2

    inp, out = sys.argv[1], sys.argv[2]
    img = Image.open(inp).convert("RGBA")
    # Standard Windows icon sizes
    sizes = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    img.save(out, format="ICO", sizes=sizes)
    print(f"Generated: {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


