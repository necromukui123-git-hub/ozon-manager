<template>
  <BentoCard
    :title="title"
    :icon="icon"
    :size="size"
    :hoverable="hoverable"
    no-padding
  >
    <template v-if="$slots.actions" #actions>
      <slot name="actions" />
    </template>
    <div ref="chartRef" class="chart-container" :style="{ height: chartHeight }"></div>
  </BentoCard>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'
import { registerNeoTheme, rebuildNeoTheme } from '@/utils/echarts-theme'
import BentoCard from './BentoCard.vue'

registerNeoTheme(echarts)

const props = defineProps({
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
    default: '2x1'
  },
  hoverable: {
    type: Boolean,
    default: true
  },
  option: {
    type: Object,
    required: true
  },
  height: {
    type: String,
    default: ''
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const chartRef = ref(null)
let chartInstance = null

const chartHeight = ref(props.height || getDefaultHeight(props.size))

function getDefaultHeight(size) {
  const heights = {
    '1x1': '120px',
    '2x1': '140px',
    '2x2': '300px',
    '3x1': '140px',
    '4x1': '140px',
    '1x2': '300px'
  }
  return heights[size] || '140px'
}

function getLoadingColor() {
  return getComputedStyle(document.documentElement).getPropertyValue('--primary').trim() || '#1d4ed8'
}

function initChart() {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value, 'neo')
  chartInstance.setOption(props.option)

  if (props.loading) {
    chartInstance.showLoading({
      text: '',
      color: getLoadingColor(),
      maskColor: 'rgba(255, 255, 255, 0.8)'
    })
  }
}

function resizeChart() {
  if (chartInstance) {
    chartInstance.resize()
  }
}

function handleThemeChanged() {
  rebuildNeoTheme(echarts)
  if (!chartRef.value) return

  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }

  chartInstance = echarts.init(chartRef.value, 'neo')
  chartInstance.setOption(props.option, true)
}

watch(() => props.option, (newOption) => {
  if (chartInstance) {
    chartInstance.setOption(newOption, true)
  }
}, { deep: true })

watch(() => props.loading, (loading) => {
  if (chartInstance) {
    if (loading) {
      chartInstance.showLoading({
        text: '',
        color: getLoadingColor(),
        maskColor: 'rgba(255, 255, 255, 0.8)'
      })
    } else {
      chartInstance.hideLoading()
    }
  }
})

onMounted(() => {
  nextTick(() => {
    initChart()
    window.addEventListener('resize', resizeChart)
    window.addEventListener('ozon-theme-change', handleThemeChanged)
  })
})

onUnmounted(() => {
  window.removeEventListener('resize', resizeChart)
  window.removeEventListener('ozon-theme-change', handleThemeChanged)
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})

defineExpose({
  resize: resizeChart,
  getInstance: () => chartInstance
})
</script>

<style scoped>
.chart-container {
  width: 100%;
  padding: 12px;
}
</style>

