<script setup lang="ts">
import { Icon } from "@iconify/vue";

import type { EventLogEntry } from "../types";

defineProps<{
  events: EventLogEntry[];
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
  if (source === "website") return { icon: "solar:global-bold", label: "Website", tone: "website" };
  if (source === "webhook") return { icon: "solar:cpu-bolt-bold", label: "Webhook", tone: "webhook" };
  if (source === "imap") return { icon: "solar:letter-bold", label: "Inbox", tone: "imap" };
  if (source === "test") return { icon: "solar:test-tube-bold", label: "Test", tone: "test" };
  return { icon: "solar:server-bold", label: "System", tone: "system" };
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
  <div class="timeline">
    <p v-if="events.length === 0" class="empty">No events recorded yet.</p>
    <div v-for="entry in events" :key="entry.id" class="event-row">
      <div class="event-icon" :class="`tone-${sourceMeta(entry.source).tone}`">
        <Icon :icon="sourceMeta(entry.source).icon" width="18" />
      </div>
      <div class="event-body">
        <div class="event-header">
          <span class="event-name">{{ titleFor(entry) }}</span>
          <span class="event-time">
            <Icon icon="solar:clock-circle-bold" width="12" />
            {{ formatTime(entry.timestamp) }}
          </span>
        </div>
        <div class="event-meta">
          <span class="event-source-pill" :class="`tone-${sourceMeta(entry.source).tone}`">{{ sourceMeta(entry.source).label }}</span>
          <span v-if="entry.providerName && entry.providerName !== 'system' && entry.providerName !== 'Nook'" class="event-provider">{{ entry.providerName }}</span>
        </div>
        <p v-if="previewFor(entry)" class="event-preview">{{ previewFor(entry) }}</p>
        <div v-if="entry.deliveries.length > 0" class="event-deliveries">
          <span
            v-for="d in entry.deliveries"
            :key="d.targetId"
            class="delivery-pill"
            :class="d.status"
            :title="d.error ?? `${d.durationMs}ms`"
          >
            <Icon v-if="d.status === 'success'" icon="solar:check-circle-bold" width="12" />
            <Icon v-else icon="solar:close-circle-bold" width="12" />
            {{ d.targetName }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.timeline {
  display: grid;
  gap: 0;
}

.empty {
  color: var(--nook-muted);
  text-align: center;
  padding: var(--space-6) var(--space-5);
  border-radius: 1.2rem;
  border: 1px dashed var(--nook-border);
  background: rgb(255 255 255 / 0.02);
}

.event-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.95rem;
  padding: 1rem 1.05rem;
  border: 1px solid var(--nook-border);
  border-radius: 1.15rem;
  background: rgb(255 255 255 / 0.02);
}

.timeline > .event-row + .event-row {
  margin-top: 0.65rem;
}

.event-icon {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.9rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border: 1px solid var(--nook-border);
  background: rgb(255 255 255 / 0.03);
}

.event-body {
  min-width: 0;
  display: grid;
  gap: 0.45rem;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: var(--space-2);
}

.event-name {
  font-weight: 600;
  line-height: 1.3;
}

.event-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  align-items: center;
}

.event-source-pill,
.event-provider {
  display: inline-flex;
  align-items: center;
  min-height: 1.55rem;
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  background: rgb(255 255 255 / 0.03);
  color: var(--nook-muted);
  font-size: 0.73rem;
  white-space: nowrap;
}

.event-time {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.76rem;
  color: var(--nook-muted);
}

.event-preview {
  margin: 0;
  color: var(--nook-muted);
  font-size: 0.88rem;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tone-system {
  color: #9ec4ff;
}

.tone-website {
  color: #7af0b6;
}

.tone-webhook {
  color: #ffd67c;
}

.tone-imap {
  color: #f7a6ff;
}

.tone-test {
  color: #ffb86f;
}

.event-icon.tone-system {
  background: rgb(158 196 255 / 0.1);
}

.event-icon.tone-website {
  background: rgb(122 240 182 / 0.1);
}

.event-icon.tone-webhook {
  background: rgb(255 214 124 / 0.1);
}

.event-icon.tone-imap {
  background: rgb(247 166 255 / 0.1);
}

.event-icon.tone-test {
  background: rgb(255 184 111 / 0.1);
}

.event-source-pill.tone-system {
  border-color: rgb(158 196 255 / 0.25);
}

.event-source-pill.tone-website {
  border-color: rgb(122 240 182 / 0.25);
}

.event-source-pill.tone-webhook {
  border-color: rgb(255 214 124 / 0.25);
}

.event-source-pill.tone-imap {
  border-color: rgb(247 166 255 / 0.25);
}

.event-source-pill.tone-test {
  border-color: rgb(255 184 111 / 0.25);
}

.event-deliveries {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-top: 0.2rem;
}

.delivery-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.76rem;
  padding: 0.22rem 0.62rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  background: rgb(255 255 255 / 0.03);
}

.delivery-pill.success {
  color: #1cdb81;
  border-color: rgb(28 219 129 / 0.25);
  background: rgb(28 219 129 / 0.06);
}

.delivery-pill.error {
  color: rgb(255 131 131);
  border-color: rgb(255 131 131 / 0.25);
  background: rgb(255 131 131 / 0.06);
}

@media (max-width: 720px) {
  .event-row {
    grid-template-columns: 1fr;
    gap: 0.85rem;
  }

  .event-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .event-time {
    white-space: nowrap;
  }
}
</style>
