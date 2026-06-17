<script lang="ts">
  import { theme, type Accent } from '../../theme/theme.svelte';

  const ACCENT_NAMES: Record<Accent, string> = {
    copper: 'Copper',
    teal: 'Teal',
    violet: 'Violet',
    green: 'Green',
  };
</script>

<div class="theme">
  <div class="field">
    <span class="label">Appearance</span>
    <div class="seg" role="group" aria-label="Theme mode">
      <button
        class:on={theme.mode === 'dark'}
        aria-pressed={theme.mode === 'dark'}
        onclick={() => theme.setMode('dark')}
      >
        Dark
      </button>
      <button
        class:on={theme.mode === 'light'}
        aria-pressed={theme.mode === 'light'}
        onclick={() => theme.setMode('light')}
      >
        Light
      </button>
    </div>
  </div>

  <div class="field">
    <span class="label">Accent</span>
    <div class="swatches" role="group" aria-label="Accent color">
      {#each theme.accents as a (a)}
        <button
          class="swatch"
          class:on={theme.accent === a}
          style:--sw={theme.swatch(a)}
          aria-pressed={theme.accent === a}
          aria-label={ACCENT_NAMES[a]}
          title={ACCENT_NAMES[a]}
          onclick={() => theme.setAccent(a)}
        ></button>
      {/each}
    </div>
  </div>
</div>

<style>
  .theme {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 380px;
  }
  .field {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .label {
    width: 96px;
    flex: none;
    color: var(--text-mid);
    font-size: 13px;
  }

  /* segmented dark/light toggle */
  .seg {
    display: inline-flex;
    padding: 2px;
    border-radius: var(--radius-ui);
    background: var(--ink-2);
    border: 1px solid var(--line-soft);
  }
  .seg button {
    padding: 5px 14px;
    border-radius: 5px;
    color: var(--text-mid);
    font-size: 12.5px;
    line-height: 1;
  }
  .seg button:hover {
    color: var(--text-hi);
  }
  .seg button.on {
    background: var(--ink-4);
    color: var(--text-hi);
  }

  /* accent swatches */
  .swatches {
    display: inline-flex;
    gap: 10px;
  }
  .swatch {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: var(--sw);
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.25) inset;
    outline: 2px solid transparent;
    outline-offset: 2px;
    transition: outline-color 0.12s ease;
  }
  .swatch:hover {
    outline-color: var(--line);
  }
  .swatch.on {
    outline-color: var(--sw);
  }
</style>
