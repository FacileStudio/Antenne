import type {
  DeliveryTarget,
  DeliveryTargetType,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
} from "./types";

function createId() {
  return globalThis.crypto.randomUUID();
}

export function getDeliveryTargetLabel(type: DeliveryTargetType) {
  if (type === "matrix") {
    return "Matrix";
  }

  if (type === "discord") {
    return "Discord webhook";
  }

  return "HTTP webhook";
}

export function createDefaultDeliveryTarget(type: DeliveryTargetType): DeliveryTarget {
  if (type === "matrix") {
    return {
      id: createId(),
      type,
      name: "Matrix",
      enabled: true,
      homeserver: "",
      accessToken: "",
      roomIds: [],
    };
  }

  if (type === "discord") {
    return {
      id: createId(),
      type,
      name: "Discord webhook",
      enabled: true,
      webhookUrl: "",
    };
  }

  return {
    id: createId(),
    type,
    name: "HTTP webhook",
    enabled: true,
    url: "",
    secretHeader: "x-nook-secret",
    secretValue: null,
  };
}

export function isMatrixDeliveryTarget(target: DeliveryTarget): target is MatrixDeliveryTarget {
  return target.type === "matrix";
}

export function isDiscordDeliveryTarget(target: DeliveryTarget): target is DiscordDeliveryTarget {
  return target.type === "discord";
}

export function isHttpDeliveryTarget(target: DeliveryTarget): target is HttpDeliveryTarget {
  return target.type === "webhook";
}
