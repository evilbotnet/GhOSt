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

  type Tab = 'apps' | 'skills' | 'tools' | 'mcp' | 'ghost';
  let tab = $state<Tab>('apps');

  const TABS: { id: Tab; name: string; icon: string }[] = [
    { id: 'apps', name: 'Web Apps', icon: 'browser' },
    { id: 'skills', name: 'Skills', icon: 'editor' },
    { id: 'tools', name: 'Tools', icon: 'terminal' },
    { id: 'mcp', name: 'MCP', icon: 'wifi' },
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
</script>

<div class="hub">
  <aside>
    <div class="title">
      <Icon name="launcher" size={16} />
      <span>Hub</span>
    </div>
    {#each TABS as t (t.id)}
      <button class:active={tab === t.id} onclick={() => { tab = t.id; refreshExt(); }}>
        <Icon name={t.icon} size={15} />
        <span>{t.name}</span>
      </button>
    {/each}
  </aside>

  <section>
    {#if tab === 'apps'}
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
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--err); flex: none; }
  .dot.on { background: var(--ok); }
  .empty { color: var(--text-low); font-size: 13px; padding: 16px 0; text-align: center; }

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
