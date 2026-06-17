<script lang="ts">
  import Wallpaper from './Wallpaper.svelte';
  import Taskbar from './Taskbar.svelte';
  import Launcher from './Launcher.svelte';
  import GhostPanel from '../ghost/GhostPanel.svelte';
  import NotifyToasts from '../notify/NotifyToasts.svelte';
  import WindowFrame from '../wm/Window.svelte';
  import { wm, viewport } from '../wm/wm.svelte';
  import { notifications } from '../notify/notify.svelte';

  notifications.start();

  let launcherOpen = $state(false);
  let ghostOpen = $state(false);
  let surface = $state<HTMLElement | null>(null);

  // Keep the WM's notion of the viewport in sync with the window surface.
  $effect(() => {
    if (!surface) return;
    const ro = new ResizeObserver(([entry]) => {
      viewport.w = entry.contentRect.width;
      viewport.h = entry.contentRect.height;
    });
    ro.observe(surface);
    return () => ro.disconnect();
  });

  function onKeydown(e: KeyboardEvent) {
    // Super+Space summons Ghost; bare Super opens the launcher.
    if (e.key === ' ' && e.metaKey) {
      e.preventDefault();
      ghostOpen = !ghostOpen;
      return;
    }
    if (e.key === 'Meta' && !e.repeat) launcherOpen = !launcherOpen;
    if (e.key === 'Escape') {
      if (ghostOpen) ghostOpen = false;
      else if (launcherOpen) launcherOpen = false;
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="desktop">
  <div class="surface" bind:this={surface}>
    <Wallpaper />
    {#each wm.windows as win (win.id)}
      <WindowFrame {win} />
    {/each}
  </div>
  <Launcher bind:open={launcherOpen} />
  <GhostPanel bind:open={ghostOpen} />
  <NotifyToasts />
  <Taskbar bind:launcherOpen bind:ghostOpen />
</div>

<style>
  .desktop {
    position: relative;
    height: 100%;
    overflow: hidden;
  }
  .surface {
    position: absolute;
    inset: 0 0 var(--taskbar-h) 0;
    overflow: hidden;
  }
</style>
