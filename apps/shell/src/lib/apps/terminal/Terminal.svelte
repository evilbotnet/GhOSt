<script lang="ts">
  import { Terminal as Xterm } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';
  import { api } from '../../api/client';
  import { subscribe, send } from '../../api/ws';
  import type { Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();

  let host = $state<HTMLElement | null>(null);
  let status = $state<'connecting' | 'live' | 'dead'>('connecting');

  $effect(() => {
    if (!host) return;

    const term = new Xterm({
      fontFamily: 'JetBrains Mono, monospace',
      fontSize: 13,
      lineHeight: 1.25,
      scrollback: 2000, // capped: Pi memory discipline
      cursorBlink: true,
      theme: {
        background: '#11151c',
        foreground: '#e8e6e0',
        cursor: '#e09954',
        cursorAccent: '#11151c',
        selectionBackground: '#2a3140',
        black: '#171c25',
        red: '#d4654f',
        green: '#7fb069',
        yellow: '#e09954',
        blue: '#7793b5',
        magenta: '#b58dae',
        cyan: '#7fb0a8',
        white: '#e8e6e0',
        brightBlack: '#5d6470',
        brightRed: '#e08572',
        brightGreen: '#9bc78a',
        brightYellow: '#f0b576',
        brightBlue: '#94aecf',
        brightMagenta: '#cfa8c8',
        brightCyan: '#9ccac2',
        brightWhite: '#ffffff',
      },
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    fit.fit();

    let sessionId = '';
    let unsub = () => {};
    let disposed = false;

    api
      .post<{ id: string }>('/term', { cols: term.cols, rows: term.rows })
      .then(({ id }) => {
        if (disposed) return;
        sessionId = id;
        win.title = `Terminal — ${id.slice(0, 6)}`;
        unsub = subscribe(`term.${id}`, (env) => {
          if (env.event === 'data') {
            term.write(env.payload as string);
            status = 'live';
          } else if (env.event === 'exit') {
            status = 'dead';
            term.write('\r\n\x1b[2m[process exited]\x1b[0m\r\n');
          }
        });
        status = 'live';
        term.focus();
      })
      .catch(() => {
        status = 'dead';
        term.write('\x1b[31mghostd daemon unreachable — run scripts/dev.sh\x1b[0m\r\n');
      });

    term.onData((data) => {
      if (sessionId) send(`term.${sessionId}`, 'input', data);
    });

    const ro = new ResizeObserver(() => {
      fit.fit();
      if (sessionId) {
        send(`term.${sessionId}`, 'resize', { cols: term.cols, rows: term.rows });
      }
    });
    ro.observe(host);

    return () => {
      disposed = true;
      ro.disconnect();
      unsub();
      if (sessionId) api.del(`/term/${sessionId}`).catch(() => {});
      term.dispose();
    };
  });
</script>

<div class="term-wrap" class:dead={status === 'dead'}>
  <div class="term" bind:this={host}></div>
</div>

<style>
  .term-wrap {
    flex: 1;
    min-height: 0;
    background: #11151c;
    padding: 6px 2px 2px 8px;
  }
  .term-wrap.dead {
    opacity: 0.7;
  }
  .term {
    height: 100%;
  }
  .term :global(.xterm) {
    height: 100%;
  }
</style>
