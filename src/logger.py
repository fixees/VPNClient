"""
Logging system with levels (DEBUG, INFO, WARNING, ERROR) and automatic cleanup.
"""
import os
import logging
import threading
from datetime import datetime, timedelta
from enum import IntEnum
import queue
import time


class LogLevel(IntEnum):
    """Log levels."""
    DEBUG = 0
    INFO = 1
    WARNING = 2
    ERROR = 3


class AppLogger:
    """Application logger with file and console output."""
    
    def __init__(self, log_file, level=LogLevel.INFO, max_log_age_days=7, max_log_size_mb=10):
        """
        Initialize logger.
        
        Args:
            log_file: Path to log file
            level: Minimum log level (LogLevel enum)
            max_log_age_days: Delete logs older than this many days
            max_log_size_mb: Maximum log file size in MB before rotation
        """
        self.log_file = log_file
        self.level = level
        self.max_log_age_days = max_log_age_days
        self.max_log_size_mb = max_log_size_mb
        self._lock = threading.Lock()
        self._log_queue = queue.Queue()
        self._worker_running = False
        self._worker_thread = None
        
        # Ensure log directory exists
        try:
            os.makedirs(os.path.dirname(log_file), exist_ok=True)
        except:
            pass
        
        # Cleanup old logs on init
        self._cleanup_old_logs()
        
        # Start log worker thread
        self._start_worker()
    
    def _start_worker(self):
        """Start background log worker thread."""
        if self._worker_running:
            return
        self._worker_running = True
        self._worker_thread = threading.Thread(target=self._log_worker, daemon=True)
        self._worker_thread.start()
    
    def _log_worker(self):
        """Background worker that writes logs."""
        while self._worker_running:
            try:
                # Get log entry with timeout
                entry = self._log_queue.get(timeout=1.0)
                if entry is None:
                    break
                
                level_str, timestamp, message = entry
                log_line = f"{timestamp} [{level_str}] {message}\n"
                
                # Write to file
                try:
                    with open(self.log_file, "a", encoding="utf-8", errors="replace") as f:
                        f.write(log_line)
                        f.flush()
                except Exception as e:
                    # Fallback to print if file write fails
                    print(f"Log write error: {e}")
                
                self._log_queue.task_done()
            except queue.Empty:
                continue
            except Exception as e:
                print(f"Log worker error: {e}")
                time.sleep(0.5)
    
    def _should_log(self, level):
        """Check if message should be logged based on level."""
        return level >= self.level
    
    def _log(self, level_str, level, message):
        """Internal log method."""
        if not self._should_log(level):
            return
        
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        
        # Add to queue for async writing
        try:
            self._log_queue.put((level_str, timestamp, message), block=False)
        except queue.Full:
            # If queue is full, fallback to immediate write
            try:
                log_line = f"{timestamp} [{level_str}] {message}\n"
                with open(self.log_file, "a", encoding="utf-8", errors="replace") as f:
                    f.write(log_line)
            except:
                pass
    
    def debug(self, message):
        """Log debug message."""
        self._log("DEBUG", LogLevel.DEBUG, message)
    
    def info(self, message):
        """Log info message."""
        self._log("INFO", LogLevel.INFO, message)
    
    def warning(self, message):
        """Log warning message."""
        self._log("WARNING", LogLevel.WARNING, message)
    
    def error(self, message):
        """Log error message."""
        self._log("ERROR", LogLevel.ERROR, message)
    
    def set_level(self, level):
        """Set minimum log level."""
        self.level = level
    
    def _cleanup_old_logs(self):
        """Remove log files older than max_log_age_days."""
        try:
            log_dir = os.path.dirname(self.log_file)
            if not os.path.exists(log_dir):
                return
            
            cutoff_date = datetime.now() - timedelta(days=self.max_log_age_days)
            cutoff_timestamp = cutoff_date.timestamp()
            
            for filename in os.listdir(log_dir):
                if not filename.endswith(".log"):
                    continue
                filepath = os.path.join(log_dir, filename)
                try:
                    if os.path.getmtime(filepath) < cutoff_timestamp:
                        os.remove(filepath)
                except:
                    pass
        except Exception:
            pass
    
    def cleanup(self):
        """Cleanup old logs and rotate if needed."""
        self._cleanup_old_logs()
        
        # Rotate if file is too large
        try:
            if os.path.exists(self.log_file):
                size_mb = os.path.getsize(self.log_file) / (1024 * 1024)
                if size_mb > self.max_log_size_mb:
                    # Rotate: rename current log, start new one
                    rotated_name = f"{self.log_file}.{datetime.now().strftime('%Y%m%d_%H%M%S')}"
                    os.rename(self.log_file, rotated_name)
        except Exception:
            pass
    
    def shutdown(self):
        """Shutdown logger (flush queue and stop worker)."""
        self._worker_running = False
        if self._worker_thread and self._worker_thread.is_alive():
            # Signal worker to stop
            try:
                self._log_queue.put(None, block=False)
            except:
                pass
            self._worker_thread.join(timeout=2.0)
        
        # Flush remaining log entries
        while not self._log_queue.empty():
            try:
                entry = self._log_queue.get_nowait()
                if entry is None:
                    break
                level_str, timestamp, message = entry
                log_line = f"{timestamp} [{level_str}] {message}\n"
                try:
                    with open(self.log_file, "a", encoding="utf-8", errors="replace") as f:
                        f.write(log_line)
                except:
                    pass
                self._log_queue.task_done()
            except queue.Empty:
                break


# Global logger instance
_app_logger = None


def init_logger(log_file, level=LogLevel.INFO, max_log_age_days=7, max_log_size_mb=10):
    """Initialize global logger."""
    global _app_logger
    _app_logger = AppLogger(log_file, level, max_log_age_days, max_log_size_mb)
    return _app_logger


def get_logger():
    """Get global logger instance."""
    global _app_logger
    if _app_logger is None:
        # Create default logger
        log_file = os.path.join(os.path.expanduser("~"), ".iSecureVPN", "app.log")
        _app_logger = AppLogger(log_file)
    return _app_logger

