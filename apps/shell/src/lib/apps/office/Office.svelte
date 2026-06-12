<script lang="ts">
  import { api } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win: _win }: { win: Win } = $props();

  interface OfficeStatus {
    available: boolean;
    url: string;
    running: boolean;
  }

  let state = $state<'checking' | 'unavailable' | 'starting' | 'ready'>('checking');
  let url = $state('');

  $effect(() => {
    let stop = false;

    const poll = async () => {
      // CryptPad is started on demand and takes a few seconds cold.
      while (!stop) {
        const s = await api.get<OfficeStatus>('/office/status').catch(() => null);
        if (stop) return;
        if (!s || !s.available) {
          state = 'unavailable';
          return;
        }
        if (s.running) {
          url = s.url;
          state = 'ready';
          return;
        }
        state = 'starting';
        await new Promise((r) => setTimeout(r, 1500));
      }
    };

    api.post('/office/open').catch(() => {});
    void poll();

    return () => {
      stop = true;
      api.post('/office/close').catch(() => {});
    };
  });
</script>

{#if state === 'ready'}
  <iframe src={url} title="CryptPad" class="pad"></iframe>
{:else}
  <div class="placeholder">
    <Icon name="office" size={40} />
    <h2>Office</h2>
    {#if state === 'starting'}
      <p>Starting CryptPad…</p>
    {:else if state === 'unavailable'}
      <p>
        CryptPad isn't installed on this system yet — it ships with the device
        image (see os/vm/install-cryptpad.sh).
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
