import { MatrixClient, SimpleFsStorageProvider } from "matrix-bot-sdk";

interface SiteConfig {
  name: string;
  url: string;
  interval: number;
}

interface Config {
  matrix?: {
    homeserver?: string;
    accessToken?: string;
    roomId?: string;
  };
  sites: SiteConfig[];
  timeout: number;
}

interface SiteState {
  up: boolean;
  lastChange: Date;
  consecutiveFailures: number;
  inFlight: boolean;
}

const configPath = process.env.CONFIG_PATH ?? "config.json";
const storagePath = process.env.NOOK_STORAGE_PATH ?? "nook-storage.json";
const configFile = Bun.file(configPath);

if (!(await configFile.exists())) {
  throw new Error(`Missing config file: ${configPath}`);
}

const config: Config = await configFile.json();
const homeserver = process.env.MATRIX_HOMESERVER ?? config.matrix?.homeserver;
const accessToken = process.env.MATRIX_ACCESS_TOKEN ?? config.matrix?.accessToken;
const roomId = process.env.MATRIX_ROOM_ID ?? config.matrix?.roomId;

if (!homeserver || !accessToken || !roomId) {
  throw new Error(
    "Missing Matrix configuration. Set MATRIX_HOMESERVER, MATRIX_ACCESS_TOKEN, and MATRIX_ROOM_ID or provide them in config.json.",
  );
}

const matrixConfig = {
  homeserver,
  accessToken,
  roomId,
};

const states = new Map<string, SiteState>();

for (const site of config.sites) {
  states.set(site.url, {
    up: true,
    lastChange: new Date(),
    consecutiveFailures: 0,
    inFlight: false,
  });
}

const storage = new SimpleFsStorageProvider(storagePath);
const client = new MatrixClient(
  matrixConfig.homeserver,
  matrixConfig.accessToken,
  storage,
);
async function sendAlert(message: string) {
  try {
    await client.sendMessage(matrixConfig.roomId, {
      msgtype: "m.text",
      body: message.replace(/<[^>]*>/g, ""),
      format: "org.matrix.custom.html",
      formatted_body: message,
    });
  } catch (err) {
    console.error(`Failed to send Matrix alert: ${err}`);
  }
}

async function checkSite(site: SiteConfig) {
  const state = states.get(site.url)!;
  if (state.inFlight) return;

  state.inFlight = true;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), config.timeout * 1000);

  try {
    const res = await fetch(site.url, {
      signal: controller.signal,
      headers: { "User-Agent": "Nook/1.0 uptime-bot" },
    });

    console.log(
      `[health] ${site.name} ${site.url} -> HTTP ${res.status}`,
    );

    if (res.status < 400) {
      if (!state.up) {
        const downtime = timeSince(state.lastChange);
        await sendAlert(
          `✅ <b>${site.name}</b> is back up — was down for ${downtime}`,
        );
        state.up = true;
        state.lastChange = new Date();
      }
      state.consecutiveFailures = 0;
    } else {
      await handleFailure(site, state, `HTTP ${res.status}`);
    }
  } catch (err) {
    const reason = err instanceof Error ? err.message : "unknown error";
    console.error(`[health] ${site.name} ${site.url} -> ${reason}`);
    await handleFailure(site, state, reason);
  } finally {
    clearTimeout(timer);
    state.inFlight = false;
  }
}

async function handleFailure(site: SiteConfig, state: SiteState, reason: string) {
  state.consecutiveFailures++;
  if (state.up && state.consecutiveFailures >= 2) {
    state.up = false;
    state.lastChange = new Date();
    await sendAlert(
      `🔴 <b>${site.name}</b> is DOWN — ${reason}<br/><code>${site.url}</code>`,
    );
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

function startMonitoring(site: SiteConfig) {
  console.log(
    `[monitor] starting ${site.name} (${site.url}) every ${site.interval}s`,
  );
  checkSite(site);
  setInterval(() => checkSite(site), site.interval * 1000);
}

async function main() {
  console.log("[startup] Nook booting");
  console.log(`[startup] config path: ${configPath}`);
  console.log(`[startup] storage path: ${storagePath}`);
  console.log(`[startup] homeserver: ${matrixConfig.homeserver}`);
  console.log(`[startup] room id: ${matrixConfig.roomId}`);
  console.log(`[startup] monitoring ${config.sites.length} site(s)`);

  await sendAlert(
    `🟢 <b>Nook started</b><br/>Monitoring ${config.sites.length} site(s)`,
  );
  console.log("[startup] Matrix startup message sent");

  for (const site of config.sites) {
    startMonitoring(site);
  }
}

main();
