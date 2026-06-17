// Theme store — switchable dark/light + accent color, persisted to localStorage
// (instant, no flash) and reconciled with the daemon's /settings (key "theme").
// The CSS contract lives in tokens.css; this store only flips the
// `data-theme` attribute and overrides the four accent custom properties.
import { api } from '../api/client';

export type Mode = 'dark' | 'light';
export type Accent = 'copper' | 'teal' | 'violet' | 'green';

interface AccentPreset {
  accent: string;
  bright: string;
  dim: string;
  ink: string; // text-on-accent
}

// 'copper' MUST equal the values in tokens.css so dark+copper == today.
const PRESETS: Record<Accent, AccentPreset> = {
  copper: { accent: '#e09954', bright: '#f0b576', dim: '#8a5e34', ink: '#1a1208' },
  teal: { accent: '#43b8b0', bright: '#6fd6cf', dim: '#2c6e6a', ink: '#04201e' },
  violet: { accent: '#a583e0', bright: '#c0a6f0', dim: '#5f4a8a', ink: '#150c24' },
  green: { accent: '#7fb069', bright: '#9fcf89', dim: '#4e6e3f', ink: '#0c1808' },
};

const LS_KEY = 'ghost.theme';

interface Persisted {
  mode: Mode;
  accent: Accent;
}

function isMode(v: unknown): v is Mode {
  return v === 'dark' || v === 'light';
}
function isAccent(v: unknown): v is Accent {
  return v === 'copper' || v === 'teal' || v === 'violet' || v === 'green';
}

class ThemeStore {
  mode = $state<Mode>('dark');
  accent = $state<Accent>('copper');
  readonly accents: Accent[] = ['copper', 'teal', 'violet', 'green'];
  private started = false;

  // Color of a swatch for the UI (the base accent of each preset).
  swatch(a: Accent): string {
    return PRESETS[a].accent;
  }

  start() {
    if (this.started) return;
    this.started = true;

    // 1) localStorage first — apply synchronously to avoid a flash.
    try {
      const raw = localStorage.getItem(LS_KEY);
      if (raw) {
        const p = JSON.parse(raw) as Partial<Persisted>;
        if (isMode(p.mode)) this.mode = p.mode;
        if (isAccent(p.accent)) this.accent = p.accent;
      }
    } catch {
      /* ignore malformed storage */
    }
    this.apply();

    // 2) Reconcile with the daemon (best-effort). The value is a JSON string.
    api
      .get<{ theme?: string }>('/settings')
      .then((s) => {
        if (!s.theme) return;
        const p = JSON.parse(s.theme) as Partial<Persisted>;
        let changed = false;
        if (isMode(p.mode) && p.mode !== this.mode) {
          this.mode = p.mode;
          changed = true;
        }
        if (isAccent(p.accent) && p.accent !== this.accent) {
          this.accent = p.accent;
          changed = true;
        }
        if (changed) this.apply();
      })
      .catch(() => {});
  }

  setMode(m: Mode) {
    if (m === this.mode) return;
    this.mode = m;
    this.apply();
    this.persist();
  }

  setAccent(a: Accent) {
    if (a === this.accent) return;
    this.accent = a;
    this.apply();
    this.persist();
  }

  apply() {
    const root = document.documentElement;
    root.dataset.theme = this.mode;
    const p = PRESETS[this.accent];
    root.style.setProperty('--accent', p.accent);
    root.style.setProperty('--accent-bright', p.bright);
    root.style.setProperty('--accent-dim', p.dim);
    root.style.setProperty('--accent-ink', p.ink);
  }

  private persist() {
    const payload: Persisted = { mode: this.mode, accent: this.accent };
    const json = JSON.stringify(payload);
    try {
      localStorage.setItem(LS_KEY, json);
    } catch {
      /* ignore quota / privacy errors */
    }
    api.put('/settings', { key: 'theme', value: json }).catch(() => {});
  }
}

export const theme = new ThemeStore();
