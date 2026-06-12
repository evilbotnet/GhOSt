<script lang="ts">
  import { untrack } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState, Compartment } from '@codemirror/state';
  import { keymap } from '@codemirror/view';
  import { javascript } from '@codemirror/lang-javascript';
  import { json } from '@codemirror/lang-json';
  import { markdown } from '@codemirror/lang-markdown';
  import { api } from '../../api/client';
  import Icon from '../../desktop/Icon.svelte';
  import type { Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();

  let host = $state<HTMLElement | null>(null);
  let path = $state<string>((win.props.path as string) ?? '');
  let dirty = $state(false);
  let message = $state('');
  let view: EditorView | null = null;
  const langConf = new Compartment();

  function langFor(p: string) {
    if (/\.(ts|tsx|js|jsx|mjs)$/.test(p)) return javascript({ typescript: true });
    if (/\.json$/.test(p)) return json();
    if (/\.(md|markdown)$/.test(p)) return markdown();
    return [];
  }

  function refreshTitle() {
    const name = path ? path.split('/').pop() : 'untitled';
    win.title = `${dirty ? '● ' : ''}${name} — Editor`;
  }

  async function save() {
    if (!path) {
      const home = await api.get<{ path: string }>('/fs/home').catch(() => null);
      const suggested = home ? `${home.path}/untitled.txt` : '';
      const entered = prompt('Save as (full path):', suggested);
      if (!entered) return;
      path = entered;
      view?.dispatch({ effects: langConf.reconfigure(langFor(path)) });
    }
    try {
      await api.put('/fs/write', { path, content: view?.state.doc.toString() ?? '' });
      dirty = false;
      message = 'saved';
      setTimeout(() => (message = ''), 1500);
      refreshTitle();
    } catch (e) {
      message = e instanceof Error ? e.message : 'save failed';
    }
  }

  // The editor is created once per mount: `host` is the only tracked
  // dependency — everything else is untracked so async title/dirty updates
  // don't tear down and rebuild the view.
  $effect(() => {
    if (!host) return;
    return untrack(() => createEditor(host!));
  });

  function createEditor(parent: HTMLElement) {
    const theme = EditorView.theme(
      {
        '&': { height: '100%', fontSize: '13px', backgroundColor: '#171c25' },
        '.cm-content': { fontFamily: 'JetBrains Mono, monospace', caretColor: '#e09954' },
        '.cm-cursor': { borderLeftColor: '#e09954' },
        '.cm-gutters': {
          backgroundColor: '#11151c',
          color: '#5d6470',
          border: 'none',
        },
        '.cm-activeLine': { backgroundColor: '#1f253080' },
        '.cm-activeLineGutter': { backgroundColor: '#1f2530' },
        '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
          backgroundColor: '#2a3140 !important',
        },
      },
      { dark: true },
    );

    const state = EditorState.create({
      doc: '',
      extensions: [
        basicSetup,
        theme,
        langConf.of(langFor(path)),
        keymap.of([
          {
            key: 'Mod-s',
            run: () => {
              void save();
              return true;
            },
          },
        ]),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) {
            dirty = true;
            refreshTitle();
          }
        }),
      ],
    });
    view = new EditorView({ state, parent });

    if (path) {
      api
        .get<string>(`/fs/read?path=${encodeURIComponent(path)}`)
        .then((content) => {
          view!.dispatch({ changes: { from: 0, to: view!.state.doc.length, insert: content } });
          dirty = false;
          refreshTitle();
        })
        .catch((e) => (message = e instanceof Error ? e.message : 'open failed'));
    }
    refreshTitle();

    return () => view?.destroy();
  }
</script>

<div class="editor">
  <div class="bar">
    <span class="path" title={path}>{path || 'untitled'}</span>
    {#if message}<span class="msg">{message}</span>{/if}
    <button class="save" onclick={save} title="Save (Cmd/Ctrl+S)">
      <Icon name="save" size={14} />
      <span>Save{dirty ? ' •' : ''}</span>
    </button>
  </div>
  <div class="cm-host" bind:this={host}></div>
</div>

<style>
  .editor {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 10px;
    border-bottom: 1px solid var(--line-soft);
    flex: none;
  }
  .path {
    flex: 1;
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--text-mid);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    direction: rtl;
    text-align: left;
  }
  .msg {
    font-size: 12px;
    color: var(--accent);
  }
  .save {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 12px;
    border-radius: 6px;
    background: var(--ink-3);
    border: 1px solid var(--line-soft);
    color: var(--text-hi);
    font-size: 12.5px;
  }
  .save:hover {
    border-color: var(--accent-dim);
    color: var(--accent-bright);
  }
  .cm-host {
    flex: 1;
    min-height: 0;
    user-select: text;
  }
  .cm-host :global(.cm-editor) {
    height: 100%;
  }
</style>
