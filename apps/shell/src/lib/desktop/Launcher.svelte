<script lang="ts">
  import Icon from './Icon.svelte';
  import { wm } from '../wm/wm.svelte';
  import { apps, openBrowser } from '../apps/registry';
  import { webApps } from '../api/webapps.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  let query = $state('');
  let searchEl = $state<HTMLInputElement | null>(null);
  let installing = $state(false);
  let installName = $state('');
  let installURL = $state('');

  webApps.start();

  $effect(() => {
    if (open) {
      query = '';
      installing = false;
      void webApps.refresh();
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
      ...webApps.list.map((w) => ({
        id: w.id,
        name: w.name,
        icon: w.icon,
        run: () => webApps.launch(w.id),
      })),
    ];
    const q = query.trim().toLowerCase();
    return q ? all.filter((e) => e.name.toLowerCase().includes(q)) : all;
  });

  async function doInstall() {
    if (!installURL.trim()) return;
    let url = installURL.trim();
    if (!/^https?:\/\//.test(url)) url = 'https://' + url;
    await webApps.install(installName.trim(), url);
    installName = '';
    installURL = '';
    installing = false;
  }

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
        {#if !query}
          <button class="app add" onclick={() => (installing = true)}>
            <span class="badge"><Icon name="plus" size={24} /></span>
            <span class="name">Install web app</span>
          </button>
        {/if}
        {#if entries.length === 0 && query}
          <p class="empty">Nothing matches “{query}”</p>
        {/if}
      </div>

      {#if installing}
        <form class="install" onsubmit={(e) => { e.preventDefault(); doInstall(); }}>
          <input bind:value={installURL} placeholder="app URL (e.g. excalidraw.com)" />
          <input bind:value={installName} placeholder="name (optional)" />
          <button type="submit" class="go">Install</button>
        </form>
      {/if}
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
  .app.add .badge {
    background: transparent;
    border-style: dashed;
    color: var(--text-low);
  }
  .install {
    display: flex;
    gap: 8px;
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid var(--line-soft);
  }
  .install input {
    flex: 1;
    background: var(--ink-2);
    border: 1px solid var(--line);
    border-radius: 7px;
    padding: 8px 10px;
    outline: none;
    color: var(--text-hi);
    font-size: 12.5px;
  }
  .install input:focus { border-color: var(--accent-dim); }
  .install .go {
    padding: 8px 14px;
    border-radius: 7px;
    background: var(--accent);
    color: var(--accent-ink);
    font-weight: 600;
    font-size: 12.5px;
  }
</style>
