// Native Wayland toplevels (browser windows, Linux GUI apps), tracked by the
// daemon via wlrctl and shown in the taskbar alongside in-shell windows.
import { api } from './client';
import { subscribe } from './ws';

export interface NativeWindow {
  appId: string;
  title: string;
}

class NativeWindows {
  list = $state<NativeWindow[]>([]);
  private started = false;

  start() {
    if (this.started) return;
    this.started = true;
    api
      .get<NativeWindow[]>('/windows')
      .then((l) => (this.list = l))
      .catch(() => {});
    subscribe('windows', (env) => {
      if (env.event === 'list') this.list = env.payload as NativeWindow[];
    });
  }

  act(action: 'focus' | 'minimize' | 'close', win: NativeWindow) {
    return api.post('/windows/action', {
      appId: win.appId,
      title: win.title,
      action,
    });
  }
}

export const nativeWindows = new NativeWindows();
