<script lang="ts">
  import Desktop from './lib/desktop/Desktop.svelte';
  import Oobe from './lib/oobe/Oobe.svelte';
  import { api } from './lib/api/client';

  // First boot: the wizard owns the screen until setup completes.
  // Dev override: open with #oobe to walk the wizard on any host.
  let oobe = $state(false);
  if (location.hash.includes('oobe')) {
    oobe = true;
  } else {
    api
      .get<{ needed: boolean }>('/setup/status')
      .then((s) => (oobe = s.needed))
      .catch(() => {});
  }
</script>

<Desktop />
{#if oobe}
  <Oobe onDone={() => (oobe = false)} />
{/if}
