<script lang="ts">
  import { system } from '../../api/system.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win: _win }: { win: Win } = $props();
  system.start();
</script>

<div class="about">
  <svg viewBox="0 0 100 100" width="84" height="84" aria-hidden="true">
    <circle cx="50" cy="50" r="34" fill="none" stroke="var(--text-mid)" stroke-width="3" />
    <circle cx="50" cy="50" r="44" fill="none" stroke="var(--text-low)" stroke-width="1" stroke-dasharray="4 7" />
    <circle cx="50" cy="16" r="6" fill="var(--accent)" />
  </svg>
  <h1>GhOSt</h1>
  <p class="tag">Go · html · Operating System · typescript</p>
  <p class="ver">
    shell 0.1.0 · {system.status.platform} · daemon
    {system.online ? 'connected' : 'offline'}
  </p>
  <p class="foot">Linux + Wayland + Chromium + the web. No overlords — just a ghost in the shell.</p>
</div>

<style>
  .about {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    text-align: center;
    padding: 20px;
  }
  h1 {
    font-family: var(--font-display);
    font-size: 30px;
    font-weight: 700;
    letter-spacing: 0.02em;
    margin-top: 10px;
  }
  .tag {
    color: var(--text-mid);
    font-size: 14px;
  }
  .ver {
    color: var(--text-low);
    font-size: 12px;
    font-family: var(--font-mono);
    margin-top: 10px;
  }
  .foot {
    color: var(--text-low);
    font-size: 12px;
    margin-top: 18px;
  }
</style>
