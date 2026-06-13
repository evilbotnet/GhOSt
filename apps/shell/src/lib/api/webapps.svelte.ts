// Installed web apps (ADR 0001 Layer 1): URLs that open as their own
// chromeless browser windows and appear in the launcher.
import { api } from './client';

export interface WebApp {
  id: string;
  name: string;
  url: string;
  icon: string;
}

class WebApps {
  list = $state<WebApp[]>([]);
  private started = false;

  async refresh() {
    this.list = await api.get<WebApp[]>('/apps').catch(() => []);
  }

  start() {
    if (this.started) return;
    this.started = true;
    void this.refresh();
  }

  async install(name: string, url: string) {
    await api.post('/apps/install', { name, url });
    await this.refresh();
  }

  launch(id: string) {
    return api.post('/apps/launch', { id }).catch(() => {});
  }

  async uninstall(id: string) {
    await api.del(`/apps/${id}`);
    await this.refresh();
  }
}

export const webApps = new WebApps();
