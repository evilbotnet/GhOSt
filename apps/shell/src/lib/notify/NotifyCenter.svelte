<script lang="ts">
  // Notification history popover. Matches StatusTray's .pop / scrim style.
  // Bindable `open` so the tray can toggle it from a bell button.
  import { notifications } from './notify.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  function rel(ts: number): string {
    const s = Math.max(0, Math.floor((Date.now() - ts) / 1000));
    if (s < 5) return 'just now';
    if (s < 60) return `${s}s ago`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m ago`;
    const h = Math.floor(m / 60);
    if (h < 24) return `${h}h ago`;
    return `${Math.floor(h / 24)}d ago`;
  }
</script>

{#if open}
  <div class="pop">
    <div class="pop-head">
      <span class="host">Notifications</span>
      <div class="acts">
        <button class="link" onclick={() => notifications.markAllRead()}>Mark all read</button>
        <button class="link" onclick={() => notifications.clear()}>Clear</button>
      </div>
    </div>

    {#if notifications.items.length === 0}
      <div class="empty">No notifications</div>
    {:else}
      <div class="list">
        {#each notifications.items as n (n.id)}
          <div class="item" data-kind={n.kind} class:unread={!n.read}>
            <span class="dot" aria-hidden="true"></span>
            <div class="content">
              <div class="row1">
                <span class="title">{n.title}</span>
                <span class="time">{rel(n.ts)}</span>
              </div>
              {#if n.body}<div class="text">{n.body}</div>{/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
  <button class="scrim" aria-label="Close" onclick={() => (open = false)}></button>
{/if}

<style>
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
    width: 320px;
    padding: 8px;
    background: var(--ink-1);
    border-radius: var(--radius-win);
    box-shadow: var(--shadow-pop);
  }
  .pop-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 10px 12px;
    border-bottom: 1px solid var(--line-soft);
    margin-bottom: 6px;
  }
  .host {
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 15px;
  }
  .acts { display: flex; gap: 10px; }
  .link {
    font-size: 11.5px;
    color: var(--text-mid);
  }
  .link:hover { color: var(--accent); }

  .empty {
    padding: 22px 10px;
    text-align: center;
    font-size: 12.5px;
    color: var(--text-low);
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 360px;
    overflow-y: auto;
  }
  .item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 9px 10px;
    border-radius: var(--radius-ui);
  }
  .item:hover { background: var(--ink-3); }
  .item.unread { background: var(--ink-2); }
  .item.unread:hover { background: var(--ink-3); }

  .dot {
    flex: 0 0 auto;
    width: 7px;
    height: 7px;
    margin-top: 5px;
    border-radius: 50%;
    background: var(--accent);
  }
  .item[data-kind='success'] .dot { background: var(--ok); }
  .item[data-kind='warn'] .dot { background: var(--warn); }
  .item[data-kind='error'] .dot { background: var(--err); }
  .item:not(.unread) .dot { opacity: 0.35; }

  .content { flex: 1; min-width: 0; }
  .row1 {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
  }
  .title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-hi);
    line-height: 1.3;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .time {
    flex: 0 0 auto;
    font-size: 11px;
    color: var(--text-low);
    font-variant-numeric: tabular-nums;
  }
  .text {
    margin-top: 2px;
    font-size: 12px;
    color: var(--text-mid);
    line-height: 1.4;
    word-break: break-word;
  }
</style>
