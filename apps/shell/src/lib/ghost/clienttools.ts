// Client tools — the shell side of ADR 0006. A running app registers tools with
// Ghost over the WS topic "ghosttools"; Ghost (in the daemon) merges them into
// its loop and, when it calls one, emits an "invoke" we answer with a "result".
//
// The shell itself is the first such app: it exposes `open_app` and `list_apps`,
// so Ghost can drive the desktop ("open the system monitor"). Any future .osapp
// uses this same channel — this module is the reference implementation.

import { subscribe, send, onOpen } from '../api/ws';
import { wm } from '../wm/wm.svelte';
import { apps } from '../apps/registry';

interface ClientTool {
  name: string;
  description: string;
  properties: Record<string, unknown>;
  required: string[];
  mutating: boolean;
  run: (args: Record<string, unknown>) => string;
}

const tools: ClientTool[] = [
  {
    name: 'list_apps',
    description: 'List the GhOSt desktop apps that can be opened, with their ids.',
    properties: {},
    required: [],
    mutating: false,
    run: () => apps.map((a) => `${a.id} — ${a.name}`).join('\n'),
  },
  {
    name: 'open_app',
    description:
      'Open one of the GhOSt desktop apps by id (use list_apps to see ids). ' +
      'Use this when the user asks to open, launch, or show an app.',
    properties: {
      id: {
        type: 'string',
        description: 'app id, e.g. files, terminal, monitor, settings',
        enum: apps.map((a) => a.id),
      },
    },
    required: ['id'],
    mutating: false,
    run: (args) => {
      const id = String(args.id ?? '');
      const app = apps.find((a) => a.id === id);
      if (!app) {
        return `no app '${id}'. Available: ${apps.map((a) => a.id).join(', ')}`;
      }
      wm.open(app);
      return `opened ${app.name}`;
    },
  },
];

let started = false;

/** Register the shell's tools with Ghost and answer invocations. Idempotent. */
export function startClientTools() {
  if (started) return;
  started = true;

  subscribe('ghosttools', (env) => {
    if (env.event !== 'invoke') return;
    const p = env.payload as { callId: string; name: string; args?: Record<string, unknown> };
    const tool = tools.find((t) => t.name === p.name);
    let output = '';
    let error = '';
    try {
      output = tool ? tool.run(p.args ?? {}) : `unknown app tool '${p.name}'`;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
    send('ghosttools', 'result', { callId: p.callId, output, error });
  });

  // Register on every (re)connect so Ghost keeps our tools after a daemon
  // restart or socket drop. The daemon replaces its whole set on each register.
  const register = () =>
    send('ghosttools', 'register', {
      // send the schema only — the local run fn stays in the shell
      tools: tools.map((t) => ({
        name: t.name,
        description: t.description,
        properties: t.properties,
        required: t.required,
        mutating: t.mutating,
      })),
    });
  onOpen(register);
  register();
}
