<script lang="ts">
  import { api } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win: _win }: { win: Win } = $props();

  // Phase 3 wires this to the local CryptPad instance (socket-activated).
  let officeUrl = $state<string | null>(null);
  let checked = $state(false);

  api
    .get<{ url: string; running: boolean }>('/office/status')
    .then((s) => {
      officeUrl = s.url;
      checked = true;
    })
    .catch(() => (checked = true));
</script>

{#if officeUrl}
  <iframe src={officeUrl} title="CryptPad" class="pad"></iframe>
{:else}
  <div class="placeholder">
    <Icon name="office" size={40} />
    <h2>Office</h2>
    {#if checked}
      <p>
        CryptPad isn't installed on this system yet — it ships with the device
        image (docs/architecture.md, Phase 3).
      </p>
    {:else}
      <p>Checking for local CryptPad…</p>
    {/if}
  </div>
{/if}

<style>
  .pad {
    flex: 1;
    border: none;
    background: #fff;
  }
  .placeholder {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    color: var(--text-low);
    text-align: center;
    padding: 24px;
  }
  h2 {
    font-family: var(--font-display);
    color: var(--text-mid);
    font-size: 20px;
  }
  p {
    font-size: 13px;
    max-width: 380px;
    line-height: 1.5;
  }
</style>
