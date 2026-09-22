import './style.css'

const root = document.querySelector('#app')

const icons = {
  home: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 11.2 12 4l8 7.2"/><path d="M6.5 10.2V20h11V10.2"/></svg>`,
  profiles: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M5 7h14M5 12h14M5 17h14"/></svg>`,
  nodes: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><circle cx="12" cy="12" r="3.2"/><path d="M12 3.5v2.2M12 18.3v2.2M3.5 12h2.2M18.3 12h2.2M6.1 6.1l1.6 1.6M16.3 16.3l1.6 1.6M17.9 6.1l-1.6 1.6M7.7 16.3l-1.6 1.6"/></svg>`,
  connections: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="5" r="2.5"/><circle cx="6" cy="12" r="2.5"/><circle cx="18" cy="19" r="2.5"/><path d="M8.2 13.3l7.6 4.4M8.2 10.7l7.6-4.4"/></svg>`,
  settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 3.2v1.8M12 19v1.8M20.8 12h-1.8M5 12H3.2M18.2 5.8l-1.3 1.3M7.1 16.9l-1.3 1.3M18.2 18.2l-1.3-1.3M7.1 7.1 5.8 5.8"/></svg>`,
  plus: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 5v14M5 12h14"/></svg>`,
  refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 12a9 9 0 1 1-2.6-6.3"/><path d="M21 3v6h-6"/></svg>`,
  history: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 12a9 9 0 1 0 3-6.7"/><path d="M3 4v5h5"/><path d="M12 7v5l3 2"/></svg>`,
  caret: `<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="m7 10 5 6 5-6z"/></svg>`,
  chevron: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 6 6 6-6 6"/></svg>`,
  signal: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M2 20h2M7 20v-5M12 20V9M17 20V5M22 20v-9"/></svg>`,
  power: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M12 3v9"/><path d="M7.5 6.2a7.5 7.5 0 1 0 9 0"/></svg>`,
  search: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>`,
  trash: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 7h16M9 7V5h6v2M8 7l1 12h6l1-12"/></svg>`,
  logs: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M7 3.5h8.2L19 7.2V20.5H7z"/><path d="M15 3.5v4h4"/><path d="M10 11h6M10 14.5h6M10 18h4"/></svg>`,
  shield: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3 5 6v5c0 5 3.2 8.4 7 9.8 3.8-1.4 7-4.8 7-9.8V6l-7-3z"/></svg>`,
  lock: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>`,
  auto: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3.5v3M12 17.5v3M3.5 12h3M17.5 12h3"/><circle cx="12" cy="12" r="3.2"/><path d="M7.2 7.2l1.5 1.5M15.3 15.3l1.5 1.5M16.8 7.2l-1.5 1.5M8.7 15.3l-1.5 1.5"/></svg>`,
  cloudflare: `<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M16.5 17.6H6.2c-2.1 0-3.8-1.7-3.8-3.8 0-1.8 1.3-3.4 3.1-3.7.6-2.5 2.8-4.3 5.4-4.3 2.1 0 4 1.2 4.9 3.1.5-.2 1-.3 1.6-.3 2.3 0 4.1 1.8 4.1 4.1 0 2.3-1.8 4.1-4.1 4.1h-.9z"/><path d="M8.1 14.2h9.3c.9 0 1.7-.7 1.7-1.6 0-.8-.6-1.5-1.4-1.6l-.7-.1-.3-.6c-.6-1.4-2-2.3-3.5-2.3-1.7 0-3.2 1.1-3.7 2.7l-.2.6-.6.1c-1.1.1-1.9 1-1.9 2.1 0 .9.7 1.7 1.7 1.7h-.4z" opacity=".92"/></svg>`,
  close: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>`,
}

const mock = {
  async GetStatus() {
    return { state: 'disconnected', activeProfile: '', mode: 'rule', speedUp: 0, speedDown: 0, appVersion: 'dev', subscriptionQuota: null, selectedNode: '' }
  },
  async GetSettings() {
    return { mode: 'rule', tun: true, useSystemProxy: true, killSwitch: false, dnsLeakProtection: false, autoReconnect: true, autoUpdateSubscriptions: true, subscriptionIntervalMin: 360, autostart: false, closeToTray: true }
  },
  async SaveSettings() { return null },
  async ListProfiles() { return [] },
  async ListRunningApps() { return [{ name: 'AyuGram.exe', path: 'C:\\Apps\\AyuGram.exe', pid: 1 }] },
  async ListNodes() { return [] },
  async CurrentNode() { return '' },
  async SelectNode() { return null },
  async TestAllNodes() { return [] },
  async ConnectAndProbe() { return [] },
  async ToggleConnect() { return null },
  async ImportProfileText() { return null },
  async SyncSubscription() { return null },
  async SyncAllSubscriptions() { return 0 },
  async SetActiveProfile() { return null },
  async RemoveProfile() { return null },
  async CheckForUpdate() { return { available: false } },
  async DownloadUpdate() { return '' },
  async ApplyUpdate() { return null },
  async QuitApp() { return null },
  async GetLogsTail() { return '' },
  async GenerateWARPConfig() {
    return { privateKey: 'dev', localAddress: '172.16.0.2/32' }
  },
  async ListConnections() { return [] },
  async CloseConnection() { return null },
  async CloseAllConnections() { return null },
}

async function api() {
  // Wails injects bindings shortly after load; don't block the UI for long.
  for (let i = 0; i < 40; i++) {
    if (window.go?.main?.App) return window.go.main.App
    await new Promise((r) => setTimeout(r, 25))
  }
  if (window.go?.main?.App) return window.go.main.App
  try {
    return await import(/* @vite-ignore */ '../wailsjs/go/main/App.js')
  } catch {
    return mock
  }
}

// Shared with module-level helpers (app picker, chips). Boot upgrades bridge to Wails.
let bridge = mock
let state = { settings: {} }

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function fmtRate(n) {
  const v = Number(n || 0)
  if (v >= 1024 * 1024) return (v / (1024 * 1024)).toFixed(1) + ' МБ/с'
  if (v >= 1024) return (v / 1024).toFixed(1) + ' КБ/с'
  return Math.round(v) + ' Б/с'
}

function fmtBytes(n) {
  const v = Number(n || 0)
  if (!Number.isFinite(v) || v < 0) return '∞'
  if (v >= 1024 ** 3) return (v / 1024 ** 3).toFixed(2) + 'GiB'
  if (v >= 1024 ** 2) return (v / 1024 ** 2).toFixed(1) + 'MiB'
  if (v >= 1024) return (v / 1024).toFixed(1) + 'KiB'
  return Math.round(v) + 'B'
}

function daysLeft(unix) {
  let n = Number(unix || 0)
  if (!n) return '∞'
  // Guard: some cached payloads may still be ms.
  if (n >= 1e12) n = Math.floor(n / 1000)
  const days = Math.ceil((n * 1000 - Date.now()) / 86400000)
  if (days < 0) return 'истёк'
  return String(days)
}

function expireLabel(unix) {
  const d = daysLeft(unix)
  if (d === 'истёк') return 'срок истёк'
  return `осталось ${d} дней`
}

function quotaPercent(used, total) {
  const t = Number(total || 0)
  if (t <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((Number(used || 0) / t) * 100)))
}

function renderQuotaCard(state) {
  const status = state.status || {}
  const q = status.subscriptionQuota || {}
  const used = Number(q.used || 0)
  const total = Number(q.total || 0)
  const pct = quotaPercent(used, total)
  const active = status.activeProfile || ''
  const profiles = state.profiles || []
  const menuOpen = !!state.quotaMenuOpen
  const canSync = !!(active && profiles.some((p) => p.name === active && p.subscriptionURL))
  const menu = profiles.length
    ? profiles.map((p) => `
        <button class="quota-menu-item ${p.name === active ? 'active' : ''}" type="button" data-profile="${escapeHtml(p.name)}">
          <strong>${escapeHtml(p.name)}</strong>
          <span>${(p.proxies || []).length} серв.${p.subscriptionURL ? ' · подписка' : ''}</span>
        </button>`).join('')
    : `<button class="quota-menu-item" type="button" data-action="go-profiles"><strong>Добавить профиль</strong><span>Импорт подписки или URI</span></button>`

  return `
    <div class="quota-card ${menuOpen ? 'menu-open' : ''} ${q.expired ? 'is-expired' : ''}">
      <button class="quota-sync" type="button" data-action="sync-active" title="${canSync ? 'Обновить подписку' : 'Обновить подписки'}" ${active || profiles.length ? '' : 'disabled'}>
        ${icons.history}
      </button>
      <div class="quota-divider" aria-hidden="true"></div>
      <div class="quota-body">
        <div class="quota-profile-wrap">
          <button class="quota-profile" type="button" data-action="toggle-quota-menu" title="Сменить профиль">
            <strong id="active-profile">${escapeHtml(active || 'Нет профиля')}</strong>
            <span class="quota-caret">${icons.caret}</span>
          </button>
          <div class="quota-menu ${menuOpen ? 'open' : ''}" id="quota-profile-menu" role="menu">
            ${menu}
          </div>
        </div>
        <div class="bar" title="${pct ? pct + '%' : 'без лимита'}">
          <i id="quota-bar" style="width:${pct}%"></i>
        </div>
        <div class="quota-meta">
          <span id="quota-text">${fmtBytes(used)} / ${fmtBytes(total > 0 ? total : -1)}</span>
          <span id="quota-expire">${escapeHtml(expireLabel(q.expireUnix))}</span>
        </div>
      </div>
    </div>
  `
}

function modeLabel(mode) {
  switch (String(mode || '').toLowerCase()) {
    case 'global': return 'Все через VPN'
    case 'direct': return 'Без VPN'
    default: return 'По правилам'
  }
}

function nodeKind(type) {
  const t = String(type || '').toLowerCase()
  if (!t || t === 'proxy') return 'Сервер'
  if (t === 'urltest' || t === 'selector') return 'Группа'
  const labels = {
    vless: 'VLESS',
    vmess: 'VMess',
    trojan: 'Trojan',
    ss: 'Shadowsocks',
    ssr: 'SSR',
    tuic: 'TUIC',
    hysteria: 'Hysteria',
    hysteria2: 'Hysteria2',
    wireguard: 'WireGuard',
    ssh: 'SSH',
    socks5: 'SOCKS5',
    http: 'HTTP',
  }
  return labels[t] || t.toUpperCase()
}

function delayClass(ms) {
  if (!ms || ms <= 0) return 'bad'
  if (ms < 120) return 'good'
  if (ms < 300) return 'ok'
  return 'slow'
}

function delayBadgeHTML(delay, pinging) {
  if (pinging) {
    return `<span class="badge loading" title="Проверка…"><span class="ping-spinner" aria-hidden="true"></span></span>`
  }
  return `<span class="badge ${delayClass(delay)}">${delay > 0 ? delay + ' мс' : '—'}</span>`
}

function isAutoNode(n) {
  return n?.name === 'AUTO' || String(n?.type || '').toLowerCase() === 'urltest'
}

function nodeSubtitle(n) {
  if (isAutoNode(n)) {
    const now = String(n?.now || '').trim()
    if (!now) return 'Лучший сервер по задержке'
    const label = parseNodeVisual(now).rest || now
    return `Сейчас: ${label}`
  }
  return nodeKind(n?.type)
}

/** Regional Indicator Symbol A..Z → country code / Twemoji (Windows often shows flags as "DE"). */
const RI_A = 0x1F1E6
const RI_Z = 0x1F1FF

function parseNodeVisual(name) {
  const s = String(name || '').trim()
  if (!s) return { code: '', rest: '?', original: s }
  const chars = Array.from(s)
  let code = ''
  let restStart = 0

  if (chars.length >= 2) {
    const a = chars[0].codePointAt(0)
    const b = chars[1].codePointAt(0)
    if (a >= RI_A && a <= RI_Z && b >= RI_A && b <= RI_Z) {
      code = String.fromCharCode(65 + (a - RI_A)) + String.fromCharCode(65 + (b - RI_A))
      restStart = 2
      while (restStart < chars.length && /\s/u.test(chars[restStart])) restStart += 1
    }
  }

  // ASCII "DE Германия" / "DE-Germany" (no emoji in source)
  if (!code) {
    const m = s.match(/^([A-Za-z]{2})(?:\s+|[-_|·.]+)/)
    if (m) {
      code = m[1].toUpperCase()
      restStart = m[0].length
    }
  }

  let rest = chars.slice(restStart).join('').trim()
  if (!rest) rest = s
  return { code, rest, original: s }
}

function twemojiFlagURL(code) {
  const cc = String(code || '').toUpperCase()
  if (!/^[A-Z]{2}$/.test(cc)) return ''
  const hex = [...cc].map((ch) => (RI_A + ch.charCodeAt(0) - 65).toString(16)).join('-')
  return `https://cdn.jsdelivr.net/gh/twitter/twemoji@14.0.2/assets/72x72/${hex}.png`
}

function flagImgHTML(code, size = 20) {
  const cc = String(code || '').toUpperCase()
  const url = twemojiFlagURL(cc)
  if (!url) return ''
  const safe = escapeHtml(cc)
  return `<img class="flag-img" width="${size}" height="${size}" alt="${safe}" title="${safe}" data-cc="${safe}" src="${url}" loading="lazy" decoding="async" onerror="this.onerror=null;var s=document.createElement('span');s.className='flag-fallback';s.textContent=this.getAttribute('data-cc');this.replaceWith(s);" />`
}

function nodeBadgeHTML(name) {
  if (String(name || '') === 'AUTO') {
    return `<span class="auto-badge" title="Автовыбор">${icons.auto}</span>`
  }
  const v = parseNodeVisual(name)
  if (v.code) {
    const img = flagImgHTML(v.code, 22)
    if (img) return img
    return `<span class="flag-fallback">${escapeHtml(v.code)}</span>`
  }
  // Prefer letters (incl. Cyrillic); avoid "#1" → "1"
  const chars = Array.from(v.rest || name || '?')
  const letters = chars.filter((c) => /\p{L}/u.test(c))
  if (letters.length >= 2) return escapeHtml((letters[0] + letters[1]).toUpperCase())
  if (letters.length === 1) return escapeHtml(letters[0].toUpperCase())
  const grapheme = chars[0] || '?'
  return escapeHtml(grapheme)
}

function formatNodeTitleHTML(name) {
  if (String(name || '') === 'AUTO') {
    return `<span class="node-title-text">Автовыбор</span>`
  }
  const v = parseNodeVisual(name)
  if (v.code) {
    return `<span class="node-title-row"><span class="node-title-text">${escapeHtml(v.rest)}</span></span>`
  }
  return `<span class="node-title-text">${escapeHtml(v.original || name || '')}</span>`
}

/** @deprecated use nodeBadgeHTML — kept for call sites expecting plain text */
function nodeInitials(name) {
  const v = parseNodeVisual(name)
  if (v.code) return v.code
  const letters = Array.from(v.rest || '').filter((c) => /\p{L}/u.test(c))
  if (letters.length >= 2) return (letters[0] + letters[1]).toUpperCase()
  if (letters.length === 1) return letters[0].toUpperCase()
  return Array.from(String(name || '?'))[0] || '?'
}

function friendlyError(err) {
  const s = String(err || '')
  if (/no active profile/i.test(s)) return 'Сначала добавьте и выберите профиль'
  if (/not connected/i.test(s)) return 'Сначала подключитесь'
  if (/administrator|elevation|ErrNeedAdmin/i.test(s)) return 'Нужны права администратора'
  if (/kill switch requires/i.test(s)) return 'Включите «Блокировка без VPN» только после настройки сетевого интерфейса'
  return s.replace(/^Error:\s*/i, '').replace(/^.*?:\s*/, (m) => (m.length > 40 ? '' : m)) || 'Что-то пошло не так'
}

/** Replace native <select> popups (ugly in WebView2) with themed custom dropdowns. */
function enhanceSelects(scope) {
  if (!scope) return
  scope.querySelectorAll('select:not([data-enhanced])').forEach((sel) => {
    sel.setAttribute('data-enhanced', '1')
    sel.classList.add('select-native')
    const wrap = document.createElement('div')
    wrap.className = 'cselect'
    sel.parentNode.insertBefore(wrap, sel)
    wrap.appendChild(sel)

    const trigger = document.createElement('button')
    trigger.type = 'button'
    trigger.className = 'cselect-trigger'
    trigger.setAttribute('aria-haspopup', 'listbox')

    const menu = document.createElement('div')
    menu.className = 'cselect-menu'
    menu.setAttribute('role', 'listbox')

    function sync() {
      const opt = sel.options[sel.selectedIndex]
      trigger.innerHTML = `<span>${escapeHtml(opt ? opt.textContent : '')}</span><span class="cselect-chev">${icons.chevron}</span>`
      menu.innerHTML = Array.from(sel.options).map((o, i) => (
        `<button type="button" class="cselect-option${i === sel.selectedIndex ? ' active' : ''}" role="option" data-value="${escapeHtml(o.value)}" ${o.disabled ? 'disabled' : ''}>${escapeHtml(o.textContent)}</button>`
      )).join('')
    }
    sync()
    wrap.appendChild(trigger)
    wrap.appendChild(menu)

    trigger.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      const open = !wrap.classList.contains('open')
      scope.querySelectorAll('.cselect.open').forEach((el) => el.classList.remove('open'))
      wrap.classList.toggle('open', open)
    })
    menu.addEventListener('click', (e) => {
      const btn = e.target.closest('.cselect-option')
      if (!btn || btn.disabled) return
      e.preventDefault()
      e.stopPropagation()
      sel.value = btn.getAttribute('data-value')
      sel.dispatchEvent(new Event('change', { bubbles: true }))
      sel.dispatchEvent(new Event('input', { bubbles: true }))
      wrap.classList.remove('open')
      sync()
    })
  })
}

const titles = {
  home: 'Подключение',
  profiles: 'Профили',
  nodes: 'Серверы',
  connections: 'Соединения',
  logs: 'Логи',
  settings: 'Настройки',
}

function shellHTML() {
  return `
    <div class="app">
      <aside class="sidebar">
        <div class="brand-block">
          <div class="brand">Мой <span class="brand-accent">VPN</span></div>
          <div class="brand-sub">Простой и быстрый VPN</div>
        </div>
        <nav class="nav" id="nav">
          <button class="nav-btn active" data-view="home" type="button">${icons.home}<span>Главная</span></button>
          <button class="nav-btn" data-view="profiles" type="button">${icons.profiles}<span>Профили</span></button>
          <button class="nav-btn" data-view="nodes" type="button">${icons.nodes}<span>Серверы</span></button>
          <button class="nav-btn" data-view="connections" type="button">${icons.connections}<span>Соединения</span></button>
          <button class="nav-btn" data-view="logs" type="button">${icons.logs}<span>Логи</span></button>
          <button class="nav-btn" data-view="settings" type="button">${icons.settings}<span>Настройки</span></button>
        </nav>
        <div class="sidebar-foot">
          <div class="traffic-card">
            <div class="traffic-label">Скорость</div>
            <div class="traffic-rows">
              <div class="traffic-row"><span class="dir down">↓ Загрузка</span><strong id="speed-down">0 Б/с</strong></div>
              <div class="traffic-row"><span class="dir up">↑ Отдача</span><strong id="speed-up">0 Б/с</strong></div>
            </div>
          </div>
          <button class="quit-btn" type="button" data-action="quit">Выйти</button>
        </div>
      </aside>
      <section class="main">
        <div class="main-bg" aria-hidden="true"></div>
        <header class="topbar">
          <div>
            <h1 id="view-title">Подключение</h1>
            <div class="meta" id="view-meta">—</div>
          </div>
          <div class="top-actions" id="top-actions"></div>
        </header>
        <div class="content" id="content"></div>
        <div class="toast" id="toast"></div>
      </section>
    </div>
  `
}

function topActionsFor(view) {
  if (view === 'home' || view === 'profiles') {
    return `
      <button class="icon-btn" type="button" data-action="sync-all" title="Обновить подписки">${icons.refresh}</button>
      <button class="icon-btn" type="button" data-action="go-profiles" title="Добавить профиль">${icons.plus}</button>`
  }
  if (view === 'nodes') {
    return `<button class="ghost compact" type="button" data-action="test-all">Проверить пинг</button>`
  }
  if (view === 'connections') {
    return `<button class="ghost compact" type="button" data-action="close-all-connections">Закрыть все</button>`
  }
  if (view === 'logs') {
    return `<button class="ghost compact" type="button" data-action="refresh-logs">Обновить</button>`
  }
  return ''
}

function renderHome(state) {
  const { status, currentNode, nodes, busy } = state
  const connected = status.state === 'connected'
  const node = currentNode || status.selectedNode || ''
  const nodeInfo = (nodes || []).find((n) => n.name === node)
  const delay = nodeInfo?.delay || 0
  const hasProfile = !!status.activeProfile
  const warpOn = !!(status.warpEnabled && connected)
  const titleHTML = warpOn && node
    ? `<span class="node-title-row">${icons.lock}<span class="warp-chain">WARP →</span>${formatNodeTitleHTML(node)}</span>`
    : (node ? formatNodeTitleHTML(node) : 'Автовыбор')

  return `
    <div class="home">
      ${renderQuotaCard(state)}

      <div class="stage">
        <button id="power-btn" class="power ${connected ? 'on' : 'off'} ${busy ? 'busy' : ''}" type="button" data-action="toggle" ${hasProfile ? '' : 'disabled'} aria-label="Подключить">
          <span class="power-ring"></span>
          <div class="power-mark">${icons.power}</div>
        </button>
        <div id="status-line" class="status-line ${connected ? 'on' : 'off'}">${connected ? 'Подключено' : hasProfile ? 'Отключено' : 'Добавьте профиль'}</div>
        <div id="warp-badge" class="warp-badge ${warpOn ? 'show' : ''}">${icons.shield}<span>Защищено с помощью WARP</span></div>
        <div class="latency">
          ${icons.signal}
          <strong id="latency-value">${delay > 0 ? delay + ' мс' : connected ? '—' : 'ожидание'}</strong>
          <span class="dot">·</span>
          <span id="mode-value">${escapeHtml(modeLabel(status.mode))}</span>
        </div>
        <div class="ip-line" id="public-ip-line" title="Публичный IP">
          <span class="ip-label">IP</span>
          <strong id="public-ip-value">${escapeHtml(status.publicIP || (connected ? '...' : '-'))}</strong>
        </div>
        ${!hasProfile ? `<button class="primary cta-inline" type="button" data-action="go-profiles">Добавить профиль</button>` : ''}
      </div>

      <button class="server-card ${warpOn ? 'warp-on' : ''}" type="button" data-action="go-nodes" ${connected || hasProfile ? '' : 'disabled'}>
        <div class="server-flag" id="server-flag">${node ? nodeBadgeHTML(node) : 'AU'}${warpOn ? `<span class="warp-cloud" title="Cloudflare WARP">${icons.cloudflare}</span>` : ''}</div>
        <div>
          <div class="title" id="server-title">${titleHTML}</div>
          <div class="sub" id="server-sub">${escapeHtml(
            node
              ? (node === 'AUTO' || String(nodeInfo?.type || '').toLowerCase() === 'urltest'
                  ? nodeSubtitle(nodeInfo || { name: 'AUTO', type: 'URLTest' })
                  : `${nodeKind(nodeInfo?.type)} · нажмите, чтобы выбрать`)
              : 'нажмите, чтобы выбрать'
          )}</div>
        </div>
        <div class="server-right">
          ${warpOn ? '<span class="warp-tag">WARP</span>' : ''}
          <div class="chev">${icons.chevron}</div>
        </div>
      </button>
    </div>
  `
}

function renderProfiles(state) {
  const { status, profiles, showImport } = state
  const rows = (profiles || []).map((p) => {
    const count = (p.proxies || []).length
    const kind = p.subscriptionURL ? 'Подписка' : 'Конфиг'
    const active = status.activeProfile === p.name
    return `
      <div class="row card-row ${active ? 'is-active' : ''}">
        <button class="item ${active ? 'active' : ''}" type="button" data-profile="${escapeHtml(p.name)}">
          <strong>${escapeHtml(p.name)}</strong>
          <span>${count} серверов · ${kind}${p.note ? ' · ' + escapeHtml(p.note) : ''}</span>
        </button>
        <div class="row-actions">
          ${p.subscriptionURL ? `<button class="icon-ghost" type="button" data-sync="${escapeHtml(p.name)}" title="Обновить">${icons.refresh}</button>` : ''}
          <button class="icon-ghost danger" type="button" data-remove="${escapeHtml(p.name)}" title="Удалить">${icons.trash}</button>
        </div>
      </div>`
  }).join('')

  return `
    <div class="panel">
      <div class="panel-title">
        <span>Ваши профили</span>
        <button class="ghost compact" type="button" data-action="toggle-import">${showImport ? 'Скрыть форму' : 'Добавить'}</button>
      </div>
      <div class="list">${rows || `
        <div class="empty">
          <p>Пока нет профилей</p>
          <button class="primary" type="button" data-action="toggle-import">Импортировать подписку</button>
        </div>`}</div>
    </div>
    ${showImport || !(profiles || []).length ? `
    <div class="panel">
      <div class="panel-title">Импорт</div>
      <form id="add-form" class="form">
        <label>Ссылка или конфиг
          <textarea name="proxy" rows="4" placeholder="Ссылка на подписку (https://…)&#10;или готовая строка сервера&#10;или текст конфигурации" required></textarea>
        </label>
        <label>Название
          <input name="name" placeholder="Например: Основной" />
        </label>
        <button class="primary" type="submit">Импортировать</button>
      </form>
    </div>` : ''}
  `
}

function sortNodesList(nodes, sortKey) {
  const list = [...(nodes || [])]
  const key = sortKey === 'name' ? 'name' : 'ping'
  list.sort((a, b) => {
    if (a.name === 'AUTO') return -1
    if (b.name === 'AUTO') return 1
    if (key === 'name') {
      const an = parseNodeVisual(a.name).rest || a.name
      const bn = parseNodeVisual(b.name).rest || b.name
      return String(an).localeCompare(String(bn), 'ru', { sensitivity: 'base', numeric: true })
    }
    const da = Number(a.delay || 0)
    const db = Number(b.delay || 0)
    if (da <= 0 && db <= 0) {
      return String(a.name).localeCompare(String(b.name), 'ru', { sensitivity: 'base' })
    }
    if (da <= 0) return 1
    if (db <= 0) return -1
    if (da !== db) return da - db
    return String(a.name).localeCompare(String(b.name), 'ru', { sensitivity: 'base' })
  })
  return list
}

function renderNodes(state) {
  const { status, nodes, currentNode, nodeQuery, nodeSort } = state
  const connected = status.state === 'connected'
  if (!connected) {
    return `
      <div class="panel nodes-panel">
        <div class="empty">
          <p>Список серверов доступен после подключения</p>
          <button class="primary" type="button" data-action="go-home">На главную</button>
        </div>
      </div>`
  }

  const sort = nodeSort === 'name' ? 'name' : 'ping'
  const q = String(nodeQuery || '').trim().toLowerCase()
  const filtered = sortNodesList(
    (nodes || []).filter((n) => !q || String(n.name).toLowerCase().includes(q)),
    sort
  )
  const rows = filtered.map((n) => {
    const delay = n.delay || 0
    const pinging = !!(state.pinging && state.pinging[n.name])
    const auto = isAutoNode(n)
    return `
      <button class="item node-item ${auto ? 'node-auto' : ''} ${currentNode === n.name ? 'active' : ''} ${pinging ? 'is-pinging' : ''}" type="button" data-node="${escapeHtml(n.name)}">
        <div class="node-flag ${auto ? 'node-flag-auto' : ''}">${nodeBadgeHTML(n.name)}</div>
        <div class="node-main">
          <strong class="node-title">${formatNodeTitleHTML(n.name)}</strong>
          <span class="node-sub">${escapeHtml(nodeSubtitle(n))}</span>
        </div>
        ${delayBadgeHTML(delay, pinging)}
      </button>`
  }).join('')

  return `
    <div class="panel nodes-panel">
      <div class="nodes-toolbar">
        <div class="search-box">
          ${icons.search}
          <input id="node-search" type="search" placeholder="Поиск сервера" value="${escapeHtml(nodeQuery || '')}" />
        </div>
        <div class="sort-group" role="group" aria-label="Сортировка">
          <button class="sort-btn ${sort === 'ping' ? 'active' : ''}" type="button" data-node-sort="ping">По пингу</button>
          <button class="sort-btn ${sort === 'name' ? 'active' : ''}" type="button" data-node-sort="name">По названию</button>
        </div>
      </div>
      <div class="list nodes-list">${rows || '<div class="empty">Ничего не найдено</div>'}</div>
    </div>
  `
}

function switchRow(name, checked, label, hint) {
  return `
    <label class="switch-row">
      <div>
        <div class="switch-title">${label}</div>
        ${hint ? `<div class="switch-hint">${hint}</div>` : ''}
      </div>
      <input type="checkbox" name="${name}" ${checked ? 'checked' : ''}/>
      <span class="switch"></span>
    </label>`
}

const settingsMenu = [
  { id: 'general', title: 'Общие', hint: 'Автозапуск, трей, обновления', icon: 'layers' },
  { id: 'routing', title: 'Маршрутизация', hint: 'Режим VPN, приложения и сайты', icon: 'route' },
  { id: 'dns', title: 'DNS', hint: 'Как приложение узнаёт адреса сайтов', icon: 'dns' },
  { id: 'inbound', title: 'Подключение', hint: 'Порт, полный перехват трафика', icon: 'inbox' },
  { id: 'tricks', title: 'Ускорение и обход', hint: 'Доп. настройки соединения', icon: 'scissors' },
  { id: 'warp', title: 'WARP', hint: 'Доп. защита через Cloudflare', icon: 'cloud' },
]

const settingsIcons = {
  layers: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="m12 3 9 5-9 5-9-5 9-5z"/><path d="m3 12 9 5 9-5"/><path d="m3 17 9 5 9-5"/></svg>`,
  route: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M4 17h6a4 4 0 0 0 4-4V7"/><circle cx="18" cy="7" r="2"/><circle cx="6" cy="17" r="2"/></svg>`,
  dns: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><rect x="4" y="4" width="16" height="6" rx="1.5"/><rect x="4" y="14" width="16" height="6" rx="1.5"/><path d="M8 7h.01M8 17h.01"/></svg>`,
  inbox: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M4 13h4l2 3h4l2-3h4v5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-5z"/><path d="M4 13 6.5 5h11L20 13"/></svg>`,
  scissors: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><circle cx="6" cy="6" r="2.5"/><circle cx="6" cy="18" r="2.5"/><path d="m20 4-9.5 9.5M20 20 8.5 8.5"/></svg>`,
  cloud: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M7 18h10a4 4 0 0 0 .5-8 6 6 0 0 0-11.5-1.5A3.5 3.5 0 0 0 7 18z"/></svg>`,
}

function settingsBack() {
  return `<button class="settings-back" type="button" data-settings-section="">${icons.chevron.replace('m9 6 6 6-6 6', 'm15 6-6 6 6 6')}<span>Назад</span></button>`
}

function renderSettingsMenu(state = {}) {
  const status = state.status || {}
  const s = state.settings || {}
  const connected = status.state === 'connected'
  const version = status.appVersion || 'dev'
  const core = status.coreVersion || '—'
  const profile = status.activeProfile || 'нет'
  const mode = modeLabel(status.mode || s.mode || 'rule')
  const port = status.mixedPort || s.mixedPort || 7890
  const tun = !!(status.tun ?? s.tun)
  const warp = !!status.warpEnabled
  const whitelist = !!(s.routeWhitelist)
  const node = state.currentNode || status.selectedNode || ''
  const ip = state.publicIP || status.publicIP || ''

  const chips = [
    `<span class="about-chip ${connected ? 'ok' : 'off'}">${connected ? 'Подключено' : 'Отключено'}</span>`,
    `<span class="about-chip">${escapeHtml(mode)}</span>`,
    `<span class="about-chip ${tun ? 'on' : ''}">${tun ? 'Весь трафик' : 'Только приложения'}</span>`,
    warp ? `<span class="about-chip on">WARP</span>` : '',
    whitelist ? `<span class="about-chip warn">Только свои сайты</span>` : '',
  ].filter(Boolean).join('')

  return `
    <div class="settings-menu">
      ${settingsMenu.map((item) => `
        <button class="settings-link" type="button" data-settings-section="${item.id}">
          <span class="settings-link-icon">${settingsIcons[item.icon]}</span>
          <span class="settings-link-text">
            <strong>${item.title}</strong>
            <span>${item.hint}</span>
          </span>
          <span class="settings-link-chev">${icons.chevron}</span>
        </button>
      `).join('')}
      <div class="panel about-mini">
        <div class="about-mini-top">
          <img class="logo" src="./appicon.png" width="48" height="48" alt="" />
          <div class="about-mini-title">
            <div class="about-mini-name">
              <strong>Мой VPN Client</strong>
              <span class="about-ver">v${escapeHtml(version)}</span>
            </div>
            <div class="about-chips">${chips}</div>
          </div>
        </div>
        <div class="about-mini-grid">
          <div class="about-stat">
            <span class="about-stat-label">Профиль</span>
            <strong id="about-profile">${escapeHtml(profile)}</strong>
          </div>
          <div class="about-stat">
            <span class="about-stat-label">Сервер</span>
            <strong id="about-node">${escapeHtml(node || '—')}</strong>
          </div>
          <div class="about-stat">
            <span class="about-stat-label">Порт</span>
            <strong>${escapeHtml(String(port))}</strong>
          </div>
          <div class="about-stat">
            <span class="about-stat-label">Движок</span>
            <strong>${escapeHtml(core)}</strong>
          </div>
          <div class="about-stat about-stat-wide">
            <span class="about-stat-label">Публичный IP</span>
            <strong id="about-ip">${escapeHtml(ip || (connected ? '…' : '—'))}</strong>
          </div>
        </div>
      </div>
    </div>
  `
}

function renderSettingsGeneral(s, v) {
  return `
    <form id="settings-form" class="settings" data-section="general">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">Общие</div>
        ${switchRow('autostart', s.autostart, 'Автозапуск', 'Запускать VPN вместе с Windows')}
        ${switchRow('closeToTray', s.closeToTray, 'Сворачивать в трей', 'Крестик скрывает окно, а не закрывает программу')}
        ${switchRow('autoReconnect', s.autoReconnect, 'Автопереподключение', 'Самим восстановить связь, если VPN оборвался')}
        ${switchRow('killSwitch', s.killSwitch, 'Блокировка без VPN', 'Без VPN интернет будет недоступен')}
        ${switchRow('dnsLeakProtection', s.dnsLeakProtection, 'Защита DNS', 'Не отдавать запросы к сайтам мимо VPN')}
        ${switchRow('autoUpdateSubscriptions', s.autoUpdateSubscriptions, 'Автообновление подписок', 'Периодически обновлять список серверов')}
        <label class="field">Как часто обновлять подписки
          <select name="subscriptionIntervalMin">
            <option value="60" ${Number(s.subscriptionIntervalMin) === 60 ? 'selected' : ''}>Каждый час</option>
            <option value="180" ${Number(s.subscriptionIntervalMin) === 180 ? 'selected' : ''}>Каждые 3 часа</option>
            <option value="360" ${![60, 180, 720, 1440].includes(Number(s.subscriptionIntervalMin)) ? 'selected' : ''}>Каждые 6 часов</option>
            <option value="720" ${Number(s.subscriptionIntervalMin) === 720 ? 'selected' : ''}>Каждые 12 часов</option>
            <option value="1440" ${Number(s.subscriptionIntervalMin) === 1440 ? 'selected' : ''}>Раз в сутки</option>
          </select>
        </label>
        <label class="field">Проверка связи (секунды)
          <input name="healthIntervalSec" type="number" min="5" max="600" value="${Number(s.healthIntervalSec) || 30}"/>
        </label>
        <label class="field">Подробность логов
          <select name="logLevel">
            <option value="silent" ${s.logLevel === 'silent' ? 'selected' : ''}>Выключены</option>
            <option value="error" ${s.logLevel === 'error' ? 'selected' : ''}>Только ошибки</option>
            <option value="warning" ${s.logLevel === 'warning' ? 'selected' : ''}>Предупреждения</option>
            <option value="info" ${!s.logLevel || s.logLevel === 'info' ? 'selected' : ''}>Обычные</option>
            <option value="debug" ${s.logLevel === 'debug' ? 'selected' : ''}>Подробные</option>
          </select>
        </label>
        <div class="actions">
          <button class="ghost" type="button" data-action="update">Проверить обновления</button>
        </div>
        <div class="muted" style="margin-top:10px">Версия ${escapeHtml(v)} · настройки сохраняются сразу</div>
        <div class="muted" style="margin-top:8px">Можно открывать программу по ссылкам вида myvpn://… (подключение, импорт профиля).</div>
      </div>
    </form>
  `
}

function parseAppListText(text) {
  return String(text || '')
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#'))
}

function renderAppChips(apps) {
  if (!apps.length) return '<span class="muted">Список пуст — добавьте приложение</span>'
  return apps.map((name) => `
    <button type="button" class="app-chip" data-app-remove="${escapeHtml(name)}" title="Убрать">
      <span>${escapeHtml(name)}</span><span class="app-chip-x" aria-hidden="true">×</span>
    </button>`).join('')
}

function syncAppRouteListField(apps) {
  const ta = document.getElementById('app-route-list')
  const chips = document.getElementById('app-chip-list')
  const text = apps.join('\n')
  if (ta) {
    ta.value = text
    ta.dispatchEvent(new Event('input', { bubbles: true }))
  }
  if (chips) chips.innerHTML = renderAppChips(apps)
  if (state.settings) state.settings.appRouteList = text
}

function currentAppRouteList() {
  const ta = document.getElementById('app-route-list')
  return parseAppListText(ta ? ta.value : (state.settings?.appRouteList || ''))
}

function addAppToRouteList(name) {
  name = String(name || '').trim().replace(/^["']|["']$/g, '')
  if (!name) return
  // Prefer basename for PROCESS-NAME
  const base = name.split(/[/\\]/).pop() || name
  const apps = currentAppRouteList()
  if (apps.some((a) => a.toLowerCase() === base.toLowerCase())) return
  apps.push(base)
  apps.sort((a, b) => a.localeCompare(b, 'en', { sensitivity: 'base' }))
  syncAppRouteListField(apps)
}

function removeAppFromRouteList(name) {
  const key = String(name || '').toLowerCase()
  const apps = currentAppRouteList().filter((a) => a.toLowerCase() !== key)
  syncAppRouteListField(apps)
}

async function openAppPicker() {
  const picker = document.getElementById('app-picker')
  const list = document.getElementById('app-picker-list')
  const q = document.getElementById('app-picker-q')
  if (!picker || !list) return
  picker.classList.remove('hidden')
  list.innerHTML = '<div class="muted">Загрузка…</div>'
  let apps = []
  try {
    apps = (await bridge.ListRunningApps()) || []
  } catch (err) {
    list.innerHTML = `<div class="muted">${escapeHtml(friendlyError(err))}</div>`
    return
  }
  const selected = new Set(currentAppRouteList().map((a) => a.toLowerCase()))
  const paint = () => {
    const query = String(q?.value || '').trim().toLowerCase()
    const filtered = apps.filter((a) => {
      const n = String(a.name || '').toLowerCase()
      const p = String(a.path || '').toLowerCase()
      return !query || n.includes(query) || p.includes(query)
    })
    if (!filtered.length) {
      list.innerHTML = '<div class="muted">Ничего не найдено</div>'
      return
    }
    list.innerHTML = filtered.map((a) => {
      const name = a.name || ''
      const on = selected.has(name.toLowerCase())
      return `<button type="button" class="app-picker-item ${on ? 'on' : ''}" data-app-add="${escapeHtml(name)}">
        <strong>${escapeHtml(name)}</strong>
        <span>${escapeHtml(a.path || '')}</span>
      </button>`
    }).join('')
  }
  if (q && !q._bound) {
    q._bound = true
    q.addEventListener('input', paint)
  }
  paint()
  if (q) q.focus()
}

function renderAppRoutePanel(s) {
  const mode = String(s.appRouteMode || 'off').toLowerCase()
  const apps = parseAppListText(s.appRouteList)
  return `
    <div class="panel">
      <div class="panel-title">Приложения</div>
      <p class="muted rules-hint">Можно пускать через VPN только выбранные программы (или наоборот — исключить их). Нужен режим «Весь трафик системы». Пример: белый список с Telegram — в VPN уйдёт только он, браузер и остальное работают как обычно.</p>
      <p class="muted rules-hint">Пока включён этот режим, блокировка интернета без VPN и защита DNS отключаются сами — иначе сайты вне списка перестанут открываться. Смена списка применяется сразу, без ручного перезапуска.</p>
      <label class="field">Режим для приложений
        <select name="appRouteMode" id="app-route-mode">
          <option value="off" ${mode === 'off' ? 'selected' : ''}>Выключено — как настроено выше</option>
          <option value="whitelist" ${mode === 'whitelist' ? 'selected' : ''}>Только выбранные через VPN</option>
          <option value="blacklist" ${mode === 'blacklist' ? 'selected' : ''}>Выбранные без VPN, остальное через VPN</option>
        </select>
      </label>
      <textarea name="appRouteList" id="app-route-list" class="sr-only" aria-hidden="true">${escapeHtml(s.appRouteList || '')}</textarea>
      <div class="app-route-toolbar">
        <button class="ghost compact" type="button" data-action="pick-apps">Из запущенных…</button>
        <button class="ghost compact" type="button" data-action="add-app-manual">Добавить вручную</button>
      </div>
      <div class="app-chip-list" id="app-chip-list">${renderAppChips(apps)}</div>
      <div id="app-picker" class="app-picker hidden">
        <div class="app-picker-head">
          <input id="app-picker-q" type="search" placeholder="Найти программу…" autocomplete="off" />
          <button class="ghost compact" type="button" data-action="close-app-picker">Закрыть</button>
        </div>
        <div id="app-picker-list" class="app-picker-list"><div class="muted">Загрузка…</div></div>
      </div>
    </div>
  `
}

function renderSettingsRouting(s) {
  const geo = String(s.bypassGeoip || '').toUpperCase()
  const ping = s.pingMethod || 'proxy-http-get'
  const preset = s.urlTestPreset || 'gstatic'
  return `
    <form id="settings-form" class="settings" data-section="routing">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">Маршрутизация</div>
        <label class="field">Режим
          <select name="mode">
            <option value="rule" ${s.mode === 'rule' ? 'selected' : ''}>По правилам</option>
            <option value="global" ${s.mode === 'global' ? 'selected' : ''}>Все через VPN</option>
            <option value="direct" ${s.mode === 'direct' ? 'selected' : ''}>Без VPN</option>
          </select>
        </label>
        ${switchRow('ipv6', s.ipv6, 'IPv6', 'Разрешить современный формат адресов (обычно можно не включать)')}
        ${switchRow('bypassLan', s.bypassLan !== false, 'Обход домашней сети', 'Принтеры, роутер и устройства в локальной сети — без VPN')}
        <label class="field">Сайты своей страны без VPN
          <select name="bypassGeoip">
            <option value="CN" ${geo === 'CN' || geo === '' ? 'selected' : ''}>Китай</option>
            <option value="RU" ${geo === 'RU' ? 'selected' : ''}>Россия</option>
            <option value="IR" ${geo === 'IR' ? 'selected' : ''}>Иран</option>
            <option value="OFF" ${geo === 'OFF' || geo === 'NONE' ? 'selected' : ''}>Выключено</option>
          </select>
        </label>
      </div>
      ${renderAppRoutePanel(s)}
      <div class="panel">
        <div class="panel-title">Свои правила для сайтов</div>
        ${switchRow('routeWhitelist', !!s.routeWhitelist, 'Только перечисленные сайты', 'Всё, чего нет в списках ниже, будет заблокировано')}
        <p class="muted rules-hint">По одному адресу в строке: сайт (example.com), подсеть (192.168.0.0/16) или шаблон. Сначала действует «Блок», потом «Без VPN», потом «Через VPN»${s.routeWhitelist ? ', остальное закрыто' : ', затем правило страны выше'}. Вместе с режимом «Приложения» доменные списки тоже работают (по SNI) — например Cursor через VPN по списку сайтов, даже если самого Cursor.exe нет в приложениях.</p>
        <label class="field">Блокировать
          <textarea name="routeBlock" rows="5" placeholder="ads.example.com&#10;tracker.example.com">${escapeHtml(s.routeBlock || '')}</textarea>
        </label>
        <label class="field">Без VPN (напрямую)
          <textarea name="routeDirect" rows="5" placeholder="example.com&#10;bank.ru&#10;192.168.0.0/16">${escapeHtml(s.routeDirect || '')}</textarea>
        </label>
        <label class="field">Всегда через VPN
          <textarea name="routeProxy" rows="5" placeholder="blocked-site.com&#10;another-site.net">${escapeHtml(s.routeProxy || '')}</textarea>
        </label>
      </div>
      <div class="panel">
        <div class="panel-title">Проверка серверов</div>
        <label class="field">Способ проверки скорости
          <select name="pingMethod">
            <option value="proxy-http-get" ${ping === 'proxy-http-get' ? 'selected' : ''}>Через VPN (обычно лучше)</option>
            <option value="proxy-http-head" ${ping === 'proxy-http-head' ? 'selected' : ''}>Через VPN (быстрый запрос)</option>
            <option value="tcp" ${ping === 'tcp' ? 'selected' : ''}>Прямое соединение с сервером</option>
            <option value="http-get" ${ping === 'http-get' ? 'selected' : ''}>Сайт напрямую (без VPN)</option>
            <option value="icmp" ${ping === 'icmp' ? 'selected' : ''}>Простой пинг</option>
          </select>
        </label>
        <label class="field">Сайт для проверки
          <select name="urlTestPreset">
            <option value="gstatic" ${preset === 'gstatic' ? 'selected' : ''}>Google</option>
            <option value="cloudflare" ${preset === 'cloudflare' ? 'selected' : ''}>Cloudflare</option>
            <option value="apple" ${preset === 'apple' ? 'selected' : ''}>Apple</option>
            <option value="custom" ${preset === 'custom' ? 'selected' : ''}>Свой адрес</option>
          </select>
        </label>
        <label class="field">Свой адрес для проверки
          <input name="urlTestUrl" value="${escapeHtml(s.urlTestUrl || 'https://www.gstatic.com/generate_204')}" placeholder="https://…"/>
        </label>
        <label class="field">Как часто автовыбор лучшего сервера (сек)
          <input name="urlTestIntervalSec" type="number" min="30" max="3600" value="${Number(s.urlTestIntervalSec) || 300}"/>
        </label>
      </div>
    </form>
  `
}

function renderSettingsDNS(s) {
  return `
    <form id="settings-form" class="settings" data-section="dns">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">DNS</div>
        <p class="muted rules-hint">DNS — это «телефонная книга» интернета: по имени сайта находится его адрес. Обычно достаточно значений по умолчанию.</p>
        <label class="field">Режим
          <select name="dnsEnhancedMode">
            <option value="fake-ip" ${s.dnsEnhancedMode !== 'redir-host' ? 'selected' : ''}>Быстрый (рекомендуется)</option>
            <option value="redir-host" ${s.dnsEnhancedMode === 'redir-host' ? 'selected' : ''}>Совместимый (если что-то не открывается)</option>
          </select>
        </label>
        <label class="field">Основные DNS-серверы
          <input name="dnsNameservers" value="${escapeHtml(s.dnsNameservers || '8.8.8.8, 1.1.1.1')}" placeholder="8.8.8.8, 1.1.1.1"/>
        </label>
        <label class="field">Запасные DNS-серверы
          <input name="dnsFallbacks" value="${escapeHtml(s.dnsFallbacks || 'tls://1.1.1.1:853')}" placeholder="1.1.1.1"/>
        </label>
        <label class="field">Служебный диапазон адресов
          <input name="dnsFakeIpRange" value="${escapeHtml(s.dnsFakeIpRange || '198.18.0.1/16')}"/>
        </label>
      </div>
    </form>
  `
}

function renderSettingsInbound(s) {
  const stack = s.tunStack || 'system'
  return `
    <form id="settings-form" class="settings" data-section="inbound">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">Подключение</div>
        <label class="field">Локальный порт
          <input name="mixedPort" type="number" min="1024" max="65535" value="${Number(s.mixedPort) || 7890}"/>
        </label>
        ${switchRow('allowLan', s.allowLan, 'Доступ из домашней сети', 'Другие устройства в Wi‑Fi смогут использовать этот VPN')}
        ${switchRow('tun', s.tun, 'Весь трафик системы', 'Через VPN идут все программы, а не только браузер')}
        <label class="field">Способ перехвата трафика
          <select name="tunStack">
            <option value="system" ${stack === 'system' ? 'selected' : ''}>Системный (обычно лучше)</option>
            <option value="gvisor" ${stack === 'gvisor' ? 'selected' : ''}>Изолированный</option>
            <option value="mixed" ${stack === 'mixed' ? 'selected' : ''}>Смешанный</option>
          </select>
        </label>
        ${switchRow('useSystemProxy', s.useSystemProxy, 'Системный прокси', 'Если «весь трафик» выключен — настроить прокси Windows')}
      </div>
    </form>
  `
}

function renderSettingsTricks(s) {
  return `
    <form id="settings-form" class="settings" data-section="tricks">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">Ускорение и обход</div>
        <p class="settings-note">Дополнительные настройки соединения. В большинстве случаев достаточно оставить включёнными.</p>
        ${switchRow('sniffer', s.sniffer !== false, 'Умное определение сайтов', 'Лучше понимать, куда идёт трафик')}
        ${switchRow('tcpConcurrent', s.tcpConcurrent !== false, 'Параллельные соединения', 'Быстрее открывать сайты при задержках')}
        ${switchRow('unifiedDelay', s.unifiedDelay !== false, 'Точный замер пинга', 'Аккуратнее показывать задержку серверов')}
      </div>
    </form>
  `
}

function renderSettingsWARP(s) {
  const mode = s.warpMode === 'proxy-via-warp' ? 'proxy-via-warp' : 'via-proxy'
  const hasKey = !!(s.warpPrivateKey && String(s.warpPrivateKey).trim())
  return `
    <form id="settings-form" class="settings" data-section="warp">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">WARP</div>
        <p class="settings-note">Дополнительная защита от Cloudflare поверх вашего VPN. Это не отдельный сервер в списке — включается вместе с подключением.</p>
        ${switchRow('warpEnabled', s.warpEnabled, 'Включить WARP', 'Добавить защиту Cloudflare при работе VPN')}
        <button class="ghost" type="button" data-action="generate-warp" style="width:100%;margin:4px 0 12px">${hasKey ? 'Конфигурация WARP готова ✓' : 'Получить конфигурацию WARP'}</button>
        <label class="field">Как сочетать с VPN
          <select name="warpMode">
            <option value="via-proxy" ${mode === 'via-proxy' ? 'selected' : ''}>Сначала VPN, затем WARP</option>
            <option value="proxy-via-warp" ${mode === 'proxy-via-warp' ? 'selected' : ''}>Сначала WARP, затем VPN</option>
          </select>
        </label>
        <label class="field">Лицензионный ключ (необязательно)
          <input name="warpLicenseKey" value="${escapeHtml(s.warpLicenseKey || '')}" placeholder="Если есть — вставьте сюда" autocomplete="off"/>
        </label>
        <label class="field">Предпочтительный адрес
          <input name="warpCleanIp" value="${escapeHtml(s.warpCleanIp || 'auto')}" placeholder="auto"/>
        </label>
        <label class="field">Порт
          <input name="warpPort" type="number" min="0" max="65535" value="${Number(s.warpPort || 0)}"/>
        </label>
        <label class="field">Количество шума
          <input name="warpNoiseCount" value="${escapeHtml(s.warpNoiseCount || '1-3')}"/>
        </label>
        <label class="field">Режим шума
          <input name="warpNoiseMode" value="${escapeHtml(s.warpNoiseMode || 'm4')}"/>
        </label>
        <label class="field">Размер шума
          <input name="warpNoiseSize" value="${escapeHtml(s.warpNoiseSize || '10-30')}"/>
        </label>
        <label class="field">Задержка шума
          <input name="warpNoiseDelay" value="${escapeHtml(s.warpNoiseDelay || '10-30')}"/>
        </label>
        <details class="warp-advanced">
          <summary>Расширенные параметры</summary>
          <label class="field">Закрытый ключ
            <input name="warpPrivateKey" value="${escapeHtml(s.warpPrivateKey || '')}" placeholder="заполняется автоматически" autocomplete="off"/>
          </label>
          <label class="field">Локальный адрес
            <input name="warpLocalAddress" value="${escapeHtml(s.warpLocalAddress || '172.16.0.2/32')}"/>
          </label>
          <label class="field">Сервер подключения
            <input name="warpEndpoint" value="${escapeHtml(s.warpEndpoint || 'engage.cloudflareclient.com:2408')}"/>
          </label>
          <label class="field">Открытый ключ Cloudflare
            <input name="warpPublicKey" value="${escapeHtml(s.warpPublicKey || 'bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=')}"/>
          </label>
        </details>
      </div>
    </form>
  `
}

function renderSettings(state) {
  const s = state.settings || {}
  const v = state.status?.appVersion || 'dev'
  switch (state.settingsSection) {
    case 'general': return renderSettingsGeneral(s, v)
    case 'routing': return renderSettingsRouting(s)
    case 'dns': return renderSettingsDNS(s)
    case 'inbound': return renderSettingsInbound(s)
    case 'tricks': return renderSettingsTricks(s)
    case 'warp': return renderSettingsWARP(s)
    default: return renderSettingsMenu(state)
  }
}

function collectSettingsPatch(section, fd, prev) {
  const on = (name) => fd.get(name) === 'on'
  const base = { ...prev }
  switch (section) {
    case 'general':
      return {
        ...base,
        autostart: on('autostart'),
        closeToTray: on('closeToTray'),
        autoReconnect: on('autoReconnect'),
        killSwitch: on('killSwitch'),
        dnsLeakProtection: on('dnsLeakProtection'),
        autoUpdateSubscriptions: on('autoUpdateSubscriptions'),
        subscriptionIntervalMin: Number(fd.get('subscriptionIntervalMin') || 360),
        healthIntervalSec: Number(fd.get('healthIntervalSec') || 30),
        logLevel: String(fd.get('logLevel') || 'info'),
      }
    case 'routing':
      return {
        ...base,
        mode: String(fd.get('mode') || 'rule'),
        ipv6: on('ipv6'),
        bypassLan: on('bypassLan'),
        bypassGeoip: String(fd.get('bypassGeoip') || 'CN'),
        routeWhitelist: on('routeWhitelist'),
        routeBlock: String(fd.get('routeBlock') || ''),
        routeDirect: String(fd.get('routeDirect') || ''),
        routeProxy: String(fd.get('routeProxy') || ''),
        appRouteMode: String(fd.get('appRouteMode') || 'off'),
        appRouteList: String(fd.get('appRouteList') || ''),
        pingMethod: String(fd.get('pingMethod') || 'proxy-http-get'),
        urlTestPreset: String(fd.get('urlTestPreset') || 'gstatic'),
        urlTestUrl: String(fd.get('urlTestUrl') || ''),
        urlTestIntervalSec: Number(fd.get('urlTestIntervalSec') || 300),
      }
    case 'dns':
      return {
        ...base,
        dnsEnhancedMode: String(fd.get('dnsEnhancedMode') || 'fake-ip'),
        dnsNameservers: String(fd.get('dnsNameservers') || ''),
        dnsFallbacks: String(fd.get('dnsFallbacks') || ''),
        dnsFakeIpRange: String(fd.get('dnsFakeIpRange') || ''),
      }
    case 'inbound':
      return {
        ...base,
        mixedPort: Number(fd.get('mixedPort') || 7890),
        allowLan: on('allowLan'),
        tun: on('tun'),
        tunStack: String(fd.get('tunStack') || 'system'),
        useSystemProxy: on('useSystemProxy'),
      }
    case 'tricks':
      return {
        ...base,
        sniffer: on('sniffer'),
        tcpConcurrent: on('tcpConcurrent'),
        unifiedDelay: on('unifiedDelay'),
      }
    case 'warp':
      return {
        ...base,
        warpEnabled: on('warpEnabled'),
        warpMode: String(fd.get('warpMode') || 'via-proxy'),
        warpLicenseKey: String(fd.get('warpLicenseKey') || ''),
        warpCleanIp: String(fd.get('warpCleanIp') || 'auto'),
        warpPort: Number(fd.get('warpPort') || 0),
        warpNoiseCount: String(fd.get('warpNoiseCount') || '1-3'),
        warpNoiseMode: String(fd.get('warpNoiseMode') || 'm4'),
        warpNoiseSize: String(fd.get('warpNoiseSize') || '10-30'),
        warpNoiseDelay: String(fd.get('warpNoiseDelay') || '10-30'),
        warpPrivateKey: String(fd.get('warpPrivateKey') || ''),
        warpLocalAddress: String(fd.get('warpLocalAddress') || ''),
        warpEndpoint: String(fd.get('warpEndpoint') || ''),
        warpPublicKey: String(fd.get('warpPublicKey') || ''),
      }
    default:
      return base
  }
}

const LOG_LEVELS = ['trace', 'debug', 'info', 'warn', 'error']
const LOG_LEVEL_LABELS = {
  trace: 'Все детали',
  debug: 'Подробные',
  info: 'Обычные',
  warn: 'Предупреждения',
  error: 'Ошибки',
}

function parseLogLine(line) {
  const raw = String(line || '')
  const trimmed = raw.trim()
  if (!trimmed) return { raw, level: '', time: null, isHeader: false, empty: true }
  if (/^——\s*.+\s*——$/.test(trimmed) || /^---+\s/.test(trimmed)) {
    return { raw, level: '', time: null, isHeader: true, empty: false }
  }

  let level = ''
  const bracket = trimmed.match(/\[(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|ERR|FATAL|DBG|INF|WRN)\]/i)
  if (bracket) level = normalizeLogLevel(bracket[1])
  if (!level) {
    const key = trimmed.match(/(?:^|[\s"])level[=:]["']?(trace|debug|info|warn(?:ing)?|error|err|fatal)["']?/i)
    if (key) level = normalizeLogLevel(key[1])
  }
  if (!level) {
    const prefix = trimmed.match(/^(TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|ERR|FATAL)\b/i)
    if (prefix) level = normalizeLogLevel(prefix[1])
  }

  let time = null
  const iso = trimmed.match(/(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)/)
  if (iso) {
    const ms = Date.parse(iso[1].replace(' ', 'T'))
    if (!Number.isNaN(ms)) time = ms
  }
  if (time == null) {
    const quoted = trimmed.match(/time=["']([^"']+)["']/i)
    if (quoted) {
      const ms = Date.parse(quoted[1])
      if (!Number.isNaN(ms)) time = ms
    }
  }

  return { raw, level, time, isHeader: false, empty: false }
}

function normalizeLogLevel(v) {
  const s = String(v || '').toLowerCase()
  if (s === 'warning' || s === 'wrn') return 'warn'
  if (s === 'err') return 'error'
  if (s === 'inf') return 'info'
  if (s === 'dbg') return 'debug'
  if (s === 'fatal') return 'error'
  return s
}

function logTimeCutoff(preset) {
  const now = Date.now()
  switch (String(preset || 'all')) {
    case '5m': return now - 5 * 60 * 1000
    case '15m': return now - 15 * 60 * 1000
    case '1h': return now - 60 * 60 * 1000
    case '6h': return now - 6 * 60 * 60 * 1000
    case '24h': return now - 24 * 60 * 60 * 1000
    default: return 0
  }
}

function filterLogsText(raw, { query = '', level = 'all', time = 'all' } = {}) {
  const text = String(raw || '')
  if (!text) return ''
  const q = String(query || '').trim().toLowerCase()
  const wantLevel = normalizeLogLevel(level)
  const cutoff = logTimeCutoff(time)
  const useLevel = wantLevel && wantLevel !== 'all'
  const useTime = cutoff > 0
  if (!q && !useLevel && !useTime) return text

  const lines = text.split('\n')
  const out = []
  for (const line of lines) {
    const parsed = parseLogLine(line)
    if (parsed.empty) {
      if (out.length && out[out.length - 1] !== '') out.push('')
      continue
    }
    if (parsed.isHeader) {
      out.push(parsed.raw)
      continue
    }
    if (q && !parsed.raw.toLowerCase().includes(q)) continue
    if (useLevel) {
      if (!parsed.level || parsed.level !== wantLevel) continue
    }
    if (useTime) {
      // Lines without a timestamp stay visible unless a level/query already excluded them.
      if (parsed.time != null && parsed.time < cutoff) continue
    }
    out.push(parsed.raw)
  }
  while (out.length && out[out.length - 1] === '') out.pop()
  return out.join('\n')
}

function filteredLogs(state) {
  const raw = state.logs || ''
  if (!raw) return 'Загрузка…'
  const filtered = filterLogsText(raw, {
    query: state.logQuery,
    level: state.logLevel,
    time: state.logTime,
  })
  if (!filtered.trim()) return 'Нет записей по выбранным фильтрам'
  return filtered
}

function renderConnections(state) {
  const { status, connections, connQuery } = state
  const connected = status.state === 'connected'
  if (!connected) {
    return `
      <div class="panel connections-panel">
        <div class="empty">
          <p>Список соединений доступен после подключения</p>
          <button class="primary" type="button" data-action="go-home">На главную</button>
        </div>
      </div>`
  }

  const list = connections || []
  const q = String(connQuery || '').trim().toLowerCase()
  const filtered = list.filter((c) => {
    if (!q) return true
    const host = String(c.host || '').toLowerCase()
    const rule = String(c.rule || '').toLowerCase()
    const chains = String(c.chains || '').toLowerCase()
    const process = String(c.process || '').toLowerCase()
    return host.includes(q) || rule.includes(q) || chains.includes(q) || process.includes(q)
  })

  const totalUp = list.reduce((sum, c) => sum + Number(c.upload || 0), 0)
  const totalDown = list.reduce((sum, c) => sum + Number(c.download || 0), 0)

  function relativeTime(start) {
    if (!start) return '—'
    const ms = Date.now() - Date.parse(start)
    if (ms < 0) return 'сейчас'
    const s = Math.floor(ms / 1000)
    if (s < 60) return s + 'с'
    const m = Math.floor(s / 60)
    if (m < 60) return m + 'м'
    const h = Math.floor(m / 60)
    return h + 'ч'
  }

  const rows = filtered.map((c) => {
    const host = c.host || c.destIP || '—'
    const rule = c.rule || '—'
    const chains = c.chains || '—'
    const up = fmtBytes(c.upload || 0)
    const down = fmtBytes(c.download || 0)
    const network = c.network || ''
    const type = c.type || ''
    const netType = [network, type].filter(Boolean).join(' / ') || '—'
    const elapsed = relativeTime(c.start)
    return `
      <div class="conn-item">
        <div class="conn-main">
          <div class="conn-host">${escapeHtml(host)}</div>
          <div class="conn-meta">
            <span>${escapeHtml(netType)}</span>
            <span>·</span>
            <span>Правило: ${escapeHtml(rule)}</span>
            ${chains !== '—' ? `<span>·</span><span>Цепь: ${escapeHtml(chains)}</span>` : ''}
          </div>
        </div>
        <div class="conn-stats">
          <div class="conn-traffic">
            <span class="dir up">↑ ${up}</span>
            <span class="dir down">↓ ${down}</span>
            <span class="conn-time">${elapsed}</span>
          </div>
          <button class="icon-ghost danger" type="button" data-close-conn="${escapeHtml(c.id)}" title="Закрыть">${icons.close}</button>
        </div>
      </div>`
  }).join('')

  return `
    <div class="panel connections-panel">
      <div class="connections-head">
        <div class="connections-summary">
          <div class="summary-stat">
            <span class="summary-label">Соединений</span>
            <strong>${list.length}</strong>
          </div>
          <div class="summary-stat">
            <span class="summary-label">Всего загружено</span>
            <strong>${fmtBytes(totalDown)}</strong>
          </div>
          <div class="summary-stat">
            <span class="summary-label">Всего отправлено</span>
            <strong>${fmtBytes(totalUp)}</strong>
          </div>
        </div>
        <div class="search-box">
          ${icons.search}
          <input id="conn-search" type="search" placeholder="Поиск по хосту, правилу, цепи…" value="${escapeHtml(connQuery || '')}" autocomplete="off"/>
        </div>
      </div>
      <div class="connections-list" id="connections-list">${rows || '<div class="empty">Нет активных соединений</div>'}</div>
    </div>
  `
}

function renderLogs(state) {
  const level = state.logLevel || 'all'
  const time = state.logTime || 'all'
  const query = state.logQuery || ''
  return `
    <div class="panel logs-panel">
      <div class="panel-title">
        <span>Журнал событий</span>
        <button class="ghost compact" type="button" data-action="refresh-logs">Обновить</button>
      </div>
      <div class="logs-filters">
        <label class="logs-filter logs-filter-query">
          <span>Поиск</span>
          <input id="log-query" type="search" placeholder="Текст, имя, узел…" value="${escapeHtml(query)}" autocomplete="off"/>
        </label>
        <label class="logs-filter">
          <span>Уровень</span>
          <select id="log-level">
            <option value="all" ${level === 'all' ? 'selected' : ''}>Все</option>
            ${LOG_LEVELS.map((lvl) => `<option value="${lvl}" ${level === lvl ? 'selected' : ''}>${LOG_LEVEL_LABELS[lvl] || lvl}</option>`).join('')}
          </select>
        </label>
        <label class="logs-filter">
          <span>Время</span>
          <select id="log-time">
            <option value="all" ${time === 'all' ? 'selected' : ''}>Все время</option>
            <option value="5m" ${time === '5m' ? 'selected' : ''}>5 минут</option>
            <option value="15m" ${time === '15m' ? 'selected' : ''}>15 минут</option>
            <option value="1h" ${time === '1h' ? 'selected' : ''}>1 час</option>
            <option value="6h" ${time === '6h' ? 'selected' : ''}>6 часов</option>
            <option value="24h" ${time === '24h' ? 'selected' : ''}>24 часа</option>
          </select>
        </label>
      </div>
      <pre class="logs-view" id="logs-view">${escapeHtml(filteredLogs(state))}</pre>
    </div>
  `
}

function renderView(state) {
  switch (state.view) {
    case 'profiles': return renderProfiles(state)
    case 'nodes': return renderNodes(state)
    case 'connections': return renderConnections(state)
    case 'logs': return renderLogs(state)
    case 'settings': return renderSettings(state)
    default: return renderHome(state)
  }
}

async function boot() {
  if (!root) {
    document.body.innerHTML = '<p style="color:#fff;padding:24px;font-family:sans-serif">UI root missing</p>'
    return
  }

  // Paint shell immediately so a slow/failed backend call never leaves a blank window.
  root.innerHTML = shellHTML()
  const content = document.getElementById('content')
  const toast = document.getElementById('toast')

  // Start with mock; upgrade to Wails bindings when ready (never block click wiring).
  bridge = mock
  state = {
    view: 'home',
    status: { state: 'disconnected', activeProfile: '', mode: 'rule', speedUp: 0, speedDown: 0, appVersion: 'dev' },
    profiles: [],
    settings: {},
    nodes: [],
    currentNode: '',
    connections: [],
    connQuery: '',
    logs: '',
    logQuery: '',
    logLevel: 'all',
    logTime: 'all',
    message: '',
    messageOk: false,
    busy: false,
    lastZip: '',
    delayMap: {},
    pinging: {},
    formDirty: false,
    showImport: false,
    nodeQuery: '',
    nodeSort: 'ping',
    publicIP: '',
    toastTimer: 0,
    settingsSection: '',
    quotaMenuOpen: false,
  }

  function showToast(msg, ok = false) {
    state.message = msg || ''
    state.messageOk = ok
    window.clearTimeout(state.toastTimer)
    if (!state.message) {
      toast.className = 'toast'
      toast.textContent = ''
      return
    }
    toast.textContent = state.message
    toast.className = `toast show${ok ? ' ok' : ''}`
    state.toastTimer = window.setTimeout(() => {
      toast.className = 'toast'
      state.message = ''
    }, 3200)
  }

  function paintChrome() {
    const sectionMeta = settingsMenu.find((x) => x.id === state.settingsSection)
    document.getElementById('view-title').textContent = state.view === 'settings' && sectionMeta
      ? sectionMeta.title
      : (titles[state.view] || 'Мой VPN')
    const connected = state.status.state === 'connected'
    document.getElementById('view-meta').textContent = connected
      ? `Активно · ${modeLabel(state.status.mode)}`
      : (state.status.activeProfile ? 'Готово к подключению' : 'Нужен профиль')
    document.querySelectorAll('.nav-btn').forEach((btn) => {
      btn.classList.toggle('active', btn.getAttribute('data-view') === state.view)
    })
    document.getElementById('top-actions').innerHTML = topActionsFor(state.view)
    document.getElementById('speed-down').textContent = fmtRate(state.status.speedDown)
    document.getElementById('speed-up').textContent = fmtRate(state.status.speedUp)
  }

  function patchHomeLive() {
    if (state.view !== 'home') return
    const connected = state.status.state === 'connected'
    const q = state.status.subscriptionQuota || {}
    const used = Number(q.used || 0)
    const total = Number(q.total || 0)
    const pct = quotaPercent(used, total)
    const node = state.currentNode || state.status.selectedNode || ''
    const nodeInfo = (state.nodes || []).find((n) => n.name === node)
    const delay = nodeInfo?.delay || 0
    const hasProfile = !!state.status.activeProfile

    const power = document.getElementById('power-btn')
    if (power) {
      power.classList.toggle('on', connected)
      power.classList.toggle('off', !connected)
      power.classList.toggle('busy', !!state.busy)
      power.disabled = !hasProfile
    }
    const statusLine = document.getElementById('status-line')
    if (statusLine) {
      statusLine.textContent = connected ? 'Подключено' : hasProfile ? 'Отключено' : 'Добавьте профиль'
      statusLine.classList.toggle('on', connected)
      statusLine.classList.toggle('off', !connected)
    }
    const warpOn = !!(state.status.warpEnabled && connected)
    const warpBadge = document.getElementById('warp-badge')
    if (warpBadge) warpBadge.classList.toggle('show', warpOn)
    const serverCard = content.querySelector('.server-card')
    if (serverCard) serverCard.classList.toggle('warp-on', warpOn)
    const latency = document.getElementById('latency-value')
    if (latency) latency.textContent = delay > 0 ? delay + ' мс' : connected ? '—' : 'ожидание'
    const mode = document.getElementById('mode-value')
    if (mode) mode.textContent = modeLabel(state.status.mode)
    const ipEl = document.getElementById('public-ip-value')
    if (ipEl) {
      const masked = state.publicIP || state.status.publicIP || ''
      ipEl.textContent = masked || (connected ? '...' : '-')
    }
    const profileChip = document.getElementById('active-profile')
    if (profileChip) profileChip.textContent = state.status.activeProfile || 'Нет профиля'
    const quotaText = document.getElementById('quota-text')
    if (quotaText) quotaText.textContent = `${fmtBytes(used)} / ${fmtBytes(total > 0 ? total : -1)}`
    const bar = document.getElementById('quota-bar')
    if (bar) bar.style.width = pct + '%'
    const expire = document.getElementById('quota-expire')
    if (expire) expire.textContent = expireLabel(q.expireUnix)
    const quotaCard = content.querySelector('.quota-card')
    if (quotaCard) {
      quotaCard.classList.toggle('is-expired', !!q.expired)
      quotaCard.classList.toggle('menu-open', !!state.quotaMenuOpen)
    }
    const quotaMenu = document.getElementById('quota-profile-menu')
    if (quotaMenu) quotaMenu.classList.toggle('open', !!state.quotaMenuOpen)
    const serverTitle = document.getElementById('server-title')
    if (serverTitle) {
      if (warpOn && node) {
        serverTitle.innerHTML = `<span class="node-title-row">${icons.lock}<span class="warp-chain">WARP →</span>${formatNodeTitleHTML(node)}</span>`
      } else {
        serverTitle.innerHTML = node ? formatNodeTitleHTML(node) : 'Автовыбор'
      }
    }
    const serverSub = document.getElementById('server-sub')
    if (serverSub) {
      if (node === 'AUTO' || String(nodeInfo?.type || '').toLowerCase() === 'urltest') {
        serverSub.textContent = nodeSubtitle(nodeInfo || { name: 'AUTO', type: 'URLTest', now: nodeInfo?.now })
      } else {
        serverSub.textContent = `${nodeKind(nodeInfo?.type)} · нажмите, чтобы выбрать`
      }
    }
    const serverFlag = document.getElementById('server-flag')
    if (serverFlag) {
      serverFlag.innerHTML = (node ? nodeBadgeHTML(node) : 'AU') + (warpOn ? `<span class="warp-cloud" title="Cloudflare WARP">${icons.cloudflare}</span>` : '')
    }
  }

  function patchNodeRow(name) {
    if (state.view !== 'nodes') return
    const row = Array.from(content.querySelectorAll('.node-item')).find(
      (el) => el.getAttribute('data-node') === name
    )
    if (!row) return
    const node = (state.nodes || []).find((n) => n.name === name) || { name, delay: state.delayMap[name] || 0 }
    const delay = state.delayMap[name] ?? node.delay ?? 0
    const pinging = !!state.pinging[name]
    row.classList.toggle('is-pinging', pinging)
    row.classList.toggle('active', state.currentNode === name)
    const badge = row.querySelector('.badge')
    if (badge) {
      const tmp = document.createElement('div')
      tmp.innerHTML = delayBadgeHTML(delay, pinging)
      badge.replaceWith(tmp.firstElementChild)
    }
    const sub = row.querySelector('.node-sub')
    if (sub && node) sub.textContent = nodeSubtitle({ ...node, delay })
    const title = row.querySelector('.node-title')
    if (title && node) title.innerHTML = formatNodeTitleHTML(node.name)
  }

  function onNodePingEvent(payload) {
    const name = payload?.name
    if (!name) return
    if (payload.phase === 'start') {
      state.pinging[name] = true
      patchNodeRow(name)
      return
    }
    if (payload.phase === 'done') {
      delete state.pinging[name]
      const delay = Number(payload.delay || 0)
      state.delayMap[name] = delay
      state.nodes = (state.nodes || []).map((n) => (n.name === name ? { ...n, delay } : n))
      patchNodeRow(name)
      paintLive()
    }
  }

  function bindPingEvents() {
    const rt = window.runtime
    if (!rt?.EventsOn || bindPingEvents.done) return
    bindPingEvents.done = true
    rt.EventsOn('node:ping', onNodePingEvent)
    rt.EventsOn('deeplink', (payload) => {
      const ok = !!payload?.ok
      const msg = payload?.message || (ok ? 'Ссылка обработана' : 'Не удалось открыть ссылку')
      showToast(msg, ok)
      if (ok && (payload?.kind === 'import' || payload?.kind === 'connect' || payload?.kind === 'disconnect' || payload?.kind === 'toggle')) {
        refresh({ force: true, soft: false }).catch(() => {})
      }
    })
  }

  async function refreshPublicIP({ force = false } = {}) {
    if (!bridge.GetPublicIP) return
    const connected = state.status.state === 'connected'
    if (!connected) {
      state.publicIP = ''
      if (state.status) state.status.publicIP = ''
      const ipEl = document.getElementById('public-ip-value')
      if (ipEl) ipEl.textContent = '-'
      return
    }
    if (!force && state.publicIP && state.status.publicIPReady) {
      const ipEl = document.getElementById('public-ip-value')
      if (ipEl) ipEl.textContent = state.publicIP
      return
    }
    try {
      const res = await bridge.GetPublicIP()
      const masked = res?.masked || '-'
      state.publicIP = masked
      if (state.status) {
        state.status.publicIP = masked
        state.status.publicIPReady = !!res?.ip
      }
      const ipEl = document.getElementById('public-ip-value')
      if (ipEl) ipEl.textContent = masked
    } catch (_) {
      const ipEl = document.getElementById('public-ip-value')
      if (ipEl && !state.publicIP) ipEl.textContent = '-'
    }
  }

  function paintLive() {
    paintChrome()
    patchHomeLive()
    if (state.view === 'logs') {
      const el = document.getElementById('logs-view')
      if (el && state.logs) {
        const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40
        el.textContent = filteredLogs(state)
        if (atBottom) el.scrollTop = el.scrollHeight
      }
    }
  }

  async function loadLogs() {
    try {
      state.logs = (await bridge.GetLogsTail(250)) || ''
    } catch (_) {
      state.logs = state.logs || 'Не удалось прочитать лог'
    }
  }

  function paintContent({ force = false } = {}) {
    if (!force && state.formDirty) {
      paintLive()
      return
    }
    content.innerHTML = renderView(state)
    state.formDirty = false
    enhanceSelects(content)
    paintChrome()
    if (state.message) showToast(state.message, state.messageOk)
  }

  async function loadData({ soft = false } = {}) {
    if (soft) {
      state.status = await bridge.GetStatus()
      if (state.status.publicIP) state.publicIP = state.status.publicIP
      if (state.view === 'logs') {
        await loadLogs()
      } else if (state.view === 'connections' && state.status.state === 'connected') {
        try {
          state.connections = (await bridge.ListConnections()) || []
        } catch (_) {}
      } else if (state.view === 'nodes' && state.status.state === 'connected') {
        try {
          const nodes = await bridge.ListNodes()
          const currentNode = (await bridge.CurrentNode()) || state.status.selectedNode || ''
          state.nodes = (nodes || []).map((n) => ({
            ...n,
            delay: state.delayMap[n.name] || n.delay || 0,
          }))
          state.currentNode = currentNode
        } catch (_) {}
      } else if (state.view === 'home' && state.status.state === 'connected' && !state.nodes.length) {
        try {
          const nodes = await bridge.ListNodes()
          state.nodes = (nodes || []).map((n) => ({
            ...n,
            delay: state.delayMap[n.name] || n.delay || 0,
          }))
          state.currentNode = (await bridge.CurrentNode()) || state.status.selectedNode || ''
        } catch (_) {}
      } else {
        state.currentNode = state.status.selectedNode || state.currentNode || ''
      }
      if (state.view === 'home') refreshPublicIP().catch(() => {})
      return
    }

    const [status, profiles, settings] = await Promise.all([
      bridge.GetStatus(),
      bridge.ListProfiles(),
      bridge.GetSettings(),
    ])
    state.status = status
    if (status.publicIP) state.publicIP = status.publicIP
    state.profiles = profiles || []
    if (!(state.formDirty && state.view === 'settings')) {
      state.settings = settings || {}
    }

    let nodes = []
    let currentNode = status.selectedNode || ''
    if (status.state === 'connected') {
      try {
        nodes = await bridge.ListNodes()
        currentNode = (await bridge.CurrentNode()) || currentNode
        nodes = (nodes || []).map((n) => ({
          ...n,
          delay: state.delayMap[n.name] || n.delay || 0,
        }))
      } catch (_) {}
    }
    state.nodes = nodes
    state.currentNode = currentNode
    if (state.view === 'logs') await loadLogs()
    if (state.view === 'connections' && status.state === 'connected') {
      try {
        state.connections = (await bridge.ListConnections()) || []
      } catch (_) {}
    }
    if (state.view === 'home') refreshPublicIP({ force: true }).catch(() => {})
  }

  async function refresh({ force = false, soft = true } = {}) {
    await loadData({ soft })
    if (force) paintContent({ force: true })
    else paintLive()
  }

  async function setView(view) {
    if (!titles[view]) view = 'home'
    // Flush pending settings before leaving the page.
    if (state.view === 'settings' && settingsSaveTimer) {
      window.clearTimeout(settingsSaveTimer)
      settingsSaveTimer = 0
      const form = document.getElementById('settings-form')
      if (form) {
        try { await persistSettingsForm(form) } catch (_) {}
      }
    }
    state.view = view
    state.formDirty = false
    if (view !== 'settings') state.settingsSection = ''
    await loadData({ soft: false })
    paintContent({ force: true })
  }

  let settingsSaveTimer = 0
  let settingsSaveSeq = 0

  async function persistSettingsForm(form) {
    if (!form || !bridge?.SaveSettings) return
    const fd = new FormData(form)
    const section = form.getAttribute('data-section') || state.settingsSection
    const next = collectSettingsPatch(section, fd, state.settings)
    const seq = ++settingsSaveSeq
    try {
      await bridge.SaveSettings(next)
      if (seq !== settingsSaveSeq) return
      state.settings = next
      // Keep formDirty so soft refresh doesn't wipe caret / open details.
      state.formDirty = true
      await refresh({ soft: true })
    } catch (err) {
      if (seq !== settingsSaveSeq) return
      showToast(friendlyError(err), false)
    }
  }

  function scheduleSettingsAutosave(form) {
    window.clearTimeout(settingsSaveTimer)
    settingsSaveTimer = window.setTimeout(() => {
      settingsSaveTimer = 0
      persistSettingsForm(form).catch(() => {})
    }, 900)
  }

  // Paint + wire clicks before any backend await (hang must not block UI).
  bindPingEvents()
  paintContent({ force: true })

  root.addEventListener('input', (e) => {
    if (e.target.id === 'node-search') {
      state.nodeQuery = e.target.value
      // Re-render nodes list only, keep focus/caret by not full wipe if possible
      const list = content.querySelector('.nodes-list')
      if (list) {
        const tmp = document.createElement('div')
        tmp.innerHTML = renderNodes(state)
        const next = tmp.querySelector('.nodes-list')
        if (next) list.replaceWith(next)
      }
      return
    }
    if (e.target.id === 'conn-search') {
      state.connQuery = e.target.value
      const list = content.querySelector('.connections-list')
      if (list) {
        const tmp = document.createElement('div')
        tmp.innerHTML = renderConnections(state)
        const next = tmp.querySelector('.connections-list')
        if (next) list.replaceWith(next)
      }
      return
    }
    if (e.target.id === 'log-query') {
      state.logQuery = e.target.value
      const el = document.getElementById('logs-view')
      if (el) el.textContent = filteredLogs(state)
      return
    }
    const form = e.target.closest('#settings-form')
    if (form) {
      state.formDirty = true
      scheduleSettingsAutosave(form)
    }
  })
  root.addEventListener('change', (e) => {
    if (e.target.id === 'log-level') {
      state.logLevel = e.target.value || 'all'
      const el = document.getElementById('logs-view')
      if (el) el.textContent = filteredLogs(state)
      return
    }
    if (e.target.id === 'log-time') {
      state.logTime = e.target.value || 'all'
      const el = document.getElementById('logs-view')
      if (el) el.textContent = filteredLogs(state)
      return
    }
    const form = e.target.closest('#settings-form')
    if (form) {
      state.formDirty = true
      window.clearTimeout(settingsSaveTimer)
      settingsSaveTimer = 0
      persistSettingsForm(form).catch(() => {})
    }
  })

  root.addEventListener('click', async (e) => {
    if (!e.target.closest('.cselect')) {
      root.querySelectorAll('.cselect.open').forEach((el) => el.classList.remove('open'))
    }
    if (!e.target.closest('.quota-profile-wrap')) {
      if (state.quotaMenuOpen) {
        state.quotaMenuOpen = false
        const menu = document.getElementById('quota-profile-menu')
        const card = content.querySelector('.quota-card')
        if (menu) menu.classList.remove('open')
        if (card) card.classList.remove('menu-open')
      }
    }

    const nav = e.target.closest('[data-view]')
    if (nav) {
      state.quotaMenuOpen = false
      await setView(nav.getAttribute('data-view'))
      return
    }

    const settingsSectionBtn = e.target.closest('[data-settings-section]')
    if (settingsSectionBtn && state.view === 'settings') {
      if (settingsSaveTimer) {
        window.clearTimeout(settingsSaveTimer)
        settingsSaveTimer = 0
        const form = document.getElementById('settings-form')
        if (form) {
          try { await persistSettingsForm(form) } catch (_) {}
        }
      }
      state.settingsSection = settingsSectionBtn.getAttribute('data-settings-section') || ''
      state.formDirty = false
      paintContent({ force: true })
      return
    }

    const appRemove = e.target.closest('[data-app-remove]')
    if (appRemove) {
      e.preventDefault()
      removeAppFromRouteList(appRemove.getAttribute('data-app-remove') || '')
      return
    }
    const appAdd = e.target.closest('[data-app-add]')
    if (appAdd) {
      e.preventDefault()
      addAppToRouteList(appAdd.getAttribute('data-app-add') || '')
      appAdd.classList.add('on')
      return
    }
    const actionBtn = e.target.closest('[data-action]')
    if (actionBtn) {
      const act = actionBtn.getAttribute('data-action')
      if (act === 'pick-apps') {
        e.preventDefault()
        await openAppPicker()
        return
      }
      if (act === 'close-app-picker') {
        e.preventDefault()
        const picker = document.getElementById('app-picker')
        if (picker) picker.classList.add('hidden')
        return
      }
      if (act === 'add-app-manual') {
        e.preventDefault()
        const name = window.prompt('Имя программы (например Telegram.exe):', '')
        if (name) addAppToRouteList(name)
        return
      }
    }

    const sortBtn = e.target.closest('[data-node-sort]')
    if (sortBtn && state.view === 'nodes') {
      const next = sortBtn.getAttribute('data-node-sort') === 'name' ? 'name' : 'ping'
      if (state.nodeSort !== next) {
        state.nodeSort = next
        paintContent({ force: true })
        const search = document.getElementById('node-search')
        if (search) {
          const val = search.value
          search.focus()
          search.setSelectionRange(val.length, val.length)
        }
      }
      return
    }

    const action = e.target.closest('[data-action]')?.getAttribute('data-action')
    const profile = e.target.closest('[data-profile]')?.getAttribute('data-profile')
    const syncOne = e.target.closest('[data-sync]')?.getAttribute('data-sync')
    const remove = e.target.closest('[data-remove]')?.getAttribute('data-remove')
    const node = e.target.closest('[data-node]')?.getAttribute('data-node')
    const closeConn = e.target.closest('[data-close-conn]')?.getAttribute('data-close-conn')

    try {
      if (closeConn) {
        await bridge.CloseConnection(closeConn)
        showToast('Соединение закрыто', true)
        await refresh({ force: true })
        return
      }
      if (action === 'close-all-connections') {
        if (!confirm('Закрыть все активные соединения?')) return
        await bridge.CloseAllConnections()
        showToast('Все соединения закрыты', true)
        await refresh({ force: true })
        return
      }
      if (action === 'toggle') {
        if (!state.status.activeProfile) {
          showToast('Сначала добавьте профиль', false)
          await setView('profiles')
          return
        }
        const connecting = state.status.state !== 'connected'
        state.busy = true
        paintLive()
        if (connecting) {
          showToast('Подключаю, обновляю подписки и проверяю серверы…', true)
          state.pinging = {}
        }
        await bridge.ToggleConnect()
        state.busy = false
        state.pinging = {}
        state.publicIP = ''
        await refresh({ force: true })
        refreshPublicIP({ force: true }).catch(() => {})
        showToast(state.status.state === 'connected' ? 'Подключено' : 'Отключено', true)
        return
      }
      if (action === 'generate-warp') {
        showToast('Генерирую WARP…', true)
        const res = await bridge.GenerateWARPConfig()
        if (res?.privateKey) {
          state.settings = { ...(state.settings || {}), warpPrivateKey: res.privateKey, warpLocalAddress: res.localAddress || state.settings.warpLocalAddress }
        }
        showToast('Конфиг WARP готов', true)
        await refresh({ force: true, soft: false })
        if (state.view === 'settings') paintContent({ force: true })
        return
      }
      if (action === 'go-home') { await setView('home'); return }
      if (action === 'go-nodes') { await setView('nodes'); return }
      if (action === 'go-profiles') {
        state.showImport = true
        await setView('profiles')
        return
      }
      if (action === 'toggle-import') {
        state.showImport = !state.showImport
        paintContent({ force: true })
        return
      }
      if (action === 'quit') { await bridge.QuitApp?.(); return }
      if (action === 'refresh-logs') {
        await loadLogs()
        paintContent({ force: true })
        return
      }
      if (action === 'toggle-quota-menu') {
        state.quotaMenuOpen = !state.quotaMenuOpen
        const menu = document.getElementById('quota-profile-menu')
        const card = content.querySelector('.quota-card')
        if (menu) menu.classList.toggle('open', state.quotaMenuOpen)
        if (card) card.classList.toggle('menu-open', state.quotaMenuOpen)
        return
      }
      if (action === 'sync-active') {
        const name = state.status.activeProfile
        const profile = (state.profiles || []).find((p) => p.name === name)
        showToast('Обновляю подписку…', true)
        if (profile?.subscriptionURL) {
          await bridge.SyncSubscription(name)
          showToast('Подписка обновлена', true)
        } else {
          const n = await bridge.SyncAllSubscriptions()
          showToast(n ? `Обновлено: ${n}` : 'Нет подписок для обновления', !!n)
        }
        state.quotaMenuOpen = false
        await refresh({ force: true })
        return
      }
      if (action === 'sync-all') {
        showToast('Обновляю подписки…', true)
        const n = await bridge.SyncAllSubscriptions()
        showToast(n ? `Обновлено: ${n}` : 'Нечего обновлять', true)
        await refresh({ force: true })
        return
      }
      if (action === 'test-all') {
        showToast('Проверяю серверы…', true)
        state.pinging = {}
        if (state.view === 'nodes') paintContent({ force: true })
        const results = await bridge.TestAllNodes()
        Object.keys(state.delayMap).forEach((k) => delete state.delayMap[k])
        for (const r of results || []) state.delayMap[r.name] = r.delay || 0
        state.pinging = {}
        const ok = (results || []).filter((r) => r.delay > 0).length
        showToast(`Доступны ${ok} из ${(results || []).length}`, true)
        await refresh({ force: true })
        return
      }
      if (action === 'update') {
        showToast('Проверяю обновления…', true)
        const res = await bridge.CheckForUpdate()
        if (!res?.available) {
          showToast(res?.reason || 'У вас актуальная версия', true)
          return
        }
        const label = res.name || res.tag || 'новая версия'
        const size = Number(res.size || 0)
        const sizeHint = size > 0 ? `\nРазмер: ${Math.round(size / (1024 * 1024))} МБ` : ''
        if (!window.confirm(`Доступно обновление: ${label}${sizeHint}\n\nСкачать и установить сейчас? Клиент перезапустится.`)) {
          showToast('Обновление отменено', true)
          return
        }
        showToast('Скачиваю обновление…', true)
        state.lastZip = await bridge.DownloadUpdate()
        showToast('Устанавливаю обновление…', true)
        await bridge.ApplyUpdate(state.lastZip)
        showToast('Обновление установлено, перезапуск…', true)
        return
      }
      if (profile) {
        state.quotaMenuOpen = false
        await bridge.SetActiveProfile(profile)
        showToast(`Выбран профиль «${profile}»`, true)
        await refresh({ force: true })
        return
      }
      if (syncOne) {
        showToast('Обновляю…', true)
        await bridge.SyncSubscription(syncOne)
        showToast('Подписка обновлена', true)
        await refresh({ force: true })
        return
      }
      if (remove) {
        if (!confirm(`Удалить профиль «${remove}»?`)) return
        await bridge.RemoveProfile(remove)
        showToast('Профиль удалён', true)
        await refresh({ force: true })
        return
      }
      if (node) {
        await bridge.SelectNode(node)
        state.publicIP = ''
        if (node === 'AUTO') {
          showToast('Выбираю лучший сервер…', true)
          state.pinging.AUTO = true
        } else {
          showToast('Сервер выбран', true)
        }
        await setView('home')
        refreshPublicIP({ force: true }).catch(() => {})
        return
      }
    } catch (err) {
      state.busy = false
      showToast(friendlyError(err), false)
      paintContent({ force: true })
    }
  })

  root.addEventListener('submit', async (e) => {
    if (e.target.id === 'add-form') {
      e.preventDefault()
      const fd = new FormData(e.target)
      const raw = String(fd.get('proxy') || '').trim()
      let name = String(fd.get('name') || '').trim()
      if (!name) {
        name = raw.startsWith('http') ? 'Подписка' : 'Импорт'
      }
      try {
        await bridge.ImportProfileText(name, '', raw)
        state.showImport = false
        state.formDirty = false
        showToast('Профиль добавлен, проверяю пинг…', true)
        await refresh({ force: true })
        try {
          state.pinging = {}
          if (state.view === 'nodes') paintContent({ force: true })
          const results = await bridge.ConnectAndProbe()
          Object.keys(state.delayMap).forEach((k) => delete state.delayMap[k])
          for (const r of results || []) state.delayMap[r.name] = r.delay || 0
          state.pinging = {}
          const ok = (results || []).filter((r) => r.delay > 0).length
          showToast(ok ? `Пинг готов: ${ok} из ${(results || []).length}` : 'Профиль добавлен', true)
          await setView('nodes')
        } catch (probeErr) {
          state.pinging = {}
          showToast('Профиль добавлен. Пинг: ' + friendlyError(probeErr), false)
        }
      } catch (err) {
        showToast(friendlyError(err), false)
      }
      return
    }
    if (e.target.id === 'settings-form') {
      e.preventDefault()
      window.clearTimeout(settingsSaveTimer)
      persistSettingsForm(e.target).catch(() => {})
    }
  })

  // Resolve real Wails bindings, then load data (clicks already work).
  api().then((b) => {
    bridge = b
    return refresh({ force: true, soft: false })
  }).catch((err) => {
    showToast(friendlyError(err), false)
    paintContent({ force: true })
  })
  setInterval(() => { refresh({ soft: true }).catch(() => {}) }, 2000)
  setInterval(async () => {
    if (state.status.state !== 'connected' || !bridge.GetTraffic) return
    try {
      const t = await bridge.GetTraffic()
      if (!t) return
      state.status.speedUp = Number(t.up || 0)
      state.status.speedDown = Number(t.down || 0)
      const downEl = document.getElementById('speed-down')
      const upEl = document.getElementById('speed-up')
      if (downEl) downEl.textContent = fmtRate(state.status.speedDown)
      if (upEl) upEl.textContent = fmtRate(state.status.speedUp)
    } catch (_) {}
  }, 1000)
}

boot().catch((err) => {
  const msg = String(err?.message || err || 'UI boot failed')
  if (root) {
    root.innerHTML = `<div style="padding:28px;color:#edf7ff;font-family:Segoe UI,sans-serif"><h2>Не удалось открыть интерфейс</h2><p style="opacity:.8">${msg}</p></div>`
  } else {
    document.body.textContent = msg
  }
})
