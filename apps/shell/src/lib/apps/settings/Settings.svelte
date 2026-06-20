<script lang="ts">
  import Icon from '../../desktop/Icon.svelte';
  import { api, getToken, type WifiNetwork } from '../../api/client';
  import { system } from '../../api/system.svelte';
  import { wm, type Win } from '../../wm/wm.svelte';
  import { getApp } from '../registry';
  import ThemeControls from './ThemeControls.svelte';

  let { win }: { win: Win } = $props();

  type Panel = 'wifi' | 'sound' | 'appearance' | 'ghost' | 'updates' | 'backup' | 'about';
  let panel = $state<Panel>(((win.props.panel as Panel) ?? 'wifi'));

  const PANELS: { id: Panel; name: string; icon: string }[] = [
    { id: 'wifi', name: 'Network', icon: 'wifi' },
    { id: 'sound', name: 'Sound', icon: 'volume' },
    { id: 'appearance', name: 'Appearance', icon: 'image' },
    { id: 'ghost', name: 'Ghost AI', icon: 'info' },
    { id: 'updates', name: 'Updates', icon: 'refresh' },
    { id: 'backup', name: 'Backup', icon: 'files' },
    { id: 'about', name: 'About', icon: 'info' },
  ];

  // --- devkit (pi/herdr coding agents) ---
  interface DevkitStatus { nodePresent: boolean; state: string; tools: string[]; available: boolean }
  let devkit = $state<DevkitStatus | null>(null);
  let devkitBusy = $state(false);
  function loadDevkit() {
    api.get<DevkitStatus>('/devkit/status').then((d) => (devkit = d)).catch(() => {});
  }
  async function installDevkit() {
    devkitBusy = true;
    try {
      await api.post('/devkit/install', {});
      const poll = setInterval(() => {
        loadDevkit();
        if (devkit?.state === 'ok' || devkit?.state?.startsWith('failed')) {
          clearInterval(poll);
          devkitBusy = false;
        }
      }, 2000);
    } catch {
      devkitBusy = false;
    }
  }

  // --- backup & restore ---
  let backupBusy = $state('');
  let backupMsg = $state('');
  let restored = $state(false);
  async function exportBackup() {
    backupBusy = 'export';
    backupMsg = '';
    try {
      const res = await fetch('/api/v1/backup/export', { headers: { Authorization: 'Bearer ' + getToken() } });
      if (!res.ok) throw new Error('export failed');
      const blob = await res.blob();
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      const date = new Date().toISOString().slice(0, 10);
      a.download = `ghost-backup-${date}.tar.gz`;
      a.click();
      URL.revokeObjectURL(a.href);
      backupMsg = 'Backup downloaded.';
    } catch (e) {
      backupMsg = e instanceof Error ? e.message : String(e);
    } finally {
      backupBusy = '';
    }
  }
  async function importBackup(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    backupBusy = 'import';
    backupMsg = '';
    try {
      const res = await fetch('/api/v1/backup/import', {
        method: 'POST',
        headers: { Authorization: 'Bearer ' + getToken() },
        body: file,
      });
      if (!res.ok) throw new Error((await res.json()).error ?? 'restore failed');
      restored = true;
      backupMsg = 'Restored. Reload to apply.';
    } catch (err) {
      backupMsg = err instanceof Error ? err.message : String(err);
    } finally {
      backupBusy = '';
      input.value = '';
    }
  }

  // --- updates panel ---
  let updates = $state<{ count: number; packages: string[] } | null>(null);
  let checking = $state(false);
  async function checkUpdates() {
    checking = true;
    updates = await api.get<{ count: number; packages: string[] }>('/system/updates').catch(() => null);
    checking = false;
  }
  function openTerminal() {
    wm.open(getApp('terminal'));
  }
  $effect(() => {
    if (panel === 'updates' && updates === null) void checkUpdates();
  });

  // --- Ghost AI panel ---
  type AIMode = 'off' | 'lan' | 'cloud';
  let aiMode = $state<AIMode>('off');
  let aiURL = $state('http://192.168.1.10:11434/v1');
  let aiModel = $state('');
  let aiKey = $state('');
  let aiSaved = $state(false);
  let aiConfigured = $state(false);

  $effect(() => {
    if (panel === 'ghost') {
      api.get<{ configured: boolean }>('/ai/status')
        .then((s) => (aiConfigured = s.configured))
        .catch(() => {});
      if (devkit === null) loadDevkit();
    }
  });

  async function saveAI() {
    await api.post('/setup/ai', { mode: aiMode, url: aiURL, model: aiModel, key: aiKey });
    aiSaved = true;
    aiConfigured = aiMode !== 'off';
    setTimeout(() => (aiSaved = false), 2000);
  }

  // --- personality (SOUL) ---
  let soulName = $state('Ghost');
  let soulBody = $state('');
  let soulSaved = $state(false);
  $effect(() => {
    if (panel === 'ghost') {
      api.get<{ name: string; body: string }>('/ai/soul')
        .then((s) => { soulName = s.name || 'Ghost'; soulBody = s.body || ''; })
        .catch(() => {});
    }
  });
  async function saveSoul() {
    await api.post('/setup/soul', { name: soulName.trim() || 'Ghost', body: soulBody });
    soulSaved = true;
    setTimeout(() => (soulSaved = false), 2000);
  }

  system.start();
  let s = $derived(system.status);

  // --- Wi-Fi panel state ---
  let networks = $state<WifiNetwork[]>([]);
  let scanning = $state(false);
  let connectTo = $state<WifiNetwork | null>(null);
  let password = $state('');
  let wifiError = $state('');

  async function scan() {
    scanning = true;
    wifiError = '';
    try {
      networks = await api.get<WifiNetwork[]>('/system/wifi/networks');
    } catch (e) {
      wifiError = e instanceof Error ? e.message : 'scan failed';
    } finally {
      scanning = false;
    }
  }

  async function connect(net: WifiNetwork) {
    if (net.secured && !net.known && connectTo?.ssid !== net.ssid) {
      connectTo = net;
      password = '';
      return;
    }
    wifiError = '';
    try {
      await api.post('/system/wifi/connect', {
        ssid: net.ssid,
        password: password || undefined,
      });
      connectTo = null;
      await scan();
    } catch (e) {
      wifiError = e instanceof Error ? e.message : 'connect failed';
    }
  }

  $effect(() => {
    if (panel === 'wifi') void scan();
  });
</script>

<div class="settings">
  <aside>
    {#each PANELS as p (p.id)}
      <button class:active={panel === p.id} onclick={() => (panel = p.id)}>
        <Icon name={p.icon} size={15} />
        <span>{p.name}</span>
      </button>
    {/each}
  </aside>

  <section>
    {#if panel === 'wifi'}
      <header>
        <h2>Network</h2>
        <button class="action" onclick={scan} disabled={scanning}>
          <Icon name="refresh" size={13} />
          {scanning ? 'Scanning…' : 'Scan'}
        </button>
      </header>
      {#if wifiError}<p class="error">{wifiError}</p>{/if}
      {#if !s.wifi.available}
        <p class="hint">No Wi-Fi hardware detected on this device.</p>
      {/if}
      <div class="nets">
        {#each networks as net (net.ssid)}
          <div class="net">
            <button class="net-main" onclick={() => connect(net)}>
              <Icon name="wifi" size={15} />
              <span class="ssid">{net.ssid}</span>
              {#if net.active}<span class="tag">connected</span>{/if}
              {#if net.secured}<Icon name="lock" size={12} />{/if}
              <span class="sig">{net.signal}%</span>
            </button>
            {#if connectTo?.ssid === net.ssid}
              <form
                class="pw"
                onsubmit={(e) => {
                  e.preventDefault();
                  connect(net);
                }}
              >
                <!-- svelte-ignore a11y_autofocus -->
                <input
                  type="password"
                  placeholder="Password"
                  autofocus
                  bind:value={password}
                />
                <button class="action" type="submit">Join</button>
              </form>
            {/if}
          </div>
        {/each}
      </div>
    {:else if panel === 'sound'}
      <header><h2>Sound</h2></header>
      <div class="field">
        <span>Output volume</span>
        <input
          type="range"
          min="0"
          max="100"
          value={s.volume.percent}
          oninput={(e) => system.setVolume(Number(e.currentTarget.value))}
        />
        <span class="val">{s.volume.percent}%</span>
      </div>
    {:else if panel === 'appearance'}
      <header><h2>Appearance</h2></header>
      <ThemeControls />
    {:else if panel === 'updates'}
      <header>
        <h2>Updates</h2>
        <button class="action" onclick={checkUpdates} disabled={checking}>
          <Icon name="refresh" size={13} />
          {checking ? 'Checking…' : 'Check'}
        </button>
      </header>
      {#if updates === null}
        <p class="hint">Checking for updates…</p>
      {:else if updates.count === 0}
        <p class="hint">Everything is up to date.</p>
      {:else}
        <p class="hint">{updates.count} update{updates.count === 1 ? '' : 's'} available.</p>
        <div class="upd-list">
          {#each updates.packages as pkg (pkg)}
            <span class="pkg">{pkg}</span>
          {/each}
        </div>
        <button class="action" onclick={openTerminal}>Update in Terminal (sudo apt upgrade)</button>
      {/if}
    {:else if panel === 'ghost'}
      <header><h2>Ghost AI</h2></header>
      <p class="hint">
        Ghost's tools are this OS itself. Choose where its thinking happens —
        nothing leaves the device except to the endpoint you pick.
        {aiConfigured ? ' Currently configured.' : ' Not configured yet.'}
      </p>
      <div class="modes">
        <button class="mode" class:picked={aiMode === 'off'} onclick={() => (aiMode = 'off')}>
          <strong>Off</strong><span>no AI</span>
        </button>
        <button class="mode" class:picked={aiMode === 'lan'} onclick={() => (aiMode = 'lan')}>
          <strong>My own model</strong><span>Ollama / vLLM / llama.cpp</span>
        </button>
        <button class="mode" class:picked={aiMode === 'cloud'} onclick={() => (aiMode = 'cloud')}>
          <strong>Anthropic</strong><span>bring your own key</span>
        </button>
      </div>
      {#if aiMode === 'lan'}
        <label class="f">Endpoint (OpenAI-compatible)
          <input bind:value={aiURL} placeholder="http://host:11434/v1" />
        </label>
        <label class="f">Model
          <input bind:value={aiModel} placeholder="qwen3:8b" />
        </label>
      {:else if aiMode === 'cloud'}
        <label class="f">API key
          <input type="password" bind:value={aiKey} placeholder="sk-ant-…" />
        </label>
        <label class="f">Model
          <input bind:value={aiModel} placeholder="claude-opus-4-8" />
        </label>
      {/if}
      <button class="action" onclick={saveAI}>{aiSaved ? 'Saved ✓' : 'Save'}</button>

      <h3 class="subhead">Personality</h3>
      <p class="hint">The assistant's name and soul — injected into every conversation.</p>
      <label class="f">Name
        <input bind:value={soulName} maxlength="24" />
      </label>
      <label class="f">Persona
        <textarea bind:value={soulBody} rows="5"
          placeholder="You are calm, concise, and quietly capable…"></textarea>
      </label>
      <button class="action" onclick={saveSoul}>{soulSaved ? 'Saved ✓' : 'Save personality'}</button>

      <h3 class="subhead">Developer tools</h3>
      <p class="hint">
        Install pi &amp; herdr — terminal AI coding agents — pre-wired to your
        model through GhOSt's gateway (no keys to paste).
      </p>
      {#if devkit?.state === 'ok'}
        <p class="hint ok">✦ Installed: {devkit.tools.join(', ')}. Run <code>pi</code> in a Terminal.</p>
      {:else if devkit && !devkit.nodePresent}
        <p class="hint">Node.js isn't installed — add the Office suite first (it brings Node).</p>
      {:else if devkit?.available === false}
        <p class="hint">Not available on this host.</p>
      {:else}
        <button class="action" disabled={devkitBusy} onclick={installDevkit}>
          {devkitBusy || devkit?.state === 'installing' ? 'Installing… (a few minutes)' : 'Install dev tools'}
        </button>
        {#if devkit?.state?.startsWith('failed')}<p class="error">{devkit.state}</p>{/if}
      {/if}
    {:else if panel === 'backup'}
      <header><h2>Backup &amp; restore</h2></header>
      <p class="hint">
        Save or restore everything GhOSt holds — settings, skills, tools, memory,
        personality, schedules, installed apps and store config — as a single
        archive. Your documents in Files aren't included; back those up separately.
      </p>
      <div class="backup-actions">
        <button class="action" disabled={backupBusy === 'export'} onclick={exportBackup}>
          {backupBusy === 'export' ? 'Exporting…' : 'Export backup'}
        </button>
        <label class="action file-action">
          {backupBusy === 'import' ? 'Restoring…' : 'Restore from file…'}
          <input type="file" accept=".gz,.tar.gz,application/gzip" onchange={importBackup} hidden />
        </label>
      </div>
      {#if backupMsg}<p class="hint" class:ok={restored}>{backupMsg}</p>{/if}
      {#if restored}
        <button class="action primary" onclick={() => location.reload()}>Reload now</button>
      {/if}
    {:else}
      <header><h2>About</h2></header>
      <dl>
        <dt>Hostname</dt>
        <dd>{s.hostname}</dd>
        <dt>Platform</dt>
        <dd>{s.platform}</dd>
        <dt>Shell</dt>
        <dd>GhOSt 0.1.0</dd>
        <dt>Daemon</dt>
        <dd>{system.online ? 'connected' : 'offline'}</dd>
      </dl>
    {/if}
  </section>
</div>

<style>
  .settings {
    display: flex;
    height: 100%;
  }
  aside {
    width: 168px;
    flex: none;
    padding: 10px 8px;
    border-right: 1px solid var(--line-soft);
    display: flex;
    flex-direction: column;
    gap: 2px;
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
  aside button:hover {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  aside button.active {
    background: var(--ink-3);
    color: var(--accent-bright);
  }

  section {
    flex: 1;
    padding: 18px 22px;
    overflow-y: auto;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  h2 {
    font-family: var(--font-display);
    font-size: 19px;
    font-weight: 600;
  }
  .action {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    border-radius: 6px;
    background: var(--ink-3);
    border: 1px solid var(--line-soft);
    font-size: 12.5px;
  }
  .action:hover:not(:disabled) {
    border-color: var(--accent-dim);
  }
  .action:disabled { opacity: 0.5; }
  .action.primary { background: var(--accent); color: var(--accent-ink); border-color: var(--accent); margin-top: 4px; }
  .backup-actions { display: flex; gap: 10px; margin-bottom: 10px; }
  .file-action { cursor: pointer; }
  .hint.ok { color: var(--ok); }
  .error {
    color: var(--err);
    font-size: 13px;
    margin-bottom: 10px;
  }
  .hint {
    color: var(--text-low);
    font-size: 13px;
    margin-bottom: 10px;
  }

  .nets {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .net-main {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 10px 12px;
    border-radius: var(--radius-ui);
    font-size: 13.5px;
    color: var(--text-hi);
    text-align: left;
  }
  .net-main:hover {
    background: var(--ink-3);
  }
  .ssid {
    flex: 1;
  }
  .tag {
    font-size: 11px;
    color: var(--accent-ink);
    background: var(--accent);
    border-radius: 99px;
    padding: 2px 8px;
    font-weight: 600;
  }
  .sig {
    color: var(--text-mid);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .pw {
    display: flex;
    gap: 8px;
    padding: 4px 12px 10px 37px;
  }
  .pw input {
    flex: 1;
    background: var(--ink-1);
    border: 1px solid var(--line);
    border-radius: 6px;
    padding: 6px 10px;
    outline: none;
  }
  .pw input:focus {
    border-color: var(--accent-dim);
  }

  .field {
    display: flex;
    align-items: center;
    gap: 14px;
    font-size: 13.5px;
  }
  .field input[type='range'] {
    flex: 1;
    max-width: 320px;
    accent-color: var(--accent);
  }
  .val {
    color: var(--text-mid);
    font-variant-numeric: tabular-nums;
  }

  dl {
    display: grid;
    grid-template-columns: 120px 1fr;
    row-gap: 12px;
    font-size: 13.5px;
  }
  dt {
    color: var(--text-mid);
  }

  .modes {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 8px;
    margin-bottom: 14px;
  }
  .mode {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px;
    border-radius: 9px;
    background: var(--ink-2);
    border: 1px solid var(--line-soft);
    text-align: left;
  }
  .mode:hover { border-color: var(--line); }
  .mode.picked { border-color: var(--accent); }
  .mode strong { font-size: 13px; }
  .mode span { font-size: 11px; color: var(--text-low); }
  .f {
    display: flex;
    flex-direction: column;
    gap: 5px;
    font-size: 12px;
    color: var(--text-mid);
    margin-bottom: 12px;
    max-width: 380px;
  }
  .f input {
    background: var(--ink-2);
    border: 1px solid var(--line);
    border-radius: 7px;
    padding: 8px 10px;
    outline: none;
    color: var(--text-hi);
    font-size: 13px;
  }
  .f input:focus { border-color: var(--accent-dim); }
  .f textarea {
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
  .f textarea:focus { border-color: var(--accent-dim); }
  .subhead {
    font-family: var(--font-display);
    font-size: 15px;
    margin: 22px 0 4px;
  }
  .upd-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin: 4px 0 14px;
    max-height: 220px;
    overflow-y: auto;
  }
  .pkg {
    font-family: var(--font-mono);
    font-size: 11.5px;
    background: var(--ink-2);
    border: 1px solid var(--line-soft);
    border-radius: 5px;
    padding: 3px 8px;
    color: var(--text-mid);
  }
</style>
