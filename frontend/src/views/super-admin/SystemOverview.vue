<template>
  <div class="system-overview">
    <div class="page-header">
      <h2 class="gradient">系统概览</h2>
      <el-button @click="fetchOverview" :loading="loading">
        <el-icon><Refresh /></el-icon>
        刷新数据
      </el-button>
    </div>

    <!-- Bento Grid 布局 -->
    <div class="bento-grid">
      <!-- 统计卡片 -->
      <StatCard
        :value="overview.shop_admin_count || 0"
        label="店铺管理员"
        :icon="User"
        variant="primary"
      />
      <StatCard
        :value="overview.staff_count || 0"
        label="员工总数"
        :icon="UserFilled"
        variant="accent"
      />
      <StatCard
        :value="overview.shop_count || 0"
        label="店铺总数"
        :icon="Shop"
        variant="info"
      />
      <StatCard
        :value="overview.product_count || 0"
        label="商品总数"
        :icon="Goods"
        variant="success"
      />

      <!-- 用户分布图表 -->
      <ChartCard
        title="用户分布"
        :icon="PieChart"
        size="2x2"
        :option="pieChartOption"
        :loading="loading"
        height="280px"
      />

      <!-- 店铺管理员列表 -->
      <BentoCard title="店铺管理员列表" :icon="UserFilled" size="2x2" no-padding>
        <template #actions>
          <el-tag type="info" effect="plain" size="small">
            {{ overview.shop_admins?.length || 0 }} 人
          </el-tag>
        </template>
        <div class="admin-list-wrapper">
          <el-table :data="overview.shop_admins || []" size="small" :show-header="false">
            <el-table-column width="50">
              <template #default="{ row }">
                <div class="admin-avatar">
                  {{ row.display_name?.charAt(0) || 'A' }}
                </div>
              </template>
            </el-table-column>
            <el-table-column>
              <template #default="{ row }">
                <div class="admin-info">
                  <span class="admin-name">{{ row.display_name }}</span>
                  <span class="admin-username">@{{ row.username }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                  {{ row.status === 'active' ? '正常' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column width="100" align="right">
              <template #default="{ row }">
                <div class="admin-stats">
                  <span>{{ row.shop_count || 0 }} 店铺</span>
                  <span>{{ row.staff_count || 0 }} 员工</span>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </BentoCard>

      <!-- 资源统计柱状图 -->
      <ChartCard
        title="资源统计"
        :icon="DataAnalysis"
        size="4x1"
        :option="barChartOption"
        :loading="loading"
        height="180px"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { User, UserFilled, Shop, Goods, Refresh, PieChart, DataAnalysis } from '@element-plus/icons-vue'
import { getSystemOverview } from '@/api/admin'
import { StatCard, ChartCard, BentoCard } from '@/components/bento'
import { getThemeChartTokens } from '@/utils/echarts-theme'
import { getTheme } from '@/utils/theme'

const loading = ref(false)
const overview = reactive({
  shop_admin_count: 0,
  staff_count: 0,
  shop_count: 0,
  product_count: 0,
  shop_admins: []
})
const currentTheme = getTheme()
const chartToken = computed(() => {
  currentTheme.value
  return getThemeChartTokens()
})

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
    itemHeight: 12
  },
  series: [
    {
      name: '用户分布',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: {
        borderRadius: 0,
        borderColor: chartToken.value.border,
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
        { value: overview.shop_admin_count, name: '店铺管理员', itemStyle: { color: chartToken.value.color[0] } },
        { value: overview.staff_count, name: '员工', itemStyle: { color: chartToken.value.color[1] } }
      ]
    }
  ]
}))

// 柱状图配置
const barChartOption = computed(() => ({
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
    data: ['店铺管理员', '员工', '店铺', '商品'],
    axisLine: {
      lineStyle: {
        color: chartToken.value.border,
        width: 2
      }
    },
    axisLabel: {
      color: chartToken.value.muted
    }
  },
  yAxis: {
    type: 'value',
    axisLine: {
      show: false
    },
    splitLine: {
      lineStyle: {
        color: chartToken.value.border,
        opacity: 0.2
      }
    },
    axisLabel: {
      color: chartToken.value.muted
    }
  },
  series: [
    {
      name: '数量',
      type: 'bar',
      barWidth: '50%',
      itemStyle: {
        borderRadius: [0, 0, 0, 0],
        borderColor: chartToken.value.border,
        borderWidth: 2,
        color: function(params) {
          const colors = chartToken.value.color
          return colors[params.dataIndex]
        }
      },
      data: [
        overview.shop_admin_count,
        overview.staff_count,
        overview.shop_count,
        overview.product_count
      ]
    }
  ]
}))

onMounted(async () => {
  await fetchOverview()
})

async function fetchOverview() {
  loading.value = true
  try {
    const res = await getSystemOverview()
    Object.assign(overview, res.data)
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.system-overview {
  min-height: 100%;
}

.admin-list-wrapper {
  height: 100%;
  padding: 8px;
}

.admin-avatar {
  width: 36px;
  height: 36px;
  border-radius: var(--neo-radius);
  background: var(--primary);
  border: 2px solid var(--neo-border-color);
  box-shadow: 2px 2px 0 var(--neo-border-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
}

.admin-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.admin-name {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
}

.admin-username {
  font-size: 12px;
  color: var(--text-muted);
}

.admin-stats {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 11px;
  color: var(--text-muted);
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
