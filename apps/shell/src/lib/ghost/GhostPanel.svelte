<script lang="ts">
  import Icon from '../desktop/Icon.svelte';
  import { ghost } from '../api/ghost.svelte';
  import { wm } from '../wm/wm.svelte';
  import { getApp } from '../apps/registry';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  let input = $state('');
  let inputEl = $state<HTMLInputElement | null>(null);
  let scroller = $state<HTMLElement | null>(null);

  ghost.start();

  $effect(() => {
    if (open) {
      ghost.start();
      queueMicrotask(() => inputEl?.focus());
    }
  });

  // autoscroll on new entries / thinking
  $effect(() => {
    void ghost.entries.length;
    void ghost.thinking;
    void ghost.confirm;
    if (scroller) queueMicrotask(() => (scroller!.scrollTop = scroller!.scrollHeight));
  });

  function submit() {
    if (!input.trim()) return;
    ghost.ask(input);
    input = '';
  }

  function openSettings() {
    open = false;
    const app = getApp('settings');
    const win = wm.open(app);
    (win.props as Record<string, unknown>).panel = 'ghost';
  }

  function fmtArgs(args: Record<string, unknown>): string {
    return Object.entries(args)
      .map(([k, v]) => `${k}: ${typeof v === 'string' && v.length > 60 ? v.slice(0, 60) + '…' : v}`)
      .join('\n');
  }

  let extOpen = $state(false);
  let extCount = $derived(ghost.skills.length + ghost.tools.length);
</script>

{#if open}
  <div class="scrim" onclick={() => (open = false)} role="presentation"></div>
  <aside class="ghost" role="dialog" aria-label="Ghost assistant">
    <header>
      <span class="brand">
        <svg viewBox="0 0 100 100" width="18" height="18" aria-hidden="true">
          <circle cx="50" cy="50" r="34" fill="none" stroke="currentColor" stroke-width="7" />
          <circle cx="50" cy="16" r="11" fill="var(--accent)" stroke="var(--ink-1)" stroke-width="4" />
        </svg>
        Ghost
      </span>
      {#if ghost.provenance}
        <span class="prov" title="Which model answered">{ghost.provenance}</span>
      {/if}
      <button class="x" aria-label="Close" onclick={() => (open = false)}>
        <Icon name="close" size={14} />
      </button>
    </header>

    <div class="stream" bind:this={scroller}>
      {#if !ghost.configured}
        <div class="empty">
          <Icon name="info" size={26} />
          <p>Ghost isn't awake yet.</p>
          <p class="dim">
            Point it at a model — your own Ollama/vLLM endpoint or the Anthropic
            API — and it can act on this machine through the OS itself.
          </p>
          <button class="cfg" onclick={openSettings}>Configure Ghost</button>
        </div>
      {:else if ghost.entries.length === 0}
        <div class="empty">
          <p class="dim">Ask Ghost to do something.</p>
          <ul class="suggest">
            <li>"organize my Downloads into folders by type"</li>
            <li>"what's taking up space in my home folder?"</li>
            <li>"open the GhOSt repo on github"</li>
          </ul>
        </div>
      {/if}

      {#each ghost.entries as e, i (i)}
        {#if e.kind === 'user'}
          <div class="row user"><p>{e.text}</p></div>
        {:else if e.kind === 'message'}
          <div class="row ghost-msg"><p>{e.text}</p></div>
        {:else if e.kind === 'tool'}
          <div class="row tool">
            <Icon name="terminal" size={12} />
            <span class="tname">{e.tool}</span>
            {#if e.error}<span class="terr">{e.error}</span>
            {:else if e.output === '…'}<span class="tdim">running…</span>
            {:else}<span class="tok">done</span>{/if}
          </div>
        {:else if e.kind === 'denied'}
          <div class="row tool"><span class="tdim">declined {e.tool}</span></div>
        {:else if e.kind === 'error'}
          <div class="row err"><p>{e.text}</p></div>
        {/if}
      {/each}

      {#if ghost.thinking}
        <div class="row ghost-msg"><span class="dots"><i></i><i></i><i></i></span></div>
      {/if}

      {#if ghost.confirm}
        <div class="confirm">
          <p class="ctitle">Ghost wants to <strong>{ghost.confirm.name.replace(/_/g, ' ')}</strong></p>
          <pre>{fmtArgs(ghost.confirm.args)}</pre>
          <div class="cbtns">
            <button class="deny" onclick={() => ghost.decide(ghost.confirm!.callId, false)}>Deny</button>
            <button class="allow" onclick={() => ghost.decide(ghost.confirm!.callId, true)}>Allow</button>
          </div>
        </div>
      {/if}
    </div>

    {#if ghost.configured && extCount > 0}
      <div class="ext">
        <button class="ext-toggle" onclick={() => (extOpen = !extOpen)}>
          <Icon name="settings" size={12} />
          {ghost.skills.length} skill{ghost.skills.length === 1 ? '' : 's'} ·
          {ghost.tools.length} tool{ghost.tools.length === 1 ? '' : 's'}
          <Icon name={extOpen ? 'chevron-up' : 'plus'} size={11} />
        </button>
        {#if extOpen}
          <div class="ext-list">
            {#each ghost.skills as sk (sk.name)}
              <div class="ext-row" title={sk.description}>
                <span class="ext-kind skill">skill</span>
                <span class="ext-name">{sk.name}</span>
              </div>
            {/each}
            {#each ghost.tools as t (t.name)}
              <div class="ext-row" title={t.description}>
                <span class="ext-kind tool" class:mut={t.mutating}>tool</span>
                <span class="ext-name">{t.name}</span>
              </div>
            {/each}
            <p class="ext-hint">Drop skills in ~/.config/ghost/skills, tools in ~/.config/ghost/tools</p>
          </div>
        {/if}
      </div>
    {/if}

    <form class="ask" onsubmit={(e) => { e.preventDefault(); submit(); }}>
      <input
        bind:this={inputEl}
        bind:value={input}
        placeholder={ghost.configured ? 'Ask Ghost…' : 'Configure Ghost first'}
        disabled={!ghost.configured}
      />
      <button type="submit" aria-label="Send" disabled={!ghost.configured || !input.trim()}>
        <Icon name="arrow-up" size={16} />
      </button>
    </form>
  </aside>
{/if}

<style>
  .scrim {
    position: absolute;
    inset: 0;
    z-index: 6500;
  }
  .ghost {
    position: absolute;
    top: 0;
    right: 0;
    bottom: var(--taskbar-h);
    z-index: 6600;
    width: 380px;
    max-width: 100vw;
    display: flex;
    flex-direction: column;
    background: var(--ink-1);
    border-left: 1px solid var(--line-soft);
    box-shadow: -8px 0 28px rgba(0, 0, 0, 0.4);
    animation: slide 160ms cubic-bezier(0.2, 0.9, 0.3, 1);
  }
  @keyframes slide { from { transform: translateX(20px); opacity: 0; } }

  header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 12px 10px 14px;
    border-bottom: 1px solid var(--line-soft);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    font-family: var(--font-display);
    font-weight: 650;
    font-size: 15px;
    color: var(--accent);
    flex: 1;
  }
  .prov {
    font-family: var(--font-mono);
    font-size: 10.5px;
    color: var(--text-low);
    background: var(--ink-2);
    padding: 2px 7px;
    border-radius: 5px;
  }
  .x {
    display: grid;
    place-items: center;
    width: 26px;
    height: 26px;
    border-radius: 6px;
    color: var(--text-mid);
  }
  .x:hover { background: var(--ink-3); color: var(--text-hi); }

  .stream {
    flex: 1;
    overflow-y: auto;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .empty {
    margin: auto;
    text-align: center;
    color: var(--text-mid);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 20px;
  }
  .empty p { font-size: 13px; }
  .dim { color: var(--text-low); font-size: 12px; line-height: 1.5; }
  .suggest {
    list-style: none;
    text-align: left;
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .suggest li {
    font-size: 12px;
    color: var(--text-mid);
    background: var(--ink-2);
    padding: 7px 10px;
    border-radius: 7px;
  }
  .cfg {
    margin-top: 10px;
    padding: 8px 16px;
    border-radius: 8px;
    background: var(--accent);
    color: var(--accent-ink);
    font-weight: 600;
    font-size: 13px;
  }
  .cfg:hover { background: var(--accent-bright); }

  .row { font-size: 13.5px; line-height: 1.5; }
  .row.user {
    align-self: flex-end;
    max-width: 85%;
    background: var(--ink-3);
    padding: 8px 12px;
    border-radius: 12px 12px 3px 12px;
  }
  .row.ghost-msg { max-width: 92%; color: var(--text-hi); }
  .row.ghost-msg p { white-space: pre-wrap; }
  .row.tool {
    display: flex;
    align-items: center;
    gap: 7px;
    font-family: var(--font-mono);
    font-size: 11.5px;
    color: var(--text-mid);
    background: var(--ink-2);
    padding: 5px 9px;
    border-radius: 6px;
  }
  .tname { color: var(--text-hi); }
  .tok { color: var(--ok); margin-left: auto; }
  .terr { color: var(--err); margin-left: auto; }
  .tdim { color: var(--text-low); margin-left: auto; }
  .row.err {
    color: var(--err);
    font-size: 12.5px;
    background: rgba(212, 101, 79, 0.1);
    padding: 8px 12px;
    border-radius: 8px;
  }

  .dots { display: inline-flex; gap: 4px; padding: 4px 0; }
  .dots i {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--text-low);
    animation: blink 1.2s infinite;
  }
  .dots i:nth-child(2) { animation-delay: 0.2s; }
  .dots i:nth-child(3) { animation-delay: 0.4s; }
  @keyframes blink { 0%, 60%, 100% { opacity: 0.3; } 30% { opacity: 1; } }

  .confirm {
    border: 1px solid var(--accent-dim);
    border-radius: 10px;
    padding: 12px;
    background: var(--ink-2);
  }
  .ctitle { font-size: 13px; margin-bottom: 8px; }
  .confirm pre {
    font-family: var(--font-mono);
    font-size: 11.5px;
    color: var(--text-mid);
    background: var(--ink-1);
    padding: 8px 10px;
    border-radius: 6px;
    white-space: pre-wrap;
    word-break: break-word;
    max-height: 120px;
    overflow-y: auto;
  }
  .cbtns { display: flex; gap: 8px; margin-top: 10px; }
  .cbtns button {
    flex: 1;
    padding: 8px;
    border-radius: 7px;
    font-weight: 600;
    font-size: 13px;
  }
  .allow { background: var(--accent); color: var(--accent-ink); }
  .allow:hover { background: var(--accent-bright); }
  .deny { background: var(--ink-3); color: var(--text-hi); }
  .deny:hover { background: var(--ink-4); }

  .ext {
    border-top: 1px solid var(--line-soft);
  }
  .ext-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 8px 14px;
    font-size: 11.5px;
    color: var(--text-low);
  }
  .ext-toggle:hover {
    color: var(--text-mid);
    background: var(--ink-2);
  }
  .ext-list {
    padding: 4px 12px 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 160px;
    overflow-y: auto;
  }
  .ext-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }
  .ext-kind {
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 1px 5px;
    border-radius: 4px;
    color: var(--ink-0);
    background: var(--text-low);
  }
  .ext-kind.skill {
    background: var(--accent);
    color: var(--accent-ink);
  }
  .ext-kind.tool {
    background: var(--text-mid);
  }
  .ext-kind.tool.mut {
    background: var(--warn);
    color: var(--accent-ink);
  }
  .ext-name {
    font-family: var(--font-mono);
    color: var(--text-hi);
  }
  .ext-hint {
    font-size: 10.5px;
    color: var(--text-low);
    margin-top: 4px;
  }

  .ask {
    display: flex;
    gap: 8px;
    padding: 12px;
    border-top: 1px solid var(--line-soft);
  }
  .ask input {
    flex: 1;
    background: var(--ink-2);
    border: 1px solid var(--line);
    border-radius: 9px;
    padding: 10px 12px;
    outline: none;
    color: var(--text-hi);
    font-size: 13.5px;
  }
  .ask input:focus { border-color: var(--accent-dim); }
  .ask button {
    display: grid;
    place-items: center;
    width: 40px;
    border-radius: 9px;
    background: var(--accent);
    color: var(--accent-ink);
  }
  .ask button:disabled { opacity: 0.4; }
  .ask button:not(:disabled):hover { background: var(--accent-bright); }
</style>
