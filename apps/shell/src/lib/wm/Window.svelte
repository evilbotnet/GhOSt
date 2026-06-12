<script lang="ts">
  import { wm, Win } from './wm.svelte';
  import Icon from '../desktop/Icon.svelte';

  let { win }: { win: Win } = $props();

  let focused = $derived(wm.focusedId === win.id);

  type Edge = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw';
  const EDGES: Edge[] = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw'];

  function startDrag(e: PointerEvent) {
    if (win.maximized || e.button !== 0) return;
    const start = { ...win.rect };
    const ox = e.clientX;
    const oy = e.clientY;
    const el = e.currentTarget as HTMLElement;
    el.setPointerCapture(e.pointerId);
    const move = (ev: PointerEvent) => {
      win.rect.x = start.x + (ev.clientX - ox);
      win.rect.y = Math.max(0, start.y + (ev.clientY - oy));
    };
    const up = () => {
      el.removeEventListener('pointermove', move);
      el.removeEventListener('pointerup', up);
    };
    el.addEventListener('pointermove', move);
    el.addEventListener('pointerup', up);
  }

  function startResize(e: PointerEvent, edge: Edge) {
    if (win.maximized || e.button !== 0) return;
    e.stopPropagation();
    const start = { ...win.rect };
    const ox = e.clientX;
    const oy = e.clientY;
    const min = win.app.minSize;
    const el = e.currentTarget as HTMLElement;
    el.setPointerCapture(e.pointerId);
    const move = (ev: PointerEvent) => {
      const dx = ev.clientX - ox;
      const dy = ev.clientY - oy;
      const r = { ...win.rect };
      if (edge.includes('e')) r.w = Math.max(min.w, start.w + dx);
      if (edge.includes('s')) r.h = Math.max(min.h, start.h + dy);
      if (edge.includes('w')) {
        r.w = Math.max(min.w, start.w - dx);
        r.x = start.x + start.w - r.w;
      }
      if (edge.includes('n')) {
        r.h = Math.max(min.h, start.h - dy);
        r.y = start.y + start.h - r.h;
      }
      win.rect = r;
    };
    const up = () => {
      el.removeEventListener('pointermove', move);
      el.removeEventListener('pointerup', up);
    };
    el.addEventListener('pointermove', move);
    el.addEventListener('pointerup', up);
  }

  const AppBody = $derived(win.app.component);
</script>

<section
  class="window"
  class:focused
  class:maximized={win.maximized}
  hidden={win.minimized}
  style:left={win.maximized ? '0' : `${win.rect.x}px`}
  style:top={win.maximized ? '0' : `${win.rect.y}px`}
  style:width={win.maximized ? '100%' : `${win.rect.w}px`}
  style:height={win.maximized ? '100%' : `${win.rect.h}px`}
  style:z-index={win.z}
  onpointerdowncapture={() => wm.focus(win.id)}
>
  <header
    class="titlebar"
    onpointerdown={startDrag}
    ondblclick={() => win.toggleMaximize()}
  >
    <span class="app-icon"><Icon name={win.app.icon} size={14} /></span>
    <span class="title">{win.title}</span>
    <span class="controls" onpointerdown={(e) => e.stopPropagation()}>
      <button aria-label="Minimize" onclick={() => wm.minimize(win.id)}>
        <Icon name="minimize" size={13} />
      </button>
      <button aria-label="Maximize" onclick={() => win.toggleMaximize()}>
        <Icon name={win.maximized ? 'restore' : 'maximize'} size={13} />
      </button>
      <button class="close" aria-label="Close" onclick={() => wm.close(win.id)}>
        <Icon name="close" size={13} />
      </button>
    </span>
  </header>

  <div class="body">
    <AppBody {win} />
  </div>

  {#if !win.maximized}
    {#each EDGES as edge (edge)}
      <span class="grip grip-{edge}" onpointerdown={(e) => startResize(e, edge)}></span>
    {/each}
  {/if}
</section>

<style>
  .window {
    position: absolute;
    display: flex;
    flex-direction: column;
    background: var(--ink-1);
    border-radius: var(--radius-win);
    box-shadow: var(--shadow-win);
    overflow: hidden;
    contain: layout;
  }
  .window.focused {
    box-shadow: var(--shadow-win-focus);
  }
  .window.maximized {
    border-radius: 0;
  }

  .titlebar {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 34px;
    padding: 0 6px 0 12px;
    background: var(--ink-1);
    border-bottom: 1px solid var(--line-soft);
    flex: none;
    touch-action: none;
  }
  .app-icon {
    display: grid;
    place-items: center;
    color: var(--text-low);
  }
  .focused .app-icon {
    color: var(--accent);
  }
  .title {
    flex: 1;
    font-size: 12.5px;
    font-weight: 500;
    letter-spacing: 0.01em;
    color: var(--text-mid);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .focused .title {
    color: var(--text-hi);
  }

  .controls {
    display: flex;
    gap: 2px;
  }
  .controls button {
    display: grid;
    place-items: center;
    width: 30px;
    height: 24px;
    border-radius: 5px;
    color: var(--text-mid);
  }
  .controls button:hover {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .controls button.close:hover {
    background: var(--err);
    color: #fff;
  }

  .body {
    flex: 1;
    min-height: 0;
    background: var(--ink-2);
    display: flex;
    flex-direction: column;
  }

  .grip {
    position: absolute;
    touch-action: none;
  }
  .grip-n { top: -3px; left: 8px; right: 8px; height: 6px; cursor: ns-resize; }
  .grip-s { bottom: -3px; left: 8px; right: 8px; height: 6px; cursor: ns-resize; }
  .grip-e { right: -3px; top: 8px; bottom: 8px; width: 6px; cursor: ew-resize; }
  .grip-w { left: -3px; top: 8px; bottom: 8px; width: 6px; cursor: ew-resize; }
  .grip-ne { top: -3px; right: -3px; width: 12px; height: 12px; cursor: nesw-resize; }
  .grip-nw { top: -3px; left: -3px; width: 12px; height: 12px; cursor: nwse-resize; }
  .grip-se { bottom: -3px; right: -3px; width: 12px; height: 12px; cursor: nwse-resize; }
  .grip-sw { bottom: -3px; left: -3px; width: 12px; height: 12px; cursor: nesw-resize; }
</style>
