<script setup lang="ts">
defineProps<{
  items: Array<{ label: string; value: string }>;
}>();
</script>

<template>
  <div class="status-grid">
    <article
      v-for="item in items"
      :key="item.label"
      class="status-card"
    >
      <span>{{ item.label }}</span>
      <strong>{{ item.value }}</strong>
    </article>
  </div>
</template>

<style scoped>
.status-grid {
  display: grid;
  gap: var(--space-4);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.status-card {
  display: grid;
  gap: 0.4rem;
  padding: var(--space-5);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.status-card span {
  color: var(--nook-muted);
  font-size: 0.8rem;
}

@media (max-width: 720px) {
  .status-grid {
    grid-template-columns: 1fr;
  }
}
</style>
