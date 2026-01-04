import tkinter as tk
from tkinter import messagebox, ttk, filedialog
import subprocess
import os
import json
import threading
import sys
import time
import platform
import urllib.parse as urlparse
import hashlib
import tempfile
import uuid
from datetime import datetime
import socket
import atexit
import queue
import re

# Import new modules
from i18n import init_translator, get_translator, _
from logger import init_logger, get_logger, LogLevel
from vpn_utils import KillSwitch, DNSLeakProtection, VPNHealthCheck, AutoReconnect

if os.name == 'nt':
    import winreg

class CustomNekoGUI:
    def _tray_supported(self) -> bool:
        return os.name == "nt"

    def _ensure_tray(self):
        """Create tray icon if enabled; safe to call multiple times."""
        if not self._tray_supported():
            return
        if getattr(self, "_tray_icon", None) is not None:
            return
        try:
            import importlib
            pystray = importlib.import_module("pystray")
            from PIL import Image
        except Exception as e:
            try:
                self.log_message(f"[GUI] Tray not available: {e}")
            except:
                pass
            return

        def on_show(_icon=None, _item=None):
            try:
                self.root.after(0, self._tray_show)
            except:
                pass

        def on_hide(_icon=None, _item=None):
            try:
                self.root.after(0, self._tray_hide)
            except:
                pass

        def on_exit(_icon=None, _item=None):
            try:
                self.root.after(0, self.exit_app)
            except:
                pass

        def on_quick_connect(_icon=None, _item=None):
            try:
                self.root.after(0, self.toggle_vpn)
            except:
                pass

        def on_disconnect(_icon=None, _item=None):
            try:
                if self.is_running:
                    self.root.after(0, self.stop_vpn)
            except:
                pass

        def on_reconnect(_icon=None, _item=None):
            try:
                if self.is_running:
                    self.root.after(0, lambda: (self.stop_vpn(), self.root.after(1000, self.start_vpn)))
            except:
                pass

        def on_stats(_icon=None, _item=None):
            try:
                self.root.after(0, self._tray_show)
            except:
                pass

        try:
            img_path = self.resource_path("nekobox.png")
            image = Image.open(img_path).convert("RGBA")
        except Exception:
            image = None

        # Build menu with translations
        t = self.translator if hasattr(self, 'translator') else None
        show_text = t("tray_show") if t else "Show"
        hide_text = t("tray_hide") if t else "Hide"
        exit_text = t("tray_exit") if t else "Exit"
        quick_text = t("tray_quick_connect") if t else "Quick Connect"
        disconnect_text = t("tray_disconnect") if t else "Disconnect"
        reconnect_text = t("tray_reconnect") if t else "Reconnect"
        stats_text = t("tray_stats") if t else "Statistics"

        menu_items = [
            pystray.MenuItem(show_text, on_show, default=True),
            pystray.MenuItem(hide_text, on_hide),
            pystray.Menu.SEPARATOR,
        ]
        
        # Add VPN controls if running
        if self.is_running:
            menu_items.append(pystray.MenuItem(disconnect_text, on_disconnect))
            menu_items.append(pystray.MenuItem(reconnect_text, on_reconnect))
        else:
            menu_items.append(pystray.MenuItem(quick_text, on_quick_connect))
        
        menu_items.extend([
            pystray.Menu.SEPARATOR,
            pystray.MenuItem(stats_text, on_stats),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem(exit_text, on_exit),
        ])

        menu = pystray.Menu(*menu_items)

        try:
            self._tray_icon = pystray.Icon("iSecureVPN", image, "iSecure VPN", menu)
            # Detached run so Tk mainloop isn't blocked.
            self._tray_icon.run_detached()
        except Exception as e:
            self._tray_icon = None
            try:
                self.log_message(f"[GUI] Tray init failed: {e}")
            except:
                pass

    def _tray_hide(self):
        try:
            self.root.withdraw()
        except:
            pass

    def _tray_show(self):
        try:
            self.root.deiconify()
            self.root.lift()
            self.root.focus_force()
        except:
            pass

    def exit_app(self):
        """Full exit (used by tray 'Exit')."""
        try:
            self._closing = True
        except:
            pass
        try:
            self.save_settings()
        except:
            pass
        try:
            # Stop health check
            if hasattr(self, 'health_check') and self.health_check:
                self.health_check.stop()
        except:
            pass
        try:
            # Disable auto reconnect
            if hasattr(self, 'auto_reconnect') and self.auto_reconnect:
                self.auto_reconnect.disable()
        except:
            pass
        try:
            # Deactivate kill switch
            if hasattr(self, 'kill_switch') and self.kill_switch:
                self.kill_switch.deactivate()
        except:
            pass
        try:
            # Shutdown logger
            if hasattr(self, 'app_logger') and self.app_logger:
                self.app_logger.shutdown()
        except:
            pass
        try:
            self.cleanup_core()
        except:
            pass
        try:
            # Stop tray icon loop if present
            if getattr(self, "_tray_icon", None) is not None:
                try:
                    self._tray_icon.stop()
                except:
                    pass
                self._tray_icon = None
        except:
            pass
        try:
            # Release mutex handle if present
            if hasattr(self, "_mutex_handle") and self._mutex_handle is not None:
                try:
                    import ctypes
                    from ctypes import wintypes
                    kernel32 = ctypes.windll.kernel32
                    kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
                    kernel32.CloseHandle(self._mutex_handle)
                except:
                    pass
                self._mutex_handle = None
        except:
            pass
        try:
            self.root.destroy()
        except:
            pass
    def load_settings(self):
        """
        Load persisted UI/app settings (non-sensitive) from per-user settings.json.
        """
        try:
            if not getattr(self, "settings_file", None):
                return
            if not os.path.exists(self.settings_file):
                return
            with open(self.settings_file, "r", encoding="utf-8") as f:
                data = json.load(f)
            if not isinstance(data, dict):
                return
            # Only allow known keys to be overridden
            for k, v in data.items():
                if k in self.app_settings:
                    self.app_settings[k] = v
        except Exception:
            pass

    def save_settings(self):
        """
        Persist app settings so toggles (e.g. run_as_admin) survive restarts.
        """
        try:
            if not getattr(self, "settings_file", None):
                return
            # Do not persist volatile runtime fields
            data = dict(self.app_settings or {})
            self.atomic_write_json(self.settings_file, data)
        except Exception:
            pass

    def relaunch_as_admin(self) -> bool:
        """
        Relaunch current app elevated (UAC prompt). Returns True if launch was initiated.
        """
        if os.name != "nt":
            return False
        try:
            import ctypes
            # Build command line
            if getattr(sys, "frozen", False):
                exe = sys.executable
                params = " ".join([f'"{a}"' for a in sys.argv[1:]])
            else:
                exe = sys.executable
                script = os.path.abspath(sys.argv[0])
                params = " ".join([f'"{script}"'] + [f'"{a}"' for a in sys.argv[1:]])
            rc = ctypes.windll.shell32.ShellExecuteW(None, "runas", exe, params, None, 1)
            return int(rc) > 32
        except Exception:
            return False
    def resource_path(self, name: str) -> str:
        """
        PyInstaller-friendly resource lookup.
        - When frozen: files live in sys._MEIPASS
        - When running from source: files live in project/app directory
        Maps resource names to resources/ subdirectories:
        - *.mp3 -> resources/sounds/
        - *.png, *.ico -> resources/images/
        - *.db -> resources/data/
        - nekobox_core.* -> resources/core/
        """
        # Map file to resources subdirectory
        if name.endswith(".mp3"):
            resource_subdir = "resources/sounds"
        elif name.endswith((".png", ".ico")):
            resource_subdir = "resources/images"
        elif name.endswith(".db"):
            resource_subdir = "resources/data"
        elif "nekobox_core" in name:
            resource_subdir = "resources/core"
        else:
            resource_subdir = "resources"
        
        # Try PyInstaller path first
        try:
            base = getattr(sys, "_MEIPASS", None)
            if base:
                full_path = os.path.join(base, resource_subdir, os.path.basename(name))
                if os.path.exists(full_path):
                    return full_path
                # Fallback to root for compatibility
                root_path = os.path.join(base, name)
                if os.path.exists(root_path):
                    return root_path
        except Exception:
            pass
        
        # Try source directory
        app_dir = self.get_app_dir()
        full_path = os.path.join(app_dir, resource_subdir, os.path.basename(name))
        if os.path.exists(full_path):
            return full_path
        # Fallback to root for compatibility
        return os.path.join(app_dir, name)

    def pick_tun_interface_name(self, base: str = "iSecure-TUN") -> str:
        """
        Pick a Windows TUN interface name that avoids collisions.
        Some sing-box builds create a Wintun adapter and will fail if a same-named adapter already exists.
        """
        base = (base or "iSecure-TUN").strip() or "iSecure-TUN"
        if os.name != "nt":
            return base
        try:
            import psutil  # bundled in EXE; optional in source
            per = psutil.net_io_counters(pernic=True)
            if isinstance(per, dict) and base in per:
                return f"{base}-{uuid.uuid4().hex[:4]}"
            return base
        except Exception:
            # If we can't detect adapters, still avoid hard collision by suffixing.
            return f"{base}-{uuid.uuid4().hex[:4]}"

    def pick_tun_inet4_address(self, default: str = "172.19.0.1/30") -> str:
        """
        Pick a private IPv4 CIDR for tun that doesn't collide with existing interface IPs.
        Some Windows setups keep the previous IP on the adapter, causing:
          'set ipv4 address: The object already exists.'
        """
        if os.name != "nt":
            return default
        try:
            import psutil
            used = set()
            try:
                addrs = psutil.net_if_addrs()
                for _, lst in (addrs or {}).items():
                    for a in lst or []:
                        ip = getattr(a, "address", "") or ""
                        if ip.count(".") == 3:
                            used.add(ip.strip())
            except Exception:
                used = set()

            # Try a bunch of random RFC1918 /30 blocks.
            # 172.16.0.0/12 gives plenty of room; avoid low subnets that are common.
            for _ in range(64):
                b = 16 + (uuid.uuid4().int % 16)          # 16..31
                c = (uuid.uuid4().int >> 8) % 250         # 0..249
                ip = f"172.{b}.{c}.1"
                if ip not in used:
                    return f"{ip}/30"
        except Exception:
            pass
        return default
    def is_port_free(self, port: int) -> bool:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        try:
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            s.bind(("127.0.0.1", int(port)))
            return True
        except Exception:
            return False
        finally:
            try:
                s.close()
            except:
                pass

    def pick_stats_port(self, preferred=9090):
        # NekoBox typically uses 9090; use it if available, else random free port.
        try:
            if self.is_port_free(int(preferred)):
                return int(preferred)
        except Exception:
            pass
        return self.pick_free_port()
    def pick_free_port(self):
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        try:
            s.bind(("127.0.0.1", 0))
            return s.getsockname()[1]
        finally:
            try:
                s.close()
            except:
                pass

    def clash_api_request_json(self, path: str, timeout=2.5):
        """
        Query local sing-box Clash API (like NekoBox).
        Returns dict or None.
        """
        if not getattr(self, "clash_api_port", None) or not getattr(self, "clash_api_secret", None):
            return None
        import urllib.request
        import json as _json
        url = f"http://127.0.0.1:{self.clash_api_port}{path}"
        raw = None
        # Try common auth styles (+ no auth). Different builds vary.
        auth_headers = [
            {"Authorization": f"Bearer {self.clash_api_secret}"},
            {"Authorization": f"{self.clash_api_secret}"},
            {},  # no auth
        ]
        last_err = None
        for h in auth_headers:
            try:
                req = urllib.request.Request(url, headers=h)
                with urllib.request.urlopen(req, timeout=timeout) as resp:
                    raw = resp.read().decode("utf-8", errors="ignore") or "{}"
                break
            except Exception as e:
                last_err = e
                continue
        if raw is None:
            raise last_err or Exception("Clash API request failed")
        try:
            data = _json.loads(raw)
        except:
            return None
        return data if isinstance(data, dict) else None

    def is_clash_api_listening(self) -> bool:
        try:
            port = int(getattr(self, "clash_api_port", 0) or 0)
            if port <= 0:
                return False
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(0.5)
            try:
                s.connect(("127.0.0.1", port))
                return True
            finally:
                try:
                    s.close()
                except:
                    pass
        except Exception:
            return False

    # ---- Fallback stats (interface-based) ----
    def find_tun_nic_name(self, psutil):
        try:
            per = psutil.net_io_counters(pernic=True)
        except Exception:
            return None
        if not isinstance(per, dict):
            return None
        # User override
        chosen = (self.app_settings.get("traffic_iface") or "auto").strip()
        if chosen and chosen.lower() != "auto":
            # If the exact NIC isn't present (names vary), fall back to auto-detect below.
            if chosen in per:
                return chosen
        # Try to match our tun interface name first; then common wintun/tun keywords.
        targets = ["isecure-tun", "isecure", "wintun", "tun"]
        for name in per.keys():
            low = (name or "").lower()
            if any(t in low for t in targets):
                return name
        return None

    def list_net_ifaces(self):
        """Return list of NIC names for fallback traffic selection."""
        try:
            import psutil
            per = psutil.net_io_counters(pernic=True)
            if isinstance(per, dict):
                names = [n for n in per.keys() if n]
                names.sort(key=lambda x: x.lower())
                return names
        except Exception:
            return []
        return []

    def start_interface_stats_monitor(self):
        """
        Fallback when Clash API isn't exposed by this core build.
        Uses OS interface counters (best-effort).
        """
        # IMPORTANT: avoid any potentially-blocking psutil calls on Tk main thread.
        def boot():
            try:
                import psutil
            except Exception as e:
                try:
                    self.root.after(0, lambda: self.speed_val.config(text=f"Stats unavailable (psutil error)"))
                    self.log_message(f"[GUI] psutil import failed: {e}")
                except:
                    pass
                return

            self.traffic_stop_flag = False
            self.stats_source = "iface"
            try:
                nic = self.find_tun_nic_name(psutil)
                base = psutil.net_io_counters(pernic=True) if nic else None
                if nic and base and nic in base:
                    b = base[nic]
                    base_rx, base_tx = b.bytes_recv, b.bytes_sent
                else:
                    # total system traffic fallback
                    tot = psutil.net_io_counters()
                    base_rx, base_tx = tot.bytes_recv, tot.bytes_sent
                    nic = None
            except Exception:
                # If counters hang/fail, just bail out quietly.
                return

            last_ts = time.time()
            last_rx, last_tx = base_rx, base_tx

            preferred = (self.app_settings.get("traffic_iface") or "auto").strip()

            def loop():
                nonlocal last_ts, last_rx, last_tx, base_rx, base_tx, nic
                while self.is_running and not getattr(self, "traffic_stop_flag", False) and getattr(self, "stats_source", "iface") == "iface":
                    try:
                        now = time.time()
                        dt = max(now - last_ts, 0.2)
                        # If we didn't find the interface at start, retry (TUN can appear a bit later)
                        if nic is None:
                            cand = self.find_tun_nic_name(psutil)
                            if cand:
                                try:
                                    base = psutil.net_io_counters(pernic=True)
                                    if cand in base:
                                        b = base[cand]
                                        nic = cand
                                        base_rx, base_tx = b.bytes_recv, b.bytes_sent
                                        last_rx, last_tx = base_rx, base_tx
                                        last_ts = now
                                except:
                                    pass
                        if nic:
                            cur = psutil.net_io_counters(pernic=True).get(nic)
                            if not cur:
                                tot = psutil.net_io_counters()
                                rx, tx = tot.bytes_recv, tot.bytes_sent
                            else:
                                rx, tx = cur.bytes_recv, cur.bytes_sent
                        else:
                            tot = psutil.net_io_counters()
                            rx, tx = tot.bytes_recv, tot.bytes_sent

                        drx, dtx = rx - last_rx, tx - last_tx
                        total_rx, total_tx = rx - base_rx, tx - base_tx
                        total = max(total_rx, 0) + max(total_tx, 0)

                        dl_txt = self._fmt_bytes(int(max(total_rx, 0))) if total > 0 else "--"
                        ul_txt = self._fmt_bytes(int(max(total_tx, 0))) if total > 0 else "--"
                        total_txt = self._fmt_bytes(int(total)) if total > 0 else "--"
                        speed_line = f"↓ {self._fmt_rate(float(max(drx, 0)) / dt)}   ↑ {self._fmt_rate(float(max(dtx, 0)) / dt)}"

                        def apply():
                            self.traffic_dl.config(text=dl_txt)
                            self.traffic_ul.config(text=ul_txt)
                            self.traffic_total.config(text=total_txt)
                            self.speed_val.config(text=speed_line + "  •  fallback")

                        self.root.after(0, apply)

                        last_ts, last_rx, last_tx = now, rx, tx
                    except Exception:
                        pass
                    time.sleep(1)

            threading.Thread(target=loop, daemon=True).start()

        threading.Thread(target=boot, daemon=True).start()

    def start_core_stats_monitor(self):
        """
        NekoBox-like traffic counter: read totals and live rates from Clash API.
        """
        self.traffic_stop_flag = False

        # UI defaults
        try:
            self.traffic_dl.config(text="--")
            self.traffic_ul.config(text="--")
            self.traffic_total.config(text="--")
            self.speed_val.config(text="Connecting stats…")
        except:
            pass

        # Quick probe: verify the API actually responds; otherwise keep fallback.
        def probe_async():
            # IMPORTANT: never do urllib requests on Tk main thread.
            def work():
                if not self.is_running:
                    return
                ok = False
                try:
                    t = self.clash_api_request_json("/traffic", timeout=1.0) or {}
                    ok = ("down" in t) or ("up" in t)
                except:
                    ok = False

                def apply():
                    if not getattr(self, "is_running", False):
                        return
                    if not ok:
                        try:
                            self.speed_val.config(text="Stats: fallback (no Clash API)")
                        except:
                            pass
                        try:
                            self.log_message(f"[GUI] Stats probe: Clash API not listening on 127.0.0.1:{getattr(self,'clash_api_port',None)}")
                        except:
                            pass
                        try:
                            self.start_interface_stats_monitor()
                        except:
                            pass
                    else:
                        self.stats_source = "core"

                try:
                    self.root.after(0, apply)
                except:
                    pass

            threading.Thread(target=work, daemon=True).start()
        try:
            self.root.after(2500, probe_async)
        except:
            pass

        # Fallback accumulators (used if /connections is slow/unavailable)
        self._core_total_dl = getattr(self, "_core_total_dl", 0)
        self._core_total_ul = getattr(self, "_core_total_ul", 0)
        self._core_last_ts = time.time()

        def loop():
            last_ok = 0
            last_conn_refresh = 0
            while self.is_running and not getattr(self, "traffic_stop_flag", False) and getattr(self, "stats_source", "iface") == "core":
                try:
                    now = time.time()

                    # Live rate (Clash API) — poll frequently
                    t = self.clash_api_request_json("/traffic", timeout=2.5) or {}
                    down_rate = t.get("down")
                    up_rate = t.get("up")

                    # Parse numeric strings if needed
                    try:
                        if isinstance(down_rate, str):
                            down_rate = float(down_rate)
                        if isinstance(up_rate, str):
                            up_rate = float(up_rate)
                    except:
                        pass

                    # Totals (Clash API) — poll less frequently (can be heavier)
                    dl_total = ul_total = None
                    if now - last_conn_refresh > 6:
                        last_conn_refresh = now
                        c = self.clash_api_request_json("/connections", timeout=6.0) or {}
                        dl_total = c.get("downloadTotal", c.get("download_total"))
                        ul_total = c.get("uploadTotal", c.get("upload_total"))
                        try:
                            if isinstance(dl_total, str):
                                dl_total = float(dl_total)
                            if isinstance(ul_total, str):
                                ul_total = float(ul_total)
                        except:
                            dl_total = ul_total = None
                        if isinstance(dl_total, (int, float)) and isinstance(ul_total, (int, float)):
                            self._core_total_dl = int(dl_total)
                            self._core_total_ul = int(ul_total)

                    # If totals weren't refreshed, integrate from speeds (fallback)
                    dt = max(now - getattr(self, "_core_last_ts", now), 0.2)
                    self._core_last_ts = now
                    if isinstance(down_rate, (int, float)):
                        self._core_total_dl += int(float(down_rate) * dt)
                    if isinstance(up_rate, (int, float)):
                        self._core_total_ul += int(float(up_rate) * dt)

                    total = int(self._core_total_dl) + int(self._core_total_ul)
                    dl_txt = self._fmt_bytes(int(self._core_total_dl)) if total > 0 else "--"
                    ul_txt = self._fmt_bytes(int(self._core_total_ul)) if total > 0 else "--"
                    total_txt = self._fmt_bytes(int(total)) if total > 0 else "--"

                    if isinstance(down_rate, (int, float)) and isinstance(up_rate, (int, float)):
                        speed_line = f"↓ {self._fmt_rate(float(down_rate))}   ↑ {self._fmt_rate(float(up_rate))}"
                    else:
                        speed_line = "↓ --   ↑ --"

                    def apply():
                        self.traffic_dl.config(text=dl_txt)
                        self.traffic_ul.config(text=ul_txt)
                        self.traffic_total.config(text=total_txt)
                        self.speed_val.config(text=speed_line)

                    self.root.after(0, apply)
                    last_ok = time.time()
                except Exception:
                    # If API isn't available, show hint after a short grace period.
                    if time.time() - last_ok > 5:
                        try:
                            self.root.after(0, lambda: self.speed_val.config(text="Speed: core stats unavailable"))
                        except:
                            pass
                time.sleep(1)

        threading.Thread(target=loop, daemon=True).start()
    def get_app_dir(self):
        # Directory where the app (or packaged exe) lives
        try:
            # When frozen, sys.executable points to the real exe path
            if getattr(sys, "frozen", False):
                return os.path.dirname(os.path.abspath(sys.executable))
            return os.path.dirname(os.path.abspath(sys.argv[0]))
        except:
            return os.getcwd()

    def get_data_dir(self):
        # Per-user storage (avoid writing secrets/logs into working directory)
        if os.name == "nt":
            base = os.getenv("APPDATA") or os.path.expanduser("~")
            d = os.path.join(base, "InternetSecure")
        else:
            d = os.path.join(os.path.expanduser("~"), ".config", "InternetSecure")
        try:
            os.makedirs(d, exist_ok=True)
        except:
            pass
        return d

    def atomic_write_json(self, path, data):
        tmp_dir = os.path.dirname(path)
        try:
            os.makedirs(tmp_dir, exist_ok=True)
        except:
            pass
        fd, tmp_path = tempfile.mkstemp(prefix="tmp_", suffix=".json", dir=tmp_dir)
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(data, f, indent=4, ensure_ascii=False)
                f.flush()
                try:
                    os.fsync(f.fileno())
                except:
                    pass
            os.replace(tmp_path, path)
        finally:
            try:
                if os.path.exists(tmp_path):
                    os.remove(tmp_path)
            except:
                pass

    def compute_sha256(self, path):
        h = hashlib.sha256()
        with open(path, "rb") as f:
            for chunk in iter(lambda: f.read(1024 * 1024), b""):
                h.update(chunk)
        return h.hexdigest()

    def verify_core_integrity(self, core_path):
        """
        Integrity check: if `nekobox_core.sha256` exists next to the core, verify the hash matches.
        This mitigates binary-replacement attacks when running as Admin.
        """
        try:
            # Prefer sha file next to core, but allow bundled sha in onefile (_MEIPASS)
            sha_path = os.path.join(os.path.dirname(core_path), "nekobox_core.sha256")
            if not os.path.exists(sha_path):
                sha_path = self.resource_path("nekobox_core.sha256")
            if not os.path.exists(sha_path):
                # No pinned hash file shipped; don't block, but record a warning.
                try:
                    self.log_message("[SECURITY] Warning: nekobox_core.sha256 not found; core integrity is not pinned.")
                except:
                    pass
                return True
            with open(sha_path, "r", encoding="utf-8", errors="ignore") as f:
                expected = (f.read() or "").strip().split()[0].lower()
            if not expected or len(expected) < 32:
                return False
            actual = self.compute_sha256(core_path).lower()
            return actual == expected
        except:
            return False

    def sanitize_template(self, tpl):
        """
        Defensive sanitizer for untrusted NekoBox sing-box templates.
        Goal: prevent accidental/hostile exposure (e.g., binding inbounds on 0.0.0.0).
        """
        if not isinstance(tpl, dict):
            raise ValueError("Template must be a JSON object")

        # Force safe logging (avoid arbitrary file outputs from template)
        tpl.setdefault("log", {})
        if isinstance(tpl.get("log"), dict):
            tpl["log"].pop("output", None)
            tpl["log"].pop("output_file", None)
            tpl["log"].setdefault("level", "info")
            tpl["log"].setdefault("timestamp", True)

        # Sanitize inbounds: never allow listening on all interfaces for local proxy ports
        inbounds = tpl.get("inbounds")
        if isinstance(inbounds, list):
            for ib in inbounds:
                if not isinstance(ib, dict):
                    continue
                ib_type = (ib.get("type") or "").lower()
                # tun inbound has no "listen" in many templates; skip
                if ib_type == "tun":
                    continue
                listen = ib.get("listen")
                if isinstance(listen, str):
                    if listen.strip() in ("0.0.0.0", "::", "[::]"):
                        # force localhost
                        ib["listen"] = "127.0.0.1"
                # if listen missing, keep default (sing-box often binds 127.0.0.1 anyway)

        return tpl

    def mask_ip(self, ip: str) -> str:
        ip = (ip or "").strip()
        if not ip:
            return "--"
        # "Cool blur" effect: keep first part readable, mask rest with stable shaded blocks.
        palette = ["░", "▒", "▓", "█"]
        digest = hashlib.sha256(ip.encode("utf-8", errors="ignore")).digest()

        # IPv4: show first octet, blur the rest
        if ip.count(".") == 3:
            first = ip.split(".")[0]
            segs = []
            k = 0
            for _ in range(3):
                # 3 chars per segment, stable per IP
                seg = "".join(palette[digest[k + j] % len(palette)] for j in range(3))
                segs.append(seg)
                k += 3
            return f"{first}.{segs[0]}.{segs[1]}.{segs[2]}"

        # IPv6: show first hextet, blur the rest (4 chars chunks)
        if ":" in ip:
            first = ip.split(":")[0]
            chunks = []
            k = 0
            for _ in range(7):
                chunk = "".join(palette[digest[k + j] % len(palette)] for j in range(4))
                chunks.append(chunk)
                k += 4
            return f"{first}:" + ":".join(chunks[:7])

        return ip

    def draw_eye_icon(self, canvas, opened: bool):
        """Draw a small eye icon on a canvas (no emoji)."""
        try:
            canvas.delete("all")
        except Exception:
            return
        w = int(canvas.winfo_width() or 24)
        h = int(canvas.winfo_height() or 18)
        cx, cy = w // 2, h // 2
        # outer eye
        canvas.create_oval(2, 4, w - 2, h - 4, outline=self.colors["text_dim"], width=1)
        # pupil
        if opened:
            canvas.create_oval(cx - 3, cy - 3, cx + 3, cy + 3, fill=self.colors["text"], outline="")
        else:
            # crossed line when hidden
            canvas.create_line(4, h - 5, w - 4, 5, fill=self.colors["text_dim"], width=2)

    def fetch_public_ip_geo(self, timeout=6):
        """
        Returns (ip, country, city). Uses a public HTTPS endpoint.
        """
        import urllib.request
        import json as _json

        # ipwho.is provides IP + geo in one call over HTTPS
        req = urllib.request.Request(
            "https://ipwho.is/",
            headers={"User-Agent": "InternetSecure/1.0"}
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            data = _json.loads(resp.read().decode("utf-8", errors="ignore") or "{}")
        if not isinstance(data, dict) or not data.get("success", False):
            return ("", "", "")
        ip = (data.get("ip") or "").strip()
        country = (data.get("country") or "").strip()
        city = (data.get("city") or "").strip()
        return (ip, country, city)

    def _fmt_bytes(self, n: int) -> str:
        try:
            n = int(n)
        except:
            return "--"
        if n < 0:
            n = 0
        units = ["B", "KB", "MB", "GB", "TB"]
        v = float(n)
        i = 0
        while v >= 1024 and i < len(units) - 1:
            v /= 1024.0
            i += 1
        if i == 0:
            return f"{int(v)} {units[i]}"
        return f"{v:.2f} {units[i]}"

    def _fmt_rate(self, bps: float) -> str:
        if bps is None:
            return "--"
        try:
            bps = float(bps)
        except:
            return "--"
        if bps < 0:
            bps = 0
        return f"{self._fmt_bytes(int(bps))}/s"

    # NOTE: Interface-based traffic sampling was replaced with core stats (Clash API).

    def update_ip_labels(self, which: str):
        # which: "direct" or "vpn"
        def task():
            try:
                ip, country, city = self.fetch_public_ip_geo(timeout=6)
            except Exception:
                ip, country, city = ("", "", "")
            masked = self.mask_ip(ip)
            loc = ", ".join([x for x in [city, country] if x])
            if not loc:
                loc = "--"
            def apply():
                if which == "direct":
                    self.ip_direct_val_full = ip
                    self.ip_direct_loc = loc
                else:
                    self.ip_vpn_val_full = ip
                    self.ip_vpn_loc = loc
                # Refresh the single "Ваш айпи" field
                try:
                    self.refresh_current_ip_field()
                except:
                    pass
            try:
                self.root.after(0, apply)
            except Exception:
                pass
        threading.Thread(target=task, daemon=True).start()

    def refresh_vpn_ip_with_retries(self, attempts=6, delay_ms=1500):
        """
        When VPN just started, routes may not be fully applied yet.
        Retry fetching VPN IP a few times so the UI actually switches.
        """
        try:
            self.ip_vpn_val_full = ""
            self.ip_vpn_loc = "--"
            self.ip_current_meta.config(text="Checking…  •  VPN")
        except:
            pass

        def step(n):
            if not self.is_running:
                return
            # fetch vpn ip
            self.update_ip_labels("vpn")
            # if still empty, retry
            if n > 0:
                try:
                    if not (self.ip_vpn_val_full or "").strip():
                        self.root.after(delay_ms, lambda: step(n - 1))
                except:
                    pass

        try:
            self.root.after(delay_ms, lambda: step(attempts))
        except:
            pass

    def refresh_current_ip_field(self):
        # Show VPN IP when connected, otherwise direct IP
        if getattr(self, "is_running", False) and (getattr(self, "ip_vpn_val_full", "") or "").strip():
            ip = self.ip_vpn_val_full
            loc = getattr(self, "ip_vpn_loc", "--")
            tag = "VPN"
        else:
            ip = getattr(self, "ip_direct_val_full", "")
            loc = getattr(self, "ip_direct_loc", "--")
            tag = "Direct"
        try:
            if getattr(self, "ip_revealed", False):
                shown = ip or "--"
                self.ip_current_ip_label.config(text=shown)
                self.draw_eye_icon(self.ip_eye_canvas, opened=True)
            else:
                self.ip_current_ip_label.config(text="BLURED")
                self.draw_eye_icon(self.ip_eye_canvas, opened=False)
            self.ip_current_meta.config(text=f"{loc}  •  {tag}")
        except:
            pass

    # NOTE: Interface-based traffic monitor was replaced with NekoBox-like core stats (Clash API).
    # NOTE: Gradient background was removed intentionally (per request).

    def _make_glow_title(self, parent, text):
        # Subtle title (keeps the same colors, but avoids "neon glow" look)
        wrap = tk.Frame(parent, bg=self.colors["bg"])
        tk.Label(
            wrap,
            text=text,
            font=("Segoe UI", 22, "bold"),
            bg=self.colors["bg"],
            fg=self.colors["accent"]
        ).pack()
        return wrap

    def _pill_button(self, parent, text, command, w=260, h=44, fill=None, outline=None):
        fill = fill or self.colors["accent2"] if "accent2" in self.colors else self.colors["accent"]
        outline = outline or self.colors["accent"]

        c = tk.Canvas(parent, width=w, height=h, bg=self.colors["bg"], highlightthickness=0, bd=0)
        r = h // 2
        # shadow
        c.create_oval(3, 3, 3 + h, 3 + h, fill="#041023", outline="")
        c.create_oval(w - h + 3, 3, w + 3, 3 + h, fill="#041023", outline="")
        c.create_rectangle(3 + r, 3, w - r + 3, h + 3, fill="#041023", outline="")
        # pill
        c.create_oval(0, 0, h, h, fill=fill, outline=outline, width=2, tags=("pill",))
        c.create_oval(w - h, 0, w, h, fill=fill, outline=outline, width=2, tags=("pill",))
        c.create_rectangle(r, 0, w - r, h, fill=fill, outline=outline, width=2, tags=("pill",))
        t = c.create_text(w // 2, h // 2, text=text, fill="#04121f", font=("Segoe UI", 11, "bold"), tags=("pill_text",))

        def on_enter(_=None):
            c.itemconfig("pill", outline=self.colors["accent_hover"])
        def on_leave(_=None):
            c.itemconfig("pill", outline=outline)
        def on_click(_=None):
            try:
                command()
            except Exception as e:
                print(e)

        c.bind("<Enter>", on_enter)
        c.bind("<Leave>", on_leave)
        c.bind("<Button-1>", on_click)
        c.tag_bind("pill", "<Button-1>", on_click)
        c.tag_bind(t, "<Button-1>", on_click)
        c.configure(cursor="hand2")
        return c

    def build_internetsecure_ui(self):
        # Solid background (no gradient)
        self.root.configure(bg=self.colors["bg"])

        # Root container
        root_wrap = tk.Frame(self.root, bg=self.colors["bg"])
        root_wrap.pack(fill=tk.BOTH, expand=True)

        # Phone-like centered content (simple pack, no canvas layer issues)
        self.content = tk.Frame(root_wrap, bg=self.colors["bg"])
        self.content.pack(fill=tk.BOTH, expand=True, padx=18, pady=18)

        # Top row
        top = tk.Frame(self.content, bg=self.colors["bg"])
        top.pack(fill=tk.X, padx=8, pady=(2, 10))
        self.settings_btn = tk.Button(
            top,
            text="Settings",
            command=self.open_settings,
            bg=self.colors["bg"],
            fg=self.colors["text_dim"],
            activebackground=self.colors["bg"],
            activeforeground=self.colors["text"],
            relief=tk.FLAT,
            bd=0,
            font=("Segoe UI", 9, "bold"),
            cursor="hand2"
        )
        self.settings_btn.pack(side=tk.RIGHT)

        # Header (no marketing block; this is a VPN client UI)
        hero = tk.Frame(self.content, bg=self.colors["bg"])
        hero.pack(fill=tk.X, padx=8, pady=(0, 12))
        self._make_glow_title(hero, "InternetSecure").pack(anchor="center")
        tk.Label(
            hero,
            text="VPN Client",
            bg=self.colors["bg"],
            fg=self.colors["text_dim"],
            font=("Segoe UI", 9)
        ).pack(pady=(4, 0))

        # Main panel (controls)
        panel = tk.Frame(
            self.content,
            bg=self.colors["card"],
            padx=14,
            pady=14,
            highlightthickness=1,
            highlightbackground=self.colors["card2"]
        )
        panel.pack(fill=tk.X, padx=8, pady=(0, 0))

        # Start/Stop button
        self.toggle_btn = tk.Button(
            panel, text="START VPN", command=self.toggle_vpn,
            bg=self.colors["accent2"],
            fg=self.colors["text"],
            font=("Segoe UI", 11, "bold"),
            activebackground=self.colors["accent"],
            activeforeground=self.colors["text"],
            relief=tk.FLAT,
            bd=0,
            height=2,
            cursor="hand2"
        )
        self.toggle_btn.pack(fill=tk.X, pady=(0, 12))

        status_row = tk.Frame(panel, bg=self.colors["card"])
        status_row.pack(fill=tk.X, pady=(0, 10))
        tk.Label(status_row, text="Status", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9)).pack(side=tk.LEFT)
        self.status_label = tk.Label(status_row, text="Disconnected", bg=self.colors["card"], fg=self.colors["status_off"], font=("Segoe UI", 9, "bold"))
        self.status_label.pack(side=tk.LEFT, padx=(8, 0))
        self.timer_label = tk.Label(status_row, text="00:00:00", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9, "bold"))
        self.timer_label.pack(side=tk.RIGHT)

        # Info block: IP + Traffic
        info = tk.Frame(panel, bg=self.colors["card"])
        info.pack(fill=tk.X, pady=(0, 10))

        # IP (single field, masked; click to copy full)
        self.ip_direct_val_full = ""
        self.ip_vpn_val_full = ""
        self.ip_direct_loc = "--"
        self.ip_vpn_loc = "--"

        tk.Label(info, text="Ваш айпи", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 8)).pack(anchor="w")
        ip_box = tk.Frame(info, bg=self.colors["card2"], padx=10, pady=8)
        ip_box.pack(fill=tk.X, pady=(2, 10))
        # IP line + eye toggle (no symbol masking / no emoji)
        ip_row = tk.Frame(ip_box, bg=self.colors["card2"])
        ip_row.pack(fill=tk.X)
        self.ip_revealed = False
        self.ip_current_ip_label = tk.Label(ip_row, text="BLURED", bg=self.colors["card2"], fg=self.colors["text"], font=("Consolas", 11, "bold"))
        self.ip_current_ip_label.pack(side=tk.LEFT, anchor="w")

        self.ip_eye_canvas = tk.Canvas(ip_row, width=26, height=18, bg=self.colors["card2"], highlightthickness=0, bd=0, cursor="hand2")
        self.ip_eye_canvas.pack(side=tk.RIGHT, padx=(8, 0))
        # initial icon
        self.draw_eye_icon(self.ip_eye_canvas, opened=False)

        def toggle_eye():
            self.ip_revealed = not bool(getattr(self, "ip_revealed", False))
            self.refresh_current_ip_field()

        self.ip_eye_canvas.bind("<Button-1>", lambda e: toggle_eye())
        # Location line
        self.ip_current_meta = tk.Label(ip_box, text="--", bg=self.colors["card2"], fg=self.colors["text_dim"], font=("Segoe UI", 9))
        self.ip_current_meta.pack(anchor="w", pady=(2, 0))

        def copy_ip_current():
            try:
                val = self.ip_vpn_val_full if (self.is_running and (self.ip_vpn_val_full or "").strip()) else self.ip_direct_val_full
                if not val:
                    return
                self.root.clipboard_clear()
                self.root.clipboard_append(val)
                self.show_notification("Copied", "IP copied")
            except:
                pass

        # Click anywhere in the IP box to copy full IP
        ip_box.bind("<Button-1>", lambda e: copy_ip_current())
        self.ip_current_ip_label.bind("<Button-1>", lambda e: copy_ip_current())
        self.ip_current_meta.bind("<Button-1>", lambda e: copy_ip_current())

        # Traffic (clean layout, no weird spacing/wrap)
        tk.Label(info, text="Traffic", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 8)).pack(anchor="w")

        traffic_row = tk.Frame(info, bg=self.colors["card"])
        traffic_row.pack(fill=tk.X, pady=(2, 2))

        def _stat_box(parent, title, initial="--"):
            box = tk.Frame(parent, bg=self.colors["card2"], padx=8, pady=6)
            tk.Label(box, text=title, bg=self.colors["card2"], fg=self.colors["text_dim"], font=("Segoe UI", 8)).pack(anchor="w")
            val = tk.Label(box, text=initial, bg=self.colors["card2"], fg=self.colors["text"], font=("Segoe UI", 9, "bold"))
            val.pack(anchor="w")
            return box, val

        b1, self.traffic_dl = _stat_box(traffic_row, "DL")
        b2, self.traffic_ul = _stat_box(traffic_row, "UL")
        b3, self.traffic_total = _stat_box(traffic_row, "Total")
        b1.pack(side=tk.LEFT, expand=True, fill=tk.X, padx=(0, 6))
        b2.pack(side=tk.LEFT, expand=True, fill=tk.X, padx=(0, 6))
        b3.pack(side=tk.LEFT, expand=True, fill=tk.X)

        speed_row = tk.Frame(info, bg=self.colors["card"])
        speed_row.pack(fill=tk.X, pady=(4, 0))
        self.speed_val = tk.Label(speed_row, text="↓ --   ↑ --", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9))
        self.speed_val.pack(anchor="w")

        tk.Label(panel, text="Servers", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9, "bold")).pack(anchor="w", pady=(0, 6))

        # List container with subtle border
        self.list_frame = tk.Frame(panel, bg=self.colors["card2"], highlightthickness=1, highlightbackground=self.colors["card2"])
        self.list_frame.pack(fill=tk.X)

        self.proxy_list = tk.Listbox(
            self.list_frame,
            height=8,
            bg=self.colors["card2"],
            fg=self.colors["text"],
            font=("Segoe UI", 10),
            borderwidth=0,
            highlightthickness=0,
            selectbackground=self.colors["accent2"],
            selectforeground=self.colors["text"],
            activestyle="none"
        )
        self.proxy_list.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        self.proxy_list.bind("<Double-Button-1>", lambda e: self.open_edit_proxy())

        self.context_menu = tk.Menu(self.proxy_list, tearoff=0, bg=self.colors["card"], fg=self.colors["text"], activebackground=self.colors["accent"])
        self.context_menu.add_command(label="Set as Default", command=self.set_as_default)
        self.proxy_list.bind("<Button-3>", self.show_context_menu)

        self.scrollbar = tk.Scrollbar(self.list_frame, bg=self.colors["card"])
        self.scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        self.proxy_list.config(yscrollcommand=self.scrollbar.set)
        self.scrollbar.config(command=self.proxy_list.yview)

        # Under-list buttons
        self.btn_frame = tk.Frame(panel, bg=self.colors["card"])
        self.btn_frame.pack(fill=tk.X, pady=(10, 0))

        tk.Button(self.btn_frame, text="+ ADD", command=self.open_add_proxy,
                  bg=self.colors["card2"], fg=self.colors["text"], font=("Segoe UI", 9, "bold"),
                  relief=tk.FLAT, bd=0, padx=12, cursor="hand2").pack(side=tk.LEFT)

        def delete_selected():
            idx = self.proxy_list.curselection()
            if idx:
                if self.show_custom_popup("Confirm", "Remove this server from list?", is_confirm=True):
                    i = idx[0]
                    self.proxies_data.pop(i)
                    self.save_proxies()
                    self.proxy_list.delete(i)
            else:
                self.show_custom_popup("Warning", "Select a server to delete")

        tk.Button(self.btn_frame, text="- REMOVE", command=delete_selected,
                  bg=self.colors["card2"], fg=self.colors["text"], font=("Segoe UI", 9, "bold"),
                  relief=tk.FLAT, bd=0, padx=12, cursor="hand2").pack(side=tk.RIGHT)

        # Initialize List selection
        self.initial_list_fill()
        default_idx = self.app_settings.get("default_proxy_idx", 0)
        if self.proxies_data:
            if default_idx < len(self.proxies_data):
                self.proxy_list.selection_set(default_idx)
            else:
                self.proxy_list.selection_set(0)

        # Footer links (small)
        footer = tk.Frame(self.content, bg=self.colors["bg"])
        footer.pack(fill=tk.X, padx=8, pady=14)
        socials = [("TG BOT", "https://t.me/intsecure_bot"), ("CHANNEL", "https://t.me/internet_secure")]
        for name, url in socials:
            lbl = tk.Label(footer, text=name, fg=self.colors["text_dim"], cursor="hand2", bg=self.colors["bg"], font=("Segoe UI", 9, "bold"))
            lbl.pack(side=tk.LEFT, padx=14, expand=True)
            lbl.bind("<Button-1>", lambda e, u=url: self.open_link(u))
    def __init__(self, root):
        self.root = root
        # Set basic title early (may be updated after settings load)
        self.root.title("iSecure VPN")
        # Phone-like portrait window
        self.root.geometry("390x720")
        try:
            self.root.minsize(360, 640)
        except:
            pass
        
        self.app_dir = self.get_app_dir()
        self.data_dir = self.get_data_dir()
        self.config_file = os.path.join(self.data_dir, "proxies.json")
        self.log_file = os.path.join(self.data_dir, "InternetSecure.log")
        self.settings_file = os.path.join(self.data_dir, "settings.json")
        # Core lives next to exe in onedir builds; in onefile it lives in _MEIPASS.
        self.core_path = self.resource_path("nekobox_core.exe")
        self.temp_config_path = None
        
        self.proxies_data = []
        self.load_proxies()
        
        # Set App Icon
        self.icon_path = self.resource_path("nekobox.png")
        if os.path.exists(self.icon_path):
            try:
                from PIL import Image, ImageTk
                img = Image.open(self.icon_path)
                self.app_icon = ImageTk.PhotoImage(img)
                self.root.iconphoto(False, self.app_icon)
            except Exception as e:
                try:
                    self.app_icon = tk.PhotoImage(file=self.icon_path)
                    self.root.iconphoto(False, self.app_icon)
                except:
                    print(f"Could not load icon: {e}")
        
        # Colors - InternetSecure landing style
        self.colors = {
            "bg": "#071427",          # base background (top)
            "card": "#0a1b34",        # panels (slightly softer)
            "card2": "#0b2446",       # inner panels
            "accent": "#20c7ff",      # softer cyan
            "accent2": "#2b7cff",     # blue
            "accent_hover": "#35ddff",
            "text": "#eaf6ff",
            "text_dim": "#b7d4e6",
            "status_off": "#ff3b3b",
            "status_on": "#16d981"
        }
        
        # App Settings
        self.app_settings = {
            "autostart": self.check_autostart_reg(),
            "run_as_admin": False,
            "autoconnect": False,
            "minimized": False,          # start minimized
            "minimize_to_tray": False,   # close-to-tray
            "rule_preset": "Global",
            "default_proxy_idx": 0,  # Index of default proxy
            # Compatibility toggles
            # Some sites (e.g. cursor.com on Vercel) may block specific VPN egress IPs and close connections.
            # This option routes cursor.com/cursor.sh directly (outside VPN) as a workaround.
            "compat_cursor_direct": False,
            # NekoBox-compatible mode:
            # Use a sing-box JSON template exported from original NekoBox and only swap the "proxy" outbound.
            "nekobox_compatible": False,
            "nekobox_template_json": "",
            # Network compatibility: disable QUIC (UDP/443) to avoid ERR_SSL_PROTOCOL_ERROR on some networks.
            "disable_quic": True,

            # --- NekoBox-like TUN/DNS defaults ---
            # Match typical NekoBox defaults seen in UI screenshots
            # Default (NekoBox-like) values
            "tun_stack": "system",          # system / gvisor
            "tun_mtu": 1500,
            "tun_strict_route": False,
            "tun_auto_route": True,
            "tun_ipv6": False,
            "dns_remote": "tls://8.8.8.8",
            # IMPORTANT: sing-box "local" transport cannot do raw queries (breaks IN HTTPS / SVCB records).
            # Use an actual DNS server IP by default to avoid mass DNS failures/timeouts.
            "dns_direct": "8.8.8.8"
            ,
            # Traffic stats: choose interface for fallback mode ("auto" or NIC name)
            # Use auto by default (Windows may name the adapter differently, or we may suffix to avoid collisions).
            "traffic_iface": "auto",
            
            # New features
            "language": "en",  # "en" or "ru"
            "log_level": 1,  # 0=DEBUG, 1=INFO, 2=WARNING, 3=ERROR
            "kill_switch": False,
            "auto_reconnect": False,
            "dns_leak_protection": False,
            "health_check_interval": 30,  # seconds
            "max_log_age_days": 7,
            "max_log_size_mb": 10
        }

        # Load persisted settings (then refresh autostart from registry to reflect actual state)
        self.load_settings()
        try:
            self.app_settings["autostart"] = self.check_autostart_reg()
        except:
            pass

        # Initialize localization
        try:
            lang = self.app_settings.get("language", "en")
            init_translator(lang)
            self.translator = get_translator()
        except Exception:
            self.translator = get_translator()  # Fallback to default

        # Initialize logger with level
        try:
            log_level_int = self.app_settings.get("log_level", 1)
            log_level = LogLevel(min(max(log_level_int, 0), 3))  # Clamp to 0-3
            max_log_age = self.app_settings.get("max_log_age_days", 7)
            max_log_size = self.app_settings.get("max_log_size_mb", 10)
            init_logger(self.log_file, log_level, max_log_age, max_log_size)
            self.app_logger = get_logger()
        except Exception:
            self.app_logger = None

        # Initialize VPN utilities
        try:
            self.kill_switch = KillSwitch(self.app_logger)
            self.dns_leak_protection = DNSLeakProtection(self.app_logger)
            health_interval = self.app_settings.get("health_check_interval", 30)
            self.health_check = VPNHealthCheck(self.app_logger, health_interval)
            self.auto_reconnect = AutoReconnect(self.app_logger)
            
            # Setup health check callback
            self.health_check.add_callback(self._on_health_check_result)
        except Exception as e:
            if self.app_logger:
                self.app_logger.error(f"Failed to initialize VPN utilities: {e}")
            self.kill_switch = None
            self.dns_leak_protection = None
            self.health_check = None
            self.auto_reconnect = None

        # If user wants always-elevated, relaunch once (guard to avoid loop if UAC cancelled)
        try:
            if (os.name == "nt"
                and self.app_settings.get("run_as_admin", False)
                and not self.is_admin()
                and os.getenv("ISECURE_RUA_TRIED") != "1"):
                os.environ["ISECURE_RUA_TRIED"] = "1"
                if self.relaunch_as_admin():
                    os._exit(0)
        except:
            pass

        # Window title: prefix with [Admin] when elevated
        try:
            admin_prefix = "[Admin] " if self.is_admin() else ""
        except Exception:
            admin_prefix = ""
        self.root.title(f"{admin_prefix}iSecure VPN")
        
        self.rule_var = tk.StringVar(value="Global")
        self.root.configure(bg=self.colors["bg"])
        self.root.resizable(False, False)

        self.process = None
        self.is_running = False
        self.start_time = None
        self._closing = False
        self._log_lock = threading.Lock()
        self._log_q = queue.Queue(maxsize=10000)

        def _log_writer_loop():
            buf = []
            last_flush = time.time()
            while not getattr(self, "_closing", False):
                try:
                    # Wait briefly for a message
                    try:
                        msg = self._log_q.get(timeout=0.25)
                        if msg is not None:
                            buf.append(str(msg))
                    except queue.Empty:
                        pass

                    now = time.time()
                    # Flush periodically or when buffer is large
                    if buf and (len(buf) >= 200 or (now - last_flush) >= 0.5):
                        try:
                            with self._log_lock:
                                with open(self.log_file, "a", encoding="utf-8") as f:
                                    f.write("\n".join(buf) + "\n")
                        except:
                            pass
                        buf.clear()
                        last_flush = now
                except:
                    # Never let logger thread die
                    time.sleep(0.2)

            # Final flush
            try:
                if buf:
                    with self._log_lock:
                        with open(self.log_file, "a", encoding="utf-8") as f:
                            f.write("\n".join(buf) + "\n")
            except:
                pass

        threading.Thread(target=_log_writer_loop, daemon=True).start()

        # Tray init + start minimized behavior (Windows)
        try:
            if self.app_settings.get("minimize_to_tray", False):
                self._ensure_tray()
            if self.app_settings.get("minimized", False):
                if self.app_settings.get("minimize_to_tray", False):
                    self.root.after(0, self._tray_hide)
                else:
                    self.root.after(0, self.root.iconify)
        except:
            pass

        # Ensure core is terminated on normal close and on interpreter exit
        try:
            self.root.protocol("WM_DELETE_WINDOW", self.on_close)
        except:
            pass
        try:
            atexit.register(self.cleanup_core)
        except:
            pass

        # --- UI Elements ---
        self.build_internetsecure_ui()
        # Fetch direct IP on startup
        try:
            self.update_ip_labels("direct")
        except:
            pass

        # Threads
        threading.Thread(target=self.update_pings, daemon=True).start()

        # Core check
        if not os.path.exists(self.core_path):
            self.root.after(1000, lambda: self.show_custom_popup("Core Error", "nekobox_core.exe not found! Place it in the app folder."))

    # --- Core Logic ---

    def load_proxies(self):
        if os.path.exists(self.config_file):
            try:
                with open(self.config_file, "r", encoding="utf-8") as f:
                    self.proxies_data = json.load(f)
            except:
                self.proxies_data = []
        else:
            self.proxies_data = []

        # NOTE: Current nekobox_core (sing-box 1.9.7-neko-1) in this bundle does NOT support SpiderX field.
        # Remove any stored SpiderX keys to avoid config decode failure.
        changed = False
        try:
            for p in self.proxies_data:
                cfg = p.get("config")
                if isinstance(cfg, dict):
                    tls = cfg.get("tls")
                    if isinstance(tls, dict):
                        reality = tls.get("reality")
                        if isinstance(reality, dict):
                            if "spiderX" in reality:
                                reality.pop("spiderX", None)
                                changed = True
                            if "spider_x" in reality:
                                reality.pop("spider_x", None)
                                changed = True
        except:
            pass
        
        # Do NOT seed any default servers. User manages servers explicitly.
        if changed and self.proxies_data:
            # Persist migration
            try:
                self.save_proxies()
            except:
                pass

    def save_proxies(self):
        # Store in per-user data dir; atomic write to avoid partial/corrupt files
        self.atomic_write_json(self.config_file, self.proxies_data)

    def initial_list_fill(self):
        self.proxy_list.delete(0, tk.END)
        default_idx = self.app_settings.get("default_proxy_idx", 0)
        
        for i, p in enumerate(self.proxies_data):
            name_parts = p['name'].split()
            code = "".join([word[0] for word in name_parts[:2]]).upper() if name_parts else "??"
            prefix = "★ " if i == default_idx else ""
            self.proxy_list.insert(tk.END, f" {prefix}[{code}]  {p['name']}  |  Checking...")

    def show_context_menu(self, event):
        try:
            # Select the item under cursor
            idx = self.proxy_list.nearest(event.y)
            self.proxy_list.selection_clear(0, tk.END)
            self.proxy_list.selection_set(idx)
            self.context_menu.post(event.x_root, event.y_root)
        except: pass

    def set_as_default(self):
        selection = self.proxy_list.curselection()
        if selection:
            idx = selection[0]
            # Move selected proxy to top of the data list
            proxy = self.proxies_data.pop(idx)
            self.proxies_data.insert(0, proxy)
            
            # Update setting to always point to index 0
            self.app_settings["default_proxy_idx"] = 0
            
            # Save and refresh UI
            self.save_proxies()
            self.initial_list_fill()
            self.proxy_list.selection_set(0)
            self.show_notification("Default Set", f"{proxy['name']} is now at the top and set as default.")

    def generate_basic_config(self, proxy_data, out_path=None):
        import copy
        # Prepare Clash API stats (NekoBox-like traffic counter)
        if not getattr(self, "clash_api_port", None):
            self.clash_api_port = self.pick_stats_port(9090)
        # rotate secret per config build/session
        self.clash_api_secret = uuid.uuid4().hex

        # Avoid Windows TUN name collisions (can hard-fail startup)
        tun_ifname = self.pick_tun_interface_name("iSecure-TUN")
        tun_inet4 = self.pick_tun_inet4_address("172.19.0.1/30")
        try:
            self.log_message(f"[GUI] TUN: ifname={tun_ifname} inet4={tun_inet4}")
        except:
            pass

        # --- NekoBox-compatible mode ---
        # If user provided a sing-box JSON template exported from NekoBox, we use it and only replace the proxy outbound.
        if self.app_settings.get("nekobox_compatible", False):
            template_raw = (self.app_settings.get("nekobox_template_json") or "").strip()
            if template_raw:
                try:
                    tpl = json.loads(template_raw)
                    if not isinstance(tpl, dict) or "outbounds" not in tpl or not isinstance(tpl["outbounds"], list):
                        raise ValueError("Template must be a sing-box JSON object with an 'outbounds' list.")
                    tpl = self.sanitize_template(tpl)

                    outbound = copy.deepcopy(proxy_data.get("config", {"type": "direct"}))
                    outbound["tag"] = "proxy"

                    # Current core doesn't support Reality SpiderX field; strip if present.
                    try:
                        tls = outbound.get("tls")
                        if isinstance(tls, dict):
                            reality = tls.get("reality")
                            if isinstance(reality, dict):
                                reality.pop("spiderX", None)
                                reality.pop("spider_x", None)
                    except:
                        pass

                    # Find the outbound to replace: tag == "proxy", else first non-direct/dns outbound, else index 0.
                    ob_idx = None
                    for i, ob in enumerate(tpl["outbounds"]):
                        if isinstance(ob, dict) and ob.get("tag") == "proxy":
                            ob_idx = i
                            break
                    if ob_idx is None:
                        for i, ob in enumerate(tpl["outbounds"]):
                            if isinstance(ob, dict) and ob.get("type") not in ("direct", "dns"):
                                ob_idx = i
                                break
                    if ob_idx is None:
                        ob_idx = 0

                    tpl["outbounds"][ob_idx] = outbound

                    # Ensure template tun inbound doesn't collide (templates often hardcode interface/ip)
                    try:
                        ibs = tpl.get("inbounds")
                        if isinstance(ibs, list):
                            for ib in ibs:
                                if isinstance(ib, dict) and (ib.get("type") or "").lower() == "tun":
                                    ib["interface_name"] = tun_ifname
                                    ib["inet4_address"] = tun_inet4
                                    break
                    except:
                        pass

                    # Prevent routing loop: always allow connection to the proxy server itself via direct.
                    proxy_host = proxy_data.get("host", "")
                    exclusion_rule = {"ip_cidr": [f"{proxy_host}/32"], "outbound": "direct"}
                    if any(c.isalpha() for c in proxy_host):
                        exclusion_rule = {"domain": [proxy_host], "outbound": "direct"}

                    # Optional compatibility: bypass Cursor domains
                    compat_rules = []
                    if self.app_settings.get("compat_cursor_direct", False):
                        compat_rules.append({"domain_suffix": ["cursor.com", "cursor.sh"], "outbound": "direct"})

                    # Apply traffic preset as a SAFE prepend (do not rewrite template routing completely)
                    preset = self.app_settings.get("rule_preset", "Global")
                    prepend = []
                    if preset == "Bypass RU":
                        prepend = [{"geoip": ["ru"], "outbound": "direct"}]
                    elif preset == "Bypass LAN+RU":
                        prepend = [{"ip_is_private": True, "outbound": "direct"}, {"geoip": ["ru"], "outbound": "direct"}]
                    elif preset == "Socials Only":
                        # Inverse mode: default direct, only socials via proxy (implemented by putting socials rule first + setting final)
                        prepend = [{"geosite": ["telegram", "youtube", "instagram", "facebook"], "outbound": "proxy"}]
                        tpl.setdefault("route", {})["final"] = "direct"
                    elif preset == "Streaming":
                        prepend = [{"geosite": ["netflix", "disney", "hulu", "primevideo", "youtube"], "outbound": "proxy"}]
                        tpl.setdefault("route", {})["final"] = "direct"

                    tpl.setdefault("route", {})
                    tpl_route_rules = tpl["route"].get("rules", [])
                    if not isinstance(tpl_route_rules, list):
                        tpl_route_rules = []

                    # Ensure our loop-avoid and optional compat rules are first, then preset rules, then the template rules.
                    # Optional: disable QUIC (UDP/443) so browsers fall back to TCP (fixes many SSL/protocol errors)
                    quic_rules = []
                    if self.app_settings.get("disable_quic", True):
                        quic_rules.append({"network": "udp", "port": [443], "outbound": "block"})
                        # Ensure block outbound exists in template
                        has_block = any(isinstance(ob, dict) and ob.get("tag") == "block" for ob in tpl.get("outbounds", []))
                        if not has_block:
                            tpl["outbounds"].append({"type": "block", "tag": "block"})

                    tpl["route"]["rules"] = quic_rules + compat_rules + [exclusion_rule] + prepend + tpl_route_rules

                    # Ensure we have reasonable logging (leave template as-is if it already has log config)
                    tpl.setdefault("log", {})
                    if isinstance(tpl["log"], dict):
                        tpl["log"].setdefault("level", "info")
                        tpl["log"].setdefault("timestamp", True)

                    # Enable Clash API on localhost for stats (like NekoBox)
                    tpl.setdefault("experimental", {})
                    if isinstance(tpl["experimental"], dict):
                        tpl["experimental"]["clash_api"] = {
                            "external_controller": f"127.0.0.1:{self.clash_api_port}",
                            "secret": self.clash_api_secret
                        }

                    if not out_path:
                        out_path = os.path.join(self.data_dir, "temp_config.json")
                    self.atomic_write_json(out_path, tpl)
                    # Keep last config for debugging (even if temp file is deleted on stop)
                    try:
                        self.atomic_write_json(os.path.join(self.data_dir, "last_config.json"), tpl)
                    except:
                        pass
                    return
                except Exception as e:
                    # Fall back to legacy config and record reason in log (GUI will still run)
                    try:
                        self.log_message(f"[GUI] NekoBox-compatible template error: {e}")
                    except:
                        pass
            else:
                try:
                    self.log_message("[GUI] NekoBox-compatible enabled but template is empty; using legacy config.")
                except:
                    pass

        # --- Legacy (custom) config generator ---
        outbound = copy.deepcopy(proxy_data.get("config", {"type": "direct"}))
        
        # Ensure tag is proxy
        outbound["tag"] = "proxy"

        # Current core doesn't support Reality SpiderX field; strip if present.
        try:
            tls = outbound.get("tls")
            if isinstance(tls, dict):
                reality = tls.get("reality")
                if isinstance(reality, dict):
                    reality.pop("spiderX", None)
                    reality.pop("spider_x", None)
        except:
            pass
        
        # SSL FIX: flow MUST NOT exist if TLS is disabled or not properly set
        if not outbound.get("tls") or not isinstance(outbound.get("tls"), dict):
            outbound.pop("flow", None)
        
        # Do not force outbound domain strategy in legacy mode; NekoBox defaults vary and forcing can break some sites.
        outbound.pop("domain_strategy", None)

        preset = self.app_settings.get("rule_preset", "Global")
        rules = []
        final_outbound = "proxy"

        if preset == "Bypass RU":
            rules = [{"geoip": ["ru"], "outbound": "direct"}]
            final_outbound = "proxy"
        elif preset == "Bypass LAN+RU":
            rules = [
                {"ip_is_private": True, "outbound": "direct"},
                {"geoip": ["ru"], "outbound": "direct"}
            ]
            final_outbound = "proxy"
        elif preset == "Socials Only":
            rules = [{"geosite": ["telegram", "youtube", "instagram", "facebook"], "outbound": "proxy"}]
            final_outbound = "direct"
        elif preset == "Streaming":
            rules = [{"geosite": ["netflix", "disney", "hulu", "primevideo", "youtube"], "outbound": "proxy"}]
            final_outbound = "direct"
        else: # Global
            rules = []
            final_outbound = "proxy"

        # Core Route Configuration
        # Prepare rules for proxy server exclusion
        proxy_host = proxy_data.get('host', '')
        exclusion_rule = {"ip_cidr": [f"{proxy_host}/32"], "outbound": "direct"}
        # If host is a domain, use domain rule instead
        if any(c.isalpha() for c in proxy_host):
            exclusion_rule = {"domain": [proxy_host], "outbound": "direct"}

        # If proxy host is a domain, ensure its DNS resolution uses dns-direct (bootstrap) instead of proxy DNS.
        dns_rules_extra = []
        if any(c.isalpha() for c in proxy_host):
            dns_rules_extra.append({"domain": [proxy_host], "server": "dns-direct"})

        # Optional compatibility: bypass Cursor domains (some VPN exits get blocked)
        compat_rules = []
        if self.app_settings.get("compat_cursor_direct", False):
            compat_rules.append({"domain_suffix": ["cursor.com", "cursor.sh"], "outbound": "direct"})

        dns_remote = (self.app_settings.get("dns_remote", "tls://8.8.8.8") or "tls://8.8.8.8").strip()
        dns_direct_raw = (self.app_settings.get("dns_direct", "8.8.8.8") or "8.8.8.8").strip()
        # Normalize "local/localhost/system" to a real upstream to support raw query types (HTTPS/SVCB).
        if dns_direct_raw.lower() in ("local", "localhost", "system", "windows"):
            dns_direct = "8.8.8.8"
        else:
            dns_direct = dns_direct_raw
        tun_stack = self.app_settings.get("tun_stack", "system")
        tun_mtu = int(self.app_settings.get("tun_mtu", 1500))
        tun_strict = bool(self.app_settings.get("tun_strict_route", True))
        tun_auto = bool(self.app_settings.get("tun_auto_route", True))
        tun_ipv6 = bool(self.app_settings.get("tun_ipv6", False))

        config = {
            # Reduce log verbosity to limit sensitive metadata leakage
            "log": {"level": "warn", "timestamp": True},
            "experimental": {
                "clash_api": {
                    "external_controller": f"127.0.0.1:{self.clash_api_port}",
                    "secret": self.clash_api_secret
                }
            },
            "dns": {
                "servers": [
                    {"tag": "dns-remote", "address": dns_remote, "detour": "proxy"},
                    {"tag": "dns-direct", "address": dns_direct, "detour": "direct"}
                ],
                "rules": [
                    # Use direct DNS only for direct outbound requests (prevents leaking all queries to "direct")
                    *dns_rules_extra,
                    {"outbound": "direct", "server": "dns-direct"}
                ],
                "final": "dns-remote",
                "strategy": "ipv4_only"
            },
            "inbounds": [
                {
                    "type": "tun",
                    "tag": "tun-in",
                    "interface_name": tun_ifname,
                    "inet4_address": tun_inet4,
                    "mtu": tun_mtu,
                    "auto_route": tun_auto,
                    "strict_route": tun_strict,
                    "stack": tun_stack,
                    "sniff": True,
                    "inet6_address": "fd00::1/126" if tun_ipv6 else None
                },
                {"type": "mixed", "listen": "127.0.0.1", "listen_port": 2080}
            ],
            "outbounds": [
                outbound, 
                {"type": "direct", "tag": "direct"},
                {"type": "dns", "tag": "dns-out"},
                {"type": "block", "tag": "block"}
            ],
            "route": {
                "rules": (
                    ([{"network": "udp", "port": [443], "outbound": "block"}] if self.app_settings.get("disable_quic", True) else [])
                    + [
                        {"protocol": "dns", "outbound": "dns-out"},
                        *compat_rules,
                        {"ip_is_private": True, "outbound": "direct"},
                        exclusion_rule # Dynamic rule to avoid loop
                    ]
                    + rules
                ),
                "geoip": {"path": self.resource_path("geoip.db")},
                "geosite": {"path": self.resource_path("geosite.db")},
                "final": final_outbound,
                "auto_detect_interface": True
            }
        }
        # Remove None fields (sing-box will error on unknown/null in some places)
        try:
            tun = config["inbounds"][0]
            if tun.get("inet6_address") is None:
                tun.pop("inet6_address", None)
        except:
            pass
        if not out_path:
            out_path = os.path.join(self.data_dir, "temp_config.json")
        self.atomic_write_json(out_path, config)
        try:
            self.atomic_write_json(os.path.join(self.data_dir, "last_config.json"), config)
        except:
            pass

    def load_nekobox_template_from_file(self):
        path = filedialog.askopenfilename(
            title="Select NekoBox sing-box template JSON",
            filetypes=[("JSON files", "*.json"), ("All files", "*.*")]
        )
        if not path:
            return
        try:
            with open(path, "r", encoding="utf-8", errors="ignore") as f:
                raw = f.read()
            # Validate basic structure
            data = json.loads(raw)
            if not isinstance(data, dict) or "outbounds" not in data:
                raise ValueError("Not a valid sing-box JSON template (missing 'outbounds').")
            self.app_settings["nekobox_template_json"] = raw
            self.show_notification("Template Loaded", os.path.basename(path))
        except Exception as e:
            self.show_custom_popup("Error", f"Failed to load template: {e}")

    def paste_nekobox_template(self):
        w = tk.Toplevel(self.root)
        w.title("Paste NekoBox Template")
        w.geometry("620x520")
        w.configure(bg=self.colors["bg"])
        self.root.update_idletasks()
        w.geometry(f"+{self.root.winfo_x() + 30}+{self.root.winfo_y() + 30}")
        w.transient(self.root); w.grab_set()

        tk.Label(w, text="Paste sing-box JSON exported from NekoBox", bg=self.colors["bg"], fg=self.colors["text"], font=("Segoe UI", 11, "bold")).pack(pady=10)
        txt = tk.Text(w, bg=self.colors["card"], fg=self.colors["text"], insertbackground="white", font=("Consolas", 9), wrap="none")
        txt.pack(fill=tk.BOTH, expand=True, padx=15, pady=10)
        existing = self.app_settings.get("nekobox_template_json", "")
        if existing:
            txt.insert("1.0", existing)

        def save():
            raw = txt.get("1.0", tk.END).strip()
            try:
                data = json.loads(raw)
                if not isinstance(data, dict) or "outbounds" not in data:
                    raise ValueError("Template must be a sing-box JSON object with 'outbounds'.")
                self.app_settings["nekobox_template_json"] = raw
                self.show_notification("Template Saved", "OK")
                w.destroy()
            except Exception as e:
                self.show_custom_popup("Error", f"Invalid JSON: {e}")

        tk.Button(w, text="SAVE TEMPLATE", command=save, bg=self.colors["accent"], fg="white",
                  font=("Segoe UI", 10, "bold"), relief=tk.FLAT, bd=0, height=2, cursor="hand2").pack(pady=10)

    def toggle_vpn(self):
        if not self.is_running:
            self.start_vpn()
        else:
            self.stop_vpn()

    def _on_health_check_result(self, is_healthy):
        """Callback for health check results."""
        try:
            if not is_healthy and self.is_running:
                if self.app_logger:
                    self.app_logger.warning("VPN health check failed")
                # Trigger auto reconnect if enabled
                if self.app_settings.get("auto_reconnect", False) and self.auto_reconnect:
                    self.root.after(0, lambda: self.auto_reconnect.on_connection_lost())
        except:
            pass
    
    def log_message(self, message):
        # Async buffered logging (prevents UI hangs under heavy core output)
        try:
            s = str(message)
            # Strip ANSI color sequences (core uses them)
            s = re.sub(r"\x1b\[[0-9;]*m", "", s)
            try:
                self._log_q.put_nowait(s)
            except queue.Full:
                # Drop if log flood; keep UI responsive
                pass
            # Also log to new logger if available
            if hasattr(self, 'app_logger') and self.app_logger:
                if '[GUI]' in s or '[SECURITY]' in s:
                    self.app_logger.info(s)
                elif 'ERROR' in s or 'FATAL' in s:
                    self.app_logger.error(s)
                elif 'WARNING' in s or 'WARN' in s:
                    self.app_logger.warning(s)
                else:
                    self.app_logger.debug(s)
        except:
            pass

    def is_admin(self):
        try:
            import ctypes
            return ctypes.windll.shell32.IsUserAnAdmin()
        except:
            return False

    def start_vpn(self):
        if not self.is_admin():
            self.show_custom_popup("Admin Required", "TUN mode requires Administrator privileges. Please restart the app as Admin.")
            return
        
        try:
            if not self.proxies_data:
                self.show_custom_popup("Warning", "Server list is empty. Add a server first.")
                return

            selection = self.proxy_list.curselection()
            if selection:
                idx = selection[0]
            else:
                # Auto-pick: default server if set, else first server
                idx = int(self.app_settings.get("default_proxy_idx", 0) or 0)
                if idx < 0 or idx >= len(self.proxies_data):
                    idx = 0
                # Reflect the choice in UI
                try:
                    self.proxy_list.selection_clear(0, tk.END)
                    self.proxy_list.selection_set(idx)
                    self.proxy_list.activate(idx)
                    self.proxy_list.see(idx)
                except:
                    pass

            proxy_data = self.proxies_data[idx]

            # Prevent double-start (spamming START can wedge core + UI)
            if getattr(self, "start_in_progress", False):
                return
            self.start_in_progress = True

            # UI: update immediately (no freeze)
            try:
                self.toggle_btn.config(text="STARTING…", state=tk.DISABLED, bg=self.colors["card2"])
                self.status_label.config(text="Starting…", fg=self.colors["text_dim"])
            except:
                pass

            def worker():
                """
                Heavy work off the Tkinter thread:
                - write logs
                - integrity check (SHA256)
                - generate/write config
                - spawn core process
                """
                try:
                    # Clear log for new session (do not keep old sensitive metadata around)
                    try:
                        with self._log_lock:
                            with open(self.log_file, "w", encoding="utf-8") as f:
                                f.write(f"--- iSecure VPN Session Start: {time.strftime('%Y-%m-%d %H:%M:%S')} ---\n")
                    except:
                        pass

                    # Verify core integrity if a pinned hash exists
                    if not os.path.exists(self.core_path):
                        raise FileNotFoundError("nekobox_core.exe not found")
                    if not self.verify_core_integrity(self.core_path):
                        raise RuntimeError("Core integrity check failed (hash mismatch)")

                    # Unique temp config per session (avoid predictable filenames)
                    self.temp_config_path = os.path.join(self.data_dir, f"temp_config_{uuid.uuid4().hex}.json")
                    self.generate_basic_config(proxy_data, out_path=self.temp_config_path)
                    try:
                        self.log_message(f"[GUI] Stats: Clash API controller 127.0.0.1:{getattr(self,'clash_api_port',None)}")
                    except:
                        pass

                    # Reset last-fatal marker
                    self.last_core_fatal = None

                    proc = subprocess.Popen(
                        [self.core_path, "run", "-c", self.temp_config_path],
                        stdout=subprocess.PIPE,
                        stderr=subprocess.STDOUT,
                        text=True,
                        bufsize=1,
                        creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0,
                        cwd=self.app_dir
                    )
                    self.process = proc

                    def on_started():
                        self.is_running = True
                        self.start_time = time.time()
                        self.toggle_btn.config(text="STOP VPN", state=tk.NORMAL, bg=self.colors["status_off"])
                        self.status_label.config(text="Connecting…", fg=self.colors["text_dim"])

                        def post_start_check():
                            try:
                                if not self.process:
                                    return
                                if self.process.poll() is not None:
                                    msg = "Core exited immediately. Check InternetSecure.log for details."
                                    if self.last_core_fatal == "tun_exists":
                                        msg = (
                                            "TUN interface name conflict.\n\n"
                                            "Close/stop other NekoBox instances (nekobox_core.exe) and try again.\n"
                                            "If it still fails, disable/remove the existing 'iSecure-TUN' adapter in Windows network adapters."
                                        )
                                    elif self.last_core_fatal == "tun_ip_exists":
                                        msg = (
                                            "TUN IPv4 address conflict.\n\n"
                                            "Try closing other VPN clients and retry.\n"
                                            "If it persists, disable/remove the existing sing-tun adapter in Windows network adapters."
                                        )
                                    self.stop_vpn()
                                    self.show_custom_popup("Core Error", msg)
                                    return
                                self.status_label.config(text="Connected", fg=self.colors["status_on"])
                                self.show_notification("VPN Status", f"Connected to {proxy_data['name']}")
                                self.update_timer()
                            except:
                                pass

                        self.root.after(1200, post_start_check)
                        threading.Thread(target=self.read_output, daemon=True).start()

                        # Update IPs and start traffic monitor
                        try:
                            self.refresh_vpn_ip_with_retries()
                        except:
                            pass
                        try:
                            self.root.after(800, self.start_interface_stats_monitor)
                            self.root.after(1500, self.start_core_stats_monitor)
                        except:
                            pass

                        # Activate kill switch if enabled
                        try:
                            if self.app_settings.get("kill_switch", False) and self.kill_switch:
                                vpn_interface = getattr(self, 'tun_interface_name', None)
                                self.kill_switch.activate(vpn_interface)
                                if self.app_logger:
                                    self.app_logger.info("Kill Switch activated")
                        except Exception as e:
                            if self.app_logger:
                                self.app_logger.error(f"Kill Switch activation failed: {e}")

                        # Start health check if enabled
                        try:
                            if self.health_check:
                                self.health_check.start()
                                if self.app_logger:
                                    self.app_logger.info("Health check started")
                        except Exception as e:
                            if self.app_logger:
                                self.app_logger.error(f"Health check start failed: {e}")

                        # Enable auto reconnect if enabled
                        try:
                            if self.app_settings.get("auto_reconnect", False) and self.auto_reconnect:
                                self.auto_reconnect.enable(self._auto_reconnect_callback)
                                if self.app_logger:
                                    self.app_logger.info("Auto reconnect enabled")
                        except Exception as e:
                            if self.app_logger:
                                self.app_logger.error(f"Auto reconnect enable failed: {e}")

                        # DNS leak protection check
                        try:
                            if self.app_settings.get("dns_leak_protection", False) and self.dns_leak_protection:
                                # Check DNS leak after connection
                                self.root.after(3000, self._check_dns_leak)
                        except Exception as e:
                            if self.app_logger:
                                self.app_logger.error(f"DNS leak protection setup failed: {e}")

                    self.root.after(0, on_started)
                except Exception as e:
                    def on_fail():
                        try:
                            self.toggle_btn.config(text="START VPN", state=tk.NORMAL, bg=self.colors["accent"])
                            self.status_label.config(text="Disconnected", fg=self.colors["status_off"])
                        except:
                            pass
                        self.show_custom_popup("Error", f"Failed to start: {e}")
                    self.root.after(0, on_fail)
                finally:
                    self.start_in_progress = False

            threading.Thread(target=worker, daemon=True).start()
            
        except Exception as e:
            self.show_custom_popup("Error", f"Failed to start: {e}")
            self.start_in_progress = False

    def _auto_reconnect_callback(self):
        """Callback for auto reconnect."""
        try:
            if not self.is_running:
                self.start_vpn()
        except:
            pass

    def _check_dns_leak(self):
        """Check for DNS leaks."""
        try:
            if self.dns_leak_protection and self.is_running:
                vpn_dns = [self.app_settings.get("dns_remote", "").replace("tls://", "").replace("://", "")]
                is_leak, details = self.dns_leak_protection.check_dns_leak(vpn_dns)
                if is_leak:
                    if self.app_logger:
                        self.app_logger.warning(f"DNS leak detected: {details.get('leaks', [])}")
                    self.show_custom_popup(_("dns_leak_detected"), f"DNS servers: {', '.join(details.get('leaks', []))}")
        except:
            pass

    def stop_vpn(self):
        # Stop health check
        try:
            if self.health_check:
                self.health_check.stop()
        except:
            pass

        # Disable auto reconnect
        try:
            if self.auto_reconnect:
                self.auto_reconnect.disable()
        except:
            pass

        # Deactivate kill switch
        try:
            if self.kill_switch:
                self.kill_switch.deactivate()
        except:
            pass

        self.cleanup_core()
        
        self.is_running = False
        self.start_time = None
        self.toggle_btn.config(text="START VPN", bg=self.colors["accent"])
        self.status_label.config(text="Disconnected", fg=self.colors["status_off"])
        self.timer_label.config(text="00:00:00")
        self.show_notification("VPN Status", "Disconnected successfully!", status="error")

        # Stop traffic monitor & clear VPN IP
        try:
            self.traffic_stop_flag = True
        except:
            pass
        try:
            self.ip_vpn_val_full = ""
            try:
                self.traffic_dl.config(text="--")
                self.traffic_ul.config(text="--")
                self.traffic_total.config(text="--")
            except:
                pass
            self.speed_val.config(text="↓ --   ↑ --")
            self.refresh_current_ip_field()
        except:
            pass
        # Refresh direct IP after disconnect (so the single field shows non-VPN IP)
        try:
            self.update_ip_labels("direct")
        except:
            pass
        
        if self.temp_config_path and os.path.exists(self.temp_config_path):
            try:
                os.remove(self.temp_config_path)
            except:
                pass
        self.temp_config_path = None

    def cleanup_core(self):
        """
        Terminate nekobox_core reliably (and its children) so it doesn't survive GUI exit.
        Best-effort: terminate -> short wait -> kill.
        """
        proc = getattr(self, "process", None)
        if not proc:
            return
        self.process = None
        try:
            pid = getattr(proc, "pid", None)
        except:
            pid = None

        # Prefer psutil process tree kill on Windows
        try:
            import psutil
            if pid:
                try:
                    p = psutil.Process(int(pid))
                    children = []
                    try:
                        children = p.children(recursive=True)
                    except:
                        children = []
                    # terminate children first
                    for c in children:
                        try:
                            c.terminate()
                        except:
                            pass
                    try:
                        p.terminate()
                    except:
                        pass
                    try:
                        psutil.wait_procs(children + [p], timeout=1.8)
                    except:
                        pass
                    # hard kill if still alive
                    for c in children:
                        try:
                            if c.is_running():
                                c.kill()
                        except:
                            pass
                    try:
                        if p.is_running():
                            p.kill()
                    except:
                        pass
                    return
                except:
                    pass
        except:
            pass

        # Fallback: Popen terminate/kill
        try:
            proc.terminate()
        except:
            pass
        try:
            proc.wait(timeout=1.5)
        except:
            try:
                proc.kill()
            except:
                pass

    def on_close(self):
        """
        Window close handler:
        - If minimize-to-tray enabled: hide to tray
        - Else: full exit
        """
        try:
            if self.app_settings.get("minimize_to_tray", False) and self._tray_supported():
                self._ensure_tray()
                self._tray_hide()
                return
        except:
            pass
        if getattr(self, "_closing", False):
            try:
                self.root.destroy()
            except:
                pass
            return
        self._closing = True
        try:
            try:
                self.save_settings()
            except:
                pass
            # Stop monitors + core
            try:
                self.traffic_stop_flag = True
            except:
                pass
            self.cleanup_core()
        finally:
            try:
                # Stop tray icon if present
                if getattr(self, "_tray_icon", None) is not None:
                    try:
                        self._tray_icon.stop()
                    except:
                        pass
                    self._tray_icon = None
            except:
                pass
            try:
                # Release mutex handle if present
                if hasattr(self, "_mutex_handle") and self._mutex_handle is not None:
                    try:
                        import ctypes
                        from ctypes import wintypes
                        kernel32 = ctypes.windll.kernel32
                        kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
                        kernel32.CloseHandle(self._mutex_handle)
                    except:
                        pass
                    self._mutex_handle = None
            except:
                pass
            try:
                self.root.destroy()
            except:
                pass

    def read_output(self):
        if self.process:
            try:
                for line in self.process.stdout:
                    clean_line = line.strip()
                    if clean_line:
                        # Clean ANSI sequences early too (keeps log smaller and parsing easier)
                        try:
                            clean_line = re.sub(r"\x1b\[[0-9;]*m", "", clean_line)
                        except:
                            pass
                        self.log_message(clean_line)
                        try:
                            if "Cannot create a file when that file already exists" in clean_line and "tun" in clean_line.lower():
                                self.last_core_fatal = "tun_exists"
                            if "set ipv4 address" in clean_line.lower() and "object already exists" in clean_line.lower():
                                self.last_core_fatal = "tun_ip_exists"
                            # Detect Windows WFP/AV connection abort spam; show one-time hint.
                            if ("wsasend" in clean_line.lower()
                                and "aborted by the software in your host machine" in clean_line.lower()):
                                now = time.time()
                                try:
                                    self._wsasend_hits = [t for t in getattr(self, "_wsasend_hits", []) if now - t < 10]
                                except:
                                    self._wsasend_hits = []
                                self._wsasend_hits.append(now)
                                if len(self._wsasend_hits) >= 3 and not getattr(self, "_wsasend_hint_shown", False):
                                    self._wsasend_hint_shown = True
                                    def hint():
                                        self.show_notification(
                                            "Network conflict",
                                            "Windows is aborting TUN connections (AV/Firewall/other VPN). "
                                            "Try disabling other VPN/filter apps, or keep TUN stack=gVisor and MTU=1280.",
                                            status="error",
                                        )
                                    self.root.after(0, hint)
                        except:
                            pass
                        # Optional: also print to python console for debugging
                        # print(f"[Core]: {clean_line}")
            except: pass

    # --- UI Helpers ---

    def show_custom_popup(self, title, message, is_confirm=False):
        popup = tk.Toplevel(self.root)
        popup.title(title)
        popup.configure(bg=self.colors["bg"])
        popup.resizable(False, False)
        
        w, h = 320, 200
        self.root.update_idletasks()
        x = self.root.winfo_x() + (self.root.winfo_width() // 2) - (w // 2)
        y = self.root.winfo_y() + (self.root.winfo_height() // 2) - (h // 2)
        popup.geometry(f"{w}x{h}+{x}+{y}")
        popup.transient(self.root)
        popup.grab_set()

        tk.Label(popup, text=title.upper(), font=("Segoe UI", 10, "bold"), 
                 bg=self.colors["bg"], fg=self.colors["accent"]).pack(pady=(20, 10))
        tk.Label(popup, text=message, font=("Segoe UI", 9), wraplength=280,
                 bg=self.colors["bg"], fg=self.colors["text"]).pack(pady=10)

        result = {"val": False}
        def close(val):
            result["val"] = val
            popup.destroy()

        btn_frame = tk.Frame(popup, bg=self.colors["bg"])
        btn_frame.pack(side=tk.BOTTOM, pady=20)

        if is_confirm:
            tk.Button(btn_frame, text="YES", command=lambda: close(True), width=10,
                      bg=self.colors["accent"], fg="white", relief=tk.FLAT, bd=0, cursor="hand2").pack(side=tk.LEFT, padx=10)
            tk.Button(btn_frame, text="NO", command=lambda: close(False), width=10,
                      bg=self.colors["card"], fg="white", relief=tk.FLAT, bd=0, cursor="hand2").pack(side=tk.LEFT, padx=10)
        else:
            tk.Button(btn_frame, text="OK", command=lambda: close(True), width=15,
                      bg=self.colors["accent"], fg="white", relief=tk.FLAT, bd=0, cursor="hand2").pack()

        self.root.wait_window(popup)
        return result["val"]

    def show_notification(self, title, message, status="success"):
        toast = tk.Toplevel(self.root)
        toast.overrideredirect(True)
        toast.attributes("-topmost", True)
        toast.attributes("-alpha", 0.0)
        toast.configure(bg=self.colors["card"])
        
        success_mp3 = self.resource_path("success_start.mp3")
        failed_mp3 = self.resource_path("failed_start.mp3")
        if status == "success" and os.path.exists(success_mp3):
            threading.Thread(target=lambda: self.play_sound(success_mp3), daemon=True).start()
        elif status == "error" and os.path.exists(failed_mp3):
            threading.Thread(target=lambda: self.play_sound(failed_mp3), daemon=True).start()

        accent_color = self.colors["status_on"] if status == "success" else self.colors["status_off"]
        w, h = 250, 80
        sw, sh = self.root.winfo_screenwidth(), self.root.winfo_screenheight()
        toast.geometry(f"{w}x{h}+{sw - w - 20}+{sh - h - 60}")

        tk.Frame(toast, bg=accent_color, height=4).pack(side=tk.TOP, fill=tk.X)
        content = tk.Frame(toast, bg=self.colors["card"], padx=15, pady=10)
        content.pack(fill=tk.BOTH, expand=True)
        tk.Label(content, text=title, font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["text"]).pack(anchor="w")
        tk.Label(content, text=message, font=("Segoe UI", 9), bg=self.colors["card"], fg=self.colors["text_dim"]).pack(anchor="w")

        def fade_in():
            alpha = toast.attributes("-alpha")
            if alpha < 1.0:
                toast.attributes("-alpha", alpha + 0.1)
                self.root.after(20, fade_in)
        
        def fade_out():
            alpha = toast.attributes("-alpha")
            if alpha > 0.0:
                toast.attributes("-alpha", alpha - 0.1)
                self.root.after(20, fade_out)
            else: toast.destroy()

        fade_in()
        self.root.after(3000, fade_out)

    def play_sound(self, file):
        try:
            from playsound import playsound
            playsound(file)
        except: pass

    def update_pings(self):
        while True:
            default_idx = self.app_settings.get("default_proxy_idx", 0)
            for i, p in enumerate(self.proxies_data):
                param = '-n' if platform.system().lower() == 'windows' else '-c'
                command = ['ping', param, '1', '-w', '1000', p["host"]]
                try:
                    start = time.time()
                    output = subprocess.run(command, capture_output=True, text=True, creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
                    end = time.time()
                    ping_str = f"{int((end - start) * 1000)}ms" if output.returncode == 0 else "Timeout"
                except: ping_str = "Error"
                
                # Tkinter is not thread-safe: update UI via main thread
                try:
                    name_parts = p['name'].split()
                    code = "".join([word[0] for word in name_parts[:2]]).upper() if name_parts else "??"
                    prefix = "★ " if i == default_idx else ""
                    text = f" {prefix}[{code}]  {p['name']}  |  {ping_str}"
                    def apply(idx=i, txt=text):
                        try:
                            # Keep list size stable
                            if idx < self.proxy_list.size():
                                self.proxy_list.delete(idx)
                                self.proxy_list.insert(idx, txt)
                        except:
                            pass
                    self.root.after(0, apply)
                except:
                    pass
            time.sleep(10)

    def update_timer(self):
        if self.is_running and self.start_time:
            elapsed = int(time.time() - self.start_time)
            h, rem = divmod(elapsed, 3600)
            m, s = divmod(rem, 60)
            self.timer_label.config(text=f"{h:02}:{m:02}:{s:02}")
            self.root.after(1000, self.update_timer)

    def open_link(self, url):
        import webbrowser
        webbrowser.open(url)

    # --- Windows Registry ---

    def check_autostart_reg(self):
        if os.name != 'nt': return False
        try:
            key = winreg.OpenKey(winreg.HKEY_CURRENT_USER, r"Software\Microsoft\Windows\CurrentVersion\Run", 0, winreg.KEY_READ)
            winreg.QueryValueEx(key, "iSecureVPN")
            winreg.CloseKey(key)
            return True
        except: return False

    def toggle_autostart(self, var):
        if os.name != 'nt': return
        val = var.get()
        key = winreg.OpenKey(winreg.HKEY_CURRENT_USER, r"Software\Microsoft\Windows\CurrentVersion\Run", 0, winreg.KEY_WRITE)
        if val:
            # Prefer sys.executable when frozen (PyInstaller), else script path.
            exe_path = sys.executable if getattr(sys, "frozen", False) else os.path.abspath(sys.argv[0])
            winreg.SetValueEx(key, "iSecureVPN", 0, winreg.REG_SZ, f'"{exe_path}"')
        else:
            try: winreg.DeleteValue(key, "iSecureVPN")
            except: pass
        winreg.CloseKey(key)
        self.app_settings["autostart"] = val
        try:
            self.save_settings()
        except:
            pass

    # --- Windows Management ---

    def run_latency_test(self):
        self.latency_res.config(text="Testing...", fg=self.colors["accent"])
        def task():
            import http.client
            import time
            from urllib.parse import urlparse
            
            results = []
            url = "https://cp.cloudflare.com/"
            parsed = urlparse(url)
            
            def single_ping():
                try:
                    start = time.time()
                    conn = http.client.HTTPSConnection(parsed.netloc, timeout=6)
                    conn.request("GET", parsed.path if parsed.path else "/")
                    res = conn.getresponse()
                    res.read()
                    end = time.time()
                    results.append((end - start) * 1000)
                except:
                    pass

            threads = []
            for _ in range(10):
                t = threading.Thread(target=single_ping)
                t.start()
                threads.append(t)
            
            for t in threads: t.join()
            
            def apply_ok(txt, color):
                try:
                    self.latency_res.config(text=txt, fg=color)
                except:
                    pass
            if results:
                avg = sum(results) / len(results)
                self.root.after(0, lambda: apply_ok(f"Avg: {int(avg)} ms | Success: {len(results)}/10", "#0f0"))
            else:
                self.root.after(0, lambda: apply_ok("Test failed (Timeout)", "#f00"))
        
        threading.Thread(target=task, daemon=True).start()

    def run_speed_test(self):
        self.speed_res.config(text="Testing Speed...", fg=self.colors["accent"])
        def task():
            import urllib.request
            import time
            
            # Download Test
            try:
                start = time.time()
                with urllib.request.urlopen("https://cachefly.cachefly.net/1mb.test", timeout=10) as response:
                    data = response.read()
                end = time.time()
                dl_speed = (len(data) * 8) / (end - start) / 1_000_000 # Mbps
                dl_str = f"{dl_speed:.2f} Mbps"
            except:
                dl_str = "Error"

            # Upload Test (Simulated using httpbin or similar as CacheFly doesn't support upload)
            try:
                dummy_data = b"0" * 512_000 # 0.5MB dummy data
                start = time.time()
                req = urllib.request.Request("https://httpbin.org/post", data=dummy_data, method="POST")
                with urllib.request.urlopen(req, timeout=10) as response:
                    response.read()
                end = time.time()
                ul_speed = (len(dummy_data) * 8) / (end - start) / 1_000_000 # Mbps
                ul_str = f"{ul_speed:.2f} Mbps"
            except:
                ul_str = "Error"

            def apply_ok():
                try:
                    self.speed_res.config(text=f"DL: {dl_str} | UL: {ul_str}", fg="#0f0")
                except:
                    pass
            self.root.after(0, apply_ok)
            
        threading.Thread(target=task, daemon=True).start()

    def open_settings(self):
        win = tk.Toplevel(self.root)
        win.title("Settings")
        # Phone-like settings window (no weird wide layout)
        win.geometry("390x720")
        try:
            win.minsize(360, 640)
        except:
            pass
        win.configure(bg=self.colors["bg"])
        self.root.update_idletasks()
        win.geometry(f"+{self.root.winfo_x() + 20}+{self.root.winfo_y() + 20}")
        win.transient(self.root); win.grab_set()
        win.protocol("WM_DELETE_WINDOW", win.destroy)

        # Header bar (title + close)
        header = tk.Frame(win, bg=self.colors["bg"])
        header.pack(fill=tk.X, padx=16, pady=(14, 8))
        tk.Label(header, text="SETTINGS", font=("Segoe UI", 14, "bold"), bg=self.colors["bg"], fg=self.colors["text"]).pack(side=tk.LEFT)
        tk.Button(
            header, text="Close", command=win.destroy,
            bg=self.colors["card2"], fg=self.colors["text"],
            font=("Segoe UI", 9, "bold"),
            relief=tk.FLAT, bd=0, padx=12, pady=6, cursor="hand2"
        ).pack(side=tk.RIGHT)

        # Scrollable body (in its own frame, so it doesn't fight pack geometry)
        body = tk.Frame(win, bg=self.colors["bg"])
        body.pack(fill=tk.BOTH, expand=True, padx=12, pady=(0, 12))

        canvas = tk.Canvas(body, bg=self.colors["bg"], highlightthickness=0, bd=0)
        scrollbar = tk.Scrollbar(body, orient="vertical", command=canvas.yview)
        canvas.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        canvas.configure(yscrollcommand=scrollbar.set)

        scroll_frame = tk.Frame(canvas, bg=self.colors["bg"])
        window_id = canvas.create_window((0, 0), window=scroll_frame, anchor="nw")

        def _on_frame_configure(_):
            canvas.configure(scrollregion=canvas.bbox("all"))

        def _on_canvas_configure(event):
            # Keep content width synced to canvas width (no huge empty area on the right)
            canvas.itemconfig(window_id, width=event.width)

        scroll_frame.bind("<Configure>", _on_frame_configure)
        canvas.bind("<Configure>", _on_canvas_configure)

        # Mouse Wheel Support (only when cursor is over settings canvas)
        def _on_mousewheel(event):
            try:
                canvas.yview_scroll(int(-1 * (event.delta / 120)), "units")
            except:
                pass

        canvas.bind("<Enter>", lambda e: canvas.bind_all("<MouseWheel>", _on_mousewheel))
        canvas.bind("<Leave>", lambda e: canvas.unbind_all("<MouseWheel>"))

        container = tk.Frame(scroll_frame, bg=self.colors["card"], padx=14, pady=14)
        container.pack(fill=tk.X, padx=10, pady=10)

        # Traffic Rules
        tk.Label(container, text="TRAFFIC RULES", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(0, 10))
        self.rule_var.set(self.app_settings["rule_preset"])
        
        rules_data = [
            ("Global", "Global", "Включает VPN для всего интернета. Все сайты и программы будут работать через защищенный канал. Максимальная приватность, но скорость на российских сайтах может быть ниже."),
            ("Bypass RU", "Bypass RU", "Российские сайты и сервисы (Яндекс, Госуслуги и др.) работают напрямую на полной скорости. VPN включается автоматически только для заблокированных и зарубежных ресурсов."),
            ("Bypass LAN+RU", "Bypass LAN+RU", "Российские сайты, а также ваша домашняя/офисная сеть (принтеры, умные устройства) работают напрямую. Остальное — через VPN. Рекомендуется для работы в офисе."),
            ("Socials Only", "Socials Only", "VPN работает ТОЛЬКО для Telegram, YouTube, Instagram и Facebook. Весь остальной интернет (браузер, игры, почта) работает без VPN. Идеально для экономии трафика."),
            ("Streaming", "Streaming", "VPN включается только при просмотре видео на Netflix, Disney+, YouTube и других кинотеатрах. Помогает обходить региональные ограничения при просмотре фильмов.")
        ]

        def set_r(v):
            self.app_settings["rule_preset"] = v
            self.show_notification("Rules Updated", f"Preset set to {v}")

        for text, mode, help_text in rules_data:
            row = tk.Frame(container, bg=self.colors["card"])
            row.pack(fill=tk.X, pady=2)
            tk.Radiobutton(row, text=text, variable=self.rule_var, value=mode, command=lambda m=mode: set_r(m),
                           bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"],
                           activebackground=self.colors["card"], activeforeground=self.colors["accent"], font=("Segoe UI", 9)).pack(side=tk.LEFT)
            info = tk.Label(row, text="ⓘ", font=("Segoe UI", 10), bg=self.colors["card"], fg=self.colors["text_dim"], cursor="hand2")
            info.pack(side=tk.RIGHT, padx=5)
            info.bind("<Button-1>", lambda e, t=text, m=help_text: self.show_custom_popup(t, m))

        # Connection Tests
        tk.Label(container, text="CONNECTION TESTS", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(20, 10))
        
        # Latency Test UI
        latency_btn = tk.Button(container, text="TEST LATENCY (Cloudflare)", command=self.run_latency_test, bg=self.colors["bg"], fg="white", 
                                relief=tk.FLAT, bd=0, height=1, cursor="hand2", font=("Segoe UI", 9, "bold"))
        latency_btn.pack(fill=tk.X, pady=2)
        self.latency_res = tk.Label(container, text="Avg: -- ms | Success: 0/10", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 8))
        self.latency_res.pack(anchor="w")

        # Speed Test UI
        speed_btn = tk.Button(container, text="TEST SPEED (CacheFly)", command=self.run_speed_test, bg=self.colors["bg"], fg="white", 
                              relief=tk.FLAT, bd=0, height=1, cursor="hand2", font=("Segoe UI", 9, "bold"))
        speed_btn.pack(fill=tk.X, pady=(10, 2))
        self.speed_res = tk.Label(container, text="DL: -- Mbps | UL: -- Mbps", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 8))
        self.speed_res.pack(anchor="w")

        # Maintenance
        tk.Label(container, text="MAINTENANCE", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(20, 10))
        def open_log_viewer():
            log_win = tk.Toplevel(win)
            log_win.title("App Logs")
            log_win.geometry("500x400")
            log_win.configure(bg="#000")
            txt = tk.Text(log_win, bg="#000", fg="#0f0", font=("Consolas", 9), padx=10, pady=10)
            txt.pack(fill=tk.BOTH, expand=True)
            if os.path.exists(self.log_file):
                with open(self.log_file, "r", encoding="utf-8", errors="ignore") as f:
                    txt.insert(tk.END, f.read())
            else: txt.insert(tk.END, "No logs yet...")
            txt.config(state=tk.DISABLED); txt.see(tk.END)

        tk.Button(container, text="VIEW CORE LOGS", command=open_log_viewer, bg=self.colors["bg"], fg="white", 
                  relief=tk.FLAT, bd=0, height=1, cursor="hand2", font=("Segoe UI", 9, "bold")).pack(fill=tk.X, pady=5)

        # Traffic Stats (interface fallback selection)
        tk.Label(container, text="TRAFFIC", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(20, 10))
        tk.Label(container, text="Interface for traffic counter (fallback mode)", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 8)).pack(anchor="w", pady=(0, 6))

        iface_names = ["auto"] + self.list_net_ifaces()
        self.traffic_iface_var = tk.StringVar(value=self.app_settings.get("traffic_iface", "auto"))

        iface_row = tk.Frame(container, bg=self.colors["card"])
        iface_row.pack(fill=tk.X, pady=(0, 6))
        tk.OptionMenu(iface_row, self.traffic_iface_var, *iface_names).pack(side=tk.LEFT, fill=tk.X, expand=True)

        def apply_iface():
            val = (self.traffic_iface_var.get() or "auto").strip()
            self.app_settings["traffic_iface"] = val
            # If VPN is running and we're in fallback mode, restart the interface monitor to apply immediately.
            try:
                if self.is_running:
                    self.traffic_stop_flag = True
                    self.root.after(300, self.start_interface_stats_monitor)
            except:
                pass

        tk.Button(
            iface_row,
            text="Apply",
            command=apply_iface,
            bg=self.colors["bg"],
            fg="white",
            relief=tk.FLAT,
            bd=0,
            padx=10,
            cursor="hand2",
            font=("Segoe UI", 9, "bold")
        ).pack(side=tk.RIGHT, padx=(8, 0))

        def refresh_ifaces():
            # rebuild the option menu with latest interface list
            names = ["auto"] + self.list_net_ifaces()
            menu = iface_row.winfo_children()[0]["menu"]
            menu.delete(0, "end")
            for n in names:
                menu.add_command(label=n, command=lambda v=n: self.traffic_iface_var.set(v))

        tk.Button(
            container,
            text="Refresh interface list",
            command=refresh_ifaces,
            bg=self.colors["card2"],
            fg=self.colors["text"],
            relief=tk.FLAT,
            bd=0,
            cursor="hand2",
            font=("Segoe UI", 9, "bold")
        ).pack(fill=tk.X, pady=(0, 4))

        # General
        tk.Label(container, text="GENERAL", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(20, 10))

        # NekoBox-compatible mode (template-based)
        self.nb_mode_var = tk.BooleanVar(value=self.app_settings.get("nekobox_compatible", False))
        def toggle_nb_mode():
            val = bool(self.nb_mode_var.get())
            self.app_settings["nekobox_compatible"] = val
            if val and not (self.app_settings.get("nekobox_template_json") or "").strip():
                self.show_custom_popup(
                    "NekoBox-compatible",
                    "Включено, но шаблон не задан.\n\n"
                    "Сделай Export sing-box JSON в оригинальном NekoBox и загрузить/вставь его сюда."
                )
            else:
                self.show_notification("Mode", "NekoBox-compatible ON" if val else "NekoBox-compatible OFF")

        tk.Checkbutton(
            container,
            text="NekoBox-compatible mode (use template JSON)",
            variable=self.nb_mode_var,
            command=toggle_nb_mode,
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w", pady=(0, 6))

        tpl_status = "Loaded" if (self.app_settings.get("nekobox_template_json") or "").strip() else "Not set"
        self.nb_tpl_label = tk.Label(container, text=f"Template: {tpl_status}", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 8))
        self.nb_tpl_label.pack(anchor="w", pady=(0, 6))

        btns = tk.Frame(container, bg=self.colors["card"])
        btns.pack(fill=tk.X, pady=(0, 10))

        def refresh_tpl_label():
            s = "Loaded" if (self.app_settings.get("nekobox_template_json") or "").strip() else "Not set"
            self.nb_tpl_label.config(text=f"Template: {s}")

        tk.Button(
            btns,
            text="Load Template",
            command=lambda: (self.load_nekobox_template_from_file(), refresh_tpl_label()),
            bg=self.colors["bg"],
            fg="white",
            relief=tk.FLAT,
            bd=0,
            height=1,
            cursor="hand2",
            font=("Segoe UI", 9, "bold")
        ).pack(side=tk.LEFT, expand=True, fill=tk.X, padx=(0, 5))

        tk.Button(
            btns,
            text="Paste Template",
            command=lambda: (self.paste_nekobox_template(), refresh_tpl_label()),
            bg=self.colors["bg"],
            fg="white",
            relief=tk.FLAT,
            bd=0,
            height=1,
            cursor="hand2",
            font=("Segoe UI", 9, "bold")
        ).pack(side=tk.LEFT, expand=True, fill=tk.X, padx=(5, 0))

        # Compatibility: Cursor direct bypass
        self.cursor_direct_var = tk.BooleanVar(value=self.app_settings.get("compat_cursor_direct", False))
        def toggle_cursor_direct():
            val = bool(self.cursor_direct_var.get())
            self.app_settings["compat_cursor_direct"] = val
            if val:
                self.show_notification("Compatibility", "Cursor будет открываться напрямую (без VPN)")
            else:
                self.show_notification("Compatibility", "Cursor будет открываться через VPN")

        tk.Checkbutton(
            container,
            text="Fix Cursor.com (open direct)",
            variable=self.cursor_direct_var,
            command=toggle_cursor_direct,
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w", pady=(0, 8))

        # Compatibility: Disable QUIC (UDP/443)
        self.quic_var = tk.BooleanVar(value=self.app_settings.get("disable_quic", True))
        def toggle_quic():
            val = bool(self.quic_var.get())
            self.app_settings["disable_quic"] = val
            if val:
                self.show_notification("Compatibility", "QUIC отключен (UDP/443). Браузер будет использовать TCP.")
            else:
                self.show_notification("Compatibility", "QUIC включен (UDP/443).")

        tk.Checkbutton(
            container,
            text="Disable QUIC (fix SSL errors)",
            variable=self.quic_var,
            command=toggle_quic,
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w", pady=(0, 8))

        # NekoBox-like TUN/DNS controls
        tk.Label(container, text="TUN / DNS (NekoBox-like)", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(10, 8))

        # Stack
        self.tun_stack_var = tk.StringVar(value=self.app_settings.get("tun_stack", "system"))
        def set_tun_stack(*_):
            self.app_settings["tun_stack"] = self.tun_stack_var.get()
        row = tk.Frame(container, bg=self.colors["card"])
        row.pack(fill=tk.X, pady=2)
        tk.Label(row, text="Stack", bg=self.colors["card"], fg=self.colors["text_dim"], width=10, anchor="w").pack(side=tk.LEFT)
        tk.OptionMenu(row, self.tun_stack_var, "system", "gvisor", command=lambda *_: set_tun_stack()).pack(side=tk.RIGHT)

        # MTU
        self.tun_mtu_var = tk.StringVar(value=str(self.app_settings.get("tun_mtu", 1500)))
        def save_mtu():
            try:
                v = int(self.tun_mtu_var.get().strip())
                if v < 576 or v > 9000:
                    raise ValueError("bad mtu")
                self.app_settings["tun_mtu"] = v
            except:
                self.show_custom_popup("MTU", "MTU должно быть числом (пример: 1500).")
                self.tun_mtu_var.set(str(self.app_settings.get("tun_mtu", 1500)))
        row = tk.Frame(container, bg=self.colors["card"])
        row.pack(fill=tk.X, pady=2)
        tk.Label(row, text="MTU", bg=self.colors["card"], fg=self.colors["text_dim"], width=10, anchor="w").pack(side=tk.LEFT)
        mtu_e = tk.Entry(row, textvariable=self.tun_mtu_var, bg=self.colors["bg"], fg=self.colors["text"], insertbackground="white", relief=tk.FLAT)
        mtu_e.pack(side=tk.RIGHT, fill=tk.X, expand=True)
        mtu_e.bind("<FocusOut>", lambda e: save_mtu())

        # Checkboxes
        self.tun_ipv6_var = tk.BooleanVar(value=self.app_settings.get("tun_ipv6", False))
        self.tun_strict_var = tk.BooleanVar(value=self.app_settings.get("tun_strict_route", True))
        self.tun_auto_var = tk.BooleanVar(value=self.app_settings.get("tun_auto_route", True))

        def sync_tun_flags():
            self.app_settings["tun_ipv6"] = bool(self.tun_ipv6_var.get())
            self.app_settings["tun_strict_route"] = bool(self.tun_strict_var.get())
            self.app_settings["tun_auto_route"] = bool(self.tun_auto_var.get())

        tk.Checkbutton(container, text="Enable IPv6 in TUN", variable=self.tun_ipv6_var, command=sync_tun_flags,
                       bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"], font=("Segoe UI", 9)).pack(anchor="w")
        tk.Checkbutton(container, text="Strict route", variable=self.tun_strict_var, command=sync_tun_flags,
                       bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"], font=("Segoe UI", 9)).pack(anchor="w")
        tk.Checkbutton(container, text="Enable TUN routing (auto_route)", variable=self.tun_auto_var, command=sync_tun_flags,
                       bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"], font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 6))

        # DNS fields
        self.dns_remote_var = tk.StringVar(value=self.app_settings.get("dns_remote", "tls://8.8.8.8"))
        self.dns_direct_var = tk.StringVar(value=self.app_settings.get("dns_direct", "8.8.8.8"))

        def sync_dns():
            self.app_settings["dns_remote"] = self.dns_remote_var.get().strip() or "tls://8.8.8.8"
            self.app_settings["dns_direct"] = self.dns_direct_var.get().strip() or "8.8.8.8"

        row = tk.Frame(container, bg=self.colors["card"])
        row.pack(fill=tk.X, pady=2)
        tk.Label(row, text="Remote DNS", bg=self.colors["card"], fg=self.colors["text_dim"], width=10, anchor="w").pack(side=tk.LEFT)
        e = tk.Entry(row, textvariable=self.dns_remote_var, bg=self.colors["bg"], fg=self.colors["text"], insertbackground="white", relief=tk.FLAT)
        e.pack(side=tk.RIGHT, fill=tk.X, expand=True)
        e.bind("<FocusOut>", lambda e: sync_dns())

        row = tk.Frame(container, bg=self.colors["card"])
        row.pack(fill=tk.X, pady=2)
        tk.Label(row, text="Direct DNS", bg=self.colors["card"], fg=self.colors["text_dim"], width=10, anchor="w").pack(side=tk.LEFT)
        e = tk.Entry(row, textvariable=self.dns_direct_var, bg=self.colors["bg"], fg=self.colors["text"], insertbackground="white", relief=tk.FLAT)
        e.pack(side=tk.RIGHT, fill=tk.X, expand=True)
        e.bind("<FocusOut>", lambda e: sync_dns())

        # Autostart (fixed: don't create new BooleanVar in callback)
        self.autostart_var = tk.BooleanVar(value=bool(self.app_settings.get("autostart", False)))
        tk.Checkbutton(
            container,
            text="Start with Windows",
            variable=self.autostart_var,
            command=lambda: self.toggle_autostart(self.autostart_var),
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w")

        # Run as Admin
        self.run_as_admin_var = tk.BooleanVar(value=bool(self.app_settings.get("run_as_admin", False)))
        def toggle_run_as_admin():
            val = bool(self.run_as_admin_var.get())
            self.app_settings["run_as_admin"] = val
            try:
                self.save_settings()
            except:
                pass
            if val and os.name == "nt" and not self.is_admin():
                ok = self.show_custom_popup(
                    "Run as Admin",
                    "To enable this, the app must restart with Administrator privileges (UAC prompt).\n\nRestart now?",
                    is_confirm=True,
                )
                if ok:
                    if self.relaunch_as_admin():
                        try:
                            self.on_close()
                        except:
                            try:
                                self.root.destroy()
                            except:
                                pass
                else:
                    # Revert toggle if user declined
                    self.run_as_admin_var.set(False)
                    self.app_settings["run_as_admin"] = False
                    try:
                        self.save_settings()
                    except:
                        pass

        tk.Checkbutton(
            container,
            text="Run as admin (always)",
            variable=self.run_as_admin_var,
            command=toggle_run_as_admin,
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w", pady=(6, 0))

        # Minimize / Tray
        self.tray_var = tk.BooleanVar(value=bool(self.app_settings.get("minimize_to_tray", False)))
        self.start_min_var = tk.BooleanVar(value=bool(self.app_settings.get("minimized", False)))

        def toggle_tray():
            val = bool(self.tray_var.get())
            self.app_settings["minimize_to_tray"] = val
            try:
                self.save_settings()
            except:
                pass
            if val:
                self._ensure_tray()
                self.show_notification("Tray", "Minimize-to-tray enabled. Close button will hide to tray.")
            else:
                self.show_notification("Tray", "Minimize-to-tray disabled.")

        def toggle_start_min():
            val = bool(self.start_min_var.get())
            self.app_settings["minimized"] = val
            try:
                self.save_settings()
            except:
                pass

        tk.Checkbutton(
            container,
            text="Minimize to tray (close hides)",
            variable=self.tray_var,
            command=toggle_tray,
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w", pady=(10, 0))

        tk.Checkbutton(
            container,
            text="Start minimized",
            variable=self.start_min_var,
            command=toggle_start_min,
            bg=self.colors["card"],
            fg=self.colors["text"],
            selectcolor=self.colors["bg"],
            font=("Segoe UI", 9)
        ).pack(anchor="w", pady=(6, 0))

        # New Features Section
        tk.Label(container, text="ADVANCED", font=("Segoe UI", 10, "bold"), bg=self.colors["card"], fg=self.colors["accent"]).pack(anchor="w", pady=(20, 10))

        # Kill Switch
        self.kill_switch_var = tk.BooleanVar(value=bool(self.app_settings.get("kill_switch", False)))
        def toggle_kill_switch():
            val = bool(self.kill_switch_var.get())
            self.app_settings["kill_switch"] = val
            try:
                self.save_settings()
            except:
                pass
        tk.Checkbutton(container, text="Kill Switch (block internet when VPN drops)", variable=self.kill_switch_var, command=toggle_kill_switch,
                       bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"], font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 6))

        # Auto Reconnect
        self.auto_reconnect_var = tk.BooleanVar(value=bool(self.app_settings.get("auto_reconnect", False)))
        def toggle_auto_reconnect():
            val = bool(self.auto_reconnect_var.get())
            self.app_settings["auto_reconnect"] = val
            try:
                self.save_settings()
            except:
                pass
        tk.Checkbutton(container, text="Auto Reconnect (reconnect on connection loss)", variable=self.auto_reconnect_var, command=toggle_auto_reconnect,
                       bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"], font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 6))

        # DNS Leak Protection
        self.dns_leak_var = tk.BooleanVar(value=bool(self.app_settings.get("dns_leak_protection", False)))
        def toggle_dns_leak():
            val = bool(self.dns_leak_var.get())
            self.app_settings["dns_leak_protection"] = val
            try:
                self.save_settings()
            except:
                pass
        tk.Checkbutton(container, text="DNS Leak Protection", variable=self.dns_leak_var, command=toggle_dns_leak,
                       bg=self.colors["card"], fg=self.colors["text"], selectcolor=self.colors["bg"], font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 10))

        # Language
        tk.Label(container, text="Language", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 6))
        self.language_var = tk.StringVar(value=self.app_settings.get("language", "en"))
        lang_row = tk.Frame(container, bg=self.colors["card"])
        lang_row.pack(fill=tk.X, pady=(0, 10))
        def set_language(*_):
            lang = self.language_var.get()
            self.app_settings["language"] = lang
            try:
                init_translator(lang)
                self.translator = get_translator()
                self.save_settings()
            except:
                pass
        tk.OptionMenu(lang_row, self.language_var, "en", "ru", command=set_language).pack(side=tk.LEFT)

        # Log Level
        tk.Label(container, text="Log Level", bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 6))
        self.log_level_var = tk.StringVar(value=str(self.app_settings.get("log_level", 1)))
        log_level_row = tk.Frame(container, bg=self.colors["card"])
        log_level_row.pack(fill=tk.X, pady=(0, 10))
        def set_log_level(*_):
            level = int(self.log_level_var.get())
            self.app_settings["log_level"] = level
            try:
                if hasattr(self, 'app_logger') and self.app_logger:
                    self.app_logger.set_level(LogLevel(level))
                self.save_settings()
            except:
                pass
        level_map = {"0": "Debug", "1": "Info", "2": "Warning", "3": "Error"}
        tk.OptionMenu(log_level_row, self.log_level_var, *["0", "1", "2", "3"], command=set_log_level).pack(side=tk.LEFT)

    def open_add_proxy(self, edit_idx=None):
        win = tk.Toplevel(self.root)
        win.title("Edit Proxy" if edit_idx is not None else "Add Proxy")
        win.geometry("600x750")
        win.configure(bg=self.colors["bg"])
        self.root.update_idletasks()
        win.geometry(f"+{self.root.winfo_x() + 50}+{self.root.winfo_y() + 20}")
        win.transient(self.root); win.grab_set()

        # Canvas for scrolling
        canvas = tk.Canvas(win, bg=self.colors["bg"], highlightthickness=0)
        scrollbar = tk.Scrollbar(win, orient="vertical", command=canvas.yview)
        scroll_frame = tk.Frame(canvas, bg=self.colors["bg"])
        scroll_frame.bind("<Configure>", lambda e: canvas.configure(scrollregion=canvas.bbox("all")))
        canvas.create_window((0, 0), window=scroll_frame, anchor="nw", width=580)
        canvas.configure(yscrollcommand=scrollbar.set)
        canvas.pack(side="left", fill="both", expand=True)
        scrollbar.pack(side="right", fill="y")

        # Helper to create styled entries
        def create_field(parent, label, default=""):
            f = tk.Frame(parent, bg=self.colors["card"], pady=5)
            f.pack(fill=tk.X, padx=10)
            tk.Label(f, text=label, bg=self.colors["card"], fg=self.colors["text_dim"], font=("Segoe UI", 9), width=12, anchor="w").pack(side=tk.LEFT, padx=5)
            e = tk.Entry(f, bg=self.colors["bg"], fg=self.colors["text"], insertbackground="white", 
                         relief=tk.FLAT, font=("Segoe UI", 10))
            e.pack(side=tk.RIGHT, fill=tk.X, expand=True, padx=5)
            e.insert(0, str(default))
            return e

        # --- Link Parser ---
        parse_frame = tk.Frame(scroll_frame, bg=self.colors["card"], pady=10)
        parse_frame.pack(fill=tk.X, padx=20, pady=10)
        link_entry = tk.Entry(parse_frame, bg=self.colors["bg"], fg=self.colors["text"], font=("Segoe UI", 9))
        link_entry.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=10)

        # Placeholder behavior (so user doesn't need to delete text)
        placeholder_text = "Вставь vless:// или socks5:// ссылку сюда..."
        placeholder_color = self.colors["text_dim"]
        normal_color = self.colors["text"]

        def _set_placeholder():
            if not link_entry.get().strip():
                link_entry.delete(0, tk.END)
                link_entry.insert(0, placeholder_text)
                link_entry.config(fg=placeholder_color)

        def _clear_placeholder():
            if link_entry.get().strip() == placeholder_text:
                link_entry.delete(0, tk.END)
                link_entry.config(fg=normal_color)

        link_entry.bind("<FocusIn>", lambda e: _clear_placeholder())
        link_entry.bind("<FocusOut>", lambda e: _set_placeholder())
        _set_placeholder()
        
        # --- BASIC ---
        main_f = tk.LabelFrame(scroll_frame, text=" BASIC SETTINGS ", bg=self.colors["bg"], fg=self.colors["accent"], font=("Segoe UI", 10, "bold"), padx=10, pady=10)
        main_f.pack(fill=tk.X, padx=20, pady=5)
        name_e = create_field(main_f, "Name:", "New Server")
        host_e = create_field(main_f, "Address:", "0.0.0.0")
        port_e = create_field(main_f, "Port:", "443")

        # --- VLESS ---
        vless_f = tk.LabelFrame(scroll_frame, text=" VLESS / PROTOCOL ", bg=self.colors["bg"], fg=self.colors["accent"], font=("Segoe UI", 10, "bold"), padx=10, pady=10)
        vless_f.pack(fill=tk.X, padx=20, pady=5)
        uuid_e = create_field(vless_f, "UUID:")
        flow_var = tk.StringVar(value="none")
        flow_f = tk.Frame(vless_f, bg=self.colors["card"], pady=5)
        flow_f.pack(fill=tk.X, padx=10)
        tk.Label(flow_f, text="Flow:", bg=self.colors["card"], fg=self.colors["text_dim"], width=12, anchor="w").pack(side=tk.LEFT, padx=5)
        tk.OptionMenu(flow_f, flow_var, "none", "xtls-rprx-vision").pack(side=tk.RIGHT)

        # Packet encoding (xudp improves UDP over VLESS in many configs)
        packet_var = tk.StringVar(value="xudp")
        packet_f = tk.Frame(vless_f, bg=self.colors["card"], pady=5)
        packet_f.pack(fill=tk.X, padx=10)
        tk.Label(packet_f, text="UDP Enc:", bg=self.colors["card"], fg=self.colors["text_dim"], width=12, anchor="w").pack(side=tk.LEFT, padx=5)
        tk.OptionMenu(packet_f, packet_var, "none", "xudp").pack(side=tk.RIGHT)

        # --- SECURITY ---
        sec_f = tk.LabelFrame(scroll_frame, text=" SECURITY (TLS/REALITY) ", bg=self.colors["bg"], fg=self.colors["accent"], font=("Segoe UI", 10, "bold"), padx=10, pady=10)
        sec_f.pack(fill=tk.X, padx=20, pady=5)
        sec_type_var = tk.StringVar(value="none")
        st_f = tk.Frame(sec_f, bg=self.colors["card"], pady=5)
        st_f.pack(fill=tk.X, padx=10)
        tk.Label(st_f, text="Security:", bg=self.colors["card"], fg=self.colors["text_dim"], width=12, anchor="w").pack(side=tk.LEFT, padx=5)
        tk.OptionMenu(st_f, sec_type_var, "none", "tls", "reality").pack(side=tk.RIGHT)
        sni_e = create_field(sec_f, "SNI:")
        fp_var = tk.StringVar(value="chrome")
        fp_f = tk.Frame(sec_f, bg=self.colors["card"], pady=5)
        fp_f.pack(fill=tk.X, padx=10)
        tk.Label(fp_f, text="Fingerprint:", bg=self.colors["card"], fg=self.colors["text_dim"], width=12, anchor="w").pack(side=tk.LEFT, padx=5)
        tk.OptionMenu(fp_f, fp_var, "chrome", "safari", "firefox", "edge", "random").pack(side=tk.RIGHT)
        pbk_e = create_field(sec_f, "Reality Pbk:")
        sid_e = create_field(sec_f, "Reality SID:")
        spx_e = create_field(sec_f, "Reality SPX:")

        # --- TRANSPORT ---
        trans_f = tk.LabelFrame(scroll_frame, text=" TRANSPORT ", bg=self.colors["bg"], fg=self.colors["accent"], font=("Segoe UI", 10, "bold"), padx=10, pady=10)
        trans_f.pack(fill=tk.X, padx=20, pady=5)
        net_var = tk.StringVar(value="tcp")
        net_f = tk.Frame(trans_f, bg=self.colors["card"], pady=5)
        net_f.pack(fill=tk.X, padx=10)
        tk.Label(net_f, text="Network:", bg=self.colors["card"], fg=self.colors["text_dim"], width=12, anchor="w").pack(side=tk.LEFT, padx=5)
        tk.OptionMenu(net_f, net_var, "tcp", "ws", "grpc").pack(side=tk.RIGHT)
        host_hdr_e = create_field(trans_f, "Header Host:")
        path_hdr_e = create_field(trans_f, "Header Path:")

        def do_parse():
            link = link_entry.get().strip()
            if link == placeholder_text:
                link = ""
            try:
                from urllib.parse import urlparse, parse_qs, unquote
                p = urlparse(link)
                if p.scheme != 'vless': raise Exception("Only vless supported")
                name_e.delete(0, tk.END); name_e.insert(0, unquote(p.fragment) if p.fragment else "Imported Server")
                host_e.delete(0, tk.END); host_e.insert(0, p.hostname)
                port_e.delete(0, tk.END); port_e.insert(0, p.port if p.port else 443)
                uuid_e.delete(0, tk.END); uuid_e.insert(0, p.username)
                q = parse_qs(p.query)
                if 'flow' in q: flow_var.set(q['flow'][0])
                if 'security' in q: sec_type_var.set(q['security'][0])
                if 'sni' in q: sni_e.delete(0, tk.END); sni_e.insert(0, q['sni'][0])
                if 'pbk' in q: pbk_e.delete(0, tk.END); pbk_e.insert(0, q['pbk'][0])
                if 'sid' in q: sid_e.delete(0, tk.END); sid_e.insert(0, q['sid'][0])
                if 'spx' in q: spx_e.delete(0, tk.END); spx_e.insert(0, unquote(q['spx'][0]))
                if 'fp' in q: fp_var.set(q['fp'][0])
                if 'type' in q: net_var.set(q['type'][0])
                if 'host' in q: host_hdr_e.delete(0, tk.END); host_hdr_e.insert(0, q['host'][0])
                if 'path' in q: path_hdr_e.delete(0, tk.END); path_hdr_e.insert(0, q['path'][0])
                self.show_notification("Parsed", "Link data applied to fields")
            except Exception as e: self.show_custom_popup("Error", f"Invalid link: {e}")

        # Buttons (Paste + Parse)
        btns_frame = tk.Frame(parse_frame, bg=self.colors["card"])
        btns_frame.pack(side=tk.RIGHT, padx=10)

        paste_btn = tk.Button(
            btns_frame,
            text="Вставить",
            bg=self.colors["bg"],
            fg="white",
            font=("Segoe UI", 8, "bold"),
            relief=tk.FLAT,
            bd=0,
            cursor="hand2",
            state=tk.DISABLED
        )
        paste_btn.pack(side=tk.LEFT, padx=(0, 6))

        parse_btn = tk.Button(
            btns_frame,
            text="PARSE LINK",
            command=do_parse,
            bg=self.colors["accent"],
            fg="white",
            font=("Segoe UI", 8, "bold"),
            relief=tk.FLAT,
            bd=0,
            cursor="hand2"
        )
        parse_btn.pack(side=tk.LEFT)

        def _clipboard_has_link(txt: str) -> bool:
            t = (txt or "").strip().lower()
            return t.startswith("vless://") or t.startswith("socks5://")

        # (clipboard polling removed intentionally)
        def do_paste():
            try:
                clip = self.root.clipboard_get()
            except Exception:
                clip = ""
            if not _clipboard_has_link(clip):
                self.show_custom_popup("Clipboard", "В буфере обмена нет vless:// или socks5:// ссылки.")
                return
            _clear_placeholder()
            link_entry.delete(0, tk.END)
            link_entry.insert(0, clip.strip())
            link_entry.config(fg=normal_color)

        paste_btn.config(command=do_paste)
        # Do not poll clipboard (privacy). Validate only on click.
        paste_btn.config(state=tk.NORMAL)

        if edit_idx is not None:
            p = self.proxies_data[edit_idx]
            name_e.delete(0, tk.END); name_e.insert(0, p['name'])
            host_e.delete(0, tk.END); host_e.insert(0, p['host'])
            c = p.get('config', {})
            port_e.delete(0, tk.END); port_e.insert(0, c.get('server_port', 443))
            uuid_e.delete(0, tk.END); uuid_e.insert(0, c.get('uuid', ''))
            flow_var.set(c.get('flow', 'none'))
            packet_var.set(c.get('packet_encoding', 'xudp') or 'xudp')
            tls = c.get('tls', {})
            if tls:
                if tls.get('reality'): 
                    sec_type_var.set('reality')
                    pbk_e.delete(0, tk.END); pbk_e.insert(0, tls['reality'].get('public_key', ''))
                    sid_e.delete(0, tk.END); sid_e.insert(0, tls['reality'].get('short_id', ''))
                    # Display-only: this core build doesn't support SpiderX, but show if user pasted it.
                    spx_e.delete(0, tk.END); spx_e.insert(0, tls['reality'].get('spiderX', tls['reality'].get('spider_x', '')))
                else: sec_type_var.set('tls')
                sni_e.delete(0, tk.END); sni_e.insert(0, tls.get('server_name', ''))
                fp_var.set(tls.get('utls', {}).get('fingerprint', 'chrome'))
            transport = c.get('transport', {})
            if transport:
                net_var.set(transport.get('type', 'tcp'))
                host_hdr_e.delete(0, tk.END); host_hdr_e.insert(0, transport.get('host', ''))
                path_hdr_e.delete(0, tk.END); path_hdr_e.insert(0, transport.get('path', ''))

        def save():
            config = {
                "type": "vless", "tag": "proxy",
                "server": host_e.get(), "server_port": int(port_e.get()),
                "uuid": uuid_e.get()
            }
            if flow_var.get() and flow_var.get() != "none":
                config["flow"] = flow_var.get()
            if packet_var.get() and packet_var.get() != "none":
                config["packet_encoding"] = packet_var.get()
            st = sec_type_var.get()
            if st != "none":
                config["tls"] = {"enabled": True, "server_name": sni_e.get(), "utls": {"enabled": True, "fingerprint": fp_var.get()}}
                if st == "reality":
                    config["tls"]["reality"] = {"enabled": True, "public_key": pbk_e.get(), "short_id": sid_e.get()}
                    # NOTE: Do NOT write SpiderX field; current core build rejects it ("unknown field").
            nt = net_var.get()
            if nt != "tcp" or host_hdr_e.get() or path_hdr_e.get():
                config["transport"] = {"type": nt}
                if host_hdr_e.get(): config["transport"]["host"] = host_hdr_e.get()
                if path_hdr_e.get(): config["transport"]["path"] = path_hdr_e.get()
            
            new_proxy = {"name": name_e.get(), "host": host_e.get(), "config": config}
            if edit_idx is not None: self.proxies_data[edit_idx] = new_proxy
            else: self.proxies_data.append(new_proxy)
            self.save_proxies(); self.initial_list_fill(); win.destroy()

        tk.Button(scroll_frame, text="SAVE CONFIG", command=save, bg=self.colors["status_on"], fg="white", 
                  font=("Segoe UI", 10, "bold"), relief=tk.FLAT, height=2).pack(fill=tk.X, padx=40, pady=20)

    def open_edit_proxy(self):
        selection = self.proxy_list.curselection()
        if selection: self.open_add_proxy(edit_idx=selection[0])
        else: self.show_custom_popup("Warning", "Please select a server to edit.")
