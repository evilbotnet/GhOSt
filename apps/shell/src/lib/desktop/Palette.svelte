<script lang="ts">
  // The unified command palette (ADR 0002): one box for apps + commands + Ghost.
  // Super+Space. Type to filter installed apps (built-in, web, .osapp) and open
  // one instantly, or hit Enter on free text to hand it to Ghost — whose daemon
  // router then picks the command/agent tier. Apps open locally (no round-trip);
  // anything else routes through Ghost.
  import Icon from './Icon.svelte';
  import { apps as builtinApps, openOSApp } from '../apps/registry';
  import { webApps } from '../api/webapps.svelte';
  import { wm } from '../wm/wm.svelte';
  import { api } from '../api/client';

  let {
    open = $bindable(false),
    onAskGhost,
  }: { open?: boolean; onAskGhost: (q: string) => void } = $props();

  interface Item {
    key: string;
    name: string;
    icon: string;
    badge: string;
    run: () => void;
  }

  let query = $state('');
  let selected = $state(0);
  let inputEl = $state<HTMLInputElement | null>(null);
  let osApps = $state<{ id: string; name: string; icon?: string }[]>([]);

  webApps.start();

  $effect(() => {
    if (open) {
      query = '';
      selected = 0;
      api.get<{ id: string; name: string; icon?: string }[]>('/osapps').then((v) => (osApps = v)).catch(() => {});
      queueMicrotask(() => inputEl?.focus());
    }
  });

  const allApps = $derived<Item[]>([
    ...builtinApps.map((a) => ({
      key: `b:${a.id}`,
      name: a.name,
      icon: a.icon,
      badge: 'App',
      run: () => wm.open(a),
    })),
    ...webApps.list.map((a) => ({
      key: `w:${a.id}`,
      name: a.name,
      icon: a.icon || 'browser',
      badge: 'Web',
      run: () => webApps.launch(a.id),
    })),
    ...osApps.map((a) => ({
      key: `o:${a.id}`,
      name: a.name,
      icon: a.icon || 'launcher',
      badge: 'Package',
      run: () => openOSApp(a.id, a.name, a.icon || 'launcher'),
    })),
  ]);

  const results = $derived.by<Item[]>(() => {
    const q = query.trim().toLowerCase();
    const matched = q ? allApps.filter((a) => a.name.toLowerCase().includes(q)) : allApps;
    const items = [...matched];
    if (q) {
      items.push({
        key: 'ask',
        name: `Ask Ghost: “${query.trim()}”`,
        icon: 'launcher',
        badge: 'Ghost',
        run: () => onAskGhost(query.trim()),
      });
    }
    return items;
  });

  // Keep the selection in range as results change.
  $effect(() => {
    if (selected >= results.length) selected = Math.max(0, results.length - 1);
  });

  function choose(item: Item) {
    item.run();
    open = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      selected = Math.min(selected + 1, results.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selected = Math.max(selected - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (results[selected]) choose(results[selected]);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      open = false;
    }
  }
</script>

{#if open}
  <div class="scrim" onclick={() => (open = false)} role="presentation"></div>
  <div class="palette" role="dialog" aria-label="Command palette">
    <div class="box">
      <Icon name="launcher" size={17} />
      <input
        bind:this={inputEl}
        bind:value={query}
        onkeydown={onKeydown}
        placeholder="Search apps or ask Ghost…"
        aria-label="Search apps or ask Ghost"
        autocomplete="off"
        spellcheck="false"
      />
    </div>
    <ul class="results">
      {#each results as item, i (item.key)}
        <li>
          <button
            class:active={i === selected}
            onmousemove={() => (selected = i)}
            onclick={() => choose(item)}
          >
            <Icon name={item.icon} size={16} />
            <span class="name">{item.name}</span>
            <span class="badge" class:ghost={item.badge === 'Ghost'}>{item.badge}</span>
          </button>
        </li>
      {/each}
      {#if results.length === 0}
        <li class="empty">No matches — type a request and press Enter to ask Ghost.</li>
      {/if}
    </ul>
  </div>
{/if}

<style>
  .scrim {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.35);
    z-index: 60;
  }
  .palette {
    position: fixed;
    top: 12vh;
    left: 50%;
    transform: translateX(-50%);
    width: min(560px, 92vw);
    z-index: 61;
    background: var(--ink-1);
    border: 1px solid var(--line);
    border-radius: 12px;
    box-shadow: 0 0 0 1px var(--line-soft), 0 18px 50px rgba(0, 0, 0, 0.55);
    overflow: hidden;
  }
  .box {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 13px 16px;
    border-bottom: 1px solid var(--line-soft);
    color: var(--accent);
  }
  .box input {
    flex: 1;
    background: none;
    border: 0;
    outline: none;
    color: var(--text-hi);
    font-size: 15px;
    font-family: var(--font-ui);
  }
  .results {
    list-style: none;
    margin: 0;
    padding: 6px;
    max-height: 50vh;
    overflow-y: auto;
  }
  .results button {
    display: flex;
    align-items: center;
    gap: 11px;
    width: 100%;
    padding: 9px 10px;
    border-radius: 8px;
    color: var(--text-mid);
    text-align: left;
    background: none;
  }
  .results button.active {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .name {
    flex: 1;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .badge {
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--text-low);
    color: var(--ink-0);
  }
  .badge.ghost {
    background: var(--accent);
    color: var(--accent-ink);
  }
  .empty {
    padding: 18px 12px;
    text-align: center;
    color: var(--text-low);
    font-size: 13px;
  }
</style>
