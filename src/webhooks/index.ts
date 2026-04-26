import type { AppSettings } from "../types";
import type {
  ConfiguredWebhookProvider,
  ProvidersConfigInput,
  WebhookProviderDefinition,
  WebhookProviderSettings,
  WebhookProviderType,
  WebhookDispatchContext,
} from "./types";
import { docuSealWebhookProvider } from "./providers/docuseal";
import { dokployWebhookProvider } from "./providers/dokploy";
import { uptimeKumaWebhookProvider } from "./providers/uptime-kuma";
import { isWebhookProviderSettings } from "./types";

const providerDefinitions = {
  "uptime-kuma": uptimeKumaWebhookProvider,
  docuseal: docuSealWebhookProvider,
  dokploy: dokployWebhookProvider,
} satisfies Record<WebhookProviderType, WebhookProviderDefinition>;

function getConfiguredProviders(config: ProvidersConfigInput): ConfiguredWebhookProvider[] {
  return config.items
    .filter(isWebhookProviderSettings)
    .filter((provider) => provider.enabled)
    .map((provider) => ({
      definition: providerDefinitions[provider.type],
      settings: provider,
    }));
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

export function describeWebhookProviders(config: ProvidersConfigInput) {
  return getConfiguredProviders(config)
    .map((provider) => `${provider.settings.name} (${provider.definition.type})=${provider.settings.path}`);
}

function createDispatchContext(provider: WebhookProviderSettings): WebhookDispatchContext {
  return {
    providerId: provider.id,
    providerName: provider.name,
    providerType: provider.type,
    path: provider.path,
  };
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
    getConfiguredProviders(settings.providers).map((provider) => [provider.settings.path, provider]),
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

  console.log(`[webhook] ${provider.settings.name} (${provider.definition.type}): ${provider.definition.describeEvent(payload)}`);
  const message = provider.definition.formatMessage(payload, createDispatchContext(provider.settings));
  if (message) {
    await sendAlert(message);
  }

  return new Response("ok", { status: 200 });
}
