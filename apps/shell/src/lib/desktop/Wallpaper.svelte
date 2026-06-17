<script lang="ts">
  // Procedural wallpaper: layered radial gradients + SVG grain, zero assets.
</script>

<div class="wallpaper">
  <div class="grain"></div>
  <div class="mark" aria-hidden="true">
    <svg viewBox="0 0 100 100" width="160" height="160">
      <circle cx="50" cy="50" r="34" fill="none" stroke="currentColor" stroke-width="2.5" />
      <circle cx="50" cy="50" r="44" fill="none" stroke="currentColor" stroke-width="1" opacity="0.45" stroke-dasharray="4 7" />
      <circle cx="50" cy="16" r="4.5" fill="currentColor" />
    </svg>
    <span>GhOSt</span>
  </div>
</div>

<style>
  .wallpaper {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(110% 90% at 78% 110%, rgba(224, 153, 84, 0.16) 0%, transparent 55%),
      radial-gradient(90% 70% at 12% -10%, rgba(58, 90, 110, 0.35) 0%, transparent 60%),
      radial-gradient(60% 50% at 60% 40%, rgba(31, 41, 55, 0.5) 0%, transparent 70%),
      linear-gradient(160deg, #0d1118 0%, #0b0e13 45%, #0e1015 100%);
  }
  /* Light theme: warm paper with the same copper/slate tints, but gentle. */
  :global([data-theme='light']) .wallpaper {
    background:
      radial-gradient(110% 90% at 78% 110%, rgba(224, 153, 84, 0.18) 0%, transparent 55%),
      radial-gradient(90% 70% at 12% -10%, rgba(58, 90, 110, 0.12) 0%, transparent 60%),
      linear-gradient(160deg, #f3efe8 0%, #efece6 50%, #e9e4da 100%);
  }
  :global([data-theme='light']) .grain {
    opacity: 0.03;
  }
  .grain {
    position: absolute;
    inset: 0;
    opacity: 0.05;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/%3E%3C/filter%3E%3Crect width='160' height='160' filter='url(%23n)'/%3E%3C/svg%3E");
  }
  .mark {
    position: absolute;
    right: 48px;
    bottom: 78px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    color: var(--text-low);
    opacity: 0.35;
  }
  .mark span {
    font-family: var(--font-display);
    font-size: 22px;
    font-weight: 600;
    letter-spacing: 0.04em;
  }
</style>
