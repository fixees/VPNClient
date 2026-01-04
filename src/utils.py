"""
Utility functions for iSecure VPN.
"""
import os
import sys

if os.name == 'nt':
    import ctypes
    from ctypes import wintypes


def check_single_instance():
    """
    Check if another instance is already running using Windows mutex.
    Returns (is_first_instance: bool, mutex_handle: int|None).
    """
    if os.name != "nt":
        # On non-Windows, just allow multiple instances for now
        return (True, None)
    
    try:
        # Windows API constants
        ERROR_ALREADY_EXISTS = 183
        
        # CreateMutexW signature
        kernel32 = ctypes.windll.kernel32
        kernel32.CreateMutexW.argtypes = [wintypes.LPVOID, wintypes.BOOL, wintypes.LPCWSTR]
        kernel32.CreateMutexW.restype = wintypes.HANDLE
        kernel32.GetLastError.restype = wintypes.DWORD
        kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
        kernel32.CloseHandle.restype = wintypes.BOOL
        
        mutex_name = "iSecureVPN_Mutex_SingleInstance"
        handle = kernel32.CreateMutexW(None, False, mutex_name)
        
        if handle == 0:
            # Failed to create mutex (unlikely), but allow instance anyway
            return (True, None)
        
        error = kernel32.GetLastError()
        if error == ERROR_ALREADY_EXISTS:
            # Another instance is running
            kernel32.CloseHandle(handle)
            return (False, None)
        
        # This is the first instance
        return (True, handle)
    
    except Exception:
        # If mutex check fails, allow instance (graceful degradation)
        return (True, None)


def resource_path(relative_path):
    """
    Get absolute path to resource, works for dev and for PyInstaller.
    """
    try:
        # PyInstaller creates a temp folder and stores path in _MEIPASS
        base_path = sys._MEIPASS
    except Exception:
        base_path = os.path.abspath(".")
    
    return os.path.join(base_path, relative_path)

