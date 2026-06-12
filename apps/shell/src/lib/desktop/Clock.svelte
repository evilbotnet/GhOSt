<script lang="ts">
  let now = $state(new Date());
  $effect(() => {
    const t = setInterval(() => (now = new Date()), 1000);
    return () => clearInterval(t);
  });
  let time = $derived(
    now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
  );
  let date = $derived(
    now.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' }),
  );
</script>

<div class="clock">
  <span class="time">{time}</span>
  <span class="date">{date}</span>
</div>

<style>
  .clock {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    line-height: 1.15;
    padding: 0 4px;
  }
  .time {
    font-size: 13px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
  .date {
    font-size: 10.5px;
    color: var(--text-mid);
  }
</style>
