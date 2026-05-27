import type { ConnectedApp } from "./types";

export class ConnectionRegistry {
  private connections = new Map<string, ConnectedApp>();
  private channelSubscribers = new Map<string, Set<string>>();

  add(app: ConnectedApp) {
    this.connections.set(app.app_id, app);
  }

  remove(appId: string) {
    const app = this.connections.get(appId);
    if (!app) return;

    for (const channel of app.channels) {
      const subscribers = this.channelSubscribers.get(channel);
      if (subscribers) {
        subscribers.delete(appId);
        if (subscribers.size === 0) {
          this.channelSubscribers.delete(channel);
        }
      }
    }

    this.connections.delete(appId);
  }

  subscribe(appId: string, channel: string) {
    const app = this.connections.get(appId);
    if (!app) return;

    app.channels.add(channel);

    let subscribers = this.channelSubscribers.get(channel);
    if (!subscribers) {
      subscribers = new Set();
      this.channelSubscribers.set(channel, subscribers);
    }
    subscribers.add(appId);
  }

  getSubscribers(channel: string): ConnectedApp[] {
    const subscribers = this.channelSubscribers.get(channel);
    if (!subscribers) return [];

    const apps: ConnectedApp[] = [];
    for (const appId of subscribers) {
      const app = this.connections.get(appId);
      if (app) apps.push(app);
    }
    return apps;
  }

  get(appId: string): ConnectedApp | undefined {
    return this.connections.get(appId);
  }

  getAll(): ConnectedApp[] {
    return Array.from(this.connections.values());
  }

  size(): number {
    return this.connections.size;
  }
}
