// main.ts — TypeScript for admin UI (tabs, fetch, render, accordion, filter, refresh)
// ... (interfaces unchanged) ...

interface PluginConfig {
    name: string;
    config: Record<string, any>;
}
interface BackendConfig {
    url: string;
    method: string;
    timeout?: number;
}
interface RouteConfig {
    path: string;
    method: string;
    backends: BackendConfig[];
    plugins: PluginConfig[];
    aggregate: string;
    transform: string;
}

interface ServerConfig { port: number; timeout: number; }
interface GatewayConfig {
    schema?: string; name?: string; version?: string;
    server: ServerConfig;
    plugins: PluginConfig[]; routes: RouteConfig[];
}

const CONFIG_URL = '/api/config';

async function fetchConfig(): Promise<GatewayConfig> {
    const resp = await fetch(CONFIG_URL);
    if (!resp.ok) throw new Error(`Config load failed: ${resp.status}`);
    return resp.json();
}

function setVersionInHeader(version?: string) {
    const el = document.getElementById('version-tag');
    if (el) el.textContent = version ? `v${version}` : 'v?';
}

function renderServerInfo(server?: ServerConfig) {
    const container = document.getElementById('server-info');
    if (!container) return;
    if (!server) {
        container.innerHTML = '<div class="glass-card">No server config</div>';
        return;
    }
    container.innerHTML = `
    <div class="glass-card">
      <p class="meta"><span class="label">Port</span><span class="value">${server.port}</span></p>
      <p class="meta"><span class="label">Timeout</span><span class="value">${server.timeout} ms</span></p>
    </div>
  `;
}

/* === Updated: renderPlugins uses .plugin-name and .plugin-config === */
function renderPlugins(plugins?: PluginConfig[]) {
    const container = document.getElementById('plugins-list');
    if (!container) return;
    if (!plugins || plugins.length === 0) {
        container.innerHTML = `<div class="glass-card">No plugins configured</div>`;
        return;
    }

    container.innerHTML = plugins.map(p => `
    <div class="plugin-item glass-card">
      <div class="plugin-name">${escapeHtml(p.name)}</div>
      <div class="plugin-config">${escapeHtml(shortConfig(p.config))}</div>
    </div>
  `).join('');
}

/* === renderRoutes: plugin entries now use .plugin-name and .plugin-config inside list items === */
function renderRoutes(routes?: RouteConfig[]) {
    const container = document.getElementById('routes-list');
    if (!container) return;
    if (!routes || routes.length === 0) {
        container.innerHTML = `<div class="glass-card">No routes configured</div>`;
        return;
    }

    container.innerHTML = routes.map(r => {
        const methodClass = ['GET','POST','PUT','DELETE','PATCH'].includes(r.method.toUpperCase())
            ? r.method.toUpperCase()
            : 'OTHER';
        const backends = r.backends?.map(b =>
            `<li><code>${escapeHtml(b.method)} ${escapeHtml(b.url)}</code>${b.timeout ? ` (${b.timeout}ms)` : ''}</li>`
        ).join('') || '<li>No backends</li>';

        const plugins = r.plugins?.map(p =>
            `<li><span class="plugin-name">${escapeHtml(p.name)}</span> – <span class="plugin-config">${escapeHtml(shortConfig(p.config))}</span></li>`
        ).join('') || '<li>No plugins</li>';

        return `
    <div class="route-card" data-path="${escapeAttr(r.path)}" data-method="${escapeAttr(r.method)}">
      <div class="route-header">
        <div class="left">
          <div class="method ${methodClass}">${escapeHtml(r.method)}</div>
          <div class="path">${escapeHtml(r.path)}</div>
        </div>
        <div class="right"><span class="toggle">▼</span></div>
      </div>
      <div class="route-details">
        <p class="meta"><span class="label">Aggregate:</span> <span class="value">${escapeHtml(r.aggregate)}</span></p>
        <p class="meta"><span class="label">Transform:</span> <span class="value">${escapeHtml(r.transform)}</span></p>
        <div class="meta"><span class="label">Backends:</span></div>
        <ul>${backends}</ul>
        <div class="meta"><span class="label">Plugins:</span></div>
        <ul>${plugins}</ul>
      </div>
    </div>`;
    }).join('');

    // attach accordion handlers
    container.querySelectorAll<HTMLElement>('.route-card').forEach(card => {
        const header = card.querySelector<HTMLElement>('.route-header');
        const toggle = card.querySelector<HTMLElement>('.toggle');
        const details = card.querySelector<HTMLElement>('.route-details');
        if (!header || !details) return;

        // start collapsed
        details.style.maxHeight = '0';
        details.style.opacity = '0';
        details.style.paddingTop = '0';

        header.addEventListener('click', () => {
            const opened = card.classList.toggle('open');
            if (toggle) toggle.textContent = opened ? '▲' : '▼';

            if (opened) {
                details.style.maxHeight = details.scrollHeight + 'px';
                details.style.opacity = '1';
                details.style.paddingTop = '10px';
            } else {
                details.style.maxHeight = '0';
                details.style.opacity = '0';
                details.style.paddingTop = '0';
            }
        });
    });
}


/* tabs, filter, helpers (unchanged) */
function setupTabs() {
    const buttons = document.querySelectorAll<HTMLButtonElement>('aside nav button');
    const sections = document.querySelectorAll<HTMLElement>('main section');
    if (!buttons || buttons.length === 0) return;
    buttons.forEach(btn => {
        btn.addEventListener('click', () => {
            buttons.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const target = btn.getAttribute('data-section');
            sections.forEach(s => s.classList.toggle('visible', s.id === target));
        });
    });
}

function setupFilter(allRoutes: RouteConfig[] = []) {
    const input = document.getElementById('route-filter') as HTMLInputElement | null;
    if (!input) return;
    input.addEventListener('input', () => {
        const q = input.value.trim().toLowerCase();
        if (q === '') { renderRoutes(allRoutes); return; }
        const filtered = allRoutes.filter(r => r.path.toLowerCase().includes(q) || r.method.toLowerCase().includes(q));
        renderRoutes(filtered);
    });
}

/* helpers */
function escapeHtml(s: unknown): string {
    if (s === null || s === undefined) return '';
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;').replace(/'/g, '&#039;');
}
function escapeAttr(s: unknown): string { return escapeHtml(s).replace(/\s+/g, ' '); }
function shortConfig(cfg: Record<string, any>): string {
    try {
        const entries = Object.entries(cfg || {});
        if (entries.length === 0) return '{}';
        const short = entries.slice(0, 3).map(([k, v]) => `${k}: ${String(v)}`).join(', ');
        return entries.length > 3 ? `${short}, …` : short;
    } catch { return '{}'; }
}

/* init */
document.addEventListener('DOMContentLoaded', async () => {
    setupTabs();
    const refreshBtn = document.getElementById('refresh');
    if (refreshBtn) refreshBtn.addEventListener('click', () => void init());
    await init();
});

async function init() {
    try {
        const cfg = await fetchConfig();
        setVersionInHeader(cfg.version);
        renderServerInfo(cfg.server);
        renderPlugins(cfg.plugins);
        renderRoutes(cfg.routes);
        setupFilter(cfg.routes || []);
    } catch (err) {
        console.error('Admin init failed:', err);
        const main = document.querySelector('main');
        if (main) main.innerHTML = `<div style="padding:20px;color:#b91c1c">Failed to load config: ${(err as Error).message}</div>`;
    }
}
