import type { AppSettings } from "../types";
import type {
  ConfiguredWebhookProvider,
  WebhookConfig,
  WebhookProviderDefinition,
  WebhookProviderId,
} from "./types";
import { docuSealWebhookProvider } from "./providers/docuseal";
import { uptimeKumaWebhookProvider } from "./providers/uptime-kuma";

const providerDefinitions = {
  "uptime-kuma": uptimeKumaWebhookProvider,
  docuseal: docuSealWebhookProvider,
} satisfies Record<WebhookProviderId, WebhookProviderDefinition>;

function getConfiguredProviders(config: WebhookConfig): ConfiguredWebhookProvider[] {
  return Object.values(providerDefinitions)
    .map((definition) => ({
      definition,
      settings: config.providers[definition.id],
    }))
    .filter((provider) => provider.settings.enabled);
}

function authenticate(provider: ConfiguredWebhookProvider, req: Request): boolean {
  if (!provider.settings.secretValue) {
    return true;
  }

  if (!provider.settings.secretHeader) {
    return false;
  }

  return req.headers.get(provider.settings.secretHeader) === provider.settings.secretValue;
}

export function describeWebhookProviders(config: WebhookConfig) {
  return getConfiguredProviders(config)
    .map((provider) => `${provider.definition.id}=${provider.settings.path}`);
}

export async function handleWebhookRequest(
  req: Request,
  settings: AppSettings,
  sendAlert: (message: string) => Promise<void>,
): Promise<Response> {
  if (req.method !== "POST") {
    return new Response("not found", { status: 404 });
  }

  const providersByPath = new Map(
    getConfiguredProviders(settings.webhook).map((provider) => [provider.settings.path, provider]),
  );
  const url = new URL(req.url);
  const provider = providersByPath.get(url.pathname);

  if (!provider) {
    return new Response("not found", { status: 404 });
  }

  if (!authenticate(provider, req)) {
    return new Response(`${provider.definition.label} unauthorized`, { status: 401 });
  }

  let payload: unknown;
  try {
    payload = await req.json();
  } catch {
    return new Response("bad request", { status: 400 });
  }

  console.log(`[webhook] ${provider.definition.id}: ${provider.definition.describeEvent(payload)}`);
  const message = provider.definition.formatMessage(payload);
  if (message) {
    await sendAlert(message);
  }

  return new Response("ok", { status: 200 });
}
