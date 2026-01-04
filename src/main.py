"""
iSecure VPN - Main entry point.
"""
import sys
import os
import tkinter as tk

# Add src to path for imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from utils import check_single_instance
from gui import CustomNekoGUI

if __name__ == "__main__":
    # Check single instance before creating GUI
    is_first, mutex_handle = check_single_instance()
    if not is_first:
        # Another instance is already running
        try:
            import ctypes
            ctypes.windll.user32.MessageBoxW(
                0,
                "iSecure VPN уже запущен и работает в системном трее.\n\n"
                "Нажмите правой кнопкой мыши на иконку в трее, чтобы открыть окно приложения.",
                "iSecure VPN",
                0x40  # MB_ICONINFORMATION
            )
        except:
            pass
        sys.exit(0)
    
    root = tk.Tk()
    gui = CustomNekoGUI(root)
    # Store mutex handle for cleanup on exit
    gui._mutex_handle = mutex_handle
    root.mainloop()

