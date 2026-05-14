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
  <div class="event-list">
    <p v-if="events.length === 0" class="empty-event">No events recorded yet.</p>
    <article v-for="entry in events" :key="entry.id" class="event-card">
      <div class="event-content">
        <div class="event-heading">
          <span class="event-title">{{ titleFor(entry) }}</span>
          <span class="event-time">
            <Icon icon="solar:clock-circle-linear" :width="12" :height="12" aria-hidden="true" />
            {{ formatTime(entry.timestamp) }}
          </span>
        </div>
        <div class="pill-row">
          <span
            class="event-pill source-pill"
            :data-tone="sourceMeta(entry.source).tone"
          >{{ sourceMeta(entry.source).label }}</span>
          <span v-if="entry.providerName && entry.providerName !== 'system' && entry.providerName !== 'Nook'" class="event-pill">{{ entry.providerName }}</span>
        </div>
        <p v-if="previewFor(entry)" class="event-preview">{{ previewFor(entry) }}</p>
        <div v-if="entry.deliveries.length > 0" class="delivery-row">
          <span
            v-for="d in entry.deliveries"
            :key="d.targetId"
            class="delivery-pill"
            :data-status="d.status"
            :title="d.error ?? `${d.durationMs}ms`"
          >
            <Icon v-if="d.status === 'success'" icon="solar:check-circle-linear" :width="12" :height="12" aria-hidden="true" />
            <Icon v-else icon="solar:close-circle-linear" :width="12" :height="12" aria-hidden="true" />
            {{ d.targetName }}
          </span>
        </div>
        <button v-if="entry.envelope" class="ghost action-button replay-button" type="button" title="Replay this event" @click="emit('replay', entry.id)">
          <Icon icon="solar:refresh-linear" :width="13" :height="13" aria-hidden="true" />
          Replay
        </button>
      </div>
    </article>
  </div>
</template>

<style scoped>
.event-list,
.event-content {
  display: grid;
  gap: 0.75rem;
  min-width: 0;
}

.empty-event,
.event-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  background: var(--card);
}

.empty-event {
  margin: 0;
  padding: var(--space-5);
  border-style: dashed;
  color: var(--muted-foreground);
  text-align: center;
}

.event-card {
  padding: var(--space-4);
}

.event-heading {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: var(--space-2);
}

.event-title {
  min-width: 0;
  font-weight: 600;
  line-height: 1.25;
}

.event-time {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  color: var(--muted-foreground);
  font-size: 0.75rem;
}

.pill-row,
.delivery-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4rem;
}

.event-pill,
.delivery-pill {
  display: inline-flex;
  align-items: center;
  min-height: 1.25rem;
  padding: 0.15rem 0.55rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--border);
  background: var(--muted);
  color: var(--muted-foreground);
  font-size: 0.73rem;
  line-height: 1;
  white-space: nowrap;
}

.source-pill[data-tone="system"] {
  border-color: oklch(0.84 0.07 250);
  background: oklch(0.95 0.04 250);
  color: oklch(0.38 0.14 250);
}

.source-pill[data-tone="website"] {
  border-color: var(--success-border);
  background: var(--success-muted);
  color: var(--success-foreground);
}

.source-pill[data-tone="webhook"] {
  border-color: oklch(0.86 0.07 65);
  background: oklch(0.96 0.04 65);
  color: oklch(0.42 0.11 65);
}

.source-pill[data-tone="imap"] {
  border-color: oklch(0.84 0.07 310);
  background: oklch(0.95 0.04 310);
  color: oklch(0.4 0.14 310);
}

.source-pill[data-tone="test"] {
  border-color: oklch(0.86 0.07 45);
  background: oklch(0.96 0.04 45);
  color: oklch(0.44 0.12 45);
}

.event-preview {
  margin: 0;
  color: var(--muted-foreground);
  font-size: 0.875rem;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.delivery-pill {
  gap: 0.35rem;
}

.delivery-pill[data-status="success"] {
  border-color: var(--success-border);
  background: var(--success-muted);
  color: var(--success-foreground);
}

.delivery-pill[data-status="error"] {
  border-color: var(--danger-border);
  background: var(--danger-muted);
  color: var(--destructive);
}

.replay-button {
  width: fit-content;
  min-height: 1.75rem;
  padding-inline: 0.6rem;
  font-size: 0.73rem;
}

@media (max-width: 640px) {
  .event-heading {
    display: grid;
  }
}
</style>
