// Notifications store — transient toasts + a persistent history (notification
// center). Fed locally via push() and by ghostd over the `notify` WS topic.
// Singleton, same shape as system.svelte.ts.
import { subscribe } from '../api/ws';

export type NotifyKind = 'info' | 'success' | 'warn' | 'error';

export interface Notification {
  id: string;
  title: string;
  body?: string;
  kind: NotifyKind;
  ts: number;
  read: boolean;
}

export interface NotifyInput {
  title: string;
  body?: string;
  kind?: NotifyKind;
}

const MAX_ITEMS = 50;
const TOAST_MS = 5000;

function newId(): string {
  return crypto.randomUUID?.() ?? `n_${Date.now()}_${Math.random().toString(36).slice(2)}`;
}

class NotifyStore {
  items = $state<Notification[]>([]);
  toasts = $state<Notification[]>([]);
  unread = $derived(this.items.filter((n) => !n.read).length);

  private started = false;
  private timers = new Map<string, ReturnType<typeof setTimeout>>();

  start() {
    if (this.started) return;
    this.started = true;
    subscribe('notify', (env) => {
      if (env.event === 'show') this.push(env.payload as NotifyInput);
    });
  }

  push({ title, body, kind = 'info' }: NotifyInput): Notification {
    const n: Notification = { id: newId(), title, body, kind, ts: Date.now(), read: false };
    this.items = [n, ...this.items].slice(0, MAX_ITEMS);
    this.toasts = [n, ...this.toasts];
    const t = setTimeout(() => this.dismissToast(n.id), TOAST_MS);
    this.timers.set(n.id, t);
    return n;
  }

  dismissToast(id: string) {
    const t = this.timers.get(id);
    if (t) {
      clearTimeout(t);
      this.timers.delete(id);
    }
    this.toasts = this.toasts.filter((n) => n.id !== id);
  }

  markAllRead() {
    this.items = this.items.map((n) => (n.read ? n : { ...n, read: true }));
  }

  clear() {
    this.items = [];
  }
}

export const notifications = new NotifyStore();
