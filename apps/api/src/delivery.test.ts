import { afterEach, describe, expect, test } from "bun:test";

import {
  getDeliveryRouteSummary,
  sendAlert,
  targetAcceptsProvider,
} from "./delivery";
import { buildNookEventEnvelope } from "./event-envelope";
import { SYSTEM_PROVIDER_ID, type DeliveryMessage, type DeliveryTarget } from "./types";
import { providerTypes } from "./webhooks/types";

const originalFetch = globalThis.fetch;
const providerIds = [SYSTEM_PROVIDER_ID, ...providerTypes.map((type) => `${type}-provider`)];

function createDeliveryMessage(message = "<b>event</b>"): DeliveryMessage {
  return {
    message,
    envelope: buildNookEventEnvelope({
      source: "test",
      providerId: "website-provider",
      providerName: "Website",
      providerType: "website",
      eventType: "test.event",
      eventTitle: "Test event",
      messageHtml: message,
      subject: {
        type: "system",
        id: "test",
        name: "Test",
        url: null,
      },
      normalizedPayload: {
        ok: true,
      },
      rawPayload: {
        secret: "raw",
      },
      rawPayloadAllowed: true,
    }),
  };
}

function createTargets(providerIdsForTargets = providerIds): DeliveryTarget[] {
  return [
    {
      id: "matrix-target",
      type: "matrix",
      name: "Matrix target",
      enabled: true,
      providerIds: providerIdsForTargets,
      providerTags: [],
      homeserver: "https://matrix.example",
      accessToken: "matrix-token",
      refreshToken: null,
      userId: "",
      password: null,
      deviceId: null,
      roomIds: ["!room:matrix.example"],
    },
    {
      id: "discord-target",
      type: "discord",
      name: "Discord target",
      enabled: true,
      providerIds: providerIdsForTargets,
      providerTags: [],
      webhookUrl: "https://discord.example/webhook",
    },
    {
      id: "http-target",
      type: "webhook",
      name: "HTTP target",
      enabled: true,
      providerIds: providerIdsForTargets,
      providerTags: [],
      url: "https://hooks.example/nook",
      secretValue: "secret",
      includeRawPayload: false,
    },
    {
      id: "smtp-target",
      type: "smtp",
      name: "SMTP target",
      enabled: true,
      providerIds: providerIdsForTargets,
      providerTags: [],
      host: "smtp.example",
      port: 587,
      secure: true,
      username: "nook",
      password: null,
      fromAddress: "nook@example.com",
      toAddresses: ["alerts@example.com"],
      subjectPrefix: "[Nook]",
    },
    {
      id: "ntfy-target",
      type: "ntfy",
      name: "ntfy target",
      enabled: true,
      providerIds: providerIdsForTargets,
      providerTags: [],
      serverUrl: "https://ntfy.example",
      topic: "alerts",
      accessToken: null,
      priority: 3,
      tags: ["rotating_light"],
      clickUrl: null,
    },
    {
      id: "perception-target",
      type: "perception",
      name: "Perception target",
      enabled: true,
      providerIds: providerIdsForTargets,
      providerTags: [],
      url: "https://perception.example/ingest",
      workspace: "test-workspace",
      secretValue: "perception-secret",
      includeRawPayload: false,
    },
  ];
}

afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("delivery provider filters", () => {
  test("all delivery target types accept only explicitly selected provider ids", () => {
    for (const target of createTargets()) {
      for (const providerId of providerIds) {
        expect(targetAcceptsProvider(target, providerId)).toBe(true);
      }

      expect(targetAcceptsProvider(target, "not-selected")).toBe(false);
      expect(targetAcceptsProvider(target, null)).toBe(false);
      expect(targetAcceptsProvider({ ...target, providerIds: [] }, providerIds[0]!)).toBe(false);
    }
  });

  test("route summary covers every provider source and delivery target type", () => {
    const targets = createTargets();

    for (const providerId of providerIds) {
      const summary = getDeliveryRouteSummary({ items: targets }, providerId);
      expect(summary.selectedTargets.map((target) => target.type).sort()).toEqual(["discord", "matrix", "ntfy", "perception", "smtp", "webhook"]);
      expect(summary.sendableTargets.map((target) => target.type).sort()).toEqual(["discord", "matrix", "ntfy", "perception", "smtp", "webhook"]);
      expect(summary.skippedByFilterCount).toBe(0);
      expect(summary.skippedUnconfiguredTargets).toHaveLength(0);
    }
  });

  test("unselected events produce no delivery attempts and do not touch network senders", async () => {
    let fetchCalls = 0;
    globalThis.fetch = (async () => {
      fetchCalls += 1;
      throw new Error("fetch should not run");
    }) as unknown as typeof fetch;

    const deliveries = await sendAlert({ items: createTargets([]) }, createDeliveryMessage(), "website-provider");

    expect(deliveries).toEqual([]);
    expect(fetchCalls).toBe(0);
  });

  test("selected Discord target sends while unselected Discord target stays quiet", async () => {
    const urls: string[] = [];
    globalThis.fetch = (async (input: Parameters<typeof fetch>[0]) => {
      urls.push(String(input));
      return new Response("ok", { status: 204 });
    }) as unknown as typeof fetch;

    const selectedTarget = createTargets(["website-provider"])
      .find((target) => target.type === "discord");
    const deliveries = await sendAlert({
      items: [
        selectedTarget!,
        {
          ...selectedTarget!,
          id: "discord-unselected",
          name: "Discord unselected",
          webhookUrl: "https://discord.example/unselected",
          providerIds: ["imap-provider"],
        },
      ],
    }, createDeliveryMessage(), "website-provider");

    expect(deliveries).toHaveLength(1);
    expect(deliveries[0]?.targetId).toBe("discord-target");
    expect(urls).toEqual(["https://discord.example/webhook"]);
  });

  test("HTTP target sends signed event envelope without raw payload by default", async () => {
    let headers = new Headers();
    let requestBody: any = null;
    globalThis.fetch = (async (_input: Parameters<typeof fetch>[0], init?: Parameters<typeof fetch>[1]) => {
      headers = new Headers(init?.headers);
      requestBody = JSON.parse(String(init?.body));
      return new Response("ok", { status: 204 });
    }) as unknown as typeof fetch;

    const target = createTargets(["website-provider"]).find((item) => item.type === "webhook")!;
    const deliveries = await sendAlert({ items: [target] }, createDeliveryMessage(), "website-provider");

    expect(deliveries).toHaveLength(1);
    expect(headers.get("x-nook-event-id")).toBe(requestBody.id);
    expect(headers.get("x-nook-timestamp")).toBeTruthy();
    expect(headers.get("x-nook-signature-256")).toStartWith("sha256=");
    expect(requestBody.schemaVersion).toBe(1);
    expect(requestBody.payload.raw).toBeNull();
  });

  test("perception target sends signed event envelope", async () => {
    let url = "";
    let headers = new Headers();
    let requestBody: any = null;
    globalThis.fetch = (async (input: Parameters<typeof fetch>[0], init?: Parameters<typeof fetch>[1]) => {
      url = String(input);
      headers = new Headers(init?.headers);
      requestBody = JSON.parse(String(init?.body));
      return new Response("ok", { status: 204 });
    }) as unknown as typeof fetch;

    const target = createTargets(["website-provider"]).find((item) => item.type === "perception")!;
    const deliveries = await sendAlert({ items: [target] }, createDeliveryMessage(), "website-provider");

    expect(deliveries).toHaveLength(1);
    expect(url).toBe("https://perception.example/ingest");
    expect(headers.get("x-nook-event-id")).toBe(requestBody.id);
    expect(headers.get("x-nook-signature-256")).toStartWith("sha256=");
    expect(headers.get("x-perception-workspace")).toBe("test-workspace");
    expect(requestBody.provider.type).toBe("website");
    expect(requestBody.payload.raw).toBeNull();
  });
});
