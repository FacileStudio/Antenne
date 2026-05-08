import type { AlertContext, AppSettings, SiteState } from "./types";
import { isWebsiteProviderSettings, type WebsiteProviderSettings } from "./webhooks/types";

interface MonitorEntry {
  timerId: Timer;
  state: SiteState;
}

function timeSince(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ${minutes % 60}m`;
  const days = Math.floor(hours / 24);
  return `${days}d ${hours % 24}h`;
}

export interface MonitorAlertContext extends AlertContext {
  providerType: "website";
}

export function createMonitorManager(sendAlert: (message: string, context: MonitorAlertContext) => Promise<void>) {
  const entries = new Map<string, MonitorEntry>();

  async function handleFailure(site: WebsiteProviderSettings, state: SiteState, reason: string) {
    state.consecutiveFailures += 1;
    if (state.up && state.consecutiveFailures >= 2) {
      state.up = false;
      state.lastChange = new Date();
      await sendAlert(`<b>[Website] ${site.name}</b> is DOWN — ${reason}<br/><code>${site.url}</code>`, {
        providerId: site.id,
        providerName: site.name,
        providerType: "website",
        event: "site DOWN",
        eventType: "website.down",
        eventTitle: `${site.name} is down`,
        subject: {
          type: "service",
          id: site.id,
          name: site.name,
          url: site.url,
        },
        normalizedPayload: {
          url: site.url,
          status: "down",
          reason,
          consecutiveFailures: state.consecutiveFailures,
        },
      });
    }
  }

  async function checkSite(site: WebsiteProviderSettings, timeout: number) {
    const entry = entries.get(site.id);
    if (!entry || entry.state.inFlight) {
      return;
    }

    entry.state.inFlight = true;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeout * 1000);

    try {
      const res = await fetch(site.url, {
        signal: controller.signal,
        headers: { "User-Agent": "Nook/1.0 uptime-bot" },
      });

      console.log(`[health] ${site.name} ${site.url} -> HTTP ${res.status}`);

      if (res.status < 400) {
        if (!entry.state.up) {
          const downtime = timeSince(entry.state.lastChange);
          await sendAlert(`<b>[Website] ${site.name}</b> is back up — was down for ${downtime}`, {
            providerId: site.id,
            providerName: site.name,
            providerType: "website",
            event: "site UP",
            eventType: "website.up",
            eventTitle: `${site.name} is back up`,
            subject: {
              type: "service",
              id: site.id,
              name: site.name,
              url: site.url,
            },
            normalizedPayload: {
              url: site.url,
              status: "up",
              downtime,
            },
          });
          entry.state.up = true;
          entry.state.lastChange = new Date();
        }

        entry.state.consecutiveFailures = 0;
      } else {
        await handleFailure(site, entry.state, `HTTP ${res.status}`);
      }
    } catch (error) {
      const reason = error instanceof Error ? error.message : "unknown error";
      console.error(`[health] ${site.name} ${site.url} -> ${reason}`);
      await handleFailure(site, entry.state, reason);
    } finally {
      clearTimeout(timer);
      entry.state.inFlight = false;
    }
  }

  function stopAll() {
    for (const entry of entries.values()) {
      clearInterval(entry.timerId);
    }

    entries.clear();
  }

  function apply(settings: AppSettings) {
    stopAll();

    for (const provider of settings.providers.items.filter(isWebsiteProviderSettings).filter((provider) => provider.enabled)) {
      const state: SiteState = {
        up: true,
        lastChange: new Date(),
        consecutiveFailures: 0,
        inFlight: false,
      };

      const timerId = setInterval(() => void checkSite(provider, settings.timeout), provider.interval * 1000);
      entries.set(provider.id, { timerId, state });
      console.log(`[monitor] starting ${provider.name} (${provider.url}) every ${provider.interval}s`);
      void checkSite(provider, settings.timeout);
    }
  }

  return {
    apply,
    stopAll,
  };
}
