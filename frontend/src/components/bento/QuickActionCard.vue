<template>
  <div
    class="quick-action-card"
    :class="[`quick-action-card--${variant}`, { 'quick-action-card--disabled': disabled }]"
    @click="handleClick"
  >
    <div class="quick-action-card__icon" :class="`quick-action-card__icon--${variant}`">
      <el-icon>
        <component :is="icon" />
      </el-icon>
    </div>
    <div class="quick-action-card__content">
      <span class="quick-action-card__title">{{ title }}</span>
      <span v-if="description" class="quick-action-card__desc">{{ description }}</span>
    </div>
    <el-icon v-if="showArrow" class="quick-action-card__arrow">
      <ArrowRight />
    </el-icon>
  </div>
</template>

<script setup>
import { ArrowRight } from '@element-plus/icons-vue'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  description: {
    type: String,
    default: ''
  },
  icon: {
    type: [Object, String],
    required: true
  },
  variant: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'success', 'warning', 'danger', 'accent', 'info'].includes(value)
  },
  disabled: {
    type: Boolean,
    default: false
  },
  showArrow: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['click'])

function handleClick(e) {
  if (!props.disabled) {
    emit('click', e)
  }
}
</script>

<style scoped>
.quick-action-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  background: var(--bg-secondary);
  border: var(--neo-border-width) solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  cursor: pointer;
  transition: all var(--transition-normal);
}

.quick-action-card:hover {
  border-color: var(--neo-border-color);
  transform: translate(-1px, -1px);
  box-shadow: 3px 3px 0 var(--neo-border-color);
}

.quick-action-card:hover .quick-action-card__icon {
  transform: scale(1.05);
}

.quick-action-card:hover .quick-action-card__arrow {
  transform: translateX(4px);
}

.quick-action-card:active {
  transform: translate(1px, 1px);
  box-shadow: 1px 1px 0 var(--neo-border-color);
}

.quick-action-card--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.quick-action-card--disabled:hover {
  transform: none;
  box-shadow: none;
}

.quick-action-card__icon {
  width: 44px;
  height: 44px;
  border-radius: var(--neo-radius);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
  transition: transform var(--transition-normal);
  border: 2px solid var(--neo-border-color);
  box-shadow: 2px 2px 0 var(--neo-border-color);
}

.quick-action-card__icon--primary {
  background: var(--primary);
  color: white;
}

.quick-action-card__icon--success {
  background: var(--success);
  color: white;
}

.quick-action-card__icon--warning {
  background: var(--warning);
  color: #111;
}

.quick-action-card__icon--danger {
  background: var(--danger);
  color: white;
}

.quick-action-card__icon--accent {
  background: var(--accent);
  color: white;
}

.quick-action-card__icon--info {
  background: var(--info);
  color: white;
}

.quick-action-card__content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.quick-action-card__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.quick-action-card__desc {
  font-size: 12px;
  color: var(--text-muted);
}

.quick-action-card__arrow {
  color: var(--text-muted);
  font-size: 16px;
  transition: transform var(--transition-normal);
}
</style>
