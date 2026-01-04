# -*- mode: python ; coding: utf-8 -*-


a = Analysis(
    ['src\\main.py'],
    pathex=['src'],
    binaries=[('.\\resources\\core\\nekobox_core.exe', 'resources/core')],
    datas=[('.\\resources\\core\\nekobox_core.sha256', 'resources/core'), ('.\\resources\\data\\geoip.db', 'resources/data'), ('.\\resources\\data\\geosite.db', 'resources/data'), ('.\\resources\\images\\nekobox.png', 'resources/images'), ('.\\resources\\sounds\\success_start.mp3', 'resources/sounds'), ('.\\resources\\sounds\\failed_start.mp3', 'resources/sounds')],
    hiddenimports=['psutil', 'pystray', 'utils', 'gui', 'i18n', 'logger', 'vpn_utils'],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    noarchive=False,
    optimize=0,
)
pyz = PYZ(a.pure)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name='iSecureClient',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon=['resources\\images\\iSecureVPN.ico'],
)
