export async function GetStatus() {
  return { state: 'disconnected', appVersion: 'dev', selectedNode: '' }
}
export async function GetSettings() {
  return { mode: 'rule', tun: true, autostart: false, closeToTray: true, mixedPort: 7890 }
}
export async function SaveSettings() { return null }
export async function SetMode() { return null }
export async function ListProfiles() { return [] }
export async function ListRunningApps() { return [] }
export async function ListNodes() { return [] }
export async function CurrentNode() { return '' }
export async function SelectNode() { return null }
export async function TestNodeDelay() { return 0 }
export async function TestAllNodes() { return [] }
export async function UpsertProfile() { return null }
export async function ImportProfileText() { return null }
export async function ImportSubscription() { return null }
export async function SyncSubscription() { return null }
export async function SyncAllSubscriptions() { return 0 }
export async function GetSubscriptionInfo() { return {} }
export async function RemoveProfile() { return null }
export async function SetActiveProfile() { return null }
export async function Connect() { return null }
export async function Disconnect() { return null }
export async function ToggleConnect() { return null }
export async function GetTraffic() { return { up: 0, down: 0 } }
export async function IsAdmin() { return false }
export async function RelaunchAsAdmin() { return null }
export async function CheckForUpdate() { return { available: false } }
export async function DownloadUpdate() { return '' }
export async function ApplyUpdate() { return null }
export async function ShowWindow() { return null }
export async function HideWindow() { return null }
export async function QuitApp() { return null }
export async function GetLogsTail() { return '' }
export async function GetProxyEndpoints() { return { mixedPort: 7890, allowLan: false, loopback: '127.0.0.1:7890', lan: [] } }
export async function CopyProxyEndpoint() { return null }
