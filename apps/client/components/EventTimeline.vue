<script setup lang="ts">
import { Clock, CheckCircle2, XCircle, RotateCcw } from "lucide-vue-next";

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
  <div class="grid space-y-2.5">
    <p v-if="events.length === 0" class="text-center text-muted-foreground py-6 px-5 rounded-xl border border-dashed border-border bg-black/[0.02]">No events recorded yet.</p>
    <div v-for="entry in events" :key="entry.id" class="p-4 border border-border rounded-xl bg-black/[0.02]">
      <div class="grid gap-1.5 min-w-0">
        <div class="flex justify-between items-baseline gap-2 max-sm:flex-col max-sm:items-start">
          <span class="font-semibold leading-tight">{{ titleFor(entry) }}</span>
          <span class="flex-none inline-flex items-center gap-1 text-xs text-muted-foreground">
            <Clock :size="12" />
            {{ formatTime(entry.timestamp) }}
          </span>
        </div>
        <div class="flex flex-wrap gap-1.5 items-center">
          <span
            class="inline-flex items-center min-h-5 px-2 py-0.5 rounded-full border border-border bg-black/[0.03] text-muted-foreground text-[0.73rem] whitespace-nowrap"
            :class="[
              sourceMeta(entry.source).tone === 'system' && 'text-[oklch(38%_0.14_250)] bg-[oklch(95%_0.04_250)] border-[oklch(84%_0.07_250)]',
              sourceMeta(entry.source).tone === 'website' && 'text-[oklch(37%_0.13_155)] bg-[oklch(95%_0.05_155)] border-[oklch(83%_0.08_155)]',
              sourceMeta(entry.source).tone === 'webhook' && 'text-[oklch(42%_0.11_65)] bg-[oklch(96%_0.04_65)] border-[oklch(86%_0.07_65)]',
              sourceMeta(entry.source).tone === 'imap' && 'text-[oklch(40%_0.14_310)] bg-[oklch(95%_0.04_310)] border-[oklch(84%_0.07_310)]',
              sourceMeta(entry.source).tone === 'test' && 'text-[oklch(44%_0.12_45)] bg-[oklch(96%_0.04_45)] border-[oklch(86%_0.07_45)]',
            ]"
          >{{ sourceMeta(entry.source).label }}</span>
          <span v-if="entry.providerName && entry.providerName !== 'system' && entry.providerName !== 'Nook'" class="inline-flex items-center min-h-5 px-2 py-0.5 rounded-full border border-border bg-black/[0.03] text-muted-foreground text-[0.73rem] whitespace-nowrap">{{ entry.providerName }}</span>
        </div>
        <p v-if="previewFor(entry)" class="m-0 text-muted-foreground text-sm leading-relaxed line-clamp-2">{{ previewFor(entry) }}</p>
        <div v-if="entry.deliveries.length > 0" class="flex flex-wrap gap-1.5 mt-0.5">
          <span
            v-for="d in entry.deliveries"
            :key="d.targetId"
            class="inline-flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full border border-border bg-black/[0.03]"
            :class="[
              d.status === 'success' && 'text-[oklch(37%_0.14_155)] border-[oklch(83%_0.08_155)] bg-[oklch(95%_0.05_155)]',
              d.status === 'error' && 'text-[oklch(43%_0.16_25)] border-[oklch(84%_0.08_25)] bg-[oklch(96%_0.04_25)]',
            ]"
            :title="d.error ?? `${d.durationMs}ms`"
          >
            <CheckCircle2 v-if="d.status === 'success'" :size="12" />
            <XCircle v-else :size="12" />
            {{ d.targetName }}
          </span>
        </div>
        <button v-if="entry.envelope" class="inline-flex items-center gap-1 px-2.5 py-1 border border-border rounded-full bg-transparent text-muted-foreground text-[0.73rem] cursor-pointer transition-colors mt-0.5 w-fit hover:text-foreground hover:border-[oklch(0.7_0.1_250)]" type="button" title="Replay this event" @click="emit('replay', entry.id)">
          <RotateCcw :size="13" />
          Replay
        </button>
      </div>
    </div>
  </div>
</template>
