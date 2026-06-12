import type { AppDef } from '../wm/wm.svelte';
import Files from './files/Files.svelte';
import Terminal from './terminal/Terminal.svelte';
import Editor from './editor/Editor.svelte';
import Settings from './settings/Settings.svelte';
import Office from './office/Office.svelte';
import About from './about/About.svelte';
import { api } from '../api/client';

export const apps: AppDef[] = [
  {
    id: 'files',
    name: 'Files',
    icon: 'files',
    component: Files,
    defaultSize: { w: 860, h: 560 },
    minSize: { w: 420, h: 300 },
  },
  {
    id: 'terminal',
    name: 'Terminal',
    icon: 'terminal',
    component: Terminal,
    defaultSize: { w: 760, h: 480 },
    minSize: { w: 360, h: 240 },
  },
  {
    id: 'editor',
    name: 'Editor',
    icon: 'editor',
    component: Editor,
    defaultSize: { w: 820, h: 600 },
    minSize: { w: 400, h: 280 },
  },
  {
    id: 'office',
    name: 'Office',
    icon: 'office',
    component: Office,
    defaultSize: { w: 980, h: 640 },
    minSize: { w: 520, h: 360 },
  },
  {
    id: 'settings',
    name: 'Settings',
    icon: 'settings',
    component: Settings,
    defaultSize: { w: 760, h: 540 },
    minSize: { w: 520, h: 360 },
    single: true,
  },
  {
    id: 'about',
    name: 'About OpenOS',
    icon: 'info',
    component: About,
    defaultSize: { w: 460, h: 380 },
    minSize: { w: 380, h: 320 },
    single: true,
  },
];

export function getApp(id: string): AppDef {
  const app = apps.find((a) => a.id === id);
  if (!app) throw new Error(`unknown app: ${id}`);
  return app;
}

/** The Browser is not an in-shell window: the daemon opens a native Chromium
 *  window above the shell (macOS dev: the default browser). */
export function openBrowser(url = 'https://duckduckgo.com') {
  api.post('/browser/open', { url }).catch(() => {
    // daemon unreachable in plain-browser dev: open a tab so the flow works
    window.open(url, '_blank');
  });
}
