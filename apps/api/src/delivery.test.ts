import { afterEach, describe, expect, test } from "bun:test";

import {
  getDeliveryRouteSummary,
  sendAlert,
  targetAcceptsProvider,
} from "./delivery";
import { SYSTEM_PROVIDER_ID, type DeliveryTarget } from "./types";
import { providerTypes } from "./webhooks/types";

const originalFetch = globalThis.fetch;
const providerIds = [SYSTEM_PROVIDER_ID, ...providerTypes.map((type) => `${type}-provider`)];

function createTargets(providerIdsForTargets = providerIds): DeliveryTarget[] {
  return [
    {
      id: "matrix-target",
      type: "matrix",
      name: "Matrix target",
      enabled: true,
      providerIds: providerIdsForTargets,
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
      webhookUrl: "https://discord.example/webhook",
    },
    {
      id: "http-target",
      type: "webhook",
      name: "HTTP target",
      enabled: true,
      providerIds: providerIdsForTargets,
      url: "https://hooks.example/nook",
      secretHeader: "x-nook-secret",
      secretValue: "secret",
    },
    {
      id: "smtp-target",
      type: "smtp",
      name: "SMTP target",
      enabled: true,
      providerIds: providerIdsForTargets,
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
      serverUrl: "https://ntfy.example",
      topic: "alerts",
      accessToken: null,
      priority: 3,
      tags: ["rotating_light"],
      clickUrl: null,
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
      expect(summary.selectedTargets.map((target) => target.type).sort()).toEqual(["discord", "matrix", "ntfy", "smtp", "webhook"]);
      expect(summary.sendableTargets.map((target) => target.type).sort()).toEqual(["discord", "matrix", "ntfy", "smtp", "webhook"]);
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

    const deliveries = await sendAlert({ items: createTargets([]) }, "<b>event</b>", "website-provider");

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
    }, "<b>event</b>", "website-provider");

    expect(deliveries).toHaveLength(1);
    expect(deliveries[0]?.targetId).toBe("discord-target");
    expect(urls).toEqual(["https://discord.example/webhook"]);
  });
});
