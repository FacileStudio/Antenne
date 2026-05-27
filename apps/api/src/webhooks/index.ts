import { timingSafeEqual } from "node:crypto";
import type { AppSettings } from "../types";
import type { DeliveryAttempt } from "../event-log";
import type { AlertContext } from "../types";
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
import { gitWebhookProvider } from "./providers/git";
import { uptimeKumaWebhookProvider } from "./providers/uptime-kuma";
import { vikunjaWebhookProvider } from "./providers/vikunja";
import { ardoiseWebhookProvider } from "./providers/ardoise";
import { sablierWebhookProvider } from "./providers/sablier";
import { perceptionWebhookProvider } from "./providers/perception";
import { opusWebhookProvider } from "./providers/opus";
import { visionWebhookProvider } from "./providers/vision";
import { plumeWebhookProvider } from "./providers/plume";
import { nuageWebhookProvider } from "./providers/nuage";
import { isWebhookProviderSettings } from "./types";

const providerDefinitions = {
  "uptime-kuma": uptimeKumaWebhookProvider,
  docuseal: docuSealWebhookProvider,
  dokploy: dokployWebhookProvider,
  git: gitWebhookProvider,
  vikunja: vikunjaWebhookProvider,
  ardoise: ardoiseWebhookProvider,
  sablier: sablierWebhookProvider,
  perception: perceptionWebhookProvider,
  opus: opusWebhookProvider,
  vision: visionWebhookProvider,
  plume: plumeWebhookProvider,
  nuage: nuageWebhookProvider,
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

function safeHeaderEquals(header: string | null, expected: string): boolean {
  if (header === null) return false;
  const a = Buffer.from(header);
  const b = Buffer.from(expected);
  if (a.length !== b.length) return false;
  return timingSafeEqual(a, b);
}

function authenticate(provider: ConfiguredWebhookProvider, req: Request, rawBody: string): boolean | Promise<boolean> {
  if (provider.definition.authenticate) {
    return provider.definition.authenticate(req, rawBody, provider.settings);
  }

  if (!provider.settings.secretValue) {
    return true;
  }

  if (!provider.settings.secretHeader) {
    return false;
  }

  return safeHeaderEquals(req.headers.get(provider.settings.secretHeader), provider.settings.secretValue);
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

export interface WebhookAlertContext extends AlertContext {
  providerType: WebhookProviderType;
}

export async function handleWebhookRequest(
  req: Request,
  settings: AppSettings,
  sendAlert: (message: string, context: WebhookAlertContext) => Promise<DeliveryAttempt[]>,
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

  const rawBody = await req.text();

  if (!(await authenticate(provider, req, rawBody))) {
    return new Response(`${provider.definition.label} unauthorized`, { status: 401 });
  }

  let payload: unknown;
  try {
    payload = JSON.parse(rawBody);
  } catch {
    return new Response("bad request", { status: 400 });
  }

  const eventDescription = provider.definition.describeEvent(payload);
  console.log(`[webhook] ${provider.settings.name} (${provider.definition.type}): ${eventDescription}`);
  const message = provider.definition.formatMessage(payload, createDispatchContext(provider.settings));
  if (message) {
    const event = provider.definition.buildEvent?.(payload, createDispatchContext(provider.settings)) ?? {};
    await sendAlert(message, {
      providerId: provider.settings.id,
      providerName: provider.settings.name,
      providerType: provider.settings.type,
      event: eventDescription,
      providerTags: provider.settings.tags,
      eventType: event.eventType ?? eventDescription,
      eventTitle: event.eventTitle ?? eventDescription,
      eventUrl: event.eventUrl ?? null,
      actor: event.actor ?? null,
      subject: event.subject ?? null,
      normalizedPayload: event.normalizedPayload ?? {},
      rawPayload: payload,
      occurredAt: event.occurredAt ?? null,
      containsPii: event.containsPii ?? (provider.settings.type === "docuseal" || provider.settings.type === "ardoise"),
      rawPayloadAllowed: event.rawPayloadAllowed ?? false,
    });
  }

  return new Response("ok", { status: 200 });
}
