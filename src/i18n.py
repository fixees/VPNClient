"""
Internationalization (i18n) support for iSecure VPN Client.
Supports Russian and English.
"""
import os
import json
import locale


class Translator:
    """Simple translation system."""
    
    # Default translations
    TRANSLATIONS = {
        "en": {
            "app_name": "iSecure VPN",
            "start_vpn": "START VPN",
            "stop_vpn": "STOP VPN",
            "starting": "Starting…",
            "stopping": "Stopping…",
            "connected": "Connected",
            "disconnected": "Disconnected",
            "settings": "Settings",
            "your_ip": "Your IP",
            "location": "Location",
            "traffic": "Traffic",
            "downloaded": "Downloaded",
            "uploaded": "Uploaded",
            "total": "Total",
            "speed": "Speed",
            "servers": "Servers",
            "add_server": "ADD SERVER",
            "remove_server": "REMOVE SERVER",
            "edit_server": "Edit Server",
            "save_config": "SAVE CONFIG",
            "admin_required": "Admin Required",
            "admin_message": "TUN mode requires Administrator privileges. Please restart the app as Admin.",
            "warning": "Warning",
            "error": "Error",
            "info": "Information",
            "blurred": "BLURRED",
            "checking": "Checking…",
            "vpn": "VPN",
            "direct": "Direct",
            "run_as_admin": "Run as admin",
            "autostart": "Autostart",
            "minimize_to_tray": "Minimize to tray",
            "start_minimized": "Start minimized",
            "kill_switch": "Kill Switch",
            "kill_switch_desc": "Block all internet if VPN disconnects",
            "auto_reconnect": "Auto Reconnect",
            "auto_reconnect_desc": "Automatically reconnect on connection loss",
            "dns_leak_protection": "DNS Leak Protection",
            "dns_leak_protection_desc": "Prevent DNS leaks",
            "log_level": "Log Level",
            "log_level_debug": "Debug",
            "log_level_info": "Info",
            "log_level_warning": "Warning",
            "log_level_error": "Error",
            "language": "Language",
            "language_english": "English",
            "language_russian": "Russian",
            "tray_show": "Show",
            "tray_hide": "Hide",
            "tray_exit": "Exit",
            "tray_quick_connect": "Quick Connect",
            "tray_stats": "Statistics",
            "tray_disconnect": "Disconnect",
            "tray_reconnect": "Reconnect",
            "already_running": "iSecure VPN is already running in system tray.",
            "already_running_hint": "Right-click the tray icon to open the application window.",
            "kill_switch_active": "Kill Switch Active",
            "kill_switch_blocked": "Internet blocked due to VPN disconnection",
            "reconnecting": "Reconnecting…",
            "connection_lost": "Connection Lost",
            "dns_leak_detected": "DNS Leak Detected",
            "health_check_failed": "Health Check Failed",
            "auto_reconnecting": "Auto Reconnecting…",
        },
        "ru": {
            "app_name": "iSecure VPN",
            "start_vpn": "ЗАПУСТИТЬ VPN",
            "stop_vpn": "ОСТАНОВИТЬ VPN",
            "starting": "Запуск…",
            "stopping": "Остановка…",
            "connected": "Подключено",
            "disconnected": "Отключено",
            "settings": "Настройки",
            "your_ip": "Ваш IP",
            "location": "Местоположение",
            "traffic": "Трафик",
            "downloaded": "Скачано",
            "uploaded": "Загружено",
            "total": "Всего",
            "speed": "Скорость",
            "servers": "Серверы",
            "add_server": "ДОБАВИТЬ СЕРВЕР",
            "remove_server": "УДАЛИТЬ СЕРВЕР",
            "edit_server": "Редактировать сервер",
            "save_config": "СОХРАНИТЬ КОНФИГ",
            "admin_required": "Требуются права администратора",
            "admin_message": "Режим TUN требует прав администратора. Перезапустите приложение от имени администратора.",
            "warning": "Предупреждение",
            "error": "Ошибка",
            "info": "Информация",
            "blurred": "СКРЫТО",
            "checking": "Проверка…",
            "vpn": "VPN",
            "direct": "Прямое",
            "run_as_admin": "Запуск от имени администратора",
            "autostart": "Автозапуск",
            "minimize_to_tray": "Сворачивать в трей",
            "start_minimized": "Запуск свернутым",
            "kill_switch": "Kill Switch",
            "kill_switch_desc": "Блокировать интернет при отключении VPN",
            "auto_reconnect": "Автопереподключение",
            "auto_reconnect_desc": "Автоматически переподключаться при обрыве",
            "dns_leak_protection": "Защита от утечек DNS",
            "dns_leak_protection_desc": "Предотвращать утечки DNS",
            "log_level": "Уровень логирования",
            "log_level_debug": "Отладка",
            "log_level_info": "Информация",
            "log_level_warning": "Предупреждение",
            "log_level_error": "Ошибка",
            "language": "Язык",
            "language_english": "Английский",
            "language_russian": "Русский",
            "tray_show": "Показать",
            "tray_hide": "Скрыть",
            "tray_exit": "Выход",
            "tray_quick_connect": "Быстрое подключение",
            "tray_stats": "Статистика",
            "tray_disconnect": "Отключить",
            "tray_reconnect": "Переподключить",
            "already_running": "iSecure VPN уже запущен в системном трее.",
            "already_running_hint": "Нажмите правой кнопкой мыши на иконку в трее, чтобы открыть окно приложения.",
            "kill_switch_active": "Kill Switch активен",
            "kill_switch_blocked": "Интернет заблокирован из-за отключения VPN",
            "reconnecting": "Переподключение…",
            "connection_lost": "Соединение потеряно",
            "dns_leak_detected": "Обнаружена утечка DNS",
            "health_check_failed": "Проверка работоспособности не удалась",
            "auto_reconnecting": "Автопереподключение…",
        }
    }
    
    def __init__(self, lang=None):
        """Initialize translator with language."""
        if lang is None:
            lang = self._detect_language()
        self.lang = lang if lang in self.TRANSLATIONS else "en"
        self._translations = self.TRANSLATIONS.get(self.lang, self.TRANSLATIONS["en"])
    
    def _detect_language(self):
        """Detect system language."""
        try:
            sys_lang = locale.getdefaultlocale()[0]
            if sys_lang:
                if sys_lang.startswith("ru"):
                    return "ru"
        except:
            pass
        return "en"
    
    def get(self, key, default=None):
        """Get translation for key."""
        return self._translations.get(key, default or key)
    
    def __call__(self, key, default=None):
        """Allow translator instance to be called directly."""
        return self.get(key, default)


# Global translator instance (will be initialized in main)
_translator = None


def init_translator(lang=None):
    """Initialize global translator."""
    global _translator
    _translator = Translator(lang)
    return _translator


def get_translator():
    """Get global translator instance."""
    global _translator
    if _translator is None:
        _translator = Translator()
    return _translator


def _(key, default=None):
    """Translation shortcut function."""
    return get_translator().get(key, default)

