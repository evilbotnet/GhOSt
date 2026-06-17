<script lang="ts">
  import { api, getToken, type DirEntry } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();

  let path = $state<string>((win.props.path as string) ?? '');
  let siblings = $state<DirEntry[]>([]);
  let error = $state('');
  let imgError = $state(false);

  const IMAGE_RE = /\.(jpe?g|png|gif|webp|svg|avif|bmp|ico)$/i;
  const PDF_RE = /\.pdf$/i;

  function isViewable(p: string) {
    return IMAGE_RE.test(p) || PDF_RE.test(p);
  }

  let isPdf = $derived(PDF_RE.test(path));
  let fileName = $derived(path ? (path.split('/').pop() ?? '') : '');
  let src = $derived(
    path ? `/api/v1/fs/raw?path=${encodeURIComponent(path)}&token=${getToken() ?? ''}` : '',
  );

  let index = $derived(siblings.findIndex((e) => e.path === path));
  let count = $derived(siblings.length);

  // Reload the media when the path changes (clear any prior image error).
  $effect(() => {
    void path;
    imgError = false;
  });

  // Keep the window title in sync with the current file.
  $effect(() => {
    win.title = fileName ? `${fileName} — Viewer` : 'Viewer';
  });

  function parentOf(p: string) {
    return p.replace(/\/[^/]+\/?$/, '') || '/';
  }

  async function loadSiblings() {
    if (!path) return;
    try {
      const res = await api.get<{ path: string; entries: DirEntry[] }>(
        `/fs/list?path=${encodeURIComponent(parentOf(path))}`,
      );
      siblings = res.entries.filter((e) => !e.dir && isViewable(e.path));
      error = '';
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }
  loadSiblings();

  function go(delta: number) {
    if (count < 2 || index < 0) return;
    const next = (index + delta + count) % count;
    path = siblings[next].path;
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'ArrowLeft') {
      go(-1);
      e.preventDefault();
    } else if (e.key === 'ArrowRight') {
      go(1);
      e.preventDefault();
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="viewer">
  <div class="bar">
    <span class="fi"><Icon name={isPdf ? 'file' : 'image'} size={15} /></span>
    <span class="name" title={path}>{fileName || 'No file'}</span>
    {#if count > 0 && index >= 0}
      <span class="pos">{index + 1} / {count}</span>
    {/if}
    <div class="spacer"></div>
    {#if count > 1}
      <button title="Previous (←)" onclick={() => go(-1)}>
        <Icon name="arrow-left" size={15} />
      </button>
      <button class="next" title="Next (→)" onclick={() => go(1)}>
        <Icon name="arrow-left" size={15} />
      </button>
    {/if}
  </div>

  <div class="stage" class:checker={!isPdf}>
    {#if !path}
      <div class="empty">
        <Icon name="image" size={40} />
        <p>No file to display</p>
      </div>
    {:else if error}
      <div class="empty">
        <Icon name="info" size={40} />
        <p>{error}</p>
      </div>
    {:else if isPdf}
      <iframe src={src} title="pdf"></iframe>
    {:else if imgError}
      <div class="empty">
        <Icon name="info" size={40} />
        <p>Could not load image</p>
      </div>
    {:else}
      <img src={src} alt={fileName} onerror={() => (imgError = true)} />
    {/if}
  </div>
</div>

<style>
  .viewer {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--ink-1);
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-bottom: 1px solid var(--line-soft);
    flex: none;
  }
  .fi {
    display: grid;
    place-items: center;
    color: var(--text-low);
    flex: none;
  }
  .name {
    font-size: 12.5px;
    color: var(--text-hi);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    max-width: 50%;
  }
  .pos {
    font-size: 12px;
    color: var(--text-mid);
    font-variant-numeric: tabular-nums;
    flex: none;
  }
  .spacer {
    flex: 1;
  }
  .bar button {
    display: grid;
    place-items: center;
    width: 30px;
    height: 28px;
    border-radius: 6px;
    color: var(--text-mid);
    flex: none;
  }
  .bar button:hover {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .bar button.next :global(svg) {
    transform: rotate(180deg);
  }

  .stage {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    background: var(--ink-0);
  }
  .stage.checker {
    /* subtle dark checkerboard so transparent PNGs read clearly */
    background-color: var(--ink-1);
    background-image:
      linear-gradient(45deg, var(--ink-0) 25%, transparent 25%),
      linear-gradient(-45deg, var(--ink-0) 25%, transparent 25%),
      linear-gradient(45deg, transparent 75%, var(--ink-0) 75%),
      linear-gradient(-45deg, transparent 75%, var(--ink-0) 75%);
    background-size: 24px 24px;
    background-position:
      0 0,
      0 12px,
      12px -12px,
      -12px 0;
  }
  img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
    display: block;
  }
  iframe {
    width: 100%;
    height: 100%;
    border: none;
    background: var(--ink-2);
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--text-low);
  }
  .empty p {
    font-size: 13px;
    margin: 0;
    text-align: center;
    max-width: 80%;
  }
</style>
