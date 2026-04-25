import type { Config, RawConfig, RawSiteConfig, RawSiteOptions, WebhookConfig } from "./types";

const defaultConfigPath = "/data/config.yml";
const configPath = process.env.CONFIG_PATH ?? defaultConfigPath;
const defaultIntervalSeconds = 60;
const defaultTimeoutSeconds = 10;
const defaultWebhookPort = 9090;

function isPositiveNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value > 0;
}

function deriveSiteName(siteUrl: string): string {
  const parsedUrl = new URL(siteUrl);
  const hostname = parsedUrl.hostname.replace(/^www\./, "");
  const path = `${parsedUrl.pathname}${parsedUrl.search}`.replace(/\/$/, "");
  return path && path !== "/" ? `${hostname}${path}` : hostname;
}

function parseYamlScalar(rawValue: string): string | number {
  const value = rawValue.trim();
  if (
    (value.startsWith("\"") && value.endsWith("\""))
    || (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1);
  }

  const numberValue = Number(value);
  if (!Number.isNaN(numberValue) && value !== "") {
    return numberValue;
  }

  return value;
}

function splitYamlKeyValue(line: string) {
  const separatorIndex = line.indexOf(":");
  if (separatorIndex === -1) {
    throw new Error(`Invalid YAML line: ${line}`);
  }

  return {
    key: line.slice(0, separatorIndex).trim(),
    value: line.slice(separatorIndex + 1).trim(),
  };
}

function parseSimpleYamlConfig(text: string): RawConfig {
  const rawConfig: RawConfig = {};
  const sites: RawSiteConfig[] = [];
  const siteOptionKeys = new Set(["name", "url", "every", "interval"]);
  const rawLines = text
    .replace(/^\uFEFF/, "")
    .split(/\r?\n/)
    .map((line) => line.replace(/\t/g, "  "));
  const baseIndent = rawLines.reduce((smallestIndent, line) => {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      return smallestIndent;
    }

    const indent = line.length - line.trimStart().length;
    if (smallestIndent === null) {
      return indent;
    }

    return Math.min(smallestIndent, indent);
  }, null as number | null) ?? 0;
  const lines = rawLines.map((line) => line.slice(Math.min(baseIndent, line.length)));

  let inSites = false;
  let currentSite: RawSiteOptions | null = null;

  const pushCurrentSite = () => {
    if (currentSite) {
      sites.push(currentSite);
      currentSite = null;
    }
  };

  for (const rawLine of lines) {
    const trimmed = rawLine.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const indent = rawLine.length - rawLine.trimStart().length;

    if (indent === 0) {
      pushCurrentSite();

      if (trimmed === "sites:") {
        inSites = true;
        continue;
      }

      inSites = false;
      const { key, value } = splitYamlKeyValue(trimmed);
      rawConfig[key as keyof RawConfig] = parseYamlScalar(value) as never;
      continue;
    }

    if (!inSites) {
      throw new Error(`Unexpected indentation outside \`sites\`: ${trimmed}`);
    }

    if (indent < 2) {
      throw new Error(`Invalid site indentation: ${trimmed}`);
    }

    if (trimmed.startsWith("- ")) {
      pushCurrentSite();
      const item = trimmed.slice(2).trim();

      if (!item) {
        currentSite = {};
        continue;
      }

      const separatorIndex = item.indexOf(":");
      const maybeKey = separatorIndex === -1 ? "" : item.slice(0, separatorIndex).trim();
      if (siteOptionKeys.has(maybeKey)) {
        const { key, value } = splitYamlKeyValue(item);
        currentSite = {
          [key]: parseYamlScalar(value),
        };
        continue;
      }

      sites.push(parseYamlScalar(item).toString());
      continue;
    }

    if (!currentSite) {
      throw new Error(`Invalid site entry: ${trimmed}`);
    }

    const { key, value } = splitYamlKeyValue(trimmed);
    currentSite[key as keyof RawSiteOptions] = parseYamlScalar(value) as never;
  }

  pushCurrentSite();

  if (sites.length > 0) {
    rawConfig.sites = sites;
  }

  return rawConfig;
}

function parseRawConfig(text: string, path: string): RawConfig {
  if (path.endsWith(".json")) {
    throw new Error(`JSON config is no longer supported at ${path}. Use config.yml instead.`);
  }

  const normalizedText = text.replace(/^\uFEFF/, "").trim();
  const parsers = [
    () => Bun.YAML.parse(normalizedText),
    () => parseSimpleYamlConfig(normalizedText),
  ];

  const errors: string[] = [];

  for (const parse of parsers) {
    try {
      return parse() as RawConfig;
    } catch (error) {
      errors.push(error instanceof Error ? error.message : "unknown parse error");
    }
  }

  throw new Error(`Failed to parse YAML config at ${path}: ${errors.join(" | ")}`);
}

function normalizeConfig(rawConfig: RawConfig): Config {
  if (!rawConfig || typeof rawConfig !== "object") {
    throw new Error("Configuration file must contain a YAML object.");
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

  const rawWebhook = rawConfig.webhook;
  const webhook: WebhookConfig = {
    enabled: rawWebhook?.enabled ?? false,
    port: rawWebhook?.port ?? defaultWebhookPort,
  };

  if (!isPositiveNumber(webhook.port) || !Number.isInteger(webhook.port)) {
    throw new Error("Configuration `webhook.port` must be a positive integer.");
  }

  return {
    timeout,
    sites,
    webhook,
  };
}

export async function loadConfig() {
  const fallbackConfigPaths = configPath === defaultConfigPath
    ? [
      "/data/config.yml",
      "config.yml",
      "config.yaml",
      "config.example.yml",
      "config.example.yaml",
    ]
    : [
      "config.example.yml",
      "config.example.yaml",
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
      config: normalizeConfig(parseRawConfig(text, candidatePath)),
      resolvedPath: candidatePath,
    };
  }

  throw new Error(
    `Configuration file missing. Please provide a config.yml at ${configPath} or include config.example.yml in the app directory.`
  );
}
