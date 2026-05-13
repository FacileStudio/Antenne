import { createDefaultDeliveryTarget } from "./delivery-config";
import type {
  AppSettings,
  DeliveryTarget,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
  NtfyDeliveryTarget,
  PerceptionDeliveryTarget,
  SmtpDeliveryTarget,
} from "./types";
import {
  normalizeBoolean,
  normalizeInteger,
  normalizeList,
  normalizeOptionalInteger,
  normalizeProviderIds,
  normalizeString,
} from "./settings-normalize-shared";

export function normalizeDeliveryTarget(input: unknown): DeliveryTarget {
  const candidate = input && typeof input === "object" ? input as Partial<DeliveryTarget> & Record<string, unknown> : {};
  const type = candidate.type;
  const providerIds = normalizeProviderIds(candidate.providerIds);
  const providerTags = normalizeList(candidate.providerTags);

  if (type === "matrix") {
    const fallback = createDefaultDeliveryTarget("matrix") as MatrixDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      providerTags,
      homeserver: "homeserver" in candidate ? normalizeString(candidate.homeserver) : fallback.homeserver,
      accessToken: "accessToken" in candidate ? normalizeString(candidate.accessToken) : fallback.accessToken,
      refreshToken: "refreshToken" in candidate ? (normalizeString(candidate.refreshToken) || null) : fallback.refreshToken,
      userId: "userId" in candidate ? normalizeString(candidate.userId) : fallback.userId,
      password: "password" in candidate ? (normalizeString(candidate.password) || null) : fallback.password,
      deviceId: "deviceId" in candidate ? (normalizeString(candidate.deviceId) || null) : fallback.deviceId,
      roomIds: "roomIds" in candidate ? normalizeList(candidate.roomIds) : fallback.roomIds,
    };
  }

  if (type === "discord") {
    const fallback = createDefaultDeliveryTarget("discord") as DiscordDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      providerTags,
      webhookUrl: "webhookUrl" in candidate ? normalizeString(candidate.webhookUrl) : fallback.webhookUrl,
    };
  }

  if (type === "smtp") {
    const fallback = createDefaultDeliveryTarget("smtp") as SmtpDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      providerTags,
      host: "host" in candidate ? normalizeString(candidate.host) : fallback.host,
      port: "port" in candidate ? normalizeInteger(candidate.port, fallback.port, "SMTP port") : fallback.port,
      secure: "secure" in candidate ? normalizeBoolean(candidate.secure, fallback.secure) : fallback.secure,
      username: "username" in candidate ? normalizeString(candidate.username) : fallback.username,
      password: "password" in candidate ? (normalizeString(candidate.password) || null) : fallback.password,
      fromAddress: "fromAddress" in candidate ? normalizeString(candidate.fromAddress) : fallback.fromAddress,
      toAddresses: "toAddresses" in candidate ? normalizeList(candidate.toAddresses) : fallback.toAddresses,
      subjectPrefix: "subjectPrefix" in candidate ? normalizeString(candidate.subjectPrefix) : fallback.subjectPrefix,
    };
  }

  if (type === "ntfy") {
    const fallback = createDefaultDeliveryTarget("ntfy") as NtfyDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      providerTags,
      serverUrl: "serverUrl" in candidate ? normalizeString(candidate.serverUrl) : fallback.serverUrl,
      topic: "topic" in candidate ? normalizeString(candidate.topic) : fallback.topic,
      accessToken: "accessToken" in candidate ? (normalizeString(candidate.accessToken) || null) : fallback.accessToken,
      priority: "priority" in candidate ? normalizeOptionalInteger(candidate.priority, fallback.priority, "ntfy priority") : fallback.priority,
      tags: "tags" in candidate ? normalizeList(candidate.tags) : fallback.tags,
      clickUrl: "clickUrl" in candidate ? (normalizeString(candidate.clickUrl) || null) : fallback.clickUrl,
    };
  }

  if (type === "perception") {
    const fallback = createDefaultDeliveryTarget("perception") as PerceptionDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      providerTags,
      url: "url" in candidate ? normalizeString(candidate.url) : fallback.url,
      workspace: "workspace" in candidate ? normalizeString(candidate.workspace) : fallback.workspace,
      secretValue: "secretValue" in candidate ? (normalizeString(candidate.secretValue) || null) : fallback.secretValue,
      includeRawPayload: "includeRawPayload" in candidate ? normalizeBoolean(candidate.includeRawPayload, fallback.includeRawPayload) : fallback.includeRawPayload,
    };
  }

  if (typeof type === "string" && type !== "webhook") {
    throw new Error(`Unknown delivery target type \`${type}\`.`);
  }

  const fallback = createDefaultDeliveryTarget("webhook") as HttpDeliveryTarget;
  return {
    ...fallback,
    id: normalizeString(candidate.id) || fallback.id,
    name: normalizeString(candidate.name) || fallback.name,
    enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
    providerIds,
    providerTags,
    url: "url" in candidate ? normalizeString(candidate.url) : fallback.url,
    secretValue: "secretValue" in candidate ? (normalizeString(candidate.secretValue) || null) : fallback.secretValue,
    includeRawPayload: "includeRawPayload" in candidate ? normalizeBoolean(candidate.includeRawPayload, fallback.includeRawPayload) : fallback.includeRawPayload,
  };
}

export function normalizeDeliveryConfig(input: unknown, fallback: AppSettings["delivery"]): AppSettings["delivery"] {
  const candidate = input && typeof input === "object" ? input as {
    delivery?: unknown;
    matrix?: unknown;
    discord?: unknown;
    webhook?: unknown;
  } : {};
  const itemsContainer = candidate.delivery && typeof candidate.delivery === "object"
    ? candidate.delivery as { items?: unknown; matrix?: unknown; discord?: unknown; webhook?: unknown }
    : {};

  const rawItems = Array.isArray(itemsContainer.items) ? itemsContainer.items : null;
  if (rawItems) {
    return { items: rawItems.map((target) => normalizeDeliveryTarget(target)) };
  }

  const matrixInput = itemsContainer.matrix ?? candidate.matrix;
  const discordInput = itemsContainer.discord ?? candidate.discord;
  const webhookInput = itemsContainer.webhook ?? candidate.webhook;

  return {
    items: fallback.items.map((target) => {
      if (target.type === "matrix") {
        const matrix = matrixInput && typeof matrixInput === "object" ? matrixInput as Partial<typeof target> : {};
        return {
          ...target,
          enabled: typeof matrix.enabled === "boolean" ? matrix.enabled : target.enabled,
          homeserver: "homeserver" in matrix ? normalizeString(matrix.homeserver) : target.homeserver,
          accessToken: "accessToken" in matrix ? normalizeString(matrix.accessToken) : target.accessToken,
          refreshToken: "refreshToken" in matrix ? (normalizeString(matrix.refreshToken) || null) : target.refreshToken,
          userId: "userId" in matrix ? normalizeString(matrix.userId) : target.userId,
          password: "password" in matrix ? (normalizeString(matrix.password) || null) : target.password,
          deviceId: "deviceId" in matrix ? (normalizeString(matrix.deviceId) || null) : target.deviceId,
          roomIds: "roomIds" in matrix ? normalizeList(matrix.roomIds) : target.roomIds,
        };
      }

      if (target.type === "discord") {
        const discord = discordInput && typeof discordInput === "object" ? discordInput as Partial<typeof target> : {};
        return {
          ...target,
          enabled: typeof discord.enabled === "boolean" ? discord.enabled : target.enabled,
          webhookUrl: "webhookUrl" in discord ? normalizeString(discord.webhookUrl) : target.webhookUrl,
        };
      }

      if (target.type === "smtp" || target.type === "ntfy" || target.type === "perception") {
        return target;
      }

      const webhook = webhookInput && typeof webhookInput === "object" ? webhookInput as Partial<typeof target> : {};
      return {
        ...target,
        enabled: typeof webhook.enabled === "boolean" ? webhook.enabled : target.enabled,
        url: "url" in webhook ? normalizeString(webhook.url) : target.url,
        secretValue: "secretValue" in webhook ? (normalizeString(webhook.secretValue) || null) : target.secretValue,
      };
    }),
  };
}
