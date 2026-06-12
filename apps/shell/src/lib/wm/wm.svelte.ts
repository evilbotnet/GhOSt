import type { Component } from 'svelte';

export interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface AppDef {
  id: string;
  name: string;
  icon: string;
  // Component receives the Win so apps can set their own title, close, etc.
  component: Component<{ win: Win }, {}, ''>;
  defaultSize: { w: number; h: number };
  minSize: { w: number; h: number };
  single?: boolean;
}

let nextWinId = 1;
let zCounter = 10;
let cascade = 0;

/** Viewport size available to windows (kept in sync by the desktop container). */
export const viewport = $state({ w: 1280, h: 720 });

export class Win {
  readonly id: number;
  readonly app: AppDef;
  title = $state('');
  rect = $state<Rect>({ x: 0, y: 0, w: 0, h: 0 });
  z = $state(0);
  minimized = $state(false);
  maximized = $state(false);
  /** App-specific launch payload (e.g. file path for the editor). */
  props: Record<string, unknown>;
  private prevRect: Rect | null = null;

  constructor(app: AppDef, props: Record<string, unknown> = {}) {
    this.id = nextWinId++;
    this.app = app;
    this.title = app.name;
    this.props = props;
  }

  placeInitial() {
    const { w: vw, h: vh } = viewport;
    const w = Math.min(this.app.defaultSize.w, vw - 40);
    const h = Math.min(this.app.defaultSize.h, vh - 40);
    const step = (cascade++ % 6) * 32;
    this.rect = {
      x: Math.max(12, Math.round((vw - w) / 2) + step - 80),
      y: Math.max(12, Math.round((vh - h) / 2.4) + step - 60),
      w,
      h,
    };
  }

  toggleMaximize() {
    if (this.maximized) {
      this.maximized = false;
      if (this.prevRect) this.rect = this.prevRect;
    } else {
      this.prevRect = { ...this.rect };
      this.maximized = true;
    }
  }
}

class WindowManager {
  windows = $state<Win[]>([]);
  focusedId = $state<number | null>(null);

  open(app: AppDef, props: Record<string, unknown> = {}): Win {
    if (app.single) {
      const existing = this.windows.find((w) => w.app.id === app.id);
      if (existing) {
        this.focus(existing.id);
        return existing;
      }
    }
    const win = new Win(app, props);
    win.placeInitial();
    this.windows.push(win);
    this.focus(win.id);
    return win;
  }

  close(id: number) {
    const i = this.windows.findIndex((w) => w.id === id);
    if (i === -1) return;
    this.windows.splice(i, 1);
    if (this.focusedId === id) this.focusTopmost();
  }

  focus(id: number) {
    const win = this.windows.find((w) => w.id === id);
    if (!win) return;
    win.minimized = false;
    win.z = ++zCounter;
    this.focusedId = id;
  }

  minimize(id: number) {
    const win = this.windows.find((w) => w.id === id);
    if (!win) return;
    win.minimized = true;
    if (this.focusedId === id) this.focusTopmost();
  }

  /** Taskbar click: restore/focus, or minimize when already focused. */
  toggleFromTaskbar(id: number) {
    const win = this.windows.find((w) => w.id === id);
    if (!win) return;
    if (win.minimized) this.focus(id);
    else if (this.focusedId === id) this.minimize(id);
    else this.focus(id);
  }

  private focusTopmost() {
    const candidates = this.windows.filter((w) => !w.minimized);
    if (candidates.length === 0) {
      this.focusedId = null;
      return;
    }
    const top = candidates.reduce((a, b) => (a.z > b.z ? a : b));
    this.focusedId = top.id;
  }
}

export const wm = new WindowManager();
