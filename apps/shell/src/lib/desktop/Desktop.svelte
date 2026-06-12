<script lang="ts">
  import Wallpaper from './Wallpaper.svelte';
  import Taskbar from './Taskbar.svelte';
  import Launcher from './Launcher.svelte';
  import WindowFrame from '../wm/Window.svelte';
  import { wm, viewport } from '../wm/wm.svelte';

  let launcherOpen = $state(false);
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
    // Meta/Super opens the launcher, like every desktop since 1995.
    if (e.key === 'Meta' && !e.repeat) launcherOpen = !launcherOpen;
    if (e.key === 'Escape' && launcherOpen) launcherOpen = false;
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
  <Taskbar bind:launcherOpen />
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
