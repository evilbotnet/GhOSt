<script lang="ts">
  // Renders an installed .osapp inside a sandboxed iframe (ADR 0009 isolation).
  // The `sandbox` attribute WITHOUT allow-same-origin gives the app a unique
  // opaque origin, so it cannot reach the shell's window or read its session
  // token — and the shell cannot read into it. This is belt-and-suspenders with
  // the daemon's `Content-Security-Policy: sandbox …` header on /apps/<id>/,
  // which enforces the same opaque origin even if this attribute were missing.
  import type { Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();
  const appId = String(win.props.appId ?? '');
  // Served by ghostd at /apps/<id>/ — same path in dev (Vite proxies it) and
  // in production (single origin on :7700).
  const src = `/apps/${appId}/`;
</script>

<iframe
  title={win.title}
  {src}
  sandbox="allow-scripts allow-forms allow-modals"
  referrerpolicy="no-referrer"
></iframe>

<style>
  iframe {
    width: 100%;
    height: 100%;
    border: 0;
    background: var(--surface, #fff);
  }
</style>
