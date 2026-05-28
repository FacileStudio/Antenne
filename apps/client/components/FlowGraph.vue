<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import FlowNode from "./FlowNode.vue";
import type { ProviderSettings, DeliveryTarget, ProviderType } from "../types";
import { SYSTEM_PROVIDER_ID, getProviderLabel, getDeliveryTargetLabel } from "../types";

const props = defineProps<{
  providers: ProviderSettings[];
  targets: DeliveryTarget[];
}>();

const NODE_H = 60;
const NODE_GAP = 12;
const NODE_W = 220;
const PAD_X = 40;
const TOP_PAD = 48;
const BOTTOM_PAD = 40;

const graphEl = ref<HTMLElement | null>(null);
const containerWidth = ref(900);
const selectedId = ref<string | null>(null);

const PROVIDER_ICONS: Record<string, string> = {
  system: "solar:settings-bold-duotone",
  website: "solar:global-bold-duotone",
  imap: "solar:letter-bold-duotone",
  rss: "solar:rss-bold-duotone",
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

const DELIVERY_ICONS: Record<string, string> = {
  matrix: "solar:chat-round-dots-bold-duotone",
  discord: "solar:gamepad-bold-duotone",
  webhook: "solar:link-bold-duotone",
  smtp: "solar:letter-bold-duotone",
  ntfy: "solar:bell-bold-duotone",
  perception: "solar:eye-bold-duotone",
};

interface VirtualProvider {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
  tags: string[];
}

const allProviders = computed<VirtualProvider[]>(() => [
  { id: SYSTEM_PROVIDER_ID, type: "system", name: "System Events", enabled: true, tags: [] },
  ...props.providers,
]);

function providerLabel(type: string): string {
  if (type === "system") return "System";
  return getProviderLabel(type as ProviderType);
}

interface FlowEdge {
  id: string;
  providerId: string;
  targetId: string;
  providerIndex: number;
  targetIndex: number;
  type: "direct" | "tag";
}

const edges = computed(() => {
  const result: FlowEdge[] = [];
  for (let ti = 0; ti < props.targets.length; ti++) {
    const target = props.targets[ti];
    for (let pi = 0; pi < allProviders.value.length; pi++) {
      const provider = allProviders.value[pi];

      if (target.providerIds.includes(provider.id)) {
        result.push({ id: `${provider.id}--${target.id}`, providerId: provider.id, targetId: target.id, providerIndex: pi, targetIndex: ti, type: "direct" });
      } else if (target.providerTags.length > 0 && provider.tags?.length > 0 && provider.tags.some((t) => target.providerTags.includes(t))) {
        result.push({ id: `${provider.id}--${target.id}`, providerId: provider.id, targetId: target.id, providerIndex: pi, targetIndex: ti, type: "tag" });
      }
    }
  }
  return result;
});

function colHeight(count: number) {
  return count * NODE_H + Math.max(0, count - 1) * NODE_GAP;
}

const maxColH = computed(() => Math.max(colHeight(allProviders.value.length), colHeight(props.targets.length), 200));
const providerOffY = computed(() => (maxColH.value - colHeight(allProviders.value.length)) / 2);
const targetOffY = computed(() => (maxColH.value - colHeight(props.targets.length)) / 2);
const totalHeight = computed(() => TOP_PAD + maxColH.value + BOTTOM_PAD);

function nodeY(index: number, offset: number) {
  return TOP_PAD + offset + index * (NODE_H + NODE_GAP);
}

const edgePaths = computed(() => {
  const x1 = PAD_X + NODE_W;
  const x2 = containerWidth.value - PAD_X - NODE_W;
  const dx = Math.max((x2 - x1) * 0.4, 60);

  return edges.value.map((e) => {
    const sy = nodeY(e.providerIndex, providerOffY.value) + NODE_H / 2;
    const ey = nodeY(e.targetIndex, targetOffY.value) + NODE_H / 2;
    const highlighted = selectedId.value === e.providerId || selectedId.value === e.targetId;
    const dimmed = selectedId.value !== null && !highlighted;

    return {
      ...e,
      d: `M ${x1} ${sy} C ${x1 + dx} ${sy}, ${x2 - dx} ${ey}, ${x2} ${ey}`,
      highlighted,
      dimmed,
    };
  });
});

function connectedTo(nodeId: string): Set<string> {
  const ids = new Set<string>();
  for (const e of edges.value) {
    if (e.providerId === nodeId) ids.add(e.targetId);
    if (e.targetId === nodeId) ids.add(e.providerId);
  }
  return ids;
}

function isNodeDimmed(nodeId: string) {
  if (!selectedId.value) return false;
  if (selectedId.value === nodeId) return false;
  return !connectedTo(selectedId.value).has(nodeId);
}

function toggleSelect(id: string) {
  selectedId.value = selectedId.value === id ? null : id;
}

let observer: ResizeObserver | null = null;

onMounted(() => {
  if (graphEl.value) {
    containerWidth.value = graphEl.value.clientWidth;
    observer = new ResizeObserver(([entry]) => {
      containerWidth.value = entry.contentRect.width;
    });
    observer.observe(graphEl.value);
  }
});

onUnmounted(() => observer?.disconnect());
</script>

<template>
  <div ref="graphEl" class="flow-graph" :style="{ height: `${totalHeight}px` }" @click.self="selectedId = null">
    <span class="flow-col-label flow-col-label--left">Providers</span>
    <span class="flow-col-label flow-col-label--right">Delivery</span>

    <svg class="flow-svg" :width="containerWidth" :height="totalHeight" :viewBox="`0 0 ${containerWidth} ${totalHeight}`">
      <path
        v-for="ep in edgePaths"
        :key="ep.id"
        class="flow-edge"
        :d="ep.d"
        fill="none"
        :stroke="ep.highlighted ? 'var(--foreground)' : 'var(--nook-border)'"
        :stroke-width="ep.highlighted ? 1.5 : 1"
        :stroke-dasharray="ep.type === 'tag' ? '6 4' : 'none'"
        :opacity="ep.dimmed ? 0.08 : ep.highlighted ? 1 : 0.5"
        stroke-linecap="round"
      />
    </svg>

    <div
      v-for="(p, i) in allProviders"
      :key="p.id"
      class="flow-node-slot flow-node-slot--left"
      :style="{ top: `${nodeY(i, providerOffY)}px` }"
      @click.stop="toggleSelect(p.id)"
    >
      <FlowNode
        :icon="PROVIDER_ICONS[p.type] ?? 'solar:widget-2-bold-duotone'"
        :name="p.name"
        :label="providerLabel(p.type)"
        :enabled="p.enabled"
        :selected="selectedId === p.id"
        :dimmed="isNodeDimmed(p.id)"
      />
    </div>

    <div
      v-for="(t, j) in targets"
      :key="t.id"
      class="flow-node-slot flow-node-slot--right"
      :style="{ top: `${nodeY(j, targetOffY)}px` }"
      @click.stop="toggleSelect(t.id)"
    >
      <FlowNode
        :icon="DELIVERY_ICONS[t.type] ?? 'solar:widget-2-bold-duotone'"
        :name="t.name"
        :label="getDeliveryTargetLabel(t.type)"
        :enabled="t.enabled"
        :selected="selectedId === t.id"
        :dimmed="isNodeDimmed(t.id)"
      />
    </div>

    <p v-if="!targets.length" class="flow-empty flow-empty--right">No delivery targets configured</p>
  </div>
</template>

<style scoped>
.flow-graph {
  position: relative;
  width: 100%;
  min-width: 600px;
  min-height: 300px;
}

.flow-svg {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
}

.flow-edge {
  transition: stroke 200ms, stroke-width 200ms, opacity 200ms;
}

.flow-col-label {
  position: absolute;
  top: 14px;
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--nook-muted);
  pointer-events: none;
}

.flow-col-label--left {
  left: 40px;
}

.flow-col-label--right {
  right: 40px;
}

.flow-node-slot {
  position: absolute;
}

.flow-node-slot--left {
  left: 40px;
}

.flow-node-slot--right {
  right: 40px;
}

.flow-empty {
  position: absolute;
  top: 50%;
  color: var(--nook-muted);
  font-size: 0.84rem;
}

.flow-empty--right {
  right: 40px;
}
</style>
