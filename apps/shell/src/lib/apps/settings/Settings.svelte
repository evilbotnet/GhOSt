<script lang="ts">
  import Icon from '../../desktop/Icon.svelte';
  import { api, type WifiNetwork } from '../../api/client';
  import { system } from '../../api/system.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();

  type Panel = 'wifi' | 'sound' | 'ghost' | 'about';
  let panel = $state<Panel>(((win.props.panel as Panel) ?? 'wifi'));

  const PANELS: { id: Panel; name: string; icon: string }[] = [
    { id: 'wifi', name: 'Network', icon: 'wifi' },
    { id: 'sound', name: 'Sound', icon: 'volume' },
    { id: 'ghost', name: 'Ghost AI', icon: 'info' },
    { id: 'about', name: 'About', icon: 'info' },
  ];

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
    }
  });

  async function saveAI() {
    await api.post('/setup/ai', { mode: aiMode, url: aiURL, model: aiModel, key: aiKey });
    aiSaved = true;
    aiConfigured = aiMode !== 'off';
    setTimeout(() => (aiSaved = false), 2000);
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
</style>
