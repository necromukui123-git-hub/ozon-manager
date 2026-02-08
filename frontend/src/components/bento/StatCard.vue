<template>
  <div
    class="stat-card"
    :class="[`stat-card--${variant}`, { 'stat-card--clickable': clickable }]"
    @click="handleClick"
  >
    <div class="stat-card__icon" :class="`stat-card__icon--${variant}`">
      <el-icon>
        <component :is="icon" />
      </el-icon>
    </div>
    <div class="stat-card__content">
      <div class="stat-card__value">
        <span class="stat-card__number">{{ formattedValue }}</span>
        <span v-if="unit" class="stat-card__unit">{{ unit }}</span>
      </div>
      <div class="stat-card__label">{{ label }}</div>
      <div v-if="trend !== null" class="stat-card__trend" :class="trendClass">
        <el-icon>
          <component :is="trendIcon" />
        </el-icon>
        <span>{{ Math.abs(trend) }}%</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowUp, ArrowDown } from '@element-plus/icons-vue'

const props = defineProps({
  value: {
    type: [Number, String],
    default: 0
  },
  label: {
    type: String,
    required: true
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
  unit: {
    type: String,
    default: ''
  },
  trend: {
    type: Number,
    default: null
  },
  clickable: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['click'])

const formattedValue = computed(() => {
  if (typeof props.value === 'number') {
    return props.value.toLocaleString()
  }
  return props.value
})

const trendClass = computed(() => {
  if (props.trend === null) return ''
  return props.trend >= 0 ? 'stat-card__trend--up' : 'stat-card__trend--down'
})

const trendIcon = computed(() => {
  return props.trend >= 0 ? ArrowUp : ArrowDown
})

function handleClick(e) {
  if (props.clickable) {
    emit('click', e)
  }
}
</script>

<style scoped>
.stat-card {
  background: var(--bg-secondary);
  border: var(--neo-border-width) solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  padding: 20px;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  transition: all var(--transition-normal);
  position: relative;
  overflow: hidden;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  opacity: 0;
  transition: opacity var(--transition-normal);
}

.stat-card:hover {
  border-color: var(--neo-border-color);
  transform: translate(-1px, -1px);
  box-shadow: 3px 3px 0 var(--neo-border-color);
}

.stat-card:hover::before {
  opacity: 1;
}

.stat-card--clickable {
  cursor: pointer;
}

.stat-card--clickable:active {
  transform: translate(1px, 1px);
  box-shadow: 1px 1px 0 var(--neo-border-color);
}

/* 变体颜色 */
.stat-card--primary::before {
  background: var(--primary);
}

.stat-card--success::before {
  background: var(--success);
}

.stat-card--warning::before {
  background: var(--warning);
}

.stat-card--danger::before {
  background: var(--danger);
}

.stat-card--accent::before {
  background: var(--accent);
}

.stat-card--info::before {
  background: var(--info);
}

.stat-card__icon {
  width: 48px;
  height: 48px;
  border-radius: var(--neo-radius);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
  border: 2px solid var(--neo-border-color);
  box-shadow: 2px 2px 0 var(--neo-border-color);
}

.stat-card__icon--primary {
  background: var(--primary);
  color: #fff;
}

.stat-card__icon--success {
  background: var(--success);
  color: #fff;
}

.stat-card__icon--warning {
  background: var(--warning);
  color: #111;
}

.stat-card__icon--danger {
  background: var(--danger);
  color: #fff;
}

.stat-card__icon--accent {
  background: var(--accent);
  color: #fff;
}

.stat-card__icon--info {
  background: var(--info);
  color: #fff;
}

.stat-card__content {
  flex: 1;
  min-width: 0;
}

.stat-card__value {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-bottom: 4px;
}

.stat-card__number {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
  letter-spacing: -0.02em;
  font-family: 'Fira Code', monospace;
}

.stat-card__unit {
  font-size: 14px;
  color: var(--text-muted);
}

.stat-card__label {
  font-size: 13px;
  color: var(--text-muted);
}

.stat-card__trend {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-top: 8px;
  font-size: 12px;
  font-weight: 500;
  padding: 2px 6px;
  border-radius: var(--neo-radius);
  border: 2px solid var(--neo-border-color);
}

.stat-card__trend--up {
  color: var(--success);
  background: #dcfce7;
}

.stat-card__trend--down {
  color: var(--danger);
  background: #fee2e2;
}
</style>
