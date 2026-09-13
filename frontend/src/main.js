import './style.css'

const app = document.querySelector('#app')

const mock = {
  async GetStatus() {
    return {
      state: 'disconnected',
      activeProfile: '',
      mode: 'rule',
      isAdmin: false,
      speedUp: 0,
      speedDown: 0,
      appVersion: 'dev',
      subscriptionQuota: null,
      subscriptionLastSync: '',
      selectedNode: '',
    }
  },
  async GetSettings() {
    return {
      mode: 'rule',
      tun: true,
      useSystemProxy: true,
      killSwitch: false,
      dnsLeakProtection: false,
      autoReconnect: true,
      autoUpdateSubscriptions: true,
      subscriptionIntervalMin: 360,
      autostart: false,
      closeToTray: true,
      mixedPort: 7890,
    }
  },
  async SaveSettings() { return null },
  async ListProfiles() { return [] },
  async ListNodes() { return [] },
  async CurrentNode() { return '' },
  async SelectNode() { return null },
  async TestNodeDelay() { return 0 },
  async TestAllNodes() { return [] },
  async ToggleConnect() { return null },
  async ImportProfileText() { return null },
  async SyncSubscription() { return null },
  async SyncAllSubscriptions() { return 0 },
  async SetActiveProfile() { return null },
  async CheckForUpdate() { return { available: false } },
  async DownloadUpdate() { return '' },
  async ApplyUpdate() { return null },
  async RelaunchAsAdmin() { return null },
  async ShowWindow() { return null },
  async QuitApp() { return null },
}

async function api() {
  if (window.go?.main?.App) return window.go.main.App
  try {
    return await import(/* @vite-ignore */ '../wailsjs/go/main/App.js')
  } catch {
    return mock
  }
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function fmtRate(n) {
  const v = Number(n || 0)
  if (v > 1024 * 1024) return (v / (1024 * 1024)).toFixed(1) + ' MB/s'
  if (v > 1024) return (v / 1024).toFixed(1) + ' KB/s'
  return v + ' B/s'
}

function fmtBytes(n) {
  const v = Number(n || 0)
  if (v < 0) return '∞'
  if (v > 1024 ** 3) return (v / 1024 ** 3).toFixed(2) + ' GB'
  if (v > 1024 ** 2) return (v / 1024 ** 2).toFixed(1) + ' MB'
  if (v > 1024) return (v / 1024).toFixed(1) + ' KB'
  return v + ' B'
}

function fmtExpire(unix) {
  const n = Number(unix || 0)
  if (!n) return 'n/a'
  return new Date(n * 1000).toLocaleString()
}

function render(status, profiles, settings, nodes, currentNode, message) {
  const connected = status.state === 'connected'
  const q = status.subscriptionQuota || {}
  const rows = profiles
    .map((p) => {
      const sub = p.subscriptionURL ? ' · sub' : ''
      const meta = `${(p.proxies || []).length} nodes${sub}`
      return `
      <div class="profile-row">
        <button class="profile ${status.activeProfile === p.name ? 'active' : ''}" data-profile="${escapeHtml(p.name)}">
          <strong>${escapeHtml(p.name)}</strong>
          <span>${escapeHtml(p.note || meta)}</span>
        </button>
        ${p.subscriptionURL ? `<button class="sync-one" data-sync="${escapeHtml(p.name)}">Sync</button>` : ''}
      </div>`
    })
    .join('')

  const nodeRows = (nodes || [])
    .map((n) => `
      <button class="node ${currentNode === n.name ? 'active' : ''}" data-node="${escapeHtml(n.name)}">
        <strong>${escapeHtml(n.name)}</strong>
        <span>${escapeHtml(n.type || 'proxy')} · ${n.delay ? n.delay + ' ms' : '—'}</span>
      </button>
    `)
    .join('')

  app.innerHTML = `
    <div class="shell">
      <aside class="sidebar">
        <div class="brand">MyInternetVPN</div>
        <p class="tagline">Windows client for myinternetvpn.com</p>
        <nav>
          <a class="nav active" href="#">Overview</a>
          <a class="nav" href="https://myinternetvpn.com" target="_blank" rel="noreferrer">Website</a>
        </nav>
        <div class="actions sidebar-actions">
          <button id="show-window" type="button">Show</button>
          <button id="quit-app" type="button">Quit</button>
        </div>
      </aside>
      <main>
        <header class="top">
          <div>
            <h1>Overview</h1>
            <p class="muted">mihomo · v${escapeHtml(status.appVersion || 'dev')} · admin: ${status.isAdmin ? 'yes' : 'no'}</p>
          </div>
          <div class="badge ${connected ? 'on' : 'off'}">${connected ? 'Connected' : 'Disconnected'}</div>
        </header>

        <section class="hero-card">
          <div>
            <div class="label">Status</div>
            <div class="status-title">${connected ? 'Protected' : 'Not connected'}</div>
            <div class="muted">
              Profile: ${escapeHtml(status.activeProfile || 'none')} ·
              Node: ${escapeHtml(currentNode || status.selectedNode || 'AUTO')} ·
              ↓ ${fmtRate(status.speedDown)} · ↑ ${fmtRate(status.speedUp)}
            </div>
            <div class="muted sub-meta">
              Traffic: ${fmtBytes(q.used)} / ${fmtBytes(q.total)} ·
              Left: ${fmtBytes(q.remaining)} ·
              Expires: ${fmtExpire(q.expireUnix)}${q.expired ? ' (expired)' : ''}
            </div>
          </div>
          <button id="toggle" class="cta ${connected ? 'danger' : ''}">
            ${connected ? 'Disconnect' : 'Connect'}
          </button>
        </section>

        <section class="grid three">
          <div class="card">
            <div class="card-title">Profiles</div>
            <div class="profiles">${rows || '<div class="muted">Import https subscription / share link / YAML.</div>'}</div>
            <form id="add-form" class="add-form">
              <input name="name" placeholder="Profile name" required />
              <input name="note" placeholder="Note (optional)" />
              <textarea name="proxy" rows="4" placeholder="https://.../sub or vless://... or Clash YAML" required></textarea>
              <button type="submit">Import</button>
            </form>
            <div class="actions">
              <button id="sync-all" type="button">Sync all</button>
            </div>
          </div>

          <div class="card">
            <div class="card-title">Nodes</div>
            <div class="nodes">${connected ? (nodeRows || '<div class="muted">No selectable nodes</div>') : '<div class="muted">Connect to list nodes</div>'}</div>
            <div class="actions">
              <button id="test-all" type="button" ${connected ? '' : 'disabled'}>URL-test all</button>
            </div>
          </div>

          <div class="card">
            <div class="card-title">Options</div>
            <form id="settings-form" class="add-form">
              <label>Mode
                <select name="mode">
                  <option value="rule" ${settings.mode === 'rule' ? 'selected' : ''}>rule</option>
                  <option value="global" ${settings.mode === 'global' ? 'selected' : ''}>global</option>
                  <option value="direct" ${settings.mode === 'direct' ? 'selected' : ''}>direct</option>
                </select>
              </label>
              <label><input type="checkbox" name="tun" ${settings.tun ? 'checked' : ''}/> TUN (Admin)</label>
              <label><input type="checkbox" name="useSystemProxy" ${settings.useSystemProxy ? 'checked' : ''}/> System proxy</label>
              <label><input type="checkbox" name="killSwitch" ${settings.killSwitch ? 'checked' : ''}/> Kill switch</label>
              <label><input type="checkbox" name="dnsLeakProtection" ${settings.dnsLeakProtection ? 'checked' : ''}/> DNS leak protection</label>
              <label><input type="checkbox" name="autoReconnect" ${settings.autoReconnect ? 'checked' : ''}/> Auto-reconnect</label>
              <label><input type="checkbox" name="autoUpdateSubscriptions" ${settings.autoUpdateSubscriptions ? 'checked' : ''}/> Auto-update subs</label>
              <label><input type="checkbox" name="autostart" ${settings.autostart ? 'checked' : ''}/> Autostart</label>
              <label><input type="checkbox" name="closeToTray" ${settings.closeToTray ? 'checked' : ''}/> Close to tray/hide</label>
              <label>Sub interval (min)
                <input name="subscriptionIntervalMin" type="number" min="15" value="${escapeHtml(settings.subscriptionIntervalMin || 360)}" />
              </label>
              <button type="submit">Save</button>
            </form>
            <div class="actions">
              <button id="elevate" type="button">Admin</button>
              <button id="update" type="button">Update</button>
            </div>
            ${message ? `<div class="message">${escapeHtml(message)}</div>` : ''}
          </div>
        </section>
      </main>
    </div>
  `
}

async function boot() {
  const bridge = await api()
  let message = ''
  let lastZip = ''
  const delayMap = {}

  async function refresh() {
    const [status, profiles, settings] = await Promise.all([
      bridge.GetStatus(),
      bridge.ListProfiles(),
      bridge.GetSettings(),
    ])
    let nodes = []
    let currentNode = status.selectedNode || ''
    if (status.state === 'connected') {
      try {
        nodes = await bridge.ListNodes()
        currentNode = (await bridge.CurrentNode()) || currentNode
        nodes = (nodes || []).map((n) => ({
          ...n,
          delay: delayMap[n.name] || n.delay || 0,
        }))
      } catch (_) {}
    }
    render(status, profiles, settings || {}, nodes, currentNode, message)

    document.getElementById('toggle')?.addEventListener('click', async () => {
      try {
        await bridge.ToggleConnect()
        message = ''
      } catch (err) {
        message = String(err)
      }
      await refresh()
    })

    document.querySelectorAll('[data-profile]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        try {
          await bridge.SetActiveProfile(btn.getAttribute('data-profile'))
          message = ''
        } catch (err) {
          message = String(err)
        }
        await refresh()
      })
    })

    document.querySelectorAll('[data-sync]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        try {
          await bridge.SyncSubscription(btn.getAttribute('data-sync'))
          message = 'Subscription synced'
        } catch (err) {
          message = String(err)
        }
        await refresh()
      })
    })

    document.querySelectorAll('[data-node]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        try {
          await bridge.SelectNode(btn.getAttribute('data-node'))
          message = 'Node selected'
        } catch (err) {
          message = String(err)
        }
        await refresh()
      })
    })

    document.getElementById('test-all')?.addEventListener('click', async () => {
      try {
        message = 'URL-testing nodes…'
        await refresh()
        const results = await bridge.TestAllNodes()
        Object.keys(delayMap).forEach((k) => delete delayMap[k])
        for (const r of results || []) {
          delayMap[r.name] = r.delay || 0
        }
        const ok = (results || []).filter((r) => r.delay > 0).length
        message = `URL-test done: ${ok}/${(results || []).length} reachable`
      } catch (err) {
        message = String(err)
      }
      await refresh()
    })

    document.getElementById('sync-all')?.addEventListener('click', async () => {
      try {
        const n = await bridge.SyncAllSubscriptions()
        message = `Synced ${n} subscription(s)`
      } catch (err) {
        message = String(err)
      }
      await refresh()
    })

    document.getElementById('add-form')?.addEventListener('submit', async (e) => {
      e.preventDefault()
      const fd = new FormData(e.target)
      try {
        await bridge.ImportProfileText(String(fd.get('name')), String(fd.get('note') || ''), String(fd.get('proxy')))
        message = 'Imported'
      } catch (err) {
        message = String(err)
      }
      await refresh()
    })

    document.getElementById('settings-form')?.addEventListener('submit', async (e) => {
      e.preventDefault()
      const fd = new FormData(e.target)
      try {
        await bridge.SaveSettings({
          ...settings,
          mode: String(fd.get('mode')),
          tun: fd.get('tun') === 'on',
          useSystemProxy: fd.get('useSystemProxy') === 'on',
          killSwitch: fd.get('killSwitch') === 'on',
          dnsLeakProtection: fd.get('dnsLeakProtection') === 'on',
          autoReconnect: fd.get('autoReconnect') === 'on',
          autoUpdateSubscriptions: fd.get('autoUpdateSubscriptions') === 'on',
          autostart: fd.get('autostart') === 'on',
          closeToTray: fd.get('closeToTray') === 'on',
          subscriptionIntervalMin: Number(fd.get('subscriptionIntervalMin') || 360),
        })
        message = 'Settings saved'
      } catch (err) {
        message = String(err)
      }
      await refresh()
    })

    document.getElementById('elevate')?.addEventListener('click', async () => {
      try {
        await bridge.RelaunchAsAdmin()
        message = 'Elevation requested'
      } catch (err) {
        message = String(err)
      }
    })

    document.getElementById('update')?.addEventListener('click', async () => {
      try {
        const res = await bridge.CheckForUpdate()
        if (!res?.available) {
          message = res?.reason || 'No update available'
          if (lastZip) {
            await bridge.ApplyUpdate(lastZip)
            message = 'Applying previous download…'
          }
        } else {
          lastZip = await bridge.DownloadUpdate()
          await bridge.ApplyUpdate(lastZip)
          message = 'Update downloaded and applying…'
        }
      } catch (err) {
        message = String(err)
      }
      await refresh()
    })

    document.getElementById('show-window')?.addEventListener('click', () => bridge.ShowWindow?.())
    document.getElementById('quit-app')?.addEventListener('click', () => bridge.QuitApp?.())
  }

  await refresh()
  setInterval(refresh, 3000)
}

boot()
