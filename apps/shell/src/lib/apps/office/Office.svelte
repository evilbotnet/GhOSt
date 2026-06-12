<script lang="ts">
  import { api } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import { wm, type Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();

  interface OfficeStatus {
    available: boolean;
    url: string;
    running: boolean;
  }

  // CryptPad's CSP (frame-ancestors 'self') forbids iframing it from the
  // shell origin, so Office opens it as a native chromeless app window.
  // This in-shell window is just the launcher/progress surface.
  let state = $state<'checking' | 'unavailable' | 'starting' | 'launched'>('checking');

  async function launch() {
    await api.post('/office/launch').catch(() => {});
    state = 'launched';
    // The native window has its own taskbar entry; this launcher is done.
    setTimeout(() => wm.close(win.id), 1200);
  }

  $effect(() => {
    let stop = false;

    const poll = async () => {
      while (!stop) {
        const s = await api.get<OfficeStatus>('/office/status').catch(() => null);
        if (stop) return;
        if (!s || !s.available) {
          state = 'unavailable';
          return;
        }
        if (s.running) {
          await launch();
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

<div class="placeholder">
  <Icon name="office" size={40} />
  <h2>Office</h2>
  {#if state === 'starting' || state === 'checking'}
    <p>Starting CryptPad…</p>
  {:else if state === 'launched'}
    <p>CryptPad is open in its own window.</p>
  {:else}
    <p>
      CryptPad isn't installed on this system yet — it ships with the device
      image (see os/vm/install-cryptpad.sh).
    </p>
  {/if}
</div>

<style>
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
