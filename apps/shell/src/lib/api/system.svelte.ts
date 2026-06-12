// Live system status for the tray + settings. Seeded by REST, kept fresh by
// the `system` WS topic. Degrades gracefully when the daemon is unreachable
// (shell still renders; tray shows offline state).
import { api, type SystemStatus } from './client';
import { subscribe } from './ws';

const OFFLINE: SystemStatus = {
  hostname: 'ghost',
  platform: 'unknown',
  wifi: { available: false, connected: false, ssid: '', signal: 0 },
  battery: { available: false, charging: false, percent: 0 },
  volume: { percent: 0, muted: false },
};

class SystemStore {
  status = $state<SystemStatus>(OFFLINE);
  online = $state(false);
  private started = false;

  start() {
    if (this.started) return;
    this.started = true;
    api
      .get<SystemStatus>('/system/status')
      .then((s) => {
        this.status = s;
        this.online = true;
      })
      .catch(() => (this.online = false));
    subscribe('system', (env) => {
      if (env.event === 'status') {
        this.status = env.payload as SystemStatus;
        this.online = true;
      }
    });
  }

  async setVolume(percent: number) {
    this.status.volume.percent = percent;
    await api.post('/system/volume', { percent }).catch(() => {});
  }
}

export const system = new SystemStore();
