import type { Config, MatrixConfig, SiteConfig, SiteState } from "./types";
import { sendAlert } from "./matrix";

const states = new Map<string, SiteState>();

export function initStates(config: Config) {
  for (const site of config.sites) {
    states.set(site.url, {
      up: true,
      lastChange: new Date(),
      consecutiveFailures: 0,
      inFlight: false,
    });
  }
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

async function handleFailure(matrix: MatrixConfig, site: SiteConfig, state: SiteState, reason: string) {
  state.consecutiveFailures++;
  if (state.up && state.consecutiveFailures >= 2) {
    state.up = false;
    state.lastChange = new Date();
    await sendAlert(
      matrix,
      `🔴 <b>${site.name}</b> is DOWN — ${reason}<br/><code>${site.url}</code>`,
    );
  }
}

async function checkSite(matrix: MatrixConfig, site: SiteConfig, timeout: number) {
  const state = states.get(site.url)!;
  if (state.inFlight) return;

  state.inFlight = true;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout * 1000);

  try {
    const res = await fetch(site.url, {
      signal: controller.signal,
      headers: { "User-Agent": "Nook/1.0 uptime-bot" },
    });

    console.log(`[health] ${site.name} ${site.url} -> HTTP ${res.status}`);

    if (res.status < 400) {
      if (!state.up) {
        const downtime = timeSince(state.lastChange);
        await sendAlert(
          matrix,
          `✅ <b>${site.name}</b> is back up — was down for ${downtime}`,
        );
        state.up = true;
        state.lastChange = new Date();
      }
      state.consecutiveFailures = 0;
    } else {
      await handleFailure(matrix, site, state, `HTTP ${res.status}`);
    }
  } catch (err) {
    const reason = err instanceof Error ? err.message : "unknown error";
    console.error(`[health] ${site.name} ${site.url} -> ${reason}`);
    await handleFailure(matrix, site, state, reason);
  } finally {
    clearTimeout(timer);
    state.inFlight = false;
  }
}

export function startMonitoring(matrix: MatrixConfig, site: SiteConfig, timeout: number) {
  console.log(`[monitor] starting ${site.name} (${site.url}) every ${site.interval}s`);
  checkSite(matrix, site, timeout);
  setInterval(() => checkSite(matrix, site, timeout), site.interval * 1000);
}
