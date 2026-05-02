import type {
  DeliveryTarget,
  DeliveryTargetType,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
  NtfyDeliveryTarget,
  SmtpDeliveryTarget,
} from "./types";

function createId() {
  return globalThis.crypto.randomUUID();
}

function createMatrixDeviceId() {
  return `NOOK${createId().replaceAll("-", "").slice(0, 12).toUpperCase()}`;
}

export function getDeliveryTargetLabel(type: DeliveryTargetType) {
  if (type === "matrix") {
    return "Matrix";
  }

  if (type === "discord") {
    return "Discord webhook";
  }

  if (type === "smtp") {
    return "SMTP";
  }

  if (type === "ntfy") {
    return "ntfy";
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
      providerIds: [],
      homeserver: "",
      accessToken: "",
      refreshToken: null,
      userId: "",
      password: null,
      deviceId: createMatrixDeviceId(),
      roomIds: [],
    };
  }

  if (type === "discord") {
    return {
      id: createId(),
      type,
      name: "Discord webhook",
      enabled: true,
      providerIds: [],
      webhookUrl: "",
    };
  }

  if (type === "smtp") {
    return {
      id: createId(),
      type,
      name: "SMTP",
      enabled: true,
      providerIds: [],
      host: "",
      port: 587,
      secure: true,
      username: "",
      password: null,
      fromAddress: "",
      toAddresses: [],
      subjectPrefix: "[Nook]",
    };
  }

  if (type === "ntfy") {
    return {
      id: createId(),
      type,
      name: "ntfy",
      enabled: true,
      providerIds: [],
      serverUrl: "https://ntfy.sh",
      topic: "",
      accessToken: null,
      priority: 3,
      tags: ["rotating_light"],
      clickUrl: null,
    };
  }

  return {
    id: createId(),
    type,
    name: "HTTP webhook",
    enabled: true,
    providerIds: [],
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

export function isSmtpDeliveryTarget(target: DeliveryTarget): target is SmtpDeliveryTarget {
  return target.type === "smtp";
}

export function isNtfyDeliveryTarget(target: DeliveryTarget): target is NtfyDeliveryTarget {
  return target.type === "ntfy";
}
