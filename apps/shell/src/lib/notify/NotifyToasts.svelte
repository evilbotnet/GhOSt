<script lang="ts">
  // Transient toast stack, bottom-right above the taskbar. Renders
  // notifications.toasts; each auto-clears after 5s (handled in the store) or
  // on clicking the X. CSS-only slide/fade in. Renders nothing when empty.
  import Icon from '../desktop/Icon.svelte';
  import { notifications } from './notify.svelte';

  const ICON: Record<string, string> = {
    info: 'info',
    success: 'info',
    warn: 'info',
    error: 'info',
  };
</script>

{#if notifications.toasts.length}
  <div class="stack" role="region" aria-label="Notifications">
    {#each notifications.toasts as t (t.id)}
      <div class="toast" data-kind={t.kind}>
        <span class="stripe" aria-hidden="true"></span>
        <span class="ico"><Icon name={ICON[t.kind] ?? 'info'} size={16} /></span>
        <div class="body">
          <div class="title">{t.title}</div>
          {#if t.body}<div class="text">{t.body}</div>{/if}
        </div>
        <button class="x" aria-label="Dismiss" onclick={() => notifications.dismissToast(t.id)}>
          <Icon name="close" size={14} />
        </button>
      </div>
    {/each}
  </div>
{/if}

<style>
  .stack {
    position: fixed;
    right: 12px;
    bottom: calc(var(--taskbar-h) + 12px);
    z-index: 7000;
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 320px;
    max-width: calc(100vw - 24px);
    pointer-events: none;
  }
  .toast {
    pointer-events: auto;
    position: relative;
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 11px 12px 11px 14px;
    background: var(--ink-2);
    border-radius: var(--radius-ui);
    box-shadow: var(--shadow-pop);
    overflow: hidden;
    animation: toast-in 160ms ease-out;
  }
  .stripe {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 3px;
    background: var(--accent);
  }
  .toast[data-kind='success'] .stripe { background: var(--ok); }
  .toast[data-kind='warn'] .stripe { background: var(--warn); }
  .toast[data-kind='error'] .stripe { background: var(--err); }

  .ico { color: var(--accent); flex: 0 0 auto; margin-top: 1px; }
  .toast[data-kind='success'] .ico { color: var(--ok); }
  .toast[data-kind='warn'] .ico { color: var(--warn); }
  .toast[data-kind='error'] .ico { color: var(--err); }

  .body { flex: 1; min-width: 0; }
  .title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-hi);
    line-height: 1.3;
  }
  .text {
    margin-top: 2px;
    font-size: 12px;
    color: var(--text-mid);
    line-height: 1.4;
    word-break: break-word;
  }
  .x {
    flex: 0 0 auto;
    color: var(--text-low);
    border-radius: var(--radius-ui);
    padding: 2px;
  }
  .x:hover { color: var(--text-hi); background: var(--ink-3); }

  @keyframes toast-in {
    from { opacity: 0; transform: translateX(12px); }
    to { opacity: 1; transform: translateX(0); }
  }
</style>
