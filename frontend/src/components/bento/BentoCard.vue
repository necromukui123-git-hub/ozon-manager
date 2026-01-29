<template>
  <div
    class="bento-card"
    :class="[
      `bento-card--${size}`,
      { 'bento-card--hoverable': hoverable },
      { 'bento-card--clickable': clickable }
    ]"
    @click="handleClick"
  >
    <div v-if="title || $slots.header" class="bento-card__header">
      <slot name="header">
        <div class="bento-card__title">
          <el-icon v-if="icon" class="bento-card__icon">
            <component :is="icon" />
          </el-icon>
          <span>{{ title }}</span>
        </div>
        <div v-if="$slots.actions" class="bento-card__actions">
          <slot name="actions" />
        </div>
      </slot>
    </div>
    <div class="bento-card__body" :class="{ 'bento-card__body--no-padding': noPadding }">
      <slot />
    </div>
    <div v-if="$slots.footer" class="bento-card__footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: ''
  },
  icon: {
    type: [Object, String],
    default: null
  },
  size: {
    type: String,
    default: '1x1',
    validator: (value) => ['1x1', '2x1', '2x2', '3x1', '4x1', '1x2'].includes(value)
  },
  hoverable: {
    type: Boolean,
    default: true
  },
  clickable: {
    type: Boolean,
    default: false
  },
  noPadding: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['click'])

function handleClick(e) {
  emit('click', e)
}
</script>

<style scoped>
.bento-card {
  background: var(--bg-secondary);
  border: 1px solid var(--surface-border);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  transition: all var(--transition-normal);
  overflow: hidden;
}

.bento-card--hoverable:hover {
  border-color: var(--surface-border-hover);
  box-shadow: var(--shadow-md);
}

.bento-card--clickable {
  cursor: pointer;
}

.bento-card--clickable:hover {
  transform: translateY(-2px);
}

.bento-card--clickable:active {
  transform: scale(0.99);
}

/* 尺寸变体 */
.bento-card--1x1 {
  min-height: 160px;
}

.bento-card--2x1 {
  grid-column: span 2;
  min-height: 160px;
}

.bento-card--2x2 {
  grid-column: span 2;
  grid-row: span 2;
  min-height: 340px;
}

.bento-card--3x1 {
  grid-column: span 3;
  min-height: 160px;
}

.bento-card--4x1 {
  grid-column: span 4;
  min-height: 160px;
}

.bento-card--1x2 {
  grid-row: span 2;
  min-height: 340px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .bento-card--3x1,
  .bento-card--4x1 {
    grid-column: span 3;
  }
}

@media (max-width: 992px) {
  .bento-card--2x1,
  .bento-card--2x2,
  .bento-card--3x1,
  .bento-card--4x1 {
    grid-column: span 2;
  }
}

@media (max-width: 768px) {
  .bento-card--1x1,
  .bento-card--2x1,
  .bento-card--2x2,
  .bento-card--3x1,
  .bento-card--4x1,
  .bento-card--1x2 {
    grid-column: span 1;
    grid-row: span 1;
    min-height: auto;
  }
}

.bento-card__header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--surface-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.bento-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.bento-card__icon {
  color: var(--primary);
  font-size: 18px;
}

.bento-card__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bento-card__body {
  flex: 1;
  padding: 20px;
  overflow: auto;
}

.bento-card__body--no-padding {
  padding: 0;
}

.bento-card__footer {
  padding: 12px 20px;
  border-top: 1px solid var(--surface-border);
  flex-shrink: 0;
}
</style>
