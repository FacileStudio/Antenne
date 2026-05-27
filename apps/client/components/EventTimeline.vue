<script setup lang="ts">
import { Icon } from "@iconify/vue";

import type { EventLogEntry } from "../types";

defineProps<{
  events: EventLogEntry[];
}>();

const emit = defineEmits<{
  replay: [eventId: string];
}>();

function formatTime(iso: string) {
  const d = new Date(iso);
  const now = Date.now();
  const diff = now - d.getTime();
  if (diff < 60_000) return "just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

function sourceIcon(source: string) {
  if (source === "website") return "solar:global-linear";
  if (source === "webhook") return "solar:link-round-angle-linear";
  if (source === "imap") return "solar:letter-linear";
  if (source === "rss") return "solar:rss-linear";
  if (source === "test") return "solar:test-tube-linear";
  return "solar:server-linear";
}

function sourceMeta(source: string) {
  if (source === "website") return { label: "Website", tone: "website" };
  if (source === "webhook") return { label: "Webhook", tone: "webhook" };
  if (source === "imap") return { label: "Inbox", tone: "imap" };
  if (source === "test") return { label: "Test", tone: "test" };
  return { label: "System", tone: "system" };
}

function decodeHtml(value: string) {
  return value
    .replaceAll("&nbsp;", " ")
    .replaceAll("&amp;", "&")
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">")
    .replaceAll("&quot;", "\"")
    .replaceAll("&#39;", "'");
}

function plainText(value: string | null) {
  if (!value) {
    return "";
  }

  return decodeHtml(
    value
      .replace(/<br\s*\/?>/gi, "\n")
      .replace(/<\/p>/gi, "\n")
      .replace(/<[^>]+>/g, " "),
  )
    .split("\n")
    .map((line) => line.replace(/\s+/g, " ").trim())
    .filter(Boolean)
    .join("\n");
}

function titleFor(entry: EventLogEntry) {
  const cleanEvent = plainText(entry.event);
  const cleanMessage = plainText(entry.message);
  const messageLines = cleanMessage.split("\n").filter(Boolean);

  if (entry.source === "system" && messageLines.length > 0) {
    return messageLines[0]!;
  }

  if (cleanEvent) {
    return cleanEvent;
  }

  if (messageLines.length > 0) {
    return messageLines[0]!;
  }

  return sourceMeta(entry.source).label;
}

function previewFor(entry: EventLogEntry) {
  const cleanMessage = plainText(entry.message);
  if (!cleanMessage) {
    return "";
  }

  const lines = cleanMessage.split("\n").filter(Boolean);
  const title = titleFor(entry);

  if (lines.length > 1 && lines[0] === title) {
    return lines.slice(1).join(" ");
  }

  if (cleanMessage !== title) {
    return cleanMessage;
  }

  return "";
}
</script>

<template>
  <div class="rounded-xl bg-card ring-1 ring-foreground/10 overflow-hidden">
    <p v-if="events.length === 0" class="px-5 py-8 text-center text-muted-foreground text-sm m-0">
      No events recorded yet.
    </p>

    <div v-else class="divide-y divide-border">
      <div
        v-for="entry in events"
        :key="entry.id"
        class="flex items-start gap-3 px-4 py-3 hover:bg-muted/30 transition-colors"
      >
        <div class="mt-0.5 flex-none rounded-lg bg-muted p-1.5 text-muted-foreground">
          <Icon :icon="sourceIcon(entry.source)" :width="14" :height="14" aria-hidden="true" />
        </div>

        <div class="min-w-0 flex-1 grid gap-1">
          <div class="flex items-baseline justify-between gap-2">
            <span class="font-semibold text-sm leading-tight truncate">{{ titleFor(entry) }}</span>
            <span class="flex-none text-xs text-muted-foreground tabular-nums whitespace-nowrap">{{ formatTime(entry.timestamp) }}</span>
          </div>

          <div class="flex flex-wrap items-center gap-1.5">
            <span
              class="inline-flex items-center rounded-full px-2 py-0.5 text-[0.7rem] leading-none ring-1"
              :class="{
                'bg-blue-50 text-blue-700 ring-blue-200': sourceMeta(entry.source).tone === 'system',
                'bg-green-50 text-green-700 ring-green-200': sourceMeta(entry.source).tone === 'website',
                'bg-amber-50 text-amber-700 ring-amber-200': sourceMeta(entry.source).tone === 'webhook',
                'bg-purple-50 text-purple-700 ring-purple-200': sourceMeta(entry.source).tone === 'imap',
                'bg-orange-50 text-orange-700 ring-orange-200': sourceMeta(entry.source).tone === 'test',
              }"
            >{{ sourceMeta(entry.source).label }}</span>
            <span
              v-if="entry.providerName && entry.providerName !== 'system' && entry.providerName !== 'Nook'"
              class="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[0.7rem] leading-none text-muted-foreground ring-1 ring-foreground/10"
            >{{ entry.providerName }}</span>
          </div>

          <p v-if="previewFor(entry)" class="m-0 text-sm text-muted-foreground leading-relaxed line-clamp-2">
            {{ previewFor(entry) }}
          </p>

          <div v-if="entry.deliveries.length > 0" class="flex flex-wrap items-center gap-1.5 mt-0.5">
            <span
              v-for="d in entry.deliveries"
              :key="d.targetId"
              class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[0.7rem] leading-none ring-1"
              :class="{
                'bg-green-50 text-green-700 ring-green-200': d.status === 'success',
                'bg-red-50 text-red-700 ring-red-200': d.status === 'error',
              }"
              :title="d.error ?? `${d.durationMs}ms`"
            >
              <Icon v-if="d.status === 'success'" icon="solar:check-circle-linear" :width="11" :height="11" aria-hidden="true" />
              <Icon v-else icon="solar:close-circle-linear" :width="11" :height="11" aria-hidden="true" />
              {{ d.targetName }}
            </span>
          </div>

          <button
            v-if="entry.envelope"
            class="mt-1 inline-flex w-fit items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            type="button"
            title="Replay this event"
            @click="emit('replay', entry.id)"
          >
            <Icon icon="solar:refresh-linear" :width="12" :height="12" aria-hidden="true" />
            Replay
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
