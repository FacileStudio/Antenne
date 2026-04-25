import type { AppSettings, SiteConfig, SiteState } from "./types";

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

export function createMonitorManager(sendAlert: (message: string) => Promise<void>) {
  const entries = new Map<string, MonitorEntry>();

  async function handleFailure(site: SiteConfig, state: SiteState, reason: string) {
    state.consecutiveFailures += 1;
    if (state.up && state.consecutiveFailures >= 2) {
      state.up = false;
      state.lastChange = new Date();
      await sendAlert(`🔴 <b>${site.name}</b> is DOWN — ${reason}<br/><code>${site.url}</code>`);
    }
  }

  async function checkSite(site: SiteConfig, timeout: number) {
    const entry = entries.get(site.url);
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
          await sendAlert(`✅ <b>${site.name}</b> is back up — was down for ${downtime}`);
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

    for (const site of settings.sites) {
      const state: SiteState = {
        up: true,
        lastChange: new Date(),
        consecutiveFailures: 0,
        inFlight: false,
      };

      const timerId = setInterval(() => void checkSite(site, settings.timeout), site.interval * 1000);
      entries.set(site.url, { timerId, state });
      console.log(`[monitor] starting ${site.name} (${site.url}) every ${site.interval}s`);
      void checkSite(site, settings.timeout);
    }
  }

  return {
    apply,
    stopAll,
  };
}
