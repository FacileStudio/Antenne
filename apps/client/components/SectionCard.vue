<script setup lang="ts">
defineProps<{
  id?: string;
  eyebrow: string;
  title: string;
}>();
</script>

<template>
  <section :id="id" class="section-card">
    <header class="section-header">
      <div>
        <p class="eyebrow">{{ eyebrow }}</p>
        <h2>{{ title }}</h2>
      </div>
      <slot name="actions" />
    </header>

    <div class="section-body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.section-card {
  display: grid;
  gap: var(--space-5);
  scroll-margin-top: 1.5rem;
}

.section-header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-start;
}

.section-body {
  display: grid;
  gap: var(--space-5);
}

.eyebrow {
  margin: 0 0 0.35rem;
  font-size: 0.75rem;
  color: var(--nook-muted);
}

h2 {
  margin: 0;
}

@media (max-width: 960px) {
  .section-header {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
