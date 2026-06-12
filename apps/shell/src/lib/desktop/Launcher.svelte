<script lang="ts">
  import Icon from './Icon.svelte';
  import { wm } from '../wm/wm.svelte';
  import { apps, openBrowser } from '../apps/registry';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  let query = $state('');
  let searchEl = $state<HTMLInputElement | null>(null);

  $effect(() => {
    if (open) {
      query = '';
      searchEl?.focus();
    }
  });

  interface Entry {
    id: string;
    name: string;
    icon: string;
    run: () => void;
  }

  let entries = $derived.by<Entry[]>(() => {
    const all: Entry[] = [
      {
        id: 'browser',
        name: 'Browser',
        icon: 'browser',
        run: () => openBrowser(),
      },
      ...apps.map((a) => ({
        id: a.id,
        name: a.name,
        icon: a.icon,
        run: () => wm.open(a),
      })),
    ];
    const q = query.trim().toLowerCase();
    return q ? all.filter((e) => e.name.toLowerCase().includes(q)) : all;
  });

  function launch(e: Entry) {
    open = false;
    e.run();
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false;
    if (e.key === 'Enter' && entries.length > 0) launch(entries[0]);
  }
</script>

{#if open}
  <div class="launcher" role="dialog" aria-label="App launcher">
    <button class="scrim" aria-label="Close launcher" onclick={() => (open = false)}></button>
    <div class="panel">
      <div class="search">
        <Icon name="search" size={15} />
        <input
          bind:this={searchEl}
          bind:value={query}
          placeholder="Search apps"
          onkeydown={onKeydown}
        />
      </div>
      <div class="grid">
        {#each entries as entry (entry.id)}
          <button class="app" onclick={() => launch(entry)}>
            <span class="badge"><Icon name={entry.icon} size={24} /></span>
            <span class="name">{entry.name}</span>
          </button>
        {/each}
        {#if entries.length === 0}
          <p class="empty">Nothing matches “{query}”</p>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .launcher {
    position: absolute;
    inset: 0;
    z-index: 6000;
  }
  .scrim {
    position: absolute;
    inset: 0;
    background: rgba(8, 10, 14, 0.6);
    cursor: default;
  }
  .panel {
    position: absolute;
    left: 10px;
    bottom: calc(var(--taskbar-h) + 10px);
    width: 460px;
    max-width: calc(100vw - 20px);
    padding: 14px;
    background: var(--ink-1);
    border-radius: 14px;
    box-shadow: var(--shadow-pop);
    animation: rise 140ms cubic-bezier(0.2, 0.9, 0.3, 1);
  }
  @keyframes rise {
    from {
      transform: translateY(8px);
      opacity: 0;
    }
  }

  .search {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 0 12px;
    height: 40px;
    background: var(--ink-2);
    border: 1px solid var(--line-soft);
    border-radius: var(--radius-ui);
    color: var(--text-low);
    margin-bottom: 12px;
  }
  .search:focus-within {
    border-color: var(--accent-dim);
  }
  .search input {
    flex: 1;
    background: none;
    border: none;
    outline: none;
    color: var(--text-hi);
    font-size: 14px;
  }
  .search input::placeholder {
    color: var(--text-low);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 4px;
  }
  .app {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 9px;
    padding: 14px 6px 12px;
    border-radius: 10px;
  }
  .app:hover {
    background: var(--ink-3);
  }
  .badge {
    display: grid;
    place-items: center;
    width: 46px;
    height: 46px;
    border-radius: 13px;
    background: var(--ink-3);
    border: 1px solid var(--line-soft);
    color: var(--accent);
  }
  .app:hover .badge {
    border-color: var(--accent-dim);
  }
  .name {
    font-size: 12px;
    color: var(--text-mid);
  }
  .app:hover .name {
    color: var(--text-hi);
  }
  .empty {
    grid-column: 1 / -1;
    text-align: center;
    color: var(--text-low);
    padding: 18px 0;
    font-size: 13px;
  }
</style>
