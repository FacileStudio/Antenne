interface SiteConfig {
  name: string;
  url: string;
  interval: number;
}

interface SiteState {
  up: boolean;
  lastChange: Date;
  consecutiveFailures: number;
  inFlight: boolean;
}

interface RawSiteOptions {
  name?: string;
  url?: string;
  every?: number;
  interval?: number;
}

type RawSiteConfig = string | RawSiteOptions;

interface RawConfig {
  timeout?: number;
  every?: number;
  interval?: number;
  sites?: RawSiteConfig[];
}

interface Config {
  sites: SiteConfig[];
  timeout: number;
}

const defaultConfigPath = "/data/config.yml";
const configPath = process.env.CONFIG_PATH ?? defaultConfigPath;
const defaultIntervalSeconds = 60;
const defaultTimeoutSeconds = 10;

function isPositiveNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value > 0;
}

function deriveSiteName(siteUrl: string): string {
  const parsedUrl = new URL(siteUrl);
  const hostname = parsedUrl.hostname.replace(/^www\./, "");
  const path = `${parsedUrl.pathname}${parsedUrl.search}`.replace(/\/$/, "");
  return path && path !== "/" ? `${hostname}${path}` : hostname;
}

function normalizeConfig(rawConfig: RawConfig): Config {
  if (!rawConfig || typeof rawConfig !== "object") {
    throw new Error("Configuration file must contain a YAML or JSON object.");
  }

  if (!Array.isArray(rawConfig.sites) || rawConfig.sites.length === 0) {
    throw new Error("Configuration must define at least one site under `sites`.");
  }

  const timeout = rawConfig.timeout ?? defaultTimeoutSeconds;
  if (!isPositiveNumber(timeout)) {
    throw new Error("Configuration `timeout` must be a positive number.");
  }

  const defaultInterval = rawConfig.every ?? rawConfig.interval ?? defaultIntervalSeconds;
  if (!isPositiveNumber(defaultInterval)) {
    throw new Error("Configuration `every` must be a positive number.");
  }

  const sites = rawConfig.sites.map((site, index) => {
    const siteConfig = typeof site === "string" ? { url: site } : site;
    const url = siteConfig.url?.trim();
    const interval = siteConfig.every ?? siteConfig.interval ?? defaultInterval;

    if (!url) {
      throw new Error(`Site #${index + 1} is missing a valid \`url\`.`);
    }

    if (!isPositiveNumber(interval)) {
      throw new Error(`Site \`${url}\` must define a positive \`every\` value.`);
    }

    const name = siteConfig.name?.trim() || deriveSiteName(url);

    return {
      name,
      url,
      interval,
    };
  });

  return {
    timeout,
    sites,
  };
}

async function loadConfig() {
  const fallbackConfigPaths = configPath === defaultConfigPath
    ? [
      "/data/config.yml",
      "/data/config.json",
      "config.yml",
      "config.yaml",
      "config.json",
      "config.example.yml",
      "config.example.yaml",
      "config.example.json",
    ]
    : [
      "config.example.yml",
      "config.example.yaml",
      "config.example.json",
    ];

  const candidatePaths = [...new Set([configPath, ...fallbackConfigPaths])];

  for (const candidatePath of candidatePaths) {
    const file = Bun.file(candidatePath);
    if (!(await file.exists())) {
      continue;
    }

    if (candidatePath !== configPath) {
      console.warn(`[startup] Config file not found at ${configPath}. Falling back to ${candidatePath}`);
    }

    const text = await file.text();
    return {
      config: normalizeConfig(Bun.YAML.parse(text) as RawConfig),
      resolvedPath: candidatePath,
    };
  }

  throw new Error(
    `Configuration file missing. Please provide a config.yml at ${configPath} or include config.example.yml in the app directory.`
  );
}

const { config, resolvedPath: resolvedConfigPath } = await loadConfig();
const homeserver = process.env.MATRIX_HOMESERVER;
const accessToken = process.env.MATRIX_ACCESS_TOKEN;
const roomId = process.env.MATRIX_ROOM_ID;

if (!homeserver || !accessToken || !roomId) {
  throw new Error(
    "Missing Matrix configuration. Set MATRIX_HOMESERVER, MATRIX_ACCESS_TOKEN, and MATRIX_ROOM_ID in the environment.",
  );
}

const matrixConfig = {
  homeserver,
  accessToken,
  roomId,
};
let matrixTxnId = 0;

const states = new Map<string, SiteState>();

for (const site of config.sites) {
  states.set(site.url, {
    up: true,
    lastChange: new Date(),
    consecutiveFailures: 0,
    inFlight: false,
  });
}

async function matrixRequest(method: string, path: string, body?: unknown) {
  const url = new URL(
    path,
    matrixConfig.homeserver.endsWith("/")
      ? matrixConfig.homeserver
      : `${matrixConfig.homeserver}/`,
  );
  const res = await fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${matrixConfig.accessToken}`,
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(
      `Matrix API request failed: ${method} ${path} -> HTTP ${res.status}${body ? ` ${body}` : ""}`,
    );
  }

  if (res.status === 204) {
    return null;
  }

  const text = await res.text();
  if (!text) {
    return null;
  }

  return JSON.parse(text);
}

async function sendAlert(message: string) {
  try {
    matrixTxnId += 1;
    await matrixRequest(
      "PUT",
      `/_matrix/client/v3/rooms/${encodeURIComponent(matrixConfig.roomId)}/send/m.room.message/${encodeURIComponent(`${Date.now()}-${matrixTxnId}`)}`,
      {
        msgtype: "m.text",
        body: message.replace(/<[^>]*>/g, ""),
        format: "org.matrix.custom.html",
        formatted_body: message,
      },
    );
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
  console.log(`[startup] config path: ${resolvedConfigPath}`);
  console.log(`[startup] homeserver: ${matrixConfig.homeserver}`);
  console.log(`[startup] room id: ${matrixConfig.roomId}`);
  console.log(`[startup] monitoring ${config.sites.length} site(s)`);
  console.log("[startup] Matrix sender ready");

  const siteList = config.sites
    .map((s) => `• <b>${s.name}</b> (<code>${s.url}</code>) every ${s.interval}s`)
    .join("<br/>");

  await sendAlert(
    `🟢 <b>Nook started</b><br/>Starting monitors for ${config.sites.length} site(s):<br/>${siteList}`,
  );
  console.log("[startup] Matrix startup message sent");

  for (const site of config.sites) {
    startMonitoring(site);
  }
}

main();
