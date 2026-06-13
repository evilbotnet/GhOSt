<script lang="ts">
  import Icon from '../desktop/Icon.svelte';
  import { api, type WifiNetwork } from '../api/client';

  let { onDone }: { onDone: () => void } = $props();

  type Step = 'welcome' | 'password' | 'timezone' | 'wifi' | 'ghost' | 'done';
  const STEPS: Step[] = ['welcome', 'password', 'timezone', 'wifi', 'ghost', 'done'];
  let step = $state<Step>('welcome');
  let stepIdx = $derived(STEPS.indexOf(step));
  let busy = $state(false);
  let error = $state('');

  function next() {
    error = '';
    step = STEPS[Math.min(stepIdx + 1, STEPS.length - 1)];
  }
  function back() {
    error = '';
    step = STEPS[Math.max(stepIdx - 1, 0)];
  }

  // --- password ---
  let pw = $state('');
  let pw2 = $state('');
  async function submitPassword() {
    if (pw.length < 4) return (error = 'Use at least 4 characters.');
    if (pw !== pw2) return (error = "Passwords don't match.");
    busy = true;
    try {
      await api.post('/setup/password', { password: pw });
      next();
    } catch (e) {
      degradeOrShow(e);
    } finally {
      busy = false;
    }
  }

  // Dev hosts (no root helper) skip privileged steps instead of blocking.
  function degradeOrShow(e: unknown) {
    const msg = e instanceof Error ? e.message : 'failed';
    if (msg.includes('unavailable')) next();
    else error = msg;
  }

  // --- timezone ---
  let timezones = $state<string[]>([]);
  let tzQuery = $state('');
  let tzPick = $state(Intl.DateTimeFormat().resolvedOptions().timeZone ?? 'UTC');
  let tzMatches = $derived(
    tzQuery.trim()
      ? timezones.filter((t) => t.toLowerCase().includes(tzQuery.trim().toLowerCase())).slice(0, 8)
      : [],
  );
  $effect(() => {
    if (step === 'timezone' && timezones.length === 0) {
      api.get<string[]>('/setup/timezones').then((l) => (timezones = l)).catch(() => {});
    }
  });
  async function submitTimezone() {
    busy = true;
    try {
      await api.post('/setup/timezone', { timezone: tzPick });
      next();
    } catch (e) {
      degradeOrShow(e);
    } finally {
      busy = false;
    }
  }

  // --- wifi ---
  let nets = $state<WifiNetwork[]>([]);
  let wifiPick = $state<WifiNetwork | null>(null);
  let wifiPw = $state('');
  let wifiOK = $state(false);
  $effect(() => {
    if (step === 'wifi') {
      api.get<WifiNetwork[]>('/system/wifi/networks').then((l) => (nets = l)).catch(() => {});
    }
  });
  async function joinWifi() {
    if (!wifiPick) return;
    busy = true;
    error = '';
    try {
      await api.post('/system/wifi/connect', {
        ssid: wifiPick.ssid,
        password: wifiPw || undefined,
      });
      wifiOK = true;
      next();
    } catch (e) {
      error = e instanceof Error ? e.message : 'failed to join';
    } finally {
      busy = false;
    }
  }

  // --- ghost ai ---
  type AIMode = 'off' | 'lan' | 'cloud';
  let aiMode = $state<AIMode>('off');
  let aiURL = $state('http://192.168.1.10:11434/v1');
  let aiModel = $state('');
  let aiKey = $state('');
  async function submitAI() {
    busy = true;
    try {
      await api.post('/setup/ai', { mode: aiMode, url: aiURL, model: aiModel, key: aiKey });
      next();
    } catch (e) {
      error = e instanceof Error ? e.message : 'failed';
    } finally {
      busy = false;
    }
  }

  async function finish() {
    busy = true;
    try {
      await api.post('/setup/complete');
      onDone();
    } catch {
      onDone();
    }
  }
</script>

<div class="oobe">
  <div class="card">
    {#if step === 'welcome'}
      <div class="hero">
        <svg viewBox="0 0 100 100" width="110" height="110" aria-hidden="true">
          <circle cx="50" cy="50" r="34" fill="none" stroke="var(--text-mid)" stroke-width="2.5" />
          <circle cx="50" cy="50" r="44" fill="none" stroke="var(--text-low)" stroke-width="1" stroke-dasharray="4 7" class="spin" />
          <circle cx="50" cy="16" r="5.5" fill="var(--accent)" />
        </svg>
        <h1>GhOSt</h1>
        <p class="acronym">Go · html · Operating System · typescript</p>
        <p class="lead">
          A web-native operating system with no overlords — and, soon, a ghost
          in the shell. A few questions and this machine is yours.
        </p>
      </div>
    {:else if step === 'password'}
      <h2>Make it yours</h2>
      <p class="sub">
        This password protects your account (<code>ghost</code>) and unlocks
        <code>sudo</code> in the Terminal — root stays locked, you hold the key.
      </p>
      <label>Password
        <input type="password" bind:value={pw} autocomplete="new-password" />
      </label>
      <label>Confirm
        <input type="password" bind:value={pw2}
          onkeydown={(e) => e.key === 'Enter' && submitPassword()} />
      </label>
    {:else if step === 'timezone'}
      <h2>Where in the world?</h2>
      <p class="sub">Currently <strong>{tzPick}</strong></p>
      <label>Search timezones
        <input bind:value={tzQuery} placeholder="berlin, denver, tokyo…" />
      </label>
      <div class="list">
        {#each tzMatches as tz (tz)}
          <button class="row" class:picked={tz === tzPick} onclick={() => { tzPick = tz; tzQuery = ''; }}>
            {tz}
          </button>
        {/each}
      </div>
    {:else if step === 'wifi'}
      <h2>Get online</h2>
      {#if nets.length === 0}
        <p class="sub">No Wi-Fi networks found (wired or no radio) — skip ahead.</p>
      {:else}
        <div class="list tall">
          {#each nets as net (net.ssid)}
            <button class="row" class:picked={wifiPick?.ssid === net.ssid} onclick={() => (wifiPick = net)}>
              <Icon name="wifi" size={14} />
              <span class="grow">{net.ssid}</span>
              {#if net.active}<span class="ok">connected</span>{/if}
              {#if net.secured}<Icon name="lock" size={12} />{/if}
              <span class="dim">{net.signal}%</span>
            </button>
          {/each}
        </div>
        {#if wifiPick?.secured}
          <label>Password for {wifiPick.ssid}
            <input type="password" bind:value={wifiPw}
              onkeydown={(e) => e.key === 'Enter' && joinWifi()} />
          </label>
        {/if}
      {/if}
    {:else if step === 'ghost'}
      <h2>Wake the ghost?</h2>
      <p class="sub">
        Ghost is the resident AI — its tools are this OS itself. Off by
        default; everything is configurable later in Settings. Your keys never
        leave this device except to the endpoint you choose.
      </p>
      <div class="modes">
        <button class="mode" class:picked={aiMode === 'off'} onclick={() => (aiMode = 'off')}>
          <strong>Not yet</strong><span>no AI, fully offline</span>
        </button>
        <button class="mode" class:picked={aiMode === 'lan'} onclick={() => (aiMode = 'lan')}>
          <strong>My own model</strong><span>Ollama / vLLM / llama.cpp endpoint</span>
        </button>
        <button class="mode" class:picked={aiMode === 'cloud'} onclick={() => (aiMode = 'cloud')}>
          <strong>Anthropic API</strong><span>bring your own key</span>
        </button>
      </div>
      {#if aiMode === 'lan'}
        <label>OpenAI-compatible endpoint
          <input bind:value={aiURL} placeholder="http://host:11434/v1" />
        </label>
        <label>Model
          <input bind:value={aiModel} placeholder="qwen3:8b" />
        </label>
      {:else if aiMode === 'cloud'}
        <label>API key
          <input type="password" bind:value={aiKey} placeholder="sk-ant-…" />
        </label>
        <label>Model
          <input bind:value={aiModel} placeholder="claude-opus-4-8" />
        </label>
      {/if}
    {:else}
      <div class="hero">
        <h1>It's yours now.</h1>
        <p class="lead">
          Launcher is bottom-left (or the <kbd>⌘/Super</kbd> key). The Terminal
          is a real shell — <code>sudo</code> works. Settings has the rest.
        </p>
      </div>
    {/if}

    {#if error}<p class="error">{error}</p>{/if}

    <div class="nav">
      {#if stepIdx > 0 && step !== 'done'}
        <button class="ghost-btn" onclick={back}>Back</button>
      {/if}
      <span class="dots">
        {#each STEPS as s, i (s)}
          <span class="dot" class:on={i <= stepIdx}></span>
        {/each}
      </span>
      {#if step === 'welcome'}
        <button class="cta" onclick={next}>Begin</button>
      {:else if step === 'password'}
        <button class="cta" disabled={busy} onclick={submitPassword}>
          {busy ? 'Setting…' : 'Set password'}
        </button>
      {:else if step === 'timezone'}
        <button class="cta" disabled={busy} onclick={submitTimezone}>
          {busy ? 'Saving…' : 'Continue'}
        </button>
      {:else if step === 'wifi'}
        {#if wifiPick && !wifiOK}
          <button class="cta" disabled={busy} onclick={joinWifi}>
            {busy ? 'Joining…' : `Join ${wifiPick.ssid}`}
          </button>
        {:else}
          <button class="cta" onclick={next}>Continue</button>
        {/if}
      {:else if step === 'ghost'}
        <button class="cta" disabled={busy} onclick={submitAI}>
          {busy ? 'Saving…' : aiMode === 'off' ? 'Skip for now' : 'Save'}
        </button>
      {:else}
        <button class="cta" disabled={busy} onclick={finish}>Enter GhOSt</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .oobe {
    position: absolute;
    inset: 0;
    z-index: 9000;
    display: grid;
    place-items: center;
    background:
      radial-gradient(110% 90% at 78% 110%, rgba(224, 153, 84, 0.14) 0%, transparent 55%),
      radial-gradient(90% 70% at 12% -10%, rgba(58, 90, 110, 0.3) 0%, transparent 60%),
      var(--ink-0);
  }
  .card {
    width: 560px;
    max-width: calc(100vw - 48px);
    min-height: 430px;
    display: flex;
    flex-direction: column;
    padding: 36px 40px 24px;
    background: var(--ink-1);
    border: 1px solid var(--line-soft);
    border-radius: 18px;
    box-shadow: var(--shadow-pop);
  }
  .hero {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    gap: 8px;
  }
  .spin {
    transform-origin: 50% 50%;
    animation: rot 40s linear infinite;
  }
  @keyframes rot { to { transform: rotate(360deg); } }
  h1 {
    font-family: var(--font-display);
    font-size: 38px;
    font-weight: 700;
    letter-spacing: 0.02em;
  }
  .acronym {
    color: var(--accent);
    font-size: 13px;
    letter-spacing: 0.04em;
  }
  .lead {
    color: var(--text-mid);
    font-size: 14px;
    line-height: 1.6;
    max-width: 400px;
    margin-top: 8px;
  }
  h2 {
    font-family: var(--font-display);
    font-size: 24px;
    font-weight: 650;
    margin-bottom: 6px;
  }
  .sub {
    color: var(--text-mid);
    font-size: 13px;
    line-height: 1.55;
    margin-bottom: 16px;
  }
  code, kbd {
    font-family: var(--font-mono);
    font-size: 12px;
    background: var(--ink-3);
    padding: 1px 5px;
    border-radius: 4px;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 12.5px;
    color: var(--text-mid);
    margin-bottom: 12px;
  }
  input {
    background: var(--ink-2);
    border: 1px solid var(--line);
    border-radius: 7px;
    padding: 9px 12px;
    outline: none;
    color: var(--text-hi);
    font-size: 14px;
  }
  input:focus {
    border-color: var(--accent-dim);
  }
  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow-y: auto;
    max-height: 130px;
  }
  .list.tall { max-height: 190px; }
  .row {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 8px 12px;
    border-radius: 7px;
    color: var(--text-hi);
    font-size: 13px;
    text-align: left;
  }
  .row:hover { background: var(--ink-3); }
  .row.picked { background: var(--ink-3); outline: 1px solid var(--accent-dim); }
  .grow { flex: 1; }
  .dim { color: var(--text-low); font-size: 12px; }
  .ok { color: var(--ok); font-size: 11.5px; }

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
  .mode span { font-size: 11px; color: var(--text-low); line-height: 1.4; }

  .error { color: var(--err); font-size: 12.5px; margin-top: 4px; }

  .nav {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-top: auto;
    padding-top: 18px;
  }
  .dots { flex: 1; display: flex; gap: 6px; justify-content: center; }
  .dot {
    width: 7px; height: 7px; border-radius: 50%;
    background: var(--ink-4);
  }
  .dot.on { background: var(--accent-dim); }
  .cta {
    padding: 9px 22px;
    border-radius: 8px;
    background: var(--accent);
    color: var(--accent-ink);
    font-weight: 600;
    font-size: 13.5px;
  }
  .cta:hover:not(:disabled) { background: var(--accent-bright); }
  .cta:disabled { opacity: 0.6; }
  .ghost-btn {
    padding: 9px 14px;
    border-radius: 8px;
    color: var(--text-mid);
    font-size: 13px;
  }
  .ghost-btn:hover { background: var(--ink-3); color: var(--text-hi); }
</style>
