import type { ProviderType, DeliveryTargetType } from "../types";

export const PROVIDER_ICONS: Record<string, string> = {
  system: "solar:settings-bold-duotone",
  website: "solar:global-bold-duotone",
  imap: "solar:letter-bold-duotone",
  rss: "solar:feed-bold-duotone",
  "uptime-kuma": "solar:heart-pulse-bold-duotone",
  docuseal: "solar:document-bold-duotone",
  dokploy: "solar:server-bold-duotone",
  git: "solar:code-bold-duotone",
  vikunja: "solar:checklist-bold-duotone",
  ardoise: "solar:bill-list-bold-duotone",
  sablier: "solar:hourglass-bold-duotone",
  perception: "solar:eye-bold-duotone",
  opus: "solar:widget-2-bold-duotone",
  vision: "solar:chart-2-bold-duotone",
  plume: "solar:pen-new-square-bold-duotone",
  nuage: "solar:cloud-bold-duotone",
};

export const DELIVERY_ICONS: Record<string, string> = {
  matrix: "solar:chat-round-dots-bold-duotone",
  discord: "solar:gamepad-bold-duotone",
  webhook: "solar:link-bold-duotone",
  smtp: "solar:letter-bold-duotone",
  ntfy: "solar:bell-bold-duotone",
  perception: "solar:eye-bold-duotone",
};

export const DEFAULT_ICON = "solar:widget-2-bold-duotone";

export function getProviderIcon(type: string): string {
  return PROVIDER_ICONS[type] ?? DEFAULT_ICON;
}

export function getDeliveryIcon(type: string): string {
  return DELIVERY_ICONS[type] ?? DEFAULT_ICON;
}
