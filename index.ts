import { loadConfig } from "./src/config";
import { loadMatrixConfig, sendAlert } from "./src/matrix";
import { initStates, startMonitoring } from "./src/monitor";
import { startWebhookServer } from "./src/webhook";

const { config, resolvedPath } = await loadConfig();
const matrix = loadMatrixConfig();

initStates(config);

async function main() {
  console.log("[startup] Nook booting");
  console.log(`[startup] config path: ${resolvedPath ?? "none (defaults)"}`);
  console.log(`[startup] homeserver: ${matrix.homeserver}`);
  console.log(`[startup] room(s): ${matrix.roomIds.join(", ")}`);
  console.log(`[startup] monitoring ${config.sites.length} site(s)`);
  console.log(`[startup] webhook: ${config.webhook.enabled ? `enabled on port ${config.webhook.port}` : "disabled"}`);

  if (config.sites.length === 0 && !config.webhook.enabled) {
    console.error("[startup] No sites to monitor and webhook disabled. Nothing to do.");
    process.exit(1);
  }

  const parts: string[] = [];

  if (config.sites.length > 0) {
    const siteList = config.sites
      .map((s) => `• <b>${s.name}</b> (<code>${s.url}</code>) every ${s.interval}s`)
      .join("<br/>");
    parts.push(`Monitoring ${config.sites.length} site(s):<br/>${siteList}`);
  }

  if (config.webhook.enabled) {
    parts.push(`Webhook listening on port ${config.webhook.port}`);
  }

  await sendAlert(matrix, `🟢 <b>Nook started</b><br/>${parts.join("<br/>")}`);
  console.log("[startup] Matrix startup message sent");

  if (config.webhook.enabled) {
    startWebhookServer(matrix, config.webhook.port);
  }

  for (const site of config.sites) {
    startMonitoring(matrix, site, config.timeout);
  }
}

main();
