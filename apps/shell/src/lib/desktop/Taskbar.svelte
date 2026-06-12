<script lang="ts">
  import Icon from './Icon.svelte';
  import StatusTray from './StatusTray.svelte';
  import { wm } from '../wm/wm.svelte';
  import { openBrowser } from '../apps/registry';

  let { launcherOpen = $bindable(false) }: { launcherOpen?: boolean } = $props();
</script>

<nav class="taskbar">
  <button
    class="launch"
    class:active={launcherOpen}
    aria-label="Launcher"
    onclick={() => (launcherOpen = !launcherOpen)}
  >
    <svg viewBox="0 0 100 100" width="20" height="20" aria-hidden="true">
      <circle cx="50" cy="50" r="34" fill="none" stroke="currentColor" stroke-width="9" />
      <circle cx="50" cy="16" r="11" fill="var(--accent)" stroke="var(--ink-1)" stroke-width="5" />
    </svg>
  </button>

  <button class="quick" aria-label="Browser" title="Browser" onclick={() => openBrowser()}>
    <Icon name="browser" size={17} />
  </button>

  <div class="sep"></div>

  <div class="running">
    {#each wm.windows as win (win.id)}
      <button
        class="task"
        class:focused={wm.focusedId === win.id && !win.minimized}
        class:min={win.minimized}
        title={win.title}
        onclick={() => wm.toggleFromTaskbar(win.id)}
      >
        <Icon name={win.app.icon} size={16} />
        <span class="lamp"></span>
      </button>
    {/each}
  </div>

  <StatusTray />
</nav>

<style>
  .taskbar {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: var(--taskbar-h);
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 8px;
    background: var(--ink-1);
    border-top: 1px solid var(--line-soft);
    z-index: 5000;
  }

  .launch {
    display: grid;
    place-items: center;
    width: 38px;
    height: 34px;
    border-radius: var(--radius-ui);
    color: var(--text-mid);
  }
  .launch:hover,
  .launch.active {
    background: var(--ink-3);
    color: var(--text-hi);
  }

  .quick {
    display: grid;
    place-items: center;
    width: 36px;
    height: 34px;
    border-radius: var(--radius-ui);
    color: var(--text-mid);
  }
  .quick:hover {
    background: var(--ink-3);
    color: var(--text-hi);
  }

  .sep {
    width: 1px;
    height: 20px;
    background: var(--line);
    margin: 0 4px;
  }

  .running {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 2px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .running::-webkit-scrollbar {
    display: none;
  }

  .task {
    position: relative;
    display: grid;
    place-items: center;
    width: 40px;
    height: 34px;
    border-radius: var(--radius-ui);
    color: var(--text-mid);
    flex: none;
  }
  .task:hover {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .task.focused {
    background: var(--ink-3);
    color: var(--text-hi);
  }
  .lamp {
    position: absolute;
    bottom: 2px;
    width: 12px;
    height: 2.5px;
    border-radius: 2px;
    background: var(--text-low);
  }
  .task.focused .lamp {
    background: var(--accent);
  }
  .task.min .lamp {
    width: 5px;
  }
</style>
