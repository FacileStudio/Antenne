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

function sourceIcon(source: string) {
  if (source === "website") return "🌐";
  if (source === "webhook") return "📡";
  if (source === "imap") return "📨";
  if (source === "test") return "🧪";
  return "⚙️";
}
</script>

<template>
  <div class="timeline">
    <p v-if="events.length === 0" class="empty">No events recorded yet.</p>
    <div v-for="entry in events" :key="entry.id" class="event-row">
      <div class="event-icon">{{ sourceIcon(entry.source) }}</div>
      <div class="event-body">
        <div class="event-header">
          <span class="event-name">{{ entry.event }}</span>
          <span class="event-time">
            <Icon icon="solar:clock-circle-bold" width="12" />
            {{ formatTime(entry.timestamp) }}
          </span>
        </div>
        <div class="event-provider">{{ entry.providerName }}</div>
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
  padding: var(--space-5);
}

.event-row {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--nook-border);
}

.event-row:last-child {
  border-bottom: none;
}

.event-icon {
  flex: none;
  font-size: 1.2rem;
  line-height: 1.6;
}

.event-body {
  flex: 1;
  min-width: 0;
  display: grid;
  gap: 0.3rem;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: var(--space-2);
}

.event-name {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-time {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.82rem;
  color: var(--nook-muted);
}

.event-provider {
  font-size: 0.85rem;
  color: var(--nook-muted);
}

.event-deliveries {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.15rem;
}

.delivery-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.78rem;
  padding: 0.15rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
}

.delivery-pill.success {
  color: #1CDB81;
  border-color: rgb(28 219 129 / 0.25);
}

.delivery-pill.error {
  color: rgb(255 131 131);
  border-color: rgb(255 131 131 / 0.25);
}
</style>
