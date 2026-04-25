import { loadConfig } from "./src/config";
import { loadMatrixConfig, sendAlert } from "./src/matrix";
import { initStates, startMonitoring } from "./src/monitor";
import { startWebhookServer } from "./src/webhook";

const { config, resolvedPath } = await loadConfig();
const matrix = loadMatrixConfig();

initStates(config);

async function main() {
  console.log("[startup] Nook booting");
  console.log(`[startup] config path: ${resolvedPath}`);
  console.log(`[startup] homeserver: ${matrix.homeserver}`);
  console.log(`[startup] room(s): ${matrix.roomIds.join(", ")}`);
  console.log(`[startup] monitoring ${config.sites.length} site(s)`);

  const siteList = config.sites
    .map((s) => `• <b>${s.name}</b> (<code>${s.url}</code>) every ${s.interval}s`)
    .join("<br/>");

  await sendAlert(
    matrix,
    `🟢 <b>Nook started</b><br/>Starting monitors for ${config.sites.length} site(s):<br/>${siteList}`,
  );
  console.log("[startup] Matrix startup message sent");

  if (config.webhook.enabled) {
    startWebhookServer(matrix, config.webhook.port);
  }

  for (const site of config.sites) {
    startMonitoring(matrix, site, config.timeout);
  }
}

main();
