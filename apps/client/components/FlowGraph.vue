<script setup lang="ts">
import { computed, watch, markRaw } from "vue";
import { VueFlow, useVueFlow, type Node, type Edge, type Connection, MarkerType } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import FlowProviderNode from "./FlowProviderNode.vue";
import FlowTargetNode from "./FlowTargetNode.vue";
import type { ProviderSettings, DeliveryTarget, ProviderType } from "../types";
import { SYSTEM_PROVIDER_ID, getProviderLabel, getDeliveryTargetLabel } from "../types";
import { useDashboardState } from "../composables/useDashboardState";
import { pushUndo } from "../composables/useUndo";

const props = defineProps<{
  providers: ProviderSettings[];
  targets: DeliveryTarget[];
}>();

const { saveSettings, notifySuccess } = useDashboardState();

const NODE_GAP = 72;
const COL_LEFT_X = 50;
const COL_RIGHT_X = 450;

const nodeTypes = {
  provider: markRaw(FlowProviderNode),
  target: markRaw(FlowTargetNode),
};

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

function providerLabel(type: string): string {
  if (type === "system") return "System";
  return getProviderLabel(type as ProviderType);
}

function toggleProvider(nodeId: string) {
  if (nodeId === SYSTEM_PROVIDER_ID) return;
  const provider = props.providers.find((p) => p.id === nodeId);
  if (!provider) return;

  const was = provider.enabled;
  provider.enabled = !was;
  saveSettings();
  pushUndo({
    label: `${provider.enabled ? "Enabled" : "Disabled"} ${provider.name}`,
    execute: () => { provider.enabled = was; saveSettings(); },
  });
}

function toggleTarget(nodeId: string) {
  const target = props.targets.find((t) => t.id === nodeId);
  if (!target) return;

  const was = target.enabled;
  target.enabled = !was;
  saveSettings();
  pushUndo({
    label: `${target.enabled ? "Enabled" : "Disabled"} ${target.name}`,
    execute: () => { target.enabled = was; saveSettings(); },
  });
}

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

function buildNodes(): Node[] {
  const providerNodes: Node[] = allProviders.value.map((p, i) => ({
    id: `p-${p.id}`,
    type: "provider",
    position: { x: COL_LEFT_X, y: i * NODE_GAP },
    data: {
      icon: PROVIDER_ICONS[p.type] ?? "solar:widget-2-bold-duotone",
      name: p.name,
      label: providerLabel(p.type),
      enabled: p.enabled,
      nodeId: p.id,
      onToggle: toggleProvider,
    },
    draggable: true,
  }));

  const targetNodes: Node[] = props.targets.map((t, i) => ({
    id: `t-${t.id}`,
    type: "target",
    position: { x: COL_RIGHT_X, y: i * NODE_GAP },
    data: {
      icon: DELIVERY_ICONS[t.type] ?? "solar:widget-2-bold-duotone",
      name: t.name,
      label: getDeliveryTargetLabel(t.type),
      enabled: t.enabled,
      nodeId: t.id,
      onToggle: toggleTarget,
    },
    draggable: true,
  }));

  return [...providerNodes, ...targetNodes];
}

function buildEdges(): Edge[] {
  const result: Edge[] = [];

  for (const target of props.targets) {
    for (const provider of allProviders.value) {
      const isDirect = target.providerIds.includes(provider.id);
      const isTag = !isDirect && target.providerTags.length > 0 && provider.tags?.length > 0 && provider.tags.some((t) => target.providerTags.includes(t));

      if (isDirect || isTag) {
        result.push({
          id: `e-${provider.id}--${target.id}`,
          source: `p-${provider.id}`,
          target: `t-${target.id}`,
          animated: false,
          style: {
            stroke: "var(--nook-border)",
            strokeWidth: 1.5,
          },
          ...(isTag ? { style: { stroke: "var(--nook-border)", strokeWidth: 1.5, strokeDasharray: "6 4" } } : {}),
          data: { type: isDirect ? "direct" : "tag", providerId: provider.id, targetId: target.id },
        });
      }
    }
  }

  return result;
}

const nodes = computed(() => buildNodes());
const edges = computed(() => buildEdges());

const { onConnect, onEdgeClick, fitView } = useVueFlow({ id: "nook-flow" });

onConnect((connection: Connection) => {
  const providerId = connection.source.replace(/^p-/, "");
  const targetId = connection.target.replace(/^t-/, "");

  if (!connection.source.startsWith("p-") || !connection.target.startsWith("t-")) return;

  const target = props.targets.find((t) => t.id === targetId);
  if (!target) return;

  if (target.providerIds.includes(providerId)) return;

  target.providerIds.push(providerId);
  saveSettings();

  const providerName = allProviders.value.find((p) => p.id === providerId)?.name ?? "provider";
  notifySuccess(`Connected ${providerName} → ${target.name}`);
  pushUndo({
    label: `Connected ${providerName} → ${target.name}`,
    execute: () => {
      target.providerIds = target.providerIds.filter((id) => id !== providerId);
      saveSettings();
    },
  });
});

onEdgeClick(({ edge }) => {
  if (edge.data?.type !== "direct") return;

  const targetId = edge.data.targetId as string;
  const providerId = edge.data.providerId as string;
  const target = props.targets.find((t) => t.id === targetId);
  if (!target) return;

  target.providerIds = target.providerIds.filter((id) => id !== providerId);
  saveSettings();

  const providerName = allProviders.value.find((p) => p.id === providerId)?.name ?? "provider";
  notifySuccess(`Disconnected ${providerName} from ${target.name}`);
  pushUndo({
    label: `Disconnected ${providerName} from ${target.name}`,
    execute: () => {
      if (!target.providerIds.includes(providerId)) {
        target.providerIds.push(providerId);
        saveSettings();
      }
    },
  });
});

watch([() => props.providers.length, () => props.targets.length], () => {
  setTimeout(() => fitView({ padding: 0.2 }), 100);
});
</script>

<template>
  <div class="flow-canvas">
    <VueFlow
      id="nook-flow"
      :nodes="nodes"
      :edges="edges"
      :node-types="nodeTypes"
      :default-viewport="{ x: 0, y: 0, zoom: 1 }"
      :min-zoom="0.3"
      :max-zoom="2"
      :snap-to-grid="true"
      :snap-grid="[12, 12]"
      :connection-line-style="{ stroke: 'var(--foreground)', strokeWidth: 1.5 }"
      :default-edge-options="{ type: 'smoothstep' }"
      fit-view-on-init
      :fit-view-on-init-options="{ padding: 0.2 }"
      :nodes-draggable="true"
      :nodes-connectable="true"
      :edges-updatable="false"
      :delete-key-code="null"
      class="nook-vue-flow"
    >
      <Background :gap="20" :size="0.5" pattern-color="var(--nook-border)" />
      <Controls :show-fit-view="true" :show-interactive="false" position="bottom-left" />

      <template #edge-label="{ edge }">
        <div v-if="edge.data?.type === 'tag'" class="flow-edge-badge">tag</div>
      </template>
    </VueFlow>

    <div class="flow-legend">
      <span class="flow-legend-item">
        <span class="flow-legend-line flow-legend-line--solid" />
        Direct
      </span>
      <span class="flow-legend-item">
        <span class="flow-legend-line flow-legend-line--dashed" />
        Tag match
      </span>
      <span class="flow-legend-item flow-legend-hint">
        Drag between handles to connect · Click edge to disconnect
      </span>
    </div>
  </div>
</template>

<style>
@import "@vue-flow/core/dist/style.css";
@import "@vue-flow/core/dist/theme-default.css";
@import "@vue-flow/controls/dist/style.css";

.nook-vue-flow {
  width: 100%;
  height: 100%;
}

.nook-vue-flow .vue-flow__edge-path {
  transition: stroke 150ms, stroke-width 150ms;
}

.nook-vue-flow .vue-flow__edge:hover .vue-flow__edge-path {
  stroke: var(--foreground) !important;
  stroke-width: 2.5 !important;
  cursor: pointer;
}

.nook-vue-flow .vue-flow__connection-line path {
  stroke: var(--foreground);
  stroke-width: 1.5;
}

.nook-vue-flow .vue-flow__background {
  opacity: 0.4;
}

.nook-vue-flow .vue-flow__controls {
  background: var(--card);
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  box-shadow: none;
}

.nook-vue-flow .vue-flow__controls-button {
  background: transparent;
  border: none;
  color: var(--nook-muted);
  fill: var(--nook-muted);
  width: 28px;
  height: 28px;
}

.nook-vue-flow .vue-flow__controls-button:hover {
  background: var(--nook-surface);
  color: var(--foreground);
  fill: var(--foreground);
}

.nook-vue-flow .vue-flow__controls-button svg {
  max-width: 14px;
  max-height: 14px;
}

.nook-vue-flow .vue-flow__minimap {
  background: var(--card);
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
}
</style>

<style scoped>
.flow-canvas {
  position: relative;
  width: 100%;
  height: calc(100vh - 120px);
  min-height: 400px;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
  background: var(--background);
}

.flow-edge-badge {
  font-size: 0.6rem;
  padding: 1px 5px;
  background: var(--nook-surface);
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-sm);
  color: var(--nook-muted);
  pointer-events: none;
}

.flow-legend {
  position: absolute;
  bottom: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem 0.75rem;
  background: var(--card);
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  font-size: 0.72rem;
  color: var(--nook-muted);
  z-index: 10;
}

.flow-legend-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.flow-legend-hint {
  opacity: 0.6;
}

.flow-legend-line {
  display: inline-block;
  width: 20px;
  height: 0;
  border-top: 1.5px solid var(--nook-border);
}

.flow-legend-line--dashed {
  border-top-style: dashed;
}
</style>
