import './style.css'

const root = document.querySelector('#app')

const icons = {
  home: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1v-9.5z"/></svg>`,
  profiles: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M8 7h13M8 12h13M8 17h13"/><path d="M3 7h.01M3 12h.01M3 17h.01"/></svg>`,
  nodes: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9 17 7M7 17l-2.1 2.1"/></svg>`,
  settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9c.3.6.9 1 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></svg>`,
  plus: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 5v14M5 12h14"/></svg>`,
  refresh: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M21 12a9 9 0 1 1-2.6-6.3"/><path d="M21 3v6h-6"/></svg>`,
  chevron: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 6 6 6-6 6"/></svg>`,
  signal: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M2 20h2M7 20v-5M12 20V9M17 20V5M22 20v-9"/></svg>`,
  power: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M12 3v9"/><path d="M7.5 6.2a7.5 7.5 0 1 0 9 0"/></svg>`,
  search: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>`,
  trash: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 7h16M9 7V5h6v2M8 7l1 12h6l1-12"/></svg>`,
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
  async ListNodes() { return [] },
  async CurrentNode() { return '' },
  async SelectNode() { return null },
  async TestAllNodes() { return [] },
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
  if (v >= 1024 * 1024) return (v / (1024 * 1024)).toFixed(1) + ' МБ/с'
  if (v >= 1024) return (v / 1024).toFixed(1) + ' КБ/с'
  return Math.round(v) + ' Б/с'
}

function fmtBytes(n) {
  const v = Number(n || 0)
  if (!Number.isFinite(v) || v < 0) return '∞'
  if (v >= 1024 ** 3) return (v / 1024 ** 3).toFixed(2) + ' ГБ'
  if (v >= 1024 ** 2) return (v / 1024 ** 2).toFixed(1) + ' МБ'
  if (v >= 1024) return (v / 1024).toFixed(1) + ' КБ'
  return Math.round(v) + ' Б'
}

function daysLeft(unix) {
  const n = Number(unix || 0)
  if (!n) return '∞'
  const days = Math.ceil((n * 1000 - Date.now()) / 86400000)
  if (days < 0) return 'истёк'
  return String(days)
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
  return t.toUpperCase()
}

function delayClass(ms) {
  if (!ms || ms <= 0) return 'bad'
  if (ms < 120) return 'good'
  if (ms < 300) return 'ok'
  return 'slow'
}

function nodeInitials(name) {
  const s = String(name || '?').trim()
  if (!s) return '?'
  const parts = s.split(/[\s#_-]+/).filter(Boolean)
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase()
  return s.slice(0, 2).toUpperCase()
}

function friendlyError(err) {
  const s = String(err || '')
  if (/no active profile/i.test(s)) return 'Сначала добавьте и выберите профиль'
  if (/not connected/i.test(s)) return 'Сначала подключитесь'
  if (/administrator|elevation|ErrNeedAdmin/i.test(s)) return 'Нужны права администратора'
  if (/kill switch requires/i.test(s)) return 'Kill switch: укажите VPN-интерфейс в настройках'
  return s.replace(/^Error:\s*/i, '').replace(/^.*?:\s*/, (m) => (m.length > 40 ? '' : m)) || 'Что-то пошло не так'
}

const titles = {
  home: 'Подключение',
  profiles: 'Профили',
  nodes: 'Серверы',
  settings: 'Настройки',
}

function shellHTML() {
  return `
    <div class="app">
      <aside class="sidebar">
        <div class="brand-block">
          <div class="brand">My<span>Internet</span>VPN</div>
          <div class="brand-sub">Безопасное соединение</div>
        </div>
        <nav class="nav" id="nav">
          <button class="nav-btn active" data-view="home" type="button">${icons.home}<span>Главная</span></button>
          <button class="nav-btn" data-view="profiles" type="button">${icons.profiles}<span>Профили</span></button>
          <button class="nav-btn" data-view="nodes" type="button">${icons.nodes}<span>Серверы</span></button>
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
  return ''
}

function renderHome(state) {
  const { status, currentNode, nodes, busy } = state
  const connected = status.state === 'connected'
  const q = status.subscriptionQuota || {}
  const used = Number(q.used || 0)
  const total = Number(q.total || 0)
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0
  const node = currentNode || status.selectedNode || ''
  const nodeInfo = (nodes || []).find((n) => n.name === node)
  const delay = nodeInfo?.delay || 0
  const hasProfile = !!status.activeProfile

  return `
    <div class="home">
      <div class="quota-card">
        <button class="profile-chip" type="button" data-action="go-profiles" title="Сменить профиль">
          <strong id="active-profile">${escapeHtml(status.activeProfile || 'Нет профиля')}</strong>
        </button>
        <div class="quota-mid">
          <div class="quota-text">
            <span id="quota-text">${fmtBytes(used)} / ${fmtBytes(total || -1)}</span>
            <span class="left" id="quota-pct">${pct ? pct + '%' : ''}</span>
          </div>
          <div class="bar"><i id="quota-bar" style="width:${pct}%"></i></div>
        </div>
        <div class="quota-expire" id="quota-expire">${escapeHtml(daysLeft(q.expireUnix))} дн.</div>
      </div>

      <div class="stage">
        <button id="power-btn" class="power ${connected ? 'on' : 'off'} ${busy ? 'busy' : ''}" type="button" data-action="toggle" ${hasProfile ? '' : 'disabled'} aria-label="Подключить">
          <span class="power-ring"></span>
          <div class="power-mark">${icons.power}</div>
        </button>
        <div id="status-line" class="status-line ${connected ? 'on' : 'off'}">${connected ? 'Подключено' : hasProfile ? 'Отключено' : 'Добавьте профиль'}</div>
        <div class="latency">
          ${icons.signal}
          <strong id="latency-value">${delay > 0 ? delay + ' мс' : connected ? '—' : 'ожидание'}</strong>
          <span class="dot">·</span>
          <span id="mode-value">${escapeHtml(modeLabel(status.mode))}</span>
        </div>
        ${!hasProfile ? `<button class="primary cta-inline" type="button" data-action="go-profiles">Добавить профиль</button>` : ''}
      </div>

      <button class="server-card" type="button" data-action="go-nodes" ${connected || hasProfile ? '' : 'disabled'}>
        <div class="server-flag" id="server-flag">${escapeHtml(nodeInitials(node || 'AU'))}</div>
        <div>
          <div class="title" id="server-title">${escapeHtml(node || 'Автовыбор')}</div>
          <div class="sub" id="server-sub">${escapeHtml(nodeKind(nodeInfo?.type))} · нажмите, чтобы выбрать</div>
        </div>
        <div class="chev">${icons.chevron}</div>
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
          <textarea name="proxy" rows="4" placeholder="https://…/subscription&#10;или vless://… / vmess://… / ss://…&#10;или Clash YAML" required></textarea>
        </label>
        <label>Название
          <input name="name" placeholder="Например: Основной" />
        </label>
        <button class="primary" type="submit">Импортировать</button>
      </form>
    </div>` : ''}
  `
}

function renderNodes(state) {
  const { status, nodes, currentNode, nodeQuery } = state
  const connected = status.state === 'connected'
  if (!connected) {
    return `
      <div class="panel">
        <div class="empty">
          <p>Список серверов доступен после подключения</p>
          <button class="primary" type="button" data-action="go-home">На главную</button>
        </div>
      </div>`
  }

  const q = String(nodeQuery || '').trim().toLowerCase()
  const filtered = (nodes || []).filter((n) => !q || String(n.name).toLowerCase().includes(q))
  const rows = filtered.map((n) => {
    const delay = n.delay || 0
    return `
      <button class="item node-item ${currentNode === n.name ? 'active' : ''}" type="button" data-node="${escapeHtml(n.name)}">
        <div class="node-main">
          <strong>${escapeHtml(n.name)}</strong>
          <span>${escapeHtml(nodeKind(n.type))}</span>
        </div>
        <span class="badge ${delayClass(delay)}">${delay > 0 ? delay + ' мс' : '—'}</span>
      </button>`
  }).join('')

  return `
    <div class="panel">
      <div class="search-box">
        ${icons.search}
        <input id="node-search" type="search" placeholder="Поиск сервера" value="${escapeHtml(nodeQuery || '')}" />
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
  { id: 'routing', title: 'Маршрутизация', hint: 'Режим, обход LAN и GEOIP', icon: 'route' },
  { id: 'dns', title: 'DNS', hint: 'Резолверы и Fake-IP', icon: 'dns' },
  { id: 'inbound', title: 'Входящие', hint: 'Порт, TUN, LAN', icon: 'inbox' },
  { id: 'tricks', title: 'Обход DPI', hint: 'Sniffer и ускорение TCP', icon: 'scissors' },
  { id: 'warp', title: 'WARP', hint: 'Cloudflare WireGuard', icon: 'cloud' },
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

function renderSettingsMenu() {
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
        <img class="logo" src="./appicon.png" width="48" height="48" alt="" />
        <div>
          <strong>MyInternetVPN</strong>
          <div class="muted">Настройки ядра mihomo</div>
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
        ${switchRow('autostart', s.autostart, 'Автозапуск', 'При входе в Windows')}
        ${switchRow('closeToTray', s.closeToTray, 'Сворачивать в трей', 'Крестик скрывает окно')}
        ${switchRow('autoReconnect', s.autoReconnect, 'Автопереподключение', 'При обрыве связи')}
        ${switchRow('killSwitch', s.killSwitch, 'Kill switch', 'Блокировать интернет без VPN')}
        ${switchRow('dnsLeakProtection', s.dnsLeakProtection, 'Защита от DNS-утечек', 'Блокировать внешний DNS :53')}
        ${switchRow('autoUpdateSubscriptions', s.autoUpdateSubscriptions, 'Автообновление подписок')}
        <label class="field">Интервал подписок
          <select name="subscriptionIntervalMin">
            <option value="60" ${Number(s.subscriptionIntervalMin) === 60 ? 'selected' : ''}>Каждый час</option>
            <option value="180" ${Number(s.subscriptionIntervalMin) === 180 ? 'selected' : ''}>Каждые 3 часа</option>
            <option value="360" ${![60, 180, 720, 1440].includes(Number(s.subscriptionIntervalMin)) ? 'selected' : ''}>Каждые 6 часов</option>
            <option value="720" ${Number(s.subscriptionIntervalMin) === 720 ? 'selected' : ''}>Каждые 12 часов</option>
            <option value="1440" ${Number(s.subscriptionIntervalMin) === 1440 ? 'selected' : ''}>Раз в сутки</option>
          </select>
        </label>
        <label class="field">Проверка связи (сек)
          <input name="healthIntervalSec" type="number" min="5" max="600" value="${Number(s.healthIntervalSec) || 30}"/>
        </label>
        <label class="field">Уровень логов
          <select name="logLevel">
            ${['silent', 'error', 'warning', 'info', 'debug'].map((lvl) =>
              `<option value="${lvl}" ${s.logLevel === lvl ? 'selected' : ''}>${lvl}</option>`).join('')}
          </select>
        </label>
        <div class="actions">
          <button class="primary" type="submit">Сохранить</button>
          <button class="ghost" type="button" data-action="update">Проверить обновления</button>
        </div>
        <div class="muted" style="margin-top:10px">Версия ${escapeHtml(v)}</div>
      </div>
    </form>
  `
}

function renderSettingsRouting(s) {
  const geo = String(s.bypassGeoip || '').toUpperCase()
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
        ${switchRow('ipv6', s.ipv6, 'IPv6', 'Разрешить IPv6 в ядре')}
        ${switchRow('bypassLan', s.bypassLan !== false, 'Обход локальной сети', 'LAN и link-local → DIRECT')}
        <label class="field">Обход по GEOIP
          <select name="bypassGeoip">
            <option value="CN" ${geo === 'CN' || geo === '' ? 'selected' : ''}>Китай (CN)</option>
            <option value="RU" ${geo === 'RU' ? 'selected' : ''}>Россия (RU)</option>
            <option value="IR" ${geo === 'IR' ? 'selected' : ''}>Иран (IR)</option>
            <option value="OFF" ${geo === 'OFF' || geo === 'NONE' ? 'selected' : ''}>Выключено</option>
          </select>
        </label>
        <div class="actions"><button class="primary" type="submit">Сохранить</button></div>
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
        <label class="field">Режим
          <select name="dnsEnhancedMode">
            <option value="fake-ip" ${s.dnsEnhancedMode !== 'redir-host' ? 'selected' : ''}>Fake-IP</option>
            <option value="redir-host" ${s.dnsEnhancedMode === 'redir-host' ? 'selected' : ''}>Redir-Host</option>
          </select>
        </label>
        <label class="field">Nameserver
          <input name="dnsNameservers" value="${escapeHtml(s.dnsNameservers || '8.8.8.8, 1.1.1.1')}" placeholder="8.8.8.8, 1.1.1.1"/>
        </label>
        <label class="field">Fallback
          <input name="dnsFallbacks" value="${escapeHtml(s.dnsFallbacks || 'tls://1.1.1.1:853')}" placeholder="tls://1.1.1.1:853"/>
        </label>
        <label class="field">Fake-IP range
          <input name="dnsFakeIpRange" value="${escapeHtml(s.dnsFakeIpRange || '198.18.0.1/16')}"/>
        </label>
        <div class="actions"><button class="primary" type="submit">Сохранить</button></div>
      </div>
    </form>
  `
}

function renderSettingsInbound(s) {
  return `
    <form id="settings-form" class="settings" data-section="inbound">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">Входящие</div>
        <label class="field">Mixed port
          <input name="mixedPort" type="number" min="1024" max="65535" value="${Number(s.mixedPort) || 7890}"/>
        </label>
        ${switchRow('allowLan', s.allowLan, 'Разрешить LAN', 'Доступ к прокси из локальной сети')}
        ${switchRow('tun', s.tun, 'TUN-режим', 'Перехват всего трафика системы')}
        <label class="field">TUN stack
          <select name="tunStack">
            ${['system', 'gvisor', 'mixed'].map((st) =>
              `<option value="${st}" ${(s.tunStack || 'system') === st ? 'selected' : ''}>${st}</option>`).join('')}
          </select>
        </label>
        ${switchRow('useSystemProxy', s.useSystemProxy, 'Системный прокси', 'Если TUN выключен')}
        <div class="actions"><button class="primary" type="submit">Сохранить</button></div>
      </div>
    </form>
  `
}

function renderSettingsTricks(s) {
  return `
    <form id="settings-form" class="settings" data-section="tricks">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">Обход DPI</div>
        <p class="settings-note">Аналог «трюков TLS» для mihomo: sniffer и параллельные TCP. Фрагментация TLS (sing-box) здесь недоступна.</p>
        ${switchRow('sniffer', s.sniffer !== false, 'Sniffer', 'Определять домен по TLS/HTTP')}
        ${switchRow('tcpConcurrent', s.tcpConcurrent !== false, 'TCP concurrent', 'Параллельные соединения')}
        ${switchRow('unifiedDelay', s.unifiedDelay !== false, 'Unified delay', 'Точнее измерять задержку узлов')}
        <div class="actions"><button class="primary" type="submit">Сохранить</button></div>
      </div>
    </form>
  `
}

function renderSettingsWARP(s) {
  return `
    <form id="settings-form" class="settings" data-section="warp">
      ${settingsBack()}
      <div class="panel">
        <div class="panel-title">WARP</div>
        <p class="settings-note">Добавляет Cloudflare WARP (WireGuard) в список серверов. Нужен private key из WARP / wgcf.</p>
        ${switchRow('warpEnabled', s.warpEnabled, 'Включить WARP', 'Появится как сервер WARP')}
        <label class="field">Private key
          <input name="warpPrivateKey" value="${escapeHtml(s.warpPrivateKey || '')}" placeholder="base64 WireGuard private key" autocomplete="off"/>
        </label>
        <label class="field">Local address
          <input name="warpLocalAddress" value="${escapeHtml(s.warpLocalAddress || '172.16.0.2/32')}"/>
        </label>
        <label class="field">Endpoint
          <input name="warpEndpoint" value="${escapeHtml(s.warpEndpoint || 'engage.cloudflareclient.com:2408')}"/>
        </label>
        <label class="field">Cloudflare public key
          <input name="warpPublicKey" value="${escapeHtml(s.warpPublicKey || 'bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=')}"/>
        </label>
        <div class="actions"><button class="primary" type="submit">Сохранить</button></div>
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
    default: return renderSettingsMenu()
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
        warpPrivateKey: String(fd.get('warpPrivateKey') || ''),
        warpLocalAddress: String(fd.get('warpLocalAddress') || ''),
        warpEndpoint: String(fd.get('warpEndpoint') || ''),
        warpPublicKey: String(fd.get('warpPublicKey') || ''),
      }
    default:
      return base
  }
}

function renderView(state) {
  switch (state.view) {
    case 'profiles': return renderProfiles(state)
    case 'nodes': return renderNodes(state)
    case 'settings': return renderSettings(state)
    default: return renderHome(state)
  }
}

async function boot() {
  const bridge = await api()
  const state = {
    view: 'home',
    status: await bridge.GetStatus(),
    profiles: [],
    settings: {},
    nodes: [],
    currentNode: '',
    logs: '',
    message: '',
    messageOk: false,
    busy: false,
    lastZip: '',
    delayMap: {},
    formDirty: false,
    showImport: false,
    nodeQuery: '',
    toastTimer: 0,
    settingsSection: '',
  }

  root.innerHTML = shellHTML()
  const content = document.getElementById('content')
  const toast = document.getElementById('toast')

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
      : (titles[state.view] || 'MyInternetVPN')
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
    const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0
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
    const latency = document.getElementById('latency-value')
    if (latency) latency.textContent = delay > 0 ? delay + ' мс' : connected ? '—' : 'ожидание'
    const mode = document.getElementById('mode-value')
    if (mode) mode.textContent = modeLabel(state.status.mode)
    const profileChip = document.getElementById('active-profile')
    if (profileChip) profileChip.textContent = state.status.activeProfile || 'Нет профиля'
    const quotaText = document.getElementById('quota-text')
    if (quotaText) quotaText.textContent = `${fmtBytes(used)} / ${fmtBytes(total || -1)}`
    const quotaPct = document.getElementById('quota-pct')
    if (quotaPct) quotaPct.textContent = pct ? pct + '%' : ''
    const bar = document.getElementById('quota-bar')
    if (bar) bar.style.width = pct + '%'
    const expire = document.getElementById('quota-expire')
    if (expire) expire.textContent = `${daysLeft(q.expireUnix)} дн.`
    const serverTitle = document.getElementById('server-title')
    if (serverTitle) serverTitle.textContent = node || 'Автовыбор'
    const serverSub = document.getElementById('server-sub')
    if (serverSub) serverSub.textContent = `${nodeKind(nodeInfo?.type)} · нажмите, чтобы выбрать`
    const serverFlag = document.getElementById('server-flag')
    if (serverFlag) serverFlag.textContent = nodeInitials(node || 'AU')
  }

  function paintLive() {
    paintChrome()
    patchHomeLive()
  }

  function paintContent({ force = false } = {}) {
    if (!force && state.formDirty) {
      paintLive()
      return
    }
    content.innerHTML = renderView(state)
    state.formDirty = false
    paintChrome()
    if (state.message) showToast(state.message, state.messageOk)
  }

  async function loadData({ soft = false } = {}) {
    if (soft) {
      state.status = await bridge.GetStatus()
      if (state.view === 'nodes' && state.status.state === 'connected') {
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
      return
    }

    const [status, profiles, settings] = await Promise.all([
      bridge.GetStatus(),
      bridge.ListProfiles(),
      bridge.GetSettings(),
    ])
    state.status = status
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
  }

  async function refresh({ force = false, soft = true } = {}) {
    await loadData({ soft })
    if (force) paintContent({ force: true })
    else paintLive()
  }

  async function setView(view) {
    if (!titles[view]) view = 'home'
    state.view = view
    state.formDirty = false
    if (view !== 'settings') state.settingsSection = ''
    await loadData({ soft: false })
    paintContent({ force: true })
  }

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
    if (e.target.closest('form, input, textarea, select')) state.formDirty = true
  })
  root.addEventListener('change', (e) => {
    if (e.target.closest('form, input, textarea, select')) state.formDirty = true
  })

  root.addEventListener('click', async (e) => {
    const nav = e.target.closest('[data-view]')
    if (nav) {
      await setView(nav.getAttribute('data-view'))
      return
    }

    const settingsSectionBtn = e.target.closest('[data-settings-section]')
    if (settingsSectionBtn && state.view === 'settings') {
      state.settingsSection = settingsSectionBtn.getAttribute('data-settings-section') || ''
      state.formDirty = false
      paintContent({ force: true })
      return
    }

    const action = e.target.closest('[data-action]')?.getAttribute('data-action')
    const profile = e.target.closest('[data-profile]')?.getAttribute('data-profile')
    const syncOne = e.target.closest('[data-sync]')?.getAttribute('data-sync')
    const remove = e.target.closest('[data-remove]')?.getAttribute('data-remove')
    const node = e.target.closest('[data-node]')?.getAttribute('data-node')

    try {
      if (action === 'toggle') {
        if (!state.status.activeProfile) {
          showToast('Сначала добавьте профиль', false)
          await setView('profiles')
          return
        }
        state.busy = true
        paintLive()
        await bridge.ToggleConnect()
        state.busy = false
        await refresh({ force: true })
        showToast(state.status.state === 'connected' ? 'Подключено' : 'Отключено', true)
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
      if (action === 'sync-all') {
        showToast('Обновляю подписки…', true)
        const n = await bridge.SyncAllSubscriptions()
        showToast(n ? `Обновлено: ${n}` : 'Нечего обновлять', true)
        await refresh({ force: true })
        return
      }
      if (action === 'test-all') {
        showToast('Проверяю серверы…', true)
        const results = await bridge.TestAllNodes()
        Object.keys(state.delayMap).forEach((k) => delete state.delayMap[k])
        for (const r of results || []) state.delayMap[r.name] = r.delay || 0
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
        } else {
          state.lastZip = await bridge.DownloadUpdate()
          await bridge.ApplyUpdate(state.lastZip)
          showToast('Обновление установлено, перезапуск…', true)
        }
        return
      }
      if (profile) {
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
        showToast('Сервер выбран', true)
        await setView('home')
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
        showToast('Профиль добавлен', true)
        await refresh({ force: true })
      } catch (err) {
        showToast(friendlyError(err), false)
      }
      return
    }
    if (e.target.id === 'settings-form') {
      e.preventDefault()
      const fd = new FormData(e.target)
      const section = e.target.getAttribute('data-section') || state.settingsSection
      try {
        const next = collectSettingsPatch(section, fd, state.settings)
        await bridge.SaveSettings(next)
        state.settings = next
        state.formDirty = false
        showToast('Настройки сохранены', true)
        await refresh({ force: true, soft: false })
      } catch (err) {
        showToast(friendlyError(err), false)
      }
    }
  })

  await refresh({ force: true, soft: false })
  setInterval(() => { refresh({ soft: true }).catch(() => {}) }, 2500)
}

boot()
