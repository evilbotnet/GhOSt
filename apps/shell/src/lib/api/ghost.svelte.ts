// Ghost client: drives one assistant session over WS topic ai.<session>.
import { api } from './client';
import { subscribe, send } from './ws';

export interface ConfirmReq {
  callId: string;
  name: string;
  args: Record<string, unknown>;
}

export interface Entry {
  kind: 'user' | 'message' | 'tool' | 'denied' | 'error';
  text?: string;
  tool?: string;
  output?: string;
  error?: string;
}

let counter = 0;

export interface SkillInfo {
  name: string;
  description: string;
}
export interface ToolInfo {
  name: string;
  description: string;
  mutating: boolean;
}

class GhostSession {
  configured = $state(false);
  provider = $state('');
  entries = $state<Entry[]>([]);
  thinking = $state(false);
  provenance = $state('');
  confirm = $state<ConfirmReq | null>(null);
  skills = $state<SkillInfo[]>([]);
  tools = $state<ToolInfo[]>([]);
  private id = '';
  private unsub: (() => void) | null = null;

  async start() {
    const s = await api
      .get<{ configured: boolean; provider: string }>('/ai/status')
      .catch(() => ({ configured: false, provider: '' }));
    this.configured = s.configured;
    this.provider = s.provider;
    api.get<SkillInfo[]>('/ai/skills').then((v) => (this.skills = v)).catch(() => {});
    api.get<ToolInfo[]>('/ai/tools').then((v) => (this.tools = v)).catch(() => {});
    if (!this.id) {
      this.id = `s${Date.now().toString(36)}${counter++}`;
      this.unsub = subscribe(`ai.${this.id}`, (env) => this.onEvent(env.event, env.payload));
    }
  }

  ask(text: string) {
    if (!text.trim()) return;
    this.entries.push({ kind: 'user', text });
    this.thinking = true;
    send(`ai.${this.id}`, 'prompt', { text });
  }

  decide(callId: string, allow: boolean) {
    send(`ai.${this.id}`, 'confirm', { callId, allow });
    this.confirm = null;
  }

  private onEvent(event: string, payload: unknown) {
    const p = (payload ?? {}) as Record<string, unknown>;
    switch (event) {
      case 'provenance':
        this.provenance = `${p.provider}${p.model ? ' / ' + p.model : ''}`;
        break;
      case 'thinking':
        this.thinking = true;
        break;
      case 'message':
        this.thinking = false;
        this.entries.push({ kind: 'message', text: String(p.text ?? '') });
        break;
      case 'confirm_request':
        this.thinking = false;
        this.confirm = { callId: String(p.callId), name: String(p.name), args: (p.args ?? {}) as Record<string, unknown> };
        break;
      case 'tool_run':
        this.entries.push({ kind: 'tool', tool: String(p.name), output: '…' });
        break;
      case 'tool_result': {
        const last = [...this.entries].reverse().find((e) => e.kind === 'tool' && e.tool === p.name && e.output === '…');
        if (last) {
          last.output = p.output ? String(p.output) : undefined;
          last.error = p.error ? String(p.error) : undefined;
        }
        break;
      }
      case 'tool_denied':
        this.entries.push({ kind: 'denied', tool: String(p.name) });
        break;
      case 'error':
        this.thinking = false;
        this.entries.push({ kind: 'error', text: String(p.message ?? 'error') });
        break;
      case 'done':
        this.thinking = false;
        break;
    }
  }
}

export const ghost = new GhostSession();
