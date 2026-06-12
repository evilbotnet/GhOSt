<script lang="ts">
  import Icon from './Icon.svelte';
  import Clock from './Clock.svelte';
  import { api } from '../api/client';
  import { system } from '../api/system.svelte';
  import { wm } from '../wm/wm.svelte';
  import { getApp } from '../apps/registry';

  system.start();
  let s = $derived(system.status);
  let popOpen = $state(false);

  function openSettings(panel: string) {
    popOpen = false;
    const app = getApp('settings');
    const win = wm.open(app);
    (win.props as Record<string, unknown>).panel = panel;
  }

  let shotMsg = $state('');
  async function screenshot() {
    popOpen = false;
    try {
      const res = await api.post<{ path: string }>('/system/screenshot');
      shotMsg = `Saved ${res.path.split('/').pop()}`;
    } catch {
      shotMsg = 'Screenshot failed';
    }
    setTimeout(() => (shotMsg = ''), 2500);
  }
</script>

<div class="tray">
  {#if shotMsg}<span class="toast">{shotMsg}</span>{/if}
  <button class="status" class:active={popOpen} onclick={() => (popOpen = !popOpen)}>
    {#if s.wifi.available}
      <span class:dim={!s.wifi.connected}><Icon name="wifi" size={15} /></span>
    {/if}
    <span class:dim={s.volume.muted}><Icon name="volume" size={15} /></span>
    {#if s.battery.available}
      <span><Icon name="battery" size={15} /></span>
    {/if}
    <Clock />
  </button>

  {#if popOpen}
    <div class="pop">
      <div class="pop-head">
        <span class="host">{s.hostname}</span>
        <span class="net">
          {s.wifi.connected ? s.wifi.ssid : system.online ? 'wired / no wifi' : 'daemon offline'}
        </span>
      </div>
      <button class="row" onclick={() => openSettings('wifi')}>
        <Icon name="wifi" size={15} />
        <span>Network</span>
        <span class="val">{s.wifi.connected ? `${s.wifi.signal}%` : '—'}</span>
      </button>
      <div class="row">
        <Icon name="volume" size={15} />
        <input
          type="range"
          min="0"
          max="100"
          value={s.volume.percent}
          oninput={(e) => system.setVolume(Number(e.currentTarget.value))}
        />
        <span class="val">{s.volume.percent}%</span>
      </div>
      <button class="row" onclick={screenshot}>
        <Icon name="camera" size={15} />
        <span>Screenshot</span>
      </button>
      <button class="row" onclick={() => openSettings('about')}>
        <Icon name="settings" size={15} />
        <span>Settings</span>
      </button>
    </div>
    <button class="scrim" aria-label="Close" onclick={() => (popOpen = false)}></button>
  {/if}
</div>

<style>
  .tray {
    position: relative;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .toast {
    font-size: 11.5px;
    color: var(--accent);
    white-space: nowrap;
  }
  .status {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 34px;
    padding: 0 10px;
    border-radius: var(--radius-ui);
    color: var(--text-mid);
  }
  .status:hover,
  .status.active {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .dim {
    opacity: 0.4;
  }

  .scrim {
    position: fixed;
    inset: 0;
    z-index: 9998;
    cursor: default;
  }
  .pop {
    position: absolute;
    right: 0;
    bottom: 44px;
    z-index: 9999;
    width: 280px;
    padding: 8px;
    background: var(--ink-1);
    border-radius: var(--radius-win);
    box-shadow: var(--shadow-pop);
  }
  .pop-head {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px 10px 12px;
    border-bottom: 1px solid var(--line-soft);
    margin-bottom: 6px;
  }
  .host {
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 15px;
  }
  .net {
    font-size: 11.5px;
    color: var(--text-mid);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 9px 10px;
    border-radius: var(--radius-ui);
    color: var(--text-hi);
    font-size: 13px;
    text-align: left;
  }
  button.row:hover {
    background: var(--ink-3);
  }
  .row .val {
    margin-left: auto;
    color: var(--text-mid);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .row span:not(.val) {
    color: var(--text-hi);
  }
  input[type='range'] {
    flex: 1;
    accent-color: var(--accent);
  }
</style>
