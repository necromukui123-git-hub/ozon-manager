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

    <!-- 统计卡片 -->
    <div class="stat-cards">
      <div class="stat-card">
        <div class="stat-icon primary">
          <el-icon><Goods /></el-icon>
        </div>
        <div class="stat-value">{{ stats.totalProducts }}</div>
        <div class="stat-label">商品总数</div>
      </div>

      <div class="stat-card">
        <div class="stat-icon accent">
          <el-icon><TrendCharts /></el-icon>
        </div>
        <div class="stat-value">{{ stats.promotedProducts }}</div>
        <div class="stat-label">已推广商品</div>
      </div>

      <div class="stat-card">
        <div class="stat-icon warning">
          <el-icon><WarningFilled /></el-icon>
        </div>
        <div class="stat-value">{{ stats.lossProducts }}</div>
        <div class="stat-label">亏损商品</div>
      </div>

      <div class="stat-card">
        <div class="stat-icon success">
          <el-icon><CircleCheckFilled /></el-icon>
        </div>
        <div class="stat-value">{{ stats.promotableProducts }}</div>
        <div class="stat-label">可推广商品</div>
      </div>
    </div>

    <!-- 内容区 -->
    <div class="dashboard-grid">
      <!-- 快捷操作 -->
      <div class="glass-card">
        <div class="card-header">
          <span class="card-title">快捷操作</span>
        </div>
        <div class="card-body">
          <div class="quick-actions">
            <div class="action-btn" @click="$router.push('/promotions/batch-enroll')">
              <div class="action-icon primary">
                <el-icon><Upload /></el-icon>
              </div>
              <span class="action-text">批量报名</span>
            </div>
            <div class="action-btn" @click="$router.push('/promotions/loss-process')">
              <div class="action-icon warning">
                <el-icon><Edit /></el-icon>
              </div>
              <span class="action-text">亏损处理</span>
            </div>
            <div class="action-btn" @click="$router.push('/promotions/reprice')">
              <div class="action-icon success">
                <el-icon><PriceTag /></el-icon>
              </div>
              <span class="action-text">改价推广</span>
            </div>
            <div class="action-btn" @click="handleExport">
              <div class="action-icon accent">
                <el-icon><Download /></el-icon>
              </div>
              <span class="action-text">导出数据</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 最近操作 -->
      <div class="glass-card">
        <div class="card-header">
          <span class="card-title">最近操作</span>
          <el-button text size="small" @click="$router.push('/admin/logs')">
            查看全部
          </el-button>
        </div>
        <div class="card-body">
          <el-table :data="recentLogs" size="small" max-height="280">
            <el-table-column prop="operation_type" label="操作类型" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="getOperationTagType(row.operation_type)">
                  {{ formatOperationType(row.operation_type) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="affected_count" label="数量" width="80" align="center" />
            <el-table-column prop="status" label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-icon v-if="row.status === 'success'" class="status-icon success">
                  <CircleCheckFilled />
                </el-icon>
                <el-icon v-else class="status-icon danger">
                  <CircleCloseFilled />
                </el-icon>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间">
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
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getProducts } from '@/api/product'
import { exportPromotable } from '@/api/promotion'
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
  DocumentDelete
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

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24px;
}

@media (max-width: 1200px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

.status-icon {
  font-size: 18px;

  &.success {
    color: var(--success);
  }

  &.danger {
    color: var(--danger);
  }
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

  .el-icon {
    font-size: 48px;
    margin-bottom: 12px;
    opacity: 0.5;
  }

  p {
    font-size: 14px;
  }
}
</style>
