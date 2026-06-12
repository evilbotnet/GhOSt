<script lang="ts">
  import { api, type DirEntry } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import { wm, type Win } from '../../wm/wm.svelte';
  import { getApp } from '../registry';

  let { win }: { win: Win } = $props();

  let path = $state<string>((win.props.path as string) ?? '');
  let entries = $state<DirEntry[]>([]);
  let error = $state('');
  let selected = $state<string | null>(null);
  let renaming = $state<string | null>(null);
  let renameValue = $state('');
  let history = $state<string[]>([]);

  async function load(p: string, pushHistory = true) {
    try {
      const res = await api.get<{ path: string; entries: DirEntry[] }>(
        `/fs/list?path=${encodeURIComponent(p)}`,
      );
      if (pushHistory && path && path !== res.path) history.push(path);
      path = res.path;
      entries = res.entries;
      error = '';
      selected = null;
      win.title = `Files — ${shortName(res.path)}`;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  async function init() {
    if (path) return load(path, false);
    try {
      const home = await api.get<{ path: string }>('/fs/home');
      await load(home.path, false);
    } catch (e) {
      error = 'File daemon unreachable. Is ghostd running? (scripts/dev.sh)';
    }
  }
  init();

  function shortName(p: string) {
    const parts = p.split('/').filter(Boolean);
    return parts[parts.length - 1] ?? '/';
  }

  let crumbs = $derived.by(() => {
    const parts = path.split('/').filter(Boolean);
    return parts.map((name, i) => ({
      name,
      path: '/' + parts.slice(0, i + 1).join('/'),
    }));
  });

  function up() {
    const parent = path.replace(/\/[^/]+\/?$/, '') || '/';
    load(parent);
  }

  function back() {
    const prev = history.pop();
    if (prev) load(prev, false);
  }

  function open(entry: DirEntry) {
    if (entry.dir) {
      load(entry.path);
    } else if (entry.mime.startsWith('text/') || entry.size < 2 * 1024 * 1024) {
      wm.open(getApp('editor'), { path: entry.path });
    }
  }

  async function newFolder() {
    const base = `${path}/New Folder`;
    let target = base;
    for (let i = 2; entries.some((e) => e.path === target); i++) target = `${base} ${i}`;
    await api.post('/fs/mkdir', { path: target });
    await load(path, false);
    startRename(target.split('/').pop()!);
  }

  function startRename(name: string) {
    renaming = name;
    renameValue = name;
  }

  async function commitRename(entry: DirEntry) {
    renaming = null;
    const to = `${path}/${renameValue.trim()}`;
    if (!renameValue.trim() || to === entry.path) return;
    await api.post('/fs/rename', { from: entry.path, to }).catch((e) => (error = e.message));
    await load(path, false);
  }

  async function trash(entry: DirEntry) {
    await api.post('/fs/trash', { path: entry.path }).catch((e) => (error = e.message));
    await load(path, false);
  }

  function fmtSize(n: number, dir: boolean) {
    if (dir) return '—';
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
    if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
    return `${(n / 1024 ** 3).toFixed(1)} GB`;
  }

  function fmtDate(iso: string) {
    return new Date(iso).toLocaleDateString([], {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function iconFor(entry: DirEntry) {
    if (entry.dir) return 'folder';
    if (entry.mime.startsWith('image/')) return 'image';
    return 'file';
  }
</script>

<div class="files">
  <div class="toolbar">
    <button title="Back" disabled={history.length === 0} onclick={back}>
      <Icon name="arrow-left" size={15} />
    </button>
    <button title="Up" onclick={up}><Icon name="arrow-up" size={15} /></button>
    <button title="Home" onclick={() => init()}><Icon name="home" size={15} /></button>
    <div class="crumbs">
      <button class="crumb" onclick={() => load('/')}>/</button>
      {#each crumbs as crumb (crumb.path)}
        <button class="crumb" onclick={() => load(crumb.path)}>{crumb.name}</button>
      {/each}
    </div>
    <button title="New folder" onclick={newFolder}><Icon name="plus" size={15} /></button>
    <button title="Refresh" onclick={() => load(path, false)}>
      <Icon name="refresh" size={15} />
    </button>
  </div>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  <div class="list" role="listbox" aria-label="Files">
    <div class="head row">
      <span></span><span>Name</span><span>Size</span><span>Modified</span><span></span>
    </div>
    {#each entries as entry (entry.path)}
      <div
        class="row item"
        class:selected={selected === entry.path}
        role="option"
        aria-selected={selected === entry.path}
        tabindex="0"
        onclick={() => (selected = entry.path)}
        ondblclick={() => open(entry)}
        onkeydown={(e) => e.key === 'Enter' && open(entry)}
      >
        <span class="fi" class:dir={entry.dir}><Icon name={iconFor(entry)} size={15} /></span>
        {#if renaming === entry.name}
          <!-- svelte-ignore a11y_autofocus -->
          <input
            class="rename"
            autofocus
            bind:value={renameValue}
            onblur={() => commitRename(entry)}
            onkeydown={(e) => {
              if (e.key === 'Enter') commitRename(entry);
              if (e.key === 'Escape') renaming = null;
            }}
          />
        {:else}
          <span class="name">{entry.name}</span>
        {/if}
        <span class="meta">{fmtSize(entry.size, entry.dir)}</span>
        <span class="meta">{fmtDate(entry.modified)}</span>
        <span class="actions">
          <button title="Rename" onclick={(e) => { e.stopPropagation(); startRename(entry.name); }}>
            <Icon name="editor" size={13} />
          </button>
          <button title="Move to trash" onclick={(e) => { e.stopPropagation(); trash(entry); }}>
            <Icon name="trash" size={13} />
          </button>
        </span>
      </div>
    {/each}
    {#if entries.length === 0 && !error}
      <p class="empty">Empty folder</p>
    {/if}
  </div>
</div>

<style>
  .files {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 6px 8px;
    border-bottom: 1px solid var(--line-soft);
    flex: none;
  }
  .toolbar > button {
    display: grid;
    place-items: center;
    width: 30px;
    height: 28px;
    border-radius: 6px;
    color: var(--text-mid);
  }
  .toolbar > button:hover:not(:disabled) {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .toolbar > button:disabled {
    opacity: 0.35;
  }
  .crumbs {
    flex: 1;
    display: flex;
    align-items: center;
    overflow-x: auto;
    scrollbar-width: none;
    margin: 0 6px;
    padding: 0 4px;
    height: 28px;
    background: var(--ink-2);
    border: 1px solid var(--line-soft);
    border-radius: 6px;
  }
  .crumbs::-webkit-scrollbar {
    display: none;
  }
  .crumb {
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--text-mid);
    font-size: 12.5px;
    white-space: nowrap;
  }
  .crumb:hover {
    color: var(--text-hi);
    background: var(--ink-3);
  }
  .crumb:not(:first-child)::before {
    content: '/';
    margin-right: 8px;
    color: var(--text-low);
  }

  .error {
    padding: 10px 14px;
    color: var(--err);
    font-size: 13px;
  }

  .list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 8px 10px;
  }
  .row {
    display: grid;
    grid-template-columns: 28px 1fr 90px 130px 64px;
    align-items: center;
    gap: 4px;
    min-height: 30px;
    border-radius: 6px;
    font-size: 13px;
  }
  .head {
    color: var(--text-low);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    position: sticky;
    top: 0;
    background: var(--ink-2);
    z-index: 1;
  }
  .item {
    cursor: default;
  }
  .item:hover {
    background: var(--ink-3);
  }
  .item.selected {
    background: var(--ink-4);
  }
  .fi {
    display: grid;
    place-items: center;
    color: var(--text-low);
  }
  .fi.dir {
    color: var(--accent);
  }
  .name {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .rename {
    background: var(--ink-1);
    border: 1px solid var(--accent-dim);
    border-radius: 4px;
    padding: 2px 6px;
    outline: none;
    font-size: 13px;
  }
  .meta {
    color: var(--text-mid);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 2px;
    opacity: 0;
  }
  .item:hover .actions,
  .item.selected .actions {
    opacity: 1;
  }
  .actions button {
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    border-radius: 5px;
    color: var(--text-mid);
  }
  .actions button:hover {
    background: var(--ink-2);
    color: var(--text-hi);
  }
  .empty {
    text-align: center;
    color: var(--text-low);
    padding: 32px 0;
    font-size: 13px;
  }
</style>
