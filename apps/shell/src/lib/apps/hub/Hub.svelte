<script lang="ts">
  // The Hub — GhOSt's control center. One pane for everything installed:
  // web apps (install/remove), Ghost's skills, tools, MCP servers, and
  // personality. The single home for the OS's extensibility.
  import Icon from '../../desktop/Icon.svelte';
  import { api } from '../../api/client';
  import { wm, type Win } from '../../wm/wm.svelte';
  import { getApp } from '../registry';
  import { webApps } from '../../api/webapps.svelte';
  import type { SkillInfo, ToolInfo, MCPInfo } from '../../api/ghost.svelte';

  let { win: _win }: { win: Win } = $props();

  type Tab = 'store' | 'apps' | 'skills' | 'tools' | 'mcp' | 'schedules' | 'ghost';
  let tab = $state<Tab>('store');

  const TABS: { id: Tab; name: string; icon: string }[] = [
    { id: 'store', name: 'Store', icon: 'launcher' },
    { id: 'apps', name: 'Web Apps', icon: 'browser' },
    { id: 'skills', name: 'Skills', icon: 'editor' },
    { id: 'tools', name: 'Tools', icon: 'terminal' },
    { id: 'mcp', name: 'MCP', icon: 'wifi' },
    { id: 'schedules', name: 'Schedules', icon: 'info' },
    { id: 'ghost', name: 'Personality', icon: 'info' },
  ];

  // --- web apps ---
  let installName = $state('');
  let installURL = $state('');
  webApps.start();
  async function installApp() {
    if (!installURL.trim()) return;
    let url = installURL.trim();
    if (!/^https?:\/\//.test(url)) url = 'https://' + url;
    await webApps.install(installName.trim(), url);
    installName = '';
    installURL = '';
  }

  // --- ghost extensions ---
  let skills = $state<SkillInfo[]>([]);
  let tools = $state<ToolInfo[]>([]);
  let mcp = $state<MCPInfo[]>([]);
  let soulName = $state('Ghost');
  let soulBody = $state('');
  let soulSaved = $state(false);

  function refreshExt() {
    api.get<SkillInfo[]>('/ai/skills').then((v) => (skills = v)).catch(() => {});
    api.get<ToolInfo[]>('/ai/tools').then((v) => (tools = v)).catch(() => {});
    api.get<MCPInfo[]>('/ai/mcp').then((v) => (mcp = v)).catch(() => {});
    api.get<{ name: string; body: string }>('/ai/soul')
      .then((s) => { soulName = s.name || 'Ghost'; soulBody = s.body || ''; })
      .catch(() => {});
  }
  refreshExt();

  async function saveSoul() {
    await api.post('/setup/soul', { name: soulName.trim() || 'Ghost', body: soulBody });
    soulSaved = true;
    setTimeout(() => (soulSaved = false), 2000);
  }

  // Open a config folder in Files so users can drop in skills/tools.
  function reveal(path: string) {
    wm.open(getApp('files'), { path });
  }

  // --- MCP server management ---
  let mcpName = $state('');
  let mcpCommand = $state('');
  let mcpBusy = $state(false);
  async function addMCP() {
    if (!mcpName.trim() || !mcpCommand.trim()) return;
    mcpBusy = true;
    try {
      await api.post('/setup/mcp', { name: mcpName.trim(), command: mcpCommand.trim() });
      mcpName = '';
      mcpCommand = '';
      setTimeout(refreshExt, 300); // give the server a moment to connect
    } finally {
      mcpBusy = false;
    }
  }
  async function removeMCP(name: string) {
    await api.del(`/setup/mcp/${encodeURIComponent(name)}`);
    refreshExt();
  }

  // --- store (signed git-index registry, ADR 0009) ---
  interface StoreEntry {
    type: 'app' | 'skill' | 'tool' | 'mcp';
    id: string;
    name: string;
    version: string;
    description?: string;
    icon?: string;
    permissions?: string[];
  }
  interface OSApp {
    id: string;
    name: string;
    version: string;
    granted?: string[];
  }
  let storeConfigured = $state(false);
  let storeURL = $state('');
  let storeError = $state('');
  let catalog = $state<StoreEntry[]>([]);
  let osApps = $state<OSApp[]>([]);
  let busyId = $state('');
  // config form
  let cfgURL = $state('');
  let cfgKey = $state('');
  let showCfg = $state(false);

  function refreshStore() {
    api
      .get<{ configured: boolean; url: string; error?: string; entries: StoreEntry[] }>('/store')
      .then((s) => {
        storeConfigured = s.configured;
        storeURL = s.url || '';
        storeError = s.error || '';
        catalog = s.entries || [];
      })
      .catch(() => {});
    api.get<OSApp[]>('/osapps').then((v) => (osApps = v)).catch(() => {});
  }
  refreshStore();

  async function saveStoreConfig() {
    if (!cfgURL.trim() || !cfgKey.trim()) return;
    await api.put('/store/config', { indexURL: cfgURL.trim(), publicKey: cfgKey.trim() });
    showCfg = false;
    refreshStore();
  }

  function installed(id: string): boolean {
    return osApps.some((a) => a.id === id);
  }

  async function installEntry(e: StoreEntry) {
    // Apps carry permissions — confirm the grant before installing.
    let granted: string[] = [];
    if (e.type === 'app' && e.permissions?.length) {
      const ok = confirm(
        `${e.name} requests these permissions:\n\n` +
          e.permissions.map((p) => '  • ' + p).join('\n') +
          `\n\nInstall and grant them?`,
      );
      if (!ok) return;
      granted = e.permissions;
    }
    busyId = e.id;
    try {
      await api.post('/store/install', { id: e.id, granted });
      refreshStore();
    } catch (err) {
      storeError = err instanceof Error ? err.message : String(err);
    } finally {
      busyId = '';
    }
  }

  async function uninstallOSApp(id: string) {
    await api.del(`/osapps/${encodeURIComponent(id)}`);
    refreshStore();
  }

  // --- scheduled Ghost (proactive runs) ---
  interface Schedule {
    id: string;
    name: string;
    prompt: string;
    enabled: boolean;
    notify: boolean;
    every?: string;
    at?: string;
    nextRun?: string;
    lastResult?: string;
  }
  let schedules = $state<Schedule[]>([]);
  let schedName = $state('');
  let schedPrompt = $state('');
  let schedEvery = $state('6h');
  let schedBusy = $state(false);
  let runningId = $state('');

  function refreshSchedules() {
    api.get<Schedule[]>('/ai/schedules').then((v) => (schedules = v)).catch(() => {});
  }
  refreshSchedules();

  async function addSchedule() {
    if (!schedName.trim() || !schedPrompt.trim()) return;
    schedBusy = true;
    try {
      await api.post('/ai/schedules', {
        name: schedName.trim(),
        prompt: schedPrompt.trim(),
        every: schedEvery.trim(),
        enabled: true,
        notify: true,
      });
      schedName = '';
      schedPrompt = '';
      refreshSchedules();
    } finally {
      schedBusy = false;
    }
  }
  async function toggleSchedule(s: Schedule) {
    await api.post('/ai/schedules', { ...s, enabled: !s.enabled });
    refreshSchedules();
  }
  async function removeSchedule(id: string) {
    await api.del(`/ai/schedules/${encodeURIComponent(id)}`);
    refreshSchedules();
  }
  async function runSchedule(id: string) {
    runningId = id;
    try {
      await api.post(`/ai/schedules/${encodeURIComponent(id)}/run`, {});
      refreshSchedules();
    } finally {
      runningId = '';
    }
  }
</script>

<div class="hub">
  <aside>
    <div class="title">
      <Icon name="launcher" size={16} />
      <span>Hub</span>
    </div>
    {#each TABS as t (t.id)}
      <button class:active={tab === t.id} onclick={() => { tab = t.id; refreshExt(); refreshSchedules(); refreshStore(); }}>
        <Icon name={t.icon} size={15} />
        <span>{t.name}</span>
      </button>
    {/each}
  </aside>

  <section>
    {#if tab === 'store'}
      <header><h2>Store</h2></header>
      <p class="hint">
        Browse and one-click-install apps, skills, tools, and MCP servers from a
        signed git index. ghostd verifies the index signature against the pinned
        key before anything installs.
      </p>

      {#if !storeConfigured || showCfg}
        <form class="install col" onsubmit={(e) => { e.preventDefault(); saveStoreConfig(); }}>
          <input bind:value={cfgURL} placeholder="index URL (https://…/index.json or a local path)" />
          <input bind:value={cfgKey} placeholder="pinned public key (base64 Ed25519)" />
          <button class="cta" type="submit">Save store</button>
        </form>
      {:else}
        <div class="storebar">
          <span class="rsub mono">{storeURL}</span>
          <button class="action" onclick={() => { cfgURL = storeURL; showCfg = true; }}>Change…</button>
        </div>
      {/if}

      {#if storeError}<p class="storeerr">⚠ {storeError}</p>{/if}

      <div class="rows">
        {#each catalog as e (e.id)}
          <div class="row sched">
            <Icon name={e.icon || 'launcher'} size={16} />
            <div class="schedbody">
              <div class="schedhead">
                <span class="rname">{e.name}</span>
                <span class="badge">{e.type}</span>
                <span class="rsub mono">v{e.version}</span>
              </div>
              {#if e.description}<span class="schedprompt">{e.description}</span>{/if}
              {#if e.permissions?.length}
                <span class="schedlast">needs: {e.permissions.join(' · ')}</span>
              {/if}
            </div>
            {#if installed(e.id)}
              <span class="badge ok">installed</span>
            {:else}
              <button class="cta sm" disabled={busyId === e.id} onclick={() => installEntry(e)}>
                {busyId === e.id ? 'Installing…' : 'Install'}
              </button>
            {/if}
          </div>
        {/each}
        {#if storeConfigured && !storeError && catalog.length === 0}
          <p class="empty">The store index is empty.</p>
        {/if}
      </div>

      {#if osApps.length > 0}
        <header><h2>Installed packages</h2></header>
        <div class="rows">
          {#each osApps as a (a.id)}
            <div class="row">
              <span class="rname">{a.name}</span>
              <span class="rsub mono">{a.id} · v{a.version}{a.granted?.length ? ' · ' + a.granted.join(', ') : ''}</span>
              <button class="del" aria-label="Uninstall" onclick={() => uninstallOSApp(a.id)}>
                <Icon name="trash" size={14} />
              </button>
            </div>
          {/each}
        </div>
      {/if}
    {:else if tab === 'apps'}
      <header><h2>Web Apps</h2></header>
      <p class="hint">Any website, installed as its own windowed app.</p>
      <form class="install" onsubmit={(e) => { e.preventDefault(); installApp(); }}>
        <input bind:value={installURL} placeholder="excalidraw.com" />
        <input bind:value={installName} placeholder="name (optional)" />
        <button class="cta" type="submit">Install</button>
      </form>
      <div class="rows">
        {#each webApps.list as app (app.id)}
          <div class="row">
            <span class="ic"><Icon name={app.icon} size={16} /></span>
            <span class="rname">{app.name}</span>
            <span class="rsub">{app.url}</span>
            <button class="open" onclick={() => webApps.launch(app.id)}>Open</button>
            <button class="del" aria-label="Uninstall" onclick={() => webApps.uninstall(app.id)}>
              <Icon name="trash" size={14} />
            </button>
          </div>
        {/each}
        {#if webApps.list.length === 0}<p class="empty">No web apps installed yet.</p>{/if}
      </div>
    {:else if tab === 'skills'}
      <header>
        <h2>Skills</h2>
        <button class="action" onclick={() => reveal('~/.config/ghost/skills')}>Open folder</button>
      </header>
      <p class="hint">Drop-in expertise for Ghost — a folder with a SKILL.md. Loaded on demand.</p>
      <div class="rows">
        {#each skills as sk (sk.name)}
          <div class="row col">
            <span class="rname mono">{sk.name}</span>
            <span class="rdesc">{sk.description}</span>
          </div>
        {/each}
        {#if skills.length === 0}<p class="empty">No skills installed.</p>{/if}
      </div>
    {:else if tab === 'tools'}
      <header>
        <h2>Tools</h2>
        <button class="action" onclick={() => reveal('~/.config/ghost/tools')}>Open folder</button>
      </header>
      <p class="hint">New actions Ghost can take — a JSON manifest + an executable.</p>
      <div class="rows">
        {#each tools as t (t.name)}
          <div class="row col">
            <span class="rname mono">{t.name}
              {#if t.mutating}<span class="badge warn">gated</span>{:else}<span class="badge">read-only</span>{/if}
            </span>
            <span class="rdesc">{t.description}</span>
          </div>
        {/each}
        {#if tools.length === 0}<p class="empty">No external tools installed.</p>{/if}
      </div>
    {:else if tab === 'mcp'}
      <header><h2>MCP Servers</h2></header>
      <p class="hint">
        Model Context Protocol servers add whole toolsets to Ghost. Give a name
        and the launch command (stdio); it connects on the next Ghost run.
      </p>
      <form class="install" onsubmit={(e) => { e.preventDefault(); addMCP(); }}>
        <input class="narrow" bind:value={mcpName} placeholder="name" />
        <input bind:value={mcpCommand} placeholder="npx -y @modelcontextprotocol/server-filesystem ~" />
        <button class="cta" type="submit" disabled={mcpBusy}>{mcpBusy ? 'Adding…' : 'Add'}</button>
      </form>
      <div class="rows">
        {#each mcp as m (m.name)}
          <div class="row">
            <span class="dot" class:on={m.connected}></span>
            <span class="rname mono">{m.name}</span>
            <span class="rsub">{m.connected ? `${m.toolCount} tools` : m.error || 'offline'}</span>
            <button class="del" aria-label="Remove" onclick={() => removeMCP(m.name)}>
              <Icon name="trash" size={14} />
            </button>
          </div>
        {/each}
        {#if mcp.length === 0}<p class="empty">No MCP servers configured.</p>{/if}
      </div>
    {:else if tab === 'schedules'}
      <header><h2>Scheduled Ghost</h2></header>
      <p class="hint">
        Proactive runs — a prompt on a timer. Each fires a read-only Ghost run
        and shows the result as a notification. Interval as a duration
        (<span class="mono">30m</span>, <span class="mono">6h</span>).
      </p>
      <form class="install" onsubmit={(e) => { e.preventDefault(); addSchedule(); }}>
        <input class="narrow" bind:value={schedName} placeholder="name" />
        <input class="narrow" bind:value={schedEvery} placeholder="6h" />
        <input bind:value={schedPrompt} placeholder="Summarise what changed in ~/Downloads today" />
        <button class="cta" type="submit" disabled={schedBusy}>{schedBusy ? 'Adding…' : 'Add'}</button>
      </form>
      <div class="rows">
        {#each schedules as s (s.id)}
          <div class="row sched">
            <button
              class="dot toggle"
              class:on={s.enabled}
              aria-label={s.enabled ? 'Disable' : 'Enable'}
              onclick={() => toggleSchedule(s)}></button>
            <div class="schedbody">
              <div class="schedhead">
                <span class="rname">{s.name}</span>
                <span class="rsub mono">every {s.every || s.at}</span>
              </div>
              <span class="schedprompt">{s.prompt}</span>
              {#if s.lastResult}<span class="schedlast">↳ {s.lastResult}</span>{/if}
            </div>
            <button class="del" disabled={runningId === s.id} onclick={() => runSchedule(s.id)}>
              {runningId === s.id ? '…' : 'Run'}
            </button>
            <button class="del" aria-label="Remove" onclick={() => removeSchedule(s.id)}>
              <Icon name="trash" size={14} />
            </button>
          </div>
        {/each}
        {#if schedules.length === 0}<p class="empty">No scheduled runs yet.</p>{/if}
      </div>
    {:else if tab === 'ghost'}
      <header><h2>Personality</h2></header>
      <p class="hint">The name and soul injected into every conversation with Ghost.</p>
      <label class="f">Name
        <input bind:value={soulName} maxlength="24" />
      </label>
      <label class="f">Persona
        <textarea bind:value={soulBody} rows="6"
          placeholder="You are calm, concise, and quietly capable…"></textarea>
      </label>
      <button class="cta" onclick={saveSoul}>{soulSaved ? 'Saved ✓' : 'Save personality'}</button>
    {/if}
  </section>
</div>

<style>
  .hub { display: flex; height: 100%; }
  aside {
    width: 172px;
    flex: none;
    padding: 12px 8px;
    border-right: 1px solid var(--line-soft);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .title {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 10px 12px;
    font-family: var(--font-display);
    font-weight: 650;
    font-size: 16px;
    color: var(--accent);
  }
  aside button {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border-radius: var(--radius-ui);
    color: var(--text-mid);
    font-size: 13px;
    text-align: left;
  }
  aside button:hover { background: var(--ink-3); color: var(--text-hi); }
  aside button.active { background: var(--ink-3); color: var(--accent-bright); }

  section { flex: 1; padding: 18px 22px; overflow-y: auto; }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }
  h2 { font-family: var(--font-display); font-size: 19px; font-weight: 600; }
  .hint { color: var(--text-low); font-size: 12.5px; line-height: 1.5; margin-bottom: 14px; }
  code {
    font-family: var(--font-mono);
    font-size: 11.5px;
    background: var(--ink-3);
    padding: 1px 5px;
    border-radius: 4px;
  }

  .install { display: flex; gap: 8px; margin-bottom: 14px; }
  .install input {
    flex: 1;
    background: var(--ink-2);
    border: 1px solid var(--line);
    border-radius: 7px;
    padding: 8px 10px;
    outline: none;
    color: var(--text-hi);
    font-size: 13px;
  }
  .install input:focus { border-color: var(--accent-dim); }
  .install input.narrow { flex: 0 0 130px; }

  .rows { display: flex; flex-direction: column; gap: 4px; }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 10px;
    border-radius: var(--radius-ui);
    background: var(--ink-2);
    font-size: 13px;
  }
  .row.col { flex-direction: column; align-items: flex-start; gap: 3px; }
  .ic { color: var(--accent); display: grid; place-items: center; }
  .rname { color: var(--text-hi); font-weight: 500; }
  .rname.mono { font-family: var(--font-mono); font-weight: 400; }
  .rsub { color: var(--text-low); font-size: 11.5px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; flex: 1; }
  .rdesc { color: var(--text-mid); font-size: 12px; line-height: 1.45; }
  .open {
    margin-left: auto;
    padding: 5px 12px;
    border-radius: 6px;
    background: var(--ink-3);
    color: var(--text-hi);
    font-size: 12px;
  }
  .open:hover { background: var(--ink-4); }
  .del {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    color: var(--text-low);
  }
  .del:hover { background: var(--err); color: #fff; }
  .badge {
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--text-low);
    color: var(--ink-0);
    margin-left: 6px;
  }
  .badge.warn { background: var(--warn); color: var(--accent-ink); }
  .badge.ok { background: var(--ok); color: var(--ink-0); margin-left: 0; align-self: center; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--err); flex: none; }
  .dot.on { background: var(--ok); }
  .empty { color: var(--text-low); font-size: 13px; padding: 16px 0; text-align: center; }

  /* scheduled Ghost rows */
  .row.sched { align-items: flex-start; }
  .dot.toggle { margin-top: 5px; cursor: pointer; padding: 0; }
  .schedbody { flex: 1; display: flex; flex-direction: column; gap: 2px; min-width: 0; }
  .schedhead { display: flex; align-items: baseline; gap: 8px; }
  .schedprompt { color: var(--text-mid); font-size: 12px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .schedlast { color: var(--text-low); font-size: 11.5px; font-style: italic; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .del { font-size: 12px; }

  .action {
    padding: 6px 12px;
    border-radius: 6px;
    background: var(--ink-3);
    border: 1px solid var(--line-soft);
    font-size: 12px;
    color: var(--text-mid);
  }
  .action:hover { border-color: var(--accent-dim); color: var(--text-hi); }
  .cta {
    padding: 8px 16px;
    border-radius: 7px;
    background: var(--accent);
    color: var(--accent-ink);
    font-weight: 600;
    font-size: 13px;
  }
  .cta:hover { background: var(--accent-bright); }
  .cta.sm { padding: 5px 12px; font-size: 12px; align-self: center; }
  .cta[disabled] { opacity: 0.5; }
  .install.col { flex-direction: column; }
  .storebar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
  .storebar .rsub { flex: 1; }
  .storeerr { color: var(--err); font-size: 12.5px; margin: 0 0 12px; }
  .f {
    display: flex;
    flex-direction: column;
    gap: 5px;
    font-size: 12px;
    color: var(--text-mid);
    margin-bottom: 12px;
    max-width: 420px;
  }
  .f input, .f textarea {
    background: var(--ink-2);
    border: 1px solid var(--line);
    border-radius: 7px;
    padding: 8px 10px;
    outline: none;
    color: var(--text-hi);
    font-size: 13px;
    font-family: var(--font-ui);
    resize: vertical;
  }
  .f input:focus, .f textarea:focus { border-color: var(--accent-dim); }
</style>
