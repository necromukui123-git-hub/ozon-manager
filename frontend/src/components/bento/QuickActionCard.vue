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
  border: 1px solid var(--surface-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-normal);
}

.quick-action-card:hover {
  border-color: var(--surface-border-hover);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.quick-action-card:hover .quick-action-card__icon {
  transform: scale(1.05);
}

.quick-action-card:hover .quick-action-card__arrow {
  transform: translateX(4px);
}

.quick-action-card:active {
  transform: scale(0.98);
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
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
  transition: transform var(--transition-normal);
}

.quick-action-card__icon--primary {
  background: linear-gradient(135deg, var(--primary), var(--primary-dark));
  color: white;
  box-shadow: 0 2px 8px var(--primary-glow);
}

.quick-action-card__icon--success {
  background: linear-gradient(135deg, var(--success), #3D8456);
  color: white;
  box-shadow: 0 2px 8px var(--success-glow);
}

.quick-action-card__icon--warning {
  background: linear-gradient(135deg, var(--warning), #A8702E);
  color: white;
  box-shadow: 0 2px 8px var(--warning-glow);
}

.quick-action-card__icon--danger {
  background: linear-gradient(135deg, var(--danger), #A84540);
  color: white;
  box-shadow: 0 2px 8px var(--danger-glow);
}

.quick-action-card__icon--accent {
  background: linear-gradient(135deg, var(--accent), #C4604A);
  color: white;
  box-shadow: 0 2px 8px var(--accent-glow);
}

.quick-action-card__icon--info {
  background: linear-gradient(135deg, var(--info), #4A6A9F);
  color: white;
  box-shadow: 0 2px 8px rgba(90, 123, 175, 0.2);
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
  font-weight: 500;
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
