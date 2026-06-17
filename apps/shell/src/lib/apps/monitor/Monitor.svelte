<script lang="ts">
  import { api } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win: _win }: { win: Win } = $props();

  interface ProcInfo {
    pid: number;
    name: string;
    cpu: number;
    memMB: number;
  }
  interface Metrics {
    cpuPercent: number;
    memUsedMB: number;
    memTotalMB: number;
    diskUsedGB: number;
    diskTotalGB: number;
    uptime: string;
    load: string;
    processes: ProcInfo[];
  }

  let metrics = $state<Metrics | null>(null);
  let error = $state('');

  async function poll() {
    try {
      metrics = await api.get<Metrics>('/system/metrics');
      error = '';
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  $effect(() => {
    void poll();
    const id = setInterval(poll, 2000);
    return () => clearInterval(id);
  });

  function pct(used: number, total: number): number {
    if (!total) return 0;
    return Math.max(0, Math.min(100, (used / total) * 100));
  }

  const memPct = $derived(metrics ? pct(metrics.memUsedMB, metrics.memTotalMB) : 0);
  const diskPct = $derived(metrics ? pct(metrics.diskUsedGB, metrics.diskTotalGB) : 0);

  function fmtMB(mb: number): string {
    if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
    return `${Math.round(mb)} MB`;
  }
</script>

<div class="monitor">
  {#if metrics === null}
    <p class="waiting">
      <Icon name="refresh" size={14} />
      {error ? error : 'waiting for ghostd…'}
    </p>
  {:else}
    {#if error}
      <p class="stale">stale — {error}</p>
    {/if}

    <div class="cards">
      <div class="card">
        <div class="card-top">
          <span class="label">CPU</span>
          <span class="value">{metrics.cpuPercent.toFixed(0)}%</span>
        </div>
        <div class="track"><div class="fill" style="width:{metrics.cpuPercent}%"></div></div>
        <span class="sub">overall load</span>
      </div>

      <div class="card">
        <div class="card-top">
          <span class="label">Memory</span>
          <span class="value">{memPct.toFixed(0)}%</span>
        </div>
        <div class="track"><div class="fill" style="width:{memPct}%"></div></div>
        <span class="sub">{fmtMB(metrics.memUsedMB)} / {fmtMB(metrics.memTotalMB)}</span>
      </div>

      <div class="card">
        <div class="card-top">
          <span class="label">Disk</span>
          <span class="value">{diskPct.toFixed(0)}%</span>
        </div>
        <div class="track"><div class="fill" style="width:{diskPct}%"></div></div>
        <span class="sub">{metrics.diskUsedGB.toFixed(1)} / {metrics.diskTotalGB.toFixed(1)} GB</span>
      </div>
    </div>

    <div class="meta-row">
      <div class="meta">
        <span class="meta-label">Uptime</span>
        <span class="meta-val">{metrics.uptime}</span>
      </div>
      <div class="meta">
        <span class="meta-label">Load avg</span>
        <span class="meta-val mono">{metrics.load}</span>
      </div>
    </div>

    <div class="proc">
      <div class="proc-head">Top processes</div>
      <div class="table">
        <div class="row head">
          <span>PID</span>
          <span>Name</span>
          <span class="num">CPU %</span>
          <span class="num">Mem MB</span>
        </div>
        {#each metrics.processes as p (p.pid)}
          <div class="row">
            <span class="mono dim">{p.pid}</span>
            <span class="name">{p.name}</span>
            <span class="mono num">{p.cpu.toFixed(1)}</span>
            <span class="mono num">{Math.round(p.memMB)}</span>
          </div>
        {/each}
        {#if metrics.processes.length === 0}
          <p class="empty">No process data</p>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .monitor {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 16px 18px;
    gap: 16px;
    overflow-y: auto;
  }

  .waiting {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-low);
    font-size: 13px;
    padding: 24px 4px;
  }
  .stale {
    color: var(--warn);
    font-size: 12px;
    margin: -4px 0 0;
  }

  .cards {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    flex: none;
  }
  .card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 14px;
    background: var(--ink-2);
    border: 1px solid var(--line-soft);
    border-radius: 9px;
  }
  .card-top {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
  }
  .label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-low);
  }
  .value {
    font-family: var(--font-display);
    font-size: 22px;
    font-weight: 600;
    color: var(--accent);
    font-variant-numeric: tabular-nums;
  }
  .track {
    height: 6px;
    border-radius: 99px;
    background: var(--ink-3);
    overflow: hidden;
  }
  .fill {
    height: 100%;
    border-radius: 99px;
    background: var(--accent);
    transition: width 0.4s ease;
  }
  .sub {
    font-size: 12px;
    color: var(--text-mid);
    font-variant-numeric: tabular-nums;
  }

  .meta-row {
    display: flex;
    gap: 28px;
    padding: 0 2px;
    flex: none;
  }
  .meta {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .meta-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-low);
  }
  .meta-val {
    font-size: 13.5px;
    color: var(--text-hi);
  }
  .meta-val.mono {
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }

  .proc {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .proc-head {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-low);
    margin-bottom: 6px;
  }
  .table {
    border: 1px solid var(--line-soft);
    border-radius: 9px;
    overflow: hidden;
  }
  .row {
    display: grid;
    grid-template-columns: 64px 1fr 72px 80px;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    font-size: 13px;
  }
  .row:not(.head):not(:last-child) {
    border-bottom: 1px solid var(--line-soft);
  }
  .row.head {
    background: var(--ink-2);
    color: var(--text-low);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--line-soft);
  }
  .num {
    text-align: right;
  }
  .mono {
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .dim {
    color: var(--text-mid);
  }
  .name {
    color: var(--text-hi);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .empty {
    text-align: center;
    color: var(--text-low);
    padding: 20px 0;
    font-size: 13px;
  }
</style>
