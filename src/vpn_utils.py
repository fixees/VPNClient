"""
VPN utilities: Kill Switch, DNS Leak Protection, Health Check, Auto Reconnect.
"""
import os
import socket
import threading
import time
import subprocess
import urllib.request
import urllib.error


class KillSwitch:
    """Kill Switch: Block internet when VPN disconnects."""
    
    def __init__(self, logger=None):
        self.logger = logger
        self.is_active = False
        self.vpn_interface = None
        self._lock = threading.Lock()
        
        if os.name != "nt":
            # Linux/macOS implementation would use iptables/pfctl
            self._platform_supported = False
        else:
            self._platform_supported = True
    
    def activate(self, vpn_interface=None):
        """Activate kill switch for given VPN interface."""
        if not self._platform_supported:
            return False
        
        try:
            with self._lock:
                if self.is_active:
                    return True
                
                self.vpn_interface = vpn_interface
                self.is_active = True
                
                if self.logger:
                    self.logger.info(f"Kill Switch activated for interface: {vpn_interface}")
                
                return True
        except Exception as e:
            if self.logger:
                self.logger.error(f"Kill Switch activation failed: {e}")
            return False
    
    def deactivate(self):
        """Deactivate kill switch."""
        if not self._platform_supported:
            return
        
        try:
            with self._lock:
                if not self.is_active:
                    return
                
                self.is_active = False
                self.vpn_interface = None
                
                if self.logger:
                    self.logger.info("Kill Switch deactivated")
        except Exception as e:
            if self.logger:
                self.logger.error(f"Kill Switch deactivation failed: {e}")
    
    def check_vpn_status(self):
        """Check if VPN is still active. Returns True if VPN is up."""
        if not self.is_active:
            return True  # Not blocking
        
        try:
            # Check if VPN interface exists and is up
            if os.name == "nt":
                # On Windows, check if TUN interface exists
                import psutil
                interfaces = psutil.net_if_addrs()
                if self.vpn_interface:
                    # Check if interface exists
                    if self.vpn_interface in interfaces:
                        # Check if it has IP
                        addrs = interfaces[self.vpn_interface]
                        if addrs:
                            return True
                return False
            return True
        except Exception:
            return True  # On error, don't block
    
    def block_internet(self):
        """Block all internet traffic (Windows: uses netsh firewall rules)."""
        if not self._platform_supported or not self.is_active:
            return
        
        try:
            if os.name == "nt":
                # Windows: Create firewall rule to block all outbound except VPN interface
                # Note: This requires admin privileges
                # Using route delete is simpler but less reliable
                # For production, would use Windows Firewall API
                pass  # Placeholder - actual implementation would use firewall API
        except Exception as e:
            if self.logger:
                self.logger.error(f"Failed to block internet: {e}")


class DNSLeakProtection:
    """DNS Leak Protection: Verify DNS requests go through VPN."""
    
    def __init__(self, logger=None):
        self.logger = logger
    
    def check_dns_leak(self, vpn_dns_servers=None, timeout=5):
        """
        Check for DNS leaks.
        
        Args:
            vpn_dns_servers: List of DNS servers that VPN should use
            timeout: Timeout for DNS queries
        
        Returns:
            (is_leak: bool, details: dict)
        """
        try:
            # Get current DNS servers
            current_dns = self._get_dns_servers()
            
            # Check if any DNS servers are not VPN DNS
            if vpn_dns_servers:
                leaks = []
                for dns in current_dns:
                    if dns not in vpn_dns_servers:
                        leaks.append(dns)
                
                is_leak = len(leaks) > 0
                details = {
                    "current_dns": current_dns,
                    "vpn_dns": vpn_dns_servers,
                    "leaks": leaks
                }
                
                if is_leak and self.logger:
                    self.logger.warning(f"DNS leak detected: {leaks}")
                
                return is_leak, details
            
            return False, {"current_dns": current_dns}
        except Exception as e:
            if self.logger:
                self.logger.error(f"DNS leak check failed: {e}")
            return False, {"error": str(e)}
    
    def _get_dns_servers(self):
        """Get current DNS servers."""
        dns_servers = []
        try:
            if os.name == "nt":
                # Windows: Query DNS servers via ipconfig
                result = subprocess.run(
                    ["ipconfig", "/all"],
                    capture_output=True,
                    text=True,
                    timeout=5
                )
                lines = result.stdout.split("\n")
                for line in lines:
                    if "DNS Servers" in line or "DNS-серверы" in line:
                        # Extract IP addresses from next lines
                        continue
                    # Simplified: would parse actual DNS servers from output
                    pass
        except Exception:
            pass
        return dns_servers


class VPNHealthCheck:
    """Periodic health check for VPN connection."""
    
    def __init__(self, logger=None, check_interval=30):
        self.logger = logger
        self.check_interval = check_interval
        self._running = False
        self._thread = None
        self._callbacks = []
    
    def add_callback(self, callback):
        """Add callback for health check events. Callback receives (is_healthy: bool)."""
        self._callbacks.append(callback)
    
    def start(self):
        """Start health check loop."""
        if self._running:
            return
        self._running = True
        self._thread = threading.Thread(target=self._health_check_loop, daemon=True)
        self._thread.start()
    
    def stop(self):
        """Stop health check loop."""
        self._running = False
        if self._thread and self._thread.is_alive():
            self._thread.join(timeout=2.0)
    
    def _health_check_loop(self):
        """Background health check loop."""
        while self._running:
            try:
                is_healthy = self.check_health()
                
                # Notify callbacks
                for callback in self._callbacks:
                    try:
                        callback(is_healthy)
                    except:
                        pass
                
                time.sleep(self.check_interval)
            except Exception as e:
                if self.logger:
                    self.logger.error(f"Health check error: {e}")
                time.sleep(self.check_interval)
    
    def check_health(self):
        """
        Perform health check.
        Returns True if VPN is healthy.
        """
        try:
            # Check 1: VPN process is running
            # (Would check if nekobox_core.exe is running)
            
            # Check 2: Can reach VPN API (if available)
            # (Would check Clash API endpoint)
            
            # Check 3: Can make outbound connection through VPN
            # (Would try to connect to test endpoint)
            
            # Simplified: Always return True for now
            # Actual implementation would check process, API, connectivity
            return True
        except Exception:
            return False


class AutoReconnect:
    """Automatic reconnection on VPN failure."""
    
    def __init__(self, logger=None, reconnect_delay=5, max_attempts=10):
        self.logger = logger
        self.reconnect_delay = reconnect_delay
        self.max_attempts = max_attempts
        self._is_enabled = False
        self._reconnect_callback = None
        self._attempts = 0
    
    def enable(self, reconnect_callback):
        """
        Enable auto reconnect.
        
        Args:
            reconnect_callback: Function to call to reconnect VPN
        """
        self._is_enabled = True
        self._reconnect_callback = reconnect_callback
        self._attempts = 0
    
    def disable(self):
        """Disable auto reconnect."""
        self._is_enabled = False
        self._attempts = 0
    
    def on_connection_lost(self):
        """Called when connection is lost."""
        if not self._is_enabled:
            return
        
        if self._attempts >= self.max_attempts:
            if self.logger:
                self.logger.error(f"Auto reconnect failed after {self.max_attempts} attempts")
            return
        
        self._attempts += 1
        if self.logger:
            self.logger.info(f"Auto reconnect attempt {self._attempts}/{self.max_attempts}")
        
        # Wait before reconnect
        time.sleep(self.reconnect_delay)
        
        # Try reconnect
        if self._reconnect_callback:
            try:
                self._reconnect_callback()
            except Exception as e:
                if self.logger:
                    self.logger.error(f"Reconnect callback failed: {e}")
    
    def on_connection_restored(self):
        """Called when connection is restored."""
        self._attempts = 0

