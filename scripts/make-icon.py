from pathlib import Path
import os
import shutil
import subprocess
import sys

try:
    from PIL import Image, ImageDraw
except ImportError:
    subprocess.check_call([sys.executable, "-m", "pip", "install", "pillow", "-q"])
    from PIL import Image, ImageDraw

root = Path(__file__).resolve().parents[1]
candidates = [
    root / "resources" / "images" / "appicon.png",
    Path(r"C:\Users\vints\.cursor\projects\c-dev-projects-experiments-iSecure-iSecure-client-VPN\assets\vpn-app-icon.png"),
]
src = next((p for p in candidates if p.exists()), None)
if src is None:
    raise SystemExit("source icon png not found")

img_dir = root / "resources" / "images"
build_dir = root / "build"
win_dir = build_dir / "windows"
frontend_public = root / "frontend" / "public"
winres_dir = root / "winres"
for d in (img_dir, win_dir, frontend_public, winres_dir):
    d.mkdir(parents=True, exist_ok=True)


def round_corners(im: Image.Image, radius_ratio: float = 0.22) -> Image.Image:
    """Make a squircle with fully transparent corners (no white fringe)."""
    im = im.convert("RGBA")
    w, h = im.size
    radius = max(1, int(min(w, h) * radius_ratio))
    mask = Image.new("L", (w, h), 0)
    draw = ImageDraw.Draw(mask)
    # Small inset removes anti-alias fringe that looks white on dark taskbars.
    inset = max(2, min(w, h) // 200)
    draw.rounded_rectangle(
        (inset, inset, w - 1 - inset, h - 1 - inset),
        radius=max(1, radius - inset),
        fill=255,
    )
    r, g, b, _ = im.split()
    return Image.merge("RGBA", (r, g, b, mask))


base = round_corners(Image.open(src))
master = img_dir / "appicon.png"
base.save(master, "PNG")
shutil.copy2(master, frontend_public / "appicon.png")
shutil.copy2(master, build_dir / "appicon.png")

sizes = [16, 24, 32, 48, 64, 128, 256]
variants = [base.resize((s, s), Image.Resampling.LANCZOS) for s in sizes]
ico_path = img_dir / "app.ico"
variants[-1].save(ico_path, format="ICO", append_images=variants[:-1])
shutil.copy2(ico_path, win_dir / "icon.ico")
shutil.copy2(ico_path, img_dir / "iSecureVPN.ico")

icon256 = base.resize((256, 256), Image.Resampling.LANCZOS)
icon256.save(winres_dir / "icon.png", "PNG")
base.resize((16, 16), Image.Resampling.LANCZOS).save(winres_dir / "icon16.png", "PNG")

print(f"wrote {master}")
print(f"wrote {ico_path} ({ico_path.stat().st_size} bytes)")

gopath = os.environ.get("GOPATH") or r"C:\dev\tools\gopath"
winres_bin = Path(gopath) / "bin" / "go-winres.exe"
if winres_bin.exists():
    subprocess.check_call([
        str(winres_bin), "simply",
        "--arch", "amd64",
        "--manifest", "gui",
        "--admin",
        "--icon", str(winres_dir / "icon.png"),
        "--out", "rsrc",
        "--product-name", "MyInternetVPN",
        "--file-description", "Windows VPN client",
        "--original-filename", "MyInternetVPN.exe",
        "--product-version", "2.0.0.0",
        "--file-version", "2.0.0.0",
    ], cwd=root)
    print("wrote rsrc_windows_amd64.syso")
else:
    print("go-winres not found; skip .syso (tray/ICO still updated)")
