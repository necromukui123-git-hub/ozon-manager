<template>
  <div class="dashboard">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">数据概览</h2>
      <div class="header-actions">
        <el-button @click="fetchStats" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新数据
        </el-button>
      </div>
    </div>

    <!-- Bento Grid 布局 -->
    <div class="bento-grid">
      <!-- 统计卡片行 -->
      <StatCard
        :value="stats.totalProducts"
        label="商品总数"
        :icon="Goods"
        variant="primary"
      />
      <StatCard
        :value="stats.promotedProducts"
        label="已推广商品"
        :icon="TrendCharts"
        variant="accent"
      />
      <StatCard
        :value="stats.lossProducts"
        label="亏损商品"
        :icon="WarningFilled"
        variant="warning"
      />
      <StatCard
        :value="stats.promotableProducts"
        label="可推广商品"
        :icon="CircleCheckFilled"
        variant="success"
      />

      <!-- 商品状态分布饼图 -->
      <ChartCard
        title="商品状态分布"
        :icon="PieChart"
        size="2x2"
        :option="pieChartOption"
        :loading="loading"
        height="280px"
      />

      <!-- 最近操作 -->
      <BentoCard title="最近操作" :icon="Clock" size="2x2" no-padding>
        <template #actions>
          <el-button text size="small" @click="$router.push('/admin/logs')">
            查看全部
          </el-button>
        </template>
        <div class="recent-logs-wrapper">
          <el-table :data="recentLogs" size="small" :show-header="false">
            <el-table-column prop="operation_type" width="100">
              <template #default="{ row }">
                <el-tag size="small" :type="getOperationTagType(row.operation_type)">
                  {{ formatOperationType(row.operation_type) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="affected_count" width="60" align="center">
              <template #default="{ row }">
                <span class="count-badge">{{ row.affected_count }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="status" width="50" align="center">
              <template #default="{ row }">
                <el-icon v-if="row.status === 'success'" class="status-icon success">
                  <CircleCheckFilled />
                </el-icon>
                <el-icon v-else class="status-icon danger">
                  <CircleCloseFilled />
                </el-icon>
              </template>
            </el-table-column>
            <el-table-column prop="created_at">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.created_at) }}</span>
              </template>
            </el-table-column>
          </el-table>

          <div v-if="recentLogs.length === 0" class="empty-state">
            <el-icon><DocumentDelete /></el-icon>
            <p>暂无操作记录</p>
          </div>
        </div>
      </BentoCard>

      <!-- 快捷操作 -->
      <QuickActionCard
        title="批量报名"
        description="将商品批量添加到促销活动"
        :icon="Upload"
        variant="primary"
        @click="$router.push('/promotions/batch-enroll')"
      />
      <QuickActionCard
        title="亏损处理"
        description="处理亏损商品退出促销"
        :icon="Edit"
        variant="warning"
        @click="$router.push('/promotions/loss-process')"
      />
      <QuickActionCard
        title="改价推广"
        description="调整价格后重新推广"
        :icon="PriceTag"
        variant="success"
        @click="$router.push('/promotions/reprice')"
      />
      <QuickActionCard
        title="导出数据"
        description="导出可推广商品列表"
        :icon="Download"
        variant="accent"
        @click="handleExport"
      />

      <!-- 促销趋势折线图 -->
      <ChartCard
        title="近7天操作趋势"
        :icon="TrendCharts"
        size="4x1"
        :option="lineChartOption"
        :loading="loading"
        height="180px"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getProducts } from '@/api/product'
import { exportPromotable } from '@/api/promotion'
import { StatCard, ChartCard, BentoCard, QuickActionCard } from '@/components/bento'
import {
  Goods,
  TrendCharts,
  WarningFilled,
  CircleCheckFilled,
  CircleCloseFilled,
  Upload,
  Edit,
  PriceTag,
  Download,
  Refresh,
  DocumentDelete,
  Clock,
  PieChart
} from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)

const stats = ref({
  totalProducts: 0,
  promotedProducts: 0,
  lossProducts: 0,
  promotableProducts: 0
})

const recentLogs = ref([])

// 饼图配置
const pieChartOption = computed(() => ({
  tooltip: {
    trigger: 'item',
    formatter: '{b}: {c} ({d}%)'
  },
  legend: {
    orient: 'vertical',
    right: '5%',
    top: 'center',
    itemWidth: 12,
    itemHeight: 12,
    textStyle: {
      fontSize: 12
    }
  },
  series: [
    {
      name: '商品状态',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: {
        borderRadius: 6,
        borderColor: '#fff',
        borderWidth: 2
      },
      label: {
        show: false
      },
      emphasis: {
        label: {
          show: true,
          fontSize: 14,
          fontWeight: 'bold'
        }
      },
      data: [
        { value: stats.value.promotedProducts, name: '已推广', itemStyle: { color: '#D77757' } },
        { value: stats.value.lossProducts, name: '亏损', itemStyle: { color: '#C4883A' } },
        { value: stats.value.promotableProducts, name: '可推广', itemStyle: { color: '#4A9668' } }
      ]
    }
  ]
}))

// 折线图配置 (模拟数据)
const lineChartOption = computed(() => {
  const days = []
  const data = []
  for (let i = 6; i >= 0; i--) {
    const date = new Date()
    date.setDate(date.getDate() - i)
    days.push(`${date.getMonth() + 1}/${date.getDate()}`)
    data.push(Math.floor(Math.random() * 50) + 10)
  }

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: days,
      axisLine: {
        lineStyle: {
          color: 'rgba(0, 0, 0, 0.08)'
        }
      },
      axisLabel: {
        color: '#8a8780'
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        show: false
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(0, 0, 0, 0.05)'
        }
      },
      axisLabel: {
        color: '#8a8780'
      }
    },
    series: [
      {
        name: '操作次数',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 8,
        lineStyle: {
          width: 3,
          color: '#C4714E'
        },
        itemStyle: {
          color: '#C4714E',
          borderWidth: 2,
          borderColor: '#fff'
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(196, 113, 78, 0.25)' },
              { offset: 1, color: 'rgba(196, 113, 78, 0.02)' }
            ]
          }
        },
        data: data
      }
    ]
  }
})

watch(() => userStore.currentShopId, () => {
  fetchStats()
})

onMounted(async () => {
  await fetchStats()
})

async function fetchStats() {
  try {
    loading.value = true
    const shopId = userStore.currentShopId
    if (!shopId) return

    const res = await getProducts({ shop_id: shopId, page_size: 1 })
    stats.value.totalProducts = res.data.total

    const promotedRes = await getProducts({ shop_id: shopId, is_promoted: true, page_size: 1 })
    stats.value.promotedProducts = promotedRes.data.total

    const lossRes = await getProducts({ shop_id: shopId, is_loss: true, page_size: 1 })
    stats.value.lossProducts = lossRes.data.total

    stats.value.promotableProducts = stats.value.totalProducts - stats.value.promotedProducts - stats.value.lossProducts
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

async function handleExport() {
  try {
    const shopId = userStore.currentShopId
    if (!shopId) {
      ElMessage.warning('请先选择店铺')
      return
    }

    const res = await exportPromotable(shopId)
    const blob = new Blob([res], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `可推广商品_${new Date().toISOString().split('T')[0]}.xlsx`
    link.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (error) {
    console.error(error)
    ElMessage.error('导出失败')
  }
}

function formatOperationType(type) {
  const map = {
    'batch_enroll': '批量报名',
    'process_loss': '亏损处理',
    'remove_reprice_promote': '改价推广',
    'sync_products': '同步商品'
  }
  return map[type] || type
}

function getOperationTagType(type) {
  const map = {
    'batch_enroll': 'primary',
    'process_loss': 'warning',
    'remove_reprice_promote': 'success',
    'sync_products': 'info'
  }
  return map[type] || ''
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.dashboard {
  min-height: 100%;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.recent-logs-wrapper {
  height: 100%;
  padding: 12px;
}

.count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 20px;
  padding: 0 6px;
  background: var(--bg-tertiary);
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.status-icon {
  font-size: 18px;
}

.status-icon.success {
  color: var(--success);
}

.status-icon.danger {
  color: var(--danger);
}

.time-text {
  font-size: 12px;
  color: var(--text-muted);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-muted);
}

.empty-state .el-icon {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.empty-state p {
  font-size: 14px;
}

/* 响应式调整 */
@media (max-width: 1200px) {
  .bento-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 992px) {
  .bento-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .bento-grid {
    grid-template-columns: 1fr;
  }
}
</style>
