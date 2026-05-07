import {
  isDiscordDeliveryTarget,
  isHttpDeliveryTarget,
  isMatrixDeliveryTarget,
  isNtfyDeliveryTarget,
  isSmtpDeliveryTarget,
} from "./delivery-config";
import { createDefaultDeliveryTarget } from "./delivery-config";
import {
  hasDiscordDelivery,
  hasHttpDelivery,
  hasNtfyDelivery,
  hasSmtpDelivery,
  normalizeNtfyPriority,
  sendDiscordAlert,
  sendHttpAlert,
  sendNtfyAlert,
  sendSmtpAlert,
} from "./delivery-senders";
import type { DeliveryAttempt } from "./event-log";
import { hasMatrixDelivery, sendMatrixAlert } from "./matrix-delivery";
import type {
  DeliveryConfig,
  DeliveryTarget,
  DeliveryTargetType,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
  NtfyDeliveryTarget,
} from "./types";

export interface DeliveryRouteSummary {
  selectedTargets: DeliveryTarget[];
  sendableTargets: DeliveryTarget[];
  skippedByFilterCount: number;
  skippedUnconfiguredTargets: DeliveryTarget[];
}

export class DeliveryError extends Error {
  readonly attempts: DeliveryAttempt[];

  constructor(message: string, attempts: DeliveryAttempt[]) {
    super(message);
    this.name = "DeliveryError";
    this.attempts = attempts;
  }
}

export {
  hasDiscordDelivery,
  hasHttpDelivery,
  hasNtfyDelivery,
  hasSmtpDelivery,
  hasMatrixDelivery,
};

export function hasAnyDelivery(config: DeliveryConfig): boolean {
  return config.items.some((target) => canSendToDeliveryTarget(target));
}

export function hasConfiguredMatrixDelivery(config: DeliveryConfig): boolean {
  return config.items.some((target) => isMatrixDeliveryTarget(target) && hasMatrixDelivery(target));
}

export function canSendToDeliveryTarget(target: DeliveryTarget): boolean {
  if (isMatrixDeliveryTarget(target)) {
    return hasMatrixDelivery(target);
  }

  if (isDiscordDeliveryTarget(target)) {
    return hasDiscordDelivery(target);
  }

  if (isSmtpDeliveryTarget(target)) {
    return hasSmtpDelivery(target);
  }

  if (isNtfyDeliveryTarget(target)) {
    return hasNtfyDelivery(target);
  }

  return hasHttpDelivery(target);
}

export function loadDeliveryConfigFromEnv(): DeliveryConfig {
  const items: DeliveryTarget[] = [];
  const roomIdsRaw = process.env.MATRIX_ROOM_IDS ?? process.env.MATRIX_ROOM_ID;
  const roomIds = roomIdsRaw
    ? roomIdsRaw.split(",").map((id) => id.trim()).filter(Boolean)
    : [];

  const matrix = createDefaultDeliveryTarget("matrix") as MatrixDeliveryTarget;
  matrix.enabled = process.env.MATRIX_ENABLED !== "false";
  matrix.homeserver = process.env.MATRIX_HOMESERVER ?? "";
  matrix.accessToken = process.env.MATRIX_ACCESS_TOKEN ?? "";
  matrix.refreshToken = process.env.MATRIX_REFRESH_TOKEN ?? null;
  matrix.userId = process.env.MATRIX_USER_ID ?? "";
  matrix.password = process.env.MATRIX_PASSWORD ?? null;
  matrix.deviceId = process.env.MATRIX_DEVICE_ID ?? matrix.deviceId;
  matrix.roomIds = roomIds;
  items.push(matrix);

  const discord = createDefaultDeliveryTarget("discord") as DiscordDeliveryTarget;
  discord.enabled = process.env.DISCORD_ENABLED === "true" || Boolean(process.env.DISCORD_WEBHOOK_URL);
  discord.webhookUrl = process.env.DISCORD_WEBHOOK_URL ?? "";
  items.push(discord);

  const webhook = createDefaultDeliveryTarget("webhook") as HttpDeliveryTarget;
  webhook.enabled = process.env.HTTP_WEBHOOK_ENABLED === "true" || Boolean(process.env.HTTP_WEBHOOK_URL);
  webhook.url = process.env.HTTP_WEBHOOK_URL ?? "";
  webhook.secretHeader = process.env.HTTP_WEBHOOK_SECRET_HEADER ?? "x-nook-secret";
  webhook.secretValue = process.env.HTTP_WEBHOOK_SECRET ?? null;
  items.push(webhook);

  const ntfy = createDefaultDeliveryTarget("ntfy") as NtfyDeliveryTarget;
  ntfy.enabled = process.env.NTFY_ENABLED === "true" || Boolean(process.env.NTFY_TOPIC);
  ntfy.serverUrl = process.env.NTFY_SERVER_URL ?? "https://ntfy.sh";
  ntfy.topic = process.env.NTFY_TOPIC ?? "";
  ntfy.accessToken = process.env.NTFY_ACCESS_TOKEN ?? null;
  ntfy.priority = normalizeNtfyPriority(process.env.NTFY_PRIORITY);
  ntfy.tags = process.env.NTFY_TAGS
    ? process.env.NTFY_TAGS.split(",").map((tag) => tag.trim()).filter(Boolean)
    : ["rotating_light"];
  ntfy.clickUrl = process.env.NTFY_CLICK_URL ?? null;
  items.push(ntfy);

  return { items };
}

function isKnownDeliveryTargetType(type: unknown): type is DeliveryTargetType {
  return type === "matrix" || type === "discord" || type === "webhook" || type === "smtp" || type === "ntfy";
}

export function targetAcceptsProvider(target: DeliveryTarget, providerId: string | null): boolean {
  if (!providerId) {
    return false;
  }

  return target.providerIds?.includes(providerId) ?? false;
}

export function getDeliveryRouteSummary(config: DeliveryConfig, providerId: string): DeliveryRouteSummary {
  const selectedTargets = config.items.filter((target) => targetAcceptsProvider(target, providerId));
  const sendableTargets = selectedTargets.filter((target) => canSendToDeliveryTarget(target));

  return {
    selectedTargets,
    sendableTargets,
    skippedByFilterCount: config.items.length - selectedTargets.length,
    skippedUnconfiguredTargets: selectedTargets.filter((target) => !sendableTargets.includes(target)),
  };
}

function createDeliveryTask(target: DeliveryTarget, message: string): Promise<void> | null {
  if (isMatrixDeliveryTarget(target)) {
    return hasMatrixDelivery(target) ? sendMatrixAlert(target, message) : null;
  }

  if (isDiscordDeliveryTarget(target)) {
    return hasDiscordDelivery(target) ? sendDiscordAlert(target, message) : null;
  }

  if (isSmtpDeliveryTarget(target)) {
    return hasSmtpDelivery(target) ? sendSmtpAlert(target, message) : null;
  }

  if (isNtfyDeliveryTarget(target)) {
    return hasNtfyDelivery(target) ? sendNtfyAlert(target, message) : null;
  }

  return isHttpDeliveryTarget(target) && hasHttpDelivery(target) ? sendHttpAlert(target, message) : null;
}

export async function sendAlert(config: DeliveryConfig, message: string, providerId: string): Promise<DeliveryAttempt[]> {
  if (!hasAnyDelivery(config)) {
    throw new Error("No delivery channel is configured.");
  }

  for (const target of config.items) {
    if (!isKnownDeliveryTargetType(target.type)) {
      throw new Error(`Unknown delivery target type: ${String(target.type)}`);
    }
  }

  const route = getDeliveryRouteSummary(config, providerId);
  const targets: DeliveryTarget[] = route.sendableTargets;
  const tasks: Promise<void>[] = [];

  for (const target of targets) {
    const task = createDeliveryTask(target, message);
    if (task) {
      tasks.push(task);
    }
  }

  if (tasks.length === 0) {
    return [];
  }

  const startTimes = targets.map(() => Date.now());
  const results = await Promise.allSettled(tasks);
  const now = Date.now();

  const attempts: DeliveryAttempt[] = results.map((result, index) => {
    const target = targets[index]!;
    return {
      targetId: target.id,
      targetName: target.name,
      targetType: target.type,
      status: result.status === "fulfilled" ? "success" as const : "error" as const,
      error: result.status === "rejected" ? String(result.reason) : undefined,
      durationMs: now - startTimes[index]!,
    };
  });

  const failures = attempts.filter((attempt) => attempt.status === "error");
  for (const failure of failures) {
    console.error(`Failed to send alert: ${failure.error}`);
  }

  if (failures.length === tasks.length) {
    throw new DeliveryError("All configured delivery channels failed.", attempts);
  }

  return attempts;
}

export async function sendAlertToTarget(target: DeliveryTarget, message: string) {
  if (!isKnownDeliveryTargetType(target.type)) {
    throw new Error(`Unknown delivery target type: ${String(target.type)}`);
  }

  if (isMatrixDeliveryTarget(target) && hasMatrixDelivery(target)) {
    await sendMatrixAlert(target, message);
    return;
  }

  if (isDiscordDeliveryTarget(target) && hasDiscordDelivery(target)) {
    await sendDiscordAlert(target, message);
    return;
  }

  if (isSmtpDeliveryTarget(target) && hasSmtpDelivery(target)) {
    await sendSmtpAlert(target, message);
    return;
  }

  if (isNtfyDeliveryTarget(target) && hasNtfyDelivery(target)) {
    await sendNtfyAlert(target, message);
    return;
  }

  if (isHttpDeliveryTarget(target) && hasHttpDelivery(target)) {
    await sendHttpAlert(target, message);
    return;
  }

  throw new Error(`Target \`${target.name}\` is not fully configured.`);
}
